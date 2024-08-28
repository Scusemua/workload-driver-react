package domain

import "go.uber.org/zap"

type WorkloadFromTemplate struct {
	*workloadImpl

	Sessions []*WorkloadTemplateSession `json:"workload_template"`
}

func (w *WorkloadFromTemplate) GetWorkloadSource() interface{} {
	return w.Sessions
}

func (w *WorkloadFromTemplate) SetSource(source interface{}) {
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

	workloadTemplateSessions := sourceSessions
	sessions := make([]WorkloadSession, 0, len(sourceSessions))

	for _, workloadTemplateSession := range workloadTemplateSessions {
		sessions = append(sessions, workloadTemplateSession)
	}

	w.SetSessions(sessions)
}

// Called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromTemplate) SessionCreated(sessionId string) {
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1

	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find newly-created session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(WorkloadSession)
	session.SetState(SessionIdle)
}

// Called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *WorkloadFromTemplate) SessionStopped(sessionId string) {
	w.NumActiveSessions -= 1

	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find freshly-terminated session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(WorkloadSession)
	session.SetState(SessionStopped)
}

// Called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *WorkloadFromTemplate) TrainingStarted(sessionId string) {
	w.NumActiveTrainings += 1

	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find now-training session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(WorkloadSession)
	session.SetState(SessionTraining)
}

// Called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *WorkloadFromTemplate) TrainingStopped(sessionId string) {
	w.NumTasksExecuted += 1
	w.NumActiveTrainings -= 1

	val, ok := w.sessionsMap.Get(sessionId)
	if !ok {
		w.logger.Error("Failed to find now-idle session in session map.", zap.String("session-id", sessionId))
		return
	}

	session := val.(WorkloadSession)
	session.SetState(SessionIdle)
	session.GetAndIncrementTrainingsCompleted()
}

func NewWorkloadFromTemplate(baseWorkload Workload, sourceSessions []*WorkloadTemplateSession) *WorkloadFromTemplate {
	if sourceSessions == nil {
		panic("WorkloadSessions slice cannot be nil when creating a new workload from a template.")
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

	workload_from_template := &WorkloadFromTemplate{
		workloadImpl: baseWorkloadImpl,
		Sessions:     sourceSessions,
	}

	baseWorkloadImpl.WorkloadType = TemplateWorkload
	baseWorkloadImpl.workload = workload_from_template

	return workload_from_template
}
