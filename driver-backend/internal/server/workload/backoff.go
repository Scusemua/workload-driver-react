package workload

import (
	"math/rand"
	"time"
)

type ExponentialBackoff struct {
	// BaseDuration is the initial duration / sleep interval.
	BaseDuration time.Duration

	// BaseDuration is multiplied by Multiplier each subsequent iteration.
	//
	// Factor should not be negative.
	Multiplier float64

	// Jitter defines the upper bound of a random additive quantity that is added to the duration.
	// The quantity chosen uniformly at random from 0 - Jitter.
	Jitter float64

	// NumAttempts is the current number of attempts.
	NumAttempts int

	// MaxDuration is the maximum duration.
	MaxDuration time.Duration
}

func (e *ExponentialBackoff) ComputeJitter(units time.Duration) time.Duration {
	jitter := rand.Float64() * e.Jitter
	return time.Duration(jitter) * units
}
