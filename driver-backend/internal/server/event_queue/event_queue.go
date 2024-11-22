package event_queue

import (
	"container/heap"
	"errors"
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

// BasicEventQueue maintains a queue of events (sorted by timestamp) for each unique session.
type BasicEventQueue struct {
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

// NewBasicEventQueue creates a new BasicEventQueue struct and returns a pointer to it.
func NewBasicEventQueue(atom *zap.AtomicLevel) *BasicEventQueue {
	queue := &BasicEventQueue{
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

func (q *BasicEventQueue) WorkloadExecutionCompleteChan() chan interface{} {
	return q.doneChan
}

// HasEventsForTick returns true if there are events available for the specified tick; otherwise return false.
func (q *BasicEventQueue) HasEventsForTick(tick time.Time) bool {
	if q.Len() == 0 {
		return false
	}

	sessionEventQueue := q.events.Peek()

	// If the queue is empty, then return false.
	if sessionEventQueue.Len() == 0 {
		return false
	}

	timestamp, _ := sessionEventQueue.NextEventTimestamp()
	return tick == timestamp || timestamp.Before(tick)
}

// HasEventsForSession returns true if there is at least 1 event in the event queue for the specified pod/Session.
// Otherwise, return false. Returns an error (and false) if no queue exists for the specified pod/Session.
func (q *BasicEventQueue) HasEventsForSession(podId string) (bool, error) {
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
func (q *BasicEventQueue) EnqueueEvent(evt *domain.Event) {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	sess := evt.Data.(domain.SessionMetadata)
	sessionId := sess.GetPod()

	if evt.Name == domain.EventSessionStarted { // Do nothing other than try to create the heap.
		sessionEventQueue, loaded := q.eventsPerSession.GetOrInsert(sessionId, NewSessionEventQueue(sessionId))

		if loaded {
			panic("we already have a SessionEventQueue for a session whose 'session-started' we just received")
		}

		heap.Push(&q.events, sessionEventQueue)
		return
	}

	var sessionEventQueue *SessionEventQueue
	val, loaded := q.eventsPerSession.Get(sessionId)
	if !loaded {
		sessionEventQueue = NewSessionEventQueue(sessionId)
		q.eventsPerSession.Set(sessionId, sessionEventQueue)
	} else {
		sessionEventQueue = val.(*SessionEventQueue)
	}

	sessionEventQueue.Push(evt)

	heap.Fix(&q.events, sessionEventQueue.HeapIndex)

	// Record that the event was enqueued.
	evt.RecordThatEventWasEnqueued()

	q.logger.Debug("Enqueued event.",
		zap.String("event_name", evt.Name.String()),
		zap.String("event_id", evt.ID),
		zap.String("session_id", evt.SessionID()),
		zap.Time("event_timestamp", evt.Timestamp),
		zap.Time("original_timestamp", evt.OriginalTimestamp))
}

// GetTimestampOfNextReadyEvent returns the timestamp of the next session event to be processed.
// The error will be nil if there is at least one session event enqueued.
// If there are no session events enqueued, then this will return time.Time{} and an ErrNoMoreEvents error.
func (q *BasicEventQueue) GetTimestampOfNextReadyEvent() (time.Time, error) {
	if q.Len() == 0 {
		return time.Time{}, ErrNoMoreEvents
	}

	nextSessionQueue := q.events.Peek()
	timestamp, ok := nextSessionQueue.NextEventTimestamp()

	if ok {
		return timestamp, nil
	}

	return time.Time{}, ErrNoMoreEvents
}

// Len returns the total number of events enqueued.
func (q *BasicEventQueue) Len() int {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	length := 0
	for kv := range q.eventsPerSession.Iter() {
		sessionEventQueue := kv.Value.(*SessionEventQueue)

		length += sessionEventQueue.Len()
	}

	return length
}

// NumSessionQueues returns the total number of session queues that are enqueued.
// This will also count empty session queues.
func (q *BasicEventQueue) NumSessionQueues() int {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	return q.events.Len()
}

// GetNextEvent return the next event that occurs at or before the given timestamp, or nil if there are no such events.
// This will remove the event from the main EventQueueServiceImpl::eventHeap, but it will NOT remove the
// event from the EventQueueServiceImpl::eventsPerSession. To do that, you must call EventQueueServiceImpl::UnregisterEvent().
func (q *BasicEventQueue) GetNextEvent(threshold time.Time) (*domain.Event, bool) {
	q.eventHeapMutex.Lock()
	defer q.eventHeapMutex.Unlock()

	if q.Len() == 0 {
		return nil, false
	}

	sessionQueue := q.events.Peek()
	timestamp, ok := sessionQueue.NextEventTimestamp()
	if !ok {
		return nil, false
	}

	if threshold == timestamp || timestamp.Before(threshold) {
		nextEvent := sessionQueue.Pop()

		nextEvent.SetIndex(q.Len())
		nextEvent.SetEnqueued(false)

		q.logger.Debug("Returning ready event.",
			zap.String("event_name", nextEvent.Name.String()),
			zap.String("event_id", nextEvent.ID),
			zap.Time("threshold", threshold),
			zap.Time("original_event_timestamp", nextEvent.OriginalTimestamp),
			zap.Time("current_event_timestamp", nextEvent.Timestamp),
			zap.Duration("event_delay", nextEvent.Delay),
			zap.String("session_id", nextEvent.SessionID()))

		heap.Fix(&q.events, 0)

		return nextEvent, true
	}

	return nil, false
}
