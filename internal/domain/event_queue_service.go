package domain

import "time"

// This is an interface that is wrapped by the EventQueueService interface.
// Entities that only need to be able to deposit events into the queue receive/use values of this type.
// This is because they have no reason to access the rest of the EventQueueService API.
type EventQueueReceiver interface {
	EnqueueEvent(Event)
}

// EventQueueService ...
type EventQueueService interface {
	EventQueueReceiver

	// Return true if there is at least 1 event in the event queue for the specified pod/Session.
	// Otherwise, return false. Returns an error (and false) if no queue exists for the specified pod/Session.
	HasEventsForSession(podId string) (bool, error)

	// Return true if there are events available for the specified tick; otherwise return false.
	HasEventsForTick(time.Time) bool

	// Return the next, ready-to-be-processed `EventSessionReady` from the queue.
	GetNextSessionStartEvent(currentTime time.Time) Event

	// Return true if there is at least one sessionReadyEvent in the `EventQueueService_Old::sessionReadyEvents` queue.
	HasSessionReadyEvents() bool

	// Return the timestamp of the next session event to be processed. The error will be nil if there is at least one session event enqueued.
	// If there are no session events enqueued, then this will return time.Time{} and an ErrNoMoreEvents error.
	GetTimestampOfNextReadyEvent() (time.Time, error)

	// Fix the heap after the `totalDelay` field for a particular session changed.
	FixEvents(sessionId string, updatedDelay time.Duration)

	// Return the total number of events enqueued in the service.
	Len() int

	// Get the length without locking. This is used for printing when the information being printed doesn't need to be particularly precise.
	LenUnsafe() int

	// Return the next event that occurs at or before the given timestamp, or nil if there are no such events.
	// This will remove the event from the main EventQueueServiceImpl::eventHeap, but it will NOT remove the
	// event from the EventQueueServiceImpl::eventsPerSession. To do that, you must call EventQueueServiceImpl::UnregisterEvent().
	GetNextEvent(threshold time.Time) (EventHeapElement, bool)
}
