package workload

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"github.com/scusemua/workload-driver-react/m/v2/pkg/statistics"
	"github.com/shopspring/decimal"
	"github.com/zhangjyr/gocsv"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	ErrTemplateFilePathNotSpecified        = errors.New("template file path was not specified")
	ErrInvalidTemplateFileSpecified        = errors.New("invalid template file path specified")
	ErrTrainingFailed                      = errors.New("training event could not be processed")
	ErrKernelCreationFailed                = errors.New("failed to create kernel")

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

type internalWorkload interface {
	domain.Workload

	// GetState returns the current State of the internalWorkload.
	GetState() State

	// SetState sets the State of the internalWorkload.
	SetState(State)

	// GetKind gets the Kind of internalWorkload (TRACE, PRESET, or TEMPLATE).
	GetKind() Kind

	// GetStatistics returns the Statistics struct of the internalWorkload.
	GetStatistics() *Statistics

	// UpdateStatistics provides an atomic mechanism to update the internalWorkload's Statistics.
	UpdateStatistics(func(stats *Statistics))

	RecordSessionExecutionTime(sessionId string, execTimeMillis int64)

	getSessionTrainingEvent(sessionId string, trainingIndex int) *domain.TrainingEvent
	unsafeSessionDiscarded(sessionId string) error
	unsafeSetSource(source interface{}) error
}

// BasicWorkloadDriver consumes events from the Workload Generator and takes action accordingly.
type BasicWorkloadDriver struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	clockTime                          domain.SimulationClock                     // Contains the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
	clockTrigger                       *clock.Trigger                             // Trigger for the clock ticks
	currentTick                        domain.SimulationClock                     // Contains the current tick of the workload.
	workloadExecutionCompleteChan      chan interface{}                           // Used to signal that the workload has successfully processed all events and is complete.
	workloadEventGeneratorCompleteChan chan interface{}                           // Used to signal that the generators have submitted all events. Once all remaining, already-enqueued events have been processed, the workload will be complete.
	driverTimescale                    float64                                    // Multiplier that impacts the timescale at the Driver will operate on with respect to the trace data. For example, if each tick is 60 seconds, then a DriverTimescale value of 0.5 will mean that each tick will take 30 seconds.
	errorChan                          chan error                                 // Used to stop the workload due to a critical error.
	eventChan                          chan *domain.Event                         // Receives events from the Synthesizer.
	eventQueue                         *event_queue.EventQueue                    // Maintains a queue of events to be processed for each session.
	getSchedulingPolicyCallback        func() (string, bool)                      // getSchedulingPolicyCallback is a callback to retrieve the configured scheduling policy of the cluster.
	schedulingPolicy                   string                                     // Cached scheduling policy value
	id                                 string                                     // Unique ID (relative to other drivers). The workload registered with this driver will be assigned this ID.
	kernelManager                      jupyter.KernelSessionManager               // Simplified Go implementation of the Jupyter JavaScript API.
	mu                                 sync.Mutex                                 // Synchronizes access to internal data structures. Can be locked externally using the Lock/Unlock API exposed by the WorkloadDriver.
	opts                               *domain.Configuration                      // The system's configuration, read from a file.
	performClockTicks                  bool                                       // If true, then we'll issue clock ticks. Otherwise, don't issue them. Mostly used for testing/debugging.
	servingTicks                       atomic.Bool                                // The WorkloadDriver::ServeTicks() method will continue looping as long as this flag is set to true.
	sessionConnections                 map[string]*jupyter.SessionConnection      // Map from internal session ID to session connection.
	sessionConnectionsMutex            sync.Mutex                                 // sessionConnections ensures atomic access to the sessionConnections map
	sessions                           *hashmap.HashMap                           // Responsible for creating sessions and maintaining a collection of all the sessions active within the simulation.
	stopChan                           chan interface{}                           // Used to stop the workload early/prematurely (i.e., before all events have been processed).
	targetTickDuration                 time.Duration                              // How long each tick is supposed to last. This is the tick interval/step rate of the simulation.
	targetTickDurationSeconds          int64                                      // Cached total number of seconds of targetTickDuration
	tickDurationsSecondsMovingWindow   *statistics.MovingStat                     // Moving average of observed tick durations in seconds.
	tickDurationsAll                   []time.Duration                            // All tick durations from the entire workload.
	ticker                             *clock.Ticker                              // Receive Tick events this way.
	ticksHandled                       atomic.Int64                               // Incremented/accessed atomically.
	timescaleAdjustmentFactor          float64                                    // Adjusts the timescale of the simulation. Setting this to 1 means that each tick is simulated as a whole minute. Setting this to 0.5 means each tick will be simulated for half its real time. So, if ticks are 60 seconds, and this variable is set to 0.5, then each tick will be simulated for 30 seconds before continuing to the next tick.
	websocket                          domain.ConcurrentWebSocket                 // Shared Websocket used to communicate with frontend.
	workload                           internalWorkload                           // The workload being driven by this driver.
	workloadStartTime                  time.Time                                  // The time at which the workload began.
	workloadEndTime                    time.Time                                  // The time at which the workload completed.
	workloadGenerator                  domain.WorkloadGenerator                   // The entity generating the workload (from trace data, a preset, or a template).
	workloadPreset                     *domain.WorkloadPreset                     // The preset used by the associated workload. Will only be non-nil if the associated workload is a preset-based workload, rather than a template-based workload.
	workloadPresets                    map[string]*domain.WorkloadPreset          // All the available workload presets.
	workloadRegistrationRequest        *domain.WorkloadRegistrationRequest        // The request that registered the workload that is being driven by this driver.
	workloadSessions                   []*domain.WorkloadTemplateSession          // The template used by the associated workload. Will only be non-nil if the associated workload is a template-based workload, rather than a preset-based workload.
	workloadSessionsMap                map[string]*domain.WorkloadTemplateSession // Map from Session ID to *domain.WorkloadTemplateSession
	paused                             bool                                       // Paused indicates whether the workload has been paused.
	trainingSubmittedTimes             *hashmap.HashMap                           // trainingSubmittedTimes keeps track of when "execute_request" messages were sent for different sessions. Keys are internal session IDs, values are unix millisecond timestamps.
	outputFile                         io.ReadWriteCloser                         // The opened .CSV output statistics file.
	outputFileDirectory                string                                     // outputFileDirectory is the directory where all the workload-specific output directories live
	outputFilePath                     string                                     // Path to the outputFile
	outputFileMutex                    sync.Mutex                                 // Atomic access to output file
	appendToOutputFile                 bool                                       // Flag that is set to true after the first write
	misbehavingSessions                map[string]interface{}                     // Map from session ID to sessions for sessions whose events we did not finish processing in a previous tick.
	misbehavingSessionsMutex           sync.Mutex                                 // misbehavingSessionsMutex ensures atomic access to the misbehavingSessions
	trainingStartedChannels            map[string]chan interface{}                // trainingStartedChannels are channels used to notify that training has started
	trainingStartedChannelMutex        sync.Mutex                                 // trainingStartedChannelMutex ensures atomic access to the trainingStartedChannels
	trainingStoppedChannels            map[string]chan interface{}                // trainingStartedChannels are channels used to notify that training has ended
	trainingStoppedChannelsMutex       sync.Mutex                                 // trainingStoppedChannelsMutex ensures atomic access to the trainingStoppedChannels
	workloadOutputInterval             time.Duration                              // workloadOutputInterval defines how often we should collect and write workload output statistics to the CSV file
	clients                            map[string]*Client
	clientsWaitGroup                   sync.WaitGroup

	pauseMutex sync.Mutex
	pauseCond  *sync.Cond

	// refreshClusterStatistics is used to fresh the ClusterStatistics from the Cluster Gateway.
	refreshClusterStatistics ClusterStatisticsRefresher

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
	websocket domain.ConcurrentWebSocket, atom *zap.AtomicLevel, callbackProvider CallbackProvider) *BasicWorkloadDriver {

	jupyterAddress := path.Join(opts.InternalJupyterServerAddress, opts.JupyterServerBasePath)

	driver := &BasicWorkloadDriver{
		id:                                 GenerateWorkloadID(8),
		eventChan:                          make(chan *domain.Event),
		outputFileDirectory:                opts.WorkloadOutputDirectory,
		clockTrigger:                       clock.NewTrigger(),
		opts:                               opts,
		workloadExecutionCompleteChan:      make(chan interface{}, 1),
		workloadEventGeneratorCompleteChan: make(chan interface{}),
		stopChan:                           make(chan interface{}, 1),
		errorChan:                          make(chan error, 2),
		misbehavingSessions:                make(map[string]interface{}),
		atom:                               atom,
		trainingStartedChannels:            make(map[string]chan interface{}),
		trainingStoppedChannels:            make(map[string]chan interface{}),
		targetTickDuration:                 time.Second * time.Duration(opts.TraceStep),
		targetTickDurationSeconds:          opts.TraceStep,
		tickDurationsSecondsMovingWindow:   statistics.NewMovingStat(5),
		tickDurationsAll:                   make([]time.Duration, 0),
		driverTimescale:                    opts.DriverTimescale,
		sessionConnections:                 make(map[string]*jupyter.SessionConnection),
		performClockTicks:                  performClockTicks,
		eventQueue:                         event_queue.NewEventQueue(atom),
		trainingSubmittedTimes:             hashmap.New(100),
		sessions:                           hashmap.New(100),
		websocket:                          websocket,
		timescaleAdjustmentFactor:          timescaleAdjustmentFactor,
		currentTick:                        clock.NewSimulationClock(),
		clockTime:                          clock.NewSimulationClock(),
		onCriticalErrorOccurred:            callbackProvider.HandleCriticalWorkloadError,
		onNonCriticalErrorOccurred:         callbackProvider.HandleWorkloadError,
		notifyCallback:                     callbackProvider.SendNotification,
		refreshClusterStatistics:           callbackProvider.RefreshAndClearClusterStatistics,
		getSchedulingPolicyCallback:        callbackProvider.GetSchedulingPolicy,
		paused:                             false,
		clients:                            make(map[string]*Client),
		workloadOutputInterval:             time.Second * time.Duration(opts.WorkloadOutputIntervalSec),
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

	driver.kernelManager = jupyter.NewKernelSessionManager(jupyterAddress, true, atom, driver)

	if driver.onNonCriticalErrorOccurred != nil {
		driver.kernelManager.RegisterOnErrorHandler(func(sessionId string, kernelId string, err error) {
			err = fmt.Errorf("error occurred for kernel=%s,session=%s: %w", kernelId, sessionId, err)
			driver.onNonCriticalErrorOccurred(driver.id, err)
		})
	}

	return driver
}

//// GetStatisticsFileOutputPath returns the path to the statistics CSV file.
//func (d *BasicWorkloadDriver) GetStatisticsFileOutputPath() string {
//	return d.outputFilePath
//}
//
//func (d *BasicWorkloadDriver) GetOutputFile() io.ReadWriteCloser {
//	return d.outputFile
//}

func (d *BasicWorkloadDriver) GetOutputFileContents() ([]byte, error) {
	d.outputFileMutex.Lock()
	defer d.outputFileMutex.Unlock()

	if d.outputFile == nil {
		d.logger.Warn("Cannot return contents of output file. It is nil (i.e., hasn't been created yet).",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()))
		return []byte{}, nil
	}

	csvBuffer, err := io.ReadAll(d.outputFile)

	if err != nil {
		d.logger.Warn("Failed to read contents of workload statistics file. Will try opening the file explicitly.",
			zap.String("workload_id", d.workload.GetId()),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.Error(err))

		outputFile, err := os.Open(d.outputFilePath)
		if err != nil {
			d.logger.Error("Failed to open workload statistics file explicitly.",
				zap.String("workload_id", d.workload.GetId()),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.Error(err))

			return nil, err
		}

		csvBuffer, err = io.ReadAll(outputFile)
		if err != nil {
			d.logger.Error("Failed to read contents of workload statistics file after opening it explicitly.",
				zap.String("workload_id", d.workload.GetId()),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.Error(err))

			return nil, err
		}

		_ = outputFile.Close()
	}

	return csvBuffer, nil
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

	d.logger.Debug("Submitting session-level event.",
		zap.String("session_id_field", evt.SessionId),
		zap.String("session_id", evt.SessionID()),
		zap.String("event_name", evt.Name.String()),
		zap.Time("event_timestamp", evt.Timestamp))

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
func (d *BasicWorkloadDriver) createWorkloadFromPreset(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (internalWorkload, error) {
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

	basicWorkload := NewBuilder(d.atom).
		SetID(d.id).
		SetWorkloadName(workloadRegistrationRequest.WorkloadName).
		SetSeed(workloadRegistrationRequest.Seed).
		EnableDebugLogging(workloadRegistrationRequest.DebugLogging).
		SetTimescaleAdjustmentFactor(workloadRegistrationRequest.TimescaleAdjustmentFactor).
		SetRemoteStorageDefinition(workloadRegistrationRequest.RemoteStorageDefinition).
		SetSessionsSamplePercentage(workloadRegistrationRequest.SessionsSamplePercentage).
		Build()

	workloadFromPreset := NewWorkloadFromPreset(basicWorkload, d.workloadPreset)

	err := workloadFromPreset.SetSource(d.workloadPreset)

	if err != nil {
		return nil, err
	}

	return workloadFromPreset, nil
}

// loadWorkloadTemplateFromFile is used for workload templates that come pre-defined
// on the server because of how large they are.
func (d *BasicWorkloadDriver) loadWorkloadTemplateFromFile(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) error {
	if workloadRegistrationRequest.TemplateFilePath == "" {
		return ErrTemplateFilePathNotSpecified
	}

	d.logger.Debug("Registering workload from pre-loaded template.",
		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath))

	templateJsonFile, err := os.Open(workloadRegistrationRequest.TemplateFilePath)
	if err != nil {
		return fmt.Errorf("%w: %w: \"%s\"",
			ErrInvalidTemplateFileSpecified, err, workloadRegistrationRequest.TemplateFilePath)
	}
	defer func() {
		err := templateJsonFile.Close()
		if err != nil {
			d.logger.Warn("Error while closing workload template file.",
				zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
				zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
				zap.Error(err))
		}
	}()

	d.logger.Debug("Creating new workload from template file.",
		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath))

	st := time.Now()

	// Read the file contents
	templateJsonFileContents, err := io.ReadAll(templateJsonFile)
	if err != nil {
		d.logger.Error("Failed to read workload template file.",
			zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
			zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
			zap.Error(err))
		return err
	}

	d.logger.Debug("Successfully read workload from template file.",
		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
		zap.Duration("time_elapsed", time.Since(st)),
		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath))

	// Unmarshal JSON into the struct
	var unmarshalledRegistrationRequest *domain.WorkloadRegistrationRequest
	err = json.Unmarshal(templateJsonFileContents, &unmarshalledRegistrationRequest)
	if err != nil {
		d.logger.Error("Failed to unmarshal workload template from file.",
			zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
			zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
			zap.Error(err))
		return err
	}

	d.logger.Debug("Successfully loaded workload from template.",
		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
		zap.Duration("time_elapsed", time.Since(st)),
		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
		zap.Int("num_sessions", len(unmarshalledRegistrationRequest.Sessions)))

	workloadRegistrationRequest.Sessions = unmarshalledRegistrationRequest.Sessions

	return nil
}

// Create a workload that was created using a template.
func (d *BasicWorkloadDriver) createWorkloadFromTemplate(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (*Template, error) {
	// The workload request needs to have a workload template in it.
	// If the registration request does not contain a workload template,
	// then the request is invalid, and we'll return an error.

	if workloadRegistrationRequest.Sessions == nil || len(workloadRegistrationRequest.Sessions) == 0 {
		if workloadRegistrationRequest.TemplateFilePath == "" {
			d.logger.Error("Workload Registration Request for template-based workload is missing the sessions or template file path!")
			return nil, ErrWorkloadRegistrationMissingTemplate
		}

		err := d.loadWorkloadTemplateFromFile(workloadRegistrationRequest)
		if err != nil {
			return nil, err
		}
	}

	d.logger.Debug("Creating new workload from template.",
		zap.String("workload_name", workloadRegistrationRequest.WorkloadName))

	d.workloadSessions = workloadRegistrationRequest.Sessions
	d.workloadSessionsMap = make(map[string]*domain.WorkloadTemplateSession, len(d.workloadSessions))
	for _, session := range d.workloadSessions {
		d.workloadSessionsMap[session.Id] = session
	}

	d.workloadRegistrationRequest = workloadRegistrationRequest
	basicWorkload := NewBuilder(d.atom).
		SetID(d.id).
		SetWorkloadName(workloadRegistrationRequest.WorkloadName).
		SetSeed(workloadRegistrationRequest.Seed).
		EnableDebugLogging(workloadRegistrationRequest.DebugLogging).
		SetTimescaleAdjustmentFactor(workloadRegistrationRequest.TimescaleAdjustmentFactor).
		SetRemoteStorageDefinition(workloadRegistrationRequest.RemoteStorageDefinition).
		SetSessionsSamplePercentage(workloadRegistrationRequest.SessionsSamplePercentage).
		Build()

	workloadFromTemplate, err := NewWorkloadFromTemplate(basicWorkload, workloadRegistrationRequest.Sessions)
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
		zap.String("workload-key", workloadRegistrationRequest.Key))

	// We create the workload a little differently depending on its type (either 'preset' or 'template').
	// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
	// have properties that the user can specify and change before submitting the workload for registration.
	var (
		// If this is created successfully, then d.workload will be assigned the value of this variable.
		workload internalWorkload
		err      error // If the workload is not created successfully, then we'll return this error.
	)
	switch strings.ToLower(workloadRegistrationRequest.Type) {
	case "preset":
		{
			// Preset-workload-specific workload creation and initialization steps.
			workload, err = d.createWorkloadFromPreset(workloadRegistrationRequest)

			if err != nil {
				d.logger.Error("Failed to create workload from preset.",
					zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
					zap.Error(err))
				return nil, err
			}
		}
	case "template":
		{
			// Template-workload-specific workload creation and initialization steps.
			workload, err = d.createWorkloadFromTemplate(workloadRegistrationRequest)

			if err != nil {
				d.logger.Error("Failed to create workload from template.",
					zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
					zap.Error(err))
				return nil, err
			}
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
	return d.workload.GetState() == Finished
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

	d.logger.Debug("Stopping workload.", zap.String("workload_id", d.id), zap.String("workload-state", string(d.workload.GetState())))
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
	d.workload.SetState(Erred)
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

// incrementClockTime sets the d.clockTime clock to the given timestamp, verifying that the new timestamp is either
// equal to or occurs after the old one.
//
// incrementClockTime returns a tuple where the first element is the new time, and the second element is the difference
// between the new time and the old time.
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

// publishStatisticsReport writes the current Statistics struct attached to the internalWorkload to the CSV file.
func (d *BasicWorkloadDriver) publishStatisticsReport() {
	clusterStatistics, err := d.refreshClusterStatistics(true, false)
	if err != nil {
		d.logger.Error("Failed to refresh cluster statistics.", zap.Error(err))

		if d.notifyCallback != nil {
			go d.notifyCallback(&proto.Notification{
				Id:    uuid.NewString(),
				Title: "Failed to Refresh Cluster Statistics",
				Message: fmt.Sprintf("Failed to refresh cluster statistics during workload %s (ID=%s)",
					d.workload.WorkloadName(), d.workload.GetId()),
				Panicked:         false,
				NotificationType: domain.WarningNotification.Int32(),
			})
		}
	}

	stats := d.workload.GetStatistics()
	stats.ClusterStatistics = clusterStatistics
	PatchCSVHeader(stats)

	d.outputFileMutex.Lock()

	if d.appendToOutputFile {
		err = gocsv.MarshalWithoutHeaders([]*Statistics{stats}, d.outputFile)
	} else {
		err = gocsv.Marshal([]*Statistics{stats}, d.outputFile)
		d.appendToOutputFile = true
	}

	d.outputFileMutex.Unlock()

	// If marshalError is not nil, then we'll either join it with the previous error, if the previous error is non-nil,
	// or we'll just assign err to equal the
	if err != nil {
		d.logger.Error("Failed to publish statistics report.", zap.Error(err))

		if d.notifyCallback != nil {
			go d.notifyCallback(&proto.Notification{
				Id:    uuid.NewString(),
				Title: "Failed to Publish Statistics Report",
				Message: fmt.Sprintf("Failed to publish statistics report for workload %s (ID=%s)",
					d.workload.WorkloadName(), d.workload.GetId()),
				Panicked:         false,
				NotificationType: domain.WarningNotification.Int32(),
			})
		}
	}
}

// DriveWorkload accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
// This issues clock ticks as events are submitted.
//
// DriveWorkload should be called from its own goroutine.
func (d *BasicWorkloadDriver) DriveWorkload() {
	var err error

	outputSubdir := time.Now().Format("01-02-2006 15:04:05")
	outputSubdir = strings.ReplaceAll(outputSubdir, ":", "-")
	outputSubdir = fmt.Sprintf("%s - %s", outputSubdir, d.workload.GetId())
	outputSubdirectoryPath := filepath.Join(d.outputFileDirectory, outputSubdir)

	err = os.MkdirAll(outputSubdirectoryPath, os.ModePerm)
	if err != nil {
		d.logger.Error("Failed to create parent directories for workload .CSV output.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("path", outputSubdirectoryPath),
			zap.Error(err))
		d.handleCriticalError(err)
		return
	}

	d.outputFilePath = filepath.Join(outputSubdirectoryPath, "workload_stats.csv")

	d.outputFileMutex.Lock()
	d.outputFile, err = os.OpenFile(d.outputFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	d.outputFileMutex.Unlock()

	if err != nil {
		d.logger.Error("Failed to create .CSV output file for workload.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("path", outputSubdirectoryPath),
			zap.Error(err))
		d.handleCriticalError(err)
		return
	}

	// First, clear the cluster statistics. This will return whatever they were before we called clear.
	_, err = d.refreshClusterStatistics(true, true)
	if err != nil {
		d.logger.Error("Failed to clear and/or retrieve Cluster Statistics before beginning workload.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("reason", err.Error()))
		d.handleCriticalError(err)

		d.outputFileMutex.Lock()
		_ = d.outputFile.Close()
		d.outputFileMutex.Unlock()
		return
	}

	// Fetch the freshly-cleared cluster statistics.
	clusterStats, err := d.refreshClusterStatistics(true, false)
	if err != nil {
		d.logger.Error("Failed to clear and/or retrieve Cluster Statistics before beginning workload.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("reason", err.Error()))
		d.handleCriticalError(err)

		d.outputFileMutex.Lock()
		_ = d.outputFile.Close()
		d.outputFileMutex.Unlock()
		return
	}

	d.workload.GetStatistics().ClusterStatistics = clusterStats

	d.logger.Info("Workload Simulator has started running. Bootstrapping simulation now.",
		zap.String("workload_id", d.id),
		zap.String("workload_name", d.workload.WorkloadName()))

	var statsPublisherWg sync.WaitGroup
	statsPublisherWg.Add(1)

	go func(wg *sync.WaitGroup) {
		for d.workload.IsInProgress() {
			d.publishStatisticsReport()

			time.Sleep(d.workloadOutputInterval)
		}

		// Publish one last statistics report, which will also fetch the Cluster Statistics one last time.
		d.publishStatisticsReport()

		statsPublisherWg.Done()
	}(&statsPublisherWg)

	err = d.bootstrapSimulation()
	if err != nil {
		d.logger.Error("Failed to bootstrap workload.",
			zap.String("workload_id", d.id),
			zap.String("workload_name", d.workload.WorkloadName()),
			zap.String("reason", err.Error()))
		d.handleCriticalError(err)

		d.outputFileMutex.Lock()
		_ = d.outputFile.Close()
		d.outputFileMutex.Unlock()
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
					zap.String("workload_state", d.workload.GetState().String()))

				d.outputFileMutex.Lock()
				_ = d.outputFile.Close()
				d.outputFileMutex.Unlock()
				return
			}

			// If the event occurs during this tick, then call EnqueueEvent to enqueue the event in the EventQueue.
			if evt.Timestamp.Before(nextTick) {
				d.sugaredLogger.Debugf("\"%s\" event \"%s\" targeting session \"%s\" DOES occur before next tick [%v]. Enqueuing event now (timestamp=%v).",
					evt.Name.String(), evt.ID, evt.SessionID(), nextTick, evt.Timestamp)

				d.workload.SetNextEventTick(d.convertTimestampToTickNumber(evt.Timestamp))
				d.workload.SetNextExpectedEventName(evt.Name)
				d.workload.SetNextExpectedEventSession(evt.SessionId)

				d.eventQueue.EnqueueEvent(evt)
			} else {
				d.sugaredLogger.Debugf("\"%s\" event \"%s\" targeting session \"%s\" does NOT occur before next tick [%v] (i.e., tick #%d). Will have to issue clock ticks until we get to event's timestamp of [%v] (i.e., tick #%d).",
					evt.Name.String(), evt.ID, evt.SessionID(), nextTick, nextTick.Unix()/d.targetTickDurationSeconds, evt.Timestamp, evt.Timestamp.Unix()/d.targetTickDurationSeconds)

				d.workload.SetNextEventTick(d.convertTimestampToTickNumber(evt.Timestamp))
				d.workload.SetNextExpectedEventName(evt.Name)
				d.workload.SetNextExpectedEventSession(evt.SessionId)

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

			// Signal to the goroutine running the BasicWorkloadDriver::ProcessWorkloadEvents method that the workload has completed successfully.
			d.workloadExecutionCompleteChan <- struct{}{}

			break OUTER
		}
	}

	statsPublisherWg.Wait()

	d.outputFileMutex.Lock()
	_ = d.outputFile.Close()
	d.outputFileMutex.Unlock()
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

// handlePause is called to check if the workload is paused and, if so, then block until we're un-paused.
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
					zap.String("workload_state", d.workload.GetState().String()),
					zap.Error(err))

				// We failed to pause the workload, so unpause ourselves and just continue.
				_ = d.workload.Unpause() // We don't care if this fails, as long as the workload is running.
				d.paused = false

				// Make sure that the workload is actively running.
				if !d.workload.IsRunning() {
					d.logger.Error("Workload is not actively running anymore. We're stuck.",
						zap.String("workload_id", d.id),
						zap.String("workload_name", d.workload.WorkloadName()),
						zap.String("workload_state", d.workload.GetState().String()),
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
					zap.String("workload_state", d.workload.GetState().String()),
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

		tickNumber := int(d.convertTimestampToTickNumber(tick))
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
				zap.String("workload_state", d.workload.GetState().String()))

			return nil
		}

		// How long the tick took to process.
		// If it took less than the target amount of time, then we'll sleep for a bit.
		tickElapsedBase := time.Since(tickStart)
		tickRemaining := time.Duration(d.timescaleAdjustmentFactor * float64(d.targetTickDuration-tickElapsedBase))

		// Verify that the issuing of the tick did not exceed the specified real-clock-time that a tick should last.
		// TODO: Handle this more elegantly, such as by decreasing the length of subsequent ticks or something?
		if tickRemaining < 0 {
			d.logger.Warn("Issuing clock tick lasted too long.",
				zap.Int("tick_number", tickNumber),
				zap.Time("tick_timestamp", tick),
				zap.Duration("time_elapsed", tickElapsedBase),
				zap.Duration("target_tick_duration", d.targetTickDuration),
				zap.Float64("timescale_adjustment_factor", d.timescaleAdjustmentFactor),
				zap.String("workload_id", d.id),
				zap.String("workload_name", d.workload.WorkloadName()),
				zap.String("workload_state", d.workload.GetState().String()))
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
				zap.String("workload_state", d.workload.GetState().String()))
			time.Sleep(tickRemaining)
		}

		tickDuration := time.Since(tickStart)
		tickDurationSec := decimal.NewFromFloat(tickDuration.Seconds())

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

// ProcessWorkloadEvents accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
// This processes events in response to clock ticks.
//
// If there is a critical error that causes the workload to be terminated prematurely/aborted, then that error is returned.
// If the workload is able to complete successfully, then nil is returned.
//
// ProcessWorkload should be called from its own goroutine.
func (d *BasicWorkloadDriver) ProcessWorkloadEvents() {
	d.mu.Lock()

	if d.workload == nil {
		d.logger.Error("Workload is nil. Cannot process it.")
		return
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

	if d.workload.IsTemplateWorkload() {
		go func() {
			err := d.workloadGenerator.GenerateTemplateWorkload(d, d.workloadSessions, d.workloadRegistrationRequest)
			if err != nil {
				d.logger.Error("Failed to drive/generate templated workload.",
					zap.String("workload_id", d.id),
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.Error(err))
			}
		}()
	} else if d.workload.IsPresetWorkload() {
		panic("Not supported anymore.")
		//go func() {
		//	presetWorkload := d.workload.(*Preset)
		//	err := d.workloadGenerator.GeneratePresetWorkload(d, presetWorkload, presetWorkload.WorkloadPreset, d.workloadRegistrationRequest)
		//	if err != nil {
		//		d.logger.Error("Failed to drive/generate preset workload.",
		//			zap.String("workload_id", d.id),
		//			zap.String("workload_name", d.workload.WorkloadName()),
		//			zap.Error(err))
		//	}
		//}()
	} else {
		panic(fmt.Sprintf("Workload is of presently-unsuporrted type: \"%s\" -- cannot generate workload.", d.workload.GetKind()))
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

					return
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
				return // We're done, so we can return.
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

				return // We're done, so we can return.
			}
		case <-d.workloadExecutionCompleteChan: // This is placed after eventChan so that all events are processed first.
			{
				d.logger.Debug("All events have been enqueued with their respective Clients.",
					zap.String("workload_id", d.id))

				startWait := time.Now()
				d.clientsWaitGroup.Wait()

				d.logger.Debug("All Clients have finished processing their events.",
					zap.String("workload_id", d.id),
					zap.Duration("wait_duration", time.Since(startWait)))

				d.workloadComplete()
				return
			}
		}
	}
}

// workloadComplete is called when the BasicWorkloadDriver receives a signal on its workloadExecutionCompleteChan
// field informing it that the workload has completed successfully.
//
// workloadComplete accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
func (d *BasicWorkloadDriver) workloadComplete() {
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

	d.sessionConnectionsMutex.Lock()
	defer d.sessionConnectionsMutex.Unlock()

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
}

// convertTimestampToTickNumber converts the given tick, which is specified in the form of a time.Time,
// and returns what "tick number" that tick is.
//
// Basically, you just convert the timestamp to its unix epoch timestamp (in seconds), and divide by the
// trace step value (also in seconds).
func (d *BasicWorkloadDriver) convertTimestampToTickNumber(tick time.Time) int64 {
	return tick.Unix() / d.targetTickDurationSeconds
}

// Handle a tick during the execution of a workload.
//
// This should just be called by BasicWorkloadDriver::ProcessWorkloadEvents.
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

	// Process "start/stop training" events.
	d.enqueueEventsForTick(tick)

	d.ticker.Done()

	return nil
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

// RegisterApproximateFinalTick is used to register what is the approximate final tick of the workload
// after iterating over all sessions and all training events.
func (d *BasicWorkloadDriver) RegisterApproximateFinalTick(approximateFinalTick int64) {
	d.workload.RegisterApproximateFinalTick(approximateFinalTick)
}

// EventQueue returns the event queue for this workload.
func (d *BasicWorkloadDriver) EventQueue() *event_queue.EventQueue {
	return d.eventQueue
}

// enqueueEventsForTick processes events in chronological/simulation order.
// This accepts the "current tick" as an argument. The current tick is basically an upper-bound on the times for
// which we'll process an event. For example, if `tick` is 19:05:00, then we will process all cluster and session
// events with timestamps that are (a) equal to 19:05:00 or (b) come before 19:05:00. Any events with timestamps
// that come after 19:05:00 will not be processed until the next tick.
func (d *BasicWorkloadDriver) enqueueEventsForTick(tick time.Time) {
	// Extract all the "session-ready" events for this tick.
	for d.eventQueue.HasEventsForTick(tick) {
		evt := d.eventQueue.Pop(tick)

		sessionId := evt.SessionId

		if evt.Name == domain.EventSessionReady {
			client := NewClientBuilder().
				WithSessionId(sessionId).
				WithWorkloadId(d.workload.GetId()).
				WithSessionReadyEvent(evt).
				WithStartingTick(tick).
				WithAtom(d.atom).
				WithKernelManager(d.kernelManager).
				WithTargetTickDurationSeconds(d.targetTickDurationSeconds).
				WithErrorChan(d.errorChan).
				WithWorkload(d.workload).
				WithSession(d.workloadSessionsMap[sessionId]).
				WithNotifyCallback(d.notifyCallback).
				WithWaitGroup(&d.clientsWaitGroup).
				Build()

			d.clients[sessionId] = client
			d.clientsWaitGroup.Add(1)
			go client.Run()
		} else {
			client, loaded := d.clients[sessionId]
			if !loaded {
				d.logger.Error("Client not found.",
					zap.String("workload_name", d.workload.WorkloadName()),
					zap.String("workload_id", d.workload.GetId()),
					zap.String("session_id", sessionId),
					zap.String("event_name", evt.Name.String()),
					zap.String("event", evt.String()))

				// Discard events for this Session.
				err := d.workload.SessionDiscarded(sessionId)
				if err != nil {
					d.logger.Error("Failed to discard events for session (whose client we couldn't find).",
						zap.String("workload_name", d.workload.WorkloadName()),
						zap.String("workload_id", d.workload.GetId()),
						zap.String("session_id", sessionId),
						zap.String("event_name", evt.Name.String()),
						zap.String("event", evt.String()),
						zap.Error(err))
				}

				continue
			}

			client.EventQueue.Push(evt)
		}

		// TODO: Enqueue event with respective client.
		//		 SessionReadyEvents will require a new client.
	}

	// Wait for all the goroutines to complete before returning.
	// waitGroup.Wait()
	d.workload.UpdateTimeElapsed()
}

// Given a session ID, such as from the trace data, return the ID used internally.
//
// The internal ID includes the unique ID of this workload driver, in case multiple
// workloads from the same trace are being executed concurrently.
func (d *BasicWorkloadDriver) getInternalSessionId(traceSessionId string) string {
	//return fmt.Sprintf("%s-%s", traceSessionId, d.id)
	return traceSessionId
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

// ObserveJupyterSessionCreationLatency records the latency of creating a Jupyter session
// during the execution of a particular workload, as identified by the given workload ID.
func (d *BasicWorkloadDriver) ObserveJupyterSessionCreationLatency(latencyMilliseconds int64, workloadId string) {
	stats := d.workload.GetStatistics()

	stats.CumulativeJupyterSessionCreationLatencyMillis += latencyMilliseconds
	stats.JupyterSessionCreationLatenciesMillis = append(
		stats.JupyterSessionCreationLatenciesMillis, latencyMilliseconds)

	if metrics.PrometheusMetricsWrapperInstance != nil {
		metrics.PrometheusMetricsWrapperInstance.ObserveJupyterSessionCreationLatency(latencyMilliseconds, workloadId)
	}
}

// ObserveJupyterSessionTerminationLatency records the latency of terminating a Jupyter session
// during the execution of a particular workload, as identified by the given workload ID.
func (d *BasicWorkloadDriver) ObserveJupyterSessionTerminationLatency(latencyMilliseconds int64, workloadId string) {
	stats := d.workload.GetStatistics()

	stats.CumulativeJupyterSessionTerminationLatencyMillis += latencyMilliseconds
	stats.JupyterSessionTerminationLatenciesMillis = append(
		stats.JupyterSessionTerminationLatenciesMillis, latencyMilliseconds)

	if metrics.PrometheusMetricsWrapperInstance != nil {
		metrics.PrometheusMetricsWrapperInstance.ObserveJupyterSessionTerminationLatency(latencyMilliseconds, workloadId)
	}
}

// ObserveJupyterExecuteRequestE2ELatency records the end-to-end latency of an "execute_request" message
// during the execution of a particular workload, as identified by the given workload ID.
func (d *BasicWorkloadDriver) ObserveJupyterExecuteRequestE2ELatency(latencyMilliseconds int64, workloadId string) {
	stats := d.workload.GetStatistics()

	stats.CumulativeJupyterExecRequestTimeMillis += latencyMilliseconds
	stats.JupyterExecRequestTimesMillis = append(
		stats.JupyterExecRequestTimesMillis, latencyMilliseconds)

	if metrics.PrometheusMetricsWrapperInstance != nil {
		metrics.PrometheusMetricsWrapperInstance.ObserveJupyterExecuteRequestE2ELatency(latencyMilliseconds, workloadId)
	}
}

// AddJupyterRequestExecuteTime records the time taken to process an "execute_request" for the total, aggregate,
// cumulative time spent processing "execute_request" messages.
func (d *BasicWorkloadDriver) AddJupyterRequestExecuteTime(latencyMilliseconds int64, kernelId string, workloadId string) {
	if metrics.PrometheusMetricsWrapperInstance != nil {
		metrics.PrometheusMetricsWrapperInstance.AddJupyterRequestExecuteTime(latencyMilliseconds, kernelId, workloadId)
	}
}
