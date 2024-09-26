package generator

import (
	"errors"
	"fmt"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

const (
	SessionCPUReady = 0x01
	SessionGPUReady = 0x02
	SessionMemReady = 0x04
)

var (
	ErrUnexpectedSessionState   = errors.New("unexpected session state")
	ErrUnexpectedSessionStTrans = errors.New("unexpected session state transition")

	NoSessionEvent      []domain.SessionEventName = nil
	SessionReadyExpects                           = SessionCPUReady | SessionGPUReady
	SessionStopExpects                            = SessionCPUReady | SessionGPUReady
	ErrEventPending     error                     = errors.New("event pending")

	NilTimestamp = time.Time{}
)

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

	// localIndex indicates the order in which the event was created relative to other events targeting the same Session.
	// The first event created for a session while have an localIndex of 0.
	// The last event created for a session while have an localIndex of N - 1, where N is the number of events created
	// for that Session.
	localIndex int

	// globalIndex provides a global ordering for comparing all events with each other within a workload.
	globalIndex uint64
}

func (e *eventImpl) EventSource() domain.EventSource { return e.eventSource }

func (e *eventImpl) OriginalEventSource() domain.EventSource { return e.originalEventSource }

func (e *eventImpl) Name() domain.EventName { return e.name }

func (e *eventImpl) Data() interface{} { return e.data }

// SessionSpecificEventIndex indicates the order in which the event was created relative to other events targeting
// the same Session.
// The first event created for a session while have an localIndex of 0.
// The last event created for a session while have an localIndex of N - 1, where N is the number of events created
// for that Session.
func (e *eventImpl) SessionSpecificEventIndex() int { return e.localIndex }

// GlobalEventIndex provides a global ordering for comparing all events with each other within a workload.
func (e *eventImpl) GlobalEventIndex() uint64 {
	return e.globalIndex
}

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

// SetOrderSeq sets the OrderSeq field of the target eventImpl struct.
func (e *eventImpl) SetOrderSeq(seq int64) {
	e.orderSeq = seq
}

func (e *eventImpl) String() string {
	switch e.name {
	case EventNoMore:
		if e.EventSource() == nil {
			return e.name.String()
		} else {
			return fmt.Sprintf("generator.Event[Name=%s,LocalIndex=%d,GlobalIndex=%d,src=%v,orgSrc=%v,orderSeq=%d]",
				e.name, e.localIndex, e.globalIndex, e.eventSource, e.originalEventSource, e.orderSeq)
		}
	case EventError:
		return fmt.Sprintf("generator.Event[Name=%s,LocalIndex=%d,GlobalIndex=%d,,src=%v,orgSrc=%v,orderSeq=%d,data=%v]",
			e.name, e.localIndex, e.globalIndex, e.eventSource, e.originalEventSource, e.orderSeq, e.data)
	}
	return fmt.Sprintf("generator.Event[Timestamp=%v,Name=%s,LocalIndex=%d,GlobalIndex=%d,,src=%v,orgSrc=%v,orderSeq=%d,data=%v]",
		e.Timestamp(), e.name, e.localIndex, e.globalIndex, e.eventSource, e.originalEventSource, e.orderSeq, e.data)
}

// func (e *Event) SetIndex(heapIndex int) {
// 	e.localIndex = heapIndex
// }

// func (e *Event) GetIndex() int {
// 	return e.localIndex
// }
