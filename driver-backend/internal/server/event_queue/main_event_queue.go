package event_queue

import (
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

// MainEventQueue is a queue of queues, essentially. The elements are SessionEventQueue instances, which are
// sorted according to the next event for the associated Session contained within each SessionEventQueue.
//
// Empty queues are always positioned after non-empty queues.
//
// Empty queues sorted amongst themselves according to their Session ID.
type MainEventQueue []*SessionEventQueue

func (h MainEventQueue) Len() int {
	return len(h)
}

func (h MainEventQueue) Less(i, j int) bool {
	// If the i-th queue is non-empty while the j-th queue is empty, then the i-th queue should come first.
	if h[i].IsNonEmpty() && h[j].IsEmpty() {
		fmt.Printf("SessionEventQueue %d (\"%s\") is non-empty while SessionEventQueue %d (\"%s\") is empty.\n",
			i, h[i].SessionId, j, h[j].SessionId)
		return true
	}

	// If the i-th queue is empty while the j-th queue is non-empty, then the j-th queue should come first.
	if h[i].IsEmpty() && h[j].IsNonEmpty() {
		fmt.Printf("SessionEventQueue %d (\"%s\") is empty while SessionEventQueue %d (\"%s\") is non-empty.\n",
			i, h[i].SessionId, j, h[j].SessionId)
		return false
	}

	// If both queues are empty, then sort according to the associated session IDs.
	if h[i].IsEmpty() && h[j].IsEmpty() {
		fmt.Printf("Comparing SessionID of SessionEventQueue %d (\"%s\") and SessionEventQueue %d (\"%s\").\n",
			i, h[i].SessionId, j, h[j].SessionId)
		fmt.Printf("h[%d].SessionId (\"%s\") < h[%d].SessionId (\"%s\"): %v\n",
			i, h[i].SessionId, j, h[j].SessionId, h[i].SessionId < h[j].SessionId)
		return h[i].SessionId < h[j].SessionId
	}

	// We want to ensure that TrainingEnded events are processed before SessionStopped events.
	// So, if the event at localIndex i is a TrainingEnded event while the event at localIndex j is a SessionStopped event,
	// then the event at localIndex i should be processed first.
	if h[i].NextEventTimestamp().Equal(h[j].NextEventTimestamp()) {
		// Within the same tick, session ready events should always be first.
		if h[i].NextEventName() == domain.EventSessionReady {
			return true
		}

		// Within the same tick, session ready events should always be first.
		if h[j].NextEventName() == domain.EventSessionReady {
			return false
		}

		// Within the same tick, we want to process "training-ended" events before "session-stopped" events.
		if h[i].NextEventName() == domain.EventSessionTrainingEnded && h[j].NextEventName() == domain.EventSessionStopped {
			return true
		} else if h[j].NextEventName() == domain.EventSessionTrainingEnded && h[i].NextEventName() == domain.EventSessionStopped {
			return false
		}

		// Defer to the global event index.
		return h[i].NextEventGlobalEventIndex() < h[j].NextEventGlobalEventIndex()
	}

	return h[i].NextEventTimestamp().Before(h[j].NextEventTimestamp())
}

func (h MainEventQueue) Swap(i, j int) {
	h[i].SetIndex(j)
	h[j].SetIndex(i)
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
