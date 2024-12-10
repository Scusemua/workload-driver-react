package domain

import (
	"errors"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"time"
)

var (
	ErrWorkloadNotRunning        = errors.New("the workload is currently not running")
	ErrInvalidState              = errors.New("workload is in invalid state for the specified operation")
	ErrWorkloadNotFound          = errors.New("could not find workload with the specified ID")
	ErrWorkloadNotPaused         = errors.New("the workload is currently not paused")
	ErrMissingMaxResourceRequest = errors.New("session does not have a \"max\" resource request")
)

type MaxUtilizationConsumer interface {
	SetMaxUtilizationWrapper(*MaxUtilizationWrapper)
	GetId() string
}

type WorkloadErrorHandler func(workloadId string, err error)

type WorkloadGenerator interface {
	GeneratePresetWorkload(EventConsumer, MaxUtilizationConsumer, *WorkloadPreset, *WorkloadRegistrationRequest) error // Start generating the workload.
	GenerateTemplateWorkload(EventConsumer, []*WorkloadTemplateSession, *WorkloadRegistrationRequest) error            // Start generating the workload.
	StopGeneratingWorkload()                                                                                           // Stop generating the workload prematurely.
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

	// GetState Returns the current state of the workload.
	//GetState() WorkloadState

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

	// SetState Sets the state of the workload.
	// SetState(state WorkloadState)

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
	SessionStopped(string, *Event)
	// TrainingStarted is Called when a training starts during/in the workload.
	// Just updates some internal metrics.
	TrainingStarted(sessionId string, tickNumber int64)
	// TrainingSubmitted when an "execute_request" message is sent.
	TrainingSubmitted(string, *Event)
	// TrainingStopped is Called when a training stops during/in the workload.
	// Just updates some internal metrics.
	TrainingStopped(sessionId string, evt *Event, tickNumber int64)
	// IsPresetWorkload returns true if this workload was created using a preset.
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
	// SetSource Sets the source of the workload, namely a template or a preset.
	SetSource(interface{}) error
	// GetCurrentTick Returns the current tick.
	GetCurrentTick() int64
	// GetSimulationClockTimeStr Returns the simulation clock time.
	GetSimulationClockTimeStr() string
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
	// GetTickDurationsMillis() []int64

	// AddFullTickDuration is called to record how long a tick lasted, including the "artificial" sleep that is performed
	// by the WorkloadDriver in order to fully simulate ticks that otherwise have no work/events to be processed.
	// AddFullTickDuration(timeElapsed time.Duration)

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
	// SessionDelayed should be called when events for a particular Session are delayed for processing, such as
	// due to there being too much resource contention.
	SessionDelayed(string, time.Duration)
	// IsSessionBeingSampled returns true if the specified session was selected for sampling.
	IsSessionBeingSampled(string) bool
	// GetSampleSessionsPercentage returns the configured SampleSessionsPercentage parameter for the Workload.
	GetSampleSessionsPercentage() float64
	// RegisterApproximateFinalTick is used to register what is the approximate final tick of the workload
	// after iterating over all sessions and all training events.
	RegisterApproximateFinalTick(approximateFinalTick int64)
	// GetNextEventTick returns the tick at which the next event is expected to be processed.
	GetNextEventTick() int64
	// SetNextEventTick sets the tick at which the next event is expected to be processed (for visualization purposes).
	SetNextEventTick(int64)
	// SetNextExpectedEventName is used to register, for visualization purposes, the name of the next expected event.
	SetNextExpectedEventName(EventName)
	// SetNextExpectedEventSession is used to register, for visualization purposes, the target session of the next
	// expected event.
	SetNextExpectedEventSession(string)
	// SessionDiscarded is used to record that a particular session is being discarded/not sampled.
	SessionDiscarded(string) error
}
