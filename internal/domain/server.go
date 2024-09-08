package domain

import (
	"errors"

	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
)

const (
	// BASE_API_GROUP_ENDPOINT is the Base of the API endpoint.
	BASE_API_GROUP_ENDPOINT = "/api"

	// TEST_API_GROUP_ENDPOINT is used for testing/debugging.
	TEST_API_GROUP_ENDPOINT = "/testing"

	// NODES_ENDPOINT is used internally (by the frontend) to get the current kubernetes nodes from the backend.
	NODES_ENDPOINT = "/nodes"

	// ADJUST_VGPUS_ENDPOINT is used internally (by the frontend) to adjust the vGPUs offered by a particular kubernetes nodes.
	ADJUST_VGPUS_ENDPOINT = "/vgpus"

	// SYSTEM_CONFIG_ENDPOINT is used internally (by the frontend) to get the system config from the backend.
	SYSTEM_CONFIG_ENDPOINT = "/config"

	// MIGRATION_ENDPOINT is used internally (by the frontend) to trigger kernel replica migrations.
	MIGRATION_ENDPOINT = "/migrate"

	// LOGS_ENDPOINT is used to stream logs to the frontend from Kubernetes.
	LOGS_ENDPOINT = "/logs"

	// WORKLOAD_ENDPOINT is used internally (by the frontend) to trigger the start of a new workload or retrieve the list of workloads.
	WORKLOAD_ENDPOINT = "/workload"

	// GENERAL_WEBSOCKET_ENDPOINT is used for WebSocket-based communication between the frontend and backend that is unrelated to workloads or logs.
	GENERAL_WEBSOCKET_ENDPOINT = "/ws"

	// KERNEL_SPEC_ENDPOINT is used internally (by the frontend) to get the current set of Jupyter kernel specs from the backend.
	KERNEL_SPEC_ENDPOINT = "/kernelspecs"

	// GET_KERNELS_ENDPOINT is used internally (by the frontend) to get the current set of Jupyter kernels from the backend.
	GET_KERNELS_ENDPOINT = "/get-kernels"

	// WORKLOAD_PRESET_ENDPOINT is used internally (by the frontend) to get the list of available workload presets from the backend.
	WORKLOAD_PRESET_ENDPOINT = "/workload-presets"

	// PANIC_ENDPOINT is used to cause the Cluster Gateway to panic. used for debugging/testing.
	PANIC_ENDPOINT = "/panic"

	// STOP_TRAINING_ENDPOINT is used to tell a kernel to stop training.
	STOP_TRAINING_ENDPOINT = "/stop-training"

	// YIELD_NEXT_REQUEST_ENDPOINT is used to specify that the next execution request served by a particular kernel should be yielded.
	YIELD_NEXT_REQUEST_ENDPOINT = "/yield-next-execute-request"

	JUPYTER_GROUP_ENDPOINT        = "/jupyter"
	JUPYTER_START_KERNEL_ENDPOINT = "/start"
	JUPYTER_STOP_KERNEL_ENDPOINT  = "/stop"

	// SPOOF_ERROR_ENDPOINT causes the server to broadcast a fake error via websockets for debugging/testing purposes.
	SPOOF_ERROR_ENDPOINT = "/spoof-error"

	// SPOOF_NOTIFICATIONS_ENDPOINT is used for testing notifications sent from the Cluster to the Dashboard
	SPOOF_NOTIFICATIONS_ENDPOINT = "/spoof-notifications"

	PING_KERNEL_ENDPOINT = "/ping-kernel"

	// VariablesEndpoint is queried by Grafana to query for values used to create Grafana variables that are then used to
	// dynamically create a Grafana Dashboard.
	VariablesEndpoint = "/variables"
)

var (
	KernelStatuses      = []string{"unknown", "starting", "idle", "busy", "terminating", "restarting", "autorestarting", "dead"}
	ErrEmptyGatewayAddr = errors.New("cluster gateway IP address cannot be the empty string")
)

type KernelRefreshCallback func([]*gateway.DistributedJupyterKernel)

type Server interface {
	Serve() error // Run the server. This is a blocking call.
}
