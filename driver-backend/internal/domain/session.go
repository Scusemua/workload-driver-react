package domain

import (
	"errors"
	"fmt"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

const (
	SessionAwaitingStart SessionState = "awaiting start" // The session has not yet been created.
	SessionIdle          SessionState = "idle"           // The session is running, but is not actively training.
	SessionTraining      SessionState = "training"       // The session is actively training.
	SessionStopped       SessionState = "terminated"     // The session has been terminated (without an error).
	SessionErred         SessionState = "erred"          // An error occurred, forcing the session to terminate.
)

var (
	ErrIllegalStateTransition = errors.New("illegal state transition")
)

type SessionMetadata interface {
	// GetPod returns the pod/session ID of the session.
	GetPod() string

	// GetMaxSessionCPUs returns the maximum number of CPUs that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	GetMaxSessionCPUs() float64

	// GetMaxSessionMemory returns t maximum amount of memory (in GB) that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	GetMaxSessionMemory() float64

	// GetMaxSessionGPUs returns the maximum number of GPUs that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	GetMaxSessionGPUs() int

	// GetMaxSessionVRAM returns the maximum VRAM (i.e. GPU memory) used by the Session at any point in gigabytes (GB).
	GetMaxSessionVRAM() float64

	// GetCurrentTrainingMaxCPUs returns the maximum number of CPUs that this SessionMetadata will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the SessionMetadata is attached as data to a 'training-started' event.
	GetCurrentTrainingMaxCPUs() int

	// GetCurrentTrainingMaxMemory returns the maximum amount of memory (in GB) that this SessionMetadata will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the SessionMetadata is attached as data to a 'training-started' event.
	GetCurrentTrainingMaxMemory() int

	// GetVRAM returns the VRAM.
	GetVRAM() int

	// GetCurrentTrainingMaxGPUs returns the maximum number of GPUs that this SessionMetadata will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the SessionMetadata is attached as data to a 'training-started' event.
	GetCurrentTrainingMaxGPUs() int

	// GetGPUs returns the number of GPUs that this Session is configured to use.
	GetGPUs() int
}

type SessionState string

func (s SessionState) String() string {
	return string(s)
}

type WorkloadSession interface {
	GetId() string
	GetResourceRequest() *ResourceRequest
	GetTrainingsCompleted() int
	GetState() SessionState
	GetCreatedAt() time.Time
	GetTrainingStartedAt() time.Time
	GetTrainings() []*TrainingEvent
	GetStderrIoPubMessages() []string
	GetStdoutIoPubMessages() []string
	AddStderrIoPubMessage(message string)
	AddStdoutIoPubMessage(message string)

	SetState(SessionState) error
	GetAndIncrementTrainingsCompleted() int
}

// BasicWorkloadSession corresponds to the `Session` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type BasicWorkloadSession struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	Id                  string           `json:"id"`
	ResourceRequest     *ResourceRequest `json:"resource_request"`
	TrainingsCompleted  int              `json:"trainings_completed"`
	State               SessionState     `json:"state"`
	CreatedAt           time.Time        `json:"-"`
	TrainingStartedAt   time.Time        `json:"-"`
	Meta                SessionMetadata  `json:"-"`
	TrainingEvents      []*TrainingEvent `json:"trainings"`
	StderrIoPubMessages []string         `json:"stderr_io_pub_messages"`
	StdoutIoPubMessages []string         `json:"stdout_io_pub_messages"`
}

func newWorkloadSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time, atom *zap.AtomicLevel) *BasicWorkloadSession {
	session := &BasicWorkloadSession{
		Id:                  id,
		ResourceRequest:     resourceRequest,
		TrainingsCompleted:  0,
		State:               SessionAwaitingStart,
		CreatedAt:           createdAtTime,
		Meta:                meta,
		TrainingEvents:      make([]*TrainingEvent, 0),
		StderrIoPubMessages: make([]string, 0),
		StdoutIoPubMessages: make([]string, 0),
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	session.logger = logger
	session.sugaredLogger = logger.Sugar()

	return session
}

func NewWorkloadSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time, atom *zap.AtomicLevel) *BasicWorkloadSession {
	return newWorkloadSession(id, meta, resourceRequest, createdAtTime, atom)
}

// createLoggers instantiates the BasicWorkloadSession's zap.Logger and zap.SugaredLogger.
// If the loggers already exist, then createLoggers returns immediately.
func (s *BasicWorkloadSession) createLoggers(atom *zap.AtomicLevel) {
	// If the logger is already non-nil, then it has already been created, and so we'll just return right away.
	if s.logger != nil {
		// If the sugared logger is nil, we'll create it real quick, and then we'll return right away.
		if s.sugaredLogger == nil {
			s.sugaredLogger = s.logger.Sugar()
		}

		return
	}

	// Create the zap.AtomicLevel if the parameter is nil.
	if atom == nil {
		atomStruct := zap.NewAtomicLevelAt(zapcore.DebugLevel)
		atom = &atomStruct
	}

	// Create the session's zap.Logger and zap.SugaredLogger.
	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	s.logger = logger
	s.sugaredLogger = logger.Sugar()
}

// getSugarLogger returns the BasicWorkloadSession's zap.Logger, creating it first (along with the
// BasicWorkloadSession's zap.SugaredLogger) if its zap.Logger is nil.
func (s *BasicWorkloadSession) getLogger() *zap.Logger {
	if s.logger == nil {
		s.createLoggers(nil)
	}

	return s.logger
}

// getSugarLogger returns the BasicWorkloadSession's zap.SugaredLogger, creating it first if it nil.
func (s *BasicWorkloadSession) getSugarLogger() *zap.SugaredLogger {
	if s.sugaredLogger == nil {
		s.createLoggers(nil)
	}

	return s.sugaredLogger
}

// GetAndIncrementTrainingsCompleted increments the BasicWorkloadSession's TrainingsCompleted
// field by 1 and returns the new value (post-increment).
func (s *BasicWorkloadSession) GetAndIncrementTrainingsCompleted() int {
	s.TrainingsCompleted += 1
	return s.TrainingsCompleted
}

func (s *BasicWorkloadSession) GetStderrIoPubMessages() []string {
	return s.StderrIoPubMessages
}

func (s *BasicWorkloadSession) GetStdoutIoPubMessages() []string {
	return s.StdoutIoPubMessages
}

func (s *BasicWorkloadSession) AddStderrIoPubMessage(message string) {
	s.StderrIoPubMessages = append(s.StderrIoPubMessages, message)
}

func (s *BasicWorkloadSession) AddStdoutIoPubMessage(message string) {
	s.StdoutIoPubMessages = append(s.StdoutIoPubMessages, message)
}

func (s *BasicWorkloadSession) GetId() string {
	return s.Id
}

func (s *BasicWorkloadSession) GetResourceRequest() *ResourceRequest {
	return s.ResourceRequest
}

func (s *BasicWorkloadSession) GetTrainingsCompleted() int {
	return s.TrainingsCompleted
}

func (s *BasicWorkloadSession) GetState() SessionState {
	return s.State
}

func (s *BasicWorkloadSession) SetState(targetState SessionState) error {
	if s.State == targetState {
		s.getLogger().Warn("Attempting to transition state of Session into its current state.",
			zap.String("session_id", s.Id), zap.String("state", s.State.String()))
	}

	if s.State == SessionStopped || s.State == SessionErred {
		return fmt.Errorf("%w: cannot transition from targetState '%s' to targetState '%s'; session is no longer running",
			ErrIllegalStateTransition, s.State, targetState)
	}

	sourceState := s.State
	s.getLogger().Debug("Transitioning session now.", zap.String("session_id", s.Id),
		zap.String("source_state", sourceState.String()), zap.String("target_state", targetState.String()))
	s.State = targetState

	if sourceState == SessionTraining {
		s.getLogger().Debug("Session finished training.", zap.String("session_id", s.Id),
			zap.Duration("training_duration", time.Since(s.TrainingStartedAt)))
	}

	if targetState == SessionTraining {
		s.TrainingStartedAt = time.Now()
	}

	return nil
}

// GetTrainingStartedAt returns the time.Time at which the Session last started training.
func (s *BasicWorkloadSession) GetTrainingStartedAt() time.Time {
	return s.TrainingStartedAt
}

func (s *BasicWorkloadSession) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func (s *BasicWorkloadSession) GetTrainings() []*TrainingEvent {
	return s.TrainingEvents
}

// WorkloadTemplateSession are created from Workload templates by deserializing the JSON definition(s) of the
// Sessions included within the Workload template.
//
// They have a few additional fields relative to BasicWorkloadSession structs, namely a StartTick and StopTick
// field, indicating the workload ticks at which the WorkloadTemplateSession is first created and is terminated,
// respectively. WorkloadTemplateSession structs also have a Trainings field, which is a slice of (pointers to)
// TrainingEvent structs, encoding all the training events that are to be performed by the WorkloadTemplateSession
// during the execution/orchestration of the workload.
type WorkloadTemplateSession struct {
	*BasicWorkloadSession

	StartTick int              `json:"start_tick"`
	StopTick  int              `json:"stop_tick"`
	Trainings []*TrainingEvent `json:"trainings"`
}

// NewWorkloadTemplateSession creates a new WorkloadTemplateSession struct and returns a pointer to it.
func NewWorkloadTemplateSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time, startTick int, stopTick int, atom *zap.AtomicLevel) WorkloadTemplateSession {
	workloadSession := newWorkloadSession(id, meta, resourceRequest, createdAtTime, atom)

	return WorkloadTemplateSession{
		BasicWorkloadSession: workloadSession,
		StartTick:            startTick,
		StopTick:             stopTick,
		Trainings:            make([]*TrainingEvent, 0),
	}
}

func (t WorkloadTemplateSession) GetStartTick() int {
	return t.StartTick
}

func (t WorkloadTemplateSession) GetStopTick() int {
	return t.StopTick
}

func (t WorkloadTemplateSession) GetTrainings() []*TrainingEvent {
	return t.Trainings
}

// TrainingEvent corresponds to the `TrainingEvent` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type TrainingEvent struct {
	TrainingIndex   int              `json:"training_index"`
	Millicpus       float64          `json:"millicpus"` // CPU usage in 1/1000th CPU core
	MemUsageMB      float64          `json:"mem_usage_mb"`
	VRamUsageGB     float64          `json:"vram_usage_gb"`
	GpuUtil         []GpuUtilization `json:"gpu_utilizations"`
	StartTick       int              `json:"start_tick"`
	DurationInTicks int              `json:"duration_in_ticks"`
}

// GpuUtilization is a struct here with a Utilization field so it matches the JSON generated by the form in the frontend.
type GpuUtilization struct {
	Utilization float64 `json:"utilization"`
}

func (e *TrainingEvent) NumGPUs() int {
	return len(e.GpuUtil)
}
