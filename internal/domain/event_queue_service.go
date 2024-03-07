package domain

import "time"

// EventQueueService ...
type EventQueueService interface {
	// Initialize the logger for the EventQueueServiceImpl.
	// Need because EventQueueServiceImpl contains mutexes.
	// There's an error if we put the body of this function in the "constructor" of EventQueueServiceImpl.
	Initialize()
	// Place the given event in the 'delayed event' slice for the Session identified by the given ID.
	// @param evt *EventHeapElementImpl: The event that we are delaying.
	// @param sessionId string: The ID of the Session associated with the event.
	DelayEvent(evt Event, sessionId string)
	// Requeue the events that have been delayed for the Session identified by the given ID.
	// This will clear the slice of delayed events for the associated Session.
	//
	// Return an integer indicating the number of delayed events that were requeued.
	//
	// IMPORTANT: The TotalDelay field of the associated Session should have been updated prior to calling this function.
	RequeueDelayedEvents(sessionId string) int
	// Record that the session with the given ID has been permanently terminated.
	// For the remainder of the simulation, events from the Workload Generator that target this session will simply be discarded rather than enqueued.
	RegisterTerminatedSession(sessionId string)
	// Return the next, ready-to-be-processed `EventSessionReady` from the queue.
	GetNextSessionStartEvent(currentTime time.Time) Event
	// Return true if there is at least 1 event in the event queue for the specified pod/Session.
	// Otherwise, return false. Returns an error (and false) if no queue exists for the specified pod/Session.
	HasEvents(podId string) (bool, error)
	// Return true if there is at least one sessionReadyEvent in the `EventQueueService_Old::sessionReadyEvents` queue.
	HasSessionReadyEvents() bool
	// Return the timestamp of the next session event to be processed. The error will be nil if there is at least one session event enqueued.
	// If there are no session events enqueued, then this will return time.Time{} and an ErrNoMoreEvents error.
	GetTimestampOfNextReadyEvent() (time.Time, error)
	// Return the time at which the next event for the Session with the given ID will be ready.
	// This is not very efficent and should only be used for testing/debugging.
	//
	// Returns a `ErrNoMoreEvents` error if there are no events for that Session.
	// Returns a `ErrNoEventQueue` error if there is no generator.SimpleEventHeap for the specified Session.
	GetTimestampOfNextReadyEventForSession(podId string) (time.Time, error)
	// Enqueue the given event in the event heap associated with the event's target pod/container/session.
	// If no such event heap exists already, then first create the event heap.
	EnqueueEvent(evt Event)
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
	// Remove the given event from the EventQueueServiceImpl::eventsPerSession.
	UnregisterEvent(evt EventHeapElement)
	// Create and return a new *EventHeapElementImpl.
	NewEventHeapElement(evt Event, enqueued bool) EventHeapElement
}
