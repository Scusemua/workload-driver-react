package generator

import (
	"fmt"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

type SessionStatus int

const (
	SessionInit SessionStatus = iota
	SessionInitializing
	SessionIdle
	SessionTraining
	SessionStopping
	SessionStopped
)

func (s SessionStatus) String() string {
	return [...]string{"Init", "Initializing", "Idle", "Training", "Stopping", "Stopped"}[s]
}

type Session struct {
	Timestamp time.Time
	Pod       string
	GPU       *GPUUtil
	CPU       *CPUUtil
	// The maximum number of CPUs that this Session will ever use.
	// This is obtained by performing a "pre-run".
	MaxSessionCPUs float64
	// The maximum amount of memory (in GB) that this Session will ever use.
	// This is obtained by performing a "pre-run".
	MaxSessionMemory float64
	// The maximum number of GPUs that this Session will ever use.
	// This is obtained by performing a "pre-run".
	MaxSessionGPUs int
	// The maximum number of CPUs that this Session will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the Session is attached as data to a 'training-started' event.
	CurrentTrainingMaxCPUs float64
	// The maximum amount of memory (in GB) that this Session will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the Session is attached as data to a 'training-started' event.
	CurrentTrainingMaxMemory float64
	// The maximum number of GPUs that this Session will use during its current training task.
	// This will only be set (i.e., have a non-zero/non-default value) when the Session is attached as data to a 'training-started' event.
	CurrentTrainingMaxGPUs int
	// If we're adjusting the MaxSessionGPUs value, then we also need to keep track of an "AdjustmentFactor".
	// Consider a scenario in which session "ExampleSession1" originally had NUM_GPUS: 8 and MAX_GPU_UTIL: 50%.
	// During the tick where ExampleSession1's utilization is reported as 50%, we would typically compute the "true" utilization (across all of its GPUs) as:
	// 8 GPUs * 50% utilization = 400%.
	//
	// But if we're adusting the MaxSessionGPUs value, then we'll compute MaxSessionGPUs as 8 * 0.50 = 4 GPUs.
	// So now, when ExampleSession1 is at 50% utilization, we'd compute the "true" utilization (across all of its GPUs) as:
	// 4 GPUs * 50% utilization = 200%.
	//
	// This is obviously incorrect. We need to adjust the computed utilization by an "AdjustmentFactor" equal to OriginalMaxGPUs / NewMaxGPUs = 8 / 4 = 2.
	// 200% * 2 = 400%, which is correct.
	//
	// But to simply everything, we'll adjust the utilization values within the workload generator portion -- while we're processing the records with the drivers,
	// rather than at the time when we're computing utilization in the workload simulator.
	AdjustmentFactor float64
	Memory           *MemoryUtil
	MemoryQuerier    *MemoryUtilBuffer
	Status           SessionStatus
	StatusReadyFlags int // A flag that indicates which trace is ready.

	InitedAt  time.Time
	InitDelay time.Duration

	last    domain.Event   // Track last event for debugging purpose.
	pending []domain.Event // For special cases, previous event will be saved here. See Transit implementation.
}

func (s *Session) String() string {
	return fmt.Sprintf("drv.Sess[Pod=%s, Timestamp=%v, MaxCPU=%.2f, MaxMem=%.2f, Status=%v, CPU=%v, Memory=%v GPU=%v]", s.Pod, s.Timestamp, s.MaxSessionCPUs, s.MaxSessionMemory, s.Status, s.CPU, s.Memory, s.GPU)
	// if s.Status == SessionInitializing || s.Status == SessionInit {
	// 	return fmt.Sprintf("generator.Session[pod=%s]", s.Pod)
	// }
	// return fmt.Sprintf("Session: pod %s, %d gpus, %.2f%%%%/%.2f%%%%, init cpu delay %v", s.Pod, s.GPU.GPUs, s.GPU.Value, s.CPU.Value, s.InitDelay)
	// return fmt.Sprintf("generator.Session[pod=%s, gpus=%d, %.2f%%%%/%.2f%%%%, init cpu delay %v, mem %v]", s.Pod, s.GPU.GPUs, s.GPU.Value, s.CPU.Value, s.InitDelay, s.Memory)
}

func (s *Session) GetTS() time.Time {
	return s.Timestamp
}

func (s *Session) resetReadyFlags() {
	s.StatusReadyFlags = 0
}

func (s *Session) setReadyFlag(expects int, utils int) int {
	// clear unwanted flags
	utils = utils & expects

	// set flags
	s.StatusReadyFlags |= utils
	return s.StatusReadyFlags
}

func (s *Session) GetPod() string {
	return s.Pod
}

func (s *Session) Transit(evt domain.Event, inspect bool) ([]domain.SessionEvent, error) {
	if s.pending == nil {
		s.pending = make([]domain.Event, 0, 3)
	}
	// Support the detection of series transitions
	s.Timestamp = evt.Timestamp()
	defer func() {
		s.last = evt
	}()

	events, err := s.transit(evt)
	// sugarLog.Debugf("Transitioned Session. NewStatus=%v. Events=%v.", s.Status, events)
	if err == domain.ErrEventPending {
		return events, nil
	} else if err != nil {
		return events, err
	}

	// Try apply pending events, if any pending events are applied, we need to re-evaluate the rest.
	for len(s.pending) > 0 {
		// Make a copy of pending events for looping.
		pending := s.pending
		lenPending := len(pending)
		// Clear pending events to store pending events after this iteration.
		// We reuse the backend slice, for the pending events will be appended to original slice with the same order.
		s.pending = pending[:0]
		for _, evt := range pending {
			moreEvents, err := s.transit(evt)
			if err == domain.ErrEventPending {
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

func (s *Session) transit(evt domain.Event) ([]domain.SessionEvent, error) {
	// log.Debug("Transitioning Session. CurrentStatus=%v. Event=%v.", s.Status, evt)
	switch s.Status {
	case SessionInit:
		if evt.Name() == EventGPUStarted {
			s.GPU = evt.Data().(*GPUUtil)
			s.Status = SessionInitializing
			s.InitedAt = evt.Timestamp()
			s.resetReadyFlags()
			s.setReadyFlag(domain.SessionReadyExpects, domain.SessionGPUReady)
			return []domain.SessionEvent{domain.EventSessionStarted}, nil
		} else if evt.Name() == EventCPUStarted {
			s.CPU = evt.Data().(*CPUUtil)
			s.Status = SessionInitializing
			s.InitedAt = evt.Timestamp()
			s.resetReadyFlags()
			s.setReadyFlag(domain.SessionReadyExpects, domain.SessionCPUReady)
			return []domain.SessionEvent{domain.EventSessionStarted}, nil
		} else if evt.Name() == EventMemoryStarted {
			s.MemoryQuerier = evt.Data().(*MemoryUtilBuffer)
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp())
			s.Status = SessionInitializing
			s.InitedAt = evt.Timestamp()
			s.resetReadyFlags()
			s.setReadyFlag(domain.SessionReadyExpects, domain.SessionMemReady)
			return []domain.SessionEvent{domain.EventSessionStarted}, nil
		}
		return domain.NoSessionEvent, Errorf(domain.ErrUnexpectedSessionStTrans, "SessionInit on %v", evt)
	case SessionInitializing:
		if evt.Name() == EventGPUStarted {
			s.GPU = evt.Data().(*GPUUtil)
			if s.CPU == nil {
				return domain.NoSessionEvent, Errorf(domain.ErrUnexpectedSessionState, "CPU status is unknown while \"%v\" on SessionInitializing, last event \"%v\"", evt, s.last)
			}
			s.setReadyFlag(domain.SessionReadyExpects, domain.SessionGPUReady)
			s.InitDelay = s.CPU.Timestamp.Sub(s.InitedAt)
		} else if evt.Name() == EventCPUStarted {
			s.CPU = evt.Data().(*CPUUtil)
			if s.GPU == nil {
				// Since we ignore CPU events before SessionReady(see below), we may see the duplicated CPU started before GPU started.
				break
			}
			s.setReadyFlag(domain.SessionReadyExpects, domain.SessionCPUReady)
			s.InitDelay = s.CPU.Timestamp.Sub(s.InitedAt)
		} else if evt.Name() == EventMemoryStarted {
			s.MemoryQuerier = evt.Data().(*MemoryUtilBuffer)
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp())
			s.setReadyFlag(domain.SessionReadyExpects, domain.SessionMemReady)
			s.InitDelay = s.Memory.Timestamp.Sub(s.InitedAt)
		} else {
			s.pending = append(s.pending, evt)
			return domain.NoSessionEvent, domain.ErrEventPending
		}
		if s.StatusReadyFlags == domain.SessionReadyExpects {
			s.Status = SessionIdle
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp()) // Update memory reading.
			}
			return []domain.SessionEvent{domain.EventSessionReady}, nil
		}
		return domain.NoSessionEvent, nil
	case SessionIdle:
		if evt.Name() == EventCPUActivated || evt.Name() == EventCPUDeactivated {
			return domain.NoSessionEvent, nil
		} else if evt.Name() == EventGPUActivated {
			s.GPU = evt.Data().(*GPUUtil)
			s.Status = SessionTraining

			if s.CPU != nil {
				s.CPU.MaxTaskCPU = 0
			}
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp()) // Update memory reading.
			}
			return []domain.SessionEvent{domain.EventSessionTrainingStarted}, nil
		} else if evt.Name() == EventGPUStopped {
			s.GPU = evt.Data().(*GPUUtil)
			s.Status = SessionStopping
			s.resetReadyFlags()
			s.setReadyFlag(domain.SessionStopExpects, domain.SessionGPUReady)
			return domain.NoSessionEvent, nil
		} else if evt.Name() == EventCPUStopped {
			s.CPU = evt.Data().(*CPUUtil)
			s.Status = SessionStopping
			s.resetReadyFlags()
			s.setReadyFlag(domain.SessionStopExpects, domain.SessionCPUReady)
			return domain.NoSessionEvent, nil
		} else if evt.Name() == EventMemoryStopped {
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp())
			// TODO: ignore memory events during stopping, for now.
			return domain.NoSessionEvent, nil
		}
		return domain.NoSessionEvent, Errorf(domain.ErrUnexpectedSessionStTrans, "SessionIdle on %v", evt)
	case SessionTraining:
		if evt.Name() == EventGPUDeactivated {
			s.GPU = evt.Data().(*GPUUtil)
			s.Status = SessionIdle
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp()) // Update memory reading.
			}
			return []domain.SessionEvent{domain.EventSessionTrainingEnded}, nil
		} else if evt.Name() == EventCPUActivated || evt.Name() == EventCPUDeactivated {
			break
		} else if evt.Name() == EventCPUStopped {
			// Handling the sepecial case that the session/pod continues after the end of the trace,
			// where CPU stop along with GPU deactivation and GPU may deactivated at a later time.
			s.CPU = evt.Data().(*CPUUtil)
			s.pending = append(s.pending, evt)
			return domain.NoSessionEvent, domain.ErrEventPending
		} else if evt.Name() == EventMemoryStopped {
			s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp())
			// TODO: ignore memory events during stopping, for now.
			break
		}
		// else if evt.Name() == EventGpuUpdateUtil {
		// 	s.GPU = evt.Data().(*GPUUtil)

		// 	return domain.NoSessionEvent, nil
		// 	// return []domain.SessionEvent{EventSessionUpdateGpuUtil}, nil
		// }
		return domain.NoSessionEvent, Errorf(domain.ErrUnexpectedSessionStTrans, "SessionTraining on %v", evt)
	case SessionStopping:
		if evt.Name() == EventGPUStopped {
			s.GPU = evt.Data().(*GPUUtil)
			s.setReadyFlag(domain.SessionStopExpects, domain.SessionGPUReady)
		} else if evt.Name() == EventCPUStopped {
			s.CPU = evt.Data().(*CPUUtil)
			s.setReadyFlag(domain.SessionStopExpects, domain.SessionCPUReady)
		} else if evt.Name() == EventGPUStarted && s.GPU.Status == GPUStopped {
			// Deal with situations like regaining GPU readings after missing for a while.
			s.GPU = evt.Data().(*GPUUtil)
			s.Status = SessionIdle
			break
		} else if evt.Name() == EventCPUDeactivated || evt.Name() == EventMemoryStopped {
			// Ignore irrelevant events.
			break
		} else {
			return domain.NoSessionEvent, Errorf(domain.ErrUnexpectedSessionStTrans, "SessionStopping on event %s", evt.Name())
		}

		// Check if session is ready to stop.
		if s.StatusReadyFlags == domain.SessionStopExpects {
			s.Status = SessionStopped
			if s.MemoryQuerier != nil {
				s.Memory = s.MemoryQuerier.Lookup(evt.Timestamp()) // Update memory reading.
			}
			return []domain.SessionEvent{domain.EventSessionStopped}, nil
		}
		return domain.NoSessionEvent, nil
	case SessionStopped:
		if evt.Name() == EventMemoryStopped {
			return domain.NoSessionEvent, nil
		}
		return domain.NoSessionEvent, Errorf(domain.ErrUnexpectedSessionStTrans, "SessionStopped on %v", evt)
	}

	return domain.NoSessionEvent, nil
}

func (s *Session) Snapshot() *Session {
	ss := *s
	return &ss
}
