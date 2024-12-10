package workload

//
//import (
//	"context"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"github.com/gin-gonic/gin"
//	"github.com/google/uuid"
//	"github.com/gorilla/websocket"
//	"github.com/mattn/go-colorable"
//	"github.com/mgutz/ansi"
//	"github.com/prometheus/client_golang/prometheus"
//	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
//	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
//	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
//	"github.com/scusemua/workload-driver-react/m/v2/internal/server/clock"
//	"github.com/scusemua/workload-driver-react/m/v2/internal/server/event_queue"
//	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
//	"github.com/scusemua/workload-driver-react/m/v2/pkg/jupyter"
//	"github.com/scusemua/workload-driver-react/m/v2/pkg/statistics"
//	"github.com/shopspring/decimal"
//	"github.com/zhangjyr/gocsv"
//	"github.com/zhangjyr/hashmap"
//	"go.uber.org/zap"
//	"go.uber.org/zap/zapcore"
//	"io"
//	"math"
//	"math/rand"
//	"net/http"
//	"os"
//	"path"
//	"path/filepath"
//	"reflect"
//	"strings"
//	"sync"
//	"sync/atomic"
//	"time"
//)
//
//// BasicWorkloadDriverOld consumes events from the Workload Generator and takes action accordingly.
//type BasicWorkloadDriverOld struct {
//	logger        *zap.Logger
//	sugaredLogger *zap.SugaredLogger
//	atom          *zap.AtomicLevel
//
//	clockTime                          domain.SimulationClock                // Contains the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
//	clockTrigger                       *clock.Trigger                        // Trigger for the clock ticks
//	currentTick                        domain.SimulationClock                // Contains the current tick of the workload.
//	workloadExecutionCompleteChan      chan interface{}                      // Used to signal that the workload has successfully processed all events and is complete.
//	workloadEventGeneratorCompleteChan chan interface{}                      // Used to signal that the generators have submitted all events. Once all remaining, already-enqueued events have been processed, the workload will be complete.
//	driverTimescale                    float64                               // Multiplier that impacts the timescale at the Driver will operate on with respect to the trace data. For example, if each tick is 60 seconds, then a DriverTimescale value of 0.5 will mean that each tick will take 30 seconds.
//	errorChan                          chan error                            // Used to stop the workload due to a critical error.
//	eventChan                          chan *domain.Event                    // Receives events from the Synthesizer.
//	eventQueue                         *event_queue.EventQueue               // Maintains a queue of events to be processed for each session.
//	getSchedulingPolicyCallback        func() (string, bool)                 // getSchedulingPolicyCallback is a callback to retrieve the configured scheduling policy of the cluster.
//	schedulingPolicy                   string                                // Cached scheduling policy value
//	id                                 string                                // Unique ID (relative to other drivers). The workload registered with this driver will be assigned this ID.
//	kernelManager                      jupyter.KernelSessionManager          // Simplified Go implementation of the Jupyter JavaScript API.
//	mu                                 sync.Mutex                            // Synchronizes access to internal data structures. Can be locked externally using the Lock/Unlock API exposed by the WorkloadDriver.
//	opts                               *domain.Configuration                 // The system's configuration, read from a file.
//	performClockTicks                  bool                                  // If true, then we'll issue clock ticks. Otherwise, don't issue them. Mostly used for testing/debugging.
//	servingTicks                       atomic.Bool                           // The WorkloadDriver::ServeTicks() method will continue looping as long as this flag is set to true.
//	sessionConnections                 map[string]*jupyter.SessionConnection // Map from internal session ID to session connection.
//	sessionConnectionsMutex            sync.Mutex                            // sessionConnections ensures atomic access to the sessionConnections map
//	sessions                         *hashmap.HashMap                        // Responsible for creating sessions and maintaining a collection of all the sessions active within the simulation.
//	stopChan                         chan interface{}                        // Used to stop the workload early/prematurely (i.e., before all events have been processed).
//	targetTickDuration               time.Duration                           // How long each tick is supposed to last. This is the tick interval/step rate of the simulation.
//	targetTickDurationSeconds        int64                                   // Cached total number of seconds of targetTickDuration
//	tickDurationsSecondsMovingWindow *statistics.MovingStat                  // Moving average of observed tick durations in seconds.
//	tickDurationsAll                 []time.Duration                         // All tick durations from the entire workload.
//	ticker                           *clock.Ticker                           // Receive Tick events this way.
//	ticksHandled                     atomic.Int64                            // Incremented/accessed atomically.
//	timescaleAdjustmentFactor        float64                                 // Adjusts the timescale of the simulation. Setting this to 1 means that each tick is simulated as a whole minute. Setting this to 0.5 means each tick will be simulated for half its real time. So, if ticks are 60 seconds, and this variable is set to 0.5, then each tick will be simulated for 30 seconds before continuing to the next tick.
//	websocket                        domain.ConcurrentWebSocket              // Shared Websocket used to communicate with frontend.
//	workload                         internalWorkload                        // The workload being driven by this driver.
//	workloadStartTime                time.Time                               // The time at which the workload began.
//	workloadEndTime                  time.Time                               // The time at which the workload completed.
//	workloadGenerator                domain.WorkloadGenerator                // The entity generating the workload (from trace data, a preset, or a template).
//	workloadPreset                   *domain.WorkloadPreset                  // The preset used by the associated workload. Will only be non-nil if the associated workload is a preset-based workload, rather than a template-based workload.
//	workloadPresets                  map[string]*domain.WorkloadPreset       // All the available workload presets.
//	workloadRegistrationRequest      *domain.WorkloadRegistrationRequest     // The request that registered the workload that is being driven by this driver.
//	workloadSessions                 []*domain.WorkloadTemplateSession       // The template used by the associated workload. Will only be non-nil if the associated workload is a template-based workload, rather than a preset-based workload.
//	paused                           bool                                    // Paused indicates whether the workload has been paused.
//	trainingSubmittedTimes           *hashmap.HashMap                        // trainingSubmittedTimes keeps track of when "execute_request" messages were sent for different sessions. Keys are internal session IDs, values are unix millisecond timestamps.
//	outputFile                       io.ReadWriteCloser                      // The opened .CSV output statistics file.
//	outputFileDirectory                string                                // outputFileDirectory is the directory where all the workload-specific output directories live
//	outputFilePath                     string                                // Path to the outputFile
//	outputFileMutex                    sync.Mutex                            // Atomic access to output file
//	appendToOutputFile                 bool                                  // Flag that is set to true after the first write
//	misbehavingSessions                map[string]interface{}                // Map from session ID to sessions for sessions whose events we did not finish processing in a previous tick.
//	misbehavingSessionsMutex           sync.Mutex                            // misbehavingSessionsMutex ensures atomic access to the misbehavingSessions
//	trainingStartedChannels            map[string]chan interface{}           // trainingStartedChannels are channels used to notify that training has started
//	trainingStartedChannelMutex        sync.Mutex                            // trainingStartedChannelMutex ensures atomic access to the trainingStartedChannels
//	trainingStoppedChannels            map[string]chan interface{}           // trainingStartedChannels are channels used to notify that training has ended
//	trainingStoppedChannelsMutex       sync.Mutex                            // trainingStoppedChannelsMutex ensures atomic access to the trainingStoppedChannels
//
//	pauseMutex sync.Mutex
//	pauseCond  *sync.Cond
//
//	// refreshClusterStatistics is used to fresh the ClusterStatistics from the Cluster Gateway.
//	refreshClusterStatistics ClusterStatisticsRefresher
//
//	// notifyCallback is a function used to send notifications related to this workload directly to the frontend.
//	notifyCallback func(notification *proto.Notification)
//
//	// onCriticalErrorOccurred is a handler that is called when a critical error occurs.
//	// The onCriticalErrorOccurred handler is called in its own goroutine.
//	onCriticalErrorOccurred domain.WorkloadErrorHandler
//
//	// onNonCriticalErrorOccurred is a handler that is called when a non-critical error occurs.
//	// The onNonCriticalErrorOccurred handler is called in its own goroutine.
//	onNonCriticalErrorOccurred domain.WorkloadErrorHandler
//}
//
//func NewBasicWorkloadDriverOld(opts *domain.Configuration, performClockTicks bool, timescaleAdjustmentFactor float64,
//	websocket domain.ConcurrentWebSocket, atom *zap.AtomicLevel, callbackProvider CallbackProvider) *BasicWorkloadDriverOld {
//
//	jupyterAddress := path.Join(opts.InternalJupyterServerAddress, opts.JupyterServerBasePath)
//
//	driver := &BasicWorkloadDriverOld{
//		id:                                 GenerateWorkloadID(8),
//		eventChan:                          make(chan *domain.Event),
//		outputFileDirectory:                opts.WorkloadOutputDirectory,
//		clockTrigger:                       clock.NewTrigger(),
//		opts:                               opts,
//		workloadExecutionCompleteChan:      make(chan interface{}, 1),
//		workloadEventGeneratorCompleteChan: make(chan interface{}),
//		stopChan:                           make(chan interface{}, 1),
//		errorChan:                          make(chan error, 2),
//		misbehavingSessions:                make(map[string]interface{}),
//		atom:                               atom,
//		trainingStartedChannels:            make(map[string]chan interface{}),
//		trainingStoppedChannels:            make(map[string]chan interface{}),
//		targetTickDuration:                 time.Second * time.Duration(opts.TraceStep),
//		targetTickDurationSeconds:          opts.TraceStep,
//		tickDurationsSecondsMovingWindow:   statistics.NewMovingStat(5),
//		tickDurationsAll:                   make([]time.Duration, 0),
//		driverTimescale:                    opts.DriverTimescale,
//		sessionConnections:                 make(map[string]*jupyter.SessionConnection),
//		performClockTicks:                  performClockTicks,
//		eventQueue:                         event_queue.NewEventQueue(atom),
//		trainingSubmittedTimes:             hashmap.New(100),
//		sessions:                           hashmap.New(100),
//		websocket:                          websocket,
//		timescaleAdjustmentFactor:          timescaleAdjustmentFactor,
//		currentTick:                        clock.NewSimulationClock(),
//		clockTime:                          clock.NewSimulationClock(),
//		onCriticalErrorOccurred:            callbackProvider.HandleCriticalWorkloadError,
//		onNonCriticalErrorOccurred:         callbackProvider.HandleWorkloadError,
//		notifyCallback:                     callbackProvider.SendNotification,
//		refreshClusterStatistics:           callbackProvider.RefreshAndClearClusterStatistics,
//		getSchedulingPolicyCallback:        callbackProvider.GetSchedulingPolicy,
//		paused:                             false,
//	}
//
//	driver.pauseCond = sync.NewCond(&driver.pauseMutex)
//
//	// Create the ticker for the workload.
//	driver.ticker = driver.clockTrigger.NewSyncTicker(time.Second*time.Duration(opts.TraceStep), fmt.Sprintf("Workload-%s", driver.id), driver.clockTime)
//
//	zapConfig := zap.NewDevelopmentEncoderConfig()
//	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
//	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), driver.atom)
//	logger := zap.New(core, zap.Development())
//	if logger == nil {
//		panic("failed to create logger for workload driver")
//	}
//
//	driver.logger = logger
//	driver.sugaredLogger = logger.Sugar()
//
//	// TODO: Can we just load them in from a file once? Why do this for every single workload?
//	// Load the list of workload presets from the specified file.
//	driver.logger.Debug("Loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath))
//	presets, err := domain.LoadWorkloadPresetsFromFile(opts.WorkloadPresetsFilepath)
//	if err != nil {
//		driver.logger.Error("Error encountered while loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath), zap.Error(err))
//	}
//
//	driver.workloadPresets = make(map[string]*domain.WorkloadPreset, len(presets))
//	for _, preset := range presets {
//		driver.workloadPresets[preset.GetKey()] = preset
//	}
//
//	driver.kernelManager = jupyter.NewKernelSessionManager(jupyterAddress, true, atom, driver)
//
//	if driver.onNonCriticalErrorOccurred != nil {
//		driver.kernelManager.RegisterOnErrorHandler(func(sessionId string, kernelId string, err error) {
//			err = fmt.Errorf("error occurred for kernel=%s,session=%s: %w", kernelId, sessionId, err)
//			driver.onNonCriticalErrorOccurred(driver.id, err)
//		})
//	}
//
//	return driver
//}
//
//func (d *BasicWorkloadDriverOld) GetOutputFileContents() ([]byte, error) {
//	d.outputFileMutex.Lock()
//	defer d.outputFileMutex.Unlock()
//
//	if d.outputFile == nil {
//		d.logger.Warn("Cannot return contents of output file. It is nil (i.e., hasn't been created yet).",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()))
//		return []byte{}, nil
//	}
//
//	csvBuffer, err := io.ReadAll(d.outputFile)
//
//	if err != nil {
//		d.logger.Warn("Failed to read contents of workload statistics file. Will try opening the file explicitly.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.Error(err))
//
//		outputFile, err := os.Open(d.outputFilePath)
//		if err != nil {
//			d.logger.Error("Failed to open workload statistics file explicitly.",
//				zap.String("workload_id", d.workload.GetId()),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.Error(err))
//
//			return nil, err
//		}
//
//		csvBuffer, err = io.ReadAll(outputFile)
//		if err != nil {
//			d.logger.Error("Failed to read contents of workload statistics file after opening it explicitly.",
//				zap.String("workload_id", d.workload.GetId()),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.Error(err))
//
//			return nil, err
//		}
//
//		_ = outputFile.Close()
//	}
//
//	return csvBuffer, nil
//}
//
//// StartWorkload starts the Workload that is associated with/managed by this workload driver.
////
//// If the workload is already running, then an error is returned.
//// Likewise, if the workload was previously running but has already stopped, then an error is returned.
//func (d *BasicWorkloadDriverOld) StartWorkload() error {
//	d.logger.Debug("Workload Driver is starting workload.", zap.String("workload-driver-id", d.id))
//	return d.workload.StartWorkload()
//}
//
//// GetErrorChan returns the channel used to report critical errors encountered while executing the workload.
//func (d *BasicWorkloadDriverOld) GetErrorChan() chan<- error {
//	return d.errorChan
//}
//
//// CurrentTick returns the current tick of the workload.
//func (d *BasicWorkloadDriverOld) CurrentTick() domain.SimulationClock {
//	return d.currentTick
//}
//
//// ClockTime returns the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
//func (d *BasicWorkloadDriverOld) ClockTime() domain.SimulationClock {
//	return d.clockTime
//}
//
//// WebSocket returns the WebSocket connection on which this workload was registered by a remote client and on/through which updates about the workload are reported.
//func (d *BasicWorkloadDriverOld) WebSocket() domain.ConcurrentWebSocket {
//	return d.websocket
//}
//
//// IsSessionBeingSampled returns true if the specified session was selected for sampling.
//func (d *BasicWorkloadDriverOld) IsSessionBeingSampled(sessionId string) bool {
//	return d.workload.IsSessionBeingSampled(sessionId)
//}
//
//// GetSampleSessionsPercentage returns the configured SampleSessionsPercentage parameter for the Workload.
//func (d *BasicWorkloadDriverOld) GetSampleSessionsPercentage() float64 {
//	return d.workload.GetSampleSessionsPercentage()
//}
//
//func (d *BasicWorkloadDriverOld) SubmitEvent(evt *domain.Event) {
//	if !d.workload.IsSessionBeingSampled(evt.SessionID()) {
//		return
//	}
//
//	d.logger.Debug("Submitting session-level event.",
//		zap.String("session_id_field", evt.SessionId),
//		zap.String("session_id", evt.SessionID()),
//		zap.String("event_name", evt.Name.String()),
//		zap.Time("event_timestamp", evt.Timestamp))
//
//	d.eventChan <- evt
//}
//
//func (d *BasicWorkloadDriverOld) ToggleDebugLogging(enabled bool) domain.Workload {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//
//	if enabled {
//		d.atom.SetLevel(zap.DebugLevel)
//		d.workload.SetDebugLoggingEnabled(true)
//	} else {
//		d.atom.SetLevel(zap.InfoLevel)
//		d.workload.SetDebugLoggingEnabled(false)
//	}
//
//	return d.workload
//}
//
//func (d *BasicWorkloadDriverOld) GetWorkload() internalWorkload {
//	return d.workload
//}
//
//func (d *BasicWorkloadDriverOld) GetWorkloadPreset() *domain.WorkloadPreset {
//	return d.workloadPreset
//}
//
//func (d *BasicWorkloadDriverOld) GetWorkloadRegistrationRequest() *domain.WorkloadRegistrationRequest {
//	return d.workloadRegistrationRequest
//}
//
//// Create a workload that was created using a preset.
//func (d *BasicWorkloadDriverOld) createWorkloadFromPreset(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (internalWorkload, error) {
//	// The specified preset should be in our map of workload presets.
//	// If it isn't, then the registration request is invalid, and we'll return an error.
//	var ok bool
//	if d.workloadPreset, ok = d.workloadPresets[workloadRegistrationRequest.Key]; !ok {
//		d.logger.Error("Could not find workload preset with specified key.", zap.String("key", workloadRegistrationRequest.Key))
//		return nil, ErrWorkloadPresetNotFound
//	}
//
//	d.logger.Debug("Creating new workload from preset.",
//		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//		zap.String("workload-preset-name", d.workloadPreset.GetName()))
//
//	basicWorkload := NewBuilder(d.atom).
//		SetID(d.id).
//		SetWorkloadName(workloadRegistrationRequest.WorkloadName).
//		SetSeed(workloadRegistrationRequest.Seed).
//		EnableDebugLogging(workloadRegistrationRequest.DebugLogging).
//		SetTimescaleAdjustmentFactor(workloadRegistrationRequest.TimescaleAdjustmentFactor).
//		SetRemoteStorageDefinition(workloadRegistrationRequest.RemoteStorageDefinition).
//		SetSessionsSamplePercentage(workloadRegistrationRequest.SessionsSamplePercentage).
//		Build()
//
//	workloadFromPreset := NewWorkloadFromPreset(basicWorkload, d.workloadPreset)
//
//	err := workloadFromPreset.SetSource(d.workloadPreset)
//
//	if err != nil {
//		return nil, err
//	}
//
//	return workloadFromPreset, nil
//}
//
//// loadWorkloadTemplateFromFile is used for workload templates that come pre-defined
//// on the server because of how large they are.
//func (d *BasicWorkloadDriverOld) loadWorkloadTemplateFromFile(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) error {
//	if workloadRegistrationRequest.TemplateFilePath == "" {
//		return ErrTemplateFilePathNotSpecified
//	}
//
//	d.logger.Debug("Registering workload from pre-loaded template.",
//		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath))
//
//	templateJsonFile, err := os.Open(workloadRegistrationRequest.TemplateFilePath)
//	if err != nil {
//		return fmt.Errorf("%w: %w: \"%s\"",
//			ErrInvalidTemplateFileSpecified, err, workloadRegistrationRequest.TemplateFilePath)
//	}
//	defer func() {
//		err := templateJsonFile.Close()
//		if err != nil {
//			d.logger.Warn("Error while closing workload template file.",
//				zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//				zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
//				zap.Error(err))
//		}
//	}()
//
//	d.logger.Debug("Creating new workload from template file.",
//		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath))
//
//	st := time.Now()
//
//	// Read the file contents
//	templateJsonFileContents, err := io.ReadAll(templateJsonFile)
//	if err != nil {
//		d.logger.Error("Failed to read workload template file.",
//			zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//			zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
//			zap.Error(err))
//		return err
//	}
//
//	d.logger.Debug("Successfully read workload from template file.",
//		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//		zap.Duration("time_elapsed", time.Since(st)),
//		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath))
//
//	// Unmarshal JSON into the struct
//	var unmarshalledRegistrationRequest *domain.WorkloadRegistrationRequest
//	err = json.Unmarshal(templateJsonFileContents, &unmarshalledRegistrationRequest)
//	if err != nil {
//		d.logger.Error("Failed to unmarshal workload template from file.",
//			zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//			zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
//			zap.Error(err))
//		return err
//	}
//
//	d.logger.Debug("Successfully loaded workload from template.",
//		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//		zap.Duration("time_elapsed", time.Since(st)),
//		zap.String("template_file_path", workloadRegistrationRequest.TemplateFilePath),
//		zap.Int("num_sessions", len(unmarshalledRegistrationRequest.Sessions)))
//
//	workloadRegistrationRequest.Sessions = unmarshalledRegistrationRequest.Sessions
//
//	return nil
//}
//
//// Create a workload that was created using a template.
//func (d *BasicWorkloadDriverOld) createWorkloadFromTemplate(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (*Template, error) {
//	// The workload request needs to have a workload template in it.
//	// If the registration request does not contain a workload template,
//	// then the request is invalid, and we'll return an error.
//
//	if workloadRegistrationRequest.Sessions == nil || len(workloadRegistrationRequest.Sessions) == 0 {
//		if workloadRegistrationRequest.TemplateFilePath == "" {
//			d.logger.Error("Workload Registration Request for template-based workload is missing the sessions or template file path!")
//			return nil, ErrWorkloadRegistrationMissingTemplate
//		}
//
//		err := d.loadWorkloadTemplateFromFile(workloadRegistrationRequest)
//		if err != nil {
//			return nil, err
//		}
//	}
//
//	d.logger.Debug("Creating new workload from template.",
//		zap.String("workload_name", workloadRegistrationRequest.WorkloadName))
//
//	d.workloadSessions = workloadRegistrationRequest.Sessions
//	d.workloadRegistrationRequest = workloadRegistrationRequest
//	basicWorkload := NewBuilder(d.atom).
//		SetID(d.id).
//		SetWorkloadName(workloadRegistrationRequest.WorkloadName).
//		SetSeed(workloadRegistrationRequest.Seed).
//		EnableDebugLogging(workloadRegistrationRequest.DebugLogging).
//		SetTimescaleAdjustmentFactor(workloadRegistrationRequest.TimescaleAdjustmentFactor).
//		SetRemoteStorageDefinition(workloadRegistrationRequest.RemoteStorageDefinition).
//		SetSessionsSamplePercentage(workloadRegistrationRequest.SessionsSamplePercentage).
//		Build()
//
//	workloadFromTemplate, err := NewWorkloadFromTemplate(basicWorkload, workloadRegistrationRequest.Sessions)
//	if err != nil {
//		return nil, err
//	}
//
//	return workloadFromTemplate, nil
//}
//
//// RegisterWorkload registers a workload with the driver.
//// Returns nil if the workload could not be registered.
//func (d *BasicWorkloadDriverOld) RegisterWorkload(workloadRegistrationRequest *domain.WorkloadRegistrationRequest) (domain.Workload, error) {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//
//	// Only one workload per driver.
//	if d.workload != nil {
//		return nil, ErrWorkloadAlreadyRegistered
//	}
//
//	d.workloadRegistrationRequest = workloadRegistrationRequest
//
//	// Setup log-level.
//	if !workloadRegistrationRequest.DebugLogging {
//		d.logger.Debug("Setting log-level to INFO.")
//		d.atom.SetLevel(zapcore.InfoLevel)
//		d.logger.Debug("Ignored.")
//	} else {
//		d.logger.Debug("Debug-level logging is ENABLED.")
//	}
//
//	d.logger.Debug("Registering workload.",
//		zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//		zap.String("workload-key", workloadRegistrationRequest.Key))
//
//	// We create the workload a little differently depending on its type (either 'preset' or 'template').
//	// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
//	// have properties that the user can specify and change before submitting the workload for registration.
//	var (
//		// If this is created successfully, then d.workload will be assigned the value of this variable.
//		workload internalWorkload
//		err      error // If the workload is not created successfully, then we'll return this error.
//	)
//	switch strings.ToLower(workloadRegistrationRequest.Type) {
//	case "preset":
//		{
//			// Preset-workload-specific workload creation and initialization steps.
//			workload, err = d.createWorkloadFromPreset(workloadRegistrationRequest)
//
//			if err != nil {
//				d.logger.Error("Failed to create workload from preset.",
//					zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//					zap.Error(err))
//				return nil, err
//			}
//		}
//	case "template":
//		{
//			// Template-workload-specific workload creation and initialization steps.
//			workload, err = d.createWorkloadFromTemplate(workloadRegistrationRequest)
//
//			if err != nil {
//				d.logger.Error("Failed to create workload from template.",
//					zap.String("workload_name", workloadRegistrationRequest.WorkloadName),
//					zap.Error(err))
//				return nil, err
//			}
//		}
//	default:
//		{
//			d.logger.Error("Unsupported workload type.",
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("workload_id", d.workload.GetId()),
//				zap.String("workload-type", workloadRegistrationRequest.Type))
//			return nil, fmt.Errorf("%w: \"%s\"", ErrUnsupportedWorkloadType, workloadRegistrationRequest.Type)
//		}
//	}
//
//	if workload == nil {
//		panic("Workload should not be nil at this point.")
//	}
//
//	if d.onCriticalErrorOccurred != nil {
//		workload.RegisterOnCriticalErrorHandler(d.onCriticalErrorOccurred)
//		d.logger.Debug("Registered critical error handler with workload.",
//			zap.String("workload_id", workload.GetId()),
//			zap.String("workload_name", workload.WorkloadName()),
//			zap.String("workload_driver_id", d.id))
//	} else {
//		d.logger.Warn("No critical error handler configured on workload driver.",
//			zap.String("workload_id", workload.GetId()),
//			zap.String("workload_name", workload.WorkloadName()),
//			zap.String("workload_driver_id", d.id))
//	}
//
//	if d.onNonCriticalErrorOccurred != nil {
//		workload.RegisterOnNonCriticalErrorHandler(d.onNonCriticalErrorOccurred)
//		d.logger.Debug("Registered non-critical error handler with workload.",
//			zap.String("workload_id", workload.GetId()),
//			zap.String("workload_name", workload.WorkloadName()),
//			zap.String("workload_driver_id", d.id))
//	} else {
//		d.logger.Warn("No non-critical error handler configured on workload driver.",
//			zap.String("workload_id", workload.GetId()),
//			zap.String("workload_name", workload.WorkloadName()),
//			zap.String("workload_driver_id", d.id))
//	}
//
//	// If the workload seed is negative, then assign it a random value.
//	if workloadRegistrationRequest.Seed < 0 {
//		workload.SetSeed(rand.Int63n(2147483647)) // We restrict the user to the range 0-2,147,483,647 when they specify a seed.
//		d.logger.Debug("Will use random seed for RNG.", zap.Int64("workload-seed", workloadRegistrationRequest.Seed))
//	} else {
//		d.logger.Debug("Will use user-specified seed for RNG.", zap.Int64("workload-seed", workloadRegistrationRequest.Seed))
//	}
//
//	d.workload = workload
//	d.kernelManager.AddMetadata(jupyter.WorkloadIdMetadataKey, d.workload.GetId())
//	d.kernelManager.AddMetadata(jupyter.RemoteStorageDefinitionMetadataKey, d.workload.GetRemoteStorageDefinition())
//	return d.workload, nil
//}
//
//// WriteError writes an error back to the client.
//func (d *BasicWorkloadDriverOld) WriteError(ctx *gin.Context, errorMessage string) {
//	// Write error back to front-end.
//	msg := &domain.ErrorMessage{
//		ErrorMessage: errorMessage,
//		Valid:        true,
//	}
//	ctx.JSON(http.StatusInternalServerError, msg)
//}
//
//// IsWorkloadComplete returns true if the workload has completed; otherwise, return false.
//func (d *BasicWorkloadDriverOld) IsWorkloadComplete() bool {
//	return d.workload.GetState() == Finished
//}
//
//// ID returns the unique ID of this workload driver.
//// This is not necessarily the same as the workload's unique ID (TODO: or is it?).
//func (d *BasicWorkloadDriverOld) ID() string {
//	return d.id
//}
//
//// StopWorkload stops a workload that's already running/in-progress.
//// Returns nil on success, or an error if one occurred.
//func (d *BasicWorkloadDriverOld) StopWorkload() error {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//
//	if !d.workload.IsInProgress() {
//		return domain.ErrWorkloadNotRunning
//	}
//
//	d.logger.Debug("Stopping workload.", zap.String("workload_id", d.id), zap.String("workload-state", string(d.workload.GetState())))
//	d.stopChan <- struct{}{}
//	d.logger.Debug("Sent 'STOP' instruction via BasicWorkloadDriverOld::stopChan.", zap.String("workload_id", d.id))
//
//	endTime, err := d.workload.TerminateWorkloadPrematurely(d.clockTime.GetClockTime())
//	if err != nil {
//		d.logger.Error("Failed to stop workload.", zap.String("workload_id", d.id), zap.Error(err))
//		return err
//	}
//
//	// d.workloadEndTime, _ = d.workload.GetEndTime()
//	d.workloadEndTime = endTime
//
//	d.logger.Debug("Successfully stopped workload.", zap.String("workload_id", d.id))
//	return nil
//}
//
//// StopChan returns the channel used to tell the workload to stop.
//func (d *BasicWorkloadDriverOld) StopChan() chan<- interface{} {
//	return d.stopChan
//}
//
//// handleCriticalError is used to abort the workload due to an error from which recovery is not possible.
//func (d *BasicWorkloadDriverOld) handleCriticalError(err error) {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//
//	d.logger.Error("Workload encountered a critical error and must abort.", zap.String("workload_id", d.id), zap.Error(err))
//	if abortErr := d.abortWorkload(); abortErr != nil {
//		d.logger.Error("Failed to abort workload.", zap.String("workload_id", d.workload.GetId()), zap.Error(abortErr))
//	}
//
//	d.workload.UpdateTimeElapsed()
//	d.workload.SetState(Erred)
//	d.workload.SetErrorMessage(err.Error())
//}
//
//// abortWorkload manually aborts the workload.
//// Clean up any sessions/kernels that were created.
//func (d *BasicWorkloadDriverOld) abortWorkload() error {
//	d.logger.Warn("Aborting workload.", zap.String("workload_id", d.id))
//
//	if d.workloadGenerator == nil {
//		d.logger.Error("Cannot stop workload. Workload Generator is nil.", zap.String("workload_id", d.id))
//		panic("Cannot stop workload. Workload Generator is nil.")
//	}
//
//	d.workloadGenerator.StopGeneratingWorkload()
//
//	// TODO(Ben): Clean-up any sessions/kernels.
//	d.logger.Warn("TODO: Clean up sessions and kernels.")
//	return nil
//}
//
//// incrementClockTime sets the d.clockTime clock to the given timestamp, verifying that the new timestamp is either
//// equal to or occurs after the old one.
////
//// incrementClockTime returns a tuple where the first element is the new time, and the second element is the difference
//// between the new time and the old time.
//func (d *BasicWorkloadDriverOld) incrementClockTime(time time.Time) (time.Time, time.Duration, error) {
//	newTime, timeDifference, err := d.clockTime.IncreaseClockTimeTo(time)
//
//	if err != nil {
//		d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
//	}
//
//	return newTime, timeDifference, err // err will be nil if everything was OK.
//}
//
//// Start the simulation.
//func (d *BasicWorkloadDriverOld) bootstrapSimulation() error {
//	// Get the first event.
//	firstEvent := <-d.eventChan
//
//	d.sugaredLogger.Infof("Received first event for workload %s: %v", d.workload.GetId(), firstEvent)
//
//	// Save the timestamp information.
//	if d.performClockTicks {
//		// Set the d.currentTick to the timestamp of the event.
//		_, _, err := d.currentTick.IncreaseClockTimeTo(firstEvent.Timestamp)
//		if err != nil {
//			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
//			return err
//		}
//
//		_, _, err = d.clockTime.IncreaseClockTimeTo(firstEvent.Timestamp)
//		if err != nil {
//			d.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
//			return err
//		}
//
//		d.logger.Debug("Initialized current tick.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.Time("timestamp", firstEvent.Timestamp))
//	}
//
//	// Handle the event. Basically, just enqueue it in the EventQueue.
//	d.eventQueue.EnqueueEvent(firstEvent)
//
//	return nil
//}
//
//// publishStatisticsReport writes the current Statistics struct attached to the internalWorkload to the CSV file.
//func (d *BasicWorkloadDriverOld) publishStatisticsReport() {
//	clusterStatistics, err := d.refreshClusterStatistics(true, false)
//	if err != nil {
//		d.logger.Error("Failed to refresh cluster statistics.", zap.Error(err))
//
//		if d.notifyCallback != nil {
//			go d.notifyCallback(&proto.Notification{
//				Id:    uuid.NewString(),
//				Title: "Failed to Refresh Cluster Statistics",
//				Message: fmt.Sprintf("Failed to refresh cluster statistics during workload %s (ID=%s)",
//					d.workload.WorkloadName(), d.workload.GetId()),
//				Panicked:         false,
//				NotificationType: domain.WarningNotification.Int32(),
//			})
//		}
//	}
//
//	stats := d.workload.GetStatistics()
//	stats.ClusterStatistics = clusterStatistics
//	PatchCSVHeader(stats)
//
//	d.outputFileMutex.Lock()
//
//	if d.appendToOutputFile {
//		err = gocsv.MarshalWithoutHeaders([]*Statistics{stats}, d.outputFile)
//	} else {
//		err = gocsv.Marshal([]*Statistics{stats}, d.outputFile)
//		d.appendToOutputFile = true
//	}
//
//	d.outputFileMutex.Unlock()
//
//	// If marshalError is not nil, then we'll either join it with the previous error, if the previous error is non-nil,
//	// or we'll just assign err to equal the
//	if err != nil {
//		d.logger.Error("Failed to publish statistics report.", zap.Error(err))
//
//		if d.notifyCallback != nil {
//			go d.notifyCallback(&proto.Notification{
//				Id:    uuid.NewString(),
//				Title: "Failed to Publish Statistics Report",
//				Message: fmt.Sprintf("Failed to publish statistics report for workload %s (ID=%s)",
//					d.workload.WorkloadName(), d.workload.GetId()),
//				Panicked:         false,
//				NotificationType: domain.WarningNotification.Int32(),
//			})
//		}
//	}
//}
//
//// DriveWorkload accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
//// This issues clock ticks as events are submitted.
////
//// DriveWorkload should be called from its own goroutine.
//func (d *BasicWorkloadDriverOld) DriveWorkload() {
//	var err error
//
//	outputSubdir := time.Now().Format("01-02-2006 15:04:05")
//	outputSubdir = strings.ReplaceAll(outputSubdir, ":", "-")
//	outputSubdir = fmt.Sprintf("%s - %s", outputSubdir, d.workload.GetId())
//	outputSubdirectoryPath := filepath.Join(d.outputFileDirectory, outputSubdir)
//
//	err = os.MkdirAll(outputSubdirectoryPath, os.ModePerm)
//	if err != nil {
//		d.logger.Error("Failed to create parent directories for workload .CSV output.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("path", outputSubdirectoryPath),
//			zap.Error(err))
//		d.handleCriticalError(err)
//		return
//	}
//
//	d.outputFilePath = filepath.Join(outputSubdirectoryPath, "workload_stats.csv")
//
//	d.outputFileMutex.Lock()
//	d.outputFile, err = os.OpenFile(d.outputFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
//	d.outputFileMutex.Unlock()
//
//	if err != nil {
//		d.logger.Error("Failed to create .CSV output file for workload.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("path", outputSubdirectoryPath),
//			zap.Error(err))
//		d.handleCriticalError(err)
//		return
//	}
//
//	// First, clear the cluster statistics. This will return whatever they were before we called clear.
//	_, err = d.refreshClusterStatistics(true, true)
//	if err != nil {
//		d.logger.Error("Failed to clear and/or retrieve Cluster Statistics before beginning workload.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("reason", err.Error()))
//		d.handleCriticalError(err)
//
//		d.outputFileMutex.Lock()
//		_ = d.outputFile.Close()
//		d.outputFileMutex.Unlock()
//		return
//	}
//
//	// Fetch the freshly-cleared cluster statistics.
//	clusterStats, err := d.refreshClusterStatistics(true, false)
//	if err != nil {
//		d.logger.Error("Failed to clear and/or retrieve Cluster Statistics before beginning workload.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("reason", err.Error()))
//		d.handleCriticalError(err)
//
//		d.outputFileMutex.Lock()
//		_ = d.outputFile.Close()
//		d.outputFileMutex.Unlock()
//		return
//	}
//
//	d.workload.GetStatistics().ClusterStatistics = clusterStats
//
//	d.logger.Info("Workload Simulator has started running. Bootstrapping simulation now.",
//		zap.String("workload_id", d.id),
//		zap.String("workload_name", d.workload.WorkloadName()))
//
//	err = d.bootstrapSimulation()
//	if err != nil {
//		d.logger.Error("Failed to bootstrap workload.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("reason", err.Error()))
//		d.handleCriticalError(err)
//
//		d.outputFileMutex.Lock()
//		_ = d.outputFile.Close()
//		d.outputFileMutex.Unlock()
//		return
//	}
//
//	d.logger.Info("The simulation has started.",
//		zap.String("workload_id", d.id),
//		zap.String("workload_name", d.workload.WorkloadName()))
//
//	nextTick := d.currentTick.GetClockTime().Add(d.targetTickDuration)
//
//OUTER:
//	for {
//		// Constantly poll for events from the Workload Generator.
//		// These events are then enqueued in the EventQueue.
//		select {
//		case evt := <-d.eventChan:
//			if !d.workload.IsInProgress() {
//				d.logger.Warn("Workload is no longer running. Aborting drive procedure.",
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_state", d.workload.GetState().String()))
//
//				d.outputFileMutex.Lock()
//				_ = d.outputFile.Close()
//				d.outputFileMutex.Unlock()
//				return
//			}
//
//			// If the event occurs during this tick, then call EnqueueEvent to enqueue the event in the EventQueue.
//			if evt.Timestamp.Before(nextTick) {
//				d.sugaredLogger.Debugf("\"%s\" event \"%s\" targeting session \"%s\" DOES occur before next tick [%v]. Enqueuing event now (timestamp=%v).",
//					evt.Name.String(), evt.ID, evt.SessionID(), nextTick, evt.Timestamp)
//
//				d.workload.SetNextEventTick(d.convertTimestampToTickNumber(evt.Timestamp))
//				d.workload.SetNextExpectedEventName(evt.Name)
//				d.workload.SetNextExpectedEventSession(evt.SessionId)
//
//				d.eventQueue.EnqueueEvent(evt)
//			} else {
//				d.sugaredLogger.Debugf("\"%s\" event \"%s\" targeting session \"%s\" does NOT occur before next tick [%v] (i.e., tick #%d). Will have to issue clock ticks until we get to event's timestamp of [%v] (i.e., tick #%d).",
//					evt.Name.String(), evt.ID, evt.SessionID(), nextTick, nextTick.Unix()/d.targetTickDurationSeconds, evt.Timestamp, evt.Timestamp.Unix()/d.targetTickDurationSeconds)
//
//				d.workload.SetNextEventTick(d.convertTimestampToTickNumber(evt.Timestamp))
//				d.workload.SetNextExpectedEventName(evt.Name)
//				d.workload.SetNextExpectedEventSession(evt.SessionId)
//
//				// The event occurs in the next tick. Update the current tick clock, issue/perform a tick-trigger, and then process the event.
//				err = d.issueClockTicks(evt.Timestamp)
//				if err != nil {
//					d.logger.Error("Critical error occurred while attempting to increment clock time.",
//						zap.String("workload_id", d.id),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("error-message", err.Error()))
//					d.handleCriticalError(err)
//					break OUTER
//				}
//				nextTick = d.currentTick.GetClockTime().Add(d.targetTickDuration)
//				d.eventQueue.EnqueueEvent(evt)
//			}
//		case <-d.workloadEventGeneratorCompleteChan:
//			d.logger.Debug("Drivers finished generating events.",
//				zap.Int("events_still_enqueued", d.eventQueue.Len()),
//				zap.String("workload_id", d.id),
//				zap.String("workload_name", d.workload.WorkloadName()))
//
//			// Continue issuing ticks until the cluster is finished.
//			for d.eventQueue.Len() > 0 {
//				err = d.issueClockTicks(nextTick)
//				if err != nil {
//					d.logger.Error("Critical error occurred while attempting to increment clock time.",
//						zap.String("workload_id", d.id),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("error-message", err.Error()))
//					d.handleCriticalError(err)
//					break OUTER
//				}
//
//				nextTick = d.currentTick.GetClockTime().Add(d.targetTickDuration)
//			}
//
//			// Signal to the goroutine running the BasicWorkloadDriverOld::ProcessWorkloadEvents method that the workload has completed successfully.
//			d.workloadExecutionCompleteChan <- struct{}{}
//
//			break OUTER
//		}
//	}
//
//	// Publish one last statistics report, which will also fetch the Cluster Statistics one last time.
//	d.publishStatisticsReport()
//
//	d.outputFileMutex.Lock()
//	_ = d.outputFile.Close()
//	d.outputFileMutex.Unlock()
//}
//
//func (d *BasicWorkloadDriverOld) PauseWorkload() error {
//	d.pauseMutex.Lock()
//	defer d.pauseMutex.Unlock()
//
//	if d.paused {
//		d.logger.Error("Cannot pause workload. Workload is already paused.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//
//		return ErrWorkloadAlreadyPaused
//	}
//
//	d.logger.Debug("Pausing workload.",
//		zap.String("workload_id", d.id),
//		zap.String("workload_name", d.workload.WorkloadName()))
//
//	d.paused = true
//	return d.workload.SetPausing()
//}
//
//func (d *BasicWorkloadDriverOld) UnpauseWorkload() error {
//	d.pauseMutex.Lock()
//	defer d.pauseMutex.Unlock()
//
//	if !d.paused {
//		d.logger.Error("Cannot unpause workload. Workload is already unpaused.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//
//		return ErrWorkloadAlreadyUnpaused
//	}
//
//	d.logger.Debug("Unpausing workload.",
//		zap.String("workload_id", d.id),
//		zap.String("workload_name", d.workload.WorkloadName()))
//
//	// We'll actually call d.workload.Unpause() later, in the goroutine triggering the clock ticks.
//	d.paused = false
//	d.pauseCond.Broadcast()
//	return nil
//}
//
//// handlePause is called to check if the workload is paused and, if so, then block until we're unpaused.
//func (d *BasicWorkloadDriverOld) handlePause() error {
//	d.pauseMutex.Lock()
//	defer d.pauseMutex.Unlock()
//
//	pausedWorkload := false
//	for d.paused {
//		// If we haven't transitioned the workload from 'pausing' to 'paused' yet, then do so now.
//		if !pausedWorkload {
//			// Transition the workload from 'pausing' to 'paused'.
//			// If this fails, then we'll just unpause and continue onwards.
//			err := d.workload.SetPaused()
//			if err != nil {
//				d.logger.Error("Failed to transition workload to 'paused' state.",
//					zap.String("workload_id", d.id),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("workload_state", d.workload.GetState().String()),
//					zap.Error(err))
//
//				// We failed to pause the workload, so unpause ourselves and just continue.
//				_ = d.workload.Unpause() // We don't care if this fails, as long as the workload is running.
//				d.paused = false
//
//				// Make sure that the workload is actively running.
//				if !d.workload.IsRunning() {
//					d.logger.Error("Workload is not actively running anymore. We're stuck.",
//						zap.String("workload_id", d.id),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("workload_state", d.workload.GetState().String()),
//						zap.Error(err))
//					return err
//				}
//
//				// Just return. We failed to pause the workload. Just try to keep going.
//				return nil
//			}
//
//			pausedWorkload = true
//		}
//
//		d.logger.Debug("Workload is paused. Waiting to issue next tick.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//
//		d.workload.PauseWaitBeginning()
//		d.pauseCond.Wait()
//	}
//
//	// If we paused the workload, then let's unpause it before returning.
//	if pausedWorkload {
//		err := d.workload.Unpause()
//		if err != nil {
//			d.logger.Error("Failed to unpause workload.",
//				zap.String("workload_id", d.id),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.Error(err))
//
//			// If the workload is not actively running, then it's in an unexpected state, so we'll return an error.
//			// Otherwise, we'll just return nil and ignore the fact that we failed to unpause the workload.
//			if !d.workload.IsRunning() {
//				d.logger.Error("Workload is not actively running anymore. We're stuck.",
//					zap.String("workload_id", d.id),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("workload_state", d.workload.GetState().String()),
//					zap.Error(err))
//
//				// Return the error.
//				return err
//			}
//		}
//	}
//
//	return nil
//}
//
//// issueClockTicks issues clock ticks until the d.currentTick clock has caught up to the given timestamp.
//// The given timestamp should correspond to the timestamp of the next event generated by the Workload Generator.
//// The Workload Simulation will simulate everything up until this next event is ready to process.
//// Then we will continue consuming all events for the current tick in the SimulationDriver::DriveSimulation function.
//func (d *BasicWorkloadDriverOld) issueClockTicks(timestamp time.Time) error {
//	if !d.performClockTicks {
//		return nil
//	}
//
//	currentTick := d.currentTick.GetClockTime()
//
//	// We're going to issue clock ticks up until the specified timestamp.
//	// Calculate how many ticks that requires so we can perform a quick sanity
//	// check at the end to verify that we issued the correct number of ticks.
//	numTicksToIssue := int64((timestamp.Sub(currentTick)) / d.targetTickDuration)
//
//	// This is just for debugging/logging purposes.
//	nextEventAtTime, err := d.eventQueue.GetTimestampOfNextReadyEvent()
//	if err == nil {
//		timeUntilNextEvent := nextEventAtTime.Sub(currentTick)
//		numTicksTilNextEvent := int64(timeUntilNextEvent / d.targetTickDuration)
//
//		d.logger.Debug("Preparing to issue clock ticks.",
//			zap.Time("current_tick", currentTick),
//			zap.Time("target_timestamp", timestamp),
//			zap.Int64("num_ticks_to_issue", numTicksToIssue),
//			zap.Time("next_event_timestamp", nextEventAtTime),
//			zap.Duration("time_til_next_event", timeUntilNextEvent),
//			zap.Int64("num_ticks_til_next_event", numTicksTilNextEvent),
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//	} else {
//		d.logger.Debug("Preparing to issue clock ticks. There are no events enqueued.",
//			zap.Time("current_tick", currentTick),
//			zap.Time("target_timestamp", timestamp),
//			zap.Int64("num_ticks_to_issue", numTicksToIssue),
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//	}
//
//	// Issue clock ticks.
//	var numTicksIssued int64 = 0
//	for timestamp.After(currentTick) && d.workload.IsInProgress() {
//		if err := d.handlePause(); err != nil {
//			return err
//		}
//
//		tickStart := time.Now()
//
//		// Increment the clock.
//		tick, err := d.currentTick.IncrementClockBy(d.targetTickDuration)
//		if err != nil {
//			d.logger.Error("Error while incrementing clock time.",
//				zap.Duration("tick-duration", d.targetTickDuration),
//				zap.String("workload_id", d.id),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.Error(err))
//			return err
//		}
//
//		tickNumber := int(d.convertTimestampToTickNumber(tick))
//		d.logger.Debug("Issuing tick.",
//			zap.Int("tick_number", tickNumber),
//			zap.Time("tick_timestamp", tick),
//			zap.Int("num_events", d.eventQueue.Len()),
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//
//		// Trigger the clock ticker, which will prompt the other goroutine within the workload driver to process events and whatnot for this tick.
//		d.clockTrigger.Trigger(tick)
//		numTicksIssued += 1
//		currentTick = d.currentTick.GetClockTime()
//
//		// If the workload is no longer running, then we'll just return.
//		// This can happen if the user manually stopped the workload, or if the workload encountered a critical error.
//		if !d.workload.IsInProgress() {
//			d.logger.Warn("Workload is no longer running. Aborting post-issue-clock-tick procedure early.",
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("workload_id", d.workload.GetId()),
//				zap.String("workload_state", d.workload.GetState().String()))
//
//			return nil
//		}
//
//		// How long the tick took to process.
//		// If it took less than the target amount of time, then we'll sleep for a bit.
//		tickElapsedBase := time.Since(tickStart)
//		tickRemaining := time.Duration(d.timescaleAdjustmentFactor * float64(d.targetTickDuration-tickElapsedBase))
//
//		// Verify that the issuing of the tick did not exceed the specified real-clock-time that a tick should last.
//		// TODO: Handle this more elegantly, such as by decreasing the length of subsequent ticks or something?
//		if tickRemaining < 0 {
//			d.logger.Warn("Issuing clock tick lasted too long.",
//				zap.Int("tick_number", tickNumber),
//				zap.Time("tick_timestamp", tick),
//				zap.Duration("time_elapsed", tickElapsedBase),
//				zap.Duration("target_tick_duration", d.targetTickDuration),
//				zap.Float64("timescale_adjustment_factor", d.timescaleAdjustmentFactor),
//				zap.String("workload_id", d.id),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("workload_state", d.workload.GetState().String()))
//		} else {
//			// Simulate the remainder of the tick -- however much time is left.
//			d.logger.Debug("Sleeping to simulate remainder of tick.",
//				zap.Int("tick_number", tickNumber),
//				zap.Time("tick_timestamp", tick),
//				zap.Duration("time_elapsed", tickElapsedBase),
//				zap.Duration("target_tick_duration", d.targetTickDuration),
//				zap.Float64("timescale_adjustment_factor", d.timescaleAdjustmentFactor),
//				zap.Duration("sleep_time", tickRemaining),
//				zap.String("workload_id", d.id),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("workload_state", d.workload.GetState().String()))
//			time.Sleep(tickRemaining)
//		}
//
//		tickDuration := time.Since(tickStart)
//		tickDurationSec := decimal.NewFromFloat(tickDuration.Seconds())
//		d.checkForLongTick(tickNumber, tickDurationSec)
//
//		// Update the average now, after we check if the tick was too long.
//		d.tickDurationsSecondsMovingWindow.Add(tickDurationSec)
//		d.tickDurationsAll = append(d.tickDurationsAll, tickDuration)
//		d.workload.AddFullTickDuration(tickDuration)
//	}
//
//	// Sanity check to ensure that we issued the correct/expected number of ticks.
//	if numTicksIssued != numTicksToIssue {
//		time.Sleep(time.Second * 5)
//
//		if d.workload.IsInProgress() {
//			d.logger.Error("Issued incorrect number of ticks, and workload is still running.",
//				zap.Int64("expected_ticks", numTicksToIssue),
//				zap.Int64("ticks_issued", numTicksIssued),
//				zap.String("workload_id", d.id),
//				zap.String("workload_name", d.workload.WorkloadName()))
//		} else {
//			d.logger.Warn("Issued incorrect number of ticks, but workload is no longer running, so that's probably why.",
//				zap.Int64("expected_ticks", numTicksToIssue),
//				zap.Int64("ticks_issued", numTicksIssued),
//				zap.String("workload_id", d.id),
//				zap.String("workload_name", d.workload.WorkloadName()))
//		}
//	}
//
//	return nil
//}
//
//// checkForLongTick checks if the last tick's duration was notably longer than the average tick duration.
//// If so, then a warning notification is sent to the frontend to alert the user that the tick took a
//// long time to process, and that something may be wrong.
//func (d *BasicWorkloadDriverOld) checkForLongTick(tickNumber int, tickDurationSec decimal.Decimal) {
//	// If there's at least minN tick durations in the window, then we'll check if the last tick was unusually long.
//	// minN is either 3, or a smaller value if the window size is set to something smaller than 5.
//	minN := math.Min(float64(d.tickDurationsSecondsMovingWindow.Window()), 3)
//	if d.tickDurationsSecondsMovingWindow.N() < int64(minN) {
//		return // Insufficient entries for a meaningful comparison
//	}
//
//	avgTickDurationSec := d.tickDurationsSecondsMovingWindow.Avg()
//	stdDevTickDuration := d.tickDurationsSecondsMovingWindow.SampleStandardDeviation()
//
//	if tickDurationSec.GreaterThanOrEqual(avgTickDurationSec.Mul(decimal.NewFromFloat(3))) {
//		d.logger.Warn("Last tick took longer than expected.",
//			zap.Int("tick_number", tickNumber),
//			zap.String("tick_duration_sec", tickDurationSec.StringFixed(4)),
//			zap.String("avg_tick_duration_sec", avgTickDurationSec.StringFixed(4)),
//			zap.String("sample_std_dev_tick_dur_sec", stdDevTickDuration.StringFixed(4)),
//			zap.Int64("moving_avg_window_size", d.tickDurationsSecondsMovingWindow.Window()))
//
//		//if d.notifyCallback != nil {
//		//	d.notifyCallback(&proto.Notification{
//		//		Id:    uuid.NewString(),
//		//		Title: fmt.Sprintf("Tick #%d of Workload %s Took a Long Time", tickNumber, d.workload.GetId()),
//		//		Message: fmt.Sprintf("Tick duration: %s seconds. Average tick duration: %s seconds. Standard deviation (of tick duration in seconds): %s seconds.",
//		//			tickDurationSec.StringFixed(3), avgTickDurationSec.StringFixed(3), stdDevTickDuration.StringFixed(3)),
//		//		Panicked:         false,
//		//		NotificationType: domain.WarningNotification.Int32(),
//		//	})
//		//}
//	}
//}
//
//// ProcessWorkload accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
//// This processes events in response to clock ticks.
////
//// If there is a critical error that causes the workload to be terminated prematurely/aborted, then that error is returned.
//// If the workload is able to complete successfully, then nil is returned.
////
//// ProcessWorkload should be called from its own goroutine.
//func (d *BasicWorkloadDriverOld) ProcessWorkloadEvents() {
//	d.mu.Lock()
//
//	if d.workload == nil {
//		d.logger.Error("Workload is nil. Cannot process it.")
//		return
//	}
//
//	d.workloadStartTime = time.Now()
//	// Add an event for the workload starting.
//	d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
//		WithEventId(uuid.NewString()).
//		WithSessionId("-").
//		WithEventName(domain.EventWorkloadStarted).
//		WithEventTimestamp(d.clockTime.GetClockTime()).
//		WithProcessedAtTime(d.workloadStartTime))
//
//	if d.workloadPreset != nil {
//		d.logger.Info("Starting preset-based workload.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("workload-preset-name", d.workloadPreset.GetName()),
//			zap.String("workload-preset-key", d.workloadPreset.GetKey()))
//	} else if d.workloadSessions != nil {
//		d.logger.Info("Starting template-based workload.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//	} else {
//		d.logger.Info("Starting workload.",
//			zap.String("workload_id", d.id),
//			zap.String("workload_name", d.workload.WorkloadName()))
//	}
//
//	d.workloadGenerator = generator.NewWorkloadGenerator(d.opts, d.atom, d)
//	d.mu.Unlock()
//
//	if d.workload.IsPresetWorkload() {
//		go func() {
//			presetWorkload := d.workload.(*Preset)
//			err := d.workloadGenerator.GeneratePresetWorkload(d, presetWorkload, presetWorkload.WorkloadPreset, d.workloadRegistrationRequest)
//			if err != nil {
//				d.logger.Error("Failed to drive/generate preset workload.",
//					zap.String("workload_id", d.id),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.Error(err))
//			}
//		}()
//	} else if d.workload.IsTemplateWorkload() {
//		go func() {
//			err := d.workloadGenerator.GenerateTemplateWorkload(d, d.workloadSessions, d.workloadRegistrationRequest)
//			if err != nil {
//				d.logger.Error("Failed to drive/generate templated workload.",
//					zap.String("workload_id", d.id),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.Error(err))
//			}
//		}()
//	} else {
//		panic(fmt.Sprintf("Workload is of presently-unsuporrted type: \"%s\" -- cannot generate workload.", d.workload.GetKind()))
//	}
//
//	d.logger.Info("The Workload Driver has started running.",
//		zap.String("workload_id", d.id),
//		zap.String("workload_name", d.workload.WorkloadName()))
//
//	numTicksServed := 0
//	d.servingTicks.Store(true)
//	for d.workload.IsInProgress() {
//		select {
//		case tick := <-d.ticker.TickDelivery:
//			{
//				d.logger.Debug("Received tick.",
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.Time("tick", tick))
//
//				// Handle the tick.
//				if err := d.handleTick(tick); err != nil {
//					d.logger.Error("Critical error occurred when attempting to increase clock time.",
//						zap.String("workload_id", d.id),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.Error(err))
//					d.handleCriticalError(err)
//
//					return
//				}
//
//				numTicksServed += 1
//			}
//		case err := <-d.errorChan:
//			{
//				d.logger.Error("Received error.",
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.Error(err))
//				d.handleCriticalError(err)
//				return // We're done, so we can return.
//			}
//		case <-d.stopChan:
//			{
//				d.logger.Info("Workload has been instructed to terminate early.",
//					zap.String("workload_id", d.id),
//					zap.String("workload_name", d.workload.WorkloadName()))
//
//				abortError := d.abortWorkload()
//				if abortError != nil {
//					d.logger.Error("Error while aborting workload.",
//						zap.String("workload_id", d.id),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.Error(abortError))
//				}
//
//				return // We're done, so we can return.
//			}
//		case <-d.workloadExecutionCompleteChan: // This is placed after eventChan so that all events are processed first.
//			{
//				d.workloadComplete()
//				return
//			}
//		}
//	}
//}
//
//// workloadComplete is called when the BasicWorkloadDriverOld receives a signal on its workloadExecutionCompleteChan
//// field informing it that the workload has completed successfully.
////
//// workloadComplete accepts a *sync.WaitGroup that is used to notify the caller when the workload has completed.
//func (d *BasicWorkloadDriverOld) workloadComplete() {
//	d.workload.SetWorkloadCompleted()
//
//	var ok bool
//	d.workloadEndTime, ok = d.workload.GetEndTime() // time.Now()
//	if !ok {
//		panic("`ok` should have been `true`")
//	}
//
//	// Add an event for the workload stopping.
//	d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
//		WithEventId(uuid.NewString()).
//		WithSessionId("-").
//		WithEventName(domain.EventWorkloadComplete).
//		WithEventTimestamp(d.clockTime.GetClockTime()).
//		WithProcessedAtTime(d.workloadEndTime))
//
//	d.logger.Info("The Workload Generator has finished generating events.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()))
//	d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.GetStartTime())), zap.Any("workload-start-time", d.workload.GetStartTime()), zap.Any("workload-end-time", d.workloadEndTime), zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()))
//
//	d.sessionConnectionsMutex.Lock()
//	defer d.sessionConnectionsMutex.Unlock()
//
//	d.sugaredLogger.Debugf("There is/are %d sessions.", len(d.sessionConnections))
//	for sessionId, sessionConnection := range d.sessionConnections {
//		kernel := sessionConnection.Kernel()
//		if kernel == nil {
//			continue
//		}
//
//		stdout := kernel.Stdout()
//		stderr := kernel.Stderr()
//
//		d.sugaredLogger.Debugf("Stdout IOPub messages received by Session %s (%d):", sessionId, len(stdout))
//		for _, stdoutMessage := range stdout {
//			d.sugaredLogger.Debugf(stdoutMessage)
//		}
//
//		d.sugaredLogger.Debugf("Stderr IOPub messages received by Session %s (%d):", sessionId, len(stderr))
//		for _, stderrMessage := range stderr {
//			d.sugaredLogger.Debugf(stderrMessage)
//		}
//	}
//}
//
//// convertTimestampToTickNumber converts the given tick, which is specified in the form of a time.Time,
//// and returns what "tick number" that tick is.
////
//// Basically, you just convert the timestamp to its unix epoch timestamp (in seconds), and divide by the
//// trace step value (also in seconds).
//func (d *BasicWorkloadDriverOld) convertTimestampToTickNumber(tick time.Time) int64 {
//	return tick.Unix() / d.targetTickDurationSeconds
//}
//
//// Handle a tick during the execution of a workload.
////
//// This should just be called by BasicWorkloadDriverOld::ProcessWorkloadEvents.
////
//// The 'tick' parameter is the clock time of the latest tick -- the tick that we're processing here.
////
//// This only returns critical errors.
//func (d *BasicWorkloadDriverOld) handleTick(tick time.Time) error {
//	tickStart := time.Now()
//	_, _, err := d.currentTick.IncreaseClockTimeTo(tick)
//	if err != nil {
//		return err
//	}
//
//	coloredOutput := ansi.Color(fmt.Sprintf("Serving tick: %v (processing everything up to %v)", tick, tick), "blue")
//	d.logger.Debug(coloredOutput, zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()))
//	d.ticksHandled.Add(1)
//
//	// If there are no events processed this tick, then we still need to increment the clock time so we're in-line with the simulation.
//	// Check if the current clock time is earlier than the start of the previous tick. If so, increment the clock time to the beginning of the tick.
//	prevTickStart := tick.Add(-d.targetTickDuration)
//	if d.clockTime.GetClockTime().Before(prevTickStart) {
//		if _, _, err := d.incrementClockTime(prevTickStart); err != nil {
//			return nil
//		}
//	}
//
//	// Process "start/stop training" events.
//	d.processEventsForTick(tick)
//
//	d.doneServingTick(tickStart)
//
//	if d.outputFile != nil {
//		d.publishStatisticsReport()
//	}
//
//	return nil
//}
//
//// Called from BasicWorkloadDriverOld::ProcessWorkloadEvents at the end of serving a tick to signal to the Ticker/Trigger interface that the listener (i.e., the Cluster) is done.
//func (d *BasicWorkloadDriverOld) doneServingTick(tickStart time.Time) {
//	tickDuration := time.Since(tickStart)
//	tick := d.ticksHandled.Load()
//	numEventsEnqueued := d.eventQueue.Len()
//	d.workload.TickCompleted(tick, d.clockTime.GetClockTime())
//
//	if d.sugaredLogger.Level() == zapcore.DebugLevel {
//		d.sugaredLogger.Debugf("[%v] Done serving tick #%d. "+
//			"Real-world tick duration: %v. "+
//			"Total time elapsed for workload %s: %v. "+
//			"There is/are %d more session event(s) enqueued right now.",
//			d.clockTime.GetClockTime(), tick, tickDuration, d.workload.GetId(), d.workload.GetTimeElapsed(), numEventsEnqueued)
//	}
//
//	d.ticker.Done()
//}
//
//// WorkloadExecutionCompleteChan returns the channel that is used to signal
//// that the workload has successfully processed all events and is complete.
//func (d *BasicWorkloadDriverOld) WorkloadExecutionCompleteChan() chan interface{} {
//	return d.workloadExecutionCompleteChan
//}
//
//// WorkloadEventGeneratorCompleteChan returns the channel used to signal that the generators have submitted all events.
//// Once all remaining, already-enqueued events have been processed, the workload will be complete.
//func (d *BasicWorkloadDriverOld) WorkloadEventGeneratorCompleteChan() chan interface{} {
//	return d.workloadEventGeneratorCompleteChan
//}
//
//// RegisterApproximateFinalTick is used to register what is the approximate final tick of the workload
//// after iterating over all sessions and all training events.
//func (d *BasicWorkloadDriverOld) RegisterApproximateFinalTick(approximateFinalTick int64) {
//	d.workload.RegisterApproximateFinalTick(approximateFinalTick)
//}
//
//// EventQueue returns the event queue for this workload.
//func (d *BasicWorkloadDriverOld) EventQueue() *event_queue.EventQueue {
//	return d.eventQueue
//}
//
//func (d *BasicWorkloadDriverOld) processSessionReadyEvents(sessionReadyEvents []*domain.Event, tick time.Time, timeoutInterval time.Duration) {
//	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
//	defer cancel()
//
//	sessionFinishedChannel := make(chan string, len(sessionReadyEvents))
//
//	// Keep track of which sessions have finished.
//	// The most important thing is the number of sessions that have finished,
//	// but knowing the session IDs could be useful for logging/debugging.
//	responsesReceived := map[string]struct{}{}
//	expectedNumResponses := len(sessionReadyEvents)
//	remainingSessions := make([]string, 0, len(sessionReadyEvents))
//	aborted := false
//
//	for idx, sessionReadyEvent := range sessionReadyEvents {
//		remainingSessions = append(remainingSessions, sessionReadyEvent.SessionId)
//		go d.handleSessionReadyEvent(sessionReadyEvent, idx, sessionFinishedChannel)
//	}
//
//	startedWaitingAt := time.Now()
//	// Keep looping until we've either received all responses, or until the context's timeout expires and we give up.
//	for len(responsesReceived) < expectedNumResponses && !aborted {
//		d.logger.Debug("Waiting for goroutines to finish handling session-creation events.",
//			zap.Int("num_responses_received", len(responsesReceived)),
//			zap.Int("num_responses_expected", expectedNumResponses),
//			zap.Strings("remaining_sessions", remainingSessions),
//			zap.Time("tick", tick),
//			zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("workload_id", d.workload.GetId()))
//
//		select {
//		case sessionId := <-sessionFinishedChannel:
//			{
//				// Make note that we the session has finished.
//				responsesReceived[sessionId] = struct{}{}
//
//				// Let's remove the session ID from the slice of remaining sessions.
//				idx := indexOf(remainingSessions, sessionId)
//				if idx != -1 { // This should never be -1, but just in case, we'll check...
//					remainingSessions = removeIndex(remainingSessions, idx)
//				} else {
//					d.logger.Error("indexOf returned -1",
//						zap.String("session_id", sessionId),
//						zap.Int("num_responses_received", len(responsesReceived)),
//						zap.Int("num_responses_expected", expectedNumResponses),
//						zap.Strings("remaining_sessions", remainingSessions),
//						zap.Time("tick", tick),
//						zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("workload_id", d.workload.GetId()))
//				}
//
//				d.logger.Debug("Session was created successfully.",
//					zap.String("session_id", sessionId),
//					zap.Int("num_responses_received", len(responsesReceived)),
//					zap.Int("num_responses_expected", expectedNumResponses),
//					zap.Strings("remaining_sessions", remainingSessions),
//					zap.Time("tick", tick),
//					zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("workload_id", d.workload.GetId()))
//
//				d.workload.UpdateTimeElapsed()
//			}
//		case <-ctx.Done():
//			{
//				d.logger.Error("Timed-out waiting for sessions to finish processing their events.",
//					zap.Int("num_responses_received", len(responsesReceived)),
//					zap.Int("num_responses_expected", expectedNumResponses),
//					zap.Strings("remaining_sessions", remainingSessions),
//					zap.Time("tick", tick),
//					zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("workload_id", d.workload.GetId()))
//
//				if err := ctx.Err(); err != nil {
//					d.logger.Error("There was an error attached to the context when we timed-out.",
//						zap.Time("tick", tick),
//						zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("workload_id", d.workload.GetId()),
//						zap.Error(err))
//				}
//
//				// Take note of all the sessions that did not finish being created during their window.
//				// We'll just discard those sessions, ignoring any future events targeting those sessions.
//				for _, sessionId := range remainingSessions {
//					d.logger.Warn("Session timed-out during creation. Disabling session.",
//						zap.String("session_id", sessionId),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("workload_id", d.workload.GetId()))
//
//					err := d.workload.SessionDiscarded(sessionId)
//					if err != nil {
//						d.logger.Error("Failed to disable Session that timed-out during creation.",
//							zap.String("session_id", sessionId),
//							zap.String("workload_name", d.workload.WorkloadName()),
//							zap.String("workload_id", d.workload.GetId()),
//							zap.Error(err))
//					}
//
//					misbehavingSession := d.GetSession(sessionId)
//					if misbehavingSession == nil {
//						d.logger.Error("Failed to load (misbehaving) session from (regular) session map.",
//							zap.String("session_id", sessionId),
//							zap.String("workload_name", d.workload.WorkloadName()),
//							zap.String("workload_id", d.workload.GetId()))
//						continue
//					}
//
//					// Record that the session failed to process all of its events in this tick.
//					// This should never come up again, since we disabled the session, but nevertheless
//					// it should be recorded, as the session did fail to process its events.
//					misbehavingSession.TickFailed()
//
//					d.misbehavingSessionsMutex.Lock()
//					d.misbehavingSessions[sessionId] = misbehavingSession
//					d.misbehavingSessionsMutex.Unlock()
//				}
//
//				// Give up.
//				aborted = true
//			}
//		}
//	}
//}
//
//// processEventsForTick processes events in chronological/simulation order.
//// This accepts the "current tick" as an argument. The current tick is basically an upper-bound on the times for
//// which we'll process an event. For example, if `tick` is 19:05:00, then we will process all cluster and session
//// events with timestamps that are (a) equal to 19:05:00 or (b) come before 19:05:00. Any events with timestamps
//// that come after 19:05:00 will not be processed until the next tick.
//func (d *BasicWorkloadDriverOld) processEventsForTick(tick time.Time) {
//	var (
//		// Map from session ID to a slice of events that the session is supposed to process in this tick.
//		sessionEventMap = make(map[string][]*domain.Event)
//
//		// 'Session Ready' events.
//		sessionReadyEvents = make([]*domain.Event, 0)
//	)
//
//	// Extract all the "session-ready" events for this tick.
//	for d.eventQueue.HasEventsForTick(tick) && d.eventQueue.Peek(tick).Name == domain.EventSessionReady {
//		evt := d.eventQueue.Pop(tick)
//
//		if evt == nil {
//			// Since 'HasEventsForTick' returned true, 'Pop' should return a valid value.
//			// If it doesn't, then in theory we could just ignore it, but it shouldn't happen, so there's probably a bug.
//			// Hence, we'll panic.
//			panic(fmt.Sprintf("Expected to find valid event for tick %v.", tick))
//		}
//
//		// Collect up all the "session-ready" events.
//		if evt.Name != domain.EventSessionReady {
//			panic(fmt.Sprintf("Expected 'session-ready' event. Instead, got '%s': %v", evt.Name.String(), evt))
//		}
//
//		sessionReadyEvents = append(sessionReadyEvents, evt)
//	}
//
//	// Process any 'session-ready' events that are ready at the beginning of the tick.
//	if len(sessionReadyEvents) > 0 {
//		d.logger.Debug("Processing \"session-ready\" event(s) at the very beginning of the tick.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.Time("tick", tick),
//			zap.Int("num_events", len(sessionReadyEvents)))
//
//		d.processSessionReadyEvents(sessionReadyEvents, tick, time.Minute*3)
//	}
//
//	d.workload.UpdateTimeElapsed()
//
//	// Extract all the "session-ready" events for this tick.
//	for d.eventQueue.HasEventsForTick(tick) {
//		evt := d.eventQueue.Pop(tick)
//
//		if evt == nil {
//			// Since 'HasEventsForTick' returned true, 'Pop' should return a valid value.
//			// If it doesn't, then in theory we could just ignore it, but it shouldn't happen, so there's probably a bug.
//			// Hence, we'll panic.
//			panic(fmt.Sprintf("Expected to find valid event for tick %v.", tick))
//		}
//
//		// Get the list of events for the particular session, creating said list if it does not already exist.
//		sessionId := evt.Data.(domain.PodData).GetPod()
//		sessionEvents, ok := sessionEventMap[sessionId]
//		if !ok {
//			// If the slice of events doesn't exist already, then create it.
//			sessionEvents = make([]*domain.Event, 0, 1)
//		}
//
//		// Add the event to the slice of events for this session.
//		sessionEvents = append(sessionEvents, evt)
//		// Put the updated list back into the map.
//		sessionEventMap[sessionId] = sessionEvents
//	}
//
//	// If we dequeued 0 events, then just return.
//	if len(sessionEventMap) == 0 {
//		return
//	}
//
//	// We'll create one goroutine per session that has events to be processed.
//	// We'll use the WaitGroup to block until all sessions have had their events processed.
//	//waitGroup.Add(len(sessionEventMap))
//	d.logger.Debug("Processing workload events.",
//		zap.Int("num_sessions", len(sessionEventMap)),
//		zap.Time("tick", tick),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("workload_id", d.workload.GetId()))
//
//	expectedNumResponses := len(sessionEventMap)
//
//	// Goroutines will send their assigned session ID via this channel to notify that they're done.
//	sessionFinishedChannel := make(chan string, expectedNumResponses)
//
//	// This is a slice of all the session IDs that have NOT yet finished. Mostly for logging/debugging purposes.
//	remainingSessions := make([]string, 0, len(sessionEventMap))
//
//	// Iterate over the session-event map, creating a goroutine to process each session's events.
//	for sessionId, events := range sessionEventMap {
//		// Create a go routine to process all the events for the particular session.
//		// This enables us to process events targeting multiple sessions in-parallel.
//		go d.processEventsForSession(sessionId, events, expectedNumResponses, sessionFinishedChannel, tick)
//
//		remainingSessions = append(remainingSessions, sessionId)
//	}
//
//	// Keep track of which sessions have finished.
//	// The most important thing is the number of sessions that have finished,
//	// but knowing the session IDs could be useful for logging/debugging.
//	responsesReceived := map[string]struct{}{}
//
//	// We'll flip abort to "true" if and when the context expires so that we don't wait around forever.
//	aborted := false
//
//	// We'll wait up to 5-minutes before giving up.
//	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
//	defer cancel()
//
//	startedWaitingAt := time.Now()
//	// Keep looping until we've either received all responses, or until the context's timeout expires and we give up.
//	for len(responsesReceived) < expectedNumResponses && !aborted {
//		d.logger.Debug("Waiting for goroutines to finish handling session events.",
//			zap.Int("num_responses_received", len(responsesReceived)),
//			zap.Int("num_responses_expected", len(sessionEventMap)),
//			zap.Strings("remaining_sessions", remainingSessions),
//			zap.Time("tick", tick),
//			zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("workload_id", d.workload.GetId()))
//
//		select {
//		case sessionId := <-sessionFinishedChannel:
//			{
//				// Make note that we the session has finished.
//				responsesReceived[sessionId] = struct{}{}
//
//				// Let's remove the session ID from the slice of remaining sessions.
//				idx := indexOf(remainingSessions, sessionId)
//				if idx != -1 { // This should never be -1, but just in case, we'll check...
//					remainingSessions = removeIndex(remainingSessions, idx)
//				} else {
//					d.logger.Error("indexOf returned -1",
//						zap.String("session_id", sessionId),
//						zap.Int("num_responses_received", len(responsesReceived)),
//						zap.Int("num_responses_expected", len(sessionEventMap)),
//						zap.Strings("remaining_sessions", remainingSessions),
//						zap.Time("tick", tick),
//						zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("workload_id", d.workload.GetId()))
//				}
//
//				d.logger.Debug("Session finished processing events.",
//					zap.String("session_id", sessionId),
//					zap.Int("num_responses_received", len(responsesReceived)),
//					zap.Int("num_responses_expected", len(sessionEventMap)),
//					zap.Strings("remaining_sessions", remainingSessions),
//					zap.Time("tick", tick),
//					zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("workload_id", d.workload.GetId()))
//
//				// Wait for all the goroutines to complete before returning.
//				d.workload.UpdateTimeElapsed()
//			}
//		case <-ctx.Done():
//			{
//				d.logger.Error("Timed-out waiting for sessions to finish processing their events.",
//					zap.Int("num_responses_received", len(responsesReceived)),
//					zap.Int("num_responses_expected", len(sessionEventMap)),
//					zap.Strings("remaining_sessions", remainingSessions),
//					zap.Time("tick", tick),
//					zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("workload_id", d.workload.GetId()))
//
//				if err := ctx.Err(); err != nil {
//					d.logger.Error("There was an error attached to the context when we timed-out.",
//						zap.Time("tick", tick),
//						zap.Duration("time_elapsed", time.Since(startedWaitingAt)),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("workload_id", d.workload.GetId()),
//						zap.Error(err))
//				}
//
//				// Take note of all the sessions that did not finish processing their events during the 5-min window.
//				for _, sessionId := range remainingSessions {
//					misbehavingSession := d.GetSession(sessionId)
//					if misbehavingSession == nil {
//						d.logger.Error("Failed to load (misbehaving) session from (regular) session map.",
//							zap.String("session_id", sessionId),
//							zap.String("workload_name", d.workload.WorkloadName()),
//							zap.String("workload_id", d.workload.GetId()))
//						continue
//					}
//
//					// Record that the session failed to process all of its events in this tick.
//					numFailedTicks := misbehavingSession.TickFailed()
//
//					d.misbehavingSessionsMutex.Lock()
//					// Check if this session has a history of poor behavior. For now, we just log a message if so.
//					if _, loaded := d.misbehavingSessions[sessionId]; loaded {
//						d.logger.Warn("Identified session with a history of bad behavior.",
//							zap.String("session_id", sessionId),
//							zap.Int("num_failed_ticks", numFailedTicks),
//							zap.String("workload_name", d.workload.WorkloadName()),
//							zap.String("workload_id", d.workload.GetId()))
//					}
//
//					// Take note of this session's behavior.
//					d.misbehavingSessions[sessionId] = misbehavingSession
//					d.misbehavingSessionsMutex.Unlock()
//				}
//
//				// Give up.
//				aborted = true
//			}
//		}
//	}
//
//	// Wait for all the goroutines to complete before returning.
//	// waitGroup.Wait()
//	d.workload.UpdateTimeElapsed()
//}
//
//// Process the given events for the specified session during the specified tick.
//// This is intended to be called within its own goroutine so that events for multiple sessions within the same tick can be processed concurrently by the driver.
//func (d *BasicWorkloadDriverOld) processEventsForSession(sessionId string, events []*domain.Event, numSessionsWithEventsToProcess int, doneChan chan<- string, tick time.Time) { // waitGroup *sync.WaitGroup
//	for idx, event := range events {
//		d.logger.Debug("Handling workload event.",
//			zap.Int("event_index", idx+1),
//			zap.Int("total_events", numSessionsWithEventsToProcess),
//			zap.String("session", sessionId),
//			zap.String("event_name", event.Name.String()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("workload_id", d.workload.GetId()))
//		err := d.handleEvent(event, tick)
//
//		// Record it as processed even if there was an error when processing the event.
//		d.workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
//			WithEventId(event.Id()).
//			WithSessionId(event.SessionID()).
//			WithEventName(event.Name).
//			WithEventTimestamp(event.Timestamp).
//			WithProcessedAtTime(time.Now()).
//			WithError(err))
//
//		if err != nil {
//			d.logger.Error("Failed to handle event workload event.",
//				zap.Int("event_index", idx+1),
//				zap.Int("total_events", numSessionsWithEventsToProcess),
//				zap.String("session", sessionId),
//				zap.String("event_name", event.Name.String()),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("workload_id", d.workload.GetId()),
//				zap.Error(err))
//
//			// If we're just sampling part of the trace, then we may get 'training-started' or 'training-ended'
//			// events for sessions that were never created. In this case, we'll just discard the events and continue.
//			if errors.Is(err, domain.ErrUnknownSession) {
//				// This error is only really noteworthy if we're not using a preset workload, as it shouldn't happen
//				// for template-based workloads.
//				//
//				// Either way, we'll ultimately just ignore the error.
//				if d.workload.IsTemplateWorkload() && d.onNonCriticalErrorOccurred != nil {
//					go d.onNonCriticalErrorOccurred(d.workload.GetId(), err)
//				}
//
//				continue
//			}
//
//			if errors.Is(err, ErrUnknownEventType) {
//				// We can just ignore this error.
//				if d.workload.IsTemplateWorkload() && d.onNonCriticalErrorOccurred != nil {
//					go d.onNonCriticalErrorOccurred(d.workload.GetId(), err)
//				}
//
//				continue
//			}
//
//			d.errorChan <- err
//
//			if d.onCriticalErrorOccurred != nil {
//				go d.onCriticalErrorOccurred(d.workload.GetId(), err)
//			}
//
//			return // We just return immediately, as the workload is going to be aborted due to the error.
//		}
//
//		d.logger.Debug("Successfully handled workload event.",
//			zap.Int("event_index", idx+1),
//			zap.Int("total_events", numSessionsWithEventsToProcess),
//			zap.String("session", sessionId),
//			zap.String("event_name", event.Name.String()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("workload_id", d.workload.GetId()))
//	}
//
//	d.logger.Debug("Finished processing workload events for session during current tick.",
//		zap.Time("current_tick", tick),
//		zap.Int("total_events", numSessionsWithEventsToProcess),
//		zap.String("session", sessionId),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("workload_id", d.workload.GetId()))
//
//	doneChan <- sessionId
//}
//
//// Given a session ID, such as from the trace data, return the ID used internally.
////
//// The internal ID includes the unique ID of this workload driver, in case multiple
//// workloads from the same trace are being executed concurrently.
//func (d *BasicWorkloadDriverOld) getInternalSessionId(traceSessionId string) string {
//	//return fmt.Sprintf("%s-%s", traceSessionId, d.id)
//	return traceSessionId
//}
//
//// Given a session ID, such as from the trace data, return the ID used internally.
////
//// The internal ID includes the unique ID of this workload driver, in case multiple
//// workloads from the same trace are being executed concurrently.
////func (d *BasicWorkloadDriverOld) getTraceSessionId(internalSessionId string) string {
////	rindex := strings.LastIndex(internalSessionId, "-")
////
////	if rindex < 0 {
////		panic(fmt.Sprintf("could not extract trace session id from given internal session id: \"%s\" (rindex=%d)",
////			internalSessionId, rindex))
////	}
////
////	return internalSessionId[0:rindex]
////}
//
////func (d *BasicWorkloadDriverOld) getOriginalSessionIdFromInternalSessionId(internalSessionId string) string {
////	rightIndex := strings.LastIndex(internalSessionId, "-")
////	return internalSessionId[0:rightIndex]
////}
//
//// newSession create and return a new Session with the given ID.
//func (d *BasicWorkloadDriverOld) newSession(id string, meta domain.SessionMetadata, createdAtTime time.Time) Session {
//	d.sugaredLogger.Debugf("Creating new Session %v. MaxSessionCPUs: %.2f; MaxSessionMemory: %.2f. MaxSessionGPUs: %d. MaxSessionVRAM: %.2f, TotalNumSessions: %d",
//		id, meta.GetMaxSessionCPUs(), meta.GetMaxSessionMemory(), meta.GetMaxSessionGPUs(), meta.GetMaxSessionVRAM(), d.sessions.Len())
//
//	// Make sure the Session doesn't already exist.
//	var session Session
//	if session = d.GetSession(id); session != nil {
//		panic(fmt.Sprintf("Attempted to create existing Session %s.", id))
//	}
//
//	// The Session only exposes the CPUs, Memory, and
//	resourceRequest := domain.NewResourceRequest(meta.GetMaxSessionCPUs(), meta.GetMaxSessionMemory(), meta.GetMaxSessionGPUs(), meta.GetMaxSessionVRAM(), AnyGPU)
//	session = domain.NewWorkloadSession(id, meta, resourceRequest, createdAtTime, d.atom)
//
//	internalSessionId := d.getInternalSessionId(session.GetId())
//
//	d.workload.SessionCreated(id, meta)
//
//	d.mu.Lock()
//	defer d.mu.Unlock()
//
//	d.workload.UpdateStatistics(func(stats *Statistics) {
//		stats.TotalNumSessions += 1
//	})
//
//	d.sessions.Set(internalSessionId, session)
//
//	return session
//}
//
//// GetSession gets and returns the Session identified by the given ID, if one exists. Otherwise, return nil.
//// If the caller is attempting to retrieve a Session that once existed but has since been terminated, then
//// this will return nil.
////
//// id should be the internal id of the session.
//func (d *BasicWorkloadDriverOld) GetSession(id string) Session {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//
//	session, ok := d.sessions.Get(id)
//
//	if ok {
//		return session.(Session)
//	}
//
//	return nil
//}
//
//func (d *BasicWorkloadDriverOld) getSchedulingPolicy() string {
//	if d.schedulingPolicy != "" {
//		return d.schedulingPolicy
//	}
//
//	policy, ok := d.getSchedulingPolicyCallback()
//	if ok {
//		d.schedulingPolicy = policy
//	}
//
//	return policy
//}
//
//// handleSessionReadyEvent handles a single EventSessionReady *domain.Event.
//// This function is thread-safe and may be called within its own goroutine.
//func (d *BasicWorkloadDriverOld) handleSessionReadyEvent(sessionReadyEvent *domain.Event, eventIndex int, doneChan chan<- string) {
//	sessionMeta := sessionReadyEvent.Data.(domain.SessionMetadata)
//
//	sessionId := sessionMeta.GetPod()
//	if d.sugaredLogger.Level() == zapcore.DebugLevel {
//		d.sugaredLogger.Debugf("Handling EventSessionReady %d targeting Session %s [ts: %v].", eventIndex+1, sessionId, sessionReadyEvent.Timestamp)
//	}
//
//	provisionStart := time.Now()
//	_, err := d.provisionSession(sessionId, sessionMeta, sessionReadyEvent.Timestamp)
//
//	if err == nil || !strings.Contains(err.Error(), "insufficient hosts available") {
//		// The event index will be populated automatically by the ProcessedEvent method.
//		workloadEvent := domain.NewEmptyWorkloadEvent().
//			WithEventId(sessionReadyEvent.Id()).
//			WithEventName(domain.EventInvalidName).
//			WithSessionId(sessionReadyEvent.SessionID()).
//			WithEventTimestamp(sessionReadyEvent.Timestamp).
//			WithProcessedAtTime(time.Now()).
//			WithProcessedStatus(err == nil).
//			WithSimProcessedAtTime(d.clockTime.GetClockTime()).
//			WithError(err)
//		d.workload.ProcessedEvent(workloadEvent) // this is thread-safe
//	}
//
//	// Handle the error from the above call to provisionSession.
//	if err != nil {
//		d.logger.Warn("Failed to provision new Jupyter session.",
//			zap.String(ZapInternalSessionIDKey, sessionId),
//			zap.Duration("real-time-elapsed", time.Since(provisionStart)),
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.Error(err))
//
//		payload, _ := json.Marshal(domain.ErrorMessage{
//			Description:  reflect.TypeOf(err).Name(),
//			ErrorMessage: err.Error(),
//			Valid:        true,
//		})
//
//		// This is thread-safe because the WebSocket uses a thread-safe wrapper.
//		go func() {
//			if writeError := d.websocket.WriteMessage(websocket.BinaryMessage, payload); writeError != nil {
//				d.logger.Error("Failed to write error message via WebSocket.",
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.Error(writeError))
//			}
//		}()
//
//		// We need to inspect the error here.
//		// Depending on what the error is, we'll treat it as a critical error or not.
//		err = d.handleFailureToCreateNewSession(err, sessionReadyEvent)
//
//		if err != nil && !strings.Contains(err.Error(), "insufficient hosts available") {
//			d.handleCriticalError(err)
//			doneChan <- sessionId // Could probably just skip this, but it'll unblock the waiting goroutine, I guess?
//			return
//		}
//	} else {
//		d.logger.Debug("Successfully handled SessionStarted event.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, sessionId),
//			zap.Duration("real-time-elapsed", time.Since(provisionStart)))
//	}
//
//	doneChan <- sessionId
//}
//
//func (d *BasicWorkloadDriverOld) delaySession(sessionId string, delayAmount time.Duration) {
//	err := d.eventQueue.DelaySession(sessionId, delayAmount)
//	if err != nil {
//		panic(err)
//	}
//
//	d.workload.SessionDelayed(sessionId, delayAmount)
//
//	if metrics.PrometheusMetricsWrapperInstance != nil {
//		metrics.PrometheusMetricsWrapperInstance.SessionDelayedDueToResourceContention.
//			With(prometheus.Labels{
//				"workload_id": d.workload.GetId(),
//				"session_id":  sessionId,
//			}).Add(1)
//	}
//}
//
//// handleFailureToCreateNewSession processes an error in which we failed to create a kernel for some reason.
////
//// Depending on why we failed, we will either try again later or abort the workload.
////
//// The error returned by handleFailureToCreateNewSession is NOT a new error. We reformat the error about failing
//// to create a kernel depending on the reason.
//func (d *BasicWorkloadDriverOld) handleFailureToCreateNewSession(err error, sessionReadyEvent *domain.Event) error {
//	sessionId := sessionReadyEvent.SessionID()
//	if strings.Contains(err.Error(), "insufficient hosts available") {
//		//sessionReadyEvent.PushTimestampBack(d.targetTickDuration)
//
//		d.logger.Warn("Failed to create session due to insufficient hosts available. Will requeue event and try again later.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, sessionId),
//			zap.Time("original_timestamp", sessionReadyEvent.OriginalTimestamp),
//			zap.Time("current_timestamp", sessionReadyEvent.Timestamp),
//			zap.Time("current_tick", d.currentTick.GetClockTime()),
//			zap.Int32("num_times_enqueued", sessionReadyEvent.GetNumTimesEnqueued()),
//			zap.Duration("total_delay", sessionReadyEvent.TotalDelay()))
//
//		d.delaySession(sessionId, d.targetTickDuration*2)
//
//		// Put the event back in the queue.
//		d.eventQueue.EnqueueEvent(sessionReadyEvent)
//
//		// Return a less verbose error.
//		return fmt.Errorf("%w \"%s\": insufficient hosts available", ErrKernelCreationFailed, sessionId)
//	}
//
//	d.logger.Error("Session creation failure is due to unexpected reason. Aborting workload.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, sessionId),
//		zap.Error(err))
//
//	d.errorChan <- err
//	if d.onCriticalErrorOccurred != nil {
//		go d.onCriticalErrorOccurred(d.workload.GetId(), err)
//	}
//
//	// Return the original error.
//	return errors.Join(ErrKernelCreationFailed, err)
//}
//
//// handleUpdateGpuUtilizationEvent handles a 'update-gpu-util' event.
//func (d *BasicWorkloadDriverOld) handleUpdateGpuUtilizationEvent(evt *domain.Event) error {
//	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
//	internalSessionId := d.getInternalSessionId(traceSessionId)
//
//	d.logger.Debug("Received UpdateGpuUtil event.",
//		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
//	// TODO: Update GPU utilization.
//	d.logger.Debug("Handled UpdateGpuUtil event.",
//		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
//
//	return nil
//}
//
//// createExecuteRequestArguments creates the arguments for an "execute_request" from the given event.
////
//// The event must be of type "training-started", or this will return nil.
//func (d *BasicWorkloadDriverOld) createExecuteRequestArguments(evt *domain.Event, callback func(resp jupyter.KernelMessage)) (*jupyter.RequestExecuteArgs, error) {
//	if evt.Name != domain.EventSessionTrainingStarted {
//		d.logger.Error("Attempted to create \"execute_request\" arguments for event of invalid type.",
//			zap.String("event_type", evt.Name.String()),
//			zap.String("event_id", evt.Id()),
//			zap.String("session_id", evt.SessionID()),
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()))
//
//		return nil, fmt.Errorf("invalid event type: %s", evt.Name)
//	}
//
//	sessionMetadata := evt.Data.(domain.SessionMetadata)
//
//	if sessionMetadata == nil {
//		d.logger.Error("Event has nil data.",
//			zap.String("event_type", evt.Name.String()),
//			zap.String("event_id", evt.Id()),
//			zap.String("session_id", evt.SessionID()),
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()))
//		return nil, fmt.Errorf("event has nil data")
//	}
//
//	gpus := sessionMetadata.GetCurrentTrainingMaxGPUs()
//	if gpus == 0 && sessionMetadata.HasGpus() && sessionMetadata.GetGPUs() > 0 {
//		gpus = sessionMetadata.GetGPUs()
//	}
//
//	resourceRequest := &domain.ResourceRequest{
//		Cpus:     sessionMetadata.GetCurrentTrainingMaxCPUs(),
//		MemoryMB: sessionMetadata.GetCurrentTrainingMaxMemory(),
//		VRAM:     sessionMetadata.GetVRAM(),
//		Gpus:     gpus,
//	}
//
//	argsBuilder := jupyter.NewRequestExecuteArgsBuilder().
//		Code(TrainingCode).
//		Silent(false).
//		StoreHistory(true).
//		UserExpressions(nil).
//		AllowStdin(true).
//		StopOnError(false).
//		AwaitResponse(false).
//		OnResponseCallback(callback).
//		AddMetadata("resource_request", resourceRequest)
//
//	return argsBuilder.Build(), nil
//}
//
//// submitTrainingToKernel submits a training event to be processed/executed by the kernel.
//func (d *BasicWorkloadDriverOld) submitTrainingToKernel(evt *domain.Event,
//	internalSessionId string) (sentRequestAt time.Time, trainingStartedChannel chan interface{}, err error) {
//
//	d.logger.Debug("Received TrainingStarted event.",
//		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("internal-session-id", internalSessionId))
//
//	if _, ok := d.sessions.Get(internalSessionId); !ok {
//		d.logger.Warn("Received 'training-started' event for unknown session.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("event", evt.String()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId))
//
//		err = fmt.Errorf("%w: session \"%s\"", domain.ErrUnknownSession, internalSessionId)
//		return
//	}
//
//	d.sessionConnectionsMutex.Lock()
//	sessionConnection, loadedSessionConnection := d.sessionConnections[internalSessionId]
//	d.sessionConnectionsMutex.Unlock()
//
//	if !loadedSessionConnection {
//		d.logger.Error("No session connection found for session upon receiving 'training-started' event.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId))
//		err = ErrNoSessionConnection
//		return
//	}
//
//	kernelConnection := sessionConnection.Kernel()
//	if kernelConnection == nil {
//		d.logger.Error("No kernel connection found for session connection.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId))
//		err = ErrNoKernelConnection
//		return
//	}
//
//	d.trainingStartedChannelMutex.Lock()
//	trainingStartedChannel = make(chan interface{}, 1)
//	d.trainingStartedChannels[internalSessionId] = trainingStartedChannel
//	d.trainingStartedChannelMutex.Unlock()
//
//	d.trainingStoppedChannelsMutex.Lock()
//	trainingStoppedChannel := make(chan interface{}, 1) // We could conceivably reuse the 'started' channel
//	d.trainingStoppedChannels[internalSessionId] = trainingStoppedChannel
//	d.trainingStoppedChannelsMutex.Unlock()
//
//	// Create a wrapper so that we can pass more arguments to our 'onReceiveExecuteReply' method than
//	// just the kernel's response.
//	handleExecuteReplyWrapper := func(response jupyter.KernelMessage) {
//		d.onReceiveExecuteReply(response, internalSessionId, trainingStartedChannel, trainingStoppedChannel)
//	}
//
//	var executeRequestArgs *jupyter.RequestExecuteArgs
//	executeRequestArgs, err = d.createExecuteRequestArguments(evt, handleExecuteReplyWrapper)
//	if executeRequestArgs == nil || err != nil {
//		d.logger.Error("Failed to create 'execute_request' arguments.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.Error(err))
//		return time.Time{}, nil, err
//	}
//
//	sentRequestAt = time.Now()
//	_, err = kernelConnection.RequestExecute(executeRequestArgs)
//	if err != nil {
//		d.logger.Error("Error while submitting training event to kernel.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId))
//		return time.Time{}, nil, err
//	}
//
//	d.trainingSubmittedTimes.Set(internalSessionId, time.Now().UnixMilli())
//	d.workload.TrainingSubmitted(internalSessionId, evt)
//	d.logger.Debug("Handled TrainingStarted event.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, internalSessionId))
//
//	// Hold events for the session until the training actually begins.
//	err = d.eventQueue.HoldEventsForSession(internalSessionId)
//	if err != nil {
//		d.logger.Error("Could not place hold on session events.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", internalSessionId),
//			zap.Error(err))
//
//		return time.Time{}, nil, err
//	}
//
//	return sentRequestAt, trainingStartedChannel, nil
//}
//
//// waitForTrainingToStart waits for a training to begin being processed by a kernel replica.
////
//// waitForTrainingToStart is called by handleTrainingStartedEvent after submitTrainingToKernel is called.
//func (d *BasicWorkloadDriverOld) waitForTrainingToStart(evt *domain.Event, internalSessionId string,
//	startedHandlingAt time.Time, sentRequestAt time.Time, trainingStartedChannel chan interface{}) error {
//
//	d.logger.Debug("Waiting for session to start training before continuing...",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("kernel_id", internalSessionId))
//
//	// In case the IO Pub message gets lost, we'll add a timeout.
//	// This way the whole workload won't get stuck if a message is lost.
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
//	defer cancel()
//
//	select {
//	case v := <-trainingStartedChannel:
//		{
//			switch v.(type) {
//			case error:
//				{
//					err := v.(error)
//					d.logger.Warn("Session failed to start training",
//						zap.String("workload_id", d.workload.GetId()),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("kernel_id", internalSessionId),
//						zap.Duration("time_elapsed", time.Since(sentRequestAt)),
//						zap.Error(err))
//
//					// If we fail to start training for some reason, then we'll just try again later.
//					d.delaySession(internalSessionId, time.Since(startedHandlingAt)+d.targetTickDuration*2)
//
//					// Put the event back in the queue.
//					d.eventQueue.EnqueueEvent(evt)
//				}
//			default:
//				{
//					startLatency := time.Since(sentRequestAt)
//					d.logger.Debug("Session started training",
//						zap.String("workload_id", d.workload.GetId()),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("kernel_id", internalSessionId),
//						zap.Duration("start_latency", startLatency))
//				}
//			}
//		}
//	case <-ctx.Done():
//		{
//			d.trainingStartTimedOut(internalSessionId, sentRequestAt, startedHandlingAt)
//		}
//	}
//
//	return nil
//}
//
//// trainingStartTimedOut is called by waitForTrainingToStart when we don't receive a notification that the submitted
//// training event started being processed after the timeout interval elapses.
//func (d *BasicWorkloadDriverOld) trainingStartTimedOut(internalSessionId string, sentRequestAt time.Time, startedHandlingAt time.Time) {
//	d.logger.Warn("Have not received 'training started' notification for over 1 minute. Assuming message was lost.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("kernel_id", internalSessionId),
//		zap.Duration("time_elapsed", time.Since(sentRequestAt)))
//
//	d.notifyCallback(&proto.Notification{
//		Id:    uuid.NewString(),
//		Title: "Have Spent 1+ Minute(s) Waiting for 'Training Started' Notification",
//		Message: fmt.Sprintf("Submitted \"execute_request\" to kernel \"%s\" during workload \"%s\" (ID=\"%s\") "+
//			"over 1 minute ago and have not yet received 'smr_lead_task' IOPub message. Time elapsed: %v.",
//			internalSessionId, d.workload.WorkloadName(), d.workload.GetId(), time.Since(sentRequestAt)),
//		Panicked:         false,
//		NotificationType: domain.WarningNotification.Int32(),
//	})
//
//	// TODO: Resubmit the event?
//
//	// We won't return this error (if it is non-nil), as we don't want to kill the workload.
//	errReleaseEventHold := d.eventQueue.ReleaseEventHoldForSession(internalSessionId)
//	if errReleaseEventHold != nil {
//		d.logger.Debug("Could not release hold on events for session after timing-out waiting for \"smr_lead_task\" message.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", internalSessionId),
//			zap.Duration("time_elapsed", time.Since(sentRequestAt)))
//
//		go d.notifyCallback(&proto.Notification{
//			Id:               uuid.NewString(),
//			Title:            fmt.Sprintf("Failed to Release Event Hold for Session \"%s\" After Timing-Out Waiting for \"smr_lead_task\" IOPub Message", internalSessionId),
//			Message:          errReleaseEventHold.Error(),
//			Panicked:         false,
//			NotificationType: domain.WarningNotification.Int32(),
//		})
//	}
//}
//
//// handleTrainingStartedEvent handles a 'training-started' event.
//func (d *BasicWorkloadDriverOld) handleTrainingStartedEvent(evt *domain.Event) error {
//	startedHandlingAt := time.Now()
//
//	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
//	internalSessionId := d.getInternalSessionId(traceSessionId)
//
//	sentRequestAt, trainingStartedChannel, err := d.submitTrainingToKernel(evt, internalSessionId)
//	if err != nil {
//		d.logger.Error("Failed to submit training to kernel.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", internalSessionId),
//			zap.String("event", evt.StringJson()),
//			zap.Error(err))
//		return err
//	}
//
//	return d.waitForTrainingToStart(evt, internalSessionId, startedHandlingAt, sentRequestAt, trainingStartedChannel)
//}
//
//// handleTrainingEndedEvent handles a 'training-stopped' event.
//func (d *BasicWorkloadDriverOld) handleTrainingEndedEvent(evt *domain.Event, tick time.Time) error {
//	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
//	internalSessionId := d.getInternalSessionId(traceSessionId)
//	d.logger.Debug("Received TrainingEnded event.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("event_id", evt.ID),
//		zap.Time("event_timestamp", evt.Timestamp),
//		zap.String(ZapInternalSessionIDKey, internalSessionId),
//		zap.String(ZapTraceSessionIDKey, traceSessionId))
//
//	if _, ok := d.sessions.Get(internalSessionId); !ok {
//		d.logger.Warn("Received 'training-stopped' event for unknown session.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId))
//		return fmt.Errorf("%w: session \"%s\"", domain.ErrUnknownSession, internalSessionId)
//	}
//
//	d.trainingStoppedChannelsMutex.Lock()
//	trainingStoppedChannel, foundStoppedTrainingChannel := d.trainingStoppedChannels[internalSessionId]
//	d.trainingStoppedChannelsMutex.Unlock()
//
//	if !foundStoppedTrainingChannel {
//		// TODO: We don't have a 'training stopped' channel. How do we proceed?
//		//	     We can probably just send the 'stop-training' request and call it a day, right?
//		d.logger.Warn("Failed to load a 'stop-training' channel while handling 'training-ended' event.",
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId))
//	}
//
//	d.sessionConnectionsMutex.Lock()
//	sessionConnection, ok := d.sessionConnections[internalSessionId]
//	d.sessionConnectionsMutex.Unlock()
//
//	if !ok {
//		d.logger.Error("No session connection found for session upon receiving 'training-stopped' event.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId))
//		return ErrNoSessionConnection
//	}
//
//	kernelConnection := sessionConnection.Kernel()
//	if kernelConnection == nil {
//		d.logger.Error("No kernel connection found for session connection.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId))
//		return ErrNoKernelConnection
//	}
//
//	err := d.issueStopTrainingRequest(kernelConnection, 30*time.Second)
//	if err != nil {
//		d.logger.Error("Error while attempting to stop training.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId), zap.Error(err))
//		return err
//	} else {
//		d.workload.TrainingStopped(traceSessionId, evt, d.convertTimestampToTickNumber(tick))
//		d.logger.Debug("Successfully sent 'stop-training' message'.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId))
//	}
//
//	// TODO: Retrieve the 'training-stopped' channel and listen for a response.
//	//		 Wait for response from stoppedTrainingChannel variable.
//	if trainingStoppedChannel == nil {
//		d.logger.Warn("Returning from 'training-ended' handler early because we failed to load a 'stopped-training' channel.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId))
//
//		return nil
//	}
//
//	return d.waitForTrainingToEnd(internalSessionId, evt, trainingStoppedChannel)
//}
//
//func (d *BasicWorkloadDriverOld) getTimeoutInterval(internalSessionId string, evt *domain.Event) time.Duration {
//	// Load the scheduling policy.
//	schedulingPolicy := d.getSchedulingPolicy()
//	if schedulingPolicy == "" {
//		d.logger.Warn("Could not compute meaningful timeout interval because scheduling policy is invalid.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String("event", evt.Name.String()))
//		return time.Minute
//	}
//
//	if schedulingPolicy == "static" || schedulingPolicy == "dynamic-v3" || schedulingPolicy == "dynamic-v4" {
//		// There's no network I/O on the critical path, so stopping the training should be quick.
//		return time.Second * 30
//	}
//
//	// Get the remote storage definition of the workload.
//	remoteStorageDefinition := d.workload.GetRemoteStorageDefinition()
//	if remoteStorageDefinition == nil {
//		d.logger.Warn("Could not compute meaningful timeout interval because scheduling policy is invalid.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String("event", evt.Name.String()))
//		return time.Minute * 2 // We make it a bit higher since we know I/O is on the critical path.
//	}
//
//	// Load the session and subsequently its current resource request.
//	// We already checked that this existed in handleTrainingEventEnded.
//	val, _ := d.sessions.Get(internalSessionId)
//	resourceRequest := val.(Session).GetCurrentResourceRequest()
//	if resourceRequest == nil {
//		d.logger.Warn("Could not compute meaningful timeout interval because scheduling policy is invalid.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String("event", evt.Name.String()))
//		return time.Minute * 2 // We make it a bit higher since we know I/O is on the critical path.
//	}
//
//	var expectedLatencySec float64
//	vramBytes := resourceRequest.VRAM * 1000000000
//	if evt.Name == domain.EventSessionTrainingStarted || evt.Name == domain.EventSessionStarted {
//		expectedLatencySec = (vramBytes / float64(remoteStorageDefinition.DownloadRate)) * (1 + float64(remoteStorageDefinition.DownloadRateVariancePercentage))
//	} else if evt.Name == domain.EventSessionTrainingEnded {
//		expectedLatencySec = (vramBytes / float64(remoteStorageDefinition.UploadRate)) * (1 + float64(remoteStorageDefinition.UploadRateVariancePercentage))
//	} else {
//		d.logger.Warn("Unexpected event name while computing timeout interval.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String("event", evt.Name.String()))
//	}
//
//	interval := (time.Second * 30) + (time.Second * time.Duration(expectedLatencySec))
//
//	d.logger.Debug("Computed timeout interval.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, internalSessionId),
//		zap.Float64("vram_gb", resourceRequest.VRAM),
//		zap.Float64("vram_bytes", vramBytes),
//		zap.String("remote_storage_definition", remoteStorageDefinition.String()),
//		zap.String("event", evt.Name.String()))
//
//	return interval
//}
//
//func (d *BasicWorkloadDriverOld) waitForTrainingToEnd(internalSessionId string, evt *domain.Event, trainingStoppedChannel chan interface{}) error {
//	timeoutInterval := d.getTimeoutInterval(internalSessionId, evt)
//	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
//	defer cancel()
//
//	select {
//	case v := <-trainingStoppedChannel:
//		{
//			receivedResp := time.Now().UnixMilli()
//
//			var sentRequestAt int64
//			val, ok := d.trainingSubmittedTimes.Get(internalSessionId)
//			if ok {
//				sentRequestAt = val.(int64)
//			}
//			e2eLatency := time.Since(time.UnixMilli(sentRequestAt))
//
//			switch v.(type) {
//			case error:
//				{
//					err := v.(error)
//					d.logger.Warn("Session failed to stop training...",
//						zap.String("workload_id", d.workload.GetId()),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("kernel_id", internalSessionId),
//						zap.Duration("e2e_latency", e2eLatency),
//						zap.Error(err))
//
//					return nil // to prevent workload from ending outright
//				}
//			case jupyter.KernelMessage:
//				{
//					reply := v.(jupyter.KernelMessage)
//					content := reply.GetContent().(map[string]interface{})
//
//					val = content["execution_start_unix_millis"]
//					execStartedTimeUnixMillis := int64(val.(float64))
//
//					val = content["execution_finished_unix_millis"]
//					execEndedTimeUnixMillis := int64(val.(float64))
//
//					execTimeMillis := execEndedTimeUnixMillis - execStartedTimeUnixMillis
//
//					d.workload.RecordSessionExecutionTime(internalSessionId, execTimeMillis)
//
//					delay := receivedResp - execEndedTimeUnixMillis
//
//					d.workload.UpdateStatistics(func(stats *Statistics) {
//						stats.TotalReplyLatenciesMillis = append(stats.TotalReplyLatenciesMillis, delay)
//						stats.TotalReplyLatencyMillis += delay
//					})
//
//					d.logger.Debug("Session stopped training",
//						zap.String("workload_id", d.workload.GetId()),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("kernel_id", internalSessionId),
//						zap.Int64("exec_time_millis", execTimeMillis),
//						zap.Duration("e2e_latency", e2eLatency))
//
//					return nil
//				}
//			default:
//				{
//					d.logger.Error("Received unexpected response via 'training-stopped' channel.",
//						zap.String("workload_id", d.workload.GetId()),
//						zap.String("workload_name", d.workload.WorkloadName()),
//						zap.String("kernel_id", internalSessionId),
//						zap.Duration("e2e_latency", e2eLatency),
//						zap.Any("response", v))
//
//					return fmt.Errorf("unexpected response via 'training-stopped' channel")
//				}
//			}
//		}
//	case <-ctx.Done():
//		{
//			err := ctx.Err()
//			if err != nil {
//				d.logger.Error("Timed-out waiting for \"execute_reply\" message while stopping training.",
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("kernel_id", internalSessionId),
//					zap.Error(err))
//
//				// We'll just return (nothing) so that the workload doesn't end.
//				return nil
//			}
//
//			// No error attached to the context. Just log an error message without the error struct
//			// and return an error of our own.
//			d.logger.Error("Timed-out waiting for \"execute_reply\" message while stopping training.",
//				zap.String("workload_id", d.workload.GetId()),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("kernel_id", internalSessionId))
//		}
//	}
//
//	// We'll just return (nothing) so that the workload doesn't end.
//	return nil
//}
//
//// issueStopTrainingRequest sends a 'StopRunningTrainingCode' request to a kernel with a configurable timeout.
//func (d *BasicWorkloadDriverOld) issueStopTrainingRequest(kernelConnection jupyter.KernelConnection, timeout time.Duration) error {
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	doneChan := make(chan interface{}, 1)
//
//	// Issue request using a separate goroutine.
//	go func() {
//		err := kernelConnection.StopRunningTrainingCode(true)
//		if err != nil {
//			doneChan <- err
//		} else {
//			doneChan <- struct{}{}
//		}
//	}()
//
//	select {
//	case <-ctx.Done():
//		{
//			// If there's an error, we'll log and return it.
//			err := ctx.Err()
//			if err != nil {
//				d.logger.Error("Timed-out waiting for response to 'stop-training' request.",
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("kernel_id", kernelConnection.KernelId()),
//					zap.String("connection_status", kernelConnection.ConnectionStatus().String()),
//					zap.Error(err))
//
//				return errors.Join(jupyter.ErrRequestTimedOut, err)
//			}
//
//			// No error attached to the context. Just log an error message without the error struct
//			// and return an error of our own.
//			d.logger.Error("Timed-out waiting for response to 'stop-training' request.",
//				zap.String("workload_id", d.workload.GetId()),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("kernel_id", kernelConnection.KernelId()),
//				zap.String("connection_status", kernelConnection.ConnectionStatus().String()))
//
//			return jupyter.ErrRequestTimedOut
//		}
//	case val := <-doneChan:
//		{
//			if err, ok := val.(error); ok {
//				d.logger.Error("Error encountered while sending 'stop-training' request.",
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("kernel_id", kernelConnection.KernelId()),
//					zap.String("connection_status", kernelConnection.ConnectionStatus().String()),
//					zap.Error(err))
//
//				return err
//			}
//
//			return nil
//		}
//	}
//}
//
//// handleSessionStoppedEvent handles a 'session-stopped' event.
//func (d *BasicWorkloadDriverOld) handleSessionStoppedEvent(evt *domain.Event) error {
//	traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
//	internalSessionId := d.getInternalSessionId(traceSessionId)
//
//	d.logger.Debug("Received SessionStopped event.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, internalSessionId),
//		zap.String(ZapTraceSessionIDKey, traceSessionId))
//
//	if _, ok := d.sessions.Get(internalSessionId); !ok {
//		d.logger.Warn("Received 'session-stopped' event for unknown session.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId))
//		return fmt.Errorf("%w: session \"%s\"", domain.ErrUnknownSession, internalSessionId)
//	}
//
//	d.logger.Debug("Stopping session.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("kernel_id", internalSessionId))
//
//	err := d.kernelManager.StopKernel(internalSessionId)
//	if err != nil {
//		d.logger.Error("Error encountered while stopping session.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.String(ZapTraceSessionIDKey, traceSessionId),
//			zap.Error(err))
//		return err
//	}
//
//	d.logger.Debug("Successfully stopped session.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, internalSessionId),
//		zap.String(ZapTraceSessionIDKey, traceSessionId))
//
//	// Attempt to update the Prometheus metrics for Session lifetime duration (in seconds).
//	session := d.GetSession(internalSessionId)
//	if session == nil {
//		d.logger.Error("Could not find Session with specified ID.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("session_id", internalSessionId))
//	} else {
//		sessionLifetimeDuration := time.Since(session.GetCreatedAt())
//		metrics.PrometheusMetricsWrapperInstance.WorkloadSessionLifetimeSeconds.
//			With(prometheus.Labels{"workload_id": d.workload.GetId()}).
//			Observe(sessionLifetimeDuration.Seconds())
//	}
//
//	d.workload.SessionStopped(traceSessionId, evt)
//	d.logger.Debug("Handled SessionStopped event.",
//		zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String(ZapInternalSessionIDKey, internalSessionId), zap.String(ZapTraceSessionIDKey, traceSessionId))
//
//	return nil
//}
//
//// handleEvent processes a single *domain.Event.
//func (d *BasicWorkloadDriverOld) handleEvent(evt *domain.Event, tick time.Time) error {
//	switch evt.Name {
//	case domain.EventSessionTrainingStarted:
//		return d.handleTrainingStartedEvent(evt)
//	case domain.EventSessionTrainingEnded:
//		return d.handleTrainingEndedEvent(evt, tick)
//	case domain.EventSessionStopped:
//		return d.handleSessionStoppedEvent(evt)
//	case domain.EventSessionReady:
//		d.processSessionReadyEvents([]*domain.Event{evt}, tick, time.Minute*3)
//	default:
//		traceSessionId := evt.Data.(domain.SessionMetadata).GetPod()
//		internalSessionId := d.getInternalSessionId(traceSessionId)
//
//		d.logger.Error("Received event of unknown or unexpected type.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("event_name", evt.Name.String()),
//			zap.Time("event_timestamp", evt.Timestamp),
//			zap.String("trace_session_id", traceSessionId),
//			zap.String("session_id", internalSessionId))
//
//		return fmt.Errorf("%w: \"%s\"", ErrUnknownEventType, evt.Name.String())
//	}
//
//	return nil
//}
//
//func (d *BasicWorkloadDriverOld) stopSession(sessionId string) error {
//	d.logger.Debug("Stopping session.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.String("kernel_id", sessionId))
//	return d.kernelManager.StopKernel(sessionId)
//}
//
//func (d *BasicWorkloadDriverOld) provisionSession(sessionId string, meta domain.SessionMetadata, createdAtTime time.Time) (*jupyter.SessionConnection, error) {
//	internalSessionId := d.getInternalSessionId(sessionId)
//
//	d.logger.Debug("Creating new kernel.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("ZapInternalSessionIDKey", internalSessionId))
//	st := time.Now()
//
//	var resourceSpec *jupyter.ResourceSpec
//	schedulingPolicy := d.getSchedulingPolicy()
//	if schedulingPolicy == "static" || schedulingPolicy == "dynamic-v3" || schedulingPolicy == "dynamic-v4" {
//		// Try to get the first training event of the session, and just reserve those resources.
//		firstTrainingEvent := d.workload.getSessionTrainingEvent(sessionId, 0)
//
//		if firstTrainingEvent != nil {
//			resourceSpec = &jupyter.ResourceSpec{
//				Cpu:  int(firstTrainingEvent.Millicpus),
//				Mem:  firstTrainingEvent.MemUsageMB,
//				Gpu:  firstTrainingEvent.NumGPUs(),
//				Vram: firstTrainingEvent.VRamUsageGB,
//			}
//		} else {
//			d.logger.Warn("Could not find first training event of session.",
//				zap.String("workload_id", d.workload.GetId()),
//				zap.String("workload_name", d.workload.WorkloadName()),
//				zap.String("session_id", internalSessionId))
//		}
//	}
//
//	// If we're either not using static/dynamic scheduling or we couldn't find the first training event for some
//	// reason, then we'll create the resource request using the maximum values of the session's resource usage.
//	if resourceSpec == nil {
//		resourceSpec = &jupyter.ResourceSpec{
//			Cpu:  int(meta.GetMaxSessionCPUs()),
//			Mem:  meta.GetMaxSessionMemory(),
//			Gpu:  meta.GetMaxSessionGPUs(),
//			Vram: meta.GetMaxSessionVRAM(),
//		}
//	}
//
//	// Create the kernel in Jupyter.
//	sessionConnection, err := d.kernelManager.CreateSession(
//		internalSessionId, /*strings.ToLower(sessionId) */
//		fmt.Sprintf("%s.ipynb", internalSessionId),
//		"notebook", "distributed", resourceSpec)
//
//	if err != nil {
//		d.logger.Warn("Failed to create session.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String(ZapInternalSessionIDKey, internalSessionId),
//			zap.Error(err))
//
//		// We call our OnError handlers after returning; no need to call them here.
//		return nil, err
//	}
//
//	timeElapsed := time.Since(st)
//
//	d.sessionConnectionsMutex.Lock()
//	d.sessionConnections[internalSessionId] = sessionConnection
//	d.sessionConnectionsMutex.Unlock()
//
//	d.logger.Debug("Successfully created new kernel.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.Duration("time-elapsed", timeElapsed),
//		zap.String(ZapInternalSessionIDKey, internalSessionId))
//
//	// Create a new workload session.
//	workloadSession := d.newSession(sessionId, meta, createdAtTime)
//
//	// ioPubHandler is a session-specific wrapper around the standard BasicWorkloadDriverOld::handleIOPubMessage method.
//	// This returns true if the received IOPub message is a "stream" message and is parsed successfully.
//	// Otherwise, this returns false.
//	//
//	// The return value is not really used.
//	ioPubHandler := func(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
//		// Parse the IOPub message.
//		// If it is a stream message, this will return a *parsedIoPubMessage variable.
//		parsedIoPubMsgVal := d.handleIOPubMessage(conn, kernelMessage)
//
//		if parsedIoPubMsg, ok := parsedIoPubMsgVal.(*parsedIoPubMessage); ok {
//			switch parsedIoPubMsg.Stream {
//			case "stdout":
//				{
//					workloadSession.AddStdoutIoPubMessage(parsedIoPubMsg.Text)
//				}
//			case "stderr":
//				{
//					workloadSession.AddStderrIoPubMessage(parsedIoPubMsg.Text)
//				}
//			default:
//				d.logger.Warn("Unexpected stream specified by IOPub message.",
//					zap.String("workload_id", d.workload.GetId()),
//					zap.String("workload_name", d.workload.WorkloadName()),
//					zap.String("stream", parsedIoPubMsg.Stream))
//				return false
//			}
//			return true
//		}
//
//		return false
//	}
//
//	if err := sessionConnection.RegisterIoPubHandler(d.id, ioPubHandler); err != nil {
//		d.logger.Warn("Failed to register IOPub message handler.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("id", d.id), zap.Error(err))
//	}
//
//	return sessionConnection, nil
//}
//
//// handleIOPubMessage returns the extracted text.
//// This is expected to be called within a session-specific wrapper.
////
//// If the IOPub message is a "stream" message, then this returns a *parsedIoPubMessage
//// wrapping the name of the stream and the message text.
//func (d *BasicWorkloadDriverOld) handleIOPubMessage(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
//	// We just want to extract the output from 'stream' IOPub messages.
//	// We don't care about non-stream-type IOPub messages here, so we'll just return.
//	messageType := kernelMessage.GetHeader().MessageType
//	if messageType != "stream" && messageType != "smr_lead_task" {
//		return nil
//	}
//
//	if messageType == "stream" {
//		return d.handleIOPubStreamMessage(conn, kernelMessage)
//	}
//
//	return d.handleIOPubSmrLeadTaskMessage(conn, kernelMessage)
//}
//
//func (d *BasicWorkloadDriverOld) onReceiveExecuteReply(response jupyter.KernelMessage, sessionId string, trainingStartedChannel chan interface{}, trainingStoppedChannel chan interface{}) {
//	responseContent := response.GetContent().(map[string]interface{})
//	if responseContent == nil {
//		d.logger.Error("\"execute_reply\" message does not have any content...",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("session_id", sessionId),
//			zap.String("response", response.String()))
//		return
//	}
//
//	val, ok := responseContent["status"]
//	if !ok {
//		d.logger.Error("\"execute_reply\" message does not contain a \"status\" field in its content.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("session_id", sessionId),
//			zap.String("response", response.String()))
//		return
//	}
//
//	status := val.(string)
//
//	if status == "error" {
//		errorName := responseContent["ename"].(string)
//		errorValue := responseContent["evalue"].(string)
//
//		d.logger.Warn("Received \"execute_reply\" message with error status.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("session_id", sessionId),
//			zap.String("ename", errorName),
//			zap.String("evalue", errorValue),
//			zap.String("response", response.String()))
//
//		// Notify the training started channel. There will not be a smr_lead_task sent at this point, since
//		// there was an error, so we'll send the notification to the training_started channel.
//		trainingStartedChannel <- fmt.Errorf("%s: %s", errorName, errorValue)
//	} else {
//		d.logger.Debug("Received \"execute_reply\" message with non-error status.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("session_id", sessionId),
//			zap.String("response", response.String()))
//
//		trainingStoppedChannel <- response
//	}
//}
//
//func (d *BasicWorkloadDriverOld) handleIOPubSmrLeadTaskMessage(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
//	d.logger.Debug("Received 'smr_lead_task' message from kernel.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("kernel_id", conn.KernelId()))
//
//	d.workload.TrainingStarted(conn.KernelId(), d.convertTimestampToTickNumber(d.currentTick.GetClockTime()))
//	err := d.eventQueue.ReleaseEventHoldForSession(conn.KernelId())
//	if err != nil {
//		d.logger.Error("Could not release hold on session events.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", conn.KernelId()),
//			zap.Error(err))
//
//		go d.notifyCallback(&proto.Notification{
//			Id:               uuid.NewString(),
//			Title:            fmt.Sprintf("Failed to Release Event Hold for Session \"%s\"", conn.KernelId()),
//			Message:          err.Error(),
//			NotificationType: int32(domain.WarningNotification),
//			Panicked:         false,
//		})
//	}
//
//	// Use the timestamp encoded in the IOPub message to determine when the training actually began,
//	// and then delay the session by how long it took for training to begin.
//	content := kernelMessage.GetContent().(map[string]interface{})
//
//	var trainingStartedAt int64
//	val, ok := content["msg_created_at_unix_milliseconds"]
//	if !ok {
//		d.logger.Error("Could not recover unix millisecond timestamp from \"smr_lead_task\" IOPub message.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", conn.KernelId()),
//			zap.Any("message_content", content))
//
//		panic("Could not recover unix millisecond timestamp from \"smr_lead_task\" IOPub message.")
//	}
//
//	trainingStartedAt = int64(val.(float64))
//
//	val, ok = d.trainingSubmittedTimes.Get(conn.KernelId())
//	if !ok {
//		d.logger.Error("Could not recover training-submitted-at-time either.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", conn.KernelId()),
//			zap.Any("message_content", content))
//
//		panic("cannot recover training submission timestamp from IOPub message or internal back-up map...")
//	}
//
//	sentExecRequestAt := val.(int64)
//
//	delayMilliseconds := trainingStartedAt - sentExecRequestAt
//	if delayMilliseconds < 0 {
//		d.logger.Error("Computed invalid delay between training submission and training start...",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", conn.KernelId()),
//			zap.Int64("sent_execute_request_at", sentExecRequestAt),
//			zap.Int64("training_started_at", trainingStartedAt),
//			zap.Int64("computed_delay_millis", delayMilliseconds))
//
//		delayMilliseconds = 0
//	}
//
//	d.workload.UpdateStatistics(func(stats *Statistics) {
//		stats.JupyterTrainingStartLatenciesDashboardMillis = append(
//			stats.JupyterTrainingStartLatenciesDashboardMillis, float64(delayMilliseconds))
//
//		stats.JupyterTrainingStartLatencyDashboardMillis += float64(delayMilliseconds)
//	})
//
//	d.logger.Debug("Computed training-started delay for session.",
//		zap.String("workload_id", d.workload.GetId()),
//		zap.String("workload_name", d.workload.WorkloadName()),
//		zap.String("kernel_id", conn.KernelId()),
//		zap.Int64("sent_execute_request_at", sentExecRequestAt),
//		zap.Int64("training_started_at", trainingStartedAt),
//		zap.Int64("computed_delay", delayMilliseconds))
//
//	d.delaySession(conn.KernelId(), time.Millisecond*time.Duration(delayMilliseconds))
//
//	d.trainingStartedChannelMutex.Lock()
//	channel, loadedChan := d.trainingStartedChannels[conn.KernelId()]
//	d.trainingStartedChannelMutex.Unlock()
//
//	if !loadedChan {
//		d.logger.Error("Could not find 'training started' channel for session.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.String("kernel_id", conn.KernelId()),
//			zap.Int64("sent_execute_request_at", sentExecRequestAt),
//			zap.Int64("training_started_at", trainingStartedAt),
//			zap.Int64("computed_delay", delayMilliseconds))
//
//		return conn.KernelId()
//	}
//
//	channel <- struct{}{}
//
//	d.trainingStartedChannelMutex.Lock()
//	delete(d.trainingStartedChannels, conn.KernelId()) // Clean up
//	d.trainingStartedChannelMutex.Unlock()
//
//	return conn.KernelId()
//}
//
//func (d *BasicWorkloadDriverOld) handleIOPubStreamMessage(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
//	content := kernelMessage.GetContent().(map[string]interface{})
//
//	var (
//		stream string
//		text   string
//		ok     bool
//	)
//
//	stream, ok = content["name"].(string)
//	if !ok {
//		d.logger.Warn("Content of IOPub message did not contain an entry with key \"name\" and value of type string.",
//			zap.String("workload_id", d.workload.GetId()),
//			zap.String("workload_name", d.workload.WorkloadName()),
//			zap.Any("content", content), zap.Any("message", kernelMessage),
//			zap.String("kernel_id", conn.KernelId()))
//		return nil
//	}
//
//	text, ok = content["text"].(string)
//	if !ok {
//		d.logger.Warn("Content of IOPub message did not contain an entry with key \"text\" and value of type string.", zap.String("workload_id", d.workload.GetId()), zap.String("workload_name", d.workload.WorkloadName()), zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("kernel_id", conn.KernelId()))
//		return nil
//	}
//
//	return &parsedIoPubMessage{
//		Stream: stream,
//		Text:   text,
//	}
//}
//
//// ObserveJupyterSessionCreationLatency records the latency of creating a Jupyter session
//// during the execution of a particular workload, as identified by the given workload ID.
//func (d *BasicWorkloadDriverOld) ObserveJupyterSessionCreationLatency(latencyMilliseconds int64, workloadId string) {
//	stats := d.workload.GetStatistics()
//
//	stats.CumulativeJupyterSessionCreationLatencyMillis += latencyMilliseconds
//	stats.JupyterSessionCreationLatenciesMillis = append(
//		stats.JupyterSessionCreationLatenciesMillis, latencyMilliseconds)
//
//	if metrics.PrometheusMetricsWrapperInstance != nil {
//		metrics.PrometheusMetricsWrapperInstance.ObserveJupyterSessionCreationLatency(latencyMilliseconds, workloadId)
//	}
//}
//
//// ObserveJupyterSessionTerminationLatency records the latency of terminating a Jupyter session
//// during the execution of a particular workload, as identified by the given workload ID.
//func (d *BasicWorkloadDriverOld) ObserveJupyterSessionTerminationLatency(latencyMilliseconds int64, workloadId string) {
//	stats := d.workload.GetStatistics()
//
//	stats.CumulativeJupyterSessionTerminationLatencyMillis += latencyMilliseconds
//	stats.JupyterSessionTerminationLatenciesMillis = append(
//		stats.JupyterSessionTerminationLatenciesMillis, latencyMilliseconds)
//
//	if metrics.PrometheusMetricsWrapperInstance != nil {
//		metrics.PrometheusMetricsWrapperInstance.ObserveJupyterSessionTerminationLatency(latencyMilliseconds, workloadId)
//	}
//}
//
//// ObserveJupyterExecuteRequestE2ELatency records the end-to-end latency of an "execute_request" message
//// during the execution of a particular workload, as identified by the given workload ID.
//func (d *BasicWorkloadDriverOld) ObserveJupyterExecuteRequestE2ELatency(latencyMilliseconds int64, workloadId string) {
//	stats := d.workload.GetStatistics()
//
//	stats.CumulativeJupyterExecRequestTimeMillis += latencyMilliseconds
//	stats.JupyterExecRequestTimesMillis = append(
//		stats.JupyterExecRequestTimesMillis, latencyMilliseconds)
//
//	if metrics.PrometheusMetricsWrapperInstance != nil {
//		metrics.PrometheusMetricsWrapperInstance.ObserveJupyterExecuteRequestE2ELatency(latencyMilliseconds, workloadId)
//	}
//}
//
//// AddJupyterRequestExecuteTime records the time taken to process an "execute_request" for the total, aggregate,
//// cumulative time spent processing "execute_request" messages.
//func (d *BasicWorkloadDriverOld) AddJupyterRequestExecuteTime(latencyMilliseconds int64, kernelId string, workloadId string) {
//	if metrics.PrometheusMetricsWrapperInstance != nil {
//		metrics.PrometheusMetricsWrapperInstance.AddJupyterRequestExecuteTime(latencyMilliseconds, kernelId, workloadId)
//	}
//}
