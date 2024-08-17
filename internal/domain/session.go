package domain

const (
	SessionPending  SessionState = "pending"    // The session has not yet been created.
	SessionIdle     SessionState = "idle"       // The session is running, but is not actively training.
	SessionTraining SessionState = "training"   // The session is actively training.
	SessionStopped  SessionState = "terminated" // The session has been terminated (without an error).
	SessionErred    SessionState = "erred"      // An error occurred, forcing the session to terminate.
)

type SessionState string

// Corresponds to the `Session` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type WorkloadSession struct {
	Id                 string          `json:"id"`
	MaxCPUs            float64         `json:"max_cpus"`
	MaxMemoryGB        float64         `json:"max_memory_gb"`
	MaxNumGPUs         int             `json:"max_num_gpus"`
	StartTick          int             `json:"start_tick"`
	StopTick           int             `json:"stop_tick"`
	Trainings          []TrainingEvent `json:"trainings"`
	NumEventsProcessed int             `json:"num_events_processed"`
	State              SessionState    `json:"state"`
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
