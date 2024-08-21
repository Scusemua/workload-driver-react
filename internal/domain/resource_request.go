package domain

import (
	"fmt"
)

type ResourceRequest struct {
	// memoryGB         decimal.Decimal // The amount of memory (in GB) required by the session.
	// cpus             decimal.Decimal // The number of vCPUs required by the session.
	// gpus             decimal.Decimal // The number of GPUs required by the session.

	MemoryGB         float64 `json:"mem_gb"`             // The amount of memory (in GB) required by the session.
	Cpus             float64 `json:"cpus"`               // The number of vCPUs required by the session.
	Gpus             int     `json:"gpus"`               // The number of GPUs required by the session.
	RequestedGpuName string  `json:"gpu_type,omitempty"` // The name of the specific GPU requested by the session.
}

func NewResourceRequest(vcpus float64, memGB float64, gpus int, requestedGpuName string) *ResourceRequest {
	return &ResourceRequest{
		// cpus:             decimal.NewFromFloat(vcpus),
		// gpus:             decimal.NewFromFloat(gpus),
		// memoryGB:         decimal.NewFromFloat(memGB),
		Cpus:             vcpus,
		Gpus:             gpus,
		MemoryGB:         memGB,
		RequestedGpuName: requestedGpuName,
	}
}

func (s ResourceRequest) String() string {
	return fmt.Sprintf("ContainerResourceRequest[vCPUs=%.2f, GPUs=%d, Memory=%.2fGB, GpuName=%s]", s.Cpus, s.Gpus, s.MemoryGB, s.RequestedGpuName)
}
