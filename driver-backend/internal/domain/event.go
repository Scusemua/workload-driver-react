package domain

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	// EventSessionStarted triggered when the session is first seen.
	EventSessionStarted SessionEventName = "session-started"
	// EventSessionReady triggered when we have had sufficient info on resource specification and the session is ready to launch.
	EventSessionReady           SessionEventName = "session-ready"
	EventSessionTraining        SessionEventName = "training"
	EventSessionTrainingStarted SessionEventName = "training-started"
	EventSessionTrainingEnded   SessionEventName = "training-ended"
	EventSessionStopped         SessionEventName = "session-stopped"
	EventSessionUpdateGpuUtil   SessionEventName = "update-gpu-util"

	// EventInvalidName is a placeholder/default value that should not appear during normal operation.
	EventInvalidName SessionEventName = "invalid-name"

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
	OnEvent() <-chan *Event
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
	SubmitEvent(*Event)

	// GetErrorChan returns the channel used to tell the EventConsumer that an error has occurred
	// in the generator portion of the workload simulator/driver.
	GetErrorChan() chan<- error

	// WorkloadExecutionCompleteChan returns the channel used by the EventConsumer to signal that a workload has completed.
	WorkloadExecutionCompleteChan() chan interface{}

	// WorkloadEventGeneratorCompleteChan returns the channel used to notify the EventConsumer that the generator(s) have finished generating events.
	WorkloadEventGeneratorCompleteChan() chan interface{}

	// RegisterApproximateFinalTick is used to register what is the approximate final tick of the workload
	// after iterating over all sessions and all training events.
	RegisterApproximateFinalTick(int64)
}

type Event struct {
	EventSource         EventSource   `json:"-"`
	OriginalEventSource EventSource   `json:"-"`
	Data                interface{}   `json:"data"`
	SessionId           string        `json:"session_id"`
	Name                EventName     `json:"name"`
	Timestamp           time.Time     `json:"timestamp"`
	OriginalTimestamp   time.Time     `json:"originalTimestamp"`
	Duration            time.Duration `json:"duration"`
	EndTime             time.Time     `json:"end_time"`
	Delay               time.Duration `json:"delay"`
	ID                  string        `json:"id"`
	OrderSeq            int64         `json:"order_seq"` // OrderSeq is essentially Timestamp of event, but randomized to make behavior stochastic.
	HeapIndex           int           `json:"heap_index"`
	Enqueued            bool          `json:"enqueued"`

	// LocalIndex indicates the order in which the event was created relative to other events targeting the same Session.
	// The first event created for a session while have an LocalIndex of 0.
	// The last event created for a session while have an LocalIndex of N - 1, where N is the number of events created
	// for that Session.
	LocalIndex int `json:"local_index"`

	// GlobalIndex provides a global ordering for comparing all events with each other within a workload.
	GlobalIndex uint64 `json:"global_index"`

	// numTimesEnqueued atomically records the number of times that this Event has been enqueued.
	numTimesEnqueued atomic.Int32
}

func (e *Event) SetIndex(idx int) {
	e.HeapIndex = idx
}

func (e *Event) GetIndex() int {
	return e.HeapIndex
}

func (e *Event) Dequeued() {
	e.Enqueued = false
}

// RecordThatEventWasEnqueued records that the event was enqueued for processing.
// Events can be enqueued multiple times.
func (e *Event) RecordThatEventWasEnqueued() {
	e.numTimesEnqueued.Add(1)
	e.Enqueued = true
}

// GetNumTimesEnqueued returns the number of times that the Event has been enqueued.
func (e *Event) GetNumTimesEnqueued() int32 {
	return e.numTimesEnqueued.Load()
}

// WasEnqueuedMultipleTimes returns true if the event has been enqueued more than once.
func (e *Event) WasEnqueuedMultipleTimes() bool {
	return e.numTimesEnqueued.Load() > 1 // RecordThatEventWasEnqueued more than once?
}

// SessionSpecificEventIndex indicates the order in which the event was created relative to other events targeting
// the same Session.
// The first event created for a session while have an localIndex of 0.
// The last event created for a session while have an localIndex of N - 1, where N is the number of events created
// for that Session.
func (e *Event) SessionSpecificEventIndex() int { return e.LocalIndex }

// GlobalEventIndex provides a global ordering for comparing all events with each other within a workload.
func (e *Event) GlobalEventIndex() uint64 {
	return e.GlobalIndex
}

// PushTimestampBack adds the given time.Duration to the Event's timestamp. To be used when re-enqueuing the event to process it later.
func (e *Event) PushTimestampBack(amount time.Duration) {
	e.Timestamp = e.Timestamp.Add(amount)

	e.Delay = e.Delay + amount
}

// TotalDelay returns the total time.Duration that the event has been pushed back.
func (e *Event) TotalDelay() time.Duration {
	return e.Delay
}

func (e *Event) SessionID() string {
	data, ok := e.Data.(SessionMetadata)
	if !ok {
		return "N/A"
	}

	return data.GetPod()
}

func (e *Event) Id() string { return e.ID }

func (e *Event) String() string {
	return fmt.Sprintf("generator.Event[Name=%s,OriginalTimestamp=%v,Timestamp=%v,LocalIndex=%d,GlobalIndex=%d,,src=%v,orgSrc=%v,orderSeq=%d,data=%v]",
		e.Name, e.Timestamp, e.OriginalTimestamp, e.LocalIndex, e.GlobalIndex, e.EventSource, e.OriginalEventSource, e.OrderSeq, e.Data)
}

func (e *Event) StringJson() string {
	m, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}

	return string(m)
}
