package driver

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/zhangjyr/hashmap"
)

var (
	ErrInvalidClockOperation = errors.New("illegal clock operation attempted")
)

type simulationClockImpl struct {
	clockTime  time.Time
	clockMutex sync.RWMutex
}

func NewSimulationClock() *simulationClockImpl {
	return &simulationClockImpl{
		clockTime: time.Unix(0, 0),
	}
}

// Return the current clock time.
func (sc *simulationClockImpl) GetClockTime() time.Time {
	sc.clockMutex.RLock()
	defer sc.clockMutex.RUnlock()

	return sc.clockTime
}

// Set the clock to the given timestamp. Return the updated value.
func (sc *simulationClockImpl) SetClockTime(t time.Time) time.Time {
	sc.clockMutex.Lock()
	defer sc.clockMutex.Unlock()

	sc.clockTime = t
	return sc.clockTime
}

// Set the clock to the given timestamp, verifying that the new timestamp is either equal to or occurs after the old one.
// Return a tuple where the first element is the new time, and the second element is the difference between the new time and the old time.
func (sc *simulationClockImpl) IncreaseClockTimeTo(t time.Time) (time.Time, time.Duration, error) {
	sc.clockMutex.Lock()
	defer sc.clockMutex.Unlock()

	// Verify that the new timestamp is either equal to the old/current one or that the new timestamp occurs after the old/current one.
	if !(t.After(sc.clockTime)) && sc.clockTime != t {
		return time.Time{}, time.Duration(0), fmt.Errorf("%w: attempting to increase clock time from %v to %v (not an increase)", ErrInvalidClockOperation, sc.clockTime, t)
	}

	difference := t.Sub(sc.clockTime)

	sc.clockTime = t
	return sc.clockTime, difference, nil // sc.clockTime
}

// Increment the clock by the given amount. Return the updated value.
func (sc *simulationClockImpl) IncrementClockBy(amount time.Duration) (time.Time, error) {
	sc.clockMutex.Lock()
	defer sc.clockMutex.Unlock()

	if amount < 0 {
		return time.Time{}, fmt.Errorf("%w: attempting to increment clock time by negative duration %v", ErrInvalidClockOperation, amount)
	}

	sc.clockTime = sc.clockTime.Add(amount)
	return sc.clockTime, nil
}

type triggerHandler interface {
	id() string
	setId(id string)
	onTrigger(time.Time)
}

type baseTriggerHandler struct {
	_id string
}

func (h *baseTriggerHandler) id() string {
	return h._id
}

func (h *baseTriggerHandler) setId(id string) {
	h._id = id
}

// A Trigger is a struct that registers listeners.
// When the Trigger is "triggered", it invokes all of its listeners.
// This is intended to be used to trigger events based on the simulation clock advancing forward tick-by-tick.
type Trigger struct {
	handlers *hashmap.HashMap

	// The number of times the Trigger() function has been called on this Trigger struct.
	numTimesActivated atomic.Int32

	// The number of times this Trigger struct has called the onTrigger() function on a handler.
	// Assuming all handlers are added before any calls to Trigger::Trigger(), this value should be
	// a multiple of `numTimesActivated`. (Specifically, it should equal `handlers.Len() * numTimesActivated`).
	numTriggersFired atomic.Int32
}

func NewTrigger() *Trigger {
	trigger := &Trigger{
		handlers: hashmap.New(2),
	}
	trigger.numTimesActivated.Store(0)
	trigger.numTriggersFired.Store(0)
	return trigger
}

// Stop turns off a ticker. After Stop, no more ticks will be sent.
// Stop does not close the channel, to prevent a concurrent goroutine
// reading from the channel from seeing an erroneous "tick".
func (t *Trigger) Stop(ticker *Ticker) {
	t.RemoveHandler(ticker)
}

// NewSyncTicker returns a synchronous Ticker.
// On each tick, the handler that listens on C must call Ticker.Done() after processed the tick.
func (t *Trigger) NewSyncTicker(d time.Duration, id string, clock domain.SimulationClock) *Ticker {
	return t.createAndAddTicker(d, true, id, clock)
}

func (t *Trigger) createAndAddTicker(d time.Duration, wait bool, id string, clock domain.SimulationClock) *Ticker {
	ticker := &Ticker{
		lastTick:    clock.GetClockTime(),
		step:        d,
		tickChannel: make(chan time.Time),
		clock:       clock,
	}
	ticker.ticksHandled.Store(0)
	ticker.numOnTriggerCalls.Store(0)
	ticker.numDefault.Store(0)
	ticker.setId(id)
	ticker.TickDelivery = ticker.tickChannel
	if wait {
		ticker.done = make(chan struct{})
	}
	t.AddHandler(ticker)
	return ticker
}

func (c *Trigger) AddHandler(h triggerHandler) {
	id := h.id()
	if id == "" {
		id = uuid.NewString()
		h.setId(id)
	}
	c.handlers.Set(id, h)
}

func (c *Trigger) RemoveHandler(h triggerHandler) {
	c.handlers.Del(h.id())
}

func (c *Trigger) Trigger(ts time.Time) {
	// fmt.Printf("Triggering %v. There are %d handlers.\n", ts, c.handlers.Len())
	for keyValue := range c.handlers.Iter() {
		keyValue.Value.(triggerHandler).onTrigger(ts)

		c.numTriggersFired.Add(1)
	}

	c.numTimesActivated.Add(1)
}
