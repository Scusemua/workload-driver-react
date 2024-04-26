package driver

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/mgutz/ansi"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/jupyter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Used in generating random IDs for workloads.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var (
	ErrWorkloadPresetNotFound    = errors.New("could not find workload preset with specified key")
	ErrWorkloadAlreadyRegistered = errors.New("driver already has a workload registered with it")

	ErrWorkloadNotRunning = errors.New("the workload is currently not running")

	ErrTrainingStartedUnknownSession = errors.New("received 'training-started' event for unknown session")
	ErrTrainingStoppedUnknownSession = errors.New("received 'training-ended' event for unknown session")

	src = rand.NewSource(time.Now().UnixNano())
)

// The Workload Driver consumes events from the Workload Generator and takes action accordingly.
type WorkloadDriver struct {
	id string // Unique ID (relative to other drivers). The workload registered with this driver will be assigned this ID.

	kernelManager *jupyter.KernelSessionManager

	workloadGenerator domain.WorkloadGenerator

	sessionConnections map[string]*jupyter.SessionConnection

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	// sessions map[string]*generator.Session // Responsible for creating sessions and maintaining a collection of all of the sessions active within the simulation.

	kernels map[string]struct{}

	eventQueue domain.EventQueueService
	doneChan   chan struct{} // Used to signal that the workload has successfully processed all events.
	stopChan   chan struct{} // Used to stop the workload early/prematurely (i.e., before all events have been processed).
	errorChan  chan error    // Used to stop the workload due to a critical error.

	tick         time.Duration // The tick interval/step rate of the simulation.
	ticker       *Ticker       // Receive Tick events this way.
	ticksHandled atomic.Int64  // Incremented/accessed atomically.
	servingTicks atomic.Bool   // The WorkloadDriver::ServeTicks() method will continue looping as long as this flag is set to true.

	workloadPresets             map[string]domain.WorkloadPreset
	workloadPreset              domain.WorkloadPreset
	workloadRegistrationRequest *domain.WorkloadRegistrationRequest
	workload                    *domain.Workload

	// Multiplier that impacts the timescale at the Driver will operate on with respect to the trace data. For example, if each tick is 60 seconds, then a DriverTimescale value of 0.5 will mean that each tick will take 30 seconds.
	driverTimescale float64

	workloadEndTime time.Time // The time at which the workload completed.

	mu sync.Mutex

	opts *domain.Configuration
}

func GenerateWorkloadID(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func NewWorkloadDriver(opts *domain.Configuration) *WorkloadDriver {
	atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	driver := &WorkloadDriver{
		id:                 GenerateWorkloadID(8),
		opts:               opts,
		doneChan:           make(chan struct{}, 1),
		stopChan:           make(chan struct{}, 1),
		errorChan:          make(chan error, 2),
		atom:               &atom,
		ticker:             NewSyncTicker(time.Second*time.Duration(opts.TraceStep), "Cluster"),
		driverTimescale:    opts.DriverTimescale,
		kernelManager:      jupyter.NewKernelManager(opts, &atom),
		kernels:            make(map[string]struct{}),
		sessionConnections: make(map[string]*jupyter.SessionConnection),
		eventQueue:         newEventQueue(&atom),
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, driver.atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	driver.logger = logger
	driver.sugaredLogger = logger.Sugar()

	// Load the list of workload presets from the specified file.
	driver.logger.Debug("Loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath))
	presets, err := domain.LoadWorkloadPresetsFromFile(opts.WorkloadPresetsFilepath)
	if err != nil {
		driver.logger.Error("Error encountered while loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath), zap.Error(err))
		panic(err)
	}

	driver.workloadPresets = make(map[string]domain.WorkloadPreset, len(presets))
	for _, preset := range presets {
		driver.workloadPresets[preset.Key()] = preset

		driver.logger.Debug("Discovered preset.", zap.Any(fmt.Sprintf("preset-%s", preset.Key()), preset.String()))
	}

	return driver
}

// Acquire the Driver's mutex externally.
//
// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
func (d *WorkloadDriver) LockDriver() {
	d.mu.Lock()
}

// Attempt to acquire the Driver's mutex externally.
// Returns true on successful acquiring of the lock. If lock was not acquired, return false.
//
// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
func (d *WorkloadDriver) TryLockDriver() bool {
	return d.mu.TryLock()
}

// Release the Driver's mutex externally.
func (d *WorkloadDriver) UnlockDriver() {
	d.mu.Unlock()
}

func (d *WorkloadDriver) ToggleDebugLogging(enabled bool) *domain.Workload {
	d.mu.Lock()
	defer d.mu.Unlock()

	if enabled {
		d.atom.SetLevel(zap.DebugLevel)
		d.workload.DebugLoggingEnabled = true
	} else {
		d.atom.SetLevel(zap.InfoLevel)
		d.workload.DebugLoggingEnabled = false
	}

	return d.workload
}

func (d *WorkloadDriver) GetWorkload() *domain.Workload {
	return d.workload
}

func (d *WorkloadDriver) GetWorkloadPreset() domain.WorkloadPreset {
	return d.workloadPreset
}

func (d *WorkloadDriver) GetWorkloadRegistrationRequest() *domain.WorkloadRegistrationRequest {
	return d.workloadRegistrationRequest
}

// Returns nil if the workload could not be registered.
func (d *WorkloadDriver) RegisterWorkload(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (*domain.Workload, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Only one workload per driver.
	if d.workload != nil {
		return nil, ErrWorkloadAlreadyRegistered
	}

	d.workloadRegistrationRequest = workloadRegistrationRequest
	d.sugaredLogger.Debugf("User is requesting the execution of workload '%s'", workloadRegistrationRequest.Key)

	var ok bool
	if d.workloadPreset, ok = d.workloadPresets[workloadRegistrationRequest.Key]; !ok {
		d.logger.Error("Could not find workload preset with specified key.", zap.String("key", workloadRegistrationRequest.Key))

		return nil, ErrWorkloadPresetNotFound
	}

	if !workloadRegistrationRequest.DebugLogging {
		d.logger.Debug("Setting log-level to INFO.")
		d.atom.SetLevel(zapcore.InfoLevel)
		d.logger.Debug("Ignored.")
	} else {
		d.logger.Debug("Debug-level logging is ENABLED.")
	}

	d.workload = &domain.Workload{
		ID:                  d.id, // Same ID as the driver.
		Name:                workloadRegistrationRequest.WorkloadName,
		WorkloadState:       domain.WorkloadReady,
		WorkloadPreset:      d.workloadPreset,
		WorkloadPresetName:  d.workloadPreset.Name(),
		WorkloadPresetKey:   d.workloadPreset.Key(),
		TimeElasped:         time.Duration(0).String(),
		Seed:                d.workloadRegistrationRequest.Seed,
		RegisteredTime:      time.Now(),
		NumTasksExecuted:    0,
		NumEventsProcessed:  0,
		NumSessionsCreated:  0,
		NumActiveSessions:   0,
		NumActiveTrainings:  0,
		DebugLoggingEnabled: workloadRegistrationRequest.DebugLogging,
	}

	// If the workload seed is negative, then assign it a random value.
	if d.workloadRegistrationRequest.Seed < 0 {
		d.workload.Seed = rand.Int63n(2147483647) // We restrict the user to the range 0-2,147,483,647 when they specify a seed.
		d.logger.Debug("Will use random seed for RNG.", zap.Int64("seed", d.workload.Seed))
	} else {
		d.logger.Debug("Will use user-specified seed for RNG.", zap.Int64("seed", d.workload.Seed))
	}

	return d.workload, nil
}

// Write an error back to the client.
func (d *WorkloadDriver) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	c.JSON(http.StatusInternalServerError, msg)
}

// Return true if the workload has completed; otherwise, return false.
func (d *WorkloadDriver) IsWorkloadComplete() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.workload.WorkloadState == domain.WorkloadFinished
}

func (d *WorkloadDriver) ID() string {
	return d.id
}

// Stop a workload that's already running/in-progress.
// Returns nil on success, or an error if one occurred.
func (d *WorkloadDriver) StopWorkload() error {
	if !d.workload.IsRunning() {
		return ErrWorkloadNotRunning
	}

	d.logger.Debug("Stopping workload.", zap.String("workload-id", d.id))
	d.stopChan <- struct{}{}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.workloadEndTime = time.Now()
	d.workload.WorkloadState = domain.WorkloadTerminated
	return nil
}

func (d *WorkloadDriver) StopChan() chan struct{} {
	return d.stopChan
}

func (d *WorkloadDriver) handleCriticalError(err error) {
	d.logger.Error("Workload encountered a critical error and must abort.", zap.String("workload-id", d.id), zap.Error(err))
	d.abortWorkload()

	d.mu.Lock()
	defer d.mu.Unlock()

	d.workload.TimeElasped = time.Since(d.workload.StartTime).String()
	d.workload.WorkloadState = domain.WorkloadErred
	d.workload.ErrorMessage = err.Error()
}

// Manually abort the workload.
// Clean up any sessions/kernels that were created.
func (d *WorkloadDriver) abortWorkload() error {
	d.logger.Warn("Stopping workload.", zap.String("workload-id", d.id))
	d.workloadGenerator.StopGeneratingWorkload()

	// TODO(Ben): Clean-up any sessions/kernels.
	d.logger.Warn("TODO: Clean up sessions and kernels.")
	return nil
}

// Set the ClockTime clock to the given timestamp, verifying that the new timestamp is either equal to or occurs after the old one.
// Return a tuple where the first element is the new time, and the second element is the difference between the new time and the old time.
func (d *WorkloadDriver) incrementClockTime(time time.Time) (time.Time, time.Duration) {
	newTime, timeDifference := ClockTime.IncreaseClockTimeTo(time)
	return newTime, timeDifference
}

// This should be called from its own goroutine.
// Accepts a waitgroup that is used to notify the caller when the workload has entered the 'WorkloadRunning' state.
func (d *WorkloadDriver) DriveWorkload(wg *sync.WaitGroup) {
	d.logger.Info("Starting workload.", zap.Any("workload-preset", d.workloadPreset), zap.Any("workload-request", d.workloadRegistrationRequest))

	d.workloadGenerator = generator.NewWorkloadGenerator(d.opts)
	go d.workloadGenerator.GenerateWorkload(d.eventQueue, d.workload, d.workloadPreset, d.workloadRegistrationRequest)

	d.mu.Lock()
	d.workload.StartTime = time.Now()
	d.workload.WorkloadState = domain.WorkloadRunning
	d.mu.Unlock()

	wg.Done()

	d.logger.Info("The Workload Driver has started running.")

	d.servingTicks.Store(true)
	for d.workload.IsRunning() {
		select {
		case tick := <-d.ticker.TickDelivery:
			{
				CurrentTick.IncreaseClockTimeTo(tick)
				coloredOutput := ansi.Color(fmt.Sprintf("Serving tick: %v (processing everything up to %v)", tick, tick), "blue")
				d.logger.Debug(coloredOutput)
				d.ticksHandled.Add(1)

				// If there are no events processed this tick, then we still need to increment the clocktime so we're in-line with the simulation.
				prevTickStart := tick.Add(-d.tick)
				if ClockTime.GetClockTime().Before(prevTickStart) {
					d.sugaredLogger.Debugf("Incrementing simulation clock from %v to %v (to match \"beginning\" of tick).", ClockTime.GetClockTime(), prevTickStart)
					d.incrementClockTime(prevTickStart)
				}

				d.processEvents(tick)
			}
		case err := <-d.errorChan:
			{
				d.handleCriticalError(err)
				return // We're done, so we can return.
			}
		case <-d.stopChan:
			{
				d.logger.Info("Workload has been instructed to terminate early.")
				d.abortWorkload()
				return // We're done, so we can return.
			}
		case <-d.doneChan: // This is placed after eventChan so that all events are processed first.
			{
				d.mu.Lock()
				defer d.mu.Unlock()

				d.workloadEndTime = time.Now()
				d.workload.WorkloadState = domain.WorkloadFinished

				d.logger.Info("The Workload Generator has finished generating events.")
				d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.StartTime)), zap.Any("workload-start-time", d.workload.StartTime), zap.Any("workload-end-time", d.workloadEndTime))

				return // We're done, so we can return.
			}
		}
	}
}

func (d *WorkloadDriver) EventQueue() domain.EventQueueService {
	return d.eventQueue
}

// Process events in chronological/simulation order.
// This accepts the "current tick" as an argument. The current tick is basically an upper-bound on the times for
// which we'll process an event. For example, if `tick` is 19:05:00, then we will process all cluster and session
// events with timestamps that are (a) equal to 19:05:00 or (b) come before 19:05:00. Any events with timestamps
// that come after 19:05:00 will not be processed until the next tick.
func (d *WorkloadDriver) processEvents(tick time.Time) {
	d.logger.Debug("Processing cluster/session events.", zap.Time("tick", tick))

	var (
		waitGroup     sync.WaitGroup
		eventsForTick = make(map[string][]domain.Event)
	)

	for d.eventQueue.HasEventsForTick(tick) {
		evt, ok := d.eventQueue.GetNextEvent(tick)

		if !ok {
			d.logger.Error("Expected to find valid event.", zap.Time("tick", tick))
			break
		}

		sessionId := evt.Data().(domain.PodData).GetPod()
		sessionEvents, ok := eventsForTick[sessionId]
		if !ok {
			sessionEvents = make([]domain.Event, 0, 1)
		}

		sessionEvents = append(sessionEvents, evt.GetEvent())
		eventsForTick[sessionId] = sessionEvents
	}

	if len(eventsForTick) == 0 {
		d.logger.Warn("No events generated. Skipping tick.", zap.Time("tick", tick))
		return
	}

	waitGroup.Add(len(eventsForTick))
	d.sugaredLogger.Debugf("Processing %d event(s) for tick %v.", len(eventsForTick), tick)

	for sessId, evts := range eventsForTick {
		d.sugaredLogger.Debug("Number of events for Session %s: %d", sessId, len(evts))

		// Create a go routine to process all of the events for the particular session serially.
		// This enables us to process events targeting multiple sessiosn in-parallel.
		go func(sessionId string, events []domain.Event) {
			for idx, event := range events {
				d.sugaredLogger.Debugf("Handling event %d/%d: \"%s\"", idx+1, len(eventsForTick), event.Name())
				err := d.handleEvent(event)
				if err != nil {
					d.logger.Error("Failed to handle event.", zap.Any("event-name", event.Name()), zap.Any("event-id", event.Id()), zap.String("error-message", err.Error()), zap.Int("event-index", idx))
					d.errorChan <- err
					time.Sleep(time.Millisecond * time.Duration(100))
				}
				waitGroup.Done()
			}
		}(sessId, evts)
	}

	waitGroup.Wait()

	d.mu.Lock()
	d.workload.NumEventsProcessed += 1
	d.workload.TimeElasped = time.Since(d.workload.StartTime).String()
	d.mu.Unlock() // We have to explicitly unlock here, since we aren't returning immediately in this case.
}

func (d *WorkloadDriver) handleEvent(evt domain.Event) error {
	traceSessionId := evt.Data().(*generator.Session).Pod
	// Append the workload ID to the session ID so sessions are unique across workloads.
	sessionId := fmt.Sprintf("%s-%s", traceSessionId, d.id)

	switch evt.Name() {
	case domain.EventSessionStarted:
		d.logger.Debug("Received SessionStarted event.", zap.String("session-id", sessionId))

		// TODO: Start session.
		_, err := d.createSession(sessionId)
		if err != nil {
			return err
		}

		d.kernels[sessionId] = struct{}{}

		d.mu.Lock()
		d.workload.NumActiveSessions += 1
		d.workload.NumSessionsCreated += 1
		d.mu.Unlock()
		d.logger.Debug("Handled SessionStarted event.", zap.String("session-id", sessionId))
	case domain.EventSessionTrainingStarted:
		d.logger.Debug("Received TrainingStarted event.", zap.String("session", sessionId))

		// TODO: Initiate training.
		if _, ok := d.kernels[sessionId]; !ok {
			d.logger.Error("Received 'training-started' event for unknown session.", zap.String("session-id", sessionId))
			return ErrTrainingStartedUnknownSession
		}

		d.mu.Lock()
		d.workload.NumActiveTrainings += 1
		d.mu.Unlock()
		d.logger.Debug("Handled TrainingStarted event.", zap.String("session", sessionId))
	case domain.EventSessionUpdateGpuUtil:
		d.logger.Debug("Received UpdateGpuUtil event.", zap.String("session", sessionId))
		// TODO: Update GPU utilization.
		d.logger.Debug("Handled UpdateGpuUtil event.", zap.String("session", sessionId))
	case domain.EventSessionTrainingEnded:
		d.logger.Debug("Received TrainingEnded event.", zap.String("session", sessionId))

		if _, ok := d.kernels[sessionId]; !ok {
			d.logger.Error("Received 'training-stopped' event for unknown session.", zap.String("session-id", sessionId))
			return ErrTrainingStoppedUnknownSession
		}

		err := d.kernelManager.InterruptKernel(sessionId)
		if err != nil {
			d.logger.Error("Error while interrupting kernel.", zap.String("kernel-id", sessionId), zap.Error(err))
		}

		d.mu.Lock()
		d.workload.NumTasksExecuted += 1
		d.workload.NumActiveTrainings -= 1
		d.mu.Unlock()
		d.logger.Debug("Handled TrainingEnded event.", zap.String("session", sessionId))
	case domain.EventSessionStopped:
		d.logger.Debug("Received SessionStopped event.", zap.String("session", sessionId))

		// TODO: Stop session.
		err := d.stopSession(sessionId)
		if err != nil {
			return err
		}

		delete(d.kernels, sessionId)

		d.mu.Lock()
		d.workload.NumActiveSessions -= 1
		d.mu.Unlock()
		d.logger.Debug("Handled SessionStopped event.", zap.String("session", sessionId))
	}

	return nil
}

func (d *WorkloadDriver) createSession(sessionId string) (*jupyter.SessionConnection, error) {
	d.logger.Debug("Creating new kernel.", zap.String("kernel-id", sessionId))
	st := time.Now()
	sessionConnection, err := d.kernelManager.CreateSession(sessionId, sessionId, fmt.Sprintf("%s.ipynb", sessionId), "notebook", "distributed")
	if err != nil {
		d.logger.Error("Failed to create session.", zap.String("session-id", sessionId))
		return nil, err
	}

	timeElapsed := time.Since(st)
	d.logger.Debug("Successfully created new kernel.", zap.String("kernel-id", sessionId), zap.Duration("time-elapsed", timeElapsed))
	d.sessionConnections[sessionId] = sessionConnection

	return sessionConnection, nil
}

func (d *WorkloadDriver) stopSession(sessionId string) error {
	d.logger.Debug("Stopping session.", zap.String("kernel-id", sessionId))
	return d.kernelManager.StopKernel(sessionId)
}
