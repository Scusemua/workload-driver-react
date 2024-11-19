package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	WorkloadReady      WorkloadState = "WorkloadReady"      // BasicWorkload is registered and ready to be started.
	WorkloadRunning    WorkloadState = "WorkloadRunning"    // BasicWorkload is actively running/in-progress.
	WorkloadPausing    WorkloadState = "WorkloadPausing"    // BasicWorkload is actively running/in-progress.
	WorkloadPaused     WorkloadState = "WorkloadPaused"     // BasicWorkload is actively running/in-progress.
	WorkloadFinished   WorkloadState = "WorkloadFinished"   // BasicWorkload stopped naturally/successfully after processing all events.
	WorkloadErred      WorkloadState = "WorkloadErred"      // BasicWorkload stopped due to an error.
	WorkloadTerminated WorkloadState = "WorkloadTerminated" // BasicWorkload stopped because it was explicitly terminated early/premature.

	UnspecifiedWorkload WorkloadType = "UnspecifiedWorkloadType" // Default value, before it is set.
	PresetWorkload      WorkloadType = "WorkloadFromPreset"
	TemplateWorkload    WorkloadType = "WorkloadFromTemplate"
	TraceWorkload       WorkloadType = "WorkloadFromTrace"

	WorkloadTerminatedEventName string = "workload-terminated"
)

var (
	ErrWorkloadNotRunning = errors.New("the workload is currently not running")
	ErrInvalidState       = errors.New("workload is in invalid state for the specified operation")
	ErrWorkloadNotFound   = errors.New("could not find workload with the specified ID")
	ErrWorkloadNotPaused  = errors.New("the workload is currently not paused")
)

type WorkloadErrorHandler func(workloadId string, err error)

type WorkloadState string

func (state WorkloadState) String() string {
	return string(state)
}

// WorkloadType defines a type that a workload can have/be.
//
// Workloads can be of several different types, namely 'preset' and 'template' and possibly 'trace'.
// Have not fully committed to making 'trace' a separate type from 'preset'.
//
// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
// have properties that the user can specify and change before submitting the workload for registration.
type WorkloadType string

func (typ WorkloadType) String() string {
	return string(typ)
}

type WorkloadGenerator interface {
	GeneratePresetWorkload(EventConsumer, Workload, *WorkloadPreset, *WorkloadRegistrationRequest) error              // Start generating the workload.
	GenerateTemplateWorkload(EventConsumer, Workload, []*WorkloadTemplateSession, *WorkloadRegistrationRequest) error // Start generating the workload.
	StopGeneratingWorkload()                                                                                          // Stop generating the workload prematurely.
}

// NamedEvent is intended to cover SessionEvents and WorkloadEvents
type NamedEvent interface {
	String() string
}

// Workload encapsulates a workload submitted by a user to be orchestrated and executed by the backend server.
//
// Most methods of the Workload interface are intended to be thread-safe, even if they aren't explicitly
// indicated as such in the method's documentation.
type Workload interface {
	// IsTerminated Returns true if the workload stopped because it was explicitly terminated early/premature.
	IsTerminated() bool
	// IsReady Returns true if the workload is registered and ready to be started.
	IsReady() bool
	// IsErred Returns true if the workload stopped due to an error.
	IsErred() bool
	// IsRunning returns true if the workload is actively running (i.e., not paused).
	IsRunning() bool
	// IsPausing returns true if the workload is pausing, meaning that it is finishing the processing
	// of its current tick before halting until it is un-paused.
	IsPausing() bool
	// IsPaused returns true if the workload is paused.
	IsPaused() bool
	// IsInProgress returns true if the workload is actively running, pausing, or paused.
	IsInProgress() bool
	// IsFinished Returns true if the workload finished in any capacity (i.e., either successfully or due to an error).
	IsFinished() bool
	// DidCompleteSuccessfully Returns true if the workload stopped naturally/successfully after processing all events.
	DidCompleteSuccessfully() bool
	// String returns a String representation of the Workload suitable for logging.
	String() string
	// GetId Returns the unique ID of the workload.
	GetId() string
	// WorkloadName Returns the name of the workload.
	// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
	WorkloadName() string
	// GetWorkloadState Returns the current state of the workload.
	GetWorkloadState() WorkloadState
	// GetTimeElapsed returns the time elapsed, which is computed at the time that data is requested by the user.
	GetTimeElapsed() time.Duration
	// GetTimeElapsedAsString returns the time elapsed as a string, which is computed at the time that data is requested by the user.
	GetTimeElapsedAsString() string
	// SetTimeElapsed updates the time elapsed.
	SetTimeElapsed(time.Duration)
	// UpdateTimeElapsed Instructs the Workload to recompute its 'time elapsed' field.
	UpdateTimeElapsed()
	// GetNumEventsProcessed Returns the number of events processed by the workload.
	GetNumEventsProcessed() int64
	// GetStartTime Returns the time that the workload was started.
	GetStartTime() time.Time
	// GetEndTime Gets the time at which the workload finished.
	// If the workload hasn't finished yet, the returned boolean will be false.
	// If the workload has finished, then the returned boolean will be true.
	GetEndTime() (time.Time, bool)
	// GetRegisteredTime Returns the time that the workload was registered.
	GetRegisteredTime() time.Time
	// GetSeed Returns the workload's seed.
	GetSeed() int64
	// SetSeed Sets the workload's seed. Can only be performed once. If attempted again, this will panic.
	SetSeed(seed int64)
	// GetErrorMessage Gets the error message associated with the workload.
	// If the workload is not in an ERROR state, then this returns the empty string and false.
	// If the workload is in an ERROR state, then the boolean returned will be true.
	GetErrorMessage() (string, bool)
	// SetErrorMessage Sets the error message for the workload.
	SetErrorMessage(string)
	// IsDebugLoggingEnabled Returns a flag indicating whether debug logging is enabled.
	IsDebugLoggingEnabled() bool
	// SetDebugLoggingEnabled Enables or disables debug logging for the workload.
	SetDebugLoggingEnabled(enabled bool)
	// SetWorkloadState Sets the state of the workload.
	SetWorkloadState(state WorkloadState)
	// StartWorkload Starts the Workload.
	//
	// If the workload is already running, then an error is returned.
	// Likewise, if the workload was previously running but has already stopped, then an error is returned.
	StartWorkload() error
	// TerminateWorkloadPrematurely Explicitly/manually stops the workload early.
	// Return the time at which the workload terminated and an error if one occurred.
	TerminateWorkloadPrematurely(time.Time) (time.Time, error)
	// SetWorkloadCompleted Marks the workload as having completed successfully.
	SetWorkloadCompleted()
	// ProcessedEvent is Called after an event is processed for the Workload.
	// Updates some internal metrics.
	// Automatically sets the WorkloadEvent's index field.
	//
	// This method is thread safe.
	ProcessedEvent(*WorkloadEvent)
	// SessionCreated is Called when a Session is created for/in the Workload.
	// Just updates some internal metrics.
	SessionCreated(string, SessionMetadata)
	// SessionStopped is Called when a Session is stopped for/in the Workload.
	// Just updates some internal metrics.
	SessionStopped(string, Event)
	// TrainingStarted is Called when a training starts during/in the workload.
	// Just updates some internal metrics.
	TrainingStarted(string, Event)
	// TrainingStopped is Called when a training stops during/in the workload.
	// Just updates some internal metrics.
	TrainingStopped(string, Event)
	// GetWorkloadType returns the type of workload (TRACE, PRESET, or TEMPLATE).
	GetWorkloadType() WorkloadType
	// IsPresetWorkload Returns true if this workload was created using a preset.
	IsPresetWorkload() bool
	// IsTemplateWorkload Returns true if this workload was created using a template.
	IsTemplateWorkload() bool
	// IsTraceWorkload Returns true if this workload was created using the trace data.
	IsTraceWorkload() bool
	// GetWorkloadSource returns the "source" of the workload, be it a preset, a template, or some trace data.
	// If this is a preset workload, return the name of the preset.
	// If this is a trace workload, return the trace information.
	// If this is a template workload, return the template information.
	GetWorkloadSource() interface{}
	// GetTimescaleAdjustmentFactor returns the workload's Timescale Adjustment Factor, which effects the
	// timescale at which tickets are replayed/"simulated".
	GetTimescaleAdjustmentFactor() float64
	// GetProcessedEvents Returns the events processed during this workload (so far).
	GetProcessedEvents() []*WorkloadEvent
	// GetSessions Returns the sessions involved in this workload.
	GetSessions() []WorkloadSession
	// SetSessions Sets the sessions that will be involved in this workload.
	//
	// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
	SetSessions([]WorkloadSession)
	// SetSource Sets the source of the workload, namely a template or a preset.
	SetSource(interface{})
	// GetCurrentTick Returns the current tick.
	GetCurrentTick() int64
	// GetSimulationClockTimeStr Returns the simulation clock time.
	GetSimulationClockTimeStr() string
	// TickCompleted is Called by the driver after each tick.
	// Updates the time elapsed, current tick, and the simulation clock time.
	TickCompleted(int64, time.Time)
	// RegisterOnNonCriticalErrorHandler registers a non-critical error handler for the target workload.
	//
	// If there is already a non-critical handler error registered for the target workload, then the existing
	// non-critical error handler is overwritten.
	RegisterOnNonCriticalErrorHandler(handler WorkloadErrorHandler)
	// RegisterOnCriticalErrorHandler registers a critical error handler for the target workload.
	//
	// If there is already a critical handler error registered for the target workload, then the existing
	// critical error handler is overwritten.
	RegisterOnCriticalErrorHandler(handler WorkloadErrorHandler)
	// GetTickDurationsMillis returns a slice containing the clock time that elapsed for each tick
	// of the workload in order, in milliseconds.
	GetTickDurationsMillis() []int64
	// AddFullTickDuration is called to record how long a tick lasted, including the "artificial" sleep that is performed
	// by the WorkloadDriver in order to fully simulate ticks that otherwise have no work/events to be processed.
	AddFullTickDuration(timeElapsed time.Duration)
	// PauseWaitBeginning should be called by the WorkloadDriver if it finds that the workload is paused, and it
	// actually begins waiting. This will prevent any of the time during which the workload was paused from being
	// counted towards the workload's runtime.
	PauseWaitBeginning()
	// SetPausing will set the workload to the pausing state, which means that it is finishing
	// the processing of its current tick before halting until being unpaused.
	SetPausing() error
	// SetPaused will set the workload to the paused state.
	SetPaused() error
	// Unpause will set the workload to the active/running state.
	Unpause() error
	// GetRemoteStorageDefinition returns the *proto.RemoteStorageDefinition used by the Workload.
	GetRemoteStorageDefinition() *proto.RemoteStorageDefinition
}

// GetWorkloadStateAsString will panic if an invalid workload state is specified.
func GetWorkloadStateAsString(state WorkloadState) string {
	switch state {
	case WorkloadReady:
		{
			return "WorkloadReady"
		}
	case WorkloadRunning:
		{
			return "WorkloadRunning"
		}
	case WorkloadFinished:
		{
			return "WorkloadFinished"
		}
	case WorkloadErred:
		{
			return "WorkloadErred"
		}
	case WorkloadTerminated:
		{
			return "WorkloadTerminated"
		}
	default:
		panic(fmt.Sprintf("Unknown workload state: %v", state))
	}
}

type BasicWorkload struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	Id                        string            `json:"id"`
	Name                      string            `json:"name"`
	WorkloadState             WorkloadState     `json:"workload_state"`
	CurrentTick               int64             `json:"current_tick"`
	DebugLoggingEnabled       bool              `json:"debug_logging_enabled"`
	ErrorMessage              string            `json:"error_message"`
	EventsProcessed           []*WorkloadEvent  `json:"events_processed"`
	Sessions                  []WorkloadSession `json:"sessions"`
	Seed                      int64             `json:"seed"`
	RegisteredTime            time.Time         `json:"registered_time"`
	StartTime                 time.Time         `json:"start_time"`
	EndTime                   time.Time         `json:"end_time"`
	WorkloadDuration          time.Duration     `json:"workload_duration"` // The total time that the workload executed for. This is only set once the workload has completed.
	TimeElapsed               time.Duration     `json:"time_elapsed"`      // Computed at the time that the data is requested by the user. This is the time elapsed SO far.
	TimeElapsedStr            string            `json:"time_elapsed_str"`
	NumTasksExecuted          int64             `json:"num_tasks_executed"`
	NumEventsProcessed        int64             `json:"num_events_processed"`
	NumSessionsCreated        int64             `json:"num_sessions_created"`
	NumActiveSessions         int64             `json:"num_active_sessions"`
	NumActiveTrainings        int64             `json:"num_active_trainings"`
	TimescaleAdjustmentFactor float64           `json:"timescale_adjustment_factor"`
	SimulationClockTimeStr    string            `json:"simulation_clock_time"`
	WorkloadType              WorkloadType      `json:"workload_type"`
	TickDurationsMillis       []int64           `json:"tick_durations_milliseconds"`
	TimeSpentPausedMillis     int64             `json:"time_spent_paused_milliseconds"`
	timeSpentPaused           time.Duration
	pauseWaitBegin            time.Time

	// SumTickDurationsMillis is the sum of all tick durations in milliseconds, to make it easier
	// to compute the average tick duration.
	SumTickDurationsMillis int64 `json:"sum_tick_durations_millis"`

	// This is basically the child struct.
	// So, if this is a preset workload, then this is the WorkloadFromPreset struct.
	// We use this so we can delegate certain method calls to the child/derived struct.
	workloadInstance     Workload
	workloadSource       interface{}
	mu                   sync.RWMutex
	sessionsMap          *hashmap.HashMap // Internal mapping of session ID to session.
	trainingStartedTimes *hashmap.HashMap // Internal mapping of session ID to the time at which it began training.
	seedSet              bool             // Flag keeping track of whether we've already set the seed for this workload.
	sessionsSet          bool             // Flag keeping track of whether we've already set the sessions for this workload.

	// OnError is a callback passed to WorkloadDrivers (via the WorkloadManager).
	// If a critical error occurs during the execution of the workload, then this handler is called.
	onCriticalError WorkloadErrorHandler

	// OnError is a callback passed to WorkloadDrivers (via the WorkloadManager).
	// If a non-critical error occurs during the execution of the workload, then this handler is called.
	onNonCriticalError WorkloadErrorHandler

	RemoteStorageDefinition *proto.RemoteStorageDefinition
}

// WorkloadBuilder is the builder for the Workload struct.
type WorkloadBuilder struct {
	id                        string
	workloadName              string
	seed                      int64
	debugLoggingEnabled       bool
	timescaleAdjustmentFactor float64
	remoteStorageDefinition   *proto.RemoteStorageDefinition
	atom                      *zap.AtomicLevel
}

// NewWorkloadBuilder creates a new WorkloadBuilder instance.
func NewWorkloadBuilder() *WorkloadBuilder {
	return &WorkloadBuilder{}
}

// SetID sets the ID for the workload.
func (b *WorkloadBuilder) SetID(id string) *WorkloadBuilder {
	b.id = id
	return b
}

// SetWorkloadName sets the name for the workload.
func (b *WorkloadBuilder) SetWorkloadName(workloadName string) *WorkloadBuilder {
	b.workloadName = workloadName
	return b
}

// SetSeed sets the seed value for the workload.
func (b *WorkloadBuilder) SetSeed(seed int64) *WorkloadBuilder {
	b.seed = seed
	return b
}

// EnableDebugLogging enables or disables debug logging.
func (b *WorkloadBuilder) EnableDebugLogging(enabled bool) *WorkloadBuilder {
	b.debugLoggingEnabled = enabled
	return b
}

// SetTimescaleAdjustmentFactor sets the timescale adjustment factor.
func (b *WorkloadBuilder) SetTimescaleAdjustmentFactor(factor float64) *WorkloadBuilder {
	b.timescaleAdjustmentFactor = factor
	return b
}

// SetRemoteStorageDefinition sets the remote storage definition.
func (b *WorkloadBuilder) SetRemoteStorageDefinition(def *proto.RemoteStorageDefinition) *WorkloadBuilder {
	b.remoteStorageDefinition = def
	return b
}

// SetAtom sets the zap.AtomicLevel value.
func (b *WorkloadBuilder) SetAtom(atom *zap.AtomicLevel) *WorkloadBuilder {
	b.atom = atom
	return b
}

// Build creates a Workload instance with the specified values.
func (b *WorkloadBuilder) Build() *BasicWorkload {
	return &BasicWorkload{
		Id:                        b.id, // Same ID as the driver.
		Name:                      b.workloadName,
		WorkloadState:             WorkloadReady,
		TimeElapsed:               time.Duration(0),
		Seed:                      b.seed,
		RegisteredTime:            time.Now(),
		NumTasksExecuted:          0,
		NumEventsProcessed:        0,
		NumSessionsCreated:        0,
		NumActiveSessions:         0,
		NumActiveTrainings:        0,
		DebugLoggingEnabled:       b.debugLoggingEnabled,
		TimescaleAdjustmentFactor: b.timescaleAdjustmentFactor,
		WorkloadType:              UnspecifiedWorkload,
		EventsProcessed:           make([]*WorkloadEvent, 0),
		atom:                      b.atom,
		sessionsMap:               hashmap.New(32),
		trainingStartedTimes:      hashmap.New(32),
		CurrentTick:               0,
		Sessions:                  make([]WorkloadSession, 0), // For template workloads, this will be overwritten.
		SumTickDurationsMillis:    0,
		TickDurationsMillis:       make([]int64, 0),
		RemoteStorageDefinition:   b.remoteStorageDefinition,
	}
}

func NewWorkload(id string, workloadName string, seed int64, debugLoggingEnabled bool, timescaleAdjustmentFactor float64,
	remoteStorageDefinition *proto.RemoteStorageDefinition, atom *zap.AtomicLevel) *BasicWorkload {

	workload := &BasicWorkload{
		Id:                        id, // Same ID as the driver.
		Name:                      workloadName,
		WorkloadState:             WorkloadReady,
		TimeElapsed:               time.Duration(0),
		Seed:                      seed,
		RegisteredTime:            time.Now(),
		NumTasksExecuted:          0,
		NumEventsProcessed:        0,
		NumSessionsCreated:        0,
		NumActiveSessions:         0,
		NumActiveTrainings:        0,
		DebugLoggingEnabled:       debugLoggingEnabled,
		TimescaleAdjustmentFactor: timescaleAdjustmentFactor,
		WorkloadType:              UnspecifiedWorkload,
		EventsProcessed:           make([]*WorkloadEvent, 0),
		atom:                      atom,
		sessionsMap:               hashmap.New(32),
		trainingStartedTimes:      hashmap.New(32),
		CurrentTick:               0,
		Sessions:                  make([]WorkloadSession, 0), // For template workloads, this will be overwritten.
		SumTickDurationsMillis:    0,
		TickDurationsMillis:       make([]int64, 0),
		RemoteStorageDefinition:   remoteStorageDefinition,
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	workload.logger = logger
	workload.sugaredLogger = logger.Sugar()

	return workload
}

// PauseWaitBeginning should be called by the WorkloadDriver if it finds that the workload is paused, and it
// actually begins waiting. This will prevent any of the time during which the workload was paused from being
// counted towards the workload's runtime.
func (w *BasicWorkload) PauseWaitBeginning() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.pauseWaitBegin = time.Now()
}

// GetRemoteStorageDefinition returns the *proto.RemoteStorageDefinition used by the Workload.
func (w *BasicWorkload) GetRemoteStorageDefinition() *proto.RemoteStorageDefinition {
	return w.RemoteStorageDefinition
}

// RegisterOnCriticalErrorHandler registers a critical error handler for the target workload.
//
// If there is already a critical handler error registered for the target workload, then the existing
// critical error handler is overwritten.
func (w *BasicWorkload) RegisterOnCriticalErrorHandler(handler WorkloadErrorHandler) {
	w.onCriticalError = handler
}

// RegisterOnNonCriticalErrorHandler registers a non-critical error handler for the target workload.
//
// If there is already a non-critical handler error registered for the target workload, then the existing
// non-critical error handler is overwritten.
func (w *BasicWorkload) RegisterOnNonCriticalErrorHandler(handler WorkloadErrorHandler) {
	w.onNonCriticalError = handler
}

// GetTickDurationsMillis returns a slice containing the clock time that elapsed for each tick
// of the workload in order, in milliseconds.
func (w *BasicWorkload) GetTickDurationsMillis() []int64 {
	return w.TickDurationsMillis
}

// SetPausing will set the workload to the pausing state, which means that it is finishing
// the processing of its current tick before halting until being unpaused.
func (w *BasicWorkload) SetPausing() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.WorkloadState != WorkloadRunning {
		w.logger.Error("Cannot transition workload to 'pausing' state. Workload is not running.",
			zap.String("workload_state", w.WorkloadState.String()),
			zap.String("workload_id", w.Id),
			zap.String("workload-state", string(w.WorkloadState)))
		return ErrWorkloadNotPaused
	}

	w.WorkloadState = WorkloadPausing
	return nil
}

// SetPaused will set the workload to the paused state.
func (w *BasicWorkload) SetPaused() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.WorkloadState != WorkloadPausing {
		w.logger.Error("Cannot transition workload to 'paused' state. Workload is not in 'pausing' state.",
			zap.String("workload_state", w.WorkloadState.String()),
			zap.String("workload_id", w.Id),
			zap.String("workload-state", string(w.WorkloadState)))
		return ErrWorkloadNotPaused
	}

	w.WorkloadState = WorkloadPaused
	return nil
}

// Unpause will set the workload to the unpaused state.
func (w *BasicWorkload) Unpause() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.WorkloadState != WorkloadPaused && w.WorkloadState != WorkloadPausing {
		w.logger.Error("Cannot unpause workload. Workload is not paused.",
			zap.String("workload_state", w.WorkloadState.String()),
			zap.String("workload_id", w.Id),
			zap.String("workload-state", string(w.WorkloadState)))
		return ErrWorkloadNotPaused
	}

	w.WorkloadState = WorkloadRunning

	// pauseWaitBegin is set to zero after being processed.
	// So, if it is currently zero, then we're not paused, and we should do nothing.
	if w.pauseWaitBegin.IsZero() {
		return nil
	}

	// Compute how long we were paused, increment the counters, and then zero out the pauseWaitBegin field.
	pauseDuration := time.Since(w.pauseWaitBegin)
	w.timeSpentPaused += pauseDuration
	w.TimeSpentPausedMillis = w.timeSpentPaused.Milliseconds()

	w.pauseWaitBegin = time.Time{} // Zero it out.
	return nil
}

// TickCompleted is called by the driver after each tick.
// Updates the time elapsed, current tick, and simulation clock time.
func (w *BasicWorkload) TickCompleted(tick int64, simClock time.Time) {
	w.mu.Lock()
	w.CurrentTick = tick
	w.SimulationClockTimeStr = simClock.String()
	w.mu.Unlock()

	w.UpdateTimeElapsed()
}

// AddFullTickDuration is called to record how long a tick lasted, including the "artificial" sleep that is performed
// by the WorkloadDriver in order to fully simulate ticks that otherwise have no work/events to be processed.
func (w *BasicWorkload) AddFullTickDuration(timeElapsed time.Duration) {
	timeElapsedMs := timeElapsed.Milliseconds()
	w.TickDurationsMillis = append(w.TickDurationsMillis, timeElapsedMs)
	w.SumTickDurationsMillis += timeElapsedMs
}

// GetCurrentTick returns the current tick.
func (w *BasicWorkload) GetCurrentTick() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.CurrentTick
}

// GetSimulationClockTimeStr returns the simulation clock time.
func (w *BasicWorkload) GetSimulationClockTimeStr() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.SimulationClockTimeStr
}

// SetSessions sets the sessions that will be involved in this workload.
//
// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
func (w *BasicWorkload) SetSessions(sessions []WorkloadSession) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Sessions = sessions
	w.sessionsSet = true

	// Add each session to our internal mapping and initialize the session.
	for _, session := range sessions {
		if err := session.SetState(SessionAwaitingStart); err != nil {
			w.logger.Error("Failed to set session state.", zap.String("session_id", session.GetId()), zap.Error(err))
		}

		w.sessionsMap.Set(session.GetId(), session)
	}
}

// SetSource sets the source of the workload, namely a template or a preset.
// This defers the execution of the method to the `BasicWorkload::workload` field.
func (w *BasicWorkload) SetSource(source interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.workloadInstance.SetSource(source)
}

// GetSessions returns the sessions involved in this workload.
func (w *BasicWorkload) GetSessions() []WorkloadSession {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Sessions
}

// GetWorkloadType gets the type of workload (TRACE, PRESET, or TEMPLATE).
func (w *BasicWorkload) GetWorkloadType() WorkloadType {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType
}

// IsPresetWorkload returns true if this workload was created using a preset.
func (w *BasicWorkload) IsPresetWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == PresetWorkload
}

// IsTemplateWorkload returns true if this workload was created using a template.
func (w *BasicWorkload) IsTemplateWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == TemplateWorkload
}

// IsTraceWorkload returns true if this workload was created using the trace data.
func (w *BasicWorkload) IsTraceWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == TraceWorkload
}

// GetWorkloadSource returns the "source" of the workload.
// If this is a preset workload, return the name of the preset.
// If this is a trace workload, return the trace information.
// If this is a template workload, return the template information.
func (w *BasicWorkload) GetWorkloadSource() interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.workloadInstance.GetWorkloadSource()
}

// GetProcessedEvents returns the events processed during this workload (so far).
func (w *BasicWorkload) GetProcessedEvents() []*WorkloadEvent {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.EventsProcessed
}

// TerminateWorkloadPrematurely stops the workload.
func (w *BasicWorkload) TerminateWorkloadPrematurely(simulationTimestamp time.Time) (time.Time, error) {
	if !w.IsInProgress() {
		w.logger.Error("Cannot stop as I am not running.", zap.String("workload_id", w.Id), zap.String("workload-state", string(w.WorkloadState)))
		return time.Now(), ErrWorkloadNotRunning
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()

	w.EndTime = now
	w.WorkloadState = WorkloadTerminated
	w.NumEventsProcessed += 1

	// workloadEvent := NewWorkloadEvent(len(w.EventsProcessed), uuid.NewString(), "workload-terminated", "N/A", simulationTimestamp.String(), now.String(), true, nil)
	workloadEvent := NewEmptyWorkloadEvent().
		WithIndex(len(w.EventsProcessed)).
		WithEventId(uuid.NewString()).
		WithEventNameString(WorkloadTerminatedEventName).
		WithSessionId("N/A").
		WithEventTimestamp(simulationTimestamp).
		WithProcessedAtTime(now).
		WithSimProcessedAtTime(simulationTimestamp).
		WithProcessedStatus(true)

	w.EventsProcessed = append(w.EventsProcessed, workloadEvent)

	// w.EventsProcessed = append(w.EventsProcessed, &WorkloadEvent{
	// 	Index:                 len(w.EventsProcessed),
	// 	Id:                    uuid.NewString(),
	// 	Name:                  "workload-terminated",
	// 	Session:               "N/A",
	// 	Timestamp:             simulationTimestamp.String(),
	// 	ProcessedAt:           now.String(),
	// 	ProcessedSuccessfully: true,
	// })

	w.logger.Debug("Stopped.", zap.String("workload_id", w.Id))
	return w.EndTime, nil
}

// StartWorkload starts the Workload.
//
// If the workload is already running, then an error is returned.
// Likewise, if the workload was previously running but has already stopped, then an error is returned.
func (w *BasicWorkload) StartWorkload() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.WorkloadState != WorkloadReady {
		return fmt.Errorf("%w: cannot start workload that is in state '%s'", ErrInvalidState, GetWorkloadStateAsString(w.WorkloadState))
	}

	w.WorkloadState = WorkloadRunning
	w.StartTime = time.Now()

	return nil
}

func (w *BasicWorkload) GetTimescaleAdjustmentFactor() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.TimescaleAdjustmentFactor
}

// SetWorkloadCompleted marks the workload as having completed successfully.
func (w *BasicWorkload) SetWorkloadCompleted() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.WorkloadState = WorkloadFinished
	w.EndTime = time.Now()
	w.WorkloadDuration = time.Since(w.StartTime)
}

// GetErrorMessage gets the error message associated with the workload.
// If the workload is not in an ERROR state, then this returns the empty string and false.
// If the workload is in an ERROR state, then the boolean returned will be true.
func (w *BasicWorkload) GetErrorMessage() (string, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.WorkloadState == WorkloadErred {
		return w.ErrorMessage, true
	}

	return "", false
}

// SetErrorMessage sets the error message for the workload.
func (w *BasicWorkload) SetErrorMessage(errorMessage string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.ErrorMessage = errorMessage
}

// IsDebugLoggingEnabled returns a flag indicating whether debug logging is enabled.
func (w *BasicWorkload) IsDebugLoggingEnabled() bool {
	return w.DebugLoggingEnabled
}

// SetDebugLoggingEnabled enables or disables debug logging for the workload.
func (w *BasicWorkload) SetDebugLoggingEnabled(enabled bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.DebugLoggingEnabled = enabled
}

// SetSeed sets the workload's seed. Can only be performed once. If attempted again, this will panic.
func (w *BasicWorkload) SetSeed(seed int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.seedSet {
		panic(fmt.Sprintf("Workload seed has already been set to value %d", w.Seed))
	}

	w.Seed = seed
	w.seedSet = true
}

// GetSeed returns the workload's seed.
func (w *BasicWorkload) GetSeed() int64 {
	return w.Seed
}

// GetWorkloadState returns the current state of the workload.
func (w *BasicWorkload) GetWorkloadState() WorkloadState {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.WorkloadState
}

// SetWorkloadState sets the state of the workload.
func (w *BasicWorkload) SetWorkloadState(state WorkloadState) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.WorkloadState = state
}

// GetStartTime returns the time that the workload was started.
func (w *BasicWorkload) GetStartTime() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.StartTime
}

// GetEndTime returns the time at which the workload finished.
// If the workload hasn't finished yet, the returned boolean will be false.
// If the workload has finished, then the returned boolean will be true.
func (w *BasicWorkload) GetEndTime() (time.Time, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.IsFinished() {
		return w.EndTime, true
	}

	return time.Time{}, false
}

// GetRegisteredTime returns the time that the workload was registered.
func (w *BasicWorkload) GetRegisteredTime() time.Time {
	return w.RegisteredTime
}

// GetTimeElapsed returns the time elapsed, which is computed at the time that data is requested by the user.
func (w *BasicWorkload) GetTimeElapsed() time.Duration {
	return w.TimeElapsed
}

// GetTimeElapsedAsString returns the time elapsed as a string, which is computed at the time that data is requested by the user.
//
// IMPORTANT: This updates the w.TimeElapsedStr field (setting it to w.TimeElapsed.String()) before returning it.
func (w *BasicWorkload) GetTimeElapsedAsString() string {
	w.TimeElapsedStr = w.TimeElapsed.String()
	return w.TimeElapsed.String()
}

// SetTimeElapsed updates the time elapsed.
func (w *BasicWorkload) SetTimeElapsed(timeElapsed time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.TimeElapsed = timeElapsed
	w.TimeElapsedStr = w.TimeElapsed.String()
}

// UpdateTimeElapsed instructs the Workload to recompute its 'time elapsed' field.
func (w *BasicWorkload) UpdateTimeElapsed() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// If we're currently waiting in a paused state, then don't update the time at all.
	if !w.pauseWaitBegin.IsZero() {
		return
	}

	// First, compute the total time elapsed.
	timeElapsed := time.Since(w.StartTime)

	// Second, subtract the time we have spent paused.
	w.TimeElapsed = timeElapsed - w.timeSpentPaused
	w.TimeElapsedStr = w.TimeElapsed.String()
}

// GetNumEventsProcessed returns the number of events processed by the workload.
func (w *BasicWorkload) GetNumEventsProcessed() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.NumEventsProcessed
}

// WorkloadName returns the name of the workload.
// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
func (w *BasicWorkload) WorkloadName() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Name
}

// ProcessedEvent is called after an event is processed for the Workload.
// Just updates some internal metrics.
//
// This method is thread safe.
func (w *BasicWorkload) ProcessedEvent(evt *WorkloadEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.NumEventsProcessed += 1
	evt.Index = len(w.EventsProcessed)
	w.EventsProcessed = append(w.EventsProcessed, evt)

	metrics.PrometheusMetricsWrapperInstance.WorkloadEventsProcessed.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)

	w.sugaredLogger.Debugf("Workload %s processed event '%s' targeting session '%s'", w.Name, evt.Name, evt.Session)
}

// SessionCreated is called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *BasicWorkload) SessionCreated(sessionId string, metadata SessionMetadata) {
	w.mu.Lock()
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1
	w.mu.Unlock()

	metrics.PrometheusMetricsWrapperInstance.WorkloadTotalNumSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveNumSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)

	w.sessionsMap.Get(sessionId)

	w.workloadInstance.SessionCreated(sessionId, metadata)
}

// SessionStopped is called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *BasicWorkload) SessionStopped(sessionId string, evt Event) {
	w.mu.Lock()
	w.NumActiveSessions -= 1
	w.mu.Unlock()

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveNumSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Sub(1)

	w.workloadInstance.SessionStopped(sessionId, evt)
}

// TrainingStarted is called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *BasicWorkload) TrainingStarted(sessionId string, evt Event) {
	w.workloadInstance.TrainingStarted(sessionId, evt)

	w.trainingStartedTimes.Set(sessionId, time.Now())

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveTrainingSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)
}

// TrainingStopped is called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *BasicWorkload) TrainingStopped(sessionId string, evt Event) {
	w.workloadInstance.TrainingStopped(sessionId, evt)

	metrics.PrometheusMetricsWrapperInstance.WorkloadTrainingEventsCompleted.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveTrainingSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Sub(1)

	val, loaded := w.trainingStartedTimes.Get(sessionId)
	if !loaded {
		w.logger.Error("Could not load 'training-started' time for Session upon training stopping.",
			zap.String("session_id", sessionId))
	} else {
		trainingDuration := time.Since(val.(time.Time))

		metrics.PrometheusMetricsWrapperInstance.WorkloadTrainingEventDurationMilliseconds.
			With(prometheus.Labels{"workload_id": w.Id}).
			Observe(float64(trainingDuration.Milliseconds()))
	}
}

// GetId returns the unique ID of the workload.
func (w *BasicWorkload) GetId() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Id
}

// IsTerminated returns true if the workload stopped because it was explicitly terminated early/premature.
func (w *BasicWorkload) IsTerminated() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadTerminated
}

// IsReady returns true if the workload is registered and ready to be started.
func (w *BasicWorkload) IsReady() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadReady
}

// IsErred returns true if the workload stopped due to an error.
func (w *BasicWorkload) IsErred() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadErred
}

// IsRunning returns true if the workload is actively running (i.e., not paused).
func (w *BasicWorkload) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadRunning
}

// IsPausing returns true if the workload is pausing, meaning that it is finishing the processing
// of its current tick before halting until it is un-paused.
func (w *BasicWorkload) IsPausing() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadPausing
}

// IsPaused returns true if the workload is paused.
func (w *BasicWorkload) IsPaused() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadPaused
}

// IsInProgress returns true if the workload is actively running, pausing, or paused.
func (w *BasicWorkload) IsInProgress() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.IsRunning() || w.IsPausing() || w.IsPausing()
}

// IsFinished returns true if the workload stopped naturally/successfully after processing all events.
func (w *BasicWorkload) IsFinished() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.IsErred() || w.DidCompleteSuccessfully()
}

// DidCompleteSuccessfully returns true if the workload stopped naturally/successfully
// after processing all events.
func (w *BasicWorkload) DidCompleteSuccessfully() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadFinished
}

func (w *BasicWorkload) String() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	out, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}

	return string(out)
}
