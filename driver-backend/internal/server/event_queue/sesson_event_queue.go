package event_queue

import (
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
	InternalQueue domain.BasicEventHeap

	// HeapIndex is the index of this SessionEventQueue with respect to other SessionEventQueue instances.
	HeapIndex int

	mu sync.Mutex
}

func (q *SessionEventQueue) SetIndex(idx int) {
	q.HeapIndex = idx
}

func (q *SessionEventQueue) GetIndex() int {
	return q.HeapIndex
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
