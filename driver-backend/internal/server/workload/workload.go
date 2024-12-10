package workload

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	Ready      State = "WorkloadReady"      // BasicWorkload is registered and ready to be started.
	Running    State = "WorkloadRunning"    // BasicWorkload is actively running/in-progress.
	Pausing    State = "WorkloadPausing"    // BasicWorkload is actively running/in-progress.
	Paused     State = "WorkloadPaused"     // BasicWorkload is actively running/in-progress.
	Finished   State = "WorkloadFinished"   // BasicWorkload stopped naturally/successfully after processing all events.
	Erred      State = "WorkloadErred"      // BasicWorkload stopped due to an error.
	Terminated State = "WorkloadTerminated" // BasicWorkload stopped because it was explicitly terminated early/premature.

	TerminatedEventName string = "workload-terminated"

	UnspecifiedWorkload Kind = "UnspecifiedWorkloadType" // Default value, before it is set.
	PresetWorkload      Kind = "Preset"
	TemplateWorkload    Kind = "Template"
	TraceWorkload       Kind = "WorkloadFromTrace"
)

// Kind defines a type that a workload can have/be.
//
// Workloads can be of several different types, namely 'preset' and 'template' and possibly 'trace'.
// Have not fully committed to making 'trace' a separate type from 'preset'.
//
// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
// have properties that the user can specify and change before submitting the workload for registration.
type Kind string

func (typ Kind) String() string {
	return string(typ)
}

type State string

func (state State) String() string {
	return string(state)
}

// GetWorkloadStateAsString will panic if an invalid workload state is specified.
func GetWorkloadStateAsString(state State) string {
	switch state {
	case Ready:
		{
			return "WorkloadReady"
		}
	case Running:
		{
			return "WorkloadRunning"
		}
	case Finished:
		{
			return "WorkloadFinished"
		}
	case Erred:
		{
			return "WorkloadErred"
		}
	case Terminated:
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

	// Statistics encapsulates a number of important statistics related to the workload.
	Statistics *Statistics `json:"statistics"`

	// TimeCompressTrainingDurations indicates whether the TimescaleAdjustmentFactor should be used to compress
	// (or potentially extend, if the value of TimescaleAdjustmentFactor is > 1) the duration of training events.
	TimeCompressTrainingDurations bool `json:"time_compress_training_durations"`

	// SampledSessions is a map (really, just a set; the values of the map are not used) that keeps track of the
	// sessions that this BasicWorkload is actively sampling and processing from the workload.
	//
	// The likelihood that a Session is selected for sampling is based on the SessionsSamplePercentage field.
	//
	// SampledSessions is a sort of counterpart to the UnsampledSessions field.
	SampledSessions map[string]interface{} `json:"-"`
	// UnsampledSessions keeps track of the Sessions this workload has not selected for sampling/processing.
	//
	// UnsampledSessions is a sort of counterpart to the SampledSessions field.
	UnsampledSessions map[string]interface{} `json:"-"`

	//AggregateSessionDelayMillis int64  `json:"aggregate_session_delay_ms" csv:"aggregate_session_delay_ms"`
	Id   string `json:"id"`
	Name string `json:"name"`

	Seed                      int64   `json:"seed"  csv:"seed"`
	DebugLoggingEnabled       bool    `json:"debug_logging_enabled"`
	TimescaleAdjustmentFactor float64 `json:"timescale_adjustment_factor"`

	ErrorMessage           string `json:"error_message"`
	SimulationClockTimeStr string `json:"simulation_clock_time"`
	WorkloadType           Kind   `json:"workload_type"`

	//TickDurationsMillis    []int64 `json:"tick_durations_milliseconds"`

	// SumTickDurationsMillis is the sum of all tick durations in milliseconds, to make it easier
	// to compute the average tick duration.
	//SumTickDurationsMillis int64 `json:"sum_tick_durations_millis"`

	//SessionsSamplePercentage  float64 `json:"sessions_sample_percentage"`
	//TimeSpentPausedMillis     int64   `json:"time_spent_paused_milliseconds"`
	timeSpentPaused time.Duration
	pauseWaitBegin  time.Time

	// This is basically the child struct.
	// So, if this is a preset workload, then this is the Preset struct.
	// We use this so we can delegate certain method calls to the child/derived struct.
	workloadInstance          internalWorkload
	workloadSource            interface{}
	mu                        sync.RWMutex
	sessionsMap               map[string]interface{} // Internal mapping of session ID to session.
	trainingStartedTimes      map[string]time.Time   // Internal mapping of session ID to the time at which it began training.
	trainingStartedTimesTicks map[string]int64       // Mapping from Session ID to the tick at which it began training.
	seedSet                   bool                   // Flag keeping track of whether we've already set the seed for this workload.
	sessionsSet               bool                   // Flag keeping track of whether we've already set the sessions for this workload.

	// OnError is a callback passed to WorkloadDrivers (via the WorkloadManager).
	// If a critical error occurs during the execution of the workload, then this handler is called.
	onCriticalError domain.WorkloadErrorHandler

	// OnError is a callback passed to WorkloadDrivers (via the WorkloadManager).
	// If a non-critical error occurs during the execution of the workload, then this handler is called.
	onNonCriticalError domain.WorkloadErrorHandler

	RemoteStorageDefinition *proto.RemoteStorageDefinition `json:"remote_storage_definition"`
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
func (w *BasicWorkload) RegisterOnCriticalErrorHandler(handler domain.WorkloadErrorHandler) {
	w.onCriticalError = handler
}

// RegisterOnNonCriticalErrorHandler registers a non-critical error handler for the target workload.
//
// If there is already a non-critical handler error registered for the target workload, then the existing
// non-critical error handler is overwritten.
func (w *BasicWorkload) RegisterOnNonCriticalErrorHandler(handler domain.WorkloadErrorHandler) {
	w.onNonCriticalError = handler
}

// GetTickDurationsMillis returns a slice containing the clock time that elapsed for each tick
// of the workload in order, in milliseconds.
//func (w *BasicWorkload) GetTickDurationsMillis() []int64 {
//	return w.TickDurationsMillis
//}

// SetPausing will set the workload to the pausing state, which means that it is finishing
// the processing of its current tick before halting until being unpaused.
func (w *BasicWorkload) SetPausing() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Statistics.WorkloadState != Running {
		w.logger.Error("Cannot transition workload to 'pausing' state. Workload is not running.",
			zap.String("workload_state", w.Statistics.WorkloadState.String()),
			zap.String("workload_id", w.Id),
			zap.String("workload-state", string(w.Statistics.WorkloadState)))
		return domain.ErrWorkloadNotPaused
	}

	w.Statistics.WorkloadState = Pausing
	return nil
}

// SetPaused will set the workload to the paused state.
func (w *BasicWorkload) SetPaused() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Statistics.WorkloadState != Pausing {
		w.logger.Error("Cannot transition workload to 'paused' state. Workload is not in 'pausing' state.",
			zap.String("workload_state", w.Statistics.WorkloadState.String()),
			zap.String("workload_id", w.Id),
			zap.String("workload-state", string(w.Statistics.WorkloadState)))
		return domain.ErrWorkloadNotPaused
	}

	w.Statistics.WorkloadState = Paused
	return nil
}

// Unpause will set the workload to the unpaused state.
func (w *BasicWorkload) Unpause() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Statistics.WorkloadState != Paused && w.Statistics.WorkloadState != Pausing {
		w.logger.Error("Cannot unpause workload. Workload is not paused.",
			zap.String("workload_state", w.Statistics.WorkloadState.String()),
			zap.String("workload_id", w.Id),
			zap.String("workload-state", string(w.Statistics.WorkloadState)))
		return domain.ErrWorkloadNotPaused
	}

	w.Statistics.WorkloadState = Running

	// pauseWaitBegin is set to zero after being processed.
	// So, if it is currently zero, then we're not paused, and we should do nothing.
	if w.pauseWaitBegin.IsZero() {
		return nil
	}

	// Compute how long we were paused, increment the counters, and then zero out the pauseWaitBegin field.
	pauseDuration := time.Since(w.pauseWaitBegin)
	w.timeSpentPaused += pauseDuration
	w.Statistics.TimeSpentPausedMillis = w.timeSpentPaused.Milliseconds()

	w.pauseWaitBegin = time.Time{} // Zero it out.
	return nil
}

// AddFullTickDuration is called to record how long a tick lasted, including the "artificial" sleep that is performed
// by the WorkloadDriver in order to fully simulate ticks that otherwise have no work/events to be processed.
//func (w *BasicWorkload) AddFullTickDuration(timeElapsed time.Duration) {
//	timeElapsedMs := timeElapsed.Milliseconds()
//	w.TickDurationsMillis = append(w.TickDurationsMillis, timeElapsedMs)
//	w.SumTickDurationsMillis += timeElapsedMs
//}

// GetCurrentTick returns the current tick.
func (w *BasicWorkload) GetCurrentTick() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.CurrentTick
}

// GetSimulationClockTimeStr returns the simulation clock time.
func (w *BasicWorkload) GetSimulationClockTimeStr() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.SimulationClockTimeStr
}

// SetSource sets the source of the workload, namely a template or a preset.
// This defers the execution of the method to the `BasicWorkload::workload` field.
func (w *BasicWorkload) SetSource(source interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.workloadInstance.unsafeSetSource(source)
}

// GetKind gets the type of workload (TRACE, PRESET, or TEMPLATE).
func (w *BasicWorkload) GetKind() Kind {
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
func (w *BasicWorkload) GetProcessedEvents() []*domain.WorkloadEvent {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.EventsProcessed
}

// TerminateWorkloadPrematurely stops the workload.
func (w *BasicWorkload) TerminateWorkloadPrematurely(simulationTimestamp time.Time) (time.Time, error) {
	if !w.IsInProgress() {
		w.logger.Error("Cannot stop as I am not running.", zap.String("workload_id", w.Id), zap.String("workload-state", string(w.Statistics.WorkloadState)))
		return time.Now(), domain.ErrWorkloadNotRunning
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()

	w.Statistics.EndTime = now
	w.Statistics.WorkloadState = Terminated
	w.Statistics.NumEventsProcessed += 1

	// workloadEvent := NewWorkloadEvent(len(w.Statistics.EventsProcessed), uuid.NewString(), "workload-terminated", "N/A", simulationTimestamp.String(), now.String(), true, nil)
	workloadEvent := domain.NewEmptyWorkloadEvent().
		WithIndex(len(w.Statistics.EventsProcessed)).
		WithEventId(uuid.NewString()).
		WithEventNameString(TerminatedEventName).
		WithSessionId("N/A").
		WithEventTimestamp(simulationTimestamp).
		WithProcessedAtTime(now).
		WithSimProcessedAtTime(simulationTimestamp).
		WithProcessedStatus(true)

	w.Statistics.EventsProcessed = append(w.Statistics.EventsProcessed, workloadEvent)

	// w.Statistics.EventsProcessed = append(w.Statistics.EventsProcessed, &WorkloadEvent{
	// 	Index:                 len(w.Statistics.EventsProcessed),
	// 	Id:                    uuid.NewString(),
	// 	Name:                  "workload-terminated",
	// 	Session:               "N/A",
	// 	Timestamp:             simulationTimestamp.String(),
	// 	ProcessedAt:           now.String(),
	// 	ProcessedSuccessfully: true,
	// })

	w.logger.Debug("Stopped.", zap.String("workload_id", w.Id))
	return w.Statistics.EndTime, nil
}

// StartWorkload starts the Workload.
//
// If the workload is already running, then an error is returned.
// Likewise, if the workload was previously running but has already stopped, then an error is returned.
func (w *BasicWorkload) StartWorkload() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Statistics.WorkloadState != Ready {
		return fmt.Errorf("%w: cannot start workload that is in state '%s'", domain.ErrInvalidState, GetWorkloadStateAsString(w.Statistics.WorkloadState))
	}

	w.Statistics.WorkloadState = Running
	w.Statistics.StartTime = time.Now()

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

	w.Statistics.WorkloadState = Finished
	w.Statistics.EndTime = time.Now()
	w.Statistics.WorkloadDuration = time.Since(w.Statistics.StartTime)
}

// GetErrorMessage gets the error message associated with the workload.
// If the workload is not in an ERROR state, then this returns the empty string and false.
// If the workload is in an ERROR state, then the boolean returned will be true.
func (w *BasicWorkload) GetErrorMessage() (string, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Statistics.WorkloadState == Erred {
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

// GetState returns the current state of the workload.
func (w *BasicWorkload) GetState() State {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Statistics.WorkloadState
}

// SetState sets the state of the workload.
func (w *BasicWorkload) SetState(state State) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Statistics.WorkloadState = state
}

// GetStartTime returns the time that the workload was started.
func (w *BasicWorkload) GetStartTime() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Statistics.StartTime
}

// GetEndTime returns the time at which the workload finished.
// If the workload hasn't finished yet, the returned boolean will be false.
// If the workload has finished, then the returned boolean will be true.
func (w *BasicWorkload) GetEndTime() (time.Time, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Statistics.WorkloadState == Erred || w.Statistics.WorkloadState == Finished {
		return w.Statistics.EndTime, true
	}

	return time.Time{}, false
}

// GetRegisteredTime returns the time that the workload was registered.
func (w *BasicWorkload) GetRegisteredTime() time.Time {
	return w.Statistics.RegisteredTime
}

// GetTimeElapsed returns the time elapsed, which is computed at the time that data is requested by the user.
func (w *BasicWorkload) GetTimeElapsed() time.Duration {
	return w.Statistics.TimeElapsed
}

// GetTimeElapsedAsString returns the time elapsed as a string, which is computed at the time that data is requested by the user.
//
// IMPORTANT: This updates the w.Statistics.TimeElapsedStr field (setting it to w.Statistics.TimeElapsed.String()) before returning it.
func (w *BasicWorkload) GetTimeElapsedAsString() string {
	w.Statistics.TimeElapsedStr = w.Statistics.TimeElapsed.String()
	return w.Statistics.TimeElapsed.String()
}

// SetTimeElapsed updates the time elapsed.
func (w *BasicWorkload) SetTimeElapsed(timeElapsed time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Statistics.TimeElapsed = timeElapsed
	w.Statistics.TimeElapsedStr = w.Statistics.TimeElapsed.String()
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
	timeElapsed := time.Since(w.Statistics.StartTime)

	// Second, subtract the time we have spent paused.
	w.Statistics.TimeElapsed = timeElapsed - w.timeSpentPaused
	w.Statistics.TimeElapsedStr = w.Statistics.TimeElapsed.String()
}

// SessionDelayed should be called when events for a particular Session are delayed for processing, such as
// due to there being too much resource contention.
//
// Multiple calls to SessionDelayed will treat each passed delay additively, as in they'll all be added together.
func (w *BasicWorkload) SessionDelayed(sessionId string, delayAmount time.Duration) {
	w.Statistics.AggregateSessionDelayMillis += delayAmount.Milliseconds()

	w.workloadInstance.SessionDelayed(sessionId, delayAmount)
}

// GetNumEventsProcessed returns the number of events processed by the workload.
func (w *BasicWorkload) GetNumEventsProcessed() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.NumEventsProcessed
}

// WorkloadName returns the name of the workload.
// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
func (w *BasicWorkload) WorkloadName() string {
	return w.Name
}

// ProcessedEvent is called after an event is processed for the Workload.
// Just updates some internal metrics.
//
// This method is thread safe.
func (w *BasicWorkload) ProcessedEvent(evt *domain.WorkloadEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if evt == nil {
		w.logger.Error("Workload event that was supposedly processed is nil.",
			zap.String("workload_id", w.Id),
			zap.String("workload_name", w.Name))
		return
	}

	w.Statistics.NumEventsProcessed += 1
	evt.Index = len(w.Statistics.EventsProcessed)
	w.Statistics.EventsProcessed = append(w.Statistics.EventsProcessed, evt)

	if metrics.PrometheusMetricsWrapperInstance != nil && metrics.PrometheusMetricsWrapperInstance.WorkloadEventsProcessed != nil {
		metrics.PrometheusMetricsWrapperInstance.WorkloadEventsProcessed.
			With(prometheus.Labels{"workload_id": w.Id}).
			Add(1)
	}

	w.logger.Debug("Processed workload event.",
		zap.String("workload_id", w.Id),
		zap.String("workload_name", w.Name),
		zap.String("event_id", evt.Id),
		zap.String("event_name", evt.Name),
		zap.String("session_id", evt.Session))
}

// SessionCreated is called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *BasicWorkload) SessionCreated(sessionId string, metadata domain.SessionMetadata) {
	w.mu.Lock()
	w.Statistics.NumActiveSessions += 1
	w.Statistics.NumSessionsCreated += 1
	w.mu.Unlock()

	metrics.PrometheusMetricsWrapperInstance.WorkloadTotalNumSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveNumSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)

	w.workloadInstance.SessionCreated(sessionId, metadata)
}

// SessionStopped is called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *BasicWorkload) SessionStopped(sessionId string, _ *domain.Event) {
	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveNumSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Sub(1)

	w.mu.Lock()
	defer w.mu.Unlock()
	w.Statistics.NumActiveSessions -= 1

	val, ok := w.sessionsMap[sessionId]
	if !ok {
		w.logger.Error("Failed to find freshly-terminated session in session map.", zap.String("session_id", sessionId))
		return
	}

	session := val.(*domain.WorkloadTemplateSession)
	if err := session.SetState(domain.SessionStopped); err != nil {
		w.logger.Error("Failed to set session state.", zap.String("session_id", sessionId), zap.Error(err))
	}

	session.SetCurrentResourceRequest(&domain.ResourceRequest{
		VRAM:     0,
		Cpus:     0,
		MemoryMB: 0,
		Gpus:     0,
	})
}

// TrainingSubmitted when an "execute_request" message is sent.
func (w *BasicWorkload) TrainingSubmitted(sessionId string, evt *domain.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Statistics.NumSubmittedTrainings += 1

	val, ok := w.sessionsMap[sessionId]
	if !ok {
		w.logger.Error("Failed to find now-training session in session map.", zap.String("session_id", sessionId))
		return
	}

	session := val.(*domain.WorkloadTemplateSession)
	if err := session.SetState(domain.SessionTraining); err != nil {
		w.logger.Error("Failed to set session state.", zap.String("session_id", sessionId), zap.Error(err))
	}

	eventData := evt.Data
	sessionMetadata, ok := eventData.(domain.SessionMetadata)

	if !ok {
		w.logger.Error("Could not extract SessionMetadata from event.",
			zap.String("event_id", evt.Id()),
			zap.String("event_name", evt.Name.String()),
			zap.String("session_id", sessionId))
		return
	}

	session.SetCurrentResourceRequest(&domain.ResourceRequest{
		VRAM:     sessionMetadata.GetCurrentTrainingMaxVRAM(),
		Cpus:     sessionMetadata.GetCurrentTrainingMaxCPUs(),
		MemoryMB: sessionMetadata.GetCurrentTrainingMaxMemory(),
		Gpus:     sessionMetadata.GetCurrentTrainingMaxGPUs(),
	})
}

// TrainingStarted is called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *BasicWorkload) TrainingStarted(sessionId string, tickNumber int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Statistics.NumSubmittedTrainings -= 1
	w.Statistics.NumActiveTrainings += 1

	w.trainingStartedTimes[sessionId] = time.Now()
	w.trainingStartedTimesTicks[sessionId] = tickNumber

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveTrainingSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)
}

// TrainingStopped is called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *BasicWorkload) TrainingStopped(sessionId string, evt *domain.Event, tickNumber int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	metrics.PrometheusMetricsWrapperInstance.WorkloadTrainingEventsCompleted.
		With(prometheus.Labels{"workload_id": w.Id}).
		Add(1)

	metrics.PrometheusMetricsWrapperInstance.WorkloadActiveTrainingSessions.
		With(prometheus.Labels{"workload_id": w.Id}).
		Sub(1)

	trainingStartedAt, loaded := w.trainingStartedTimes[sessionId]
	if !loaded {
		w.logger.Error("Could not load 'training-started' time for Session upon training stopping.",
			zap.String("session_id", sessionId),
			zap.String("workload_id", w.Id),
			zap.String("workload_name", w.WorkloadName()))
	} else {
		trainingDuration := time.Since(trainingStartedAt)

		metrics.PrometheusMetricsWrapperInstance.WorkloadTrainingEventDurationMilliseconds.
			With(prometheus.Labels{"workload_id": w.Id, "session_id": sessionId}).
			Observe(float64(trainingDuration.Milliseconds()))
	}

	trainingStartedAtTick, loaded := w.trainingStartedTimesTicks[sessionId]
	if !loaded {
		w.logger.Error("Could not load 'training-started' tick number for Session upon training stopping.",
			zap.String("session_id", sessionId),
			zap.String("workload_id", w.Id),
			zap.String("workload_name", w.WorkloadName()))
	} else {
		trainingDurationInTicks := tickNumber - trainingStartedAtTick
		w.Statistics.CumulativeTrainingTimeTicks += trainingDurationInTicks
	}

	w.Statistics.NumTasksExecuted += 1
	w.Statistics.NumActiveTrainings -= 1

	val, ok := w.sessionsMap[sessionId]

	if !ok {
		w.logger.Error("Failed to find now-idle session in session map.", zap.String("session_id", sessionId))
		return
	}

	session := val.(*domain.WorkloadTemplateSession)
	if err := session.SetState(domain.SessionIdle); err != nil {
		w.logger.Error("Failed to set session state.", zap.String("session_id", sessionId), zap.Error(err))
	}
	session.GetAndIncrementTrainingsCompleted()

	eventData := evt.Data
	sessionMetadata, ok := eventData.(domain.SessionMetadata)

	if !ok {
		w.logger.Error("Could not extract SessionMetadata from event.",
			zap.String("event_id", evt.Id()),
			zap.String("event_name", evt.Name.String()),
			zap.String("session_id", sessionId))
		return
	}

	session.SetCurrentResourceRequest(&domain.ResourceRequest{
		VRAM:     sessionMetadata.GetVRAM(),
		Cpus:     sessionMetadata.GetCpuUtilization(),
		MemoryMB: sessionMetadata.GetMemoryUtilization(),
		Gpus:     sessionMetadata.GetNumGPUs(),
	})
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
	return w.Statistics.WorkloadState == Terminated
}

// IsReady returns true if the workload is registered and ready to be started.
func (w *BasicWorkload) IsReady() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.WorkloadState == Ready
}

// IsErred returns true if the workload stopped due to an error.
func (w *BasicWorkload) IsErred() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.WorkloadState == Erred
}

// IsRunning returns true if the workload is actively running (i.e., not paused).
func (w *BasicWorkload) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.WorkloadState == Running
}

// IsPausing returns true if the workload is pausing, meaning that it is finishing the processing
// of its current tick before halting until it is un-paused.
func (w *BasicWorkload) IsPausing() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.WorkloadState == Pausing
}

// IsPaused returns true if the workload is paused.
func (w *BasicWorkload) IsPaused() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.WorkloadState == Paused
}

// IsInProgress returns true if the workload is actively running, pausing, or paused.
func (w *BasicWorkload) IsInProgress() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.Statistics.WorkloadState == Running || w.Statistics.WorkloadState == Pausing || w.Statistics.WorkloadState == Paused
}

// IsFinished returns true if the workload stopped naturally/successfully after processing all events.
func (w *BasicWorkload) IsFinished() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.Statistics.WorkloadState == Erred || w.Statistics.WorkloadState == Finished
}

// DidCompleteSuccessfully returns true if the workload stopped naturally/successfully
// after processing all events.
func (w *BasicWorkload) DidCompleteSuccessfully() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Statistics.WorkloadState == Finished
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

// GetSampleSessionsPercentage returns the configured SampleSessionsPercentage parameter for the Workload.
func (w *BasicWorkload) GetSampleSessionsPercentage() float64 {
	return w.Statistics.SessionsSamplePercentage
}

// RegisterApproximateFinalTick is used to register what is the approximate final tick of the workload
// after iterating over all sessions and all training events.
func (w *BasicWorkload) RegisterApproximateFinalTick(approximateFinalTick int64) {
	w.Statistics.TotalNumTicks = approximateFinalTick
}

// GetNextEventTick returns the tick at which the next event is expected to be processed.
func (w *BasicWorkload) GetNextEventTick() int64 {
	return w.Statistics.NextEventExpectedTick
}

// SetNextEventTick sets the tick at which the next event is expected to be processed (for visualization purposes).
func (w *BasicWorkload) SetNextEventTick(nextEventExpectedTick int64) {
	w.Statistics.NextEventExpectedTick = nextEventExpectedTick
}

// SessionDiscarded is used to record that a particular session is being discarded/not sampled.
func (w *BasicWorkload) SessionDiscarded(sessionId string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.unsafeSessionDiscarded(sessionId)
}

func (w *BasicWorkload) unsafeSessionDiscarded(sessionId string) error {
	return w.workloadInstance.SessionDiscarded(sessionId)
}

func (w *BasicWorkload) unsafeSetSessionSampled(sessionId string) {
	w.SampledSessions[sessionId] = struct{}{}
	w.logger.Debug("Decided to sample events targeting session.",
		zap.String("workload_id", w.Id),
		zap.String("workload_name", w.Name),
		zap.String("session_id", sessionId),
		zap.Int("num_sampled_sessions", len(w.SampledSessions)),
		zap.Int("num_discarded_sessions", len(w.UnsampledSessions)))
	w.Statistics.NumSampledSessions += 1
}

func (w *BasicWorkload) unsafeSetSessionDiscarded(sessionId string) {
	err := w.SessionDiscarded(sessionId)
	if err != nil {
		w.logger.Error("Failed to disable session.",
			zap.String("workload_id", w.Id),
			zap.String("workload_name", w.Name),
			zap.Int("num_sampled_sessions", len(w.SampledSessions)),
			zap.Int("num_discarded_sessions", len(w.UnsampledSessions)),
			zap.String("session_id", sessionId),
			zap.Error(err))
	}

	w.UnsampledSessions[sessionId] = struct{}{}
	w.logger.Debug("Decided to discard events targeting session.",
		zap.String("session_id", sessionId),
		zap.Int("num_sampled_sessions", len(w.SampledSessions)),
		zap.Int("num_discarded_sessions", len(w.UnsampledSessions)))
}

// IsSessionBeingSampled returns true if the specified session was selected for sampling.
//
// If a decision has not yet been made for the Session, then we make a decision before returning a verdict.
//
// For workloads created from a template, this is decided when the workload is created, as all the sessions
// are already known at that point.
//
// For workloads created from a preset, it is decided as the workload runs (as the sessions are generated as
// the preset data is being processed).
func (w *BasicWorkload) IsSessionBeingSampled(sessionId string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.unsafeIsSessionBeingSampled(sessionId)
}

func (w *BasicWorkload) unsafeIsSessionBeingSampled(sessionId string) bool {
	// Check if we've already decided to discard events for this session.
	_, discarded := w.UnsampledSessions[sessionId]
	if discarded {
		return false
	}

	// Check if we've already decided to process events for this session.
	_, sampled := w.SampledSessions[sessionId]
	if sampled {
		return true
	}

	// Randomly decide if we're going to sample/process [events for] this session or not.
	randomValue := rand.Float64()
	if randomValue <= w.Statistics.SessionsSamplePercentage {
		w.unsafeSetSessionSampled(sessionId)
		return true
	}

	w.unsafeSetSessionDiscarded(sessionId)
	return false
}

// SetNextExpectedEventName is used to register, for visualization purposes, the name of the next expected event.
func (w *BasicWorkload) SetNextExpectedEventName(name domain.EventName) {
	w.Statistics.NextExpectedEventName = name
}

// SetNextExpectedEventSession is used to register, for visualization purposes, the target session of the next
// expected event.
func (w *BasicWorkload) SetNextExpectedEventSession(sessionId string) {
	w.Statistics.NextExpectedEventTarget = sessionId
}

// GetStatistics returns the Statistics struct of the internalWorkload.
func (w *BasicWorkload) GetStatistics() *Statistics {
	return w.Statistics
}

// UpdateStatistics provides an atomic mechanism to update the internalWorkload's Statistics.
func (w *BasicWorkload) UpdateStatistics(f func(stats *Statistics)) {
	w.mu.Lock()
	defer w.mu.Unlock()

	f(w.Statistics)
}

// RecordSessionExecutionTime records that the specified session trained for the specified amount of time.
func (w *BasicWorkload) RecordSessionExecutionTime(sessionId string, execTimeMillis int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	val, ok := w.sessionsMap[sessionId]
	if !ok {
		w.logger.Error("Could not find specified session. Cannot record execution time.",
			zap.String("workload_id", w.Id),
			zap.String("workload_name", w.Name),
			zap.String("session_id", sessionId),
			zap.Int64("execution_time_millis", execTimeMillis))
		return
	}

	session := val.(*domain.WorkloadTemplateSession)
	if session.ExecutionTimes == nil {
		session.ExecutionTimes = make([]int64, 0, 1)
	}

	session.ExecutionTimes = append(session.ExecutionTimes, execTimeMillis)
}

func (w *BasicWorkload) GetSessionTrainingEvent(sessionId string, trainingIndex int) *domain.TrainingEvent {
	return w.workloadInstance.getSessionTrainingEvent(sessionId, trainingIndex)
}

// ShouldTimeCompressTrainingDurations returns the value of the TimeCompressTrainingDurations flag.
//
// TimeCompressTrainingDurations indicates whether the TimescaleAdjustmentFactor should be used to compress
// (or potentially extend, if the value of TimescaleAdjustmentFactor is > 1) the duration of training events.
func (w *BasicWorkload) ShouldTimeCompressTrainingDurations() bool {
	return w.TimeCompressTrainingDurations
}
