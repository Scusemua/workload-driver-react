package domain

import (
	"encoding/json"
	"go.uber.org/zap"
	"time"
)

type PreloadedWorkloadTemplate struct {
	// DisplayName is the display name of the preloaded workload template.
	DisplayName string `json:"display_name" yaml:"display_name" name:"display_name" mapstructure:"display_name"`

	// Key uniquely identifies the PreloadedWorkloadTemplate.
	Key string `json:"key" yaml:"key" mapstructure:"key" name:"key"`

	// Filepath is the file path of the .JSON workload template file.
	Filepath string `json:"filepath" yaml:"filepath" name:"filepath" mapstructure:"filepath"`

	// NumSessions is the number of sessions that will be created by/in the workload.
	NumSessions int `json:"num_sessions" yaml:"num_sessions" name:"num_sessions" mapstructure:"num_sessions"`

	// NumTrainings is the total number of training events in the workload (for all sessions).
	NumTrainings int `json:"num_training_events" yaml:"num_training_events" name:"num_training_events" mapstructure:"num_training_events"`

	// IsLarge indicates if the workload is "arbitrarily" large, as in it is up to the creator of the template
	// (or whoever creates the configuration file with all the preloaded workload templates in it) to designate
	// a workload as "large".
	IsLarge bool `json:"large" yaml:"large" name:"large" mapstructure:"large"`
}

func (t *PreloadedWorkloadTemplate) String() string {
	m, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}

	return string(m)
}

// WorkloadFromTemplate is a struct representing a Workload that is generated using the "template" option
// within the frontend dashboard.
type WorkloadFromTemplate struct {
	*BasicWorkload

	Sessions []*WorkloadTemplateSession `json:"sessions"`
}

func NewWorkloadFromTemplate(baseWorkload Workload, sourceSessions []*WorkloadTemplateSession) (*WorkloadFromTemplate, error) {
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
	}

	err := workloadFromTemplate.SetSource(sourceSessions)
	if err != nil {
		return nil, err
	}

	baseWorkloadImpl.WorkloadType = TemplateWorkload
	baseWorkloadImpl.workloadInstance = workloadFromTemplate

	return workloadFromTemplate, nil
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
	err := w.SetSessions(sourceSessions)
	if err != nil {
		w.logger.Error("Failed to assign source to WorkloadFromTemplate.", zap.Error(err))
		return err
	}

	w.logger.Debug("Assigned source to WorkloadFromTemplate.", zap.Int("num_sessions", len(sourceSessions)))

	return nil
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
