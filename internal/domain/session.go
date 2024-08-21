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
	GetPod() string
	// The maximum number of CPUs that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	GetMaxSessionCPUs() float64
	// The maximum amount of memory (in GB) that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	GetMaxSessionMemory() float64
	// The maximum number of GPUs that this SessionMeta will ever use.
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

	SetState(SessionState)
	GetAndIncrementTrainingsCompleted() int
}

// Corresponds to the `Session` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type BaseWorkloadSession struct {
	Id                 string           `json:"id"`
	ResourceRequest    *ResourceRequest `json:"resource_request"`
	TrainingsCompleted int              `json:"trainings_completed"`
	State              SessionState     `json:"state"`
	CreatedAt          time.Time        `json:"-"`
	Meta               SessionMetadata  `json:"-"`
}

func (s *BaseWorkloadSession) GetAndIncrementTrainingsCompleted() int {
	s.TrainingsCompleted += 1
	return s.TrainingsCompleted
}

func (s *BaseWorkloadSession) GetId() string {
	return s.Id
}

func (s *BaseWorkloadSession) GetResourceRequest() *ResourceRequest {
	return s.ResourceRequest
}

func (s *BaseWorkloadSession) GetTrainingsCompleted() int {
	return s.TrainingsCompleted
}

func (s *BaseWorkloadSession) GetState() SessionState {
	return s.State
}

func (s *BaseWorkloadSession) SetState(state SessionState) {
	s.State = state
}

func (s *BaseWorkloadSession) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func newWorkloadSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time) *BaseWorkloadSession {
	return &BaseWorkloadSession{
		Id:                 id,
		ResourceRequest:    resourceRequest,
		TrainingsCompleted: 0,
		State:              SessionAwaitingStart,
		CreatedAt:          createdAtTime,
		Meta:               meta,
	}
}

func NewWorkloadSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time) WorkloadSession {
	return newWorkloadSession(id, meta, resourceRequest, createdAtTime)
}

type WorkloadTemplateSession struct {
	*BaseWorkloadSession

	StartTick int             `json:"start_tick"`
	StopTick  int             `json:"stop_tick"`
	Trainings []TrainingEvent `json:"trainings"`
}

func NewWorkloadTemplateSession(id string, meta SessionMetadata, resourceRequest *ResourceRequest, createdAtTime time.Time, startTick int, stopTick int) *WorkloadTemplateSession {
	workload_session := newWorkloadSession(id, meta, resourceRequest, createdAtTime)

	return &WorkloadTemplateSession{
		BaseWorkloadSession: workload_session,
		StartTick:           startTick,
		StopTick:            stopTick,
		Trainings:           make([]TrainingEvent, 0),
	}
}

// Corresponds to the `TrainingEvent` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type TrainingEvent struct {
	SessionId       string    `json:"sessionId"`
	TrainingId      string    `json:"trainingId"`
	CpuUtil         float64   `json:"cpuUtil"`
	MemUsageGB      float64   `json:"memUsageGB"`
	GpuUtil         []float64 `json:"gpuUtil"`
	StartTick       int       `json:"startTick"`
	DurationInTicks int       `json:"durationInTicks"`
}

func (e *TrainingEvent) NumGPUs() int {
	return len(e.GpuUtil)
}
