package generator

import (
	"fmt"
	"time"
)

var (
	NilTimestamp = time.Time{}
)

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
	SubmitEvent(*Event) // Give an event to the EventConsumer so that it may be processed.
}

// type EventHandler interface {
// 	HandleEvent(*Event)
// }

type Event struct {
	EventSource         EventSource
	OriginalEventSource EventSource
	Name                EventName
	Data                interface{}
	Timestamp           time.Time
	Id                  string

	// index int

	// OrderSeq is essentially timestamp of event, but randomized to make behavior stochastic.
	OrderSeq int64
}

// func (e *Event) SetIndex(idx int) {
// 	e.index = idx
// }

// func (e *Event) GetIndex() int {
// 	return e.index
// }

func (e *Event) String() string {
	switch e.Name {
	case EventNoMore:
		if e.EventSource == nil {
			return e.Name.String()
		} else {
			return fmt.Sprintf("generator.Event[Name=%s -- src=%v, orgSrc=%v, orderSeq=%d]", e.Name, e.EventSource, e.OriginalEventSource, e.OrderSeq)
		}
	case EventError:
		return fmt.Sprintf("generator.Event[Name=%s -- src=%v, orgSrc=%v, orderSeq=%d, data=%v]", e.Name, e.EventSource, e.OriginalEventSource, e.OrderSeq, e.Data)
	}
	if e.EventSource == nil {
		return fmt.Sprintf("generator.Event[Timestamp=%v, Name=%s -- src=N/A, orgSrc=N/A, orderSeq=%d, data=%v]", e.Timestamp, e.Name, e.OrderSeq, e.Data)
	}
	return fmt.Sprintf("generator.Event[Timestamp=%v, Name=%s -- src=%v, orgSrc=%v, orderSeq=%d, data=%v]", e.Timestamp, e.Name, e.EventSource, e.OriginalEventSource, e.OrderSeq, e.Data)
}

type EventBuff []*Event

// sort.Interface implementations
func (buff EventBuff) Len() int {
	return len(buff)
}

func (buff EventBuff) Less(i, j int) bool {
	return buff[i].OrderSeq < buff[j].OrderSeq
}

func (buff EventBuff) Swap(i, j int) {
	buff[i], buff[j] = buff[j], buff[i]
}
