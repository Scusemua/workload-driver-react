package domain

import (
	"sync"

	"github.com/gin-gonic/gin"
)

type WorkloadDriver interface {
	EventConsumer

	// Acquire the Driver's mutex externally.
	//
	// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
	LockDriver()

	// Attempt to acquire the Driver's mutex externally.
	// Returns true on successful acquiring of the lock. If lock was not acquired, return false.
	//
	// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
	TryLockDriver() bool

	// Release the Driver's mutex externally.
	UnlockDriver()

	// Signal that the workload is done (being parsed) by the generator/synthesizer.
	DoneChan() chan interface{}

	// Toggle debug logging on/off (depending on the value of the 'enabled' parameter)
	// for the workload that is associated with/managed by this workload driver.
	ToggleDebugLogging(enabled bool) Workload

	// Return the channel used to report critical errors encountered while executing the workload.
	GetErrorChan() chan<- error

	// Start the Workload that is associated with/managed by this workload driver.
	//
	// If the workload is already running, then an error is returned.
	// Likewise, if the workload was previously running but has already stopped, then an error is returned.
	StartWorkload() error

	// Return the workload that is associated with/managed by this workload driver.
	GetWorkload() Workload

	// Get the workload preset of the workload that is associated with/managed by this workload driver.
	// If the workload associated with/managed by this workload driver does not have a preset
	// (i.e., if it is a template-based workload), then nil is returned.
	GetWorkloadPreset() *WorkloadPreset

	// Get the request that was submitted when the workload associated with/managed by this workload driver was first registered.
	GetWorkloadRegistrationRequest() *WorkloadRegistrationRequest

	// Returns nil if the workload could not be registered.
	RegisterWorkload(workloadRegistrationRequest *WorkloadRegistrationRequest) (Workload, error)

	// Write an error back to the client.
	WriteError(c *gin.Context, errorMessage string)

	// Return true if the workload has completed; otherwise, return false.
	IsWorkloadComplete() bool

	// Return the unique ID of this workload driver.
	// This is not necessarily the same as the workload's unique ID (TODO: or is it?).
	ID() string

	// Stop a workload that's already running/in-progress.
	// Returns nil on success, or an error if one occurred.
	StopWorkload() error

	// Return the channel used to tell the workload to stop.
	StopChan() chan<- interface{}

	// This should be called from its own goroutine.
	// Accepts a waitgroup that is used to notify the caller when the workload has entered the 'WorkloadRunning' state.
	// This issues clock ticks as events are submitted.
	DriveWorkload(wg *sync.WaitGroup)

	// This should be called from its own goroutine.
	// Accepts a waitgroup that is used to notify the caller when the workload has entered the 'WorkloadRunning' state.
	// This processes events in response to clock ticks.
	ProcessWorkload(wg *sync.WaitGroup)

	// The event queue for this workload.
	EventQueue() EventQueueService

	// Return the WebSocket connection on which this workload was registered by a remote client and on/through which updates about the workload are reported.
	WebSocket() ConcurrentWebSocket

	// Return the current tick of the workload.
	CurrentTick() SimulationClock

	// Return the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
	ClockTime() SimulationClock
}
