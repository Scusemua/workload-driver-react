package domain

import (
	"fmt"
)

type ResourceRequest struct {
	MemoryMB         float64 `json:"mem_mb"`             // The amount of memory (in MB) required by the session.
	Cpus             float64 `json:"cpus"`               // The number of vCPUs required by the session.
	Gpus             int     `json:"gpus"`               // The number of GPUs required by the session.
	RequestedGpuName string  `json:"gpu_type,omitempty"` // The name of the specific GPU requested by the session.
}

func NewResourceRequest(vcpus float64, memMB float64, gpus int, requestedGpuName string) *ResourceRequest {
	return &ResourceRequest{
		Cpus:             vcpus,
		Gpus:             gpus,
		MemoryMB:         memMB,
		RequestedGpuName: requestedGpuName,
	}
}

func (s ResourceRequest) String() string {
	return fmt.Sprintf("ContainerResourceRequest[vCPUs=%.2f, GPUs=%d, Memory=%.2fMB, GpuName=%s]", s.Cpus, s.Gpus, s.MemoryMB, s.RequestedGpuName)
}
