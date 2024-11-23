package event_queue

import (
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

// MainEventQueue is a queue of queues, essentially. The elements are SessionEventQueue instances, which are
// sorted according to the next event for the associated Session contained within each SessionEventQueue.
//
// Empty queues are always positioned after non-empty queues.
type MainEventQueue []*SessionEventQueue

func (h MainEventQueue) Len() int {
	return len(h)
}

func (h MainEventQueue) Less(i, j int) bool {
	// If the i-th queue is non-empty while the j-th queue is empty, then the i-th queue should come first.
	if h[i].IsNonEmpty() && h[j].IsEmpty() {
		//fmt.Printf("SessionEventQueue %d (\"%s\") is non-empty while SessionEventQueue %d (\"%s\") is empty.\n",
		//	i, h[i].SessionId, j, h[j].SessionId)
		return true
	}

	// If the i-th queue is empty while the j-th queue is non-empty, then the j-th queue should come first.
	if h[i].IsEmpty() && h[j].IsNonEmpty() {
		//fmt.Printf("SessionEventQueue %d (\"%s\") is empty while SessionEventQueue %d (\"%s\") is non-empty.\n",
		//	i, h[i].SessionId, j, h[j].SessionId)
		return false
	}

	// If both queues are empty, then just treat them as if they're equal, so return false.
	if h[i].IsEmpty() && h[j].IsEmpty() {
		//fmt.Printf("Both SessionEventQueue %d (\"%s\") and SessionEventQueue %d (\"%s\") are empty.\n",
		//	i, h[i].SessionId, j, h[j].SessionId)
		return false
	}

	// We want to ensure that TrainingEnded events are processed before SessionStopped events.
	// So, if the event at localIndex i is a TrainingEnded event while the event at localIndex j is a SessionStopped event,
	// then the event at localIndex i should be processed first.
	timestampI, ok := h[i].NextEventTimestamp()
	if !ok {
		panic("unexpected empty")
	}

	timestampJ, ok := h[j].NextEventTimestamp()
	if !ok {
		panic("unexpected empty")
	}

	if timestampI.Equal(timestampJ) {
		eventNameI, ok := h[i].NextEventName()
		if !ok {
			panic("unexpected empty")
		}

		eventNameJ, ok := h[j].NextEventName()
		if !ok {
			panic("unexpected empty")
		}

		// "session-ready" events should always go first within the same tick.
		if eventNameI == domain.EventSessionReady {
			return true
		}

		// "session-ready" events should always go first within the same tick.
		if eventNameJ == domain.EventSessionReady {
			return false
		}

		if eventNameI == domain.EventSessionTrainingEnded && eventNameJ == domain.EventSessionStopped {
			return true
		} else if eventNameJ == domain.EventSessionTrainingEnded && eventNameI == domain.EventSessionStopped {
			return false
		}

		globalIndexI, ok := h[i].NextEventGlobalEventIndex()
		if !ok {
			panic("unexpected empty")
		}

		globalIndexJ, ok := h[j].NextEventGlobalEventIndex()
		if !ok {
			panic("unexpected empty")
		}

		return globalIndexI < globalIndexJ
	}

	//fmt.Printf("Session \"%s\" [%v] < Session \"%s\" [%v]: %v\n", h[i].SessionId, timestampI, h[j].SessionId, timestampJ, timestampI.Before(timestampJ))
	return timestampI.Before(timestampJ)
}

func (h MainEventQueue) Swap(i, j int) {
	//fmt.Printf("Swap %d, %d (%v, %v) of %d\n", i, j, h[i], h[j], len(h))
	h[i].SetIndex(j)
	//fmt.Printf("[SWAP 1/2] Set index of SessEvtQ \"%s\" to %d.\n", h[i].SessionId, j)
	h[j].SetIndex(i)
	//fmt.Printf("[SWAP 2/2] Set index of SessEvtQ \"%s\" to %d.\n", h[j].SessionId, i)
	h[i], h[j] = h[j], h[i]
}

func (h *MainEventQueue) Push(x interface{}) {
	x.(*SessionEventQueue).SetIndex(len(*h))
	*h = append(*h, x.(*SessionEventQueue))
}

func (h *MainEventQueue) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h MainEventQueue) Peek() *SessionEventQueue {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
