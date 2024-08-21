package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type ResourceRequest struct {
	memoryGB         decimal.Decimal // The amount of memory (in GB) required by the session.
	cpus             decimal.Decimal // The number of vCPUs required by the session.
	gpus             decimal.Decimal // The number of GPUs required by the session.
	requestedGpuName string          // The name of the specific GPU requested by the session.
}

func NewResourceRequest(vcpus float64, memGB float64, gpus float64, requestedGpuName string) *ResourceRequest {
	return &ResourceRequest{
		cpus:             decimal.NewFromFloat(vcpus),
		gpus:             decimal.NewFromFloat(gpus),
		memoryGB:         decimal.NewFromFloat(memGB),
		requestedGpuName: requestedGpuName,
	}
}

func (s ResourceRequest) String() string {
	return fmt.Sprintf("ContainerResourceRequest[vCPUs=%s, GPUs=%s, Memory=%sGB, GpuName=%s]", s.cpus.StringFixed(2), s.gpus.StringFixed(2), s.memoryGB.StringFixed(2), s.requestedGpuName)
}
