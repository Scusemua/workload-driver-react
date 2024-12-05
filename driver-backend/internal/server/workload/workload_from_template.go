package workload

import (
	"encoding/json"
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
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

// Template is a struct representing a Workload that is generated using the "template" option
// within the frontend dashboard.
type Template struct {
	*BasicWorkload

	Sessions []*domain.WorkloadTemplateSession `json:"sessions"`
}

func NewWorkloadFromTemplate(baseWorkload *BasicWorkload, sourceSessions []*domain.WorkloadTemplateSession) (*Template, error) {
	if sourceSessions == nil {
		panic("WorkloadSessions slice cannot be nil when creating a new workload from a template.")
	}

	if baseWorkload == nil {
		panic("Base workload cannot be nil when creating a new workload.")
	}

	workloadFromTemplate := &Template{
		BasicWorkload: baseWorkload,
	}

	baseWorkload.WorkloadType = TemplateWorkload
	baseWorkload.workloadInstance = workloadFromTemplate

	err := workloadFromTemplate.SetSource(sourceSessions)
	if err != nil {
		return nil, err
	}

	return workloadFromTemplate, nil
}

// SessionDelayed should be called when events for a particular Session are delayed for processing, such as
// due to there being too much resource contention.
//
// Multiple calls to SessionDelayed will treat each passed delay additively, as in they'll all be added together.
func (w *Template) SessionDelayed(sessionId string, delayAmount time.Duration) {
	val, loaded := w.sessionsMap[sessionId]
	if !loaded {
		return
	}

	session := val.(*domain.WorkloadTemplateSession)
	session.TotalDelayMilliseconds += delayAmount.Milliseconds()
	session.TotalDelayIncurred += delayAmount
}

func (w *Template) GetWorkloadSource() interface{} {
	return w.Sessions
}

func (w *Template) SetSource(source interface{}) error {
	if source == nil {
		panic("Cannot use nil source for Template")
	}

	var (
		sourceSessions []*domain.WorkloadTemplateSession
		ok             bool
	)
	if sourceSessions, ok = source.([]*domain.WorkloadTemplateSession); !ok {
		panic("Workload source is not correct type for Template.")
	}

	w.workloadSource = sourceSessions
	err := w.SetSessions(sourceSessions)
	if err != nil {
		w.logger.Error("Failed to assign source to Template.", zap.Error(err))
		return err
	}

	w.logger.Debug("Assigned source to Template.", zap.Int("num_sessions", len(sourceSessions)))

	return nil
}

// SetSessions sets the sessions that will be involved in this workload.
//
// IMPORTANT: This can only be set once per workload. If it is called more than once, it will panic.
func (w *Template) SetSessions(sessions []*domain.WorkloadTemplateSession) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Sessions = sessions
	w.sessionsSet = true
	w.Statistics.TotalNumSessions = len(sessions)

	// Add each session to our internal mapping and initialize the session.
	for _, session := range sessions {
		if session.CurrentResourceRequest == nil {
			session.SetCurrentResourceRequest(domain.NewResourceRequest(0, 0, 0, 0, "ANY_GPU"))
		}

		if session.MaxResourceRequest == nil {
			w.logger.Error("Session does not have a 'max' resource request.",
				zap.String("session_id", session.GetId()),
				zap.String("workload_id", w.Id),
				zap.String("workload_name", w.Name),
				zap.String("session", session.String()))

			return domain.ErrMissingMaxResourceRequest
		}

		if session.NumTrainingEvents == 0 && len(session.Trainings) > 0 {
			session.NumTrainingEvents = len(session.Trainings)
		}

		// Need to set this before calling unsafeIsSessionBeingSampled.
		w.sessionsMap[session.GetId()] = session

		// Decide if the Session should be sampled or not.
		isSampled := w.unsafeIsSessionBeingSampled(session.Id)
		if isSampled {
			err := session.SetState(domain.SessionAwaitingStart)
			if err != nil {
				w.logger.Error("Failed to set session state.", zap.String("session_id", session.GetId()), zap.Error(err))
			}
		}
	}

	if w.Statistics.NumDiscardedSessions > 0 {
		w.logger.Debug("Discarded unsampled sessions.",
			zap.String("workload_id", w.Id),
			zap.String("workload_name", w.Name),
			zap.Int("total_num_sessions", len(sessions)),
			zap.Int("sessions_sampled", w.Statistics.NumSampledSessions),
			zap.Int("sessions_discarded", w.Statistics.NumDiscardedSessions))
	}

	return nil
}

// SessionCreated is called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *Template) SessionCreated(sessionId string, metadata domain.SessionMetadata) {
	w.Statistics.NumActiveSessions += 1
	w.Statistics.NumSessionsCreated += 1

	val, ok := w.sessionsMap[sessionId]
	if !ok {
		w.logger.Error("Failed to find newly-created session in session map.", zap.String("session_id", sessionId))
		return
	}

	session := val.(*domain.WorkloadTemplateSession)
	if err := session.SetState(domain.SessionIdle); err != nil {
		w.logger.Error("Failed to set session state.", zap.String("session_id", sessionId), zap.Error(err))
	}

	session.SetCurrentResourceRequest(&domain.ResourceRequest{
		VRAM:     metadata.GetVRAM(),
		Cpus:     metadata.GetCpuUtilization(),
		MemoryMB: metadata.GetMemoryUtilization(),
		Gpus:     metadata.GetNumGPUs(),
	})
}

// SessionDiscarded is used to record that a particular session is being discarded/not sampled.
func (w *Template) SessionDiscarded(sessionId string) error {
	val, loaded := w.sessionsMap[sessionId]
	if !loaded {
		return fmt.Errorf("%w: \"%s\"", domain.ErrUnknownSession, sessionId)
	}

	w.Statistics.NumDiscardedSessions += 1

	session := val.(*domain.WorkloadTemplateSession)
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
