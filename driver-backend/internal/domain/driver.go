package domain

import (
	"github.com/gin-gonic/gin"
)

// WorkloadDriver drives a workload.
// Each WorkloadDriver can be assigned a single workload via the WorkloadDriver::RegisterWorkload method.
type WorkloadDriver interface {
	EventConsumer

	// WorkloadExecutionCompleteChan signals that the workload is done (being parsed) by the generator/synthesizer.
	WorkloadExecutionCompleteChan() chan interface{}

	// ToggleDebugLogging toggles debug logging on/off (depending on the value of the 'enabled' parameter)
	// for the workload that is associated with/managed by this workload driver.
	ToggleDebugLogging(enabled bool) Workload

	// GetErrorChan Returns the channel used to report critical errors encountered while executing the workload.
	GetErrorChan() chan<- error

	// StartWorkload Starts the Workload that is associated with/managed by this workload driver.
	//
	// If the workload is already running, then an error is returned.
	// Likewise, if the workload was previously running but has already stopped, then an error is returned.
	StartWorkload() error

	//// GetWorkload Returns the workload that is associated with/managed by this workload driver.
	//GetWorkload() Workload

	// GetWorkloadPreset gets the workload preset of the workload that is associated with/managed by this workload driver.
	// If the workload associated with/managed by this workload driver does not have a preset
	// (i.e., if it is a template-based workload), then nil is returned.
	GetWorkloadPreset() *WorkloadPreset

	// GetWorkloadRegistrationRequest returns the request that was submitted when the workload associated with/managed
	// by this workload driver was first registered.
	GetWorkloadRegistrationRequest() *WorkloadRegistrationRequest

	// RegisterWorkload returns nil if the workload could not be registered.
	// Only one workload may be registered with a WorkloadDriver struct.
	RegisterWorkload(workloadRegistrationRequest *WorkloadRegistrationRequest) (Workload, error)

	// WriteError writes an error back to the client.
	WriteError(c *gin.Context, errorMessage string)

	// IsWorkloadComplete returns true if the workload has completed; otherwise, return false.
	IsWorkloadComplete() bool

	// ID returns the unique ID of this workload driver.
	// This is not necessarily the same as the workload's unique ID (TODO: or is it?).
	ID() string

	// StopWorkload stops a workload that's already running/in-progress.
	// returns nil on success, or an error if one occurred.
	StopWorkload() error

	// StopChan returns the channel used to tell the workload to stop.
	StopChan() chan<- interface{}

	// DriveWorkload should be called from its own goroutine.
	// This issues clock ticks as events are submitted.
	DriveWorkload()

	// ProcessWorkload should be called from its own goroutine.
	//
	// If there is a critical error that causes the workload to be terminated prematurely/aborted, then that error is returned.
	// If the workload is able to complete successfully, then nil is returned.
	ProcessWorkloadEvents()

	// WebSocket returns the WebSocket connection on which this workload was registered by a remote client and on/through which updates about the workload are reported.
	WebSocket() ConcurrentWebSocket

	// CurrentTick returns the current tick of the workload.
	CurrentTick() SimulationClock

	// ClockTime returns the current clock time of the workload, which will be sometime between currentTick and currentTick + tick_duration.
	ClockTime() SimulationClock
}
