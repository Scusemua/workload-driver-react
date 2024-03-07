package generator

import (
	"fmt"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

var (
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
}

func (e *eventImpl) EventSource() domain.EventSource { return e.eventSource }

func (e *eventImpl) OriginalEventSource() domain.EventSource { return e.originalEventSource }

func (e *eventImpl) Name() domain.EventName { return e.name }

func (e *eventImpl) Data() interface{} { return e.data }

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
