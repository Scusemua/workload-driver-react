package domain

import (
	"time"
)

// EventHeapElement ...
type EventHeapElement interface {
	// Name of the event.
	Name() EventName
	// Recompute the EventHeapElementImpl's adjustedTimestamp field based on the provided Total Delay value.
	RecalculateTimestamp(totalDelay time.Duration)
	// Update the EventHeapElementImpl's index in its heap. This just changes the field; it doesn't interact with the heap that contains the EventHeapElementImpl.
	SetIndex(idx int)
	// Return the EventHeapElementImpl's index in its heap.
	GetIndex() int
	// Return the original timestamp of the underlying event.
	OriginalTimestamp() time.Time
	// Return the timestamp of the event after being adjusted by the associated Session's totalDelay field.
	AdjustedTimestamp() time.Time
	// Return the ID of the associated Session.
	SessionID() string
	// ToString for EventHeapElementImpl.
	String() string
	// Data attached to the underlying *generator.Event struct.
	Data() interface{}
	// Return the underlying *generator.Event struct.
	GetEvent() Event
	// Return true if this event heap element is presently enqueued in/with the event queue service.
	Enqueued() bool
	// Set the enqueued value.
	SetEnqueued(bool)
	// Return ID of underlying event.
	Id() string
}
