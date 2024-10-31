package domain

import (
	"log"
	"time"
)

const (
	// EventSessionStarted triggered when the session is first seen.
	EventSessionStarted SessionEventName = "session-started"
	// EventSessionReady triggered when we have had sufficient info on resource specification and the session is ready to launch.
	EventSessionReady           SessionEventName = "session-ready"
	EventSessionTrainingStarted SessionEventName = "training-started"
	EventSessionTrainingEnded   SessionEventName = "training-ended"
	EventSessionStopped         SessionEventName = "session-stopped"
	EventSessionUpdateGpuUtil   SessionEventName = "update-gpu-util"

	EventWorkloadStarted  WorkloadEventName = "workload-started"
	EventWorkloadComplete WorkloadEventName = "workload-complete"
)

type WorkloadEventName string

func (evt WorkloadEventName) String() string {
	return string(evt)
}

type SessionEventName string

func (evt SessionEventName) String() string {
	return string(evt)
}

type EventName interface {
	String() string
}

type PodData interface {
	GetPod() string
}

type EventSource interface {
	OnEvent() <-chan Event
	String() string
	Id() int
	SetId(int)
	IsDriver() bool
	// IsLastShouldContinue returns a boolean with the following logic:
	// If the EventSource is the last one, should the simulation continue?
	// For Drivers, the answer is yes. For non-Drivers, it may depend.
	// For example, if the BufferedService is the last one, then the simulation
	// should end once all buffered events have been retriggered.
	IsLastShouldContinue() bool
	// TrainingStarted is called in pre-run mode when the Synthesizer encounters a training-started event.
	// Sets the value in the latest training max slot to 0.
	TrainingStarted(string)
	// TrainingEnded is called in pre-run mode when the Synthesizer encounters a training-stopped event.
	// Prepares the next slot in the training maxes by appending to the list a new value of -1.
	TrainingEnded(string)
}

type EventConsumer interface {
	// SubmitEvent delivers an event to the EventConsumer so that it may be processed.
	SubmitEvent(Event)

	// GetErrorChan returns the channel used to tell the EventConsumer that an error has occurred
	// in the generator portion of the workload simulator/driver.
	GetErrorChan() chan<- error

	// WorkloadExecutionCompleteChan returns the channel used by the EventConsumer to signal that a workload has completed.
	WorkloadExecutionCompleteChan() chan interface{}

	// WorkloadEventGeneratorCompleteChan returns the channel used to notify the EventConsumer that the generator(s) have finished generating events.
	WorkloadEventGeneratorCompleteChan() chan interface{}
}

type Event interface {
	EventSource() EventSource
	OriginalEventSource() EventSource
	Name() EventName
	Data() interface{}
	SessionID() string
	Timestamp() time.Time
	Id() string
	// SessionSpecificEventIndex indicates the order in which the event was created relative to other events targeting
	// the same Session.
	// The first event created for a session while have an index of 0.
	// The last event created for a session while have an index of N - 1, where N is the number of events created
	// for that Session.
	SessionSpecificEventIndex() int
	// GlobalEventIndex provides a global ordering for comparing all events with each other within a workload.
	GlobalEventIndex() uint64
	OrderSeq() int64 // OrderSeq is essentially timestamp of event, but randomized to make behavior stochastic.
	SetOrderSeq(int64)
	String() string
}

type EventBuff []Event

// sort.Interface implementations

func (buff EventBuff) Len() int {
	return len(buff)
}

func (buff EventBuff) Less(i, j int) bool {
	return buff[i].OrderSeq() < buff[j].OrderSeq()
}

func (buff EventBuff) Swap(i, j int) {
	buff[i], buff[j] = buff[j], buff[i]
}

// An EventHeap is a Heap implementation for elements of type EventHeapElementImpl.
type EventHeap []EventHeapElement

func (h EventHeap) Len() int {
	return len(h)
}

func (h EventHeap) Less(i, j int) bool {
	// We want to ensure that TrainingEnded events are processed before SessionStopped events.
	// So, if the event at index i is a TrainingEnded event while the event at index j is a SessionStopped event,
	// then the event at index i should be processed first.
	if h[i].OriginalTimestamp() == h[j].OriginalTimestamp() {
		if h[i].Name() == EventSessionTrainingEnded && h[j].Name() == EventSessionStopped {
			// Sanity check -- if we find a "session-stopped" and "training-ended" event targeting the same session,
			// then we double-check that their "event indices" are consistent with the order that they should be
			// processed. "training-ended" events should always be processed before "session-stopped" events.
			if h[i].SessionSpecificEventIndex() /* training-ended */ > h[j].SessionSpecificEventIndex() /* session-stopped */ && h[i].SessionID() == h[j].SessionID() {
				// We expect the event index of the training-ended event to be less than that of the session-stopped
				// event, since the training-ended event should have been created prior to the session-stopped event.
				log.Fatalf("Event indices do not reflect correct ordering of events. "+
					"TrainingEnded: %s. SessionStopped: %s.", h[i].String(), h[j].String())
			}

			return true
		} else if h[j].Name() == EventSessionTrainingEnded && h[i].Name() == EventSessionStopped {
			// Sanity check -- if we find a "session-stopped" and "training-ended" event targeting the same session,
			// then we double-check that their "event indices" are consistent with the order that they should be
			// processed. "training-ended" events should always be processed before "session-stopped" events.
			if h[j].SessionSpecificEventIndex() /* training-ended */ > h[i].SessionSpecificEventIndex() /* session-stopped */ && h[i].SessionID() == h[j].SessionID() {
				// We expect the event index of the training-ended event to be less than that of the session-stopped
				// event, since the training-ended event should have been created prior to the session-stopped event.
				log.Fatalf("Event indices do not reflect correct ordering of events. "+
					"TrainingEnded: %s. SessionStopped: %s.", h[j].String(), h[i].String())
			}

			return false
		}

		// Defer to the order in which the events were created to resolve the tie.
		// TODO: In theory, this would also resolve the above issue...
		return h[i].SessionSpecificEventIndex() < h[j].SessionSpecificEventIndex()
	}

	return h[i].OriginalTimestamp().Before(h[j].OriginalTimestamp())
}

func (h EventHeap) Swap(i, j int) {
	// log.Printf("Swap %d, %d (%v, %v) of %d", i, j, h[i], h[j], len(h))
	h[i].SetIndex(j)
	h[j].SetIndex(i)
	h[i], h[j] = h[j], h[i]
}

func (h *EventHeap) Push(x interface{}) {
	x.(EventHeapElement).SetIndex(len(*h))
	*h = append(*h, x.(EventHeapElement))
}

func (h *EventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h EventHeap) Peek() EventHeapElement {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}

type SimpleEventHeap []Event

func (h SimpleEventHeap) Len() int {
	return len(h)
}

func (h SimpleEventHeap) Less(i, j int) bool {
	return h[i].Timestamp().Before(h[j].Timestamp())
}

func (h SimpleEventHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *SimpleEventHeap) Push(x interface{}) {
	*h = append(*h, x.(Event))
}

func (h *SimpleEventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h SimpleEventHeap) Peek() Event {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
