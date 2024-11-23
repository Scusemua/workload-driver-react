package workload

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"github.com/scusemua/workload-driver-react/m/v2/pkg/statistics"
	"github.com/shopspring/decimal"
	"math"
	"math/rand"
	"net/http"
	"path"
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
	"github.com/scusemua/workload-driver-react/m/v2/pkg/jupyter"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Used in generating random IDs for workloads.
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	ZapInternalSessionIDKey = "internal-session-id"
	ZapTraceSessionIDKey    = "trace-session-id"

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
	ErrUnknownEventType                    = errors.New("unknown or unsupported session/workload event type")
	ErrWorkloadAlreadyPaused               = errors.New("cannot pause workload as workload is already paused")
	ErrWorkloadAlreadyUnpaused             = errors.New("cannot unpause workload as workload is already unpause")
	ErrWorkloadNil                         = errors.New("workload is nil; cannot process nil workload")

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
	eventChan                          chan *domain.Event                    // Receives events from the Synthesizer.
	eventQueue                         *event_queue.EventQueue               // Maintains a queue of events to be processed for each session.
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
	targetTickDuration                 time.Duration                         // How long each tick is supposed to last. This is the tick interval/step rate of the simulation.
	targetTickDurationSeconds          int64                                 // Cached total number of seconds of targetTickDuration
	tickDurationsSecondsMovingWindow   *statistics.MovingStat                // Moving average of observed tick durations in seconds.
	tickDurationsAll                   []time.Duration                       // All tick durations from the entire workload.
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
	paused                             bool                                  // Paused indicates whether the workload has been paused.
	pauseMutex                         sync.Mutex
	pauseCond                          *sync.Cond

	// notifyCallback is a function used to send notifications related to this workload directly to the frontend.
	notifyCallback func(notification *proto.Notification)

	// onCriticalErrorOccurred is a handler that is called when a critical error occurs.
	// The onCriticalErrorOccurred handler is called in its own goroutine.
	onCriticalErrorOccurred domain.WorkloadErrorHandler

	// onNonCriticalErrorOccurred is a handler that is called when a non-critical error occurs.
	// The onNonCriticalErrorOccurred handler is called in its own goroutine.
	onNonCriticalErrorOccurred domain.WorkloadErrorHandler
}

func NewBasicWorkloadDriver(opts *domain.Configuration, performClockTicks bool, timescaleAdjustmentFactor float64,
	websocket domain.ConcurrentWebSocket, atom *zap.AtomicLevel, criticalErrorHandler domain.WorkloadErrorHandler,
	nonCriticalErrorHandler domain.WorkloadErrorHandler, notifyCallback func(notification *proto.Notification)) *BasicWorkloadDriver {

	jupyterAddress := path.Join(opts.InternalJupyterServerAddress, opts.JupyterServerBasePath)

	driver := &BasicWorkloadDriver{
		id:                                 GenerateWorkloadID(8),
		eventChan:                          make(chan *domain.Event),
		clockTrigger:                       clock.NewTrigger(),
		opts:                               opts,
		workloadExecutionCompleteChan:      make(chan interface{}, 1),
		workloadEventGeneratorCompleteChan: make(chan interface{}),
		stopChan:                           make(chan interface{}, 1),
		errorChan:                          make(chan error, 2),
		atom:                               atom,
		targetTickDuration:                 time.Second * time.Duration(opts.TraceStep),
		targetTickDurationSeconds:          opts.TraceStep,
		tickDurationsSecondsMovingWindow:   statistics.NewMovingStat(5),
		tickDurationsAll:                   make([]time.Duration, 0),
		driverTimescale:                    opts.DriverTimescale,
		kernelManager:                      jupyter.NewKernelSessionManager(jupyterAddress, true, atom, metrics.PrometheusMetricsWrapperInstance),
		sessionConnections:                 make(map[string]*jupyter.SessionConnection),
		performClockTicks:                  performClockTicks,
		eventQueue:                         event_queue.NewEventQueue(atom),
		stats:                              NewWorkloadStats(),
		sessions:                           hashmap.New(100),
		seenSessions:                       make(map[string]struct{}),
		websocket:                          websocket,
		timescaleAdjustmentFactor:          timescaleAdjustmentFactor,
		currentTick:                        clock.NewSimulationClock(),
		clockTime:                          clock.NewSimulationClock(),
		onCriticalErrorOccurred:            criticalErrorHandler,
		onNonCriticalErrorOccurred:         nonCriticalErrorHandler,
		notifyCallback:                     notifyCallback,
		paused:                             false,
	}

	driver.pauseCond = sync.NewCond(&driver.pauseMutex)

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
	}

	driver.workloadPresets = make(map[string]*domain.WorkloadPreset, len(presets))
	for _, preset := range presets {
		driver.workloadPresets[preset.GetKey()] = preset
	}

	if driver.onNonCriticalErrorOccurred != nil {
		driver.kernelManager.RegisterOnErrorHandler(func(sessionId string, kernelId string, err error) {
			err = fmt.Errorf("error occurred for kernel=%s,session=%s: %w", kernelId, sessionId, err)
			driver.onNonCriticalErrorOccurred(driver.id, err)
		})
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

// IsSessionBeingSampled returns true if the specified session was selected for sampling.
func (d *BasicWorkloadDriver) IsSessionBeingSampled(sessionId string) bool {
	return d.workload.IsSessionBeingSampled(sessionId)
}

// GetSampleSessionsPercentage returns the configured SampleSessionsPercentage parameter for the Workload.
func (d *BasicWorkloadDriver) GetSampleSessionsPercentage() float64 {
	return d.workload.GetSampleSessionsPercentage()
}

func (d *BasicWorkloadDriver) SubmitEvent(evt *domain.Event) {
	if !d.workload.IsSessionBeingSampled(evt.SessionID()) {
		return
	}

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

	d.logger.Debug("Creating new workload from preset.",
		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
		zap.String("workload-preset-name", d.workloadPreset.GetName()))

	basicWorkload := domain.NewWorkloadBuilder(d.atom).
		SetID(d.id).
		SetWorkloadName(workloadRegistrationRequest.WorkloadName).
		SetSeed(workloadRegistrationRequest.Seed).
		EnableDebugLogging(workloadRegistrationRequest.DebugLogging).
		SetTimescaleAdjustmentFactor(workloadRegistrationRequest.TimescaleAdjustmentFactor).
		SetRemoteStorageDefinition(workloadRegistrationRequest.RemoteStorageDefinition).
		SetSessionsSamplePercentage(workloadRegistrationRequest.SampleSessionsPercent).
		Build()

	workloadFromPreset := domain.NewWorkloadFromPreset(basicWorkload, d.workloadPreset)

	err := workloadFromPreset.SetSource(d.workloadPreset)

	if err != nil {
		return nil, err
	}

	return workloadFromPreset, nil
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
	d.logger.Debug("Creating new workload from template.", zap.String("workload_name", workloadRegistrationRequest.WorkloadName))

	basicWorkload := domain.NewWorkloadBuilder(d.atom).
		SetID(d.id).
		SetWorkloadName(workloadRegistrationRequest.WorkloadName).
		SetSeed(workloadRegistrationRequest.Seed).
		EnableDebugLogging(workloadRegistrationRequest.DebugLogging).
		SetTimescaleAdjustmentFactor(workloadRegistrationRequest.TimescaleAdjustmentFactor).
		SetRemoteStorageDefinition(workloadRegistrationRequest.RemoteStorageDefinition).
		Build()

	workloadFromTemplate := domain.NewWorkloadFromTemplate(basicWorkload, d.workloadRegistrationRequest.Sessions)
	err := workloadFromTemplate.SetSource(workloadRegistrationRequest.Sessions)
	if err != nil {
		return nil, err
	}

	return workloadFromTemplate, nil
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

	d.logger.Debug("Registering workload.",
		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
		zap.String("workload-key", workloadRegistrationRequest.Key),
		zap.String("workload_registration_request", workloadRegistrationRequest.String()))

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
			d.logger.Error("Unsupported workload type.",
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.String("workload_id", d.workload.GetId()),
				zap.String("workload-type", workloadRegistrationRequest.Type))
			return nil, fmt.Errorf("%w: \"%s\"", ErrUnsupportedWorkloadType, workloadRegistrationRequest.Type)
		}
	}

	if workload == nil {
		panic("Workload should not be nil at this point.")
	}

	if d.onCriticalErrorOccurred != nil {
		workload.RegisterOnCriticalErrorHandler(d.onCriticalErrorOccurred)
		d.logger.Debug("Registered critical error handler with workload.",
			zap.String("workload_id", workload.GetId()),
			zap.String("workload_name", workload.WorkloadName()),
			zap.String("workload_driver_id", d.id))
	} else {
		d.logger.Warn("No critical error handler configured on workload driver.",
			zap.String("workload_id", workload.GetId()),
			zap.String("workload_name", workload.WorkloadName()),
			zap.String("workload_driver_id", d.id))
	}

	if d.onNonCriticalErrorOccurred != nil {
		workload.RegisterOnNonCriticalErrorHandler(d.onNonCriticalErrorOccurred)
		d.logger.Debug("Registered non-critical error handler with workload.",
			zap.String("workload_id", workload.GetId()),
			zap.String("workload_name", workload.WorkloadName()),
			zap.String("workload_driver_id", d.id))
	} else {
		d.logger.Warn("No non-critical error handler configured on workload driver.",
			zap.String("workload_id", workload.GetId()),
			zap.String("workload_name", workload.WorkloadName()),
			zap.String("workload_driver_id", d.id))
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
	d.kernelManager.AddMetadata(jupyter.RemoteStorageDefinitionMetadataKey, d.workload.GetRemoteStorageDefinition())
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

	if !d.workload.IsInProgress() {
		return domain.ErrWorkloadNotRunning
	}

	d.logger.Debug("Stopping workload.", zap.String("workload_id", d.id), zap.String("workload-state", string(d.workload.GetWorkloadState())))
	d.stopChan <- struct{}{}
	d.logger.Debug("Sent 'STOP' instruction via BasicWorkloadDriver::stopChan.", zap.String("workload_id", d.id))

	endTime, err := d.workload.TerminateWorkloadPrematurely(d.clockTime.GetClockTime())
	if err != nil {
		d.logger.Error("Failed to stop workload.", zap.String("workload_id", d.id), zap.Error(err))
		return err
	}

	// d.workloadEndTime, _ = d.workload.GetEndTime()
	d.workloadEndTime = endTime

	d.logger.Debug("Successfully stopped workload.", zap.String("workload_id", d.id))
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

	d.logger.Error("Workload encountered a critical error and must abort.", zap.String("workload_id", d.id), zap.Error(err))
	if abortErr := d.abortWorkload(); abortErr != nil {
		d.logger.Error("Failed to abort workload.", zap.String("workload_id", d.workload.GetId()), zap.Error(abortErr))
	}

	d.workload.UpdateTimeElapsed()
	d.workload.SetWorkloadState(domain.WorkloadErred)
	d.workload.SetErrorMessage(err.Error())
}

// abortWorkload manually aborts the workload.
// Clean up any sessions/kernels that were created.
func (d *BasicWorkloadDriver) abortWorkload() error {
	d.logger.Warn("Aborting workload.", zap.String("workload_id", d.id))

	if d.workloadGenerator == nil {
		d.logger.Error("Cannot stop workload. Workload Generator is nil.", zap.String("workload_id", d.id))
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
		_, _, err := d.currentTick.IncreaseClockTimeTo(firstEvent.Timestamp)
		if err != nil {
			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
			return err
		}

		_, _, err = d.clockTime.IncreaseClockTimeTo(firstEvent.Timestamp)
		if err != nil {
			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
			return err
		}

		d.logger.Debug("Initialized current tick.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.Time("timestamp", firstEvent.Timestamp))
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
	d.logger.Info("Workload Simulator has started running. Bootstrapping simulation now.",
		zap.String("workload_id", d.id),
		zap.String("workload_name", d.workload.WorkloadName()))
	err := d.bootstrapSimulation()
	if err != nil {
		d.logger.Error("Failed to bootstrap simulation.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("reason", err.Error()))
		d.handleCriticalError(err)
		wg.Done()
		return
	}

	d.logger.Info("The simulation has started.",
		zap.String("workload_id", d.id),
		zap.String("workload_name", d.workload.WorkloadName()))

	nextTick := d.currentTick.GetClockTime().Add(d.targetTickDuration)

OUTER:
	for {
		// Constantly poll for events from the Workload Generator.
		// These events are then enqueued in the EventQueue.
		select {
		case evt := <-d.eventChan:
			if !d.workload.IsInProgress() {
				d.logger.Warn("Workload is no longer running. Aborting drive procedure.",
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.String("workload_id", d.workload.GetId()),
					zap.String("workload_state", d.workload.GetWorkloadState().String()))

				return
			}

			// If the event occurs during this tick, then call EnqueueEvent to enqueue the event in the EventQueue.
			if evt.Timestamp.Before(nextTick) {
				d.sugaredLogger.Debugf("\"%s\" event \"%s\" targeting session \"%s\" DOES occur before next tick [%v]. Enqueuing event now (timestamp=%v).",
					evt.Name.String(), evt.ID, evt.SessionID(), nextTick, evt.Timestamp)

				d.eventQueue.EnqueueEvent(evt)
			} else {
				d.sugaredLogger.Debugf("\"%s\" event \"%s\" targeting session \"%s\" does NOT occur before next tick [%v]. Will have to issue clock ticks until we get to event's timestamp of [%v].",
					evt.Name.String(), evt.ID, evt.SessionID(), nextTick, evt.Timestamp)

				// The event occurs in the next tick. Update the current tick clock, issue/perform a tick-trigger, and then process the event.
				err = d.issueClockTicks(evt.Timestamp)
				if err != nil {
					d.logger.Error("Critical error occurred while attempting to increment clock time.",
						zap.String("workload_id", d.id),
						zap.String("workload_name", d.workload.WorkloadName()),
						zap.String("error-message", err.Error()))
					d.handleCriticalError(err)
					break OUTER
				}
				nextTick = d.currentTick.GetClockTime().Add(d.targetTickDuration)
				d.eventQueue.EnqueueEvent(evt)
			}
		case <-d.workloadEventGeneratorCompleteChan:
			d.logger.Debug("Drivers finished generating events.",
				zap.Int("events_still_enqueued", d.eventQueue.Len()),
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()))

			// Continue issuing ticks until the cluster is finished.
			for d.eventQueue.Len() > 0 {
				err = d.issueClockTicks(nextTick)
				if err != nil {
					d.logger.Error("Critical error occurred while attempting to increment clock time.",
						zap.String("workload_id", d.id),
						zap.String("workload_name", d.workload.WorkloadName()),
						zap.String("error-message", err.Error()))
					d.handleCriticalError(err)
					break OUTER
				}

				nextTick = d.currentTick.GetClockTime().Add(d.targetTickDuration)
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

func (d *BasicWorkloadDriver) PauseWorkload() error {
	d.pauseMutex.Lock()
	defer d.pauseMutex.Unlock()

	if d.paused {
		d.logger.Error("Cannot pause workload. Workload is already paused.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))

		return ErrWorkloadAlreadyPaused
	}

	d.logger.Debug("Pausing workload.",
		zap.String("workload_id", d.id),
		zap.String("workload_name", d.workload.WorkloadName()))

	d.paused = true
	return d.workload.SetPausing()
}

func (d *BasicWorkloadDriver) UnpauseWorkload() error {
	d.pauseMutex.Lock()
	defer d.pauseMutex.Unlock()

	if !d.paused {
		d.logger.Error("Cannot unpause workload. Workload is already unpaused.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))

		return ErrWorkloadAlreadyUnpaused
	}

	d.logger.Debug("Unpausing workload.",
		zap.String("workload_id", d.id),
		zap.String("workload_name", d.workload.WorkloadName()))

	// We'll actually call d.workload.Unpause() later, in the goroutine triggering the clock ticks.
	d.paused = false
	d.pauseCond.Broadcast()
	return nil
}

// handlePause is called to check if the workload is paused and, if so, then block until we're unpaused.
func (d *BasicWorkloadDriver) handlePause() error {
	d.pauseMutex.Lock()
	defer d.pauseMutex.Unlock()

	pausedWorkload := false
	for d.paused {
		// If we haven't transitioned the workload from 'pausing' to 'paused' yet, then do so now.
		if !pausedWorkload {
			// Transition the workload from 'pausing' to 'paused'.
			// If this fails, then we'll just unpause and continue onwards.
			err := d.workload.SetPaused()
			if err != nil {
				d.logger.Error("Failed to transition workload to 'paused' state.",
					zap.String("workload_id", d.id),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.String("workload_state", d.workload.GetWorkloadState().String()),
					zap.Error(err))

				// We failed to pause the workload, so unpause ourselves and just continue.
				_ = d.workload.Unpause() // We don't care if this fails, as long as the workload is running.
				d.paused = false

				// Make sure that the workload is actively running.
				if !d.workload.IsRunning() {
					d.logger.Error("Workload is not actively running anymore. We're stuck.",
						zap.String("workload_id", d.id),
						zap.String("workload_name", d.workload.WorkloadName()),
						zap.String("workload_state", d.workload.GetWorkloadState().String()),
						zap.Error(err))
					return err
				}

				// Just return. We failed to pause the workload. Just try to keep going.
				return nil
			}

			pausedWorkload = true
		}

		d.logger.Debug("Workload is paused. Waiting to issue next tick.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))

		d.workload.PauseWaitBeginning()
		d.pauseCond.Wait()
	}

	// If we paused the workload, then let's unpause it before returning.
	if pausedWorkload {
		err := d.workload.Unpause()
		if err != nil {
			d.logger.Error("Failed to unpause workload.",
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.Error(err))

			// If the workload is not actively running, then it's in an unexpected state, so we'll return an error.
			// Otherwise, we'll just return nil and ignore the fact that we failed to unpause the workload.
			if !d.workload.IsRunning() {
				d.logger.Error("Workload is not actively running anymore. We're stuck.",
					zap.String("workload_id", d.id),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.String("workload_state", d.workload.GetWorkloadState().String()),
					zap.Error(err))

				// Return the error.
				return err
			}
		}
	}

	return nil
}

// issueClockTicks issues clock ticks until the d.currentTick clock has caught up to the given timestamp.
// The given timestamp should correspond to the timestamp of the next event generated by the Workload Generator.
// The Workload Simulation will simulate everything up until this next event is ready to process.
// Then we will continue consuming all events for the current tick in the SimulationDriver::DriveSimulation function.
func (d *BasicWorkloadDriver) issueClockTicks(timestamp time.Time) error {
	if !d.performClockTicks {
		return nil
	}

	currentTick := d.currentTick.GetClockTime()

	// We're going to issue clock ticks up until the specified timestamp.
	// Calculate how many ticks that requires so we can perform a quick sanity
	// check at the end to verify that we issued the correct number of ticks.
	numTicksToIssue := int64((timestamp.Sub(currentTick)) / d.targetTickDuration)

	// This is just for debugging/logging purposes.
	nextEventAtTime, err := d.eventQueue.GetTimestampOfNextReadyEvent()
	if err == nil {
		timeUntilNextEvent := nextEventAtTime.Sub(currentTick)
		numTicksTilNextEvent := int64(timeUntilNextEvent / d.targetTickDuration)

		d.logger.Debug("Preparing to issue clock ticks.",
			zap.Time("current_tick", currentTick),
			zap.Time("target_timestamp", timestamp),
			zap.Int64("num_ticks_to_issue", numTicksToIssue),
			zap.Time("next_event_timestamp", nextEventAtTime),
			zap.Duration("time_til_next_event", timeUntilNextEvent),
			zap.Int64("num_ticks_til_next_event", numTicksTilNextEvent),
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))
	} else {
		d.logger.Debug("Preparing to issue clock ticks. There are no events enqueued.",
			zap.Time("current_tick", currentTick),
			zap.Time("target_timestamp", timestamp),
			zap.Int64("num_ticks_to_issue", numTicksToIssue),
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))
	}

	// Issue clock ticks.
	var numTicksIssued int64 = 0
	for timestamp.After(currentTick) && d.workload.IsInProgress() {
		if err := d.handlePause(); err != nil {
			return err
		}

		tickStart := time.Now()

		// Increment the clock.
		tick, err := d.currentTick.IncrementClockBy(d.targetTickDuration)
		if err != nil {
			d.logger.Error("Error while incrementing clock time.",
				zap.Duration("tick-duration", d.targetTickDuration),
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.Error(err))
			return err
		}

		tickNumber := int(tick.Unix() / d.targetTickDurationSeconds)
		d.logger.Debug("Issuing tick.",
			zap.Int("tick_number", tickNumber),
			zap.Time("tick_timestamp", tick),
			zap.Int("num_events", d.eventQueue.Len()),
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))

		// Trigger the clock ticker, which will prompt the other goroutine within the workload driver to process events and whatnot for this tick.
		d.clockTrigger.Trigger(tick)
		numTicksIssued += 1
		currentTick = d.currentTick.GetClockTime()

		// If the workload is no longer running, then we'll just return.
		// This can happen if the user manually stopped the workload, or if the workload encountered a critical error.
		if !d.workload.IsInProgress() {
			d.logger.Warn("Workload is no longer running. Aborting post-issue-clock-tick procedure early.",
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.String("workload_id", d.workload.GetId()),
				zap.String("workload_state", d.workload.GetWorkloadState().String()))

			return nil
		}

		// How long the tick took to process.
		// If it took less than the target amount of time, then we'll sleep for a bit.
		tickElapsedBase := time.Since(tickStart)
		tickRemaining := time.Duration(d.timescaleAdjustmentFactor * float64(d.targetTickDuration-tickElapsedBase))

		// Verify that the issuing of the tick did not exceed the specified real-clock-time that a tick should last.
		// TODO: Handle this more elegantly, such as by decreasing the length of subsequent ticks or something?
		if tickRemaining < 0 {
			d.logger.Error("Issuing clock tick lasted too long.",
				zap.Int("tick_number", tickNumber),
				zap.Time("tick_timestamp", tick),
				zap.Duration("time_elapsed", tickElapsedBase),
				zap.Duration("target_tick_duration", d.targetTickDuration),
				zap.Float64("timescale_adjustment_factor", d.timescaleAdjustmentFactor),
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.String("workload_state", d.workload.GetWorkloadState().String()))
		} else {
			// Simulate the remainder of the tick -- however much time is left.
			d.logger.Debug("Sleeping to simulate remainder of tick.",
				zap.Int("tick_number", tickNumber),
				zap.Time("tick_timestamp", tick),
				zap.Duration("time_elapsed", tickElapsedBase),
				zap.Duration("target_tick_duration", d.targetTickDuration),
				zap.Float64("timescale_adjustment_factor", d.timescaleAdjustmentFactor),
				zap.Duration("sleep_time", tickRemaining),
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.String("workload_state", d.workload.GetWorkloadState().String()))
			time.Sleep(tickRemaining)
		}

		tickDuration := time.Since(tickStart)
		tickDurationSec := decimal.NewFromFloat(tickDuration.Seconds())
		d.checkForLongTick(tickNumber, tickDurationSec)

		// Update the average now, after we check if the tick was too long.
		d.tickDurationsSecondsMovingWindow.Add(tickDurationSec)
		d.tickDurationsAll = append(d.tickDurationsAll, tickDuration)
		d.workload.AddFullTickDuration(tickDuration)
	}

	// Sanity check to ensure that we issued the correct/expected number of ticks.
	if numTicksIssued != numTicksToIssue {
		time.Sleep(time.Second * 5)

		if d.workload.IsInProgress() {
			d.logger.Error("Issued incorrect number of ticks, and workload is still running.",
				zap.Int64("expected_ticks", numTicksToIssue),
				zap.Int64("ticks_issued", numTicksIssued),
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()))
		} else {
			d.logger.Warn("Issued incorrect number of ticks, but workload is no longer running, so that's probably why.",
				zap.Int64("expected_ticks", numTicksToIssue),
				zap.Int64("ticks_issued", numTicksIssued),
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()))
		}
	}

	return nil
}

// checkForLongTick checks if the last tick's duration was notably longer than the average tick duration.
// If so, then a warning notification is sent to the frontend to alert the user that the tick took a
// long time to process, and that something may be wrong.
func (d *BasicWorkloadDriver) checkForLongTick(tickNumber int, tickDurationSec decimal.Decimal) {
	// If there's at least minN tick durations in the window, then we'll check if the last tick was unusually long.
	// minN is either 3, or a smaller value if the window size is set to something smaller than 5.
	minN := math.Min(float64(d.tickDurationsSecondsMovingWindow.Window()), 3)
	if d.tickDurationsSecondsMovingWindow.N() < int64(minN) {
		return // Insufficient entries for a meaningful comparison
	}

	avgTickDurationSec := d.tickDurationsSecondsMovingWindow.Avg()
	stdDevTickDuration := d.tickDurationsSecondsMovingWindow.SampleStandardDeviation()

	if tickDurationSec.GreaterThanOrEqual(avgTickDurationSec.Mul(decimal.NewFromFloat(1.5))) {
		d.logger.Warn("Last tick took longer than expected.",
			zap.Int("tick_number", tickNumber),
			zap.String("tick_duration_sec", tickDurationSec.StringFixed(4)),
			zap.String("avg_tick_duration_sec", avgTickDurationSec.StringFixed(4)),
			zap.String("sample_std_dev_tick_dur_sec", stdDevTickDuration.StringFixed(4)),
			zap.Int64("moving_avg_window_size", d.tickDurationsSecondsMovingWindow.Window()))

		if d.notifyCallback != nil {
			d.notifyCallback(&proto.Notification{
				Id:    uuid.NewString(),
				Title: fmt.Sprintf("Tick #%d of Workload %s Took a Long Time", tickNumber, d.workload.GetId()),
				Message: fmt.Sprintf("Tick duration: %s seconds. Average tick duration: %s seconds. Standard deviation (of tick duration in seconds): %s seconds.",
					tickDurationSec.StringFixed(3), avgTickDurationSec.StringFixed(3), stdDevTickDuration.StringFixed(3)),
				Panicked:         false,
				NotificationType: domain.WarningNotification.Int32(),
			})
		}
	}
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

	if d.workload == nil {
		d.logger.Error("Workload is nil. Cannot process it.")
		return ErrWorkloadNil
	}

	d.workloadStartTime = time.Now()
	// Add an event for the workload starting.
	d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
		WithEventId(uuid.NewString()).
		WithSessionId("-").
		WithEventName(domain.EventWorkloadStarted).
		WithEventTimestamp(d.clockTime.GetClockTime()).
		WithProcessedAtTime(d.workloadStartTime))

	if d.workloadPreset != nil {
		d.logger.Info("Starting preset-based workload.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("workload-preset-name", d.workloadPreset.GetName()),
			zap.String("workload-preset-key", d.workloadPreset.GetKey()))
	} else if d.workloadSessions != nil {
		d.logger.Info("Starting template-based workload.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))
	} else {
		d.logger.Info("Starting workload.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()))
	}

	d.workloadGenerator = generator.NewWorkloadGenerator(d.opts, d.atom, d)
	d.mu.Unlock()

	if d.workload.IsPresetWorkload() {
		go func() {
			presetWorkload := d.workload.(*domain.WorkloadFromPreset)
			err := d.workloadGenerator.GeneratePresetWorkload(d, presetWorkload, presetWorkload.WorkloadPreset, d.workloadRegistrationRequest)
			if err != nil {
				d.logger.Error("Failed to drive/generate preset workload.",
					zap.String("workload_id", d.id),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.Error(err))
			}
		}()
	} else if d.workload.IsTemplateWorkload() {
		go func() {
			templateWorkload := d.workload.(*domain.WorkloadFromTemplate)
			err := d.workloadGenerator.GenerateTemplateWorkload(d, templateWorkload, d.workloadSessions, d.workloadRegistrationRequest)
			if err != nil {
				d.logger.Error("Failed to drive/generate templated workload.",
					zap.String("workload_id", d.id),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.Error(err))
			}
		}()
	} else {
		panic(fmt.Sprintf("Workload is of presently-unsuporrted type: \"%s\" -- cannot generate workload.", d.workload.GetWorkloadType()))
	}

	d.logger.Info("The Workload Driver has started running.",
		zap.String("workload_id", d.id),
		zap.String("workload_name", d.workload.WorkloadName()))

	numTicksServed := 0
	d.servingTicks.Store(true)
	for d.workload.IsInProgress() {
		select {
		case tick := <-d.ticker.TickDelivery:
			{
				d.logger.Debug("Received tick.",
					zap.String("workload_id", d.workload.GetId()),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.Time("tick", tick))

				// Handle the tick.
				if err := d.handleTick(tick); err != nil {
					d.logger.Error("Critical error occurred when attempting to increase clock time.",
						zap.String("workload_id", d.id),
						zap.String("workload_name", d.workload.WorkloadName()),
						zap.Error(err))
					d.handleCriticalError(err)

					// If this is non-nil, then call Done() to signal to the caller that the workload has finished (in this case, because of a critical error).
					if wg != nil {
						wg.Done()
					}

					return err
				}

				numTicksServed += 1
			}
		case err := <-d.errorChan:
			{
				d.logger.Error("Received error.",
					zap.String("workload_id", d.workload.GetId()),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.Error(err))
				d.handleCriticalError(err)
				if wg != nil {
					wg.Done()
				}
				return err // We're done, so we can return.
			}
		case <-d.stopChan:
			{
				d.logger.Info("Workload has been instructed to terminate early.",
					zap.String("workload_id", d.id),
					zap.String("workload_name", d.workload.WorkloadName()))

				abortError := d.abortWorkload()
				if abortError != nil {
					d.logger.Error("Error while aborting workload.",
						zap.String("workload_id", d.id),
						zap.String("workload_name", d.workload.WorkloadName()),
						zap.Error(abortError))
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

	d.logger.Info("The Workload Generator has finished generating events.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()))
	d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.GetStartTime())), zap.Any("workload-start-time", d.workload.GetStartTime()), zap.Any("workload-end-time", d.workloadEndTime), zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()))

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
	tickStart := time.Now()
	_, _, err := d.currentTick.IncreaseClockTimeTo(tick)
	if err != nil {
		return err
	}

	coloredOutput := ansi.Color(fmt.Sprintf("Serving tick: %v (processing everything up to %v)", tick, tick), "blue")
	d.logger.Debug(coloredOutput, zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()))
	d.ticksHandled.Add(1)

	// If there are no events processed this tick, then we still need to increment the clock time so we're in-line with the simulation.
	// Check if the current clock time is earlier than the start of the previous tick. If so, increment the clock time to the beginning of the tick.
	prevTickStart := tick.Add(-d.targetTickDuration)
	if d.clockTime.GetClockTime().Before(prevTickStart) {
		if _, _, err := d.incrementClockTime(prevTickStart); err != nil {
			return nil
		}
	}

	// TODO: Update this.

	// Process "session ready" events.
	//d.handleSessionReadyEvents(tick)

	// Process "start/stop training" events.
	d.processEventsForTick(tick)

	d.doneServingTick(tickStart)

	return nil
}

// Called from BasicWorkloadDriver::ProcessWorkload at the end of serving a tick to signal to the Ticker/Trigger interface that the listener (i.e., the Cluster) is done.
func (d *BasicWorkloadDriver) doneServingTick(tickStart time.Time) {
	tickDuration := time.Since(tickStart)
	tick := d.ticksHandled.Load()
	numEventsEnqueued := d.eventQueue.Len()
	d.workload.TickCompleted(tick, d.clockTime.GetClockTime())

	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf("[%v] Done serving tick #%d. "+
			"Real-world tick duration: %v. "+
			"Total time elapsed for workload %s: %v. "+
			"There is/are %d more session event(s) enqueued right now.",
			d.clockTime.GetClockTime(), tick, tickDuration, d.workload.GetId(), d.workload.GetTimeElapsed(), numEventsEnqueued)
	}

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
func (d *BasicWorkloadDriver) EventQueue() *event_queue.EventQueue {
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
		sessionEventMap = make(map[string][]*domain.Event)

		// 'Session Ready' events.
		sessionReadyEvents = make([]*domain.Event, 0)

		// Used to wait until all goroutines finish processing events for the sessions.
		waitGroup sync.WaitGroup
	)

	d.eventQueue.PrintEvents()

	// Extract all the "session-ready" events for this tick.
	for d.eventQueue.HasEventsForTick(tick) && d.eventQueue.Peek(tick).Name == domain.EventSessionReady {
		evt := d.eventQueue.Pop(tick)

		if evt == nil {
			// Since 'HasEventsForTick' returned true, 'Pop' should return a valid value.
			// If it doesn't, then in theory we could just ignore it, but it shouldn't happen, so there's probably a bug.
			// Hence, we'll panic.
			panic(fmt.Sprintf("Expected to find valid event for tick %v.", tick))
		}

		// Collect up all the "session-ready" events.
		if evt.Name == domain.EventSessionReady {
			sessionReadyEvents = append(sessionReadyEvents, evt)
		}
	}

	// Process any 'session-ready' events.
	if len(sessionReadyEvents) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(sessionReadyEvents))

		d.logger.Debug("Processing \"session-ready\" event(s).",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.Time("tick", tick),
			zap.Int("num_events", len(sessionReadyEvents)))

		for idx, sessionReadyEvent := range sessionReadyEvents {
			go d.handleSessionReadyEvent(sessionReadyEvent, idx, &wg)
		}

		wg.Wait()
	} else {
		d.logger.Debug("No \"session-ready\" events to process.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.Time("tick", tick))
	}

	// Extract all the "session-ready" events for this tick.
	for d.eventQueue.HasEventsForTick(tick) {
		evt := d.eventQueue.Pop(tick)

		if evt == nil {
			// Since 'HasEventsForTick' returned true, 'Pop' should return a valid value.
			// If it doesn't, then in theory we could just ignore it, but it shouldn't happen, so there's probably a bug.
			// Hence, we'll panic.
			panic(fmt.Sprintf("Expected to find valid event for tick %v.", tick))
		}

		// Collect up all the "session-ready" events.
		if evt.Name == domain.EventSessionReady {
			panic(fmt.Sprintf("Discovered \"%s\" event while processing non-\"%s\" events: %v", domain.EventSessionReady, domain.EventSessionReady, evt))
		}

		// Get the list of events for the particular session, creating said list if it does not already exist.
		sessionId := evt.Data.(domain.PodData).GetPod()
		sessionEvents, ok := sessionEventMap[sessionId]
		if !ok {
			// If the slice of events doesn't exist already, then create it.
			sessionEvents = make([]*domain.Event, 0, 1)
		}

		// Add the event to the slice of events for this session.
		sessionEvents = append(sessionEvents, evt)
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
	d.logger.Debug("Processing workload events.",
		zap.Int("num_sessions", len(sessionEventMap)),
		zap.Time("tick", tick),
		zap.String("workload_name", d.workload.WorkloadName()),
		zap.String("workload_id", d.workload.GetId()))

	// Iterate over the session-event map, creating a goroutine to process each session's events.
	for sessionId, events := range sessionEventMap {
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
func (d *BasicWorkloadDriver) processEventsForSession(sessionId string, events []*domain.Event, numSessionsWithEventsToProcess int, waitGroup *sync.WaitGroup, tick time.Time) {
	for idx, event := range events {
		d.logger.Debug("Handling workload event.",
			zap.Int("event_index", idx+1),
			zap.Int("total_events", numSessionsWithEventsToProcess),
			zap.String("session", sessionId),
			zap.String("event_name", event.Name.String()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("workload_id", d.workload.GetId()))
		err := d.handleEvent(event)

		// Record it as processed even if there was an error when processing the event.
		d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
			WithEventId(event.Id()).
			WithSessionId(event.SessionID()).
			WithEventName(event.Name).
			WithEventTimestamp(event.Timestamp).
			WithProcessedAtTime(time.Now()).
			WithError(err))

		if err != nil {
			d.logger.Error("Failed to handle event workload event.",
				zap.Int("event_index", idx+1),
				zap.Int("total_events", numSessionsWithEventsToProcess),
				zap.String("session", sessionId),
				zap.String("event_name", event.Name.String()),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.String("workload_id", d.workload.GetId()),
				zap.Error(err))
			d.errorChan <- err

			if d.onCriticalErrorOccurred != nil {
				go d.onCriticalErrorOccurred(d.workload.GetId(), err)
			}

			return // We just return immediately, as the workload is going to be aborted due to the error.
		}

		d.logger.Debug("Successfully handled workload event.",
			zap.Int("event_index", idx+1),
			zap.Int("total_events", numSessionsWithEventsToProcess),
			zap.String("session", sessionId),
			zap.String("event_name", event.Name.String()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("workload_id", d.workload.GetId()))
	}

	d.logger.Debug("Finished processing workload events for session during current tick.",
		zap.Time("current_tick", tick),
		zap.Int("total_events", numSessionsWithEventsToProcess),
		zap.String("session", sessionId),
		zap.String("workload_name", d.workload.WorkloadName()),
		zap.String("workload_id", d.workload.GetId()))

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

// newSession create and return a new Session with the given ID.
func (d *BasicWorkloadDriver) newSession(id string, meta domain.SessionMetadata, createdAtTime time.Time) Session {
	d.sugaredLogger.Debugf("Creating new Session %v. MaxSessionCPUs: %.2f; MaxSessionMemory: %.2f. MaxSessionGPUs: %d. MaxSessionVRAM: %.2f, TotalNumSessions: %d",
		id, meta.GetMaxSessionCPUs(), meta.GetMaxSessionMemory(), meta.GetMaxSessionGPUs(), meta.GetMaxSessionVRAM(), d.sessions.Len())

	// Make sure the Session doesn't already exist.
	var session Session
	if session = d.GetSession(id); session != nil {
		panic(fmt.Sprintf("Attempted to create existing Session %s.", id))
	}

	// The Session only exposes the CPUs, Memory, and
	resourceRequest := domain.NewResourceRequest(meta.GetMaxSessionCPUs(), meta.GetMaxSessionMemory(), meta.GetMaxSessionGPUs(), meta.GetMaxSessionVRAM(), AnyGPU)
	session = domain.NewWorkloadSession(id, meta, resourceRequest, createdAtTime, d.atom)

	internalSessionId := d.getInternalSessionId(session.GetId())

	d.workload.SessionCreated(id, meta)

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
func (d *BasicWorkloadDriver) GetSession(id string) Session {
	d.mu.Lock()
	defer d.mu.Unlock()

	session, ok := d.sessions.Get(id)

	if ok {
		return session.(Session)
	}

	return nil
}

// handleSessionReadyEvent handles a single EventSessionReady *domain.Event.
// This function is thread-safe and may be called within its own goroutine.
func (d *BasicWorkloadDriver) handleSessionReadyEvent(sessionReadyEvent *domain.Event, eventIndex int, wg *sync.WaitGroup) {
	sessionMeta := sessionReadyEvent.Data.(domain.SessionMetadata)

	sessionId := sessionMeta.GetPod()
	if d.sugaredLogger.Level() == zapcore.DebugLevel {
		d.sugaredLogger.Debugf("Handling EventSessionReady %d targeting Session %s [ts: %v].", eventIndex+1, sessionId, sessionReadyEvent.Timestamp)
	}

	resourceSpec := &jupyter.ResourceSpec{
		Cpu: int(sessionMeta.GetMaxSessionCPUs()),
		Mem: sessionMeta.GetMaxSessionMemory(),
		Gpu: sessionMeta.GetMaxSessionGPUs(),
	}

	provisionStart := time.Now()
	_, err := d.provisionSession(sessionId, sessionMeta, sessionReadyEvent.Timestamp, resourceSpec)

	// The event index will be populated automatically by the ProcessedEvent method.
	workloadEvent := domain.NewEmptyWorkloadEvent().
		WithEventId(sessionReadyEvent.Id()).
		WithEventName(domain.EventSessionStarted).
		WithSessionId(sessionReadyEvent.SessionID()).
		WithEventTimestamp(sessionReadyEvent.Timestamp).
		WithProcessedAtTime(time.Now()).
		WithProcessedStatus(err == nil).
		WithSimProcessedAtTime(d.clockTime.GetClockTime()).
		WithError(err)
	d.workload.ProcessedEvent(workloadEvent) // this is thread-safe

	// Handle the error from the above call to provisionSession.
	if err != nil {
		d.logger.Warn("Failed to provision new Jupyter session.",
			zap.String(ZapInternalSessionIDKey, sessionId),
			zap.Duration("real-time-elapsed", time.Since(provisionStart)),
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.Error(err))

		payload, _ := json.Marshal(domain.ErrorMessage{
			Description:  reflect.TypeOf(err).Name(),
			ErrorMessage: err.Error(),
			Valid:        true,
		})

		// This is thread-safe because the WebSocket uses a thread-safe wrapper.
		go func() {
			if writeError := d.websocket.WriteMessage(websocket.BinaryMessage, payload); writeError != nil {
				d.logger.Error("Failed to write error message via WebSocket.",
					zap.String("workload_id", d.workload.GetId()),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.Error(writeError))
			}
		}()

		// We need to inspect the error here.
		// Depending on what the error is, we'll treat it as a critical error or not.
		d.handleFailureToCreateNewSession(err, sessionReadyEvent)
	} else {
		d.logger.Debug("Successfully handled SessionStarted event.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, sessionId),
			zap.Duration("real-time-elapsed", time.Since(provisionStart)))
	}

	if wg != nil {
		wg.Done()
	}
}

func (d *BasicWorkloadDriver) delaySession(sessionId string, delayAmount time.Duration) {
	err := d.eventQueue.DelaySession(sessionId, delayAmount)
	if err != nil {
		panic(err)
	}

	d.workload.SessionDelayed(sessionId, delayAmount)
}

func (d *BasicWorkloadDriver) handleFailureToCreateNewSession(err error, sessionReadyEvent *domain.Event) {
	sessionId := sessionReadyEvent.SessionID()
	if strings.Contains(err.Error(), "insufficient hosts available") {
		//sessionReadyEvent.PushTimestampBack(d.targetTickDuration)

		d.logger.Warn("Failed to create session due to insufficient resources available. Will requeue event and try again later.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, sessionId),
			zap.Time("original_timestamp", sessionReadyEvent.OriginalTimestamp),
			zap.Time("current_timestamp", sessionReadyEvent.Timestamp),
			zap.Time("current_tick", d.currentTick.GetClockTime()),
			zap.Int32("num_times_enqueued", sessionReadyEvent.GetNumTimesEnqueued()),
			zap.Duration("total_delay", sessionReadyEvent.TotalDelay()))

		d.delaySession(sessionId, d.targetTickDuration*2)

		// Put the event back in the queue.
		d.eventQueue.EnqueueEvent(sessionReadyEvent)

		return
	}

	d.logger.Error("Session creation failure is due to unexpected reason. Aborting workload.",
		zap.String("workload_id", d.workload.GetId()),
		zap.String("workload_name", d.workload.WorkloadName()),
		zap.String(ZapInternalSessionIDKey, sessionId),
		zap.Error(err))

	d.errorChan <- err
	if d.onCriticalErrorOccurred != nil {
		go d.onCriticalErrorOccurred(d.workload.GetId(), err)
	}
}

// Schedule new Sessions onto Hosts.
//func (d *BasicWorkloadDriver) handleSessionReadyEvents(latestTick time.Time) {
//	sessionReadyEvents := d.eventQueue.GetAllSessionStartEventsForTick(latestTick, -1 /* return all ready events */)
//	if len(sessionReadyEvents) == 0 {
//		return // No events to process, so just return immediately.
//	}
//
//	if d.sugaredLogger.Level() == zapcore.DebugLevel {
//		d.sugaredLogger.Debugf("[%v] Handling %d EventSessionReady events now.",
//			d.clockTime.GetClockTime(), len(sessionReadyEvents))
//	}
//
//	var wg sync.WaitGroup
//	wg.Add(len(sessionReadyEvents))
//
//	// We'll process the 'session-ready' events in-parallel.
//	st := time.Now()
//	for idx, sessionReadyEvent := range sessionReadyEvents {
//		go d.handleSessionReadyEvent(sessionReadyEvent, idx, &wg)
//	}
//
//	wg.Wait()
//
//	d.sugaredLogger.Debugf("Finished processing %d events in %v.", len(sessionReadyEvents), time.Since(st))
//}

// handleUpdateGpuUtilizationEvent handles a 'update-gpu-util' event.
func (d *BasicWorkloadDriver) handleUpdateGpuUtilizationEvent(evt *domain.Event) error {
	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
	internalSessionId := d.getInternalSessionId(traceSessionId)

	d.logger.Debug("Received UpdateGpuUtil event.",
		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
	// TODO: Update GPU utilization.
	d.logger.Debug("Handled UpdateGpuUtil event.",
		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

	return nil
}

// createExecuteRequestArguments creates the arguments for an "execute_request" from the given event.
//
// The event must be of type "training-started", or this will return nil.
func (d *BasicWorkloadDriver) createExecuteRequestArguments(evt *domain.Event) *jupyter.RequestExecuteArgs {
	if evt.Name != domain.EventSessionTrainingStarted {
		d.logger.Error("Attempted to create \"execute_request\" arguments for event of invalid type.",
			zap.String("event_type", evt.Name.String()),
			zap.String("event_id", evt.Id()),
			zap.String("session_id", evt.SessionID()),
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()))

		return nil
	}

	sessionMetadata := evt.Data.(domain.SessionMetadata)

	gpus := sessionMetadata.GetCurrentTrainingMaxGPUs()
	if gpus == 0 && sessionMetadata.GetGPUs() > 0 {
		gpus = sessionMetadata.GetGPUs()
	}

	resourceRequest := &domain.ResourceRequest{
		Cpus:     sessionMetadata.GetCurrentTrainingMaxCPUs(),
		MemoryMB: sessionMetadata.GetCurrentTrainingMaxMemory(),
		VRAM:     sessionMetadata.GetVRAM(),
		Gpus:     gpus,
	}

	argsBuilder := jupyter.NewRequestExecuteArgsBuilder().
		Code(TrainingCode).
		Silent(false).
		StoreHistory(true).
		UserExpressions(nil).
		AllowStdin(true).
		StopOnError(false).
		AwaitResponse(false).
		AddMetadata("resource_request", resourceRequest)

	return argsBuilder.Build()
}

// handleTrainingStartedEvent handles a 'training-started' event.
func (d *BasicWorkloadDriver) handleTrainingStartedEvent(evt *domain.Event) error {
	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
	internalSessionId := d.getInternalSessionId(traceSessionId)

	d.logger.Debug("Received TrainingStarted event.",
		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
		zap.String("internal-session-id", internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

	if _, ok := d.seenSessions[internalSessionId]; !ok {
		d.logger.Error("Received 'training-started' event for unknown session.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("event", evt.String()),
			zap.String(ZapInternalSessionIDKey, internalSessionId),
			zap.String(ZapTraceSessionIDKey, traceSessionId))
		return ErrUnknownSession
	}

	sessionConnection, ok := d.sessionConnections[internalSessionId]
	if !ok {
		d.logger.Error("No session connection found for session upon receiving 'training-started' event.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		return ErrNoSessionConnection
	}

	kernelConnection := sessionConnection.Kernel()
	if kernelConnection == nil {
		d.logger.Error("No kernel connection found for session connection.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		return ErrNoKernelConnection
	}

	args := d.createExecuteRequestArguments(evt)
	err := kernelConnection.RequestExecute(args)

	if err != nil {
		d.logger.Error("Error while attempting to execute training code.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId), zap.Error(err))
		return err
	}

	d.workload.TrainingStarted(traceSessionId, evt)
	d.logger.Debug("Handled TrainingStarted event.",
		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

	return nil
}

// handleTrainingEndedEvent handles a 'training-stopped' event.
func (d *BasicWorkloadDriver) handleTrainingEndedEvent(evt *domain.Event) error {
	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
	internalSessionId := d.getInternalSessionId(traceSessionId)
	d.logger.Debug("Received TrainingEnded event.",
		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

	if _, ok := d.seenSessions[internalSessionId]; !ok {
		d.logger.Error("Received 'training-stopped' event for unknown session.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		return ErrUnknownSession
	}

	if _, ok := d.seenSessions[internalSessionId]; !ok {
		d.logger.Error("Received 'training-stopped' event for unknown session.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		return ErrUnknownSession
	}

	sessionConnection, ok := d.sessionConnections[internalSessionId]
	if !ok {
		d.logger.Error("No session connection found for session upon receiving 'training-stopped' event.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		return ErrNoSessionConnection
	}

	kernelConnection := sessionConnection.Kernel()
	if kernelConnection == nil {
		d.logger.Error("No kernel connection found for session connection.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
		return ErrNoKernelConnection
	}

	err := kernelConnection.StopRunningTrainingCode(true)
	if err != nil {
		d.logger.Error("Error while attempting to stop training.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId), zap.Error(err))
		return err
	} else {
		d.workload.TrainingStopped(traceSessionId, evt)
		d.logger.Debug("Successfully handled TrainingEnded event.",
			zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
	}

	return nil
}

// handleSessionStoppedEvent handles a 'session-stopped' event.
func (d *BasicWorkloadDriver) handleSessionStoppedEvent(evt *domain.Event) error {
	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
	internalSessionId := d.getInternalSessionId(traceSessionId)

	d.logger.Debug("Received SessionStopped event.",
		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

	// TODO: Test that this actually works.
	err := d.stopSession(d.getOriginalSessionIdFromInternalSessionId(internalSessionId))
	if err != nil {
		return err
	} else {
		d.logger.Debug("Successfully stopped session.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, internalSessionId),
			zap.String(ZapTraceSessionIDKey, traceSessionId))
	}

	// Attempt to update the Prometheus metrics for Session lifetime duration (in seconds).
	session := d.GetSession(internalSessionId)
	if session == nil {
		d.logger.Error("Could not find Session with specified ID.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("session_id", internalSessionId))
	} else {
		sessionLifetimeDuration := time.Since(session.GetCreatedAt())
		metrics.PrometheusMetricsWrapperInstance.WorkloadSessionLifetimeSeconds.
			With(prometheus.Labels{"workload_id": d.workload.GetId()}).
			Observe(sessionLifetimeDuration.Seconds())
	}

	d.workload.SessionStopped(traceSessionId, evt)
	d.logger.Debug("Handled SessionStopped event.",
		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))

	return nil
}

// handleEvent processes a single *domain.Event.
func (d *BasicWorkloadDriver) handleEvent(evt *domain.Event) error {
	switch evt.Name {
	case domain.EventSessionStarted:
		panic("Received SessionStarted event.")
	case domain.EventSessionTrainingStarted:
		return d.handleTrainingStartedEvent(evt)
	case domain.EventSessionUpdateGpuUtil:
		return d.handleUpdateGpuUtilizationEvent(evt)
	case domain.EventSessionTrainingEnded:
		return d.handleTrainingEndedEvent(evt)
	case domain.EventSessionStopped:
		return d.handleSessionStoppedEvent(evt)
	default:
		traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
		internalSessionId := d.getInternalSessionId(traceSessionId)

		d.logger.Error("Received event of unknown type.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("event_name", evt.Name.String()),
			zap.Time("event_timestamp", evt.Timestamp),
			zap.String("trace_session_id", traceSessionId),
			zap.String("session_id", internalSessionId))

		return fmt.Errorf("%w: \"%s\"", ErrUnknownEventType, evt.Name.String())
	}

	return nil
}

func (d *BasicWorkloadDriver) stopSession(sessionId string) error {
	d.logger.Debug("Stopping session.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.String("kernel_id", sessionId))
	return d.kernelManager.StopKernel(sessionId)
}

func (d *BasicWorkloadDriver) provisionSession(sessionId string, meta domain.SessionMetadata, createdAtTime time.Time, resourceSpec *jupyter.ResourceSpec) (*jupyter.SessionConnection, error) {
	d.logger.Debug("Creating new kernel.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.String("kernel_id", sessionId))
	st := time.Now()

	// Create the kernel in Jupyter.
	sessionConnection, err := d.kernelManager.CreateSession(
		sessionId, /*strings.ToLower(sessionId) */
		fmt.Sprintf("%s.ipynb", sessionId),
		"notebook", "distributed", resourceSpec)

	if err != nil {
		d.logger.Warn("Failed to create session.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String(ZapInternalSessionIDKey, sessionId),
			zap.Error(err))

		// We call our OnError handlers after returning; no need to call them here.
		return nil, err
	}

	timeElapsed := time.Since(st)

	internalSessionId := d.getInternalSessionId(sessionId)

	d.mu.Lock()
	d.sessionConnections[internalSessionId] = sessionConnection
	d.mu.Unlock()

	d.logger.Debug("Successfully created new kernel.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.String("kernel_id", sessionId), zap.Duration("time-elapsed", timeElapsed), zap.String(ZapInternalSessionIDKey, internalSessionId))

	// Create a new workload session.
	workloadSession := d.newSession(sessionId, meta, createdAtTime)

	// ioPubHandler is a session-specific wrapper around the standard BasicWorkloadDriver::handleIOPubMessage method.
	// This returns true if the received IOPub message is a "stream" message and is parsed successfully.
	// Otherwise, this returns false.
	//
	// The return value is not really used.
	ioPubHandler := func(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
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
				d.logger.Error("Unexpected stream specified by IOPub message.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.String("stream", parsedIoPubMsg.Stream))
				return false
			}
			return true
		}

		return false
	}

	if err := sessionConnection.RegisterIoPubHandler(d.id, ioPubHandler); err != nil {
		d.logger.Warn("Failed to register IOPub message handler.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.String("id", d.id), zap.Error(err))
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
		d.logger.Warn("Content of IOPub message did not contain an entry with key \"name\" and value of type string.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("kernel_id", conn.KernelId()))
		return nil
	}

	text, ok = content["text"].(string)
	if !ok {
		d.logger.Warn("Content of IOPub message did not contain an entry with key \"text\" and value of type string.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("kernel_id", conn.KernelId()))
		return nil
	}

	return &parsedIoPubMessage{
		Stream: stream,
		Text:   text,
	}
}
