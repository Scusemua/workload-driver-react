package event_queue

import (
	"container/heap"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
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

	mu sync.Mutex
}

// NewSessionEventQueue creates a new SessionEventQueue struct and returns a pointer to it.
func NewSessionEventQueue(sessionId string) *SessionEventQueue {
	queue := &SessionEventQueue{
		SessionId:     sessionId,
		InternalQueue: make(domain.EventHeap, 0),
		HeapIndex:     -1,
	}

	return queue
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

	return heap.Pop(&q.InternalQueue).(*domain.Event)
}

// Push pushes the specified *domain.Event into the InternalQueue of the SessionEventQueue.
func (q *SessionEventQueue) Push(evt *domain.Event) {
	heap.Push(&q.InternalQueue, evt)
}

// Peek returns -- but does not remove -- the next *domain.Event in the InternalQueue of the SessionEventQueue.
func (q *SessionEventQueue) Peek() *domain.Event {
	if q.Len() == 0 {
		return nil
	}

	return q.InternalQueue.Peek()
}

// NextEventTimestamp returns the Timestamp of the next domain.Event in this SessionEventQueue's InternalQueue.
//
// The SessionEventQueue's Delay field is added to the domain.Event's Timestamp field before the Timestamp is returned.
func (q *SessionEventQueue) NextEventTimestamp() time.Time {
	q.mu.Lock()
	defer q.mu.Unlock()

	// What do we do if the queue is empty?
	if len(q.InternalQueue) == 0 {
		panic("SessionEventQueue is empty.")
	}

	return q.InternalQueue.Peek().Timestamp.Add(q.Delay)
}

// NextEventName returns the Name of the next domain.Event in this SessionEventQueue's InternalQueue.
func (q *SessionEventQueue) NextEventName() domain.EventName {
	q.mu.Lock()
	defer q.mu.Unlock()

	// What do we do if the queue is empty?
	if len(q.InternalQueue) == 0 {
		panic("SessionEventQueue is empty.")
	}

	return q.InternalQueue.Peek().Name
}

// NextEventGlobalEventIndex returns the GlobalEventIndex of the next domain.Event
// in this SessionEventQueue's InternalQueue.
func (q *SessionEventQueue) NextEventGlobalEventIndex() uint64 {
	q.mu.Lock()
	defer q.mu.Unlock()

	// What do we do if the queue is empty?
	if len(q.InternalQueue) == 0 {
		panic("SessionEventQueue is empty.")
	}

	return q.InternalQueue.Peek().GlobalIndex
}
