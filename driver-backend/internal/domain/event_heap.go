package domain

import (
	"log"
)

type EventHeap []*Event

func (h EventHeap) Len() int {
	return len(h)
}

func (h EventHeap) Less(i, j int) bool {
	// We want to ensure that TrainingEnded events are processed before SessionStopped events.
	// So, if the event at localIndex i is a TrainingEnded event while the event at localIndex j is a SessionStopped event,
	// then the event at localIndex i should be processed first.
	if h[i].Timestamp.Equal(h[j].Timestamp) {
		// SessionReady events should always go first.
		if h[i].Name == EventSessionReady {
			return true
		}

		// SessionReady events should always go first.
		if h[j].Name == EventSessionReady {
			return false
		}

		if h[i].Name == EventSessionTrainingEnded && h[j].Name == EventSessionStopped {
			// Sanity check -- if we find a "session-stopped" and "training-ended" event targeting the same session,
			// then we double-check that their "event indices" are consistent with the order that they should be
			// processed. "training-ended" events should always be processed before "session-stopped" events.
			if h[i].SessionSpecificEventIndex() /* training-ended */ > h[j].SessionSpecificEventIndex() /* session-stopped */ && h[i].SessionID() == h[j].SessionID() && !h[i].WasEnqueuedMultipleTimes() && !h[j].WasEnqueuedMultipleTimes() {
				// We expect the global localIndex of the training-ended event to be less than that of the session-stopped
				// event, since the training-ended event should have been created prior to the session-stopped event.
				log.Printf("[FATAL] Global event indices do not reflect correct ordering of events. "+
					"TrainingEnded: %s. SessionStopped: %s.\n", h[i].String(), h[j].String())
				panic("encountered inconsistent global event indices")
			}

			return true
		} else if h[j].Name == EventSessionTrainingEnded && h[i].Name == EventSessionStopped {
			// Sanity check -- if we find a "session-stopped" and "training-ended" event targeting the same session,
			// then we double-check that their "event indices" are consistent with the order that they should be
			// processed. "training-ended" events should always be processed before "session-stopped" events.
			if h[j].SessionSpecificEventIndex() /* training-ended */ > h[i].SessionSpecificEventIndex() /* session-stopped */ && h[i].SessionID() == h[j].SessionID() && !h[i].WasEnqueuedMultipleTimes() && !h[j].WasEnqueuedMultipleTimes() {
				// We expect the global localIndex of the training-ended event to be less than that of the session-stopped
				// event, since the training-ended event should have been created prior to the session-stopped event.
				log.Printf("[FATAL] Global event indices do not reflect correct ordering of events. "+
					"TrainingEnded: %s. SessionStopped: %s.\n", h[j].String(), h[i].String())
				panic("encountered inconsistent global event indices")
			}

			return false
		}

		return h[i].GlobalEventIndex() < h[j].GlobalEventIndex()
	}

	return h[i].Timestamp.Before(h[j].Timestamp)
}

func (h EventHeap) Swap(i, j int) {
	// log.Printf("Swap %d, %d (%v, %v) of %d", i, j, h[i], h[j], len(h))
	h[i].SetIndex(j)
	h[j].SetIndex(i)
	h[i], h[j] = h[j], h[i]
}

func (h *EventHeap) Push(x interface{}) {
	x.(*Event).SetIndex(len(*h))
	*h = append(*h, x.(*Event))
}

func (h *EventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h EventHeap) Peek() *Event {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
