package workload

import (
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"time"
)

type Session interface {
	GetId() string
	// GetMaxResourceRequest returns the ResourceRequest encoding the maximum amount of each type of resource
	// that the Session may use at some point during its lifetime.
	GetMaxResourceRequest() *domain.ResourceRequest
	// GetCurrentResourceRequest returns the ResourceRequest encoding the Session's current resource usage.
	GetCurrentResourceRequest() *domain.ResourceRequest
	// SetCurrentResourceRequest updates the ResourceRequest encoding the Session's current resource usage.
	SetCurrentResourceRequest(*domain.ResourceRequest)
	GetTrainingsCompleted() int
	GetState() domain.SessionState
	GetCreatedAt() time.Time
	GetTrainingStartedAt() time.Time
	GetTrainings() []*domain.TrainingEvent
	GetStderrIoPubMessages() []string
	GetStdoutIoPubMessages() []string
	AddStderrIoPubMessage(message string)
	AddStdoutIoPubMessage(message string)
	// NumFailedTicks returns the number of times that this Session failed to process all of its events during a tick
	// of a workload.
	NumFailedTicks() int
	// TickFailed records that the target Session failed to process all of its events during a tick
	// of a workload. It returns the updated value (i.e., the same value that a subsequent call to NumFailedTicks
	// would return for the target Session).
	TickFailed() int
	SetState(domain.SessionState) error
	GetAndIncrementTrainingsCompleted() int
}
