package event_queue

import (
	"container/heap"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrNoEventQueue = errors.New("no corresponding event queue for given Session")
	ErrNoMoreEvents = errors.New("there are no more events for the specified Session")
)

// Maintains a queue of events (sorted by timestamp) for each unique session.
type eventQueue struct {
	logger        *zap.Logger        // Logger for printing purely structured output.
	sugaredLogger *zap.SugaredLogger // Logger for printing formatted output.
	atom          *zap.AtomicLevel

	sessionReadyEvents domain.SimpleEventHeap // The `EventSessionReady` events are stored separately, as this is what actually creates/registers a Session within the Cluster. The Cluster won't have a given Session's info until after the associated `EventSessionReady` is processed, so the EventSessionReady events must go through a different path.
	terminatedSessions *hashmap.HashMap       // Map from Session ID to the last event that was returned successfully. Used to monitor Sessions who are not consuming events for a suspiciously long period of time. lastEventProcessed *hashmap.HashMap Sessions that have been permanently stopped and thus won't be consuming events anymore. We don't bother saving events for those sessions.
	eventHeapMutex     sync.Mutex             // Controls access to the underlying eventHeap.
	eventHeap          domain.EventHeap       // The heap of events, sorted by timestamp in ascending order (so future events are further in the list). Does not contain "Session Ready" events. Those are stored in a separate heap.
	eventsPerSession   *hashmap.HashMap       // Mapping from session ID to another hashmap. The second/inner hashmap is a map from event ID to the event.
	delayedEvents      *hashmap.HashMap       // Mapping from SessionID to a slice of *generator.Event that have been returned to the Cluster for processing but could not be processed at the time because the associated Session was descheduled. Once the Session is rescheduled, it will re-enqueue all of the events in its `delayedEvents` slice.
	doneChan           chan interface{}
}

func NewEventQueue(atom *zap.AtomicLevel) domain.EventQueueService {
	queue := &eventQueue{
		atom:               atom,
		eventsPerSession:   hashmap.New(100),
		terminatedSessions: hashmap.New(100),
		delayedEvents:      hashmap.New(100),
		sessionReadyEvents: make(domain.SimpleEventHeap, 0, 100),
		eventHeap:          domain.EventHeap(make([]domain.EventHeapElement, 0, 100)),
		doneChan:           make(chan interface{}),
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, queue.atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	queue.logger = logger
	queue.sugaredLogger = logger.Sugar()

	return queue
}

func (q *eventQueue) WorkloadExecutionCompleteChan() chan interface{} {
	return q.doneChan
}

// HasEventsForTick returns true if there are events available for the specified tick; otherwise return false.
func (q *eventQueue) HasEventsForTick(tick time.Time) bool {
	if q.Len() == 0 {
		return false
	}

	nextEvent := q.eventHeap.Peek()
	nextEventTimestamp := nextEvent.AdjustedTimestamp()
	if tick == nextEventTimestamp || nextEventTimestamp.Before(tick) {
		return true
	}

	return false
}

// GetAllSessionStartEventsForTick returns a slice of domain.Event containing all the EventSessionReady domain.Event
// instances that should be processed within the specified tick.
//
// If max is negative, then all events are returned.
//
// If max is a positive value, then up to `max` events are returned.
func (q *eventQueue) GetAllSessionStartEventsForTick(tick time.Time, max int) []domain.Event {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	events := make([]domain.Event, 0)

	// Check if there is at least one event in the heap. If not, then return nil.
	if q.sessionReadyEvents.Len() == 0 {
		return events
	}

	for {
		// Check if the next event is ready. If not, return nil.
		if q.sessionReadyEvents.Peek().Timestamp().After(tick) {
			break
		}

		evt := heap.Pop(&q.sessionReadyEvents).(domain.Event)

		q.sugaredLogger.Debugf(
			"SessionReadyEvent '%s' for Session %s with timestamp %v occurs during or before current tick %v.",
			evt.Id(), evt.Data().(domain.SessionMetadata).GetPod(), evt.Timestamp(), tick)

		events = append(events, evt)

		// If we've collected `max` events, then we'll break and return them.
		if max >= 0 && len(events) >= max {
			break
		}
	}

	q.sugaredLogger.Debugf("Returning %d (max: %d) SessionReadyEvent(s) with timestamps during or before tick %v",
		len(events), max, tick)

	return events
}

// GetNextSessionStartEvent returns the next, ready-to-be-processed `EventSessionReady` from the queue.
func (q *eventQueue) GetNextSessionStartEvent(currentTime time.Time) domain.Event {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	// Check if there is at least one event in the heap. If not, then return nil.
	if q.sessionReadyEvents.Len() == 0 {
		return nil
	}

	// Check if the next event is ready. If not, return nil.
	if q.sessionReadyEvents.Peek().Timestamp().After(currentTime) {
		return nil
	}

	evt := heap.Pop(&q.sessionReadyEvents).(domain.Event)

	q.sugaredLogger.Debugf(
		"SessionReadyEvent '%s' for Session %s with timestamp %v occurs during or before current tick %v.",
		evt.Id(), evt.Data().(domain.SessionMetadata).GetPod(), evt.Timestamp(), currentTime)

	return evt
}

// HasSessionReadyEvents returns true if there is at least one sessionReadyEvent in the `EventQueueService_Old::sessionReadyEvents` queue.
func (q *eventQueue) HasSessionReadyEvents() bool {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	hasEvents := q.sessionReadyEvents.Len() > 0
	return hasEvents
}

// HasEventsForSession returns true if there is at least 1 event in the event queue for the specified pod/Session.
// Otherwise, return false. Returns an error (and false) if no queue exists for the specified pod/Session.
func (q *eventQueue) HasEventsForSession(podId string) (bool, error) {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	val, ok := q.eventsPerSession.Get(podId)

	if !ok {
		return false, ErrNoEventQueue
	}

	hasEvents := val.(*hashmap.HashMap).Len() > 0

	return hasEvents, nil
}

// EnqueueEvent enqueues the given event in the event heap associated with the event's target pod/container/session.
// If no such event heap exists already, then first create the event heap.
func (q *eventQueue) EnqueueEvent(evt domain.Event) {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	sess := evt.Data().(domain.SessionMetadata)
	if evt.Name() == domain.EventSessionReady {
		// The event heap for "regular" events corresponding to the same Session.
		q.eventsPerSession.GetOrInsert(sess.GetPod(), hashmap.New(10))

		// As described above, EventSessionReady events must be processed differently.
		heap.Push(&q.sessionReadyEvents, evt)

		q.sugaredLogger.Debugf("Enqueued SessionReadyEvent: session=%s; ts=%v.", sess.GetPod(), evt.Timestamp())
	} else if evt.Name() == domain.EventSessionStarted { // Do nothing other than try to create the heap.
		q.eventsPerSession.GetOrInsert(sess.GetPod(), hashmap.New(10)) // Don't bother capturing the return value. We just don't want to overwrite the existing hashmap if it exists.
	} else if sess, ok := evt.Data().(domain.SessionMetadata); ok {
		podId := sess.GetPod()

		// If the session has been terminated permanently, then discard its events.
		if _, ok = q.terminatedSessions.Get(podId); ok {
			return
		}

		eventHeapElement := q.newEventHeapElement(evt, true)

		heap.Push(&q.eventHeap, eventHeapElement)

		val, ok := q.eventsPerSession.Get(podId)

		if !ok {
			panic(fmt.Sprintf("Expected there to be a HashMap for session %s in the EventQueueServiceImpl::eventsPerSession field.", podId))
		}

		eventsForSession := val.(*hashmap.HashMap)
		eventsForSession.Set(evt.Id(), eventHeapElement)

		q.sugaredLogger.Debugf("Enqueued \"%v\" (id=%v, ts=%v) for session %s (src=%v). Heap length: %d", evt.Name(), evt.Id(), evt.Timestamp(), podId, evt.EventSource(), q.eventHeap.Len())
	} else {
		panic(fmt.Sprintf("Event %v has no data associated with it.", evt))
	}
}

// FixEvents fixes the heap after the `totalDelay` field for a particular session changed.
func (q *eventQueue) FixEvents(sessionId string, updatedDelay time.Duration) {
	val, ok := q.eventsPerSession.Get(sessionId)

	if !ok {
		panic(fmt.Sprintf("Expected to find entry in EventQueueServiceImpl::eventsPerSession field for session %s.", sessionId))
	}

	sessionEvents := val.(*hashmap.HashMap)
	iter := sessionEvents.Iter()

	if q.logger.Level() == zapcore.DebugLevel {
		q.sugaredLogger.Debugf("Updating timestamps for event(s) targeting session %s, of which there is/are %d. New delay: %v. Current size of main event heap: %d.", sessionId, sessionEvents.Len(), updatedDelay, q.eventHeap.Len())
	}

	for kv := range iter {
		eventHeapElement := kv.Value.(domain.EventHeapElement)
		oldAdjustedTimestamp := eventHeapElement.AdjustedTimestamp()

		oldIndex := eventHeapElement.GetIndex()

		eventHeapElement.RecalculateTimestamp(updatedDelay)

		if eventHeapElement.Enqueued() {
			heap.Fix(&q.eventHeap, eventHeapElement.GetIndex())

			if q.logger.Level() == zapcore.DebugLevel {
				q.sugaredLogger.Debugf("Updated timestamp for event \"%s\" [id=%s] from %v to %v. Index changed from %d to %d.", eventHeapElement.Name(), eventHeapElement.Id(), oldAdjustedTimestamp, eventHeapElement.AdjustedTimestamp(), oldIndex, eventHeapElement.GetIndex())
			}
		}
	}
}

// GetTimestampOfNextReadyEvent returns the timestamp of the next session event to be processed.
// The error will be nil if there is at least one session event enqueued.
// If there are no session events enqueued, then this will return time.Time{} and an ErrNoMoreEvents error.
func (q *eventQueue) GetTimestampOfNextReadyEvent() (time.Time, error) {
	if q.Len() == 0 {
		return time.Time{}, ErrNoMoreEvents
	}

	nextEvent := q.eventHeap.Peek()

	return nextEvent.AdjustedTimestamp(), nil
}

// Len returns the total number of events enqueued.
func (q *eventQueue) Len() int {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()
	length := q.eventHeap.Len()

	return length
}

// LenUnsafe returns the length without locking.
// This is used for printing when the information being printed doesn't need to be particularly precise.
func (q *eventQueue) LenUnsafe() int {
	return q.eventHeap.Len()
}

// GetNextEvent return the next event that occurs at or before the given timestamp, or nil if there are no such events.
// This will remove the event from the main EventQueueServiceImpl::eventHeap, but it will NOT remove the
// event from the EventQueueServiceImpl::eventsPerSession. To do that, you must call EventQueueServiceImpl::UnregisterEvent().
func (q *eventQueue) GetNextEvent(threshold time.Time) (domain.EventHeapElement, bool) {
	if q.Len() == 0 {
		return nil, false
	}

	nextEvent := q.eventHeap.Peek()
	nextEventTimestamp := nextEvent.AdjustedTimestamp()
	if threshold == nextEventTimestamp || nextEventTimestamp.Before(threshold) {
		heap.Pop(&q.eventHeap)

		nextEvent.SetIndex(q.Len())
		nextEvent.SetEnqueued(false)

		q.sugaredLogger.Debugf("Returning ready event \"%s\" [id=%s] targeting session %s. Heap size: %d.", nextEvent.Name(), nextEvent.Id(), nextEvent.SessionID(), q.Len())
		return nextEvent, true
	}

	return nil, false
}

// Create and return a new *eventHeapElementImpl.
func (q *eventQueue) newEventHeapElement(evt domain.Event, enqueued bool) domain.EventHeapElement {
	adjustedTimestamp := evt.Timestamp()

	var totalDelay = time.Duration(0)

	eventHeapElement := &eventHeapElementImpl{
		Event:             evt,
		enqueued:          enqueued,
		idx:               -1,
		adjustedTimestamp: adjustedTimestamp.Add(totalDelay),
	}
	return eventHeapElement
}
