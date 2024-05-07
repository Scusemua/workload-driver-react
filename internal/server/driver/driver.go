package driver

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mattn/go-colorable"
	"github.com/mgutz/ansi"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/jupyter"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Used in generating random IDs for workloads.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	// Used by ResourceRequest structs when they do not require/request a specific GPU.
	AnyGPU = "ANY_GPU"
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
type workloadDriverImpl struct {
	id string // Unique ID (relative to other drivers). The workload registered with this driver will be assigned this ID.

	kernelManager *jupyter.KernelSessionManager

	workloadGenerator domain.WorkloadGenerator

	sessionConnections map[string]*jupyter.SessionConnection

	clockTrigger *Trigger

	stats *WorkloadStats

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	eventQueue          domain.EventQueueService
	tickDuration        time.Duration
	tickDurationSeconds int64               // Cached total number of seconds of tickDuration
	doneChan            chan interface{}    // Used to signal that the workload has successfully processed all events.
	stopChan            chan interface{}    // Used to stop the workload early/prematurely (i.e., before all events have been processed).
	errorChan           chan error          // Used to stop the workload due to a critical error.
	tick                time.Duration       // The tick interval/step rate of the simulation.
	ticker              *Ticker             // Receive Tick events this way.
	ticksHandled        atomic.Int64        // Incremented/accessed atomically.
	servingTicks        atomic.Bool         // The WorkloadDriver::ServeTicks() method will continue looping as long as this flag is set to true.
	eventChan           chan domain.Event   // Receives events from the Synthesizer.
	driverTimescale     float64             // Multiplier that impacts the timescale at the Driver will operate on with respect to the trace data. For example, if each tick is 60 seconds, then a DriverTimescale value of 0.5 will mean that each tick will take 30 seconds.
	performClockTicks   bool                // If true, then we'll issue clock ticks. Otherwise, don't issue them. Mostly used for testing/debugging.
	sessions            *hashmap.HashMap    // Responsible for creating sessions and maintaining a collection of all of the sessions active within the simulation.
	seenSessions        map[string]struct{} // All sessions that we've ever seen before.
	workloadEndTime     time.Time           // The time at which the workload completed.

	workloadPresets             map[string]domain.WorkloadPreset
	workloadPreset              domain.WorkloadPreset
	workloadRegistrationRequest *domain.WorkloadRegistrationRequest
	workload                    *domain.Workload

	websocket domain.ConcurrentWebSocket

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

func NewWorkloadDriver(opts *domain.Configuration, performClockTicks bool, websocket domain.ConcurrentWebSocket) domain.WorkloadDriver {
	atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	driver := &workloadDriverImpl{
		id:                  GenerateWorkloadID(8),
		eventChan:           make(chan domain.Event),
		clockTrigger:        NewTrigger(),
		opts:                opts,
		doneChan:            make(chan interface{}, 1),
		stopChan:            make(chan interface{}, 1),
		errorChan:           make(chan error, 2),
		atom:                &atom,
		tickDuration:        time.Second * time.Duration(opts.TraceStep),
		tickDurationSeconds: opts.TraceStep,
		driverTimescale:     opts.DriverTimescale,
		kernelManager:       jupyter.NewKernelManager(opts, &atom),
		sessionConnections:  make(map[string]*jupyter.SessionConnection),
		performClockTicks:   performClockTicks,
		eventQueue:          newEventQueue(&atom),
		stats:               NewWorkloadStats(),
		sessions:            hashmap.New(100),
		seenSessions:        make(map[string]struct{}),
		websocket:           websocket,
	}

	// Create the ticker for the workload.
	driver.ticker = driver.clockTrigger.NewSyncTicker(time.Second*time.Duration(opts.TraceStep), fmt.Sprintf("Workload-%s", driver.id))

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), driver.atom)
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

func (d *workloadDriverImpl) WebSocket() domain.ConcurrentWebSocket {
	return d.websocket
}

func (d *workloadDriverImpl) Stats() *WorkloadStats {
	return d.stats
}

func (d *workloadDriverImpl) SubmitEvent(evt domain.Event) {
	d.eventChan <- evt
}

// Acquire the Driver's mutex externally.
//
// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
func (d *workloadDriverImpl) LockDriver() {
	d.mu.Lock()
}

// Attempt to acquire the Driver's mutex externally.
// Returns true on successful acquiring of the lock. If lock was not acquired, return false.
//
// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
func (d *workloadDriverImpl) TryLockDriver() bool {
	return d.mu.TryLock()
}

// Release the Driver's mutex externally.
func (d *workloadDriverImpl) UnlockDriver() {
	d.mu.Unlock()
}

func (d *workloadDriverImpl) ToggleDebugLogging(enabled bool) *domain.Workload {
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

func (d *workloadDriverImpl) GetWorkload() *domain.Workload {
	return d.workload
}

func (d *workloadDriverImpl) GetWorkloadPreset() domain.WorkloadPreset {
	return d.workloadPreset
}

func (d *workloadDriverImpl) GetWorkloadRegistrationRequest() *domain.WorkloadRegistrationRequest {
	return d.workloadRegistrationRequest
}

// Returns nil if the workload could not be registered.
func (d *workloadDriverImpl) RegisterWorkload(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (*domain.Workload, error) {
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
func (d *workloadDriverImpl) WriteError(ctx *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	ctx.JSON(http.StatusInternalServerError, msg)
}

// Return true if the workload has completed; otherwise, return false.
func (d *workloadDriverImpl) IsWorkloadComplete() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.workload.WorkloadState == domain.WorkloadFinished
}

func (d *workloadDriverImpl) ID() string {
	return d.id
}

// Stop a workload that's already running/in-progress.
// Returns nil on success, or an error if one occurred.
func (d *workloadDriverImpl) StopWorkload() error {
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

func (d *workloadDriverImpl) StopChan() chan interface{} {
	return d.stopChan
}

func (d *workloadDriverImpl) handleCriticalError(err error) {
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
func (d *workloadDriverImpl) abortWorkload() error {
	d.logger.Warn("Stopping workload.", zap.String("workload-id", d.id))

	if d.workloadGenerator == nil {
		d.logger.Error("Cannot stop workload. Workload Generator is nil.", zap.String("workload-id", d.id))
		panic("Cannot stop workload. Workload Generator is nil.")
	}

	d.workloadGenerator.StopGeneratingWorkload()

	// TODO(Ben): Clean-up any sessions/kernels.
	d.logger.Warn("TODO: Clean up sessions and kernels.")
	return nil
}

// Set the ClockTime clock to the given timestamp, verifying that the new timestamp is either equal to or occurs after the old one.
// Return a tuple where the first element is the new time, and the second element is the difference between the new time and the old time.
func (d *workloadDriverImpl) incrementClockTime(time time.Time) (time.Time, time.Duration, error) {
	newTime, timeDifference, err := ClockTime.IncreaseClockTimeTo(time)

	if err != nil {
		d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
	}

	return newTime, timeDifference, err // err will be nil if everything was OK.
}

// Start the simulation.
func (d *workloadDriverImpl) bootstrapSimulation() error {
	// Get the first event.
	firstEvent := <-d.eventChan

	d.sugaredLogger.Infof("Received first event for workload %s: %v", d.workload.ID, firstEvent)

	// Save the timestamp information.
	if d.performClockTicks {
		// Set the CurrentTick to the timestamp of the event.
		_, _, err := CurrentTick.IncreaseClockTimeTo(firstEvent.Timestamp())
		if err != nil {
			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
			return err
		}

		_, _, err = ClockTime.IncreaseClockTimeTo(firstEvent.Timestamp())
		if err != nil {
			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
			return err
		}

		d.sugaredLogger.Debugf("CurrentTick has been initialized to %v.", firstEvent.Timestamp())
	}

	// Handle the event. Basically, just enqueue it in the EventQueueService.
	d.eventQueue.EnqueueEvent(firstEvent)

	return nil
}

// This should be called from its own goroutine.
// Accepts a waitgroup that is used to notify the caller when the workload has entered the 'WorkloadRunning' state.
// This issues clock ticks as events are submitted.
func (d *workloadDriverImpl) DriveWorkload(wg *sync.WaitGroup) {
	d.logger.Info("Workload Simulator has started running. Bootstrapping simulation now.")
	err := d.bootstrapSimulation()
	if err != nil {
		d.logger.Error("Failed to bootstrap simulation.", zap.String("reason", err.Error()))
		d.handleCriticalError(err)
		wg.Done()
		return
	}

	d.logger.Info("The simulation has started.")

	nextTick := CurrentTick.GetClockTime().Add(d.tickDuration)

	d.sugaredLogger.Infof("Next tick: %v", nextTick)

OUTER:
	for {
		// Constantly poll for events from the Workload Generator.
		// These events are then enqueued in the EventQueueService.
		select {
		case evt := <-d.eventChan:
			// d.logger.Debug("Extracted event from event channel.", zap.String("event-name", evt.Name().String()), zap.String("event-id", evt.Id()))

			// If the event occurs during this tick, then call SimulationDriver::HandleDriverEvent to enqueue the event in the EventQueueService.
			if evt.Timestamp().Before(nextTick) {
				d.eventQueue.EnqueueEvent(evt)
			} else {
				// The event occurs in the next tick. Update the current tick clock, issue/perform a tick-trigger, and then process the event.
				err = d.IssueClockTicks(evt.Timestamp())
				if err != nil {
					d.logger.Error("Critical error occurred while attempting to increment clock time.", zap.String("error-message", err.Error()))
					d.handleCriticalError(err)
					break OUTER
				}
				nextTick = CurrentTick.GetClockTime().Add(d.tickDuration)
				d.eventQueue.EnqueueEvent(evt)

				d.sugaredLogger.Debugf("Next tick: %v", nextTick)
			}
		case <-d.doneChan:
			d.logger.Info("Drivers are done generating events.")

			// Continue issuing ticks until the cluster is finished.
			for d.eventQueue.Len() > 0 {
				err = d.IssueClockTicks(nextTick)
				if err != nil {
					d.logger.Error("Critical error occurred while attempting to increment clock time.", zap.String("error-message", err.Error()))
					d.handleCriticalError(err)
					break OUTER
				}

				nextTick = CurrentTick.GetClockTime().Add(d.tickDuration)
			}

			break OUTER
		}
	}

	wg.Done()
}

// Issue clock ticks until the CurrentTick clock has caught up to the given timestamp.
// The given timestamp should correspond to the timestamp of the next event generated by the Workload Generator.
// The Workload Simulation will simulate everything up until this next event is ready to process.
// Then we will continue consuming all events for the current tick in the SimulationDriver::DriveSimulation function.
func (d *workloadDriverImpl) IssueClockTicks(timestamp time.Time) error {
	if !d.performClockTicks {
		return nil
	}

	currentTick := CurrentTick.GetClockTime()

	diff := timestamp.Sub(currentTick)
	numTicksRequired := int64(diff / d.tickDuration)
	d.sugaredLogger.Debugf("Next event occurs at %v, which is in %v and will require %v ticks.", timestamp, diff, numTicksRequired)

	var numTicksIssued int64 = 0
	for timestamp.After(currentTick) {
		tick, err := CurrentTick.IncrementClockBy(d.tickDuration)
		if err != nil {
			return err
		}

		tickNumber := tick.Unix() / d.tickDurationSeconds
		d.sugaredLogger.Debugf("Issuing tick #%d: %v.", tickNumber, tick)
		d.clockTrigger.Trigger(tick)
		numTicksIssued += 1
		currentTick = CurrentTick.GetClockTime()
	}

	if numTicksIssued != numTicksRequired {
		panic(fmt.Sprintf("Expected to issue %d tick(s); instead, issued %d.", numTicksRequired, numTicksIssued))
	}

	return nil
}

// This should be called from its own goroutine.
// Accepts a waitgroup that is used to notify the caller when the workload has entered the 'WorkloadRunning' state.
// This processes events in response to clock ticks.
func (d *workloadDriverImpl) ProcessWorkload(wg *sync.WaitGroup) {
	d.logger.Info("Starting workload.", zap.String("workload-preset-name", d.workloadPreset.Name()), zap.String("workload-preset-key", d.workloadPreset.Key()))

	d.workloadGenerator = generator.NewWorkloadGenerator(d.opts, d.atom, d)
	go d.workloadGenerator.GenerateWorkload(d, d.workload, d.workloadPreset, d.workloadRegistrationRequest)

	d.mu.Lock()
	d.workload.StartTime = time.Now()
	d.workload.WorkloadState = domain.WorkloadRunning
	d.mu.Unlock()

	d.logger.Info("The Workload Driver has started running.")

	numTicksServed := 0
	d.servingTicks.Store(true)
	for d.workload.IsRunning() {
		select {
		case tick := <-d.ticker.TickDelivery:
			{
				// d.logger.Debug("Recevied tick.", zap.String("workload-id", d.workload.ID), zap.Time("tick", tick))
				_, _, err := CurrentTick.IncreaseClockTimeTo(tick)
				if err != nil {
					d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
					d.handleCriticalError(err)
					wg.Done()
					return
				}

				coloredOutput := ansi.Color(fmt.Sprintf("Serving tick: %v (processing everything up to %v)", tick, tick), "blue")
				d.logger.Debug(coloredOutput)
				d.ticksHandled.Add(1)

				// If there are no events processed this tick, then we still need to increment the clocktime so we're in-line with the simulation.
				prevTickStart := tick.Add(-d.tick)
				if ClockTime.GetClockTime().Before(prevTickStart) {
					// d.sugaredLogger.Debugf("Incrementing simulation clock from %v to %v (to match \"beginning\" of tick).", ClockTime.GetClockTime(), prevTickStart)
					d.incrementClockTime(prevTickStart)
				}

				// Process "session ready" events.
				d.handleSessionReadyEvents(tick)

				// Process "start/stop training" events.
				d.processEvents(tick)

				d.doneServingTick()
				numTicksServed += 1

				if numTicksServed > 64 {
					panic("Something is wrong. We've served 64 ticks.")
				}
			}
		case err := <-d.errorChan:
			{
				d.logger.Error("Recevied error.", zap.String("workload-id", d.workload.ID), zap.Error(err))
				d.handleCriticalError(err)
				wg.Done()
				return // We're done, so we can return.
			}
		case <-d.stopChan:
			{
				d.logger.Info("Workload has been instructed to terminate early.", zap.String("workload-id", d.workload.ID))
				d.abortWorkload()
				wg.Done()
				return // We're done, so we can return.
			}
		case <-d.doneChan: // This is placed after eventChan so that all events are processed first.
			{
				d.workloadEndTime = time.Now()
				d.logger.Info("The Workload Generator has finished generating events.", zap.String("workload-id", d.workload.ID))
				d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.StartTime)), zap.Any("workload-start-time", d.workload.StartTime), zap.Any("workload-end-time", d.workloadEndTime))

				d.mu.Lock()
				defer d.mu.Unlock()

				d.workload.WorkloadState = domain.WorkloadFinished
				wg.Done()

				return // We're done, so we can return.
			}
		}
	}
}

// Called from Cluster::ServeTicks at the end of serving a tick to signal to the Ticker/Trigger interface that the listener (i.e., the Cluster) is done.
func (d *workloadDriverImpl) doneServingTick() {
	numEventsEnqueued := d.eventQueue.Len()
	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf(">> [%v] Done serving tick. There is/are %d more session event(s) enqueued right now.", ClockTime.GetClockTime(), numEventsEnqueued)
	}

	d.ticker.Done()
}

// Signal that the workload is done (being parsed) by the generator/synthesizer.
func (d *workloadDriverImpl) DoneChan() chan interface{} {
	return d.doneChan
}

func (d *workloadDriverImpl) EventQueue() domain.EventQueueService {
	return d.eventQueue
}

// Process events in chronological/simulation order.
// This accepts the "current tick" as an argument. The current tick is basically an upper-bound on the times for
// which we'll process an event. For example, if `tick` is 19:05:00, then we will process all cluster and session
// events with timestamps that are (a) equal to 19:05:00 or (b) come before 19:05:00. Any events with timestamps
// that come after 19:05:00 will not be processed until the next tick.
func (d *workloadDriverImpl) processEvents(tick time.Time) {
	// d.logger.Debug("Processing cluster/session events.", zap.Time("tick", tick))

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
		// d.logger.Warn("No events generated. Skipping tick.", zap.Time("tick", tick))
		return
	}

	waitGroup.Add(len(eventsForTick))
	d.sugaredLogger.Debugf("Processing %d event(s) for tick %v.", len(eventsForTick), tick)

	for sessId, evts := range eventsForTick {
		d.sugaredLogger.Debugf("Number of events for Session %s: %d", sessId, len(evts))

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

// Given a session ID, such as from the trace data, return the ID used internally.
//
// The internal ID includes the unique ID of this workload driver, in case multiple
// workloads from the same trace are being executed concurrently.
func (d *workloadDriverImpl) getInternalSessionId(traceSessionId string) string {
	return fmt.Sprintf("%s-%s", traceSessionId, d.id)
}

func (d *workloadDriverImpl) getOriginalSessionIdFromInternalSessionId(internalSessionId string) string {
	rightIndex := strings.LastIndex(internalSessionId, "-")
	return internalSessionId[0:rightIndex]
}

// Create and return a new Session with the given ID.
func (d *workloadDriverImpl) NewSession(id string, meta *generator.Session, createdAtTime time.Time) *Session {
	d.sugaredLogger.Debugf("Creating new Session %v. MaxSessionCPUs: %.2f; MaxSessionMemory: %.2f. MaxSessionGPUs: %d. TotalNumSessions: %d", id, meta.MaxSessionCPUs, meta.MaxSessionMemory, meta.MaxSessionGPUs, d.sessions.Len())

	var session *Session = d.GetSession(id)
	if session != nil {
		panic(fmt.Sprintf("Attempted to create existing Session %s.", id))
	}

	// The Session only exposes the CPUs, Memory, and
	resourceRequest := NewResourceRequest(meta.MaxSessionCPUs, meta.MaxSessionMemory, float64(meta.MaxSessionGPUs), AnyGPU)
	session = NewSession(id, meta, resourceRequest, createdAtTime)

	d.mu.Lock()

	internalSessionId := d.getInternalSessionId(session.Id())

	d.workload.NumActiveSessions += 1
	d.workload.NumSessionsCreated += 1
	d.Stats().TotalNumSessions += 1
	d.seenSessions[internalSessionId] = struct{}{}
	d.sessions.Set(internalSessionId, session)

	d.mu.Unlock()

	return session
}

// Get and return the Session identified by the given ID, if one exists. Otherwise, return nil.
// If the caller is attempting to retrieve a Session that once existed but has since been terminated, then this will return nil.
func (d *workloadDriverImpl) GetSession(id string) *Session {
	session, ok := d.sessions.Get(id)

	if ok {
		return session.(*Session)
	}

	return nil
}

// Schedule new Sessions onto Hosts.
func (d *workloadDriverImpl) handleSessionReadyEvents(latestTick time.Time) {
	sessionReadyEvent := d.eventQueue.GetNextSessionStartEvent(latestTick)
	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf(">> [%v] Handling EventSessionReady events now.", ClockTime.GetClockTime())
	}
	for sessionReadyEvent != nil {
		driverSession := sessionReadyEvent.Data().(*generator.Session)
		// updatedTs := RoundUpTime(sessionReadyEvent.Timestamp, d.tick)
		// driverSession.Timestamp = updatedTs
		// sessionReadyEvent.Timestamp = updatedTs

		sessionId := driverSession.Pod
		if d.sugaredLogger.Level() == zapcore.DebugLevel {
			d.sugaredLogger.Debugf("Handling EventSessionReady for Session %s [ts: %v].", sessionId, sessionReadyEvent.Timestamp)
		}

		_, err := d.provisionSession(sessionId, driverSession, sessionReadyEvent.Timestamp())
		if err != nil {
			d.logger.Error("Failed to provision new Jupyter session.", zap.String("session-id", sessionId), zap.Error(err))
			payload, _ := json.Marshal(domain.ErrorMessage{
				Description:  reflect.TypeOf(err).Name(),
				ErrorMessage: err.Error(),
				Valid:        true,
			})
			d.websocket.WriteMessage(websocket.BinaryMessage, payload)
		} else {
			d.logger.Debug("Successfully handled SessionStarted event.", zap.String("session-id", sessionId))
		}

		// Get the next ready-to-process `EventSessionReady` event if there is one. If not, then this will return nil, and we'll exit the for-loop.
		sessionReadyEvent = d.eventQueue.GetNextSessionStartEvent(latestTick)
	}
}

func (d *workloadDriverImpl) handleEvent(evt domain.Event) error {
	traceSessionId := evt.Data().(*generator.Session).Pod
	internalSessionId := d.getInternalSessionId(traceSessionId)

	switch evt.Name() {
	case domain.EventSessionStarted:
		// d.logger.Debug("Received SessionStarted event.", zap.String("session-id", sessionId))
		panic("Received SessionStarted event.")
	case domain.EventSessionTrainingStarted:
		d.logger.Debug("Received TrainingStarted event.", zap.String("session", internalSessionId))

		// TODO: Initiate training.
		if _, ok := d.seenSessions[internalSessionId]; !ok {
			d.logger.Error("Received 'training-started' event for unknown session.", zap.String("session-id", internalSessionId))
			return ErrTrainingStartedUnknownSession
		}

		d.mu.Lock()
		d.workload.NumActiveTrainings += 1
		d.mu.Unlock()
		d.logger.Debug("Handled TrainingStarted event.", zap.String("session", internalSessionId))
	case domain.EventSessionUpdateGpuUtil:
		d.logger.Debug("Received UpdateGpuUtil event.", zap.String("session", internalSessionId))
		// TODO: Update GPU utilization.
		d.logger.Debug("Handled UpdateGpuUtil event.", zap.String("session", internalSessionId))
	case domain.EventSessionTrainingEnded:
		d.logger.Debug("Received TrainingEnded event.", zap.String("session", internalSessionId))

		if _, ok := d.seenSessions[internalSessionId]; !ok {
			d.logger.Error("Received 'training-stopped' event for unknown session.", zap.String("session-id", internalSessionId))
			return ErrTrainingStoppedUnknownSession
		}

		err := d.kernelManager.InterruptKernel(internalSessionId)
		if err != nil {
			d.logger.Error("Error while interrupting kernel.", zap.String("kernel-id", internalSessionId), zap.Error(err))
		}

		d.mu.Lock()
		d.workload.NumTasksExecuted += 1
		d.workload.NumActiveTrainings -= 1
		d.mu.Unlock()
		d.logger.Debug("Handled TrainingEnded event.", zap.String("session", internalSessionId))
	case domain.EventSessionStopped:
		d.logger.Debug("Received SessionStopped event.", zap.String("session", internalSessionId))

		// TODO: Stop session.
		err := d.stopSession(d.getOriginalSessionIdFromInternalSessionId(internalSessionId))
		if err != nil {
			return err
		}

		delete(d.seenSessions, internalSessionId)

		d.mu.Lock()
		d.workload.NumActiveSessions -= 1
		d.mu.Unlock()
		d.logger.Debug("Handled SessionStopped event.", zap.String("session", internalSessionId))
	}

	return nil
}

func (d *workloadDriverImpl) provisionSession(sessionId string, meta *generator.Session, createdAtTime time.Time) (*jupyter.SessionConnection, error) {
	d.logger.Debug("Creating new kernel.", zap.String("kernel-id", sessionId))
	st := time.Now()
	sessionConnection, err := d.kernelManager.CreateSession(strings.ToLower(sessionId), fmt.Sprintf("%s.ipynb", sessionId), "notebook", "distributed")
	if err != nil {
		d.logger.Error("Failed to create session.", zap.String("session-id", sessionId))
		return nil, err
	}

	timeElapsed := time.Since(st)
	d.logger.Debug("Successfully created new kernel.", zap.String("kernel-id", sessionId), zap.Duration("time-elapsed", timeElapsed))
	d.sessionConnections[sessionId] = sessionConnection

	d.NewSession(sessionId, meta, createdAtTime)

	return sessionConnection, nil
}

func (d *workloadDriverImpl) stopSession(sessionId string) error {
	d.logger.Debug("Stopping session.", zap.String("kernel-id", sessionId))
	return d.kernelManager.StopKernel(sessionId)
}
