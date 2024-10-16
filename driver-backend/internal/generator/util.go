package generator

import "math"

// Note: if x == y, return x.
func MinInt(x int, y int) int {
	if x <= y {
		return x
	}

	return y
}

// Note: if x == y, return x.
func MaxInt(x int, y int) int {
	if x >= y {
		return x
	}

	return y
}

func RoundUp(x, unit float64) float64 {
	rounded := math.Round(x/unit) * unit

	if rounded < x*unit {
		return rounded + unit
	}

	return rounded
}

// Used to unmarshall CSV file containing max session CPU.
// The file is in format task_id,max_cpu
type SessionMaxCpu struct {
	SessionID      string `csv:"session_id"`          // The session's ID.
	MaxUtilization string `csv:"max_cpu_utilization"` // Maximum CPU utilization of the session.
}

// Used to unmarshall CSV file containing max session memory.
// The file is in format task_id,max_memory
type SessionMaxMemory struct {
	SessionID      string `csv:"session_id"`       // The session's ID.
	MaxMemoryBytes string `csv:"max_memory_bytes"` // Maximum memory (in bytes) used by the session.
}

// Used to unmarshall CSV file containing max session GPUs.
// The file is in format session_id,max_gpu_utilization,num_gpus
type SessionMaxGpu struct {
	SessionID      string `csv:"session_id"`          // The session's ID.
	MaxUtilization string `csv:"max_gpu_utilization"` // Maximum GPU utilization of the session.
	NumGPUs        string `csv:"num_gpus"`            // Number of GPUs used by the session. We may convert this to another value (by multiplying it by `MaxUtilization`), if configured to do so.
}

// Used to unmarshall CSV file containing max session CPU.
// The file is in format task_id,max_cpu
type TrainingTaskMaxCpu struct {
	SessionID       string `csv:"session_id"`          // The session's ID.
	TrainingTaskNum string `csv:"seq"`                 // ID representing the chronological order of the training event (0 is first, 1 is second, etc.)
	MaxUtilization  string `csv:"max_cpu_utilization"` // Maximum CPU utilization of the session.
}

// Used to unmarshall CSV file containing max session memory.
// The file is in format task_id,max_memory
type TrainingTaskMemory struct {
	SessionID       string `csv:"session_id"`    // The session's ID.
	TrainingTaskNum string `csv:"seq"`           // ID representing the chronological order of the training event (0 is first, 1 is second, etc.)
	MaxMemoryBytes  string `csv:"max_mem_bytes"` // Maximum memory (in bytes) used by the session.
}

// Used to unmarshall CSV file containing max session GPUs.
// The file is in format session_id,max_gpu_utilization,num_gpus
type TrainingTaskMaxGpu struct {
	SessionID       string `csv:"session_id"`          // The session's ID.
	TrainingTaskNum string `csv:"seq"`                 // ID representing the chronological order of the training event (0 is first, 1 is second, etc.)
	MaxUtilization  string `csv:"max_gpu_utilization"` // Maximum GPU utilization of the session.
	NumGPUs         string `csv:"num_gpus"`            // Number of GPUs used by the session. We may convert this to another value (by multiplying it by `MaxUtilization`), if configured to do so.
}
