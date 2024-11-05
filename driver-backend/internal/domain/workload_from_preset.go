package domain

// WorkloadFromPreset is a struct representing a workload that is generated using the "preset" option
// within the frontend dashboard.
//
// Presets are how we run workloads from trace data (among other things).
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

// SessionCreated is called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) SessionCreated(sessionId string) {
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1

	// Haven't implemented logic to add/create WorkloadSessions for preset-based workloads.
	panic("Not yet supported.")
}

// SessionStopped is called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) SessionStopped(sessionId string) {
	w.NumActiveSessions -= 1

	// Haven't implemented logic to add/create WorkloadSessions for preset-based workloads.
	panic("Not yet supported.")
}

// TrainingStarted is called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) TrainingStarted(sessionId string) {
	w.NumActiveTrainings += 1

	// Haven't implemented logic to add/create WorkloadSessions for preset-based workloads.
	panic("Not yet supported.")
}

// TrainingStopped is called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) TrainingStopped(sessionId string) {
	w.NumTasksExecuted += 1
	w.NumActiveTrainings -= 1

	// Haven't implemented logic to add/create WorkloadSessions for preset-based workloads.
	panic("Not yet supported.")
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

	workloadFromPreset := &WorkloadFromPreset{
		workloadImpl:       baseWorkloadImpl,
		WorkloadPreset:     workloadPreset,
		WorkloadPresetName: workloadPreset.GetName(),
		WorkloadPresetKey:  workloadPreset.GetKey(),
	}

	baseWorkloadImpl.WorkloadType = PresetWorkload
	baseWorkloadImpl.workloadInstance = workloadFromPreset

	return workloadFromPreset
}
