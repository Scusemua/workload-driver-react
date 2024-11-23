package domain

import (
	"go.uber.org/zap"
	"time"
)

// WorkloadFromPreset is a struct representing a workload that is generated using the "preset" option
// within the frontend dashboard.
//
// Presets are how we run workloads from trace data (among other things).
type WorkloadFromPreset struct {
	*BasicWorkload

	WorkloadPreset        *WorkloadPreset        `json:"workload_preset"`
	WorkloadPresetName    string                 `json:"workload_preset_name"`
	WorkloadPresetKey     string                 `json:"workload_preset_key"`
	MaxUtilizationWrapper *MaxUtilizationWrapper `json:"max_utilization_wrapper"`

	Sessions []*BasicWorkloadSession `json:"sessions"`
}

func (w *WorkloadFromPreset) GetWorkloadSource() interface{} {
	return w.WorkloadPreset
}

func (w *WorkloadFromPreset) SetSource(source interface{}) error {
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

	return nil
}

func (w *WorkloadFromPreset) SetMaxUtilizationWrapper(wrapper *MaxUtilizationWrapper) {
	w.MaxUtilizationWrapper = wrapper
}

// SetSessions sets the sessions that will be involved in this workload.
//
// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
func (w *WorkloadFromPreset) SetSessions(sessions []*BasicWorkloadSession) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Sessions = sessions
	w.sessionsSet = true

	// Add each session to our internal mapping and initialize the session.
	for _, session := range sessions {
		if err := session.SetState(SessionAwaitingStart); err != nil {
			w.logger.Error("Failed to set session state.", zap.String("session_id", session.GetId()), zap.Error(err))
		}

		if session.CurrentResourceRequest == nil {
			session.SetCurrentResourceRequest(NewResourceRequest(0, 0, 0, 0, "ANY_GPU"))
		}

		if session.MaxResourceRequest == nil {
			w.logger.Error("Session does not have a 'max' resource request.",
				zap.String("session_id", session.GetId()),
				zap.String("workload_id", w.Id),
				zap.String("workload_name", w.Name))

			return ErrMissingMaxResourceRequest
		}

		w.sessionsMap.Set(session.GetId(), session)
	}

	return nil
}

// SessionDelayed should be called when events for a particular Session are delayed for processing, such as
// due to there being too much resource contention.
//
// Multiple calls to SessionDelayed will treat each passed delay additively, as in they'll all be added together.
func (w *WorkloadFromPreset) SessionDelayed(sessionId string, delayAmount time.Duration) {
	val, loaded := w.sessionsMap.Get(sessionId)
	if !loaded {
		return
	}

	session := val.(*WorkloadTemplateSession)
	session.TotalDelayMilliseconds += delayAmount.Milliseconds()
	session.TotalDelayIncurred += delayAmount
}

// SessionCreated is called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromPreset) SessionCreated(sessionId string, metadata SessionMetadata) {
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1

	if w.MaxUtilizationWrapper == nil {
		panic("max utilization wrapper not set by the time sessions are being created")
	}

	maxCpu, loadedCpus := w.MaxUtilizationWrapper.CpuSessionMap[sessionId]
	if !loadedCpus {
		w.logger.Warn("Could not load maximum CPU value for session.", zap.String("sessionId", sessionId))
		maxCpu = 0
	}

	maxMemory, loadedMemory := w.MaxUtilizationWrapper.MemSessionMap[sessionId]
	if !loadedMemory {
		w.logger.Warn("Could not load maximum MEM value for session.", zap.String("sessionId", sessionId))
		maxMemory = 0
	}

	maxGpus, loadedGpus := w.MaxUtilizationWrapper.GpuSessionMap[sessionId]
	if !loadedGpus {
		w.logger.Warn("Could not load maximum GPU value for session.", zap.String("sessionId", sessionId))
		maxGpus = 0
	}

	maxVram, loadedVram := w.MaxUtilizationWrapper.VramSessionMap[sessionId]
	if !loadedVram {
		w.logger.Warn("Could not load maximum VRAM value for session.", zap.String("sessionId", sessionId))
		maxVram = 0
	}

	maxResourceRequest := NewResourceRequest(maxCpu, maxMemory, maxGpus, maxVram, "ANY_GPU")

	// Haven't implemented logic to add/create WorkloadSessions for preset-based workloads.
	session := newWorkloadSession(sessionId, metadata, maxResourceRequest, time.Now(), w.atom)

	session.SetCurrentResourceRequest(&ResourceRequest{
		VRAM:     metadata.GetVRAM(),
		Cpus:     metadata.GetCpuUtilization(),
		MemoryMB: metadata.GetMemoryUtilization(),
		Gpus:     metadata.GetNumGPUs(),
	})

	w.Sessions = append(w.Sessions, session)
	w.sessionsMap.Set(sessionId, session)
}

func NewWorkloadFromPreset(baseWorkload Workload, workloadPreset *WorkloadPreset) *WorkloadFromPreset {
	if workloadPreset == nil {
		panic("Workload preset cannot be nil when creating a new workload from a preset.")
	}

	if baseWorkload == nil {
		panic("Base workload cannot be nil when creating a new workload.")
	}

	var (
		baseWorkloadImpl *BasicWorkload
		ok               bool
	)
	if baseWorkloadImpl, ok = baseWorkload.(*BasicWorkload); !ok {
		panic("The provided workload is not a base workload, or it is not a pointer type.")
	}

	workloadFromPreset := &WorkloadFromPreset{
		BasicWorkload:      baseWorkloadImpl,
		WorkloadPreset:     workloadPreset,
		WorkloadPresetName: workloadPreset.GetName(),
		WorkloadPresetKey:  workloadPreset.GetKey(),
		Sessions:           make([]*BasicWorkloadSession, 0),
	}

	baseWorkloadImpl.WorkloadType = PresetWorkload
	baseWorkloadImpl.workloadInstance = workloadFromPreset

	return workloadFromPreset
}
