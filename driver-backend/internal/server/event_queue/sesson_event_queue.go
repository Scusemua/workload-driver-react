package event_queue

import (
	"container/heap"
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"sync"
	"time"
)

// SessionEventQueue contains all the available events for a particular Session.
type SessionEventQueue struct {
	// SessionId is the ID of the associated Session
	SessionId string

	// Delay is applied to all events in the queue.
	Delay time.Duration

	// InternalQueue contains the domain.Event instances for the associated Session.
	InternalQueue domain.EventHeap

	// HeapIndex is the index of this SessionEventQueue with respect to other SessionEventQueue instances.
	HeapIndex int

	EventsMap *hashmap.HashMap

	HoldActive bool

	logger *zap.Logger

	mu sync.Mutex
}

// NewSessionEventQueue creates a new SessionEventQueue struct and returns a pointer to it.
func NewSessionEventQueue(sessionId string) *SessionEventQueue {
	queue := &SessionEventQueue{
		SessionId:     sessionId,
		InternalQueue: make(domain.EventHeap, 0),
		EventsMap:     hashmap.New(16),
		HeapIndex:     -1,
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	queue.logger = logger

	return queue
}

func (q *SessionEventQueue) IncurDelay(amount time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.Delay += amount
}

// Len returns the length of the InternalQueue of the target SessionEventQueue.
func (q *SessionEventQueue) Len() int {
	return len(q.InternalQueue)
}

// IsEmpty returns true if the InternalQueue of the target SessionEventQueue is empty.
func (q *SessionEventQueue) IsEmpty() bool {
	return len(q.InternalQueue) == 0
}

// IsNonEmpty returns true if the InternalQueue of the target SessionEventQueue is non-empty.
func (q *SessionEventQueue) IsNonEmpty() bool {
	return len(q.InternalQueue) > 0
}

func (q *SessionEventQueue) SetIndex(idx int) {
	q.HeapIndex = idx
}

func (q *SessionEventQueue) GetIndex() int {
	return q.HeapIndex
}

// Pop removes the next domain.Event from the InternalQueue of the SessionEventQueue and returns the domain.Event.
func (q *SessionEventQueue) Pop() *domain.Event {
	if q.Len() == 0 {
		return nil
	}

	evt := heap.Pop(&q.InternalQueue).(*domain.Event)

	if q.HoldActive {
		q.logger.Warn("Popping event off of SessionEventQueue despite a hold being active.",
			zap.String("session_id", q.SessionId),
			zap.String("event", evt.String()))
	}

	return evt
}

// Push pushes the specified *domain.Event into the InternalQueue of the SessionEventQueue.
func (q *SessionEventQueue) Push(evt *domain.Event) {
	if evt.SessionID() != q.SessionId {
		panic(fmt.Sprintf("attempting to push event with session ID \"%s\" into queue for session \"%s\"\n",
			evt.SessionID(), q.SessionId))
	}

	heap.Push(&q.InternalQueue, evt)

	q.EventsMap.Set(evt.ID, evt)
}

// Peek returns -- but does not remove -- the next *domain.Event in the InternalQueue of the SessionEventQueue.
func (q *SessionEventQueue) Peek() *domain.Event {
	if q.Len() == 0 {
		return nil
	}

	return q.InternalQueue.Peek()
}

// NextEventTimestamp returns the Timestamp of the next domain.Event in this SessionEventQueue's InternalQueue.
// The SessionEventQueue's Delay field is added to the domain.Event's Timestamp field before the Timestamp is returned.
//
// NextEventTimestamp is thread-safe.
func (q *SessionEventQueue) NextEventTimestamp() (time.Time, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.unsafeNextEventTimestamp()
}

// unsafeNextEventTimestamp returns the Timestamp of the next domain.Event in this SessionEventQueue's InternalQueue.
// The SessionEventQueue's Delay field is added to the domain.Event's Timestamp field before the Timestamp is returned.
func (q *SessionEventQueue) unsafeNextEventTimestamp() (time.Time, bool) {
	// What do we do if the queue is empty?
	if len(q.InternalQueue) == 0 {
		return time.Time{}, false
	}

	timestampWithDelay := q.InternalQueue.Peek().Timestamp.Add(q.Delay)

	// If there's a hold on events, then we'll add a huge constant amount to the timestamp so that the events
	// are delayed more-or-less indefinitely.
	if q.HoldActive {
		timestampWithDelay = timestampWithDelay.Add(time.Hour * 87660) // 87,660 hours is 10 years.
	}

	return timestampWithDelay, true
}

// NextEventName returns the Name of the next domain.Event in this SessionEventQueue's InternalQueue.
//
// NextEventName is thread-safe.
func (q *SessionEventQueue) NextEventName() (domain.EventName, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// What do we do if the queue is empty?
	if len(q.InternalQueue) == 0 {
		return domain.EventInvalidName, false
	}

	return q.InternalQueue.Peek().Name, true
}

// NextEventGlobalEventIndex returns the GlobalEventIndex of the next domain.Event
// in this SessionEventQueue's InternalQueue.
//
// NextEventGlobalEventIndex is thread-safe.
func (q *SessionEventQueue) NextEventGlobalEventIndex() (uint64, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// What do we do if the queue is empty?
	if len(q.InternalQueue) == 0 {
		return 0, false
	}

	return q.InternalQueue.Peek().GlobalIndex, true
}

// HasEventsForTick returns true if there are events available for the specified tick; otherwise return false.
//
// HasEventsForTick is thread-safe.
func (q *SessionEventQueue) HasEventsForTick(tick time.Time) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	length := q.InternalQueue.Len()
	if length == 0 {
		return false
	}

	timestamp, _ := q.unsafeNextEventTimestamp()
	return tick == timestamp || timestamp.Before(tick)
}
