package workload

import (
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

type responseBuilder struct {
	messageId         string
	newWorkloads      []domain.Workload
	modifiedWorkloads []domain.Workload
	deletedWorkloads  []domain.Workload
	// messageIndex      int32
}

func newResponseBuilder() *responseBuilder {
	return &responseBuilder{
		messageId: uuid.NewString(),
		// messageIndex: messageIndex,
	}
}

func (b *responseBuilder) WithNewWorkloads(newWorkloads []domain.Workload) *responseBuilder {
	b.newWorkloads = newWorkloads
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
		// MessageIndex:      b.messageIndex,
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
