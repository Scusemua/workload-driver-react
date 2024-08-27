package domain

import "github.com/gin-gonic/gin"

// Central hub that manages/maintains all of the different workloads.
type WorkloadManager interface {
	// Return a function that can handle WebSocket requests for workload operations.
	//
	// This simply returns the handler function of the WorkloadWebsocketHandler struct of the WorkloadManager.
	GetWorkloadWebsocketHandler() gin.HandlerFunc

	// Return a slice containing all currently-registered workloads (at the time that the method is called).
	// The workloads within this slice should not be modified by the caller.
	GetWorkloads() []Workload

	// Return a map from Workload ID to Workload struct containing workloads that are active when the method is called.
	GetActiveWorkloads() map[string]Workload

	// Return the workload driver associated with the given workload ID.
	// If there is no driver associated with the provided workload ID, then nil is returned.
	GetWorkloadDriver(workloadId string) WorkloadDriver

	// Toggle debug logging on or off (depending on the value of the 'enabled' parameter) for the specified workload.
	//
	// If successful, then this returns the updated workload.
	// If there is no workload with the specified ID, then an error is returned.
	ToggleDebugLogging(workloadId string, enabled bool) (Workload, error)

	// Start the workload with the specified ID.
	// The workload must have already been registered.
	//
	// If successful, then this returns the updated workload.
	// If there is no workload with the specified ID, then an error is returned.
	// Likewise, if the specified workload is either already-running or has already been stopped, then an error is returned.
	StartWorkload(workloadId string) (Workload, error)

	// Stop the workload with the specified ID.
	// The workload must have already been registered and should be actively-running.
	//
	// If successful, then this returns the updated workload.
	// If there is no workload with the specified ID, or the specified workload is not actively-running, then an error is returned.
	StopWorkload(workloadId string) (Workload, error)

	// Register a new workload.
	RegisterWorkload(request *WorkloadRegistrationRequest, ws ConcurrentWebSocket) (Workload, error)
}