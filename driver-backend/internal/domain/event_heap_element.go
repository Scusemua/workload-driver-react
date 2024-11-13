package domain

import (
	"time"
)

// EventHeapElement ...
type EventHeapElement interface {
	// Name of the event.
	Name() EventName
	// SetIndex updates the EventHeapElement's "heap index" -- that is, the EventHeapElement's index within its heap.
	// SetIndex just changes the field; it doesn't interact with the heap that contains the EventHeapElement.
	// The heap index is dynamic and will change depending on what position the EventHeapElement is in
	// within the containing EventHeap, whereas SessionSpecificEventIndex and GlobalEventIndex are/return
	// static values that are set when the underlying domain.Event is first created.
	SetIndex(idx int)
	// GetIndex returns the EventHeapElement's "heap index" -- that is, the EventHeapElement's index within its heap.
	// The heap index is dynamic and will change depending on what position the EventHeapElement is in
	// within the containing EventHeap, whereas SessionSpecificEventIndex and GlobalEventIndex are/return
	// static values that are set when the underlying domain.Event is first created.
	GetIndex() int
	// OriginalTimestamp returns the original timestamp of the underlying event.
	OriginalTimestamp() time.Time
	// SessionID returns the ID of the associated Session.
	SessionID() string
	// String is toString for EventHeapElement.
	String() string
	// Data returns the data attached to the underlying *generator.Event struct.
	Data() interface{}
	// GetEvent returns the underlying *generator.Event struct.
	GetEvent() Event
	// Enqueued returns true if this event heap element is presently enqueued in/with the event queue service.
	Enqueued() bool
	// SessionSpecificEventIndex indicates the order in which the event was created relative to other events targeting
	// the same Session.
	// The first event created for a session while have an index of 0.
	// The last event created for a session while have an index of N - 1, where N is the number of events created
	// for that Session.
	SessionSpecificEventIndex() int
	// GlobalEventIndex provides a global ordering for comparing all events with each other within a workload.
	GlobalEventIndex() uint64
	// SetEnqueued sets the enqueued value.
	SetEnqueued(bool)
	// Id returns ID of underlying event.
	Id() string
}
