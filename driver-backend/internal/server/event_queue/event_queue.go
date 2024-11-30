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
	ErrNoEventQueue        = errors.New("no corresponding event queue for given Session")
	ErrNoMoreEvents        = errors.New("there are no more events for the specified Session")
	ErrUnregisteredSession = errors.New("specified session does not have an event queue registered")
	ErrHoldAlreadyActive   = errors.New("there is already an active event hold on the specified session")
	ErrNoHoldActive        = errors.New("there is no event hold on the specified session")
)

// EventQueue maintains a queue of events (sorted by timestamp) for each unique session.
type EventQueue struct {
	logger        *zap.Logger        // Logger for printing purely structured output.
	sugaredLogger *zap.SugaredLogger // Logger for printing formatted output.
	atom          *zap.AtomicLevel

	// events is a queue of queues. Each internal queue is associated with a particular session.
	events MainEventQueue

	eventsPerSession *hashmap.HashMap // Mapping from session ID to another hashmap. The second/inner hashmap is a map from event ID to the event.
	eventHeapMutex   sync.Mutex       // Controls access to the underlying eventHeap.
	delayedEvents    *hashmap.HashMap // Mapping from SessionID to a slice of *generator.Event that have been returned to the Cluster for processing but could not be processed at the time because the associated Session was descheduled. Once the Session is rescheduled, it will re-enqueue all of the events in its `delayedEvents` slice.
	doneChan         chan interface{}
}

// NewEventQueue creates a new EventQueue struct and returns a pointer to it.
func NewEventQueue(atom *zap.AtomicLevel) *EventQueue {
	queue := &EventQueue{
		atom:             atom,
		eventsPerSession: hashmap.New(100),
		delayedEvents:    hashmap.New(100),
		doneChan:         make(chan interface{}),
		events:           make(MainEventQueue, 0),
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

func (q *EventQueue) WorkloadExecutionCompleteChan() chan interface{} {
	return q.doneChan
}

// MainEventQueueLength returns the result of calling Len on the MainEventQueue field of the EventQueue.
func (q *EventQueue) MainEventQueueLength() int {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	return q.unsafeMainEventQueueLength()
}

func (q *EventQueue) unsafeMainEventQueueLength() int {
	return q.events.Len()
}

func (q *EventQueue) PrintEvents() {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	for _, sessionQueue := range q.events {
		q.sugaredLogger.Debugf("Session Event Queue \"%s\" [NumEvents=%d, HeapIndex=%d]",
			sessionQueue.SessionId, sessionQueue.Len(), sessionQueue.HeapIndex)
	}
}

// HasEventsForTick returns true if there are events available for the specified tick; otherwise return false.
func (q *EventQueue) HasEventsForTick(tick time.Time) bool {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	length := q.lenUnsafe()
	if length == 0 {
		return false
	}

	sessionEventQueue := q.events.Peek()
	if sessionEventQueue == nil {
		q.logger.Warn("Peeked event queue, got back nil.",
			zap.Int("events_length", length),
			zap.Int("num_session_event_queues", q.NumSessionQueues()),
			zap.Int("main_event_queue_len", q.unsafeMainEventQueueLength()),
			zap.Int("len_main_event_queue", len(q.events)))
		return false
	}

	// If the queue is empty, then return false.
	if sessionEventQueue.Len() == 0 {
		return false
	}

	timestamp, _ := sessionEventQueue.NextEventTimestamp()
	return tick == timestamp || timestamp.Before(tick)
}

// HasEventsForSession returns true if there is at least 1 event in the event queue for the specified pod/Session.
// Otherwise, return false. Returns an error (and false) if no queue exists for the specified pod/Session.
func (q *EventQueue) HasEventsForSession(podId string) (bool, error) {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	val, ok := q.eventsPerSession.Get(podId)

	if !ok {
		return false, ErrNoEventQueue
	}

	hasEvents := val.(*SessionEventQueue).Len() > 0

	return hasEvents, nil
}

// EnqueueEvent enqueues the given event in the event heap associated with the event's target pod/container/session.
// If no such event heap exists already, then first create the event heap.
func (q *EventQueue) EnqueueEvent(evt *domain.Event) {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	sess := evt.Data.(domain.SessionMetadata)
	sessionId := sess.GetPod()

	if evt.Name == domain.EventSessionStarted { // Do nothing other than try to create the heap.
		sessionEventQueue, loaded := q.eventsPerSession.GetOrInsert(sessionId, NewSessionEventQueue(sessionId))

		//q.logger.Debug("Creating new SessionEventQueue while enqueuing \"session-started\" event.",
		//	zap.String("event_name", evt.Name.String()),
		//	zap.String("event_id", evt.ID),
		//	zap.String("session_id", evt.SessionID()),
		//	zap.Time("event_timestamp", evt.Timestamp),
		//	zap.Time("original_timestamp", evt.OriginalTimestamp))

		if loaded {
			panic("we already have a SessionEventQueue for a session whose 'session-started' we just received")
		}

		heap.Push(&q.events, sessionEventQueue)
		return
	}

	var sessionEventQueue *SessionEventQueue
	val, loaded := q.eventsPerSession.Get(sessionId)
	if !loaded {
		//q.logger.Debug("Creating new SessionEventQueue while enqueuing event.",
		//	zap.String("event_name", evt.Name.String()),
		//	zap.String("event_id", evt.ID),
		//	zap.String("session_id", evt.SessionID()),
		//	zap.Time("event_timestamp", evt.Timestamp),
		//	zap.Time("original_timestamp", evt.OriginalTimestamp))

		sessionEventQueue = NewSessionEventQueue(sessionId)
		q.eventsPerSession.Set(sessionId, sessionEventQueue)

		heap.Push(&q.events, sessionEventQueue)
	} else {
		sessionEventQueue = val.(*SessionEventQueue)
	}

	sessionEventQueue.Push(evt)

	heap.Fix(&q.events, sessionEventQueue.HeapIndex)

	// Record that the event was enqueued.
	evt.RecordThatEventWasEnqueued()

	if evt.GetNumTimesEnqueued() > 1 {
		q.logger.Debug("Re-enqueued event.",
			zap.String("event_name", evt.Name.String()),
			zap.String("event_id", evt.ID),
			zap.String("session_id", evt.SessionID()),
			zap.Time("event_timestamp", evt.Timestamp),
			zap.Time("original_timestamp", evt.OriginalTimestamp),
			zap.Int32("num_times_enqueued", evt.GetNumTimesEnqueued()),
			zap.Int("num_events_enqueued_for_session", sessionEventQueue.Len()),
			zap.Int("session_event_queue_heap_index", sessionEventQueue.HeapIndex))
	} else {
		q.logger.Debug("Enqueued event for the first time.",
			zap.String("event_name", evt.Name.String()),
			zap.String("event_id", evt.ID),
			zap.String("session_id", evt.SessionID()),
			zap.Time("event_timestamp", evt.Timestamp),
			zap.Time("original_timestamp", evt.OriginalTimestamp),
			zap.Int32("num_times_enqueued", evt.GetNumTimesEnqueued()),
			zap.Int("num_events_enqueued_for_session", sessionEventQueue.Len()),
			zap.Int("session_event_queue_heap_index", sessionEventQueue.HeapIndex))
	}
}

// GetTimestampOfNextReadyEvent returns the timestamp of the next session event to be processed.
// The error will be nil if there is at least one session event enqueued.
// If there are no session events enqueued, then this will return time.Time{} and an ErrNoMoreEvents error.
func (q *EventQueue) GetTimestampOfNextReadyEvent() (time.Time, error) {
	length := q.Len()
	if length == 0 {
		return time.Time{}, ErrNoMoreEvents
	}

	nextSessionQueue := q.events.Peek()
	if nextSessionQueue == nil {
		q.logger.Warn("Peeked event queue, got back nil.",
			zap.Int("events_length", length),
			zap.Int("num_session_event_queues", q.NumSessionQueues()),
			zap.Int("main_event_queue_len", q.unsafeMainEventQueueLength()),
			zap.Int("len_main_event_queue", len(q.events)))
		return time.Time{}, ErrNoMoreEvents
	}

	timestamp, ok := nextSessionQueue.NextEventTimestamp()

	if ok {
		return timestamp, nil
	}

	return time.Time{}, ErrNoMoreEvents
}

// Len returns the total number of events enqueued.
func (q *EventQueue) Len() int {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	return q.lenUnsafe()
}

func (q *EventQueue) lenUnsafe() int {
	length := 0
	for kv := range q.eventsPerSession.Iter() {
		sessionEventQueue := kv.Value.(*SessionEventQueue)

		length += sessionEventQueue.Len()
	}

	return length
}

// NumSessionQueues returns the total number of session queues that are enqueued.
// This will also count empty session queues.
func (q *EventQueue) NumSessionQueues() int {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	return q.unsafeNumSessionQueues()
}

func (q *EventQueue) unsafeNumSessionQueues() int {
	return q.eventsPerSession.Len()
}

func (q *EventQueue) Peek(threshold time.Time) *domain.Event {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	if q.lenUnsafe() == 0 {
		return nil
	}

	sessionQueue := q.events.Peek()
	timestamp, ok := sessionQueue.NextEventTimestamp()
	if !ok {
		return nil
	}

	if threshold == timestamp || timestamp.Before(threshold) {
		return sessionQueue.Peek()
	}

	return nil
}

// HoldEventsForSession prevents the EventQueue from returning events for the specified session
// until ReleaseEventHoldForSession is called.
func (q *EventQueue) HoldEventsForSession(sessionId string) error {
	val, loaded := q.eventsPerSession.Get(sessionId)
	if !loaded {
		return fmt.Errorf("%w: \"%s\"", ErrUnregisteredSession, sessionId)
	}

	sessionEventQueue := val.(*SessionEventQueue)

	q.logger.Debug("Creating hold on events for session.",
		zap.String("session_id", sessionId),
		zap.Duration("current_delay", sessionEventQueue.Delay))

	if sessionEventQueue.HoldActive {
		q.logger.Error("There is already an active event hold for specified session.",
			zap.String("session_id", sessionId),
			zap.Duration("current_delay", sessionEventQueue.Delay))

		return fmt.Errorf("%w: \"%s\"", ErrHoldAlreadyActive, sessionId)
	}

	sessionEventQueue.HoldActive = true

	heap.Fix(&q.events, sessionEventQueue.HeapIndex)

	return nil
}

// ReleaseEventHoldForSession instructs the EventQueue to stop "holding" events for the session (i.e., to
// stop refusing to return events for the specified session).
func (q *EventQueue) ReleaseEventHoldForSession(sessionId string) error {
	val, loaded := q.eventsPerSession.Get(sessionId)
	if !loaded {
		return fmt.Errorf("%w: \"%s\"", ErrUnregisteredSession, sessionId)
	}

	sessionEventQueue := val.(*SessionEventQueue)

	q.logger.Debug("Releasing hold on events for session.",
		zap.String("session_id", sessionId),
		zap.Duration("current_delay", sessionEventQueue.Delay))

	if !sessionEventQueue.HoldActive {
		q.logger.Error("There is not an active event hold for specified session.",
			zap.String("session_id", sessionId),
			zap.Duration("current_delay", sessionEventQueue.Delay))

		return fmt.Errorf("%w: \"%s\"", ErrNoHoldActive, sessionId)
	}

	sessionEventQueue.HoldActive = false

	heap.Fix(&q.events, sessionEventQueue.HeapIndex)

	return nil
}

// DelaySession adds the specified time.Duration to the specified Session's delay.
//
// DelaySession returns nil on success and an ErrUnregisteredSession error if the specified Session
// does not have an event queue.
func (q *EventQueue) DelaySession(sessionId string, amount time.Duration) error {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	if q.unsafeNumSessionQueues() == 0 {
		return fmt.Errorf("%w: \"%s\"", ErrUnregisteredSession, sessionId)
	}

	val, loaded := q.eventsPerSession.Get(sessionId)
	if !loaded {
		return fmt.Errorf("%w: \"%s\"", ErrUnregisteredSession, sessionId)
	}

	sessionEventQueue := val.(*SessionEventQueue)
	sessionEventQueue.Delay += amount

	q.logger.Debug("Increased delay of session.",
		zap.String("session_id", sessionId),
		zap.Duration("amount", amount),
		zap.Duration("new_delay", sessionEventQueue.Delay))

	heap.Fix(&q.events, sessionEventQueue.HeapIndex)

	return nil
}

// SetSessionDelay sets the specified Session's delay to the specified time.Duration.
//
// SetSessionDelay returns nil on success and a ErrUnregisteredSession error if the specified
// Session does not have an event queue.
func (q *EventQueue) SetSessionDelay(sessionId string, delay time.Duration) error {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	if q.unsafeNumSessionQueues() == 0 {
		return fmt.Errorf("%w: \"%s\"", ErrUnregisteredSession, sessionId)
	}

	val, loaded := q.eventsPerSession.Get(sessionId)
	if !loaded {
		return fmt.Errorf("%w: \"%s\"", ErrUnregisteredSession, sessionId)
	}

	sessionEventQueue := val.(*SessionEventQueue)
	sessionEventQueue.Delay = delay

	q.logger.Debug("Set delay of session.",
		zap.String("session_id", sessionId),
		zap.Duration("delay", delay))

	heap.Fix(&q.events, sessionEventQueue.HeapIndex)

	return nil
}

// Pop return the next event that occurs at or before the given timestamp, or nil if there are no such events.
// This will remove the event from the main EventQueueServiceImpl::eventHeap, but it will NOT remove the
// event from the EventQueueServiceImpl::eventsPerSession. To do that, you must call EventQueueServiceImpl::UnregisterEvent().
func (q *EventQueue) Pop(threshold time.Time) *domain.Event {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	sessionQueue := q.events.Peek()
	if sessionQueue == nil {
		return nil
	}

	timestamp, ok := sessionQueue.NextEventTimestamp()
	if !ok {
		return nil
	}

	if threshold == timestamp || timestamp.Before(threshold) {
		nextEvent := sessionQueue.Pop()

		nextEvent.SetIndex(q.lenUnsafe())
		nextEvent.Dequeued()

		q.logger.Debug("Returning ready event.",
			zap.String("event_name", nextEvent.Name.String()),
			zap.String("event_id", nextEvent.ID),
			zap.Time("threshold", threshold),
			zap.Time("original_event_timestamp", nextEvent.OriginalTimestamp),
			zap.Time("current_event_timestamp", nextEvent.Timestamp),
			zap.Time("returned_timestamp", timestamp),
			zap.Duration("event_delay", nextEvent.Delay),
			zap.Duration("session_queue_delay", sessionQueue.Delay),
			zap.Int32("num_times_enqueued", nextEvent.GetNumTimesEnqueued()),
			zap.String("session_id", nextEvent.SessionID()))

		heap.Fix(&q.events, sessionQueue.HeapIndex)

		return nextEvent
	}

	return nil
}
