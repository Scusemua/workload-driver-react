package generator

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

const (
	SessionStatusInit SessionStatus = iota
	SessionStatusInitializing
	SessionStatusIdle
	SessionStatusTraining
	SessionStatusStopping
	SessionStatusStopped

	SessionCPUReady = 0x01
	SessionGPUReady = 0x02
	SessionMemReady = 0x04
)

var (
	NoSessionEvent []domain.SessionEventName = nil

	ErrUnexpectedSessionState   = errors.New("unexpected session state")
	ErrUnexpectedSessionStTrans = errors.New("unexpected session state transition")

	SessionReadyExpects = SessionCPUReady | SessionGPUReady
	SessionStopExpects  = SessionCPUReady | SessionGPUReady
	ErrEventPending     = errors.New("event pending")
)

type SessionStatus int

func (s SessionStatus) String() string {
	return [...]string{"Init", "Initializing", "Idle", "Training", "Stopping", "Stopped"}[s]
}

type SessionMeta struct {
	Timestamp time.Time
	Pod       string   `json:"pod"`
	GPU       *GPUUtil `json:"gpu"`
	VRAM      float64  `json:"vram"`
	CPU       *CPUUtil `json:"cpu"`

	// The maximum number of CPUs that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	MaxSessionCPUs float64 `json:"maxSessionCPUs"`

	// The maximum amount of memory (in MB) that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	MaxSessionMemory float64 `json:"maxSessionMemory"`

	// The maximum number of GPUs that this SessionMeta will ever use.
	// This is obtained by performing a "pre-run".
	MaxSessionGPUs int `json:"maxSessionGPUs"`

	// MaxSessionVRAM is the maximum amount of VRAM (i.e., GPU memory) that the SessionMeta will ever use in GB.
	MaxSessionVRAM float64 `json:"maxSessionVRAM"`

	// The maximum number of CPUs that this SessionMeta will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the SessionMeta is attached as data to a 'training-started' event.
	CurrentTrainingMaxCPUs float64 `json:"currentTrainingMaxCPUs"`

	// The maximum amount of memory (in MB) that this SessionMeta will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the SessionMeta is attached as data to a 'training-started' event.
	CurrentTrainingMaxMemory float64 `json:"currentTrainingMaxMemory"`

	// The maximum number of GPUs that this SessionMeta will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the SessionMeta is attached as data to a 'training-started' event.
	CurrentTrainingMaxGPUs int `json:"currentTrainingMaxGPUs"`

	// The maximum amount of VRAM in GB that this SessionMeta will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the SessionMeta is attached as data to a 'training-started' event.
	CurrentTrainingMaxVRAM float64 `json:"currentTrainingMaxVRAM"`

	// If we're adjusting the MaxSessionGPUs value, then we also need to keep track of an "AdjustmentFactor".
	// Consider a scenario in which session "ExampleSession1" originally had NUM_GPUS: 8 and MAX_GPU_UTIL: 50%.
	// During the tick where ExampleSession1's utilization is reported as 50%, we would typically compute the "true" utilization (across all of its GPUs) as:
	// 8 GPUs * 50% utilization = 400%.
	//
	// But if we're adjusting the MaxSessionGPUs value, then we'll compute MaxSessionGPUs as 8 * 0.50 = 4 GPUs.
	// So now, when ExampleSession1 is at 50% utilization, we'd compute the "true" utilization (across all of its GPUs) as:
	// 4 GPUs * 50% utilization = 200%.
	//
	// This is obviously incorrect. We need to adjust the computed utilization by an "AdjustmentFactor" equal to OriginalMaxGPUs / NewMaxGPUs = 8 / 4 = 2.
	// 200% * 2 = 400%, which is correct.
	//
	// But to simply everything, we'll adjust the utilization values within the workload generator portion -- while we're processing the records with the drivers,
	// rather than at the time when we're computing utilization in the workload simulator.
	AdjustmentFactor float64           `json:"adjustmentFactor"`
	Memory           *MemoryUtil       `json:"memory"`
	MemoryQuerier    *MemoryUtilBuffer `json:"-"`
	Status           SessionStatus     `json:"status"`
	StatusReadyFlags int               // A flag that indicates which trace is ready.

	InitedAt  time.Time     `json:"initedAt"`
	InitDelay time.Duration `json:"initDelay"`

	last    *domain.Event   // Track last event for debugging purpose.
	pending []*domain.Event // For special cases, previous event will be saved here. See Transit implementation.
}

// GetCurrentTrainingMaxCPUs returns the maximum number of CPUs that this SessionMeta will use during its current training task.
// This will only be set (i.e., have a non-zero/non-default value) when the SessionMeta is attached as data to a 'training-started' event.
func (s *SessionMeta) GetCurrentTrainingMaxCPUs() float64 {
	return s.CurrentTrainingMaxCPUs
}

// GetCurrentTrainingMaxMemory returns the maximum amount of memory (in GB) that this SessionMeta will use during its current training task.
// This will only be set (i.e., have a non-zero/non-default value) when the SessionMeta is attached as data to a 'training-started' event.
func (s *SessionMeta) GetCurrentTrainingMaxMemory() float64 {
	return s.CurrentTrainingMaxMemory
}

// GetVRAM returns the VRAM.
func (s *SessionMeta) GetVRAM() float64 {
	return s.VRAM
}

func (s *SessionMeta) GetNumGPUs() int {
	if s.GPU == nil {
		return 0
	}

	return s.GPU.GPUs
}

func (s *SessionMeta) GetCpuUtilization() float64 {
	if s.CPU == nil {
		return 0
	}

	return s.CPU.Value
}

func (s *SessionMeta) GetGpuUtilization() float64 {
	if s.GPU == nil {
		return 0
	}

	return s.GPU.Value
}

func (s *SessionMeta) GetMemoryUtilization() float64 {
	if s.Memory == nil {
		return 0
	}

	return s.Memory.Value
}

// GetCurrentTrainingMaxVRAM returns the SessionMeta's CurrentTrainingMaxVRAM.
func (s *SessionMeta) GetCurrentTrainingMaxVRAM() float64 {
	return s.CurrentTrainingMaxVRAM
}

// GetCurrentTrainingMaxGPUs returns the maximum number of GPUs that this SessionMeta will use during its current training task.
// This will only be set (i.e., have a non-zero/non-default value) when the SessionMeta is attached as data to a 'training-started' event.
func (s *SessionMeta) GetCurrentTrainingMaxGPUs() int {
	return s.CurrentTrainingMaxGPUs
}

// GetGPUs returns the number of GPUs that this Session is configured to use.
func (s *SessionMeta) GetGPUs() int {
	return s.GPU.GPUs
}

// GetMaxSessionCPUs returns the maximum number of CPUs that this SessionMeta will ever use.
// This is obtained by performing a "pre-run".
func (s *SessionMeta) GetMaxSessionCPUs() float64 {
	return s.MaxSessionCPUs
}

// GetMaxSessionMemory returns the maximum amount of memory (in GB) that this SessionMeta will ever use.
// This is obtained by performing a "pre-run".
func (s *SessionMeta) GetMaxSessionMemory() float64 {
	return s.MaxSessionMemory
}

// GetMaxSessionGPUs returns the maximum number of GPUs that this SessionMeta will ever use.
// This is obtained by performing a "pre-run".
func (s *SessionMeta) GetMaxSessionGPUs() int {
	return s.MaxSessionGPUs
}

// GetMaxSessionVRAM returns the maximum amount of VRAM (i.e., GPU memory) that this SessionMeta will ever use in GB.
// This is obtained by performing a "pre-run".
func (s *SessionMeta) GetMaxSessionVRAM() float64 {
	return math.Ceil(float64(s.MaxSessionGPUs) * 8 * 0.75)
}

func (s *SessionMeta) String() string {
	return fmt.Sprintf("drv.Sess[Pod=%s, Timestamp=%v, MaxCPU=%.2f, MaxMem=%.2f, Status=%v, CPU=%v, Memory=%v GPU=%v]", s.Pod, s.Timestamp, s.MaxSessionCPUs, s.MaxSessionMemory, s.Status, s.CPU, s.Memory, s.GPU)
	// if s.Status == SessionStatusInitializing || s.Status == SessionStatusInit {
	// 	return fmt.Sprintf("domain.SessionMeta[pod=%s]", s.Pod)
	// }
	// return fmt.Sprintf("SessionMeta: pod %s, %d gpus, %.2f%%%%/%.2f%%%%, init cpu delay %v", s.Pod, s.GPU.GPUs, s.GPU.Value, s.CPU.Value, s.InitDelay)
	// return fmt.Sprintf("domain.SessionMeta[pod=%s, gpus=%d, %.2f%%%%/%.2f%%%%, init cpu delay %v, mem %v]", s.Pod, s.GPU.GPUs, s.GPU.Value, s.CPU.Value, s.InitDelay, s.Memory)
}

func (s *SessionMeta) GetTS() time.Time {
	return s.Timestamp
}

func (s *SessionMeta) resetReadyFlags() {
	s.StatusReadyFlags = 0
}

func (s *SessionMeta) setReadyFlag(expects int, utils int) int {
	// clear unwanted flags
	utils = utils & expects

	// set flags
	s.StatusReadyFlags |= utils
	return s.StatusReadyFlags
}

func (s *SessionMeta) GetPod() string {
	return s.Pod
}

func (s *SessionMeta) Transit(evt *domain.Event) ([]domain.SessionEventName, error) {
	if s.pending == nil {
		s.pending = make([]*domain.Event, 0, 3)
	}
	// Support the detection of series transitions
	s.Timestamp = evt.Timestamp
	defer func() {
		s.last = evt
	}()

	events, err := s.transit(evt)
	if errors.Is(err, ErrEventPending) {
		return events, nil
	} else if err != nil {
		return events, err
	}

	// Try to apply pending events, if any pending events are applied, we need to re-evaluate the rest.
	for len(s.pending) > 0 {
		// Make a copy of pending events for looping.
		pending := s.pending
		lenPending := len(pending)
		// Clear pending events to store pending events after this iteration.
		// We reuse the backend slice, for the pending events will be appended to original slice with the same order.
		s.pending = pending[:0]
		for _, evt := range pending {
			moreEvents, err := s.transit(evt)
			if errors.Is(err, ErrEventPending) {
				// Not applied? just continue.
				break
			} else if err != nil {
				return events, err
			} else if len(moreEvents) > 0 {
				events = append(events, moreEvents...)
			}
		}
		// If no event is applied, we are done.
		if lenPending == len(s.pending) {
			break
		}
	}
	return events, nil
}

func (s *SessionMeta) transit(evt *domain.Event) ([]domain.SessionEventName, error) {
	// log.Debug("Transitioning SessionMeta. CurrentStatus=%v. Event=%v.", s.Status, evt)
	switch s.Status {
	case SessionStatusInit:
		if evt.Name == EventGPUStarted {
			s.GPU = evt.Data.(*GPUUtil)
			s.Status = SessionStatusInitializing
			s.InitedAt = evt.Timestamp
			s.resetReadyFlags()
			s.setReadyFlag(SessionGPUReady, SessionGPUReady)
			return []domain.SessionEventName{domain.EventSessionStarted}, nil
		} else if evt.Name == EventCPUStarted {
			s.CPU = evt.Data.(*CPUUtil)
			s.Status = SessionStatusInitializing
			s.InitedAt = evt.Timestamp
			s.resetReadyFlags()
			s.setReadyFlag(SessionGPUReady, SessionCPUReady)
			return []domain.SessionEventName{domain.EventSessionStarted}, nil
		} else if evt.Name == EventMemoryStarted {
			s.MemoryQuerier = evt.Data.(*MemoryUtilBuffer)
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp)
			s.Status = SessionStatusInitializing
			s.InitedAt = evt.Timestamp
			s.resetReadyFlags()
			s.setReadyFlag(SessionGPUReady, SessionMemReady)
			return []domain.SessionEventName{domain.EventSessionStarted}, nil
		}
		return NoSessionEvent, Errorf(ErrUnexpectedSessionStTrans, "SessionStatusInit on %v", evt)
	case SessionStatusInitializing:
		if evt.Name == EventGPUStarted {
			s.GPU = evt.Data.(*GPUUtil)
			if s.CPU == nil {
				return NoSessionEvent, Errorf(ErrUnexpectedSessionState, "CPU status is unknown while \"%v\" on SessionStatusInitializing, last event \"%v\"", evt, s.last)
			}
			s.setReadyFlag(SessionGPUReady, SessionGPUReady)
			s.InitDelay = s.CPU.Timestamp.Sub(s.InitedAt)
		} else if evt.Name == EventCPUStarted {
			s.CPU = evt.Data.(*CPUUtil)
			if s.GPU == nil {
				// Since we ignore CPU events before SessionReady(see below), we may see the duplicated CPU started before GPU started.
				break
			}
			s.setReadyFlag(SessionGPUReady, SessionCPUReady)
			s.InitDelay = s.CPU.Timestamp.Sub(s.InitedAt)
		} else if evt.Name == EventMemoryStarted {
			s.MemoryQuerier = evt.Data.(*MemoryUtilBuffer)
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp)
			s.setReadyFlag(SessionGPUReady, SessionMemReady)
			s.InitDelay = s.Memory.Timestamp.Sub(s.InitedAt)
		} else {
			s.pending = append(s.pending, evt)
			return NoSessionEvent, ErrEventPending
		}
		if s.StatusReadyFlags == SessionGPUReady {
			s.Status = SessionStatusIdle
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp) // Update memory reading.
			}
			return []domain.SessionEventName{domain.EventSessionReady}, nil
		}
		return NoSessionEvent, nil
	case SessionStatusIdle:
		if evt.Name == EventCPUActivated || evt.Name == EventCPUDeactivated {
			return NoSessionEvent, nil
		} else if evt.Name == EventGPUActivated {
			s.GPU = evt.Data.(*GPUUtil)
			s.Status = SessionStatusTraining

			if s.CPU != nil {
				s.CPU.MaxTaskCPU = 0
			}
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp) // Update memory reading.
			}
			return []domain.SessionEventName{domain.EventSessionTrainingStarted}, nil
		} else if evt.Name == EventGPUStopped {
			s.GPU = evt.Data.(*GPUUtil)
			s.Status = SessionStatusStopping
			s.resetReadyFlags()
			s.setReadyFlag(SessionStopExpects, SessionGPUReady)
			return NoSessionEvent, nil
		} else if evt.Name == EventCPUStopped {
			s.CPU = evt.Data.(*CPUUtil)
			s.Status = SessionStatusStopping
			s.resetReadyFlags()
			s.setReadyFlag(SessionStopExpects, SessionCPUReady)
			return NoSessionEvent, nil
		} else if evt.Name == EventMemoryStopped {
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp)
			// TODO: ignore memory events during stopping, for now.
			return NoSessionEvent, nil
		}
		return NoSessionEvent, Errorf(ErrUnexpectedSessionStTrans, "SessionStatusIdle on %v", evt)
	case SessionStatusTraining:
		if evt.Name == EventGPUDeactivated {
			s.GPU = evt.Data.(*GPUUtil)
			s.Status = SessionStatusIdle
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp) // Update memory reading.
			}
			return []domain.SessionEventName{domain.EventSessionTrainingEnded}, nil
		} else if evt.Name == EventCPUActivated || evt.Name == EventCPUDeactivated {
			break
		} else if evt.Name == EventCPUStopped {
			// Handling the special case that the session/pod continues after the end of the trace,
			// where CPU stop along with GPU deactivation and GPU may deactivated at a later time.
			s.CPU = evt.Data.(*CPUUtil)
			s.pending = append(s.pending, evt)
			return NoSessionEvent, ErrEventPending
		} else if evt.Name == EventMemoryStopped {
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp)
			// TODO: ignore memory events during stopping, for now.
			break
		}
		// else if evt.Name == EventGpuUpdateUtil {
		// 	s.GPU = evt.Data.(*GPUUtil)

		// 	return NoSessionEvent, nil
		// 	// return []SessionEventName{EventSessionUpdateGpuUtil}, nil
		// }
		return NoSessionEvent, Errorf(ErrUnexpectedSessionStTrans, "SessionStatusTraining on %v", evt)
	case SessionStatusStopping:
		if evt.Name == EventGPUStopped {
			s.GPU = evt.Data.(*GPUUtil)
			s.setReadyFlag(SessionStopExpects, SessionGPUReady)
		} else if evt.Name == EventCPUStopped {
			s.CPU = evt.Data.(*CPUUtil)
			s.setReadyFlag(SessionStopExpects, SessionCPUReady)
		} else if evt.Name == EventGPUStarted && s.GPU.Status == GPUStopped {
			// Deal with situations like regaining GPU readings after missing for a while.
			s.GPU = evt.Data.(*GPUUtil)
			s.Status = SessionStatusIdle
			break
		} else if evt.Name == EventCPUDeactivated || evt.Name == EventMemoryStopped {
			// Ignore irrelevant events.
			break
		} else {
			return NoSessionEvent, Errorf(ErrUnexpectedSessionStTrans, "SessionStatusStopping on event %s", evt.Name)
		}

		// Check if session is ready to stop.
		if s.StatusReadyFlags == SessionStopExpects {
			s.Status = SessionStatusStopped
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp) // Update memory reading.
			}
			return []domain.SessionEventName{domain.EventSessionStopped}, nil
		}
		return NoSessionEvent, nil
	case SessionStatusStopped:
		if evt.Name == EventMemoryStopped {
			return NoSessionEvent, nil
		}
		return NoSessionEvent, Errorf(ErrUnexpectedSessionStTrans, "SessionStatusStopped on %v", evt)
	}

	return NoSessionEvent, nil
}

func (s *SessionMeta) Snapshot() *SessionMeta {
	ss := *s
	return &ss
}
