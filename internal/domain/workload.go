package domain

import (
	"encoding/json"
	"errors"
	"fmt"
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

// Workloads can be of several different types, namely 'preset' and 'template' and possibly 'trace'.
// Have not fully committed to making 'trace' a separate type from 'preset'.
//
// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
// have properties that the user can specify and change before submitting the workload for registration.
type WorkloadType string

type WorkloadGenerator interface {
	GeneratePresetWorkload(EventConsumer, Workload, *WorkloadPreset, *WorkloadRegistrationRequest) error     // Start generating the workload.
	GenerateTemplateWorkload(EventConsumer, Workload, *WorkloadTemplate, *WorkloadRegistrationRequest) error // Start generating the workload.
	StopGeneratingWorkload()                                                                                 // Stop generating the workload prematurely.
}

type Workload interface {
	// Return true if the workload stopped because it was explicitly terminated early/premature.
	IsTerminated() bool
	// Return true if the workload is registered and ready to be started.
	IsReady() bool
	// Return true if the workload stopped due to an error.
	IsErred() bool
	// Return true if the workload is actively running/in-progress.
	IsRunning() bool
	// Return true if the workload finished in any capacity (i.e., either successfully or due to an error).
	IsFinished() bool
	// Return true if the workload stopped naturally/successfully after processing all events.
	DidCompleteSuccessfully() bool
	// To String.
	String() string
	// Return the unique ID of the workload.
	GetId() string
	// Return the name of the workload.
	// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
	WorkloadName() string
	// Return the current state of the workload.
	GetWorkloadState() WorkloadState
	// Return the time elapsed, which is computed at the time that data is requested by the user.
	GetTimeElasped() time.Duration
	// Return the time elapsed as a string, which is computed at the time that data is requested by the user.
	GetTimeElaspedAsString() string
	// Update the time elapsed.
	SetTimeElasped(time.Duration)
	// Instruct the Workload to recompute its 'time elapsed' field.
	UpdateTimeElapsed()
	// Return the number of events processed by the workload.
	GetNumEventsProcessed() int64
	// Return the time that the workload was started.
	GetStartTime() time.Time
	// Get the time at which the workload finished.
	// If the workload hasn't finished yet, the returned boolean will be false.
	// If the workload has finished, then the returned boolean will be true.
	GetEndTime() (time.Time, bool)
	// Return the time that the workload was registered.
	GetRegisteredTime() time.Time
	// Return the workload's seed.
	GetSeed() int64
	// Set the workload's seed. Can only be performed once. If attempted again, this will panic.
	SetSeed(seed int64)
	// Get the error message associated with the workload.
	// If the workload is not in an ERROR state, then this returns the empty string and false.
	// If the workload is in an ERROR state, then the boolean returned will be true.
	GetErrorMessage() (string, bool)
	// Set the error message for the workload.
	SetErrorMessage(string)
	// Return a flag indicating whether debug logging is enabled.
	IsDebugLoggingEnabled() bool
	// Enable or disable debug logging for the workload.
	SetDebugLoggingEnabled(enabled bool)
	// Set the state of the workload.
	SetWorkloadState(state WorkloadState)
	// Start the Workload.
	//
	// If the workload is already running, then an error is returned.
	// Likewise, if the workload was previously running but has already stopped, then an error is returned.
	StartWorkload() error
	// Explicitly/manually stop the workload early.
	// Return the time at which the workload terminated and an error if one occurred.
	TerminateWorkloadPrematurely(time.Time) (time.Time, error)
	// Mark the workload as having completed successfully.
	SetWorkloadCompleted()
	// Called after an event is processed for the Workload.
	// Updates some internal metrics.
	// Automatically sets the WorkloadEvent's index field.
	ProcessedEvent(*WorkloadEvent)
	// Called when a Session is created for/in the Workload.
	// Just updates some internal metrics.
	SessionCreated(string)
	// Called when a Session is stopped for/in the Workload.
	// Just updates some internal metrics.
	SessionStopped(string)
	// Called when a training starts during/in the workload.
	// Just updates some internal metrics.
	TrainingStarted(string)
	// Called when a training stops during/in the workload.
	// Just updates some internal metrics.
	TrainingStopped(string)
	// Get the type of workload (TRACE, PRESET, or TEMPLATE).
	GetWorkloadType() WorkloadType
	// Return true if this workload was created using a preset.
	IsPresetWorkload() bool
	// Return true if this workload was created using a template.
	IsTemplateWorkload() bool
	// Return true if this workload was created using the trace data.
	IsTraceWorkload() bool
	// If this is a preset workload, return the name of the preset.
	// If this is a trace workload, return the trace information.
	// If this is a template workload, return the template information.
	GetWorkloadSource() interface{}
	// Get the workload's Timescale Adjustment Factor, which effects the
	// timescale at which tickets are replayed/"simulated".
	GetTimescaleAdjustmentFactor() float64
	// Return the events processed during this workload (so far).
	GetProcessedEvents() []*WorkloadEvent
	// Return the sessions involved in this workload.
	GetSessions() []WorkloadSession
	// Set the sessions that will be involved in this workload.
	//
	// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
	SetSessions([]WorkloadSession)
	// Set the source of the workload, namely a template or a preset.
	SetSource(interface{})
	// Return the current tick.
	GetCurrentTick() int64
	// Return the simulation clock time.
	GetSimulationClockTimeStr() string
	// Called by the driver after each tick.
	// Updates the time elapsed, current tick, and simulation clock time.
	TickCompleted(int64, time.Time)
}

// This will panic if an invalid workload state is specified.
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

// Return an "empty" workload event -- with none of its fields populated.
// This is intended to be used with the WorkloadEvent::WithX functions, as
// another means of constructing WorkloadEvent structs.
func NewEmptyWorkloadEvent() *WorkloadEvent {
	return &WorkloadEvent{}
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

// Should be used with caution; the workload implementation should be the only entity that uses this function.
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

func (evt *WorkloadEvent) WithEventName(name EventName) *WorkloadEvent {
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

// Conditionally set the 'ErrorMessage' field of the WorkloadEvent struct if the error argument is non-nil.
func (evt *WorkloadEvent) WithError(err error) *WorkloadEvent {
	if err != nil {
		evt.ErrorMessage = err.Error()
	}
	return evt
}

func (evt *WorkloadEvent) WithErrorMessage(errorMessage string) *WorkloadEvent {
	evt.ErrorMessage = errorMessage
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
	logger        *zap.Logger        `json:"-"`
	sugaredLogger *zap.SugaredLogger `json:"-"`
	atom          *zap.AtomicLevel   `json:"-"`

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
	TimeElasped               time.Duration     `json:"time_elapsed"`      // Computed at the time that the data is requested by the user. This is the time elapsed SO far.
	TimeElaspedStr            string            `json:"time_elapsed_str"`
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
	workload       Workload         `json:"-"`
	workloadSource interface{}      `json:"-"`
	mu             sync.RWMutex     `json:"-"`
	sessionsMap    *hashmap.HashMap `json:"-"` // Internal mapping of session ID to session.
	seedSet        bool             `json:"-"` // Flag keeping track of whether we've already set the seed for this workload.
	sessionsSet    bool             `json:"-"` // Flag keeping track of whether we've already set the sessions for this workload.
}

func NewWorkload(id string, workloadName string, seed int64, debugLoggingEnabled bool, timescaleAdjustmentFactor float64, atom *zap.AtomicLevel) Workload {
	workload := &workloadImpl{
		Id:                        id, // Same ID as the driver.
		Name:                      workloadName,
		WorkloadState:             WorkloadReady,
		TimeElasped:               time.Duration(0),
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

// Called by the driver after each tick.
// Updates the time elapsed, current tick, and simulation clock time.
func (w *workloadImpl) TickCompleted(tick int64, simClock time.Time) {
	w.mu.Lock()
	w.CurrentTick = tick
	w.SimulationClockTimeStr = simClock.String()
	w.mu.Unlock()

	w.UpdateTimeElapsed()
}

// Return the current tick.
func (w *workloadImpl) GetCurrentTick() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.CurrentTick
}

// Return the simulation clock time.
func (w *workloadImpl) GetSimulationClockTimeStr() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.SimulationClockTimeStr
}

// Set the sessions that will be involved in this workload.
//
// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
func (w *workloadImpl) SetSessions(sessions []WorkloadSession) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Sessions = sessions
	w.sessionsSet = true

	// Add each session to our internal mapping and initialize the session.
	for _, session := range sessions {
		session.SetState(SessionAwaitingStart)

		w.sessionsMap.Set(session.GetId(), session)
	}
}

// Set the source of the workload, namely a template or a preset.
// This defers the execution of the method to the `workloadImpl::workload` field.
func (w *workloadImpl) SetSource(source interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.workload.SetSource(source)
}

// Return the sessions involved in this workload.
func (w *workloadImpl) GetSessions() []WorkloadSession {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Sessions
}

// Get the type of workload (TRACE, PRESET, or TEMPLATE).
func (w *workloadImpl) GetWorkloadType() WorkloadType {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType
}

// Return true if this workload was created using a preset.
func (w *workloadImpl) IsPresetWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == PresetWorkload
}

// Return true if this workload was created using a template.
func (w *workloadImpl) IsTemplateWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == TemplateWorkload
}

// Return true if this workload was created using the trace data.
func (w *workloadImpl) IsTraceWorkload() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadType == TraceWorkload
}

// If this is a preset workload, return the name of the preset.
// If this is a trace workload, return the trace information.
// If this is a template workload, return the template information.
func (w *workloadImpl) GetWorkloadSource() interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.workload.GetWorkloadSource()
}

// Return the events processed during this workload (so far).
func (w *workloadImpl) GetProcessedEvents() []*WorkloadEvent {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.EventsProcessed
}

// Stop the workload.
func (w *workloadImpl) TerminateWorkloadPrematurely(simulationTimestamp time.Time) (time.Time, error) {
	if !w.IsRunning() {
		w.logger.Error("Cannot stop as I am not running.", zap.String("workload-id", w.Id), zap.String("workload-state", string(w.WorkloadState)))
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

	w.logger.Debug("Stopped.", zap.String("workload-id", w.Id))
	return w.EndTime, nil
}

// Start the Workload.
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

// Mark the workload as having completed successfully.
func (w *workloadImpl) SetWorkloadCompleted() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.WorkloadState = WorkloadFinished
	w.EndTime = time.Now()
	w.WorkloadDuration = time.Since(w.StartTime)
}

// Get the error message associated with the workload.
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

// Set the error message for the workload.
func (w *workloadImpl) SetErrorMessage(errorMessage string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.ErrorMessage = errorMessage
}

// Return a flag indicating whether debug logging is enabled.
func (w *workloadImpl) IsDebugLoggingEnabled() bool {
	return w.DebugLoggingEnabled
}

// Enable or disable debug logging for the workload.
func (w *workloadImpl) SetDebugLoggingEnabled(enabled bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.DebugLoggingEnabled = enabled
}

// Set the workload's seed. Can only be performed once. If attempted again, this will panic.
func (w *workloadImpl) SetSeed(seed int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.seedSet {
		panic(fmt.Sprintf("Workload seed has already been set to value %d", w.Seed))
	}

	w.Seed = seed
	w.seedSet = true
}

// Return the workload's seed.
func (w *workloadImpl) GetSeed() int64 {
	return w.Seed
}

// Return the current state of the workload.
func (w *workloadImpl) GetWorkloadState() WorkloadState {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.WorkloadState
}

// Set the state of the workload.
func (w *workloadImpl) SetWorkloadState(state WorkloadState) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.WorkloadState = state
}

// Return the time that the workload was started.
func (w *workloadImpl) GetStartTime() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.StartTime
}

// Get the time at which the workload finished.
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

// Return the time that the workload was registered.
func (w *workloadImpl) GetRegisteredTime() time.Time {
	return w.RegisteredTime
}

// Return the time elapsed, which is computed at the time that data is requested by the user.
func (w *workloadImpl) GetTimeElasped() time.Duration {
	return w.TimeElasped
}

// Return the time elapsed as a string, which is computed at the time that data is requested by the user.
//
// IMPORTANT: This updates the w.TimeElaspedStr field (setting it to w.TimeElapsed.String()) before returning it.
func (w *workloadImpl) GetTimeElaspedAsString() string {
	w.TimeElaspedStr = w.TimeElasped.String()
	return w.TimeElasped.String()
}

// Update the time elapsed.
func (w *workloadImpl) SetTimeElasped(timeElapsed time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.TimeElasped = timeElapsed
	w.TimeElaspedStr = w.TimeElasped.String()
}

// Instruct the Workload to recompute its 'time elapsed' field.
func (w *workloadImpl) UpdateTimeElapsed() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.TimeElasped = time.Since(w.StartTime)
	w.TimeElaspedStr = w.TimeElasped.String()
}

// Return the number of events processed by the workload.
func (w *workloadImpl) GetNumEventsProcessed() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.NumEventsProcessed
}

// Return the name of the workload.
// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
func (w *workloadImpl) WorkloadName() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Name
}

// Called after an event is processed for the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) ProcessedEvent(evt *WorkloadEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.NumEventsProcessed += 1
	evt.Index = len(w.EventsProcessed)
	w.EventsProcessed = append(w.EventsProcessed, evt)

	w.sugaredLogger.Debugf("Workload %s processed event '%s' targeting session '%s'", w.Name, evt.Name, evt.Session)
}

// Called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) SessionCreated(sessionId string) {
	w.mu.Lock()
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1
	w.mu.Unlock()

	w.workload.SessionCreated(sessionId)
}

// Called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) SessionStopped(sessionId string) {
	w.mu.Lock()
	w.NumActiveSessions -= 1
	w.mu.Unlock()

	w.workload.SessionStopped(sessionId)
}

// Called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStarted(sessionId string) {
	w.workload.TrainingStarted(sessionId)
}

// Called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStopped(sessionId string) {
	w.workload.TrainingStopped(sessionId)
}

// Return the unique ID of the workload.
func (w *workloadImpl) GetId() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Id
}

// Return true if the workload stopped because it was explicitly terminated early/premature.
func (w *workloadImpl) IsTerminated() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadTerminated
}

// Return true if the workload is registered and ready to be started.
func (w *workloadImpl) IsReady() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadReady
}

// Return true if the workload stopped due to an error.
func (w *workloadImpl) IsErred() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadErred
}

// Return true if the workload is actively running/in-progress.
func (w *workloadImpl) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.WorkloadState == WorkloadRunning
}

// Return true if the workload stopped naturally/successfully after processing all events.
func (w *workloadImpl) IsFinished() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.IsErred() || w.DidCompleteSuccessfully()
}

// Return true if the workload stopped naturally/successfully after processing all events.
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
