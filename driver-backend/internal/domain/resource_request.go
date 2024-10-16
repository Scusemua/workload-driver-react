package domain

import (
	"fmt"
)

type ResourceRequest struct {
	MemoryMB         float64 `json:"mem_mb"`             // The amount of memory (in MB) required by the session.
	Cpus             float64 `json:"cpus"`               // The number of vCPUs required by the session.
	Gpus             int     `json:"gpus"`               // The number of GPUs required by the session.
	VRAM             float64 `json:"vram"`               // The amount of VRAM (i.e., GPU memory) required in GB.
	RequestedGpuName string  `json:"gpu_type,omitempty"` // The name of the specific GPU requested by the session.
}

func NewResourceRequest(vcpus float64, memMB float64, gpus int, vram float64, requestedGpuName string) *ResourceRequest {
	return &ResourceRequest{
		Cpus:             vcpus,
		Gpus:             gpus,
		VRAM:             vram,
		MemoryMB:         memMB,
		RequestedGpuName: requestedGpuName,
	}
}

func (s ResourceRequest) String() string {
	return fmt.Sprintf("ContainerResourceRequest[vCPUs=%.2f, GPUs=%d, VRAM=%.2fGB, Memory=%.2fMB, GpuName=%s]", s.Cpus, s.Gpus, s.VRAM, s.MemoryMB, s.RequestedGpuName)
}
