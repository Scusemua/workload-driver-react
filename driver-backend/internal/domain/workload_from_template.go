package domain

import (
	"go.uber.org/zap"
	"time"
)

// WorkloadFromTemplate is a struct representing a Workload that is generated using the "template" option
// within the frontend dashboard.
type WorkloadFromTemplate struct {
	*BasicWorkload

	Sessions []*WorkloadTemplateSession `json:"workload_template"`
}

func NewWorkloadFromTemplate(baseWorkload Workload, sourceSessions []*WorkloadTemplateSession) *WorkloadFromTemplate {
	if sourceSessions == nil {
		panic("WorkloadSessions slice cannot be nil when creating a new workload from a template.")
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

	workloadFromTemplate := &WorkloadFromTemplate{
		BasicWorkload: baseWorkloadImpl,
		Sessions:      sourceSessions,
	}

	baseWorkloadImpl.WorkloadType = TemplateWorkload
	baseWorkloadImpl.workloadInstance = workloadFromTemplate

	return workloadFromTemplate
}

// SessionDelayed should be called when events for a particular Session are delayed for processing, such as
// due to there being too much resource contention.
//
// Multiple calls to SessionDelayed will treat each passed delay additively, as in they'll all be added together.
func (w *WorkloadFromTemplate) SessionDelayed(sessionId string, delayAmount time.Duration) {
	val, loaded := w.sessionsMap.Get(sessionId)
	if !loaded {
		return
	}

	session := val.(*WorkloadTemplateSession)
	session.TotalDelayMilliseconds += delayAmount.Milliseconds()
	session.TotalDelayIncurred += delayAmount
}

func (w *WorkloadFromTemplate) GetWorkloadSource() interface{} {
	return w.Sessions
}

func (w *WorkloadFromTemplate) SetSource(source interface{}) error {
	if source == nil {
		panic("Cannot use nil source for WorkloadFromTemplate")
	}

	var (
		sourceSessions []*WorkloadTemplateSession
		ok             bool
	)
	if sourceSessions, ok = source.([]*WorkloadTemplateSession); !ok {
		panic("Workload source is not correct type for WorkloadFromTemplate.")
	}

	w.workloadSource = sourceSessions
	return w.SetSessions(sourceSessions)
}

// SetSessions sets the sessions that will be involved in this workload.
//
// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
func (w *WorkloadFromTemplate) SetSessions(sessions []*WorkloadTemplateSession) error {
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

// SessionCreated is called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromTemplate) SessionCreated(sessionId string, metadata SessionMetadata) {
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1

	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find newly-created session in session map.", zap.String("session_id", sessionId))
		return
	}

	session := val.(*WorkloadTemplateSession)
	if err := session.SetState(SessionIdle); err != nil {
		w.logger.Error("Failed to set session state.", zap.String("session_id", sessionId), zap.Error(err))
	}

	session.SetCurrentResourceRequest(&ResourceRequest{
		VRAM:     metadata.GetVRAM(),
		Cpus:     metadata.GetCpuUtilization(),
		MemoryMB: metadata.GetMemoryUtilization(),
		Gpus:     metadata.GetNumGPUs(),
	})
}
