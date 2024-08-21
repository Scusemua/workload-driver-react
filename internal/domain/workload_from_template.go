package domain

import "go.uber.org/zap"

type WorkloadFromTemplate struct {
	*workloadImpl

	Template *WorkloadTemplate `json:"workload_template"`
}

func (w *WorkloadFromTemplate) GetWorkloadSource() interface{} {
	return w.Template
}

func (w *WorkloadFromTemplate) SetSource(source interface{}) {
	if source == nil {
		panic("Cannot use nil source for WorkloadFromTemplate")
	}

	var (
		template *WorkloadTemplate
		ok       bool
	)
	if template, ok = source.(*WorkloadTemplate); !ok {
		panic("Workload source is not correct type for WorkloadFromTemplate.")
	}

	w.workloadSource = template.GetSessions()

	workloadTemplateSessions := template.GetSessions()
	sessions := make([]WorkloadSession, 0, len(workloadTemplateSessions))

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

func NewWorkloadFromTemplate(baseWorkload Workload, workloadTemplate *WorkloadTemplate) *WorkloadFromTemplate {
	if workloadTemplate == nil {
		panic("Workload template cannot be nil when creating a new workload from a template.")
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
		Template:     workloadTemplate,
	}

	baseWorkloadImpl.WorkloadType = TemplateWorkload
	baseWorkloadImpl.workload = workload_from_template

	return workload_from_template
}
