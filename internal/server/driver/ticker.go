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

// Reset stops a ticker and resets its period to the specified duration.
// The next tick will arrive after the new period elapses.
func (ticker *Ticker) Reset(d time.Duration) {
	ticker.lastTick = CurrentTick.GetClockTime()
	ticker.step = d
}
