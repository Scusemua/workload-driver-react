package workload

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"math/rand"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// AnyGPU is used by ResourceRequest structs when they do not require/request a specific GPU.
	AnyGPU = "ANY_GPU"

	// TrainingCode is the code executed by kernels to simulate GPU training.
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

// BasicWorkloadDriver consumes events from the Workload Generator and takes action accordingly.
type BasicWorkloadDriver struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	clockTime                          domain.SimulationClock                // Contains the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
	clockTrigger                       *clock.Trigger                        // Trigger for the clock ticks
	currentTick                        domain.SimulationClock                // Contains the current tick of the workload.
	workloadExecutionCompleteChan      chan interface{}                      // Used to signal that the workload has successfully processed all events and is complete.
	workloadEventGeneratorCompleteChan chan interface{}                      // Used to signal that the generators have submitted all events. Once all remaining, already-enqueued events have been processed, the workload will be complete.
	driverTimescale                    float64                               // Multiplier that impacts the timescale at the Driver will operate on with respect to the trace data. For example, if each tick is 60 seconds, then a DriverTimescale value of 0.5 will mean that each tick will take 30 seconds.
	errorChan                          chan error                            // Used to stop the workload due to a critical error.
	eventChan                          chan domain.Event                     // Receives events from the Synthesizer.
	eventQueue                         domain.EventQueue                     // Maintains a queue of events to be processed for each session.
	id                                 string                                // Unique ID (relative to other drivers). The workload registered with this driver will be assigned this ID.
	kernelManager                      jupyter.KernelSessionManager          // Simplified Go implementation of the Jupyter JavaScript API.
	mu                                 sync.Mutex                            // Synchronizes access to internal data structures. Can be locked externally using the Lock/Unlock API exposed by the WorkloadDriver.
	opts                               *domain.Configuration                 // The system's configuration, read from a file.
	performClockTicks                  bool                                  // If true, then we'll issue clock ticks. Otherwise, don't issue them. Mostly used for testing/debugging.
	seenSessions                       map[string]struct{}                   // All sessions that we've ever seen before.
	servingTicks                       atomic.Bool                           // The WorkloadDriver::ServeTicks() method will continue looping as long as this flag is set to true.
	sessionConnections                 map[string]*jupyter.SessionConnection // Map from internal session ID to session connection.
	sessions                           *hashmap.HashMap                      // Responsible for creating sessions and maintaining a collection of all the sessions active within the simulation.
	stats                              *WorkloadStats                        // Metrics related to the workload's execution.
	stopChan                           chan interface{}                      // Used to stop the workload early/prematurely (i.e., before all events have been processed).
	tickDuration                       time.Duration                         // How long each tick is supposed to last. This is the tick interval/step rate of the simulation.
	tickDurationSeconds                int64                                 // Cached total number of seconds of tickDuration
	ticker                             *clock.Ticker                         // Receive Tick events this way.
	ticksHandled                       atomic.Int64                          // Incremented/accessed atomically.
	timescaleAdjustmentFactor          float64                               // Adjusts the timescale of the simulation. Setting this to 1 means that each tick is simulated as a whole minute. Setting this to 0.5 means each tick will be simulated for half its real time. So, if ticks are 60 seconds, and this variable is set to 0.5, then each tick will be simulated for 30 seconds before continuing to the next tick.
	websocket                          domain.ConcurrentWebSocket            // Shared Websocket used to communicate with frontend.
	workload                           domain.Workload                       // The workload being driven by this driver.
	workloadStartTime                  time.Time                             // The time at which the workload began.
	workloadEndTime                    time.Time                             // The time at which the workload completed.
	workloadGenerator                  domain.WorkloadGenerator              // The entity generating the workload (from trace data, a preset, or a template).
	workloadPreset                     *domain.WorkloadPreset                // The preset used by the associated workload. Will only be non-nil if the associated workload is a preset-based workload, rather than a template-based workload.
	workloadPresets                    map[string]*domain.WorkloadPreset     // All the available workload presets.
	workloadRegistrationRequest        *domain.WorkloadRegistrationRequest   // The request that registered the workload that is being driven by this driver.
	workloadSessions                   []*domain.WorkloadTemplateSession     // The template used by the associated workload. Will only be non-nil if the associated workload is a template-based workload, rather than a preset-based workload.
}

func NewWorkloadDriver(opts *domain.Configuration, performClockTicks bool, timescaleAdjustmentFactor float64,
	websocket domain.ConcurrentWebSocket, atom *zap.AtomicLevel) *BasicWorkloadDriver {

	driver := &BasicWorkloadDriver{
		id:                                 GenerateWorkloadID(8),
		eventChan:                          make(chan domain.Event),
		clockTrigger:                       clock.NewTrigger(),
		opts:                               opts,
		workloadExecutionCompleteChan:      make(chan interface{}, 1),
		workloadEventGeneratorCompleteChan: make(chan interface{}),
		stopChan:                           make(chan interface{}, 1),
		errorChan:                          make(chan error, 2),
		atom:                               atom,
		tickDuration:                       time.Second * time.Duration(opts.TraceStep),
		tickDurationSeconds:                opts.TraceStep,
		driverTimescale:                    opts.DriverTimescale,
		kernelManager:                      jupyter.NewKernelSessionManager(opts.JupyterServerAddress, true, atom),
		sessionConnections:                 make(map[string]*jupyter.SessionConnection),
		performClockTicks:                  performClockTicks,
		eventQueue:                         event_queue.NewBasicEventQueue(atom),
		stats:                              NewWorkloadStats(),
		sessions:                           hashmap.New(100),
		seenSessions:                       make(map[string]struct{}),
		websocket:                          websocket,
		timescaleAdjustmentFactor:          timescaleAdjustmentFactor,
		currentTick:                        clock.NewSimulationClock(),
		clockTime:                          clock.NewSimulationClock(),
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

		// driver.logger.Debug("Discovered preset.", zap.Any(fmt.Sprintf("preset-%s", preset.GetKey()), preset.String()))
	}

	return driver
}

// StartWorkload starts the Workload that is associated with/managed by this workload driver.
//
// If the workload is already running, then an error is returned.
// Likewise, if the workload was previously running but has already stopped, then an error is returned.
func (d *BasicWorkloadDriver) StartWorkload() error {
	d.logger.Debug("Workload Driver is starting workload.", zap.String("workload-driver-id", d.id))
	return d.workload.StartWorkload()
}

// GetErrorChan returns the channel used to report critical errors encountered while executing the workload.
func (d *BasicWorkloadDriver) GetErrorChan() chan<- error {
	return d.errorChan
}

// CurrentTick returns the current tick of the workload.
func (d *BasicWorkloadDriver) CurrentTick() domain.SimulationClock {
	return d.currentTick
}

// ClockTime returns the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
func (d *BasicWorkloadDriver) ClockTime() domain.SimulationClock {
	return d.clockTime
}

// WebSocket returns the WebSocket connection on which this workload was registered by a remote client and on/through which updates about the workload are reported.
func (d *BasicWorkloadDriver) WebSocket() domain.ConcurrentWebSocket {
	return d.websocket
}

// Stats returns the stats of the workload.
func (d *BasicWorkloadDriver) Stats() *WorkloadStats {
	return d.stats
}

func (d *BasicWorkloadDriver) SubmitEvent(evt domain.Event) {
	d.eventChan <- evt
}

func (d *BasicWorkloadDriver) ToggleDebugLogging(enabled bool) domain.Workload {
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

func (d *BasicWorkloadDriver) GetEventQueue() domain.EventQueue {
	return d.eventQueue
}

func (d *BasicWorkloadDriver) GetWorkload() domain.Workload {
	return d.workload
}

func (d *BasicWorkloadDriver) GetWorkloadPreset() *domain.WorkloadPreset {
	return d.workloadPreset
}

func (d *BasicWorkloadDriver) GetWorkloadRegistrationRequest() *domain.WorkloadRegistrationRequest {
	return d.workloadRegistrationRequest
}

// Create a workload that was created using a preset.
func (d *BasicWorkloadDriver) createWorkloadFromPreset(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (domain.Workload, error) {
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
func (d *BasicWorkloadDriver) createWorkloadFromTemplate(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (*domain.WorkloadFromTemplate, error) {
	// The workload request needs to have a workload template in it.
	// If the registration request does not contain a workload template,
	// then the request is invalid, and we'll return an error.
	if workloadRegistrationRequest.Sessions == nil {
		d.logger.Error("Workload Registration Request for template-based workload is missing the sessions!")
		return nil, ErrWorkloadRegistrationMissingTemplate
	}

	d.workloadSessions = workloadRegistrationRequest.Sessions
	d.logger.Debug("Creating new workload from template.", zap.String("workload-name", workloadRegistrationRequest.WorkloadName))
	workload := domain.NewWorkloadFromTemplate(
		domain.NewWorkload(d.id, workloadRegistrationRequest.WorkloadName,
			d.workloadRegistrationRequest.Seed, workloadRegistrationRequest.DebugLogging, workloadRegistrationRequest.TimescaleAdjustmentFactor, d.atom), d.workloadRegistrationRequest.Sessions,
	)

	workload.SetSource(workloadRegistrationRequest.Sessions)

	return workload, nil
}

// RegisterWorkload registers a workload with the driver.
// Returns nil if the workload could not be registered.
func (d *BasicWorkloadDriver) RegisterWorkload(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (domain.Workload, error) {
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

	if workload == nil {
		panic("Workload should not be nil at this point.")
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
	d.kernelManager.AddMetadata(jupyter.WorkloadIdMetadataKey, d.workload.GetId())
	return d.workload, nil
}

// WriteError writes an error back to the client.
func (d *BasicWorkloadDriver) WriteError(ctx *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	ctx.JSON(http.StatusInternalServerError, msg)
}

// IsWorkloadComplete returns true if the workload has completed; otherwise, return false.
func (d *BasicWorkloadDriver) IsWorkloadComplete() bool {
	return d.workload.GetWorkloadState() == domain.WorkloadFinished
}

// ID returns the unique ID of this workload driver.
// This is not necessarily the same as the workload's unique ID (TODO: or is it?).
func (d *BasicWorkloadDriver) ID() string {
	return d.id
}

// StopWorkload stops a workload that's already running/in-progress.
// Returns nil on success, or an error if one occurred.
func (d *BasicWorkloadDriver) StopWorkload() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.workload.IsRunning() {
		return domain.ErrWorkloadNotRunning
	}

	d.logger.Debug("Stopping workload.", zap.String("workload-id", d.id), zap.String("workload-state", string(d.workload.GetWorkloadState())))
	d.stopChan <- struct{}{}
	d.logger.Debug("Sent 'STOP' instruction via BasicWorkloadDriver::stopChan.", zap.String("workload-id", d.id))

	endTime, err := d.workload.TerminateWorkloadPrematurely(d.clockTime.GetClockTime())
	if err != nil {
		d.logger.Error("Failed to stop workload.", zap.String("workload-id", d.id), zap.Error(err))
		return err
	}

	// d.workloadEndTime, _ = d.workload.GetEndTime()
	d.workloadEndTime = endTime

	d.logger.Debug("Successfully stopped workload.", zap.String("workload-id", d.id))
	return nil
}

// StopChan returns the channel used to tell the workload to stop.
func (d *BasicWorkloadDriver) StopChan() chan<- interface{} {
	return d.stopChan
}

// handleCriticalError is used to abort the workload due to an error from which recovery is not possible.
func (d *BasicWorkloadDriver) handleCriticalError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Error("Workload encountered a critical error and must abort.", zap.String("workload-id", d.id), zap.Error(err))
	if abortErr := d.abortWorkload(); abortErr != nil {
		d.logger.Error("Failed to abort workload.", zap.String("workload-id", d.workload.GetId()), zap.Error(abortErr))
	}

	d.workload.UpdateTimeElapsed()
	d.workload.SetWorkloadState(domain.WorkloadErred)
	d.workload.SetErrorMessage(err.Error())
}

// abortWorkload manually aborts the workload.
// Clean up any sessions/kernels that were created.
func (d *BasicWorkloadDriver) abortWorkload() error {
	d.logger.Warn("Aborting workload.", zap.String("workload-id", d.id))

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
func (d *BasicWorkloadDriver) incrementClockTime(time time.Time) (time.Time, time.Duration, error) {
	newTime, timeDifference, err := d.clockTime.IncreaseClockTimeTo(time)

	if err != nil {
		d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
	}

	return newTime, timeDifference, err // err will be nil if everything was OK.
}

// Start the simulation.
func (d *BasicWorkloadDriver) bootstrapSimulation() error {
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

	// Handle the event. Basically, just enqueue it in the EventQueue.
	d.eventQueue.EnqueueEvent(firstEvent)

	return nil
}

// DriveWorkload accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
// This issues clock ticks as events are submitted.
//
// DriveWorkload should be called from its own goroutine.
func (d *BasicWorkloadDriver) DriveWorkload(wg *sync.WaitGroup) {
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
		// These events are then enqueued in the EventQueue.
		select {
		case evt := <-d.eventChan:
			// d.logger.Debug("Extracted event from event channel.", zap.String("event-name", evt.Name().String()), zap.String("event-id", evt.Id()))

			// If the event occurs during this tick, then call SimulationDriver::HandleDriverEvent to enqueue the event in the EventQueue.
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
		case <-d.workloadEventGeneratorCompleteChan:
			d.sugaredLogger.Debugf("Drivers finished generating events. #Events still enqueued: %d.", d.eventQueue.Len())

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

			// Signal to the goroutine running the BasicWorkloadDriver::ProcessWorkload method that the workload has completed successfully.
			d.workloadExecutionCompleteChan <- struct{}{}

			break OUTER
		}
	}

	if wg != nil {
		wg.Done()
	}
}

// IssueClockTicks issues clock ticks until the d.currentTick clock has caught up to the given timestamp.
// The given timestamp should correspond to the timestamp of the next event generated by the Workload Generator.
// The Workload Simulation will simulate everything up until this next event is ready to process.
// Then we will continue consuming all events for the current tick in the SimulationDriver::DriveSimulation function.
func (d *BasicWorkloadDriver) IssueClockTicks(timestamp time.Time) error {
	if !d.performClockTicks {
		return nil
	}

	currentTick := d.currentTick.GetClockTime()

	// We're going to issue clock ticks up until the specified timestamp.
	// Calculate how many ticks that requires so we can perform a quick sanity
	// check at the end to verify that we issued the correct number of ticks.
	numTicksToIssue := int64((timestamp.Sub(currentTick)) / d.tickDuration)

	// This is just for debugging/logging purposes.
	nextEventAtTime, errNoMoreEvents := d.eventQueue.GetTimestampOfNextReadyEvent()
	if errNoMoreEvents == nil {
		timeUntilNextEvent := nextEventAtTime.Sub(currentTick)
		numTicksTilNextEvent := int64(timeUntilNextEvent / d.tickDuration)
		d.sugaredLogger.Debugf("Preparing to issue %d clock tick(s). Next event occurs at %v, which is in %v and will require %v ticks.", numTicksToIssue, nextEventAtTime, timeUntilNextEvent, numTicksTilNextEvent)
	} else {
		d.sugaredLogger.Debugf("Preparing to issue %d clock tick(s). There are no events currently enqueued.", numTicksToIssue)
	}

	// Issue clock ticks.
	var numTicksIssued int64 = 0
	for timestamp.After(currentTick) {
		tickStart := time.Now()

		// Increment the clock.
		tick, err := d.currentTick.IncrementClockBy(d.tickDuration)
		if err != nil {
			d.logger.Error("Error while incrementing clock time.", zap.Duration("tick-duration", d.tickDuration), zap.Error(err))
			return err
		}

		tickNumber := tick.Unix() / d.tickDurationSeconds
		d.sugaredLogger.Debugf("Issuing tick #%d: %v. Event queue size: %d.", tickNumber, tick, d.eventQueue.Len())

		// Trigger the clock ticker, which will prompt the other goroutine within the workload driver to process events and whatnot for this tick.
		d.clockTrigger.Trigger(tick)
		numTicksIssued += 1
		currentTick = d.currentTick.GetClockTime()

		tickElapsed := time.Since(tickStart)
		tickRemaining := time.Duration(d.timescaleAdjustmentFactor * float64(d.tickDuration-tickElapsed))

		// Verify that the issuing of the tick did not exceed the specified real-clock-time that a tick should last.
		// TODO: Handle this more elegantly, such as by decreasing the length of subsequent ticks or something?
		if tickRemaining < 0 {
			panic(fmt.Sprintf("Issuing clock tick #%d took %v, which is greater than the configured tick duration of %v.", tickNumber, tickElapsed, d.tickDuration))
		}

		// Simulate the remainder of the tick -- however much time is left.
		d.sugaredLogger.Debugf("Sleeping for %v to simulate remainder of tick #%d.", tickRemaining, tickNumber)
		time.Sleep(tickRemaining)
	}

	// Sanity check to ensure that we issued the correct/expected number of ticks.
	if numTicksIssued != numTicksToIssue {
		panic(fmt.Sprintf("Expected to issue %d tick(s); instead, issued %d.", numTicksToIssue, numTicksIssued))
	}

	return nil
}

// ProcessWorkload accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
// This processes events in response to clock ticks.
//
// If there is a critical error that causes the workload to be terminated prematurely/aborted, then that error is returned.
// If the workload is able to complete successfully, then nil is returned.
//
// ProcessWorkload should be called from its own goroutine.
func (d *BasicWorkloadDriver) ProcessWorkload(wg *sync.WaitGroup) error {
	d.mu.Lock()

	d.workloadStartTime = time.Now()
	// Add an event for the workload starting.
	d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
		WithEventId(uuid.NewString()).
		WithSessionId("-").
		WithEventName(domain.EventWorkloadStarted).
		WithEventTimestamp(d.clockTime.GetClockTime()).
		WithProcessedAtTime(d.workloadStartTime))

	if d.workloadPreset != nil {
		d.logger.Info("Starting preset-based workload.", zap.String("workload-name", d.workload.WorkloadName()), zap.String("workload-preset-name", d.workloadPreset.GetName()), zap.String("workload-preset-key", d.workloadPreset.GetKey()))
	} else if d.workloadSessions != nil {
		d.logger.Info("Starting template-based workload.", zap.String("workload-name", d.workload.WorkloadName()))
	} else {
		d.logger.Info("Starting workload.", zap.String("workload-name", d.workload.WorkloadName()))
	}

	d.workloadGenerator = generator.NewWorkloadGenerator(d.opts, d.atom, d)
	d.mu.Unlock()

	if d.workload.IsPresetWorkload() {
		go d.workloadGenerator.GeneratePresetWorkload(d, d.workload, d.workload.(*domain.WorkloadFromPreset).WorkloadPreset, d.workloadRegistrationRequest)
	} else if d.workload.IsTemplateWorkload() {
		go d.workloadGenerator.GenerateTemplateWorkload(d, d.workload, d.workloadSessions, d.workloadRegistrationRequest)
	} else {
		panic(fmt.Sprintf("Workload is of presently-unsuporrted type: \"%s\" -- cannot generate workload.", d.workload.GetWorkloadType()))
	}

	d.logger.Info("The Workload Driver has started running.")

	numTicksServed := 0
	d.servingTicks.Store(true)
	for d.workload.IsRunning() {
		select {
		case tick := <-d.ticker.TickDelivery:
			{
				d.logger.Debug("Received tick.", zap.String("workload-id", d.workload.GetId()), zap.Time("tick", tick))

				// Handle the tick.
				if err := d.handleTick(tick); err != nil {
					d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
					d.handleCriticalError(err)

					// If this is non-nil, then call Done() to signal to the caller that the workload has finished (in this case, because of a critical error).
					if wg != nil {
						wg.Done()
					}

					return err
				}

				numTicksServed += 1
				if numTicksServed > 64 {
					// For now, workloads shouldn't go this long.
					// We can remove this once we're testing longer workloads.
					panic("Something is wrong. We've served 64 ticks.")
				}
			}
		case err := <-d.errorChan:
			{
				d.logger.Error("Received error.", zap.String("workload-id", d.workload.GetId()), zap.Error(err))
				d.handleCriticalError(err)
				if wg != nil {
					wg.Done()
				}
				return err // We're done, so we can return.
			}
		case <-d.stopChan:
			{
				d.logger.Info("Workload has been instructed to terminate early.", zap.String("workload-id", d.workload.GetId()))

				abortError := d.abortWorkload()
				if abortError != nil {
					d.logger.Error("Error while aborting workload.", zap.String("workload-id", d.workload.GetId()), zap.Error(abortError))
				}

				if wg != nil {
					wg.Done()
				}

				return abortError // We're done, so we can return.
			}
		case <-d.workloadExecutionCompleteChan: // This is placed after eventChan so that all events are processed first.
			{
				d.workloadComplete(wg)
				return nil
			}
		}
	}

	return nil
}

// workloadComplete is called when the BasicWorkloadDriver receives a signal on its workloadExecutionCompleteChan
// field informing it that the workload has completed successfully.
//
// workloadComplete accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
func (d *BasicWorkloadDriver) workloadComplete(wg *sync.WaitGroup) {
	d.workload.SetWorkloadCompleted()

	var ok bool
	d.workloadEndTime, ok = d.workload.GetEndTime() // time.Now()
	if !ok {
		panic("`ok` should have been `true`")
	}

	// Add an event for the workload stopping.
	d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
		WithEventId(uuid.NewString()).
		WithSessionId("-").
		WithEventName(domain.EventWorkloadComplete).
		WithEventTimestamp(d.clockTime.GetClockTime()).
		WithProcessedAtTime(d.workloadEndTime))

	d.logger.Info("The Workload Generator has finished generating events.", zap.String("workload-id", d.workload.GetId()))
	d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.GetStartTime())), zap.Any("workload-start-time", d.workload.GetStartTime()), zap.Any("workload-end-time", d.workloadEndTime))

	d.sugaredLogger.Debugf("There is/are %d sessions.", len(d.sessionConnections))
	for sessionId, sessionConnection := range d.sessionConnections {
		kernel := sessionConnection.Kernel()
		if kernel == nil {
			continue
		}

		stdout := kernel.Stdout()
		stderr := kernel.Stderr()

		d.sugaredLogger.Debugf("Stdout IOPub messages received by Session %s (%d):", sessionId, len(stdout))
		for _, stdoutMessage := range stdout {
			d.sugaredLogger.Debugf(stdoutMessage)
		}

		d.sugaredLogger.Debugf("Stderr IOPub messages received by Session %s (%d):", sessionId, len(stderr))
		for _, stderrMessage := range stderr {
			d.sugaredLogger.Debugf(stderrMessage)
		}
	}

	// d.workload.WorkloadState = domain.WorkloadFinished
	if wg != nil {
		wg.Done()
	}
}

// Handle a tick during the execution of a workload.
//
// This should just be called by BasicWorkloadDriver::ProcessWorkload.
//
// The 'tick' parameter is the clock time of the latest tick -- the tick that we're processing here.
//
// This only returns critical errors.
func (d *BasicWorkloadDriver) handleTick(tick time.Time) error {
	_, _, err := d.currentTick.IncreaseClockTimeTo(tick)
	if err != nil {
		return err
	}

	coloredOutput := ansi.Color(fmt.Sprintf("Serving tick: %v (processing everything up to %v)", tick, tick), "blue")
	d.logger.Debug(coloredOutput)
	d.ticksHandled.Add(1)

	// If there are no events processed this tick, then we still need to increment the clock time so we're in-line with the simulation.
	// Check if the current clock time is earlier than the start of the previous tick. If so, increment the clock time to the beginning of the tick.
	prevTickStart := tick.Add(-d.tickDuration)
	if d.clockTime.GetClockTime().Before(prevTickStart) {
		if _, _, err := d.incrementClockTime(prevTickStart); err != nil {
			return nil
		}
	}

	// Process "session ready" events.
	d.handleSessionReadyEvents(tick)

	// Process "start/stop training" events.
	d.processEventsForTick(tick)

	d.doneServingTick()

	return nil
}

// Called from BasicWorkloadDriver::ProcessWorkload at the end of serving a tick to signal to the Ticker/Trigger interface that the listener (i.e., the Cluster) is done.
func (d *BasicWorkloadDriver) doneServingTick() {
	numEventsEnqueued := d.eventQueue.Len()
	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf(">> [%v] Done serving tick. There is/are %d more session event(s) enqueued right now.", d.clockTime.GetClockTime(), numEventsEnqueued)
	}

	d.workload.TickCompleted(d.ticksHandled.Load(), d.clockTime.GetClockTime())
	d.ticker.Done()
}

// WorkloadExecutionCompleteChan returns the channel that is used to signal
// that the workload has successfully processed all events and is complete.
func (d *BasicWorkloadDriver) WorkloadExecutionCompleteChan() chan interface{} {
	return d.workloadExecutionCompleteChan
}

// WorkloadEventGeneratorCompleteChan returns the channel used to signal that the generators have submitted all events.
// Once all remaining, already-enqueued events have been processed, the workload will be complete.
func (d *BasicWorkloadDriver) WorkloadEventGeneratorCompleteChan() chan interface{} {
	return d.workloadEventGeneratorCompleteChan
}

// EventQueue returns the event queue for this workload.
func (d *BasicWorkloadDriver) EventQueue() domain.EventQueue {
	return d.eventQueue
}

// processEventsForTick processes events in chronological/simulation order.
// This accepts the "current tick" as an argument. The current tick is basically an upper-bound on the times for
// which we'll process an event. For example, if `tick` is 19:05:00, then we will process all cluster and session
// events with timestamps that are (a) equal to 19:05:00 or (b) come before 19:05:00. Any events with timestamps
// that come after 19:05:00 will not be processed until the next tick.
func (d *BasicWorkloadDriver) processEventsForTick(tick time.Time) {
	var (
		// Map from session ID to a slice of events that the session is supposed to process in this tick.
		sessionEventMap = make(map[string][]domain.Event)
		// Used to wait until all goroutines finish processing events for the sessions.
		waitGroup sync.WaitGroup
	)

	// Extract all the events for this tick.
	for d.eventQueue.HasEventsForTick(tick) {
		evt, ok := d.eventQueue.GetNextEvent(tick)

		if !ok {
			// Since 'HasEventsForTick' returned true, 'GetNextEvent' should return a valid value.
			// If it doesn't, then in theory we could just ignore it, but it shouldn't happen, so there's probably a bug.
			// Hence, we'll panic.
			panic(fmt.Sprintf("Expected to find valid event for tick %v.", tick))
		}

		// Get the list of events for the particular session, creating said list if it does not already exist.
		sessionId := evt.Data().(domain.PodData).GetPod()
		sessionEvents, ok := sessionEventMap[sessionId]
		if !ok {
			// If the slice of events doesn't exist already, then create it.
			sessionEvents = make([]domain.Event, 0, 1)
		}

		// Add the event to the slice of events for this session.
		sessionEvents = append(sessionEvents, evt.GetEvent())
		// Put the updated list back into the map.
		sessionEventMap[sessionId] = sessionEvents
	}

	// If we dequeued 0 events, then just return.
	if len(sessionEventMap) == 0 {
		return
	}

	// We'll create one goroutine per session that has events to be processed.
	// We'll use the WaitGroup to block until all sessions have had their events processed.
	waitGroup.Add(len(sessionEventMap))
	d.sugaredLogger.Debugf("Processing events for %d session(s) in tick %v.", len(sessionEventMap), tick)

	// Iterate over the session-event map, creating a goroutine to process each session's events.
	for sessionId, events := range sessionEventMap {
		d.sugaredLogger.Debugf("Number of events for Session %s: %d", sessionId, len(events))

		// Create a go routine to process all the events for the particular session.
		// This enables us to process events targeting multiple sessions in-parallel.
		go d.processEventsForSession(sessionId, events, len(sessionEventMap), &waitGroup, tick)
	}

	// Wait for all the goroutines to complete before returning.
	waitGroup.Wait()
	d.workload.UpdateTimeElapsed()
}

// Process the given events for the specified session during the specified tick.
// This is intended to be called within its own goroutine so that events for multiple sessions within the same tick can be processed concurrently by the driver.
func (d *BasicWorkloadDriver) processEventsForSession(sessionId string, events []domain.Event, numSessionsWithEventsToProcess int, waitGroup *sync.WaitGroup, tick time.Time) {
	for idx, event := range events {
		d.sugaredLogger.Debugf("Handling event %d/%d \"%s\" for session %s now...", idx+1, numSessionsWithEventsToProcess, sessionId, event.Name())
		err := d.handleEvent(event)

		// Record it as processed even if there was an error when processing the event.
		d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
			WithEventId(event.Id()).
			WithSessionId(event.SessionID()).
			WithEventName(event.Name()).
			WithEventTimestamp(event.Timestamp()).
			WithProcessedAtTime(time.Now()).
			WithError(err))

		if err != nil {
			d.sugaredLogger.Errorf("Failed to handle event %d/%d \"%s\" for session %s: %v", idx+1, numSessionsWithEventsToProcess, sessionId, event.Name(), err)
			d.errorChan <- err
			return // We just return immediately, as the workload is going to be aborted due to the error.
		}

		d.sugaredLogger.Debugf("Successfully handled event %d/%d: \"%s\"", idx+1, numSessionsWithEventsToProcess, event.Name())
	}

	d.sugaredLogger.Debugf("Finished processing %d event(s) for session \"%s\" in tick %v.", len(events), sessionId, tick)
	waitGroup.Done()
}

// Given a session ID, such as from the trace data, return the ID used internally.
//
// The internal ID includes the unique ID of this workload driver, in case multiple
// workloads from the same trace are being executed concurrently.
func (d *BasicWorkloadDriver) getInternalSessionId(traceSessionId string) string {
	return fmt.Sprintf("%s-%s", traceSessionId, d.id)
}

func (d *BasicWorkloadDriver) getOriginalSessionIdFromInternalSessionId(internalSessionId string) string {
	rightIndex := strings.LastIndex(internalSessionId, "-")
	return internalSessionId[0:rightIndex]
}

// NewSession create and return a new Session with the given ID.
func (d *BasicWorkloadDriver) NewSession(id string, meta domain.SessionMetadata, createdAtTime time.Time) domain.WorkloadSession {
	d.sugaredLogger.Debugf("Creating new Session %v. MaxSessionCPUs: %.2f; MaxSessionMemory: %.2f. MaxSessionGPUs: %d. TotalNumSessions: %d", id, meta.GetMaxSessionCPUs(), meta.GetMaxSessionMemory(), meta.GetMaxSessionGPUs(), d.sessions.Len())

	// Make sure the Session doesn't already exist.
	var session domain.WorkloadSession
	if session = d.GetSession(id); session != nil {
		panic(fmt.Sprintf("Attempted to create existing Session %s.", id))
	}

	// The Session only exposes the CPUs, Memory, and
	resourceRequest := domain.NewResourceRequest(meta.GetMaxSessionCPUs(), meta.GetMaxSessionMemory(), meta.GetMaxSessionGPUs(), AnyGPU)
	session = domain.NewWorkloadSession(id, meta, resourceRequest, createdAtTime, d.atom)

	internalSessionId := d.getInternalSessionId(session.GetId())

	d.workload.SessionCreated(id)

	d.mu.Lock()
	defer d.mu.Unlock()
	d.Stats().TotalNumSessions += 1
	d.seenSessions[internalSessionId] = struct{}{}
	d.sessions.Set(internalSessionId, session)

	return session
}

// GetSession gets and returns the Session identified by the given ID, if one exists. Otherwise, return nil.
// If the caller is attempting to retrieve a Session that once existed but has since been terminated, then
// this will return nil.
//
// id should be the internal id of the session.
func (d *BasicWorkloadDriver) GetSession(id string) domain.WorkloadSession {
	d.mu.Lock()
	defer d.mu.Unlock()

	session, ok := d.sessions.Get(id)

	if ok {
		return session.(domain.WorkloadSession)
	}

	return nil
}

// handleSessionReadyEvent handles a single EventSessionReady domain.Event.
// This function is thread-safe and may be called within its own goroutine.
func (d *BasicWorkloadDriver) handleSessionReadyEvent(sessionReadyEvent domain.Event, eventIndex int, wg *sync.WaitGroup) {
	sessionMeta := sessionReadyEvent.Data().(domain.SessionMetadata)

	sessionId := sessionMeta.GetPod()
	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf("Handling EventSessionReady %d targeting Session %s [ts: %v].", eventIndex+1, sessionId, sessionReadyEvent.Timestamp())
	}

	resourceSpec := &jupyter.ResourceSpec{
		Cpu: int(sessionMeta.GetMaxSessionCPUs()),
		Mem: sessionMeta.GetMaxSessionMemory(),
		Gpu: sessionMeta.GetMaxSessionGPUs(),
	}

	provisionStart := time.Now()
	_, err := d.provisionSession(sessionId, sessionMeta, sessionReadyEvent.Timestamp(), resourceSpec)

	// The event index will be populated automatically by the ProcessedEvent method.
	// d.workload.ProcessedEvent(domain.NewWorkloadEvent(-1, sessionReadyEvent.Id(), domain.EventSessionStarted.String(), sessionReadyEvent.SessionID(), sessionReadyEvent.Timestamp().String(), time.Now().String(), (err == nil), err))
	workloadEvent := domain.NewEmptyWorkloadEvent().
		WithEventId(sessionReadyEvent.Id()).
		WithEventName(domain.EventSessionStarted).
		WithSessionId(sessionReadyEvent.SessionID()).
		WithEventTimestamp(sessionReadyEvent.Timestamp()).
		WithProcessedAtTime(time.Now()).
		WithProcessedStatus(err == nil).
		WithSimProcessedAtTime(d.clockTime.GetClockTime()).
		WithError(err)
	d.workload.ProcessedEvent(workloadEvent) // this is thread-safe

	if err != nil {
		d.logger.Error("Failed to provision new Jupyter session.", zap.String(ZapInternalSessionIDKey, sessionId), zap.Duration("real-time-elapsed", time.Since(provisionStart)), zap.Error(err))
		payload, _ := json.Marshal(domain.ErrorMessage{
			Description:  reflect.TypeOf(err).Name(),
			ErrorMessage: err.Error(),
			Valid:        true,
		})

		// This is thread-safe because the WebSocket uses a thread-safe wrapper.
		if writeError := d.websocket.WriteMessage(websocket.BinaryMessage, payload); writeError != nil {
			d.logger.Error("Failed to write error message via WebSocket.", zap.Error(writeError))
		}

		d.errorChan <- err
	} else {
		d.logger.Debug("Successfully handled SessionStarted event.", zap.String(ZapInternalSessionIDKey, sessionId), zap.Duration("real-time-elapsed", time.Since(provisionStart)))
	}

	if wg != nil {
		wg.Done()
	}

}

// Schedule new Sessions onto Hosts.
func (d *BasicWorkloadDriver) handleSessionReadyEvents(latestTick time.Time) {
	sessionReadyEvents := d.eventQueue.GetAllSessionStartEventsForTick(latestTick, -1 /* return all ready events */)
	if len(sessionReadyEvents) == 0 {
		return // No events to process, so just return immediately.
	}

	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf("[%v] Handling %d EventSessionReady events now.",
			d.clockTime.GetClockTime(), len(sessionReadyEvents))
	}

	var wg sync.WaitGroup
	wg.Add(len(sessionReadyEvents))

	// We'll process the 'session-ready' events in-parallel.
	st := time.Now()
	for idx, sessionReadyEvent := range sessionReadyEvents {
		go d.handleSessionReadyEvent(sessionReadyEvent, idx, &wg)
	}

	wg.Wait()

	d.sugaredLogger.Debugf("Finished processing %d events in %v.", len(sessionReadyEvents), time.Since(st))
}

// handleEvent processes a single domain.Event.
func (d *BasicWorkloadDriver) handleEvent(evt domain.Event) error {
	traceSessionId := evt.Data().(domain.SessionMetadata).GetPod()
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

		d.workload.TrainingStarted(traceSessionId)
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
			d.workload.TrainingStopped(traceSessionId)
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

		// Attempt to update the Prometheus metrics for Session lifetime duration (in seconds).
		internalSessionId := d.getInternalSessionId(traceSessionId)
		session := d.GetSession(internalSessionId)
		if session == nil {
			d.logger.Error("Could not find WorkloadSession with specified ID.", zap.String("session_id", internalSessionId))
		} else {
			sessionLifetimeDuration := time.Since(session.GetCreatedAt())
			metrics.PrometheusMetricsWrapperInstance.WorkloadSessionLifetimeSeconds.
				With(prometheus.Labels{"workload_id": d.workload.GetId()}).
				Observe(float64(sessionLifetimeDuration.Seconds()))
		}

		d.workload.SessionStopped(traceSessionId)
		d.logger.Debug("Handled SessionStopped event.", zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
	}

	return nil
}

func (d *BasicWorkloadDriver) stopSession(sessionId string) error {
	d.logger.Debug("Stopping session.", zap.String("kernel-id", sessionId))
	return d.kernelManager.StopKernel(sessionId)
}

func (d *BasicWorkloadDriver) provisionSession(sessionId string, meta domain.SessionMetadata, createdAtTime time.Time, resourceSpec *jupyter.ResourceSpec) (*jupyter.SessionConnection, error) {
	d.logger.Debug("Creating new kernel.", zap.String("kernel-id", sessionId))
	st := time.Now()
	sessionConnection, err := d.kernelManager.CreateSession(
		sessionId, /*strings.ToLower(sessionId) */
		fmt.Sprintf("%s.ipynb", sessionId),
		"notebook", "distributed", resourceSpec)

	if err != nil {
		d.logger.Error("Failed to create session.", zap.String(ZapInternalSessionIDKey, sessionId))
		return nil, err
	}

	timeElapsed := time.Since(st)

	internalSessionId := d.getInternalSessionId(sessionId)

	d.mu.Lock()
	// d.sessionConnections[sessionId] = sessionConnection
	d.sessionConnections[internalSessionId] = sessionConnection
	d.mu.Unlock()

	d.logger.Debug("Successfully created new kernel.", zap.String("kernel-id", sessionId), zap.Duration("time-elapsed", timeElapsed), zap.String(ZapInternalSessionIDKey, internalSessionId))

	workloadSession := d.NewSession(sessionId, meta, createdAtTime)

	// ioPubHandler is a session-specific wrapper around the standard BasicWorkloadDriver::handleIOPubMessage method.
	// This returns true if the received IOPub message is a "stream" message and is parsed successfully.
	// Otherwise, this returns false.
	//
	// The return value is not really used.
	ioPubHandler := func(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
		//d.sugaredLogger.Debugf("Handling IOPub message targeting session \"%s\", kernel \"%s\".", sessionId, conn.KernelId())

		// Parse the IOPub message.
		// If it is a stream message, this will return a *parsedIoPubMessage variable.
		parsedIoPubMsgVal := d.handleIOPubMessage(conn, kernelMessage)

		if parsedIoPubMsg, ok := parsedIoPubMsgVal.(*parsedIoPubMessage); ok {
			switch parsedIoPubMsg.Stream {
			case "stdout":
				{
					workloadSession.AddStdoutIoPubMessage(parsedIoPubMsg.Text)
				}
			case "stderr":
				{
					workloadSession.AddStderrIoPubMessage(parsedIoPubMsg.Text)
				}
			default:
				d.logger.Error("Unexpected stream specified by IOPub message.", zap.String("stream", parsedIoPubMsg.Stream))
				return false
			}
			return true
		}

		return false
	}

	if err := sessionConnection.RegisterIoPubHandler(d.id, ioPubHandler); err != nil {
		d.logger.Warn("Failed to register IOPub message handler.", zap.String("id", d.id), zap.Error(err))
	}

	return sessionConnection, nil
}

type parsedIoPubMessage struct {
	Stream string
	Text   string
}

// handleIOPubMessage returns the extracted text.
// This is expected to be called within a session-specific wrapper.
//
// If the IOPub message is a "stream" message, then this returns a *parsedIoPubMessage
// wrapping the name of the stream and the message text.
func (d *BasicWorkloadDriver) handleIOPubMessage(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
	// We just want to extract the output from 'stream' IOPub messages.
	// We don't care about non-stream-type IOPub messages here, so we'll just return.
	if kernelMessage.GetHeader().MessageType != "stream" {
		return nil
	}

	content := kernelMessage.GetContent().(map[string]interface{})

	var (
		stream string
		text   string
		ok     bool
	)

	stream, ok = content["name"].(string)
	if !ok {
		d.logger.Warn("Content of IOPub message did not contain an entry with key \"name\" and value of type string.", zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("kernel-id", conn.KernelId()))
		return nil
	}

	text, ok = content["text"].(string)
	if !ok {
		d.logger.Warn("Content of IOPub message did not contain an entry with key \"text\" and value of type string.", zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("kernel-id", conn.KernelId()))
		return nil
	}

	return &parsedIoPubMessage{
		Stream: stream,
		Text:   text,
	}
}
