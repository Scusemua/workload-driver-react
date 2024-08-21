package workload

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
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/clock"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/event_queue"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/jupyter"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Used in generating random IDs for workloads.
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	ZapInternalSessionIDKey = "internal-session-id"
	ZapTraceSessionIDKey    = "trace-session-id"

	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	// Used by ResourceRequest structs when they do not require/request a specific GPU.
	AnyGPU = "ANY_GPU"

	// TODO: Figure out a good way to do this, such as via a library like:
	// https://github.com/GaetanoCarlucci/CPULoadGenerator/tree/Python3
	// which could enable us to simulate an actual CPU load based on trace data.
	TrainingCode = `
# This is the code we run in a notebook cell to simulate training.
import socket, os
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

# Connect to the kernel's TCP socket.
sock.connect(("127.0.0.1", 5555))
print(f'Connected to local TCP server. Local addr: {sock.getsockname()}')

# Blocking call.
# When training ends, the kernel will be sent a notification. 
# It will then send us a message, unblocking us here and allowing to finish the cell execution.
sock.recv(1024)

print("Received 'stop' notification. Done training.")

del sock
`
)

var (
	ErrWorkloadPresetNotFound              = errors.New("could not find workload preset with specified key")
	ErrWorkloadAlreadyRegistered           = errors.New("driver already has a workload registered with it")
	ErrUnsupportedWorkloadType             = errors.New("unsupported workload type")
	ErrWorkloadRegistrationMissingTemplate = errors.New("workload registration request for template-based workload is missing the template itself")

	ErrUnknownSession      = errors.New("received 'training-started' or 'training-ended' event for unknown session")
	ErrNoSessionConnection = errors.New("received 'training-started' or 'training-ended' event for session for which no session connection exists")
	ErrNoKernelConnection  = errors.New("received 'training-started' or 'training-ended' event for session for which no kernel connection exists")

	src = rand.NewSource(time.Now().UnixNano())
)

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

// The Workload Driver consumes events from the Workload Generator and takes action accordingly.
type workloadDriverImpl struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	clockTime                   domain.SimulationClock                // Contains the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
	clockTrigger                *clock.Trigger                        // Trigger for the clock ticks
	currentTick                 domain.SimulationClock                // Contains the current tick of the workload.
	doneChan                    chan interface{}                      // Used to signal that the workload has successfully processed all events.
	driverTimescale             float64                               // Multiplier that impacts the timescale at the Driver will operate on with respect to the trace data. For example, if each tick is 60 seconds, then a DriverTimescale value of 0.5 will mean that each tick will take 30 seconds.
	errorChan                   chan error                            // Used to stop the workload due to a critical error.
	eventChan                   chan domain.Event                     // Receives events from the Synthesizer.
	eventQueue                  domain.EventQueueService              // Maintains a queue of events to be processed for each session.
	id                          string                                // Unique ID (relative to other drivers). The workload registered with this driver will be assigned this ID.
	kernelManager               jupyter.KernelSessionManager          // Simplified Go implementation of the Jupyter JavaScript API.
	mu                          sync.Mutex                            // Synchronizes access to internal data structures. Can be locked externally using the Lock/Unlock API exposed by the WorkloadDriver.
	opts                        *domain.Configuration                 // The system's configuration, read from a file.
	performClockTicks           bool                                  // If true, then we'll issue clock ticks. Otherwise, don't issue them. Mostly used for testing/debugging.
	seenSessions                map[string]struct{}                   // All sessions that we've ever seen before.
	servingTicks                atomic.Bool                           // The WorkloadDriver::ServeTicks() method will continue looping as long as this flag is set to true.
	sessionConnections          map[string]*jupyter.SessionConnection // Map from internal session ID to session connection.
	sessions                    *hashmap.HashMap                      // Responsible for creating sessions and maintaining a collection of all of the sessions active within the simulation.
	stats                       *WorkloadStats                        // Metrics related to the workload's execution.
	stopChan                    chan interface{}                      // Used to stop the workload early/prematurely (i.e., before all events have been processed).
	tick                        time.Duration                         // The tick interval/step rate of the simulation.
	tickDuration                time.Duration                         // How long each tick is supposed to last.
	tickDurationSeconds         int64                                 // Cached total number of seconds of tickDuration
	ticker                      *clock.Ticker                         // Receive Tick events this way.
	ticksHandled                atomic.Int64                          // Incremented/accessed atomically.
	timescaleAdjustmentFactor   float64                               // Adjusts the timescale of the simulation. Setting this to 1 means that each tick is simulated as a whole minute. Setting this to 0.5 means each tick will be simulated for half its real time. So, if ticks are 60 seconds, and this variable is set to 0.5, then each tick will be simulated for 30 seconds before continuing to the next tick.
	websocket                   domain.ConcurrentWebSocket            // Shared Websocket used to communicate with frontend.
	workload                    domain.Workload                       // The workload being driven by this driver.
	workloadEndTime             time.Time                             // The time at which the workload completed.
	workloadGenerator           domain.WorkloadGenerator              // The entity generating the workload (from trace data, a preset, or a template).
	workloadPreset              *domain.WorkloadPreset                // The preset used by the associated workload. Will only be non-nil if the associated workload is a preset-based workload, rather than a template-based workload.
	workloadPresets             map[string]*domain.WorkloadPreset     // All of the available workload presets.
	workloadRegistrationRequest *domain.WorkloadRegistrationRequest   // The request that registered the workload that is being driven by this driver.
	workloadTemplate            *domain.WorkloadTemplate              // The template used by the associated workload. Will only be non-nil if the associated workload is a template-based workload, rather than a preset-based workload.
}

func NewWorkloadDriver(opts *domain.Configuration, performClockTicks bool, timescaleAdjustmentFactor float64, websocket domain.ConcurrentWebSocket, atom *zap.AtomicLevel) domain.WorkloadDriver {
	// atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	driver := &workloadDriverImpl{
		id:                        GenerateWorkloadID(8),
		eventChan:                 make(chan domain.Event),
		clockTrigger:              clock.NewTrigger(),
		opts:                      opts,
		doneChan:                  make(chan interface{}, 1),
		stopChan:                  make(chan interface{}, 1),
		errorChan:                 make(chan error, 2),
		atom:                      atom,
		tickDuration:              time.Second * time.Duration(opts.TraceStep),
		tickDurationSeconds:       opts.TraceStep,
		driverTimescale:           opts.DriverTimescale,
		kernelManager:             jupyter.NewKernelSessionManager(opts.JupyterServerAddress, true, atom),
		sessionConnections:        make(map[string]*jupyter.SessionConnection),
		performClockTicks:         performClockTicks,
		eventQueue:                event_queue.NewEventQueue(atom),
		stats:                     NewWorkloadStats(),
		sessions:                  hashmap.New(100),
		seenSessions:              make(map[string]struct{}),
		websocket:                 websocket,
		timescaleAdjustmentFactor: timescaleAdjustmentFactor,
		currentTick:               clock.NewSimulationClock(),
		clockTime:                 clock.NewSimulationClock(),
	}

	// Create the ticker for the workload.
	driver.ticker = driver.clockTrigger.NewSyncTicker(time.Second*time.Duration(opts.TraceStep), fmt.Sprintf("Workload-%s", driver.id), driver.clockTime)

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), driver.atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	driver.logger = logger
	driver.sugaredLogger = logger.Sugar()

	// TODO: Can we just load them in from a file once? Why do this for every single workload?
	// Load the list of workload presets from the specified file.
	driver.logger.Debug("Loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath))
	presets, err := domain.LoadWorkloadPresetsFromFile(opts.WorkloadPresetsFilepath)
	if err != nil {
		driver.logger.Error("Error encountered while loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath), zap.Error(err))
		panic(err)
	}

	driver.workloadPresets = make(map[string]*domain.WorkloadPreset, len(presets))
	for _, preset := range presets {
		driver.workloadPresets[preset.GetKey()] = preset

		driver.logger.Debug("Discovered preset.", zap.Any(fmt.Sprintf("preset-%s", preset.GetKey()), preset.String()))
	}

	return driver
}

// Start the Workload that is associated with/managed by this workload driver.
//
// If the workload is already running, then an error is returned.
// Likewise, if the workload was previously running but has already stopped, then an error is returned.
func (d *workloadDriverImpl) StartWorkload() error {
	return d.workload.StartWorkload()
}

// Return the channel used to report critical errors encountered while executing the workload.
func (d *workloadDriverImpl) GetErrorChan() chan<- error {
	return d.errorChan
}

// Return the current tick of the workload.
func (d *workloadDriverImpl) CurrentTick() domain.SimulationClock {
	return d.currentTick
}

// Return the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
func (d *workloadDriverImpl) ClockTime() domain.SimulationClock {
	return d.clockTime
}

// Return the WebSocket connection on which this workload was registered by a remote client and on/through which updates about the workload are reported.
func (d *workloadDriverImpl) WebSocket() domain.ConcurrentWebSocket {
	return d.websocket
}

// Return the stats of the workload.
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

func (d *workloadDriverImpl) ToggleDebugLogging(enabled bool) domain.Workload {
	d.mu.Lock()
	defer d.mu.Unlock()

	if enabled {
		d.atom.SetLevel(zap.DebugLevel)
		d.workload.SetDebugLoggingEnabled(true)
	} else {
		d.atom.SetLevel(zap.InfoLevel)
		d.workload.SetDebugLoggingEnabled(false)
	}

	return d.workload
}

func (d *workloadDriverImpl) GetEventQueue() domain.EventQueueService {
	return d.eventQueue
}

func (d *workloadDriverImpl) GetWorkload() domain.Workload {
	return d.workload
}

func (d *workloadDriverImpl) GetWorkloadPreset() *domain.WorkloadPreset {
	return d.workloadPreset
}

func (d *workloadDriverImpl) GetWorkloadRegistrationRequest() *domain.WorkloadRegistrationRequest {
	return d.workloadRegistrationRequest
}

// Create a workload that was created using a preset.
func (d *workloadDriverImpl) createWorkloadFromPreset(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (domain.Workload, error) {
	// The specified preset should be in our map of workload presets.
	// If it isn't, then the registration request is invalid, and we'll return an error.
	var ok bool
	if d.workloadPreset, ok = d.workloadPresets[workloadRegistrationRequest.Key]; !ok {
		d.logger.Error("Could not find workload preset with specified key.", zap.String("key", workloadRegistrationRequest.Key))
		return nil, ErrWorkloadPresetNotFound
	}

	d.logger.Debug("Creating new workload from preset.", zap.String("workload-name", workloadRegistrationRequest.WorkloadName), zap.String("workload-preset-name", d.workloadPreset.GetName()))
	workload := domain.NewWorkloadFromPreset(
		domain.NewWorkload(d.id, workloadRegistrationRequest.WorkloadName,
			d.workloadRegistrationRequest.Seed, workloadRegistrationRequest.DebugLogging, workloadRegistrationRequest.TimescaleAdjustmentFactor, d.atom), d.workloadPreset)

	workload.SetSource(d.workloadPreset)

	return workload, nil
}

// Create a workload that was created using a template.
func (d *workloadDriverImpl) createWorkloadFromTemplate(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (domain.Workload, error) {
	// The workload request needs to have a workload template in it.
	// If the registration request does not contain a workload template,
	// then the request is invalid, and we'll return an error.
	if workloadRegistrationRequest.Template == nil {
		d.logger.Error("Workload Registration Request for template-based workload is missing the template!")
		return nil, ErrWorkloadRegistrationMissingTemplate
	}

	d.workloadTemplate = workloadRegistrationRequest.Template
	d.logger.Debug("Creating new workload from template.", zap.String("workload-name", workloadRegistrationRequest.WorkloadName), zap.String("workload-preset-name", d.workloadTemplate.Name))
	workload := domain.NewWorkloadFromTemplate(
		domain.NewWorkload(d.id, workloadRegistrationRequest.WorkloadName,
			d.workloadRegistrationRequest.Seed, workloadRegistrationRequest.DebugLogging, workloadRegistrationRequest.TimescaleAdjustmentFactor, d.atom), d.workloadRegistrationRequest.Template,
	)

	workload.SetSource(workloadRegistrationRequest.Template)

	return workload, nil
}

// Returns nil if the workload could not be registered.
func (d *workloadDriverImpl) RegisterWorkload(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (domain.Workload, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Only one workload per driver.
	if d.workload != nil {
		return nil, ErrWorkloadAlreadyRegistered
	}

	d.workloadRegistrationRequest = workloadRegistrationRequest

	// Setup log-level.
	if !workloadRegistrationRequest.DebugLogging {
		d.logger.Debug("Setting log-level to INFO.")
		d.atom.SetLevel(zapcore.InfoLevel)
		d.logger.Debug("Ignored.")
	} else {
		d.logger.Debug("Debug-level logging is ENABLED.")
	}

	d.sugaredLogger.Debugf("User is requesting the execution of workload '%s'", workloadRegistrationRequest.Key)

	// We create the workload a little differently depending on its type (either 'preset' or 'template').
	// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
	// have properties that the user can specify and change before submitting the workload for registration.
	var (
		// If this is created successfully, then d.workload will be assigned the value of this variable.
		workload domain.Workload
		err      error // If the workload is not created successfully, then we'll return this error.
	)
	switch strings.ToLower(workloadRegistrationRequest.Type) {
	case "preset":
		{
			// Preset-workload-specific workload creation and initialization steps.
			workload, err = d.createWorkloadFromPreset(workloadRegistrationRequest)
		}
	case "template":
		{
			// Template-workload-specific workload creation and initialization steps.
			workload, err = d.createWorkloadFromTemplate(workloadRegistrationRequest)
		}
	default:
		{
			d.logger.Error("Unsupported workload type.", zap.String("workload-type", workloadRegistrationRequest.Type))
			return nil, fmt.Errorf("%w: \"%s\"", ErrUnsupportedWorkloadType, workloadRegistrationRequest.Type)
		}
	}

	// If the workload seed is negative, then assign it a random value.
	if workloadRegistrationRequest.Seed < 0 {
		workload.SetSeed(rand.Int63n(2147483647)) // We restrict the user to the range 0-2,147,483,647 when they specify a seed.
		d.logger.Debug("Will use random seed for RNG.", zap.Int64("workload-seed", workloadRegistrationRequest.Seed))
	} else {
		d.logger.Debug("Will use user-specified seed for RNG.", zap.Int64("workload-seed", workloadRegistrationRequest.Seed))
	}

	if err != nil {
		d.logger.Error("Failed to create and register workload.", zap.Error(err))
		return nil, err
	}

	d.workload = workload
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

	return d.workload.GetWorkloadState() == domain.WorkloadFinished
}

// Return the unique ID of this workload driver.
// This is not necessarily the same as the workload's unique ID (TODO: or is it?).
func (d *workloadDriverImpl) ID() string {
	return d.id
}

// Stop a workload that's already running/in-progress.
// Returns nil on success, or an error if one occurred.
func (d *workloadDriverImpl) StopWorkload() error {
	if !d.workload.IsRunning() {
		return domain.ErrWorkloadNotRunning
	}

	d.logger.Debug("Stopping workload.", zap.String("workload-id", d.id))
	d.stopChan <- struct{}{}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.workload.TerminateWorkloadPrematurely(d.clockTime.GetClockTime())
	d.workloadEndTime, _ = d.workload.GetEndTime()

	return nil
}

// Return the channel used to tell the workload to stop.
func (d *workloadDriverImpl) StopChan() chan<- interface{} {
	return d.stopChan
}

func (d *workloadDriverImpl) handleCriticalError(err error) {
	d.logger.Error("Workload encountered a critical error and must abort.", zap.String("workload-id", d.id), zap.Error(err))
	d.abortWorkload()

	d.mu.Lock()
	defer d.mu.Unlock()

	d.workload.UpdateTimeElapsed()
	d.workload.SetWorkloadState(domain.WorkloadErred)
	d.workload.SetErrorMessage(err.Error())
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

// Set the d.clockTime clock to the given timestamp, verifying that the new timestamp is either equal to or occurs after the old one.
// Return a tuple where the first element is the new time, and the second element is the difference between the new time and the old time.
func (d *workloadDriverImpl) incrementClockTime(time time.Time) (time.Time, time.Duration, error) {
	newTime, timeDifference, err := d.clockTime.IncreaseClockTimeTo(time)

	if err != nil {
		d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
	}

	return newTime, timeDifference, err // err will be nil if everything was OK.
}

// Start the simulation.
func (d *workloadDriverImpl) bootstrapSimulation() error {
	// Get the first event.
	firstEvent := <-d.eventChan

	d.sugaredLogger.Infof("Received first event for workload %s: %v", d.workload.GetId(), firstEvent)

	// Save the timestamp information.
	if d.performClockTicks {
		// Set the d.currentTick to the timestamp of the event.
		_, _, err := d.currentTick.IncreaseClockTimeTo(firstEvent.Timestamp())
		if err != nil {
			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
			return err
		}

		_, _, err = d.clockTime.IncreaseClockTimeTo(firstEvent.Timestamp())
		if err != nil {
			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
			return err
		}

		d.sugaredLogger.Debugf("d.currentTick has been initialized to %v.", firstEvent.Timestamp())
	}

	// Handle the event. Basically, just enqueue it in the EventQueueService.
	d.eventQueue.EnqueueEvent(firstEvent)

	return nil
}

// This should be called from its own goroutine.
// Accepts a waitgroup that is used to notify the caller when the workload has completed.
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

	nextTick := d.currentTick.GetClockTime().Add(d.tickDuration)

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
				nextTick = d.currentTick.GetClockTime().Add(d.tickDuration)
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

				nextTick = d.currentTick.GetClockTime().Add(d.tickDuration)
			}

			break OUTER
		}
	}

	if wg != nil {
		wg.Done()
	}
}

// Issue clock ticks until the d.currentTick clock has caught up to the given timestamp.
// The given timestamp should correspond to the timestamp of the next event generated by the Workload Generator.
// The Workload Simulation will simulate everything up until this next event is ready to process.
// Then we will continue consuming all events for the current tick in the SimulationDriver::DriveSimulation function.
func (d *workloadDriverImpl) IssueClockTicks(timestamp time.Time) error {
	if !d.performClockTicks {
		return nil
	}

	currentTick := d.currentTick.GetClockTime()

	diff := timestamp.Sub(currentTick)
	numTicksRequired := int64(diff / d.tickDuration)
	d.sugaredLogger.Debugf("Next event occurs at %v, which is in %v and will require %v ticks.", timestamp, diff, numTicksRequired)

	var numTicksIssued int64 = 0
	for timestamp.After(currentTick) {
		tickStart := time.Now()

		tick, err := d.currentTick.IncrementClockBy(d.tickDuration)
		if err != nil {
			return err
		}

		tickNumber := tick.Unix() / d.tickDurationSeconds
		d.sugaredLogger.Debugf("Issuing tick #%d: %v.", tickNumber, tick)
		d.clockTrigger.Trigger(tick)
		numTicksIssued += 1
		currentTick = d.currentTick.GetClockTime()

		tickElapsed := time.Since(tickStart)
		tickRemaining := time.Duration(d.timescaleAdjustmentFactor * float64((d.tickDuration - tickElapsed)))

		if tickRemaining < 0 {
			panic(fmt.Sprintf("Issuing clock tick #%d took %v, which is greater than the configured tick duration of %v.", tickNumber, tickElapsed, d.tickDuration))
		}

		d.sugaredLogger.Debugf("Sleeping for %v to simulate remainder of tick #%d.", tickRemaining, tickNumber)
		time.Sleep(tickRemaining)
	}

	if numTicksIssued != numTicksRequired {
		panic(fmt.Sprintf("Expected to issue %d tick(s); instead, issued %d.", numTicksRequired, numTicksIssued))
	}

	return nil
}

// This should be called from its own goroutine.
// Accepts a waitgroup that is used to notify the caller when the workload has completed.
// This processes events in response to clock ticks.
func (d *workloadDriverImpl) ProcessWorkload(wg *sync.WaitGroup) {
	if d.workloadPreset != nil {
		d.logger.Info("Starting workload.", zap.String("workload-name", d.workload.WorkloadName()), zap.String("workload-preset-name", d.workloadPreset.GetName()), zap.String("workload-preset-key", d.workloadPreset.GetKey()))
	} else if d.workloadTemplate != nil {
		d.logger.Info("Starting workload.", zap.String("workload-name", d.workload.WorkloadName()), zap.String("workload-template-name", d.workloadTemplate.Name))
	} else {
		d.logger.Info("Starting workload.", zap.String("workload-name", d.workload.WorkloadName()))
	}

	d.workloadGenerator = generator.NewWorkloadGenerator(d.opts, d.atom, d)

	if d.workload.IsPresetWorkload() {
		go d.workloadGenerator.GeneratePresetWorkload(d, d.workload, d.workload.(*domain.WorkloadFromPreset).WorkloadPreset, d.workloadRegistrationRequest)
	} else if d.workload.IsTemplateWorkload() {
		go d.workloadGenerator.GenerateTemplateWorkload(d, d.workload, d.workloadTemplate, d.workloadRegistrationRequest)
	} else {
		panic(fmt.Sprintf("Workload is of presently-unsuporrted type: \"%s\" -- cannot generate workload.", d.workload.GetWorkloadType()))
	}

	// Commented-out:
	// Moved the starting of the workload to outside this function.
	// d.mu.Lock()
	// // d.workload.StartTime = time.Now()
	// // d.workload.WorkloadState = domain.WorkloadRunning
	// d.workload.StartWorkload()
	// d.mu.Unlock()

	d.logger.Info("The Workload Driver has started running.")

	numTicksServed := 0
	d.servingTicks.Store(true)
	for d.workload.IsRunning() {
		select {
		case tick := <-d.ticker.TickDelivery:
			{
				// d.logger.Debug("Recevied tick.", zap.String("workload-id", d.workload.GetId()), zap.Time("tick", tick))
				_, _, err := d.currentTick.IncreaseClockTimeTo(tick)
				if err != nil {
					d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
					d.handleCriticalError(err)

					if wg != nil {
						wg.Done()
					}
					return
				}

				coloredOutput := ansi.Color(fmt.Sprintf("Serving tick: %v (processing everything up to %v)", tick, tick), "blue")
				d.logger.Debug(coloredOutput)
				d.ticksHandled.Add(1)

				// If there are no events processed this tick, then we still need to increment the clocktime so we're in-line with the simulation.
				prevTickStart := tick.Add(-d.tick)
				if d.clockTime.GetClockTime().Before(prevTickStart) {
					// d.sugaredLogger.Debugf("Incrementing simulation clock from %v to %v (to match \"beginning\" of tick).", d.clockTime.GetClockTime(), prevTickStart)
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
				d.logger.Error("Recevied error.", zap.String("workload-id", d.workload.GetId()), zap.Error(err))
				d.handleCriticalError(err)
				if wg != nil {
					wg.Done()
				}
				return // We're done, so we can return.
			}
		case <-d.stopChan:
			{
				d.logger.Info("Workload has been instructed to terminate early.", zap.String("workload-id", d.workload.GetId()))
				d.abortWorkload()
				if wg != nil {
					wg.Done()
				}
				return // We're done, so we can return.
			}
		case <-d.doneChan: // This is placed after eventChan so that all events are processed first.
			{
				d.mu.Lock()
				defer d.mu.Unlock()

				d.workload.SetWorkloadCompleted()

				var ok bool
				d.workloadEndTime, ok = d.workload.GetEndTime() // time.Now()
				if !ok {
					panic("`ok` should have been `true`")
				}

				d.logger.Info("The Workload Generator has finished generating events.", zap.String("workload-id", d.workload.GetId()))
				d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.GetStartTime())), zap.Any("workload-start-time", d.workload.GetStartTime()), zap.Any("workload-end-time", d.workloadEndTime))

				// d.workload.WorkloadState = domain.WorkloadFinished
				if wg != nil {
					wg.Done()
				}

				return // We're done, so we can return.
			}
		}
	}
}

// Called from workloadDriverImpl::ProcessWorkload at the end of serving a tick to signal to the Ticker/Trigger interface that the listener (i.e., the Cluster) is done.
func (d *workloadDriverImpl) doneServingTick() {
	numEventsEnqueued := d.eventQueue.Len()
	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf(">> [%v] Done serving tick. There is/are %d more session event(s) enqueued right now.", d.clockTime.GetClockTime(), numEventsEnqueued)
	}

	d.mu.Lock()
	d.workload.TickCompleted(d.ticksHandled.Load(), d.clockTime.GetClockTime())
	d.mu.Unlock()

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

		// Get the list of events for the particular session, creating said list if it does not already exist.
		sessionId := evt.Data().(domain.PodData).GetPod()
		sessionEvents, ok := eventsForTick[sessionId]
		if !ok {
			sessionEvents = make([]domain.Event, 0, 1)
		}

		sessionEvents = append(sessionEvents, evt.GetEvent())
		eventsForTick[sessionId] = sessionEvents // Put the list back into the map.
	}

	if len(eventsForTick) == 0 {
		// d.logger.Warn("No events generated. Skipping tick.", zap.Time("tick", tick))
		return
	}

	waitGroup.Add(len(eventsForTick))
	d.sugaredLogger.Debugf("Processing events for %d session(s) in tick %v.", len(eventsForTick), tick)

	for sessId, evts := range eventsForTick {
		d.sugaredLogger.Debugf("Number of events for Session %s: %d", sessId, len(evts))

		// Create a go routine to process all of the events for the particular session.
		// This enables us to process events targeting multiple sessions in-parallel.
		go func(sessionId string, events []domain.Event) {
			for idx, event := range events {
				d.sugaredLogger.Debugf("Handling event %d/%d \"%s\" for session %s now...", idx+1, len(eventsForTick), sessionId, event.Name())
				err := d.handleEvent(event)

				processed_workload_event := &domain.WorkloadEvent{
					Id:                    event.Id(),
					Session:               event.SessionID(),
					Name:                  event.Name().String(),
					Timestamp:             event.Timestamp().String(),
					ProcessedAt:           time.Now().String(),
					ProcessedSuccessfully: (err == nil),
					ErrorMessage:          err.Error(), /* Will be nil if no error occurred */
				}

				d.mu.Lock()
				d.workload.ProcessedEvent(processed_workload_event) // Record it as processed even if there was an error when processing the event.
				d.mu.Unlock()                                       // We have to explicitly unlock here, since we aren't returning immediately in this case.

				if err != nil {
					d.sugaredLogger.Errorf("Failed to handle event %d/%d \"%s\" for session %s: %v", idx+1, len(eventsForTick), sessionId, event.Name(), err)
					d.errorChan <- err
					return // We just return immediately, as the workload is going to be aborted due to the error.
				} else {
					d.sugaredLogger.Debugf("Successfully handled event %d/%d: \"%s\"", idx+1, len(eventsForTick), event.Name())
				}
			}

			d.sugaredLogger.Debugf("Finished processing %d event(s) for session \"%s\" in tick %v.", len(events), sessionId, tick)
			waitGroup.Done()
		}(sessId, evts)
	}

	waitGroup.Wait()

	d.mu.Lock()
	defer d.mu.Unlock()
	d.workload.UpdateTimeElapsed()
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
func (d *workloadDriverImpl) NewSession(id string, meta *generator.Session, createdAtTime time.Time) *domain.WorkloadSession {
	d.sugaredLogger.Debugf("Creating new Session %v. MaxSessionCPUs: %.2f; MaxSessionMemory: %.2f. MaxSessionGPUs: %d. TotalNumSessions: %d", id, meta.MaxSessionCPUs, meta.MaxSessionMemory, meta.MaxSessionGPUs, d.sessions.Len())

	// Make sure the Session doesn't already exist.
	var session *domain.WorkloadSession
	if session = d.GetSession(id); session != nil {
		panic(fmt.Sprintf("Attempted to create existing Session %s.", id))
	}

	// The Session only exposes the CPUs, Memory, and
	resourceRequest := domain.NewResourceRequest(meta.MaxSessionCPUs, meta.MaxSessionMemory, meta.MaxSessionGPUs, AnyGPU)
	session = domain.NewWorkloadSession(id, meta, resourceRequest, createdAtTime)

	d.mu.Lock()

	internalSessionId := d.getInternalSessionId(session.Id)

	// d.workload.NumActiveSessions += 1
	// d.workload.NumSessionsCreated += 1
	d.workload.SessionCreated(id)
	d.Stats().TotalNumSessions += 1
	d.seenSessions[internalSessionId] = struct{}{}
	d.sessions.Set(internalSessionId, session)

	d.mu.Unlock()

	return session
}

// Get and return the Session identified by the given ID, if one exists. Otherwise, return nil.
// If the caller is attempting to retrieve a Session that once existed but has since been terminated, then this will return nil.
func (d *workloadDriverImpl) GetSession(id string) *domain.WorkloadSession {
	session, ok := d.sessions.Get(id)

	if ok {
		return session.(*domain.WorkloadSession)
	}

	return nil
}

// Schedule new Sessions onto Hosts.
func (d *workloadDriverImpl) handleSessionReadyEvents(latestTick time.Time) {
	sessionReadyEvent := d.eventQueue.GetNextSessionStartEvent(latestTick)
	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf("[%v] Handling EventSessionReady events now.", d.clockTime.GetClockTime())
	}

	numProcessed := 0
	st := time.Now()
	for sessionReadyEvent != nil {
		driverSession := sessionReadyEvent.Data().(*generator.Session)

		sessionId := driverSession.Pod
		if d.sugaredLogger.Level() == zapcore.DebugLevel {
			d.sugaredLogger.Debugf("Handling EventSessionReady %d targeting Session %s [ts: %v].", numProcessed+1, sessionId, sessionReadyEvent.Timestamp())
		}

		provision_start := time.Now()
		_, err := d.provisionSession(sessionId, driverSession, sessionReadyEvent.Timestamp())

		d.mu.Lock()
		d.workload.ProcessedEvent(domain.NewWorkloadEvent(-1 /* This will be populated automatically by the ProcessedEvent method */, sessionReadyEvent.Id(), sessionReadyEvent.SessionID(), domain.EventSessionStarted.String(), sessionReadyEvent.Timestamp().String(), time.Now().String(), (err == nil), err))
		// &domain.WorkloadEvent{
		// 	Id:                    sessionReadyEvent.Id(),
		// 	Session:               sessionReadyEvent.SessionID(),
		// 	Name:                  domain.EventSessionStarted.String(),
		// 	Timestamp:             sessionReadyEvent.Timestamp().String(),
		// 	ProcessedAt:           time.Now().String(),
		// 	ProcessedSuccessfully: (err == nil),
		// 	ErrorMessage:          err.Error(), /* Will be nil if no error occurred */
		// })
		d.mu.Unlock()

		if err != nil {
			d.logger.Error("Failed to provision new Jupyter session.", zap.String(ZapInternalSessionIDKey, sessionId), zap.Duration("real-time-elapsed", time.Since(provision_start)), zap.Error(err))
			payload, _ := json.Marshal(domain.ErrorMessage{
				Description:  reflect.TypeOf(err).Name(),
				ErrorMessage: err.Error(),
				Valid:        true,
			})
			d.websocket.WriteMessage(websocket.BinaryMessage, payload)
			d.errorChan <- err
			return // Just return; the workload is about to end anyway (since there was an error).
		} else {
			d.logger.Debug("Successfully handled SessionStarted event.", zap.String(ZapInternalSessionIDKey, sessionId), zap.Duration("real-time-elapsed", time.Since(provision_start)))
		}

		numProcessed += 1
		// Get the next ready-to-process `EventSessionReady` event if there is one. If not, then this will return nil, and we'll exit the for-loop.
		sessionReadyEvent = d.eventQueue.GetNextSessionStartEvent(latestTick)
	}

	d.sugaredLogger.Debugf("Finished processing %d events in %v.", numProcessed, time.Since(st))
}

func (d *workloadDriverImpl) handleEvent(evt domain.Event) error {
	traceSessionId := evt.Data().(*generator.Session).Pod
	internalSessionId := d.getInternalSessionId(traceSessionId)

	switch evt.Name() {
	case domain.EventSessionStarted:
		// d.logger.Debug("Received SessionStarted event.", zap.String(ZapInternalSessionIDKey, sessionId))
		panic("Received SessionStarted event.")
	case domain.EventSessionTrainingStarted:
		d.logger.Debug("Received TrainingStarted event.", zap.String("internal-session-id", internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

		if _, ok := d.seenSessions[internalSessionId]; !ok {
			d.logger.Error("Received 'training-started' event for unknown session.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
			return ErrUnknownSession
		}

		sessionConnection, ok := d.sessionConnections[internalSessionId]
		if !ok {
			d.logger.Error("No session connection found for session upon receiving 'training-started' event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
			return ErrNoSessionConnection
		}

		kernelConnection := sessionConnection.Kernel()
		if kernelConnection == nil {
			d.logger.Error("No kernel connection found for session connection.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
			return ErrNoKernelConnection
		}

		err := kernelConnection.RequestExecute(TrainingCode, false, true, make(map[string]interface{}), true, false, false)
		if err != nil {
			d.logger.Error("Error while attempting to execute training code.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId), zap.Error(err))
			return err
		}

		d.mu.Lock()
		// d.workload.NumActiveTrainings += 1
		d.workload.TrainingStarted(traceSessionId)
		d.mu.Unlock()
		d.logger.Debug("Handled TrainingStarted event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
	case domain.EventSessionUpdateGpuUtil:
		d.logger.Debug("Received UpdateGpuUtil event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		// TODO: Update GPU utilization.
		d.logger.Debug("Handled UpdateGpuUtil event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
	case domain.EventSessionTrainingEnded:
		d.logger.Debug("Received TrainingEnded event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

		if _, ok := d.seenSessions[internalSessionId]; !ok {
			d.logger.Error("Received 'training-stopped' event for unknown session.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
			return ErrUnknownSession
		}

		if _, ok := d.seenSessions[internalSessionId]; !ok {
			d.logger.Error("Received 'training-stopped' event for unknown session.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
			return ErrUnknownSession
		}

		sessionConnection, ok := d.sessionConnections[internalSessionId]
		if !ok {
			d.logger.Error("No session connection found for session upon receiving 'training-stopped' event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
			return ErrNoSessionConnection
		}

		kernelConnection := sessionConnection.Kernel()
		if kernelConnection == nil {
			d.logger.Error("No kernel connection found for session connection.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
			return ErrNoKernelConnection
		}

		err := kernelConnection.StopRunningTrainingCode(true)
		if err != nil {
			d.logger.Error("Error while attempting to stop training.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId), zap.Error(err))
			return err
		} else {
			d.mu.Lock()
			// d.workload.NumTasksExecuted += 1
			// d.workload.NumActiveTrainings -= 1
			d.workload.TrainingStopped(traceSessionId)
			d.mu.Unlock()
			d.logger.Debug("Successfully handled TrainingEnded event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		}
	case domain.EventSessionStopped:
		d.logger.Debug("Received SessionStopped event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

		// TODO: Test that this actually works.
		err := d.stopSession(d.getOriginalSessionIdFromInternalSessionId(internalSessionId))
		if err != nil {
			return err
		} else {
			d.logger.Debug("Successfully stopped session.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		}

		delete(d.seenSessions, internalSessionId)

		d.mu.Lock()
		d.workload.SessionStopped(traceSessionId) // NumActiveSessions -= 1
		d.mu.Unlock()
		d.logger.Debug("Handled SessionStopped event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
	}

	return nil
}

func (d *workloadDriverImpl) provisionSession(sessionId string, meta *generator.Session, createdAtTime time.Time) (*jupyter.SessionConnection, error) {
	d.logger.Debug("Creating new kernel.", zap.String("kernel-id", sessionId))
	st := time.Now()
	sessionConnection, err := d.kernelManager.CreateSession(sessionId /*strings.ToLower(sessionId) */, fmt.Sprintf("%s.ipynb", sessionId), "notebook", "distributed")
	if err != nil {
		d.logger.Error("Failed to create session.", zap.String(ZapInternalSessionIDKey, sessionId))
		return nil, err
	}

	timeElapsed := time.Since(st)

	internalSessionId := d.getInternalSessionId(sessionId)
	d.sessionConnections[sessionId] = sessionConnection
	d.sessionConnections[internalSessionId] = sessionConnection

	d.logger.Debug("Successfully created new kernel.", zap.String("kernel-id", sessionId), zap.Duration("time-elapsed", timeElapsed), zap.String(ZapInternalSessionIDKey, internalSessionId))

	d.NewSession(sessionId, meta, createdAtTime)

	return sessionConnection, nil
}

func (d *workloadDriverImpl) stopSession(sessionId string) error {
	d.logger.Debug("Stopping session.", zap.String("kernel-id", sessionId))
	return d.kernelManager.StopKernel(sessionId)
}
