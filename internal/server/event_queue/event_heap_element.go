package event_queue

import (
	"fmt"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
)

// This is a wrapper around a *generator.Event.
// We add a boolean flag indicating whether the event is enqueued or has been removed from the heap (and given to the Cluster for processing).
// We also add an index field so elements can maintain their position in the heap, which makes it easier to update their position if their timestamp changes during the simulation.
// Finally, we maintain an "adjusted timestamp" which is computed by adding the delays of the Session associated with the underlying event to the event's original timestamp.
type eventHeapElementImpl struct {
	domain.Event                // The underlying event that we're wrapping.
	enqueued          bool      // Indicates whether or not the element is actually enqueued in the main event heap of the EventQueueServiceImpl. Used to determine whether its index should be updated when a change occurs to the associated session's `totalDelay` field.
	idx               int       // Index in the heap.
	adjustedTimestamp time.Time // The timestamp of the event after being adjusted by the associated Session's totalDelay field. This is computed by adding the delays of the Session associated with the underlying event to the event's original timestamp.
}

// Recompute the eventHeapElementImpl's adjustedTimestamp field based on the provided Total Delay value.
func (e *eventHeapElementImpl) RecalculateTimestamp(totalDelay time.Duration) {
	e.adjustedTimestamp = e.OriginalTimestamp().Add(totalDelay)
}

// Recompute the eventHeapElementImpl's adjustedTimestamp field based on the provided Total Delay value.
func (e *eventHeapElementImpl) Name() domain.EventName {
	return e.Event.Name()
}

// Recompute the eventHeapElementImpl's adjustedTimestamp field based on the provided Total Delay value.
func (e *eventHeapElementImpl) Id() string {
	return e.Event.Id()
}

// Update the eventHeapElementImpl's index in its heap. This just changes the field; it doesn't interact with the heap that contains the eventHeapElementImpl.
func (e *eventHeapElementImpl) SetIndex(idx int) {
	e.idx = idx
}

// Return the eventHeapElementImpl's index in its heap.
func (e *eventHeapElementImpl) GetIndex() int {
	return e.idx
}

// Return the original timestamp of the underlying event.
func (e *eventHeapElementImpl) OriginalTimestamp() time.Time {
	return e.Timestamp()
}

// Return the timestamp of the event after being adjusted by the associated Session's totalDelay field.
func (e *eventHeapElementImpl) AdjustedTimestamp() time.Time {
	return e.adjustedTimestamp
}

// Return the ID of the associated Session.
func (e *eventHeapElementImpl) SessionID() string {
	return e.Event.Data().(*generator.Session).Pod
}

// Return the underlying *generator.Event struct.
func (e *eventHeapElementImpl) GetEvent() domain.Event {
	return e.Event
}

// ToString for eventHeapElementImpl.
func (e *eventHeapElementImpl) String() string {
	sessionId := e.Event.Data().(*generator.Session).Pod
	return fmt.Sprintf("eventHeapElementImpl[idx=%d, adjustedTimestamp=%v, eventID=%s, eventName=%v, session=%s]", e.idx, e.adjustedTimestamp, e.Id(), e.Name(), sessionId)
}

func (e *eventHeapElementImpl) Data() interface{} {
	return e.Event.Data()
}

func (e *eventHeapElementImpl) Enqueued() bool {
	return e.enqueued
}

func (e *eventHeapElementImpl) SetEnqueued(enqueued bool) {
	e.enqueued = enqueued
}
