package domain

type WorkloadManager interface {
	WorkloadProvider
	WorkloadSubscriptionHandler
}

// Provides access to the currently-registered workloads.
type WorkloadProvider interface {
	// Return a slice containing all currently-registered workloads (at the time that the method is called).
	// The workloads within this slice should not be modified by the caller.
	GetWorkloads() []Workload
}

type WorkloadSubscriptionHandler interface {
	// Register a new subscriber.
	AddSubscription() error
	// Unregister an existing subscriber.
	RemoveSubscription() error
}
