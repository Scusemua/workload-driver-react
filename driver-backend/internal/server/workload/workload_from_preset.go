package workload

import (
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"time"
)

// Preset is a struct representing a workload that is generated using the "preset" option
// within the frontend dashboard.
//
// Presets are how we run workloads from trace data (among other things).
type Preset struct {
	*BasicWorkload

	WorkloadPreset        *domain.WorkloadPreset        `json:"workload_preset"`
	WorkloadPresetName    string                        `json:"workload_preset_name"`
	WorkloadPresetKey     string                        `json:"workload_preset_key"`
	MaxUtilizationWrapper *domain.MaxUtilizationWrapper `json:"max_utilization_wrapper"`

	Sessions []*domain.BasicWorkloadSession `json:"sessions"`
}

func NewWorkloadFromPreset(baseWorkload *BasicWorkload, workloadPreset *domain.WorkloadPreset) *Preset {
	if workloadPreset == nil {
		panic("Workload preset cannot be nil when creating a new workload from a preset.")
	}

	if baseWorkload == nil {
		panic("Base workload cannot be nil when creating a new workload.")
	}

	workloadFromPreset := &Preset{
		BasicWorkload:      baseWorkload,
		WorkloadPreset:     workloadPreset,
		WorkloadPresetName: workloadPreset.GetName(),
		WorkloadPresetKey:  workloadPreset.GetKey(),
		Sessions:           make([]*domain.BasicWorkloadSession, 0),
	}

	baseWorkload.WorkloadType = PresetWorkload
	baseWorkload.workloadInstance = workloadFromPreset

	return workloadFromPreset
}

func (w *Preset) GetWorkloadSource() interface{} {
	return w.WorkloadPreset
}

func (w *Preset) unsafeSetSource(source interface{}) error {
	if source == nil {
		panic("Cannot use nil source for Preset")
	}

	var (
		preset *domain.WorkloadPreset
		ok     bool
	)
	if preset, ok = source.(*domain.WorkloadPreset); !ok {
		panic("Workload source is not correct type for Preset.")
	}

	w.workloadSource = preset

	return nil
}

func (w *Preset) SetMaxUtilizationWrapper(wrapper *domain.MaxUtilizationWrapper) {
	w.MaxUtilizationWrapper = wrapper
}

func (w *Preset) unsafeSetSessions(sessions []*domain.BasicWorkloadSession) error {
	w.Sessions = sessions
	w.sessionsSet = true

	// Add each session to our internal mapping and initialize the session.
	for _, session := range sessions {
		if err := session.SetState(domain.SessionAwaitingStart); err != nil {
			w.logger.Error("Failed to set session state.", zap.String("session_id", session.GetId()), zap.Error(err))
		}

		if session.CurrentResourceRequest == nil {
			session.SetCurrentResourceRequest(domain.NewResourceRequest(0, 0, 0, 0, "ANY_GPU"))
		}

		if session.MaxResourceRequest == nil {
			w.logger.Error("Session does not have a 'max' resource request.",
				zap.String("session_id", session.GetId()),
				zap.String("workload_id", w.Id),
				zap.String("workload_name", w.Name))

			return domain.ErrMissingMaxResourceRequest
		}

		w.sessionsMap[session.GetId()] = session
	}

	w.Statistics.TotalNumSessions = len(sessions)

	return nil
}

// SessionDelayed should be called when events for a particular Session are delayed for processing, such as
// due to there being too much resource contention.
//
// Multiple calls to SessionDelayed will treat each passed delay additively, as in they'll all be added together.
func (w *Preset) SessionDelayed(sessionId string, delayAmount time.Duration) {
	val, loaded := w.sessionsMap[sessionId]
	if !loaded {
		return
	}

	session := val.(*domain.WorkloadTemplateSession)
	session.TotalDelayMilliseconds += delayAmount.Milliseconds()
	session.TotalDelayIncurred += delayAmount
}

// SessionCreated is called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *Preset) SessionCreated(sessionId string, metadata domain.SessionMetadata) {
	w.Statistics.NumActiveSessions += 1
	w.Statistics.NumSessionsCreated += 1

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

	maxResourceRequest := domain.NewResourceRequest(maxCpu, maxMemory, maxGpus, maxVram, "ANY_GPU")

	// Haven't implemented logic to add/create WorkloadSessions for preset-based workloads.
	session := domain.NewWorkloadSession(sessionId, metadata, maxResourceRequest, time.Now(), w.atom)

	session.SetCurrentResourceRequest(&domain.ResourceRequest{
		VRAM:     metadata.GetVRAM(),
		Cpus:     metadata.GetCpuUtilization(),
		MemoryMB: metadata.GetMemoryUtilization(),
		Gpus:     metadata.GetNumGPUs(),
	})

	w.Sessions = append(w.Sessions, session)
	w.sessionsMap[sessionId] = session
}

// SessionDiscarded is used to record that a particular session is being discarded/not sampled.
func (w *Preset) SessionDiscarded(sessionId string) error {
	val, loaded := w.sessionsMap[sessionId]
	if !loaded {
		return fmt.Errorf("%w: \"%s\"", domain.ErrUnknownSession, sessionId)
	}

	w.Statistics.NumDiscardedSessions += 1

	session := val.(*domain.BasicWorkloadSession)
	err := session.SetState(domain.SessionDiscarded)
	if err != nil {
		w.logger.Error("Could not transition session to the 'discarded' state.",
			zap.String("workload_id", w.Id),
			zap.String("workload_name", w.Name),
			zap.String("session_id", sessionId),
			zap.Error(err))
		return err
	}

	return nil
}
