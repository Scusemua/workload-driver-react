package workload

import (
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

// This struct is not threadsafe.
//
// There's no reason for this to be attempted, but a responseBuilder
// struct should only ever be modified/accessed from a single goroutine.
type responseBuilder struct {
	messageId         string            // Identifier for the message. Optionally specified when the builder is created, or alternatively the builder will generate the ID automatically.
	newWorkloads      []domain.Workload // Workloads that are newly-created.
	modifiedWorkloads []domain.Workload // Workloads that already exist and have been updated.
	deletedWorkloads  []domain.Workload // Workloads that are being deleted.
}

// Pass an empty string for the 'msgId' parameter in order to have the message ID to be automatically generated (as a UUID).
func newResponseBuilder(msgId string) *responseBuilder {
	if len(msgId) == 0 {
		msgId = uuid.NewString()
	}

	return &responseBuilder{
		messageId: msgId,
	}
}

func (b *responseBuilder) WithNewWorkloads(newWorkloads []domain.Workload) *responseBuilder {
	b.newWorkloads = newWorkloads
	return b
}

func (b *responseBuilder) WithNewWorkload(newWorkload domain.Workload) *responseBuilder {
	if b.newWorkloads == nil {
		b.newWorkloads = make([]domain.Workload, 0)
	}

	b.newWorkloads = append(b.newWorkloads, newWorkload)
	return b
}

func (b *responseBuilder) WithModifiedWorkload(modifiedWorkload domain.Workload) *responseBuilder {
	if b.modifiedWorkloads == nil {
		b.modifiedWorkloads = make([]domain.Workload, 0)
	}

	b.modifiedWorkloads = append(b.modifiedWorkloads, modifiedWorkload)
	return b
}

func (b *responseBuilder) WithModifiedWorkloads(modifiedWorkloads []domain.Workload) *responseBuilder {
	b.modifiedWorkloads = modifiedWorkloads
	return b
}

func (b *responseBuilder) WithDeletedWorkloads(deletedWorkloads []domain.Workload) *responseBuilder {
	b.deletedWorkloads = deletedWorkloads
	return b
}

func (b *responseBuilder) BuildResponse() *domain.WorkloadResponse {
	response := &domain.WorkloadResponse{
		MessageId:         b.messageId,
		NewWorkloads:      b.newWorkloads,
		ModifiedWorkloads: b.modifiedWorkloads,
		DeletedWorkloads:  b.deletedWorkloads,
	}

	if response.NewWorkloads == nil {
		response.NewWorkloads = make([]domain.Workload, 0)
	}

	if response.ModifiedWorkloads == nil {
		response.ModifiedWorkloads = make([]domain.Workload, 0)
	}

	if response.DeletedWorkloads == nil {
		response.DeletedWorkloads = make([]domain.Workload, 0)
	}

	return response
}
