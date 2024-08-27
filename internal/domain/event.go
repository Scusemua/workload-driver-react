package domain

import (
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
	// If the EventSource is the last one, should the simulation continue?
	// For Drivers, the answer is yes. For non-Drivers, it may depend.
	// For example, if the BufferedService is the last one, then the simulation
	// should end once all buffered events have been retriggered.
	IsLastShouldContinue() bool
	// Called in pre-run mode when the Synthesizer encounters a training-started event.
	// Sets the value in the latest training max slot to 0.
	TrainingStarted(string)
	// Called in pre-run mode when the Synthesizer encounters a training-stopped event.
	// Prepares the next slot in the training maxes by appending to the list a new value of -1.
	TrainingEnded(string)
}

type EventConsumer interface {
	// Give an event to the EventConsumer so that it may be processed.
	SubmitEvent(Event)

	// Get the channel used to tell the EventConsumer that an error has occurred
	// in the generator portion of the workload simulator/driver.
	GetErrorChan() chan<- error

	// Get the channel used by the EventConsumer to signal that a workload has completed.
	WorkloadExecutionCompleteChan() chan interface{}

	// Get the channel used to notify the EventConsumer that the generator(s) have finished generating events.
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
	if h[i].AdjustedTimestamp().Equal(h[j].AdjustedTimestamp()) {
		// We want to ensure that TrainingEnded events are processed before SessionStopped events.
		// So, if event i is TrainingEnded and event j is SessionStopped, then event i should be processed first.
		if h[i].Name() == EventSessionTrainingEnded && h[j].Name() == EventSessionStopped {
			return true
		} else if h[j].Name() == EventSessionTrainingEnded && h[i].Name() == EventSessionStopped {
			return false
		}
	}

	return h[i].AdjustedTimestamp().Before(h[j].AdjustedTimestamp())
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
