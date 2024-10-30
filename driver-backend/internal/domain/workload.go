package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
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
	WorkloadReady      WorkloadState = "WorkloadReady"      // workloadImpl is registered and ready to be started.
	WorkloadRunning    WorkloadState = "WorkloadRunning"    // workloadImpl is actively running/in-progress.
	WorkloadFinished   WorkloadState = "WorkloadFinished"   // workloadImpl stopped naturally/successfully after processing all events.
	WorkloadErred      WorkloadState = "WorkloadErred"      // workloadImpl stopped due to an error.
	WorkloadTerminated WorkloadState = "WorkloadTerminated" // workloadImpl stopped because it was explicitly terminated early/premature.

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
)

type WorkloadState string

// WorkloadType defines a type that a workload can have/be.
//
// Workloads can be of several different types, namely 'preset' and 'template' and possibly 'trace'.
// Have not fully committed to making 'trace' a separate type from 'preset'.
//
// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
// have properties that the user can specify and change before submitting the workload for registration.
type WorkloadType string

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
// indicated as such in the method's documenation.
type Workload interface {
	// IsTerminated Returns true if the workload stopped because it was explicitly terminated early/premature.
	IsTerminated() bool
	// IsReady Returns true if the workload is registered and ready to be started.
	IsReady() bool
	// IsErred Returns true if the workload stopped due to an error.
	IsErred() bool
	// IsRunning Returns true if the workload is actively running/in-progress.
	IsRunning() bool
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
	// GetTimeElasped Returns the time elapsed, which is computed at the time that data is requested by the user.
	GetTimeElasped() time.Duration
	// GetTimeElaspedAsString Returns the time elapsed as a string, which is computed at the time that data is requested by the user.
	GetTimeElaspedAsString() string
	// SetTimeElasped Updates the time elapsed.
	SetTimeElasped(time.Duration)
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
	SessionCreated(string)
	// SessionStopped is Called when a Session is stopped for/in the Workload.
	// Just updates some internal metrics.
	SessionStopped(string)
	// TrainingStarted is Called when a training starts during/in the workload.
	// Just updates some internal metrics.
	TrainingStarted(string)
	// TrainingStopped is Called when a training stops during/in the workload.
	// Just updates some internal metrics.
	TrainingStopped(string)
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
	// Updates the time elapsed, current tick, and simulation clock time.
	TickCompleted(int64, time.Time)
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

type WorkloadEvent struct {
	Index                 int    `json:"idx"`                     // Index of the event relative to the workload in which the event occurred. The first event to occur has index 0.
	Id                    string `json:"id"`                      // Unique ID of the event.
	Name                  string `json:"name"`                    // The name of the event.
	Session               string `json:"session"`                 // The ID of the session targeted by the event.
	Timestamp             string `json:"timestamp"`               // The timestamp specified by the trace data/template/preset.
	ProcessedAt           string `json:"processed_at"`            // The real-world clocktime at which the event was processed.
	SimProcessedAt        string `json:"sim_processed_at"`        // The simulation clocktime at which the event was processed. May differ from the 'Timestamp' field if there were delays.
	ProcessedSuccessfully bool   `json:"processed_successfully"`  // True if the event was processed without error.
	ErrorMessage          string `json:"error_message,omitempty"` // Error message from the error that caused the event to not be processed successfully.
}

// NewEmptyWorkloadEvent returns an "empty" workload event -- with none of its fields populated.
// This is intended to be used with the WorkloadEvent::WithX functions, as
// another means of constructing WorkloadEvent structs.
//
// Note: WorkloadEvent::ProcessedSuccessfully field is initialized to true.
// It can be set to false explicitly via WorkloadEvent::WithProcessedStatus,
// by passing a non-nil error to WorkloadEvent::WithError,
// or by passing a non-empty string to WorkloadEvent::WithErrorMessage.
func NewEmptyWorkloadEvent() *WorkloadEvent {
	return &WorkloadEvent{
		ProcessedSuccessfully: true,
	}
}

func NewWorkloadEvent(idx int, id string, name string, session string, timestamp string, processedAt string, simulationProcessedAt string, processedSuccessfully bool, err error) *WorkloadEvent {
	event := &WorkloadEvent{
		Index:                 idx,
		Id:                    id,
		Name:                  name,
		Session:               session,
		Timestamp:             timestamp,
		ProcessedAt:           processedAt,
		ProcessedSuccessfully: processedSuccessfully,
		SimProcessedAt:        simulationProcessedAt,
	}

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return event
}

// WithIndex should be used with caution; the workload implementation should be the only entity that uses this function.
func (evt *WorkloadEvent) WithIndex(eventIndex int) *WorkloadEvent {
	evt.Index = eventIndex
	return evt
}

func (evt *WorkloadEvent) WithEventId(eventId string) *WorkloadEvent {
	evt.Id = eventId
	return evt
}

func (evt *WorkloadEvent) WithSessionId(sessionId string) *WorkloadEvent {
	evt.Session = sessionId
	return evt
}

func (evt *WorkloadEvent) WithEventName(name NamedEvent) *WorkloadEvent {
	evt.Name = name.String()
	return evt
}

func (evt *WorkloadEvent) WithEventNameString(name string) *WorkloadEvent {
	evt.Name = name
	return evt
}

func (evt *WorkloadEvent) WithEventTimestamp(eventTimestamp time.Time) *WorkloadEvent {
	evt.Timestamp = eventTimestamp.String()
	return evt
}

func (evt *WorkloadEvent) WithEventTimestampAsString(eventTimestamp string) *WorkloadEvent {
	evt.Timestamp = eventTimestamp
	return evt
}

func (evt *WorkloadEvent) WithProcessedAtTime(processedAt time.Time) *WorkloadEvent {
	evt.ProcessedAt = processedAt.String()
	return evt
}

func (evt *WorkloadEvent) WithProcessedAtTimeAsString(processedAt string) *WorkloadEvent {
	evt.ProcessedAt = processedAt
	return evt
}

func (evt *WorkloadEvent) WithSimProcessedAtTime(simulationProcessedAt time.Time) *WorkloadEvent {
	evt.SimProcessedAt = simulationProcessedAt.String()
	return evt
}

func (evt *WorkloadEvent) WithSimProcessedAtTimeAsString(simulationProcessedAt string) *WorkloadEvent {
	evt.SimProcessedAt = simulationProcessedAt
	return evt
}

func (evt *WorkloadEvent) WithProcessedStatus(success bool) *WorkloadEvent {
	evt.ProcessedSuccessfully = success
	return evt
}

// WithError conditionally sets the 'ErrorMessage' field of the WorkloadEvent struct if the error argument is non-nil.
//
// Note: If the event is non-nil, then this also updates the WorkloadEvent::ProcessedSuccessfully field, setting it to false.
// You can manually flip it back to true, if desired, by calling the WorkloadEvent::WithProcessedStatus method and passing true.
func (evt *WorkloadEvent) WithError(err error) *WorkloadEvent {
	if err != nil {
		evt.ErrorMessage = err.Error()
		evt.ProcessedSuccessfully = false
	}

	return evt
}

// WithErrorMessage sets the error message of the WorkloadEvent.
//
// Note: If the error message is non-empty (i.e., has length >= 1), then the ProcessedSuccessfully field is automatically set to false.
// You can manually flip it back to true, if desired, by calling the WorkloadEvent::WithProcessedStatus method and passing true.
func (evt *WorkloadEvent) WithErrorMessage(errorMessage string) *WorkloadEvent {
	evt.ErrorMessage = errorMessage

	if len(errorMessage) >= 1 {
		evt.ProcessedSuccessfully = false
	}

	return evt
}

func (evt *WorkloadEvent) String() string {
	out, err := json.Marshal(evt)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type workloadImpl struct {
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

	// workloadSource interface{} `json:"-"`

	// This is basically the child struct.
	// So, if this is a preset workload, then this is the WorkloadFromPreset struct.
	// We use this so we can delegate certain method calls to the child/derived struct.
	workload             Workload
	workloadSource       interface{}
	mu                   sync.RWMutex
	sessionsMap          *hashmap.HashMap // Internal mapping of session ID to session.
	trainingStartedTimes *hashmap.HashMap // Internal mapping of session ID to the time at which it began training.
	seedSet              bool             // Flag keeping track of whether we've already set the seed for this workload.
	sessionsSet          bool             // Flag keeping track of whether we've already set the sessions for this workload.
}

func NewWorkload(id string, workloadName string, seed int64, debugLoggingEnabled bool, timescaleAdjustmentFactor float64, atom *zap.AtomicLevel) Workload {
	workload := &workloadImpl{
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

// TickCompleted is called by the driver after each tick.
// Updates the time elapsed, current tick, and simulation clock time.
func (w *workloadImpl) TickCompleted(tick int64, simClock time.Time) {
	w.mu.Lock()
	w.CurrentTick = tick
	w.SimulationClockTimeStr = simClock.String()
	w.mu.Unlock()

	w.UpdateTimeElapsed()
}

// GetCurrentTick returns the current tick.
func (w *workloadImpl) GetCurrentTick() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.CurrentTick
}

// GetSimulationClockTimeStr returns the simulation clock time.
func (w *workloadImpl) GetSimulationClockTimeStr() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.SimulationClockTimeStr
}

// SetSessions sets the sessions that will be involved in this workload.
//
// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
func (w *workloadImpl) SetSessions(sessions []WorkloadSession) {
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
// This defers the execution of the method to the `workloadImpl::workload` field.
func (w *workloadImpl) SetSource(source interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.workload.SetSource(source)
}

// GetSessions returns the sessions involved in this workload.
func (w *workloadImpl) GetSessions() []WorkloadSession {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Sessions
}

// GetWorkloadType gets the type of workload (TRACE, PRESET, or TEMPLATE).
func (w *workloadImpl) GetWorkloadType() WorkloadType {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType
}

// IsPresetWorkload returns true if this workload was created using a preset.
func (w *workloadImpl) IsPresetWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == PresetWorkload
}

// IsTemplateWorkload returns true if this workload was created using a template.
func (w *workloadImpl) IsTemplateWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == TemplateWorkload
}

// IsTraceWorkload returns true if this workload was created using the trace data.
func (w *workloadImpl) IsTraceWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == TraceWorkload
}

// GetWorkloadSource returns the "source" of the workload.
// If this is a preset workload, return the name of the preset.
// If this is a trace workload, return the trace information.
// If this is a template workload, return the template information.
func (w *workloadImpl) GetWorkloadSource() interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.workload.GetWorkloadSource()
}

// GetProcessedEvents returns the events processed during this workload (so far).
func (w *workloadImpl) GetProcessedEvents() []*WorkloadEvent {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.EventsProcessed
}

// TerminateWorkloadPrematurely stops the workload.
func (w *workloadImpl) TerminateWorkloadPrematurely(simulationTimestamp time.Time) (time.Time, error) {
	if !w.IsRunning() {
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
func (w *workloadImpl) StartWorkload() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.WorkloadState != WorkloadReady {
		return fmt.Errorf("%w: cannot start workload that is in state '%s'", ErrInvalidState, GetWorkloadStateAsString(w.WorkloadState))
	}

	w.WorkloadState = WorkloadRunning
	w.StartTime = time.Now()

	return nil
}

func (w *workloadImpl) GetTimescaleAdjustmentFactor() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.TimescaleAdjustmentFactor
}

// SetWorkloadCompleted marks the workload as having completed successfully.
func (w *workloadImpl) SetWorkloadCompleted() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.WorkloadState = WorkloadFinished
	w.EndTime = time.Now()
	w.WorkloadDuration = time.Since(w.StartTime)
}

// GetErrorMessage gets the error message associated with the workload.
// If the workload is not in an ERROR state, then this returns the empty string and false.
// If the workload is in an ERROR state, then the boolean returned will be true.
func (w *workloadImpl) GetErrorMessage() (string, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.WorkloadState == WorkloadErred {
		return w.ErrorMessage, true
	}

	return "", false
}

// SetErrorMessage sets the error message for the workload.
func (w *workloadImpl) SetErrorMessage(errorMessage string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.ErrorMessage = errorMessage
}

// IsDebugLoggingEnabled returns a flag indicating whether debug logging is enabled.
func (w *workloadImpl) IsDebugLoggingEnabled() bool {
	return w.DebugLoggingEnabled
}

// SetDebugLoggingEnabled enables or disables debug logging for the workload.
func (w *workloadImpl) SetDebugLoggingEnabled(enabled bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.DebugLoggingEnabled = enabled
}

// SetSeed sets the workload's seed. Can only be performed once. If attempted again, this will panic.
func (w *workloadImpl) SetSeed(seed int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.seedSet {
		panic(fmt.Sprintf("Workload seed has already been set to value %d", w.Seed))
	}

	w.Seed = seed
	w.seedSet = true
}

// GetSeed returns the workload's seed.
func (w *workloadImpl) GetSeed() int64 {
	return w.Seed
}

// GetWorkloadState returns the current state of the workload.
func (w *workloadImpl) GetWorkloadState() WorkloadState {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.WorkloadState
}

// SetWorkloadState sets the state of the workload.
func (w *workloadImpl) SetWorkloadState(state WorkloadState) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.WorkloadState = state
}

// GetStartTime returns the time that the workload was started.
func (w *workloadImpl) GetStartTime() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.StartTime
}

// GetEndTime returns the time at which the workload finished.
// If the workload hasn't finished yet, the returned boolean will be false.
// If the workload has finished, then the returned boolean will be true.
func (w *workloadImpl) GetEndTime() (time.Time, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.IsFinished() {
		return w.EndTime, true
	}

	return time.Time{}, false
}

// GetRegisteredTime returns the time that the workload was registered.
func (w *workloadImpl) GetRegisteredTime() time.Time {
	return w.RegisteredTime
}

// GetTimeElasped returns the time elapsed, which is computed at the time that data is requested by the user.
func (w *workloadImpl) GetTimeElasped() time.Duration {
	return w.TimeElapsed
}

// GetTimeElaspedAsString returns the time elapsed as a string, which is computed at the time that data is requested by the user.
//
// IMPORTANT: This updates the w.TimeElapsedStr field (setting it to w.TimeElapsed.String()) before returning it.
func (w *workloadImpl) GetTimeElaspedAsString() string {
	w.TimeElapsedStr = w.TimeElapsed.String()
	return w.TimeElapsed.String()
}

// SetTimeElasped updates the time elapsed.
func (w *workloadImpl) SetTimeElasped(timeElapsed time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.TimeElapsed = timeElapsed
	w.TimeElapsedStr = w.TimeElapsed.String()
}

// UpdateTimeElapsed instructs the Workload to recompute its 'time elapsed' field.
func (w *workloadImpl) UpdateTimeElapsed() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.TimeElapsed = time.Since(w.StartTime)
	w.TimeElapsedStr = w.TimeElapsed.String()
}

// GetNumEventsProcessed returns the number of events processed by the workload.
func (w *workloadImpl) GetNumEventsProcessed() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.NumEventsProcessed
}

// WorkloadName returns the name of the workload.
// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
func (w *workloadImpl) WorkloadName() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Name
}

// ProcessedEvent is called after an event is processed for the Workload.
// Just updates some internal metrics.
//
// This method is thread safe.
func (w *workloadImpl) ProcessedEvent(evt *WorkloadEvent) {
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
func (w *workloadImpl) SessionCreated(sessionId string) {
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

	w.workload.SessionCreated(sessionId)
}

// SessionStopped is called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) SessionStopped(sessionId string) {
	w.mu.Lock()
	w.NumActiveSessions -= 1
	w.mu.Unlock()

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveNumSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Sub(1)

	w.workload.SessionStopped(sessionId)
}

// TrainingStarted is called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStarted(sessionId string) {
	w.workload.TrainingStarted(sessionId)

	w.trainingStartedTimes.Set(sessionId, time.Now())

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveTrainingSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)
}

// TrainingStopped is called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStopped(sessionId string) {
	w.workload.TrainingStopped(sessionId)

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
func (w *workloadImpl) GetId() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Id
}

// IsTerminated returns true if the workload stopped because it was explicitly terminated early/premature.
func (w *workloadImpl) IsTerminated() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadTerminated
}

// IsReady returns true if the workload is registered and ready to be started.
func (w *workloadImpl) IsReady() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadReady
}

// IsErred returns true if the workload stopped due to an error.
func (w *workloadImpl) IsErred() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadErred
}

// IsRunning returns true if the workload is actively running/in-progress.
func (w *workloadImpl) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadRunning
}

// IsFinished returns true if the workload stopped naturally/successfully after processing all events.
func (w *workloadImpl) IsFinished() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.IsErred() || w.DidCompleteSuccessfully()
}

// DidCompleteSuccessfully returns true if the workload stopped naturally/successfully
// after processing all events.
func (w *workloadImpl) DidCompleteSuccessfully() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadFinished
}

func (w *workloadImpl) String() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	out, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}

	return string(out)
}
