package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	WorkloadReady      WorkloadState = iota // workloadImpl is registered and ready to be started.
	WorkloadRunning    WorkloadState = 1    // workloadImpl is actively running/in-progress.
	WorkloadFinished   WorkloadState = 2    // workloadImpl stopped naturally/successfully after processing all events.
	WorkloadErred      WorkloadState = 3    // workloadImpl stopped due to an error.
	WorkloadTerminated WorkloadState = 4    // workloadImpl stopped because it was explicitly terminated early/premature.

	UnspecifiedWorkload WorkloadType = "UnspecifiedWorkloadType" // Default value, before it is set.
	PresetWorkload      WorkloadType = "WorkloadFromPreset"
	TemplateWorkload    WorkloadType = "WorkloadFromTemplate"
	TraceWorkload       WorkloadType = "WorkloadFromTrace"
)

var (
	ErrInvalidState = errors.New("workload is in invalid state for the specified operation")
)

type WorkloadState int

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
	StartWorkload() error
	// Mark the workload as having completed successfully.
	SetWorkloadCompleted()
	// Called after an event is processed for the Workload.
	// Just updates some internal metrics.
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
	GetSessions() []*WorkloadSession
	// Set the sessions that will be involved in this workload.
	//
	// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
	SetSessions([]*WorkloadSession)
	// Set the source of the workload, namely a template or a preset.
	SetSource(interface{})
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
	Id          string `json:"id"`
	Name        string `json:"name"`
	Session     string `json:"session"`
	Timestamp   string `json:"timestamp"`
	ProcessedAt string `json:"processed_at"`
}

type workloadImpl struct {
	logger        *zap.Logger        `json:"-"`
	sugaredLogger *zap.SugaredLogger `json:"-"`
	atom          *zap.AtomicLevel   `json:"-"`

	Id                        string             `json:"id"`
	Name                      string             `json:"name"`
	WorkloadState             WorkloadState      `json:"workload_state"`
	DebugLoggingEnabled       bool               `json:"debug_logging_enabled"`
	ErrorMessage              string             `json:"error_message"`
	EventsProcessed           []*WorkloadEvent   `json:"events_processed"`
	Sessions                  []*WorkloadSession `json:"sessions"`
	Seed                      int64              `json:"seed"`
	RegisteredTime            time.Time          `json:"registered_time"`
	StartTime                 time.Time          `json:"start_time"`
	EndTime                   time.Time          `json:"end_time"`
	WorkloadDuration          time.Duration      `json:"workload_duration"` // The total time that the workload executed for. This is only set once the workload has completed.
	TimeElasped               time.Duration      `json:"time_elapsed"`      // Computed at the time that the data is requested by the user. This is the time elapsed SO far.
	TimeElaspedStr            string             `json:"time_elapsed_str"`
	NumTasksExecuted          int64              `json:"num_tasks_executed"`
	NumEventsProcessed        int64              `json:"num_events_processed"`
	NumSessionsCreated        int64              `json:"num_sessions_created"`
	NumActiveSessions         int64              `json:"num_active_sessions"`
	NumActiveTrainings        int64              `json:"num_active_trainings"`
	TimescaleAdjustmentFactor float64            `json:"timescale_adjustment_factor"`
	WorkloadType              WorkloadType       `json:"workload_type"`

	// workloadSource interface{} `json:"-"`

	// This is basically the child struct.
	// So, if this is a preset workload, then this is the WorkloadFromPreset struct.
	// We use this so we can delegate certain method calls to the child/derived struct.
	workload       Workload    `json:"-"`
	workloadSource interface{} `json:"-"`

	sessionsMap *hashmap.HashMap `json:"-"` // Internal mapping of session ID to session.
	seedSet     bool             `json:"-"` // Flag keeping track of whether we've already set the seed for this workload.
	sessionsSet bool             `json:"-"` // Flag keeping track of whether we've already set the sessions for this workload.
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
		Sessions:                  make([]*WorkloadSession, 0), // For template workloads, this will be overwritten.
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

// Set the sessions that will be involved in this workload.
//
// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
func (w *workloadImpl) SetSessions(sessions []*WorkloadSession) {
	w.Sessions = sessions
	w.sessionsSet = true

	// Add each session to our internal mapping and initialize the session.
	for _, session := range sessions {
		session.State = SessionAwaitingStart

		w.sessionsMap.Set()
		w.sessionsMap.Set(session.Id, session)
	}
}

// Set the source of the workload, namely a template or a preset.
// This defers the execution of the method to the `workloadImpl::workload` field.
func (w *workloadImpl) SetSource(source interface{}) {
	w.workload.SetSource(source)
}

// Return the sessions involved in this workload.
func (w *workloadImpl) GetSessions() []*WorkloadSession {
	return w.Sessions
}

// Get the type of workload (TRACE, PRESET, or TEMPLATE).
func (w *workloadImpl) GetWorkloadType() WorkloadType {
	return w.WorkloadType
}

// Return true if this workload was created using a preset.
func (w *workloadImpl) IsPresetWorkload() bool {
	return w.WorkloadType == PresetWorkload
}

// Return true if this workload was created using a template.
func (w *workloadImpl) IsTemplateWorkload() bool {
	return w.WorkloadType == TemplateWorkload
}

// Return true if this workload was created using the trace data.
func (w *workloadImpl) IsTraceWorkload() bool {
	return w.WorkloadType == TraceWorkload
}

// If this is a preset workload, return the name of the preset.
// If this is a trace workload, return the trace information.
// If this is a template workload, return the template information.
func (w *workloadImpl) GetWorkloadSource() interface{} {
	return w.workload.GetWorkloadSource()
}

// Return the events processed during this workload (so far).
func (w *workloadImpl) GetProcessedEvents() []*WorkloadEvent {
	return w.EventsProcessed
}

func (w *workloadImpl) StartWorkload() error {
	if w.WorkloadState != WorkloadReady {
		return fmt.Errorf("%w: cannot start workload that is in state '%s'", ErrInvalidState, GetWorkloadStateAsString(w.WorkloadState))
	}

	w.WorkloadState = WorkloadRunning
	w.StartTime = time.Now()

	return nil
}

func (w *workloadImpl) GetTimescaleAdjustmentFactor() float64 {
	return w.TimescaleAdjustmentFactor
}

// Mark the workload as having completed successfully.
func (w *workloadImpl) SetWorkloadCompleted() {
	w.WorkloadState = WorkloadFinished
	w.EndTime = time.Now()
	w.WorkloadDuration = time.Since(w.StartTime)
}

// Get the error message associated with the workload.
// If the workload is not in an ERROR state, then this returns the empty string and false.
// If the workload is in an ERROR state, then the boolean returned will be true.
func (w *workloadImpl) GetErrorMessage() (string, bool) {
	if w.WorkloadState == WorkloadErred {
		return w.ErrorMessage, true
	}

	return "", false
}

// Set the error message for the workload.
func (w *workloadImpl) SetErrorMessage(errorMessage string) {
	w.ErrorMessage = errorMessage
}

// Return a flag indicating whether debug logging is enabled.
func (w *workloadImpl) IsDebugLoggingEnabled() bool {
	return w.DebugLoggingEnabled
}

// Enable or disable debug logging for the workload.
func (w *workloadImpl) SetDebugLoggingEnabled(enabled bool) {
	w.DebugLoggingEnabled = enabled
}

// Set the workload's seed. Can only be performed once. If attempted again, this will panic.
func (w *workloadImpl) SetSeed(seed int64) {
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
	return w.WorkloadState
}

// Set the state of the workload.
func (w *workloadImpl) SetWorkloadState(state WorkloadState) {
	w.WorkloadState = state
}

// Return the time that the workload was started.
func (w *workloadImpl) GetStartTime() time.Time {
	return w.StartTime
}

// Get the time at which the workload finished.
// If the workload hasn't finished yet, the returned boolean will be false.
// If the workload has finished, then the returned boolean will be true.
func (w *workloadImpl) GetEndTime() (time.Time, bool) {
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
	w.TimeElasped = timeElapsed
	w.TimeElaspedStr = w.TimeElasped.String()
}

// Instruct the Workload to recompute its 'time elapsed' field.
func (w *workloadImpl) UpdateTimeElapsed() {
	w.TimeElasped = time.Since(w.StartTime)
	w.TimeElaspedStr = w.TimeElasped.String()
}

// Return the number of events processed by the workload.
func (w *workloadImpl) GetNumEventsProcessed() int64 {
	return w.NumEventsProcessed
}

// Return the name of the workload.
// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
func (w *workloadImpl) WorkloadName() string {
	return w.Name
}

// Called after an event is processed for the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) ProcessedEvent(evt *WorkloadEvent) {
	w.NumEventsProcessed += 1
	w.EventsProcessed = append(w.EventsProcessed, evt)

	w.sugaredLogger.Debugf("Workload %s processed event '%s' targeting session '%s'", w.Name, evt.Name, evt.Session)
}

// Called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) SessionCreated(sessionId string) {
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1

	// Offload to `workload` field for session-specific steps/updates to session metrics.
	w.workload.SessionCreated(sessionId)

	// val, ok := w.sessionsMap.Get(sessionId)
	// if !ok {
	// 	w.logger.Error("Failed to find newly-created session in session map.", zap.String("session-id", sessionId))
	// 	return
	// }

	// session := val.(*WorkloadSession)
	// session.State = SessionIdle
}

// Called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) SessionStopped(sessionId string) {
	w.NumActiveSessions -= 1

	// Offload to `workload` field for session-specific steps/updates to session metrics.
	w.workload.SessionStopped(sessionId)

	// val, ok := w.sessionsMap.Get(sessionId)
	// if !ok {
	// 	w.logger.Error("Failed to find freshly-terminated session in session map.", zap.String("session-id", sessionId))
	// 	return
	// }

	// session := val.(*WorkloadSession)
	// session.State = SessionStopped
}

// Called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStarted(sessionId string) {
	w.NumActiveTrainings += 1

	// Offload to `workload` field for session-specific steps/updates to session metrics.
	w.workload.TrainingStarted(sessionId)

	// val, ok := w.sessionsMap.Get(sessionId)
	// if !ok {
	// 	w.logger.Error("Failed to find now-training session in session map.", zap.String("session-id", sessionId))
	// 	return
	// }

	// session := val.(*WorkloadSession)
	// session.State = SessionTraining
}

// Called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStopped(sessionId string) {
	w.NumTasksExecuted += 1
	w.NumActiveTrainings -= 1

	// Offload to `workload` field for session-specific steps/updates to session metrics.
	w.workload.TrainingStopped(sessionId)

	// val, ok := w.sessionsMap.Get(sessionId)
	// if !ok {
	// 	w.logger.Error("Failed to find now-idle session in session map.", zap.String("session-id", sessionId))
	// 	return
	// }

	// session := val.(*WorkloadSession)
	// session.State = SessionIdle
}

// Return the unique ID of the workload.
func (w *workloadImpl) GetId() string {
	return w.Id
}

// Return true if the workload stopped because it was explicitly terminated early/premature.
func (w *workloadImpl) IsTerminated() bool {
	return w.WorkloadState == WorkloadTerminated
}

// Return true if the workload is registered and ready to be started.
func (w *workloadImpl) IsReady() bool {
	return w.WorkloadState == WorkloadReady
}

// Return true if the workload stopped due to an error.
func (w *workloadImpl) IsErred() bool {
	return w.WorkloadState == WorkloadErred
}

// Return true if the workload is actively running/in-progress.
func (w *workloadImpl) IsRunning() bool {
	return w.WorkloadState == WorkloadRunning
}

// Return true if the workload stopped naturally/successfully after processing all events.
func (w *workloadImpl) IsFinished() bool {
	return w.IsErred() || w.DidCompleteSuccessfully()
}

// Return true if the workload stopped naturally/successfully after processing all events.
func (w *workloadImpl) DidCompleteSuccessfully() bool {
	return w.WorkloadState == WorkloadFinished
}

func (w *workloadImpl) String() string {
	out, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}

	return string(out)
}
