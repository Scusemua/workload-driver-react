package domain

import "time"

type SimulationClock interface {
	// Return the current clock time.
	GetClockTime() time.Time

	// Set the clock to the given timestamp. Return the updated value.
	SetClockTime(t time.Time) time.Time

	// Set the clock to the given timestamp, verifying that the new timestamp is either equal to or occurs after the old one.
	// Return a tuple where the first element is the new time, and the second element is the difference between the new time and the old time.
	IncreaseClockTimeTo(t time.Time) (time.Time, time.Duration, error)

	// Increment the clock by the given amount. Return the updated value.
	IncrementClockBy(amount time.Duration) (time.Time, error)
}
