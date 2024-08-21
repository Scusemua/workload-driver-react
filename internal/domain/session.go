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

type SessionState string

// Corresponds to the `Session` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type WorkloadSession struct {
	Id                 string           `json:"id"`
	ResourceRequest    *ResourceRequest `json:"resource_request"`
	TrainingsCompleted int              `json:"trainings_completed"`
	State              SessionState     `json:"state"`
	CreatedAt          time.Time        `json:"-"`
	// Meta               *generator.Session `json:"-"`
}

func NewWorkloadSession(id string, resourceRequest *ResourceRequest, createdAtTime time.Time) *WorkloadSession {
	return &WorkloadSession{
		Id:                 id,
		ResourceRequest:    resourceRequest,
		TrainingsCompleted: 0,
		State:              SessionAwaitingStart,
		CreatedAt:          createdAtTime,
	}
}

type WorkloadTemplateSession struct {
	*WorkloadSession

	StartTick int             `json:"start_tick"`
	StopTick  int             `json:"stop_tick"`
	Trainings []TrainingEvent `json:"trainings"`
}

func NewWorkloadTemplateSession(id string, resourceRequest *ResourceRequest, createdAtTime time.Time, startTick int, stopTick int) *WorkloadTemplateSession {
	workload_session := NewWorkloadSession(id, resourceRequest, createdAtTime)

	return &WorkloadTemplateSession{
		WorkloadSession: workload_session,
		StartTick:       startTick,
		StopTick:        stopTick,
		Trainings:       make([]TrainingEvent, 0),
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
