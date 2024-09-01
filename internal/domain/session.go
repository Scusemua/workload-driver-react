package domain

import (
	"time"
)

const (
	SessionAwaitingStart SessionState = "awaiting start" // The session has not yet been created.
	SessionIdle          SessionState = "idle"           // The session is running, but is not actively training.
	SessionTraining      SessionState = "training"       // The session is actively training.
	SessionStopped       SessionState = "terminated"     // The session has been terminated (without an error).
	SessionErred         SessionState = "erred"          // An error occurred, forcing the session to terminate.
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
}

type SessionState string

type WorkloadSession interface {
	GetId() string
	GetResourceRequest() *ResourceRequest
	GetTrainingsCompleted() int
	GetState() SessionState
	GetCreatedAt() time.Time
	GetTrainings() []*TrainingEvent
	GetStderrIoPubMessages() []string
	GetStdoutIoPubMessages() []string
	AddStderrIoPubMessage(message string)
	AddStdoutIoPubMessage(message string)

	SetState(SessionState)
	GetAndIncrementTrainingsCompleted() int
}

// BasicWorkloadSession corresponds to the `Session` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type BasicWorkloadSession struct {
	Id                  string           `json:"id"`
	ResourceRequest     *ResourceRequest `json:"resource_request"`
	TrainingsCompleted  int              `json:"trainings_completed"`
	State               SessionState     `json:"state"`
	CreatedAt           time.Time        `json:"-"`
	Meta                SessionMetadata  `json:"-"`
	TrainingEvents      []*TrainingEvent `json:"trainings"`
	StderrIoPubMessages []string         `json:"stderr_io_pub_messages"`
	StdoutIoPubMessages []string         `json:"stdout_io_pub_messages"`
}

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

func (s *BasicWorkloadSession) SetState(state SessionState) {
	s.State = state
}

func (s *BasicWorkloadSession) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func (s *BasicWorkloadSession) GetTrainings() []*TrainingEvent {
	return s.TrainingEvents
}

func newWorkloadSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time) *BasicWorkloadSession {
	return &BasicWorkloadSession{
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
}

func NewWorkloadSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time) *BasicWorkloadSession {
	return newWorkloadSession(id, meta, resourceRequest, createdAtTime)
}

type WorkloadTemplateSession struct {
	*BasicWorkloadSession

	StartTick int              `json:"start_tick"`
	StopTick  int              `json:"stop_tick"`
	Trainings []*TrainingEvent `json:"trainings"`
}

func NewWorkloadTemplateSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time, startTick int, stopTick int) WorkloadTemplateSession {
	workload_session := newWorkloadSession(id, meta, resourceRequest, createdAtTime)

	return WorkloadTemplateSession{
		BasicWorkloadSession: workload_session,
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

// Corresponds to the `TrainingEvent` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type TrainingEvent struct {
	TrainingIndex   int              `json:"training_index"`
	CpuUtil         float64          `json:"cpu_util"`
	MemUsageGB      float64          `json:"mem_usage_gb"`
	GpuUtil         []GpuUtilization `json:"gpu_utilizations"`
	StartTick       int              `json:"start_tick"`
	DurationInTicks int              `json:"duration_in_ticks"`
}

// We use a struct here with a Utilization field so it matches the JSON generated by the form in the frontend.
type GpuUtilization struct {
	Utilization float64 `json:"utilization"`
}

func (e *TrainingEvent) NumGPUs() int {
	return len(e.GpuUtil)
}
