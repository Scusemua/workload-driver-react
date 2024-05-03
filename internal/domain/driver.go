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

	ToggleDebugLogging(enabled bool) *Workload

	GetWorkload() *Workload

	GetWorkloadPreset() WorkloadPreset

	GetWorkloadRegistrationRequest() *WorkloadRegistrationRequest

	// Returns nil if the workload could not be registered.
	RegisterWorkload(workloadRegistrationRequest *WorkloadRegistrationRequest) (*Workload, error)

	// Write an error back to the client.
	WriteError(c *gin.Context, errorMessage string)

	// Return true if the workload has completed; otherwise, return false.
	IsWorkloadComplete() bool

	ID() string

	// Stop a workload that's already running/in-progress.
	// Returns nil on success, or an error if one occurred.
	StopWorkload() error

	StopChan() chan interface{}

	// This should be called from its own goroutine.
	// Accepts a waitgroup that is used to notify the caller when the workload has entered the 'WorkloadRunning' state.
	DriveWorkload(wg *sync.WaitGroup)

	ProcessWorkload(wg *sync.WaitGroup)

	EventQueue() EventQueueService
}
