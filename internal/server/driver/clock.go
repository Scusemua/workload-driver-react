package driver

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/zhangjyr/hashmap"
)

var (
	ClockTrigger *Trigger

	CurrentTick = NewSimulationClock()
	ClockTime   = NewSimulationClock()
)

type SimulationClock struct {
	clockTime  time.Time
	clockMutex sync.RWMutex
}

func NewSimulationClock() *SimulationClock {
	return &SimulationClock{
		clockTime: time.Unix(0, 0),
	}
}

// Return the current clock time.
func (sc *SimulationClock) GetClockTime() time.Time {
	sc.clockMutex.RLock()
	defer sc.clockMutex.RUnlock()

	return sc.clockTime
}

// Set the clock to the given timestamp. Return the updated value.
func (sc *SimulationClock) SetClockTime(t time.Time) time.Time {
	sc.clockMutex.Lock()
	defer sc.clockMutex.Unlock()

	sc.clockTime = t
	return sc.clockTime
}

// Set the clock to the given timestamp, verifying that the new timestamp is either equal to or occurs after the old one.
// Return a tuple where the first element is the new time, and the second element is the difference between the new time and the old time.
func (sc *SimulationClock) IncreaseClockTimeTo(t time.Time) (time.Time, time.Duration) {
	sc.clockMutex.Lock()
	defer sc.clockMutex.Unlock()

	// Verify that the new timestamp is either equal to the old/current one or that the new timestamp occurs after the old/current one.
	if !(t.After(sc.clockTime)) && sc.clockTime != t {
		panic(fmt.Sprintf("Attempting to increase clock time from %v to %v (not an increase).", sc.clockTime, t))
	}

	difference := t.Sub(sc.clockTime)

	sc.clockTime = t
	return sc.clockTime, difference // sc.clockTime
}

// Increment the clock by the given amount. Return the updated value.
func (sc *SimulationClock) IncrementClockBy(amount time.Duration) time.Time {
	sc.clockMutex.Lock()
	defer sc.clockMutex.Unlock()
	sc.clockTime = sc.clockTime.Add(amount)
	return sc.clockTime
}

func init() {
	ClockTrigger = NewTrigger()
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
