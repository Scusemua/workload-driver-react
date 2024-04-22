package driver

import (
	"fmt"
	"sync/atomic"
	"time"
)

// A Ticker holds a channel that delivers “ticks” of a clock
// at intervals.
type Ticker struct {
	// The channel on which the ticks are delivered.
	TickDelivery <-chan time.Time

	baseTriggerHandler
	lastTick          time.Time
	step              time.Duration
	tickChannel       chan time.Time
	done              chan struct{}
	ticksHandled      atomic.Int32
	numOnTriggerCalls atomic.Int32
	numDefault        atomic.Int32
}

// NewSyncTicker returns a synchronous Ticker.
// On each tick, the handler that listens on C must call Ticker.Done() after processed the tick.
func NewSyncTicker(d time.Duration, id string) *Ticker {
	return newTicker(d, true, id)
}

// NewTicker returns a new Ticker containing a channel that will send
// the time on the channel after each tick. The period of the ticks is
// specified by the duration argument. The ticker will adjust the time
// interval or drop ticks to make up for slow receivers.
// The duration d must be greater than zero; if not, NewTicker will
// simply adjust d to minimum tick interval.
func NewTicker(d time.Duration, id string) *Ticker {
	return newTicker(d, false, id)
}

func newTicker(d time.Duration, wait bool, id string) *Ticker {
	ticker := &Ticker{
		lastTick:    CurrentTick.GetClockTime(),
		step:        d,
		tickChannel: make(chan time.Time),
	}
	ticker.ticksHandled.Store(0)
	ticker.numOnTriggerCalls.Store(0)
	ticker.numDefault.Store(0)
	ticker.setId(id)
	ticker.TickDelivery = ticker.tickChannel
	if wait {
		ticker.done = make(chan struct{})
	}
	ClockTrigger.AddHandler(ticker)
	return ticker
}

func (ticker *Ticker) onTrigger(t time.Time) {
	ticker.numOnTriggerCalls.Add(1)
	if t.Sub(ticker.lastTick) < ticker.step {
		return
	}
	ticker.lastTick = t

	// Add to channel
	ticker.tickChannel <- t
	// sync ticker will wait for the handler to call Done()
	if ticker.done != nil {
		<-ticker.done
	}
	ticker.ticksHandled.Add(1)
}

func (ticker *Ticker) String() string {
	return fmt.Sprintf("Ticker[id: %s, doneIsNil: %v, step: %v, lastTick: %v, ticksHandled: %d, numOnTriggerCalls: %d, numDefault: %d]", ticker._id, (ticker.done == nil), ticker.step, ticker.lastTick, ticker.ticksHandled.Load(), ticker.numOnTriggerCalls.Load(), ticker.numDefault.Load())
}

func (ticker *Ticker) Done() {
	if ticker.done != nil {
		ticker.done <- struct{}{}
	}
}

// Stop turns off a ticker. After Stop, no more ticks will be sent.
// Stop does not close the channel, to prevent a concurrent goroutine
// reading from the channel from seeing an erroneous "tick".
func (ticker *Ticker) Stop() {
	ClockTrigger.RemoveHandler(ticker)
}

// Reset stops a ticker and resets its period to the specified duration.
// The next tick will arrive after the new period elapses.
func (ticker *Ticker) Reset(d time.Duration) {
	ticker.lastTick = CurrentTick.GetClockTime()
	ticker.step = d
}
