package event_queue

import (
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

// MainEventQueue is a queue of queues, essentially. The elements are SessionEventQueue instances, which are
// sorted according to the next event for the associated Session contained within each SessionEventQueue.
type MainEventQueue []*SessionEventQueue

func (h MainEventQueue) Len() int {
	return len(h)
}

func (h MainEventQueue) Less(i, j int) bool {
	// We want to ensure that TrainingEnded events are processed before SessionStopped events.
	// So, if the event at localIndex i is a TrainingEnded event while the event at localIndex j is a SessionStopped event,
	// then the event at localIndex i should be processed first.
	if h[i].NextEventTimestamp().Equal(h[j].NextEventTimestamp()) {
		if h[i].NextEventName() == domain.EventSessionTrainingEnded && h[j].NextEventName() == domain.EventSessionStopped {
			return true
		} else if h[j].NextEventName() == domain.EventSessionTrainingEnded && h[i].NextEventName() == domain.EventSessionStopped {
			return false
		}

		return h[i].NextEventGlobalEventIndex() < h[j].NextEventGlobalEventIndex()
	}

	return h[i].NextEventTimestamp().Before(h[j].NextEventTimestamp())
}

func (h MainEventQueue) Swap(i, j int) {
	// log.Printf("Swap %d, %d (%v, %v) of %d", i, j, h[i], h[j], len(h))
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
