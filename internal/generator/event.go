package generator

import (
	"errors"
	"fmt"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

const (
	// EventSessionStarted triggered when the session is first seen.
	EventSessionStarted SessionEvent = "started"
	// EventSessionReady triggered when we have had sufficient info on resource specification and the session is ready to launch.
	EventSessionReady           SessionEvent = "ready"
	EventSessionTrainingStarted SessionEvent = "training-started"
	EventSessionTrainingEnded   SessionEvent = "training-ended"
	EventSessionStopped         SessionEvent = "stopped"
	EventSessionUpdateGpuUtil   SessionEvent = "update-gpu-util"

	SessionCPUReady = 0x01
	SessionGPUReady = 0x02
	SessionMemReady = 0x04
)

var (
	ErrUnexpectedSessionState   = errors.New("unexpected session state")
	ErrUnexpectedSessionStTrans = errors.New("unexpected session state transition")

	ErrNoEventQueue = errors.New("no corresponding event queue for given Session")
	ErrNoMoreEvents = errors.New("there are no more events for the specified Session")

	NoSessionEvent      []SessionEvent = nil
	SessionReadyExpects                = SessionCPUReady | SessionGPUReady
	SessionStopExpects                 = SessionCPUReady | SessionGPUReady
	ErrEventPending     error          = errors.New("event pending")

	NilTimestamp = time.Time{}
)

type SessionEvent string

func (evt SessionEvent) String() string {
	return string(evt)
}

// type EventHandler interface {
// 	HandleEvent(*Event)
// }

type eventImpl struct {
	eventSource         domain.EventSource
	originalEventSource domain.EventSource
	name                domain.EventName
	data                interface{}
	timestamp           time.Time
	id                  string
	orderSeq            int64 // OrderSeq is essentially timestamp of event, but randomized to make behavior stochastic.
}

func (e *eventImpl) EventSource() domain.EventSource { return e.eventSource }

func (e *eventImpl) OriginalEventSource() domain.EventSource { return e.originalEventSource }

func (e *eventImpl) Name() domain.EventName { return e.name }

func (e *eventImpl) Data() interface{} { return e.data }

func (e *eventImpl) SessionID() string {
	data, ok := e.data.(*SessionMeta)
	if !ok {
		return "N/A"
	}

	return data.Pod
}

func (e *eventImpl) Timestamp() time.Time { return e.timestamp }

func (e *eventImpl) Id() string { return e.id }

// OrderSeq is essentially timestamp of event, but randomized to make behavior stochastic.
func (e *eventImpl) OrderSeq() int64 {
	return e.orderSeq
}

// OrderSeq is essentially timestamp of event, but randomized to make behavior stochastic.
func (e *eventImpl) SetOrderSeq(seq int64) {
	e.orderSeq = seq
}

// func (e *Event) SetIndex(idx int) {
// 	e.index = idx
// }

// func (e *Event) GetIndex() int {
// 	return e.index
// }

func (e *eventImpl) String() string {
	switch e.Name() {
	case EventNoMore:
		if e.EventSource() == nil {
			return e.Name().String()
		} else {
			return fmt.Sprintf("generator.Event[Name=%s -- src=%v, orgSrc=%v, orderSeq=%d]", e.Name(), e.EventSource(), e.OriginalEventSource(), e.OrderSeq())
		}
	case EventError:
		return fmt.Sprintf("generator.Event[Name=%s -- src=%v, orgSrc=%v, orderSeq=%d, data=%v]", e.Name(), e.EventSource(), e.OriginalEventSource(), e.OrderSeq(), e.Data())
	}
	return fmt.Sprintf("generator.Event[Timestamp=%v, Name=%s -- src=%v, orgSrc=%v, orderSeq=%d, data=%v]", e.Timestamp(), e.Name(), e.EventSource(), e.OriginalEventSource(), e.OrderSeq(), e.Data())
}

type EventBuff []domain.Event

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
type EventHeap []domain.EventHeapElement

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
	x.(domain.EventHeapElement).SetIndex(len(*h))
	*h = append(*h, x.(domain.EventHeapElement))
}

func (h *EventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h EventHeap) Peek() domain.EventHeapElement {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}

type SimpleEventHeap []domain.Event

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
	*h = append(*h, x.(domain.Event))
}

func (h *SimpleEventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h SimpleEventHeap) Peek() domain.Event {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
