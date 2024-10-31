package workload

import (
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

// This struct is not thread-safe.
//
// There's no reason for this to be attempted, but a responseBuilder
// struct should only ever be modified/accessed from a single goroutine.
type responseBuilder struct {
	messageId         string // Identifier for the message. Optionally specified when the builder is created, or alternatively the builder will generate the ID automatically.
	op                string
	newWorkloads      []domain.Workload         // Workloads that are newly-created.
	modifiedWorkloads []domain.Workload         // Modified workloads sent in their entirety.
	patchedWorkloads  []*domain.PatchedWorkload // Modified workloads sent as JSON merge patches.
	deletedWorkloads  []domain.Workload         // Workloads that are being deleted.
}

// Pass an empty string for the 'msgId' parameter in order to have the message ID to be automatically generated (as a UUID).
func newResponseBuilder(msgId string, op string) *responseBuilder {
	if len(msgId) == 0 {
		msgId = uuid.NewString()
	}

	return &responseBuilder{
		messageId: msgId,
		op:        op,
	}
}

func (b *responseBuilder) AddModifiedWorkloadAsPatch(patch []byte, workloadId string) {
	patchedWorkload := &domain.PatchedWorkload{
		Patch:      string(patch),
		WorkloadId: workloadId,
	}

	b.patchedWorkloads = append(b.patchedWorkloads, patchedWorkload)
}

func (b *responseBuilder) AddModifiedWorkload(workload domain.Workload) {
	b.modifiedWorkloads = append(b.modifiedWorkloads, workload)
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
	if b.modifiedWorkloads == nil {
		b.modifiedWorkloads = modifiedWorkloads
	} else {
		for _, modifiedWorkload := range modifiedWorkloads {
			b.modifiedWorkloads = append(b.modifiedWorkloads, modifiedWorkload)
		}
	}
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
		PatchedWorkloads:  b.patchedWorkloads,
		Operation:         b.op,
		Status:            domain.ResponseStatusOK,
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

	if response.PatchedWorkloads == nil {
		response.PatchedWorkloads = make([]*domain.PatchedWorkload, 0)
	}

	return response
}
