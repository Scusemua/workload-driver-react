package event_queue

import (
	"fmt"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

// This is a wrapper around a *generator.Event.
// We add a boolean flag indicating whether the event is enqueued or has been removed from the heap (and given to the Cluster for processing).
// We also add an index field so elements can maintain their position in the heap, which makes it easier to update their position if their timestamp changes during the simulation.
// Finally, we maintain an "adjusted timestamp" which is computed by adding the delays of the Session associated with the underlying event to the event's original timestamp.
type eventHeapElementImpl struct {
	domain.Event      // The underlying event that we're wrapping.
	enqueued     bool // Indicates whether the element is actually enqueued in the main event heap of the EventQueueServiceImpl. Used to determine whether its index should be updated when a change occurs to the associated session's `totalDelay` field.

	// heapIndex is the index of the eventHeapElementImpl within its containing internalEventHeap.
	// heapIndex is distinct from the domain.Event's SessionSpecificEventIndex and GlobalEventIndex methods.
	// heapIndex is dynamic and will change depending on what position the eventHeapElementImpl is in
	// within the containing internalEventHeap, whereas SessionSpecificEventIndex and GlobalEventIndex are/return
	// static values that are set when the underlying domain.Event is first created.
	heapIndex int
}

func (e *eventHeapElementImpl) Name() domain.EventName {
	return e.Event.Name()
}

func (e *eventHeapElementImpl) Id() string {
	return e.Event.Id()
}

// SetIndex updates the eventHeapElementImpl's "heap index" -- that is, the eventHeapElementImpl's index within its heap.
// SetIndex just changes the field; it doesn't interact with the heap that contains the eventHeapElementImpl.
// The heap index is dynamic and will change depending on what position the eventHeapElementImpl is in
// within the containing EventHeap, whereas SessionSpecificEventIndex and GlobalEventIndex are/return
// static values that are set when the underlying domain.Event is first created.
func (e *eventHeapElementImpl) SetIndex(idx int) {
	e.heapIndex = idx
}

// GetIndex returns the eventHeapElementImpl's "heap index" -- that is, the eventHeapElementImpl's index within its heap.
// The heap index is dynamic and will change depending on what position the eventHeapElementImpl is in
// within the containing EventHeap, whereas SessionSpecificEventIndex and GlobalEventIndex are/return
// static values that are set when the underlying domain.Event is first created.
func (e *eventHeapElementImpl) GetIndex() int {
	return e.heapIndex
}

// OriginalTimestamp returns the original timestamp of the underlying event.
func (e *eventHeapElementImpl) OriginalTimestamp() time.Time {
	return e.Event.OriginalTimestamp()
}

// SessionID returns the ID of the associated Session.
func (e *eventHeapElementImpl) SessionID() string {
	return e.Event.Data().(domain.SessionMetadata).GetPod()
}

// GetEvent returns the underlying *generator.Event struct.
func (e *eventHeapElementImpl) GetEvent() domain.Event {
	return e.Event
}

// ToString for eventHeapElementImpl.
func (e *eventHeapElementImpl) String() string {
	sessionId := e.Event.Data().(domain.SessionMetadata).GetPod()
	return fmt.Sprintf("eventHeapElementImpl[heapIndex=%d, timestamp=%v, eventID=%s, eventName=%v, eventIndex=%d, session=%s]", e.heapIndex, e.Timestamp(), e.Id(), e.Name(), e.SessionSpecificEventIndex(), sessionId)
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
