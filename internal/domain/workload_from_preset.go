package domain

import "go.uber.org/zap"

type WorkloadFromPreset struct {
	*workloadImpl

	WorkloadPreset     *WorkloadPreset `json:"workload_preset"`
	WorkloadPresetName string          `json:"workload_preset_name"`
	WorkloadPresetKey  string          `json:"workload_preset_key"`
}

func (w *WorkloadFromPreset) GetWorkloadSource() interface{} {
	return w.WorkloadPreset
}

func (w *WorkloadFromPreset) SetSource(source interface{}) {
	if source == nil {
		panic("Cannot use nil source for WorkloadFromPreset")
	}

	var (
		preset *WorkloadPreset
		ok     bool
	)
	if preset, ok = source.(*WorkloadPreset); !ok {
		panic("Workload source is not correct type for WorkloadFromPreset.")
	}

	w.workloadSource = preset
}

// Called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) SessionCreated(sessionId string) {
	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find newly-created session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(*WorkloadSession)
	session.State = SessionIdle
}

// Called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) SessionStopped(sessionId string) {
	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find freshly-terminated session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(*WorkloadSession)
	session.State = SessionStopped
}

// Called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) TrainingStarted(sessionId string) {
	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find now-training session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(*WorkloadSession)
	session.State = SessionTraining
}

// Called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) TrainingStopped(sessionId string) {
	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find now-idle session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(*WorkloadSession)
	session.State = SessionIdle
}

func NewWorkloadFromPreset(baseWorkload Workload, workloadPreset *WorkloadPreset) *WorkloadFromPreset {
	if workloadPreset == nil {
		panic("Workload preset cannot be nil when creating a new workload from a preset.")
	}

	if baseWorkload == nil {
		panic("Base workload cannot be nil when creating a new workload.")
	}

	var (
		baseWorkloadImpl *workloadImpl
		ok               bool
	)
	if baseWorkloadImpl, ok = baseWorkload.(*workloadImpl); !ok {
		panic("The provided workload is not a base workload, or it is not a pointer type.")
	}

	workload_from_preset := &WorkloadFromPreset{
		workloadImpl:       baseWorkloadImpl,
		WorkloadPreset:     workloadPreset,
		WorkloadPresetName: workloadPreset.GetName(),
		WorkloadPresetKey:  workloadPreset.GetKey(),
	}

	baseWorkloadImpl.WorkloadType = PresetWorkload
	baseWorkloadImpl.workload = workload_from_preset

	return workload_from_preset
}
