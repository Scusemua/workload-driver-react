package domain

import (
	"errors"

	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
)

const (
	// Base of the API endpoint.
	BASE_API_GROUP_ENDPOINT = "/api"

	// Used internally (by the frontend) to get the current kubernetes nodes from the backend.
	KUBERNETES_NODES_ENDPOINT = "/nodes"

	// Used internally (by the frontend) to adjust the vGPUs offered by a particular kubernetes nodes.
	ADJUST_VGPUS_ENDPOINT = "/vgpus"

	// Used to get/set resource specs of kernels.
	RESOURCE_SPEC_ENDPOINT = "/resourcespecs"

	// Used internally (by the frontend) to get the system config from the backend.
	SYSTEM_CONFIG_ENDPOINT = "/config"

	// Used internally (by the frontend) to trigger kernel replica migrations.
	MIGRATION_ENDPOINT = "/migrate"

	// Used to stream logs to the frontend from Kubernetes.
	LOGS_ENDPOINT = "/logs"

	// Used internally (by the frontend) to trigger the start of a new workload or retrieve the list of workloads.
	WORKLOAD_ENDPOINT = "/workload"

	// Used internally (by the frontend) to get the current set of Jupyter kernel specs from the backend.
	KERNEL_SPEC_ENDPOINT = "/kernelspecs"

	// Used internally (by the frontend) to get the current set of Jupyter kernels from the backend.
	GET_KERNELS_ENDPOINT = "/get-kernels"

	// Used internally (by the frontend) to get the list of available workload presets from the backend.
	WORKLOAD_PRESET_ENDPOINT = "/workload-presets"

	JUPYTER_GROUP_ENDPOINT        = "/jupyter"
	JUPYTER_START_KERNEL_ENDPOINT = "/start"
	JUPYTER_STOP_KERNEL_ENDPOINT  = "/stop"
)

var (
	KernelStatuses      = []string{"unknown", "starting", "idle", "busy", "terminating", "restarting", "autorestarting", "dead"}
	ErrEmptyGatewayAddr = errors.New("cluster gateway IP address cannot be the empty string")
)

type KernelRefreshCallback func([]*gateway.DistributedJupyterKernel)

type Server interface {
	Serve() error // Run the server. This is a blocking call.
}
