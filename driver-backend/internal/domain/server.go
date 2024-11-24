package domain

import (
	"errors"

	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
)

const (
	// BaseApiGroupEndpoint is the Base of the API endpoint.
	BaseApiGroupEndpoint = "api"

	// NodesEndpoint is used internally (by the frontend) to get the current kubernetes nodes from the backend.
	NodesEndpoint = "nodes"

	// AdjustVgpusEndpoint is used internally (by the frontend) to adjust the vGPUs offered by a
	// particular kubernetes nodes.
	AdjustVgpusEndpoint = "vgpus"

	// SystemConfigEndpoint is used internally (by the frontend) to get the system config from the backend.
	SystemConfigEndpoint = "config"

	// MigrationEndpoint is used internally (by the frontend) to trigger kernel replica migrations.
	MigrationEndpoint = "migrate"

	// LogsEndpoint is used to stream logs to the frontend from Kubernetes.
	LogsEndpoint = "logs"

	// WebsocketGroupEndpoint is used to define the group for WebSocket requests.
	WebsocketGroupEndpoint = "websocket"

	// WorkloadEndpoint is used internally (by the frontend) to trigger the start of a new workload or
	// retrieve the list of workloads.
	WorkloadEndpoint = "workload"

	// GeneralWebsocketEndpoint is used for WebSocket-based communication between the frontend and backend
	// that is unrelated to workloads or logs.
	GeneralWebsocketEndpoint = "general"

	// KernelSpecEndpoint is used internally (by the frontend) to get the current set of Jupyter kernel specs
	// from the backend.
	KernelSpecEndpoint = "kernelspecs"

	// GetKernelsEndpoint is used internally (by the frontend) to get the current set of Jupyter kernels
	// from the backend.
	GetKernelsEndpoint = "get-kernels"

	// PrometheusEndpoint is the default path on which Prometheus issues GET requests to scrape metrics.
	PrometheusEndpoint = "prometheus"

	// MetricsEndpoint is used by the frontend to post/share Prometheus metrics.
	MetricsEndpoint = "metrics"

	// WorkloadPresetEndpoint is used internally (by the frontend) to get the list of available
	// workload presets from the backend.
	WorkloadPresetEndpoint = "workload-presets"

	// WorkloadTemplatesEndpoint is used internally (by the frontend) to get the list of available
	// workload templates from the backend.
	WorkloadTemplatesEndpoint = "workload-templates"

	// PanicEndpoint is used to cause the Cluster Gateway to panic. used for debugging/testing.
	PanicEndpoint = "panic"

	// ClusterAgeEndpoint is used to retrieve the UnixMillisecond timestamp at which the Cluster was created.
	ClusterAgeEndpoint = "cluster-age"

	// SchedulingPolicyEndpoint is targeted by HTTP GET requests to get the scheduling policy of the cluster.
	SchedulingPolicyEndpoint = "scheduling-policy"

	// DeploymentModeEndpoint is used to retrieve the configured deployment mode of the cluster.
	DeploymentModeEndpoint = "deployment-mode"

	// RefreshToken is used to refresh a JWT auth token.
	RefreshToken = "refresh_token"

	// AuthenticateRequest is used to authenticate and get access to the Dashboard.
	AuthenticateRequest = "authenticate"

	// StopTrainingEndpoint is used to tell a kernel to stop training.
	StopTrainingEndpoint = "stop-training"

	// YieldNextRequestEndpoint is used to specify that the next execution request served by a
	// particular kernel should be yielded.
	YieldNextRequestEndpoint = "yield-next-execute-request"

	// QueryMessageEndpoint is used by the frontend to query the status of particular ZMQ messages.
	QueryMessageEndpoint = "query-message"

	// InstructLocalDaemonReconnect is used by the frontend to instruct a Local Daemon to
	// reconnect to the Cluster Gateway.
	InstructLocalDaemonReconnect = "instruct-ld-reconnect"

	JupyterGroupEndpoint = "jupyter"

	// JupyterAddressEndpoint is used to tell the frontend what the address of Jupyter is.
	JupyterAddressEndpoint = "jupyter-address"

	// SpoofErrorEndpoint causes the server to broadcast a fake error via websockets for debugging/testing purposes.
	SpoofErrorEndpoint = "spoof-error"

	// SpoofNotificationsEndpoint is used for testing notifications sent from the Cluster to the Dashboard
	SpoofNotificationsEndpoint = "spoof-notifications"

	PingKernelEndpoint = "ping-kernel"

	// VariablesEndpoint is queried by Grafana to query for values used to create Grafana variables that are then
	// used to dynamically create a Grafana Dashboard.
	VariablesEndpoint = "variables"

	// NoOpEndpoint is essentially just used to test the validity of the current authentication token.
	NoOpEndpoint = "no-op"
)

var (
	KernelStatuses      = []string{"unknown", "starting", "idle", "busy", "terminating", "restarting", "autorestarting", "dead"}
	ErrEmptyGatewayAddr = errors.New("cluster gateway IP address cannot be the empty string")
)

type KernelRefreshCallback func([]*gateway.DistributedJupyterKernel)

type Server interface {
	Serve() error // Run the server. This is a blocking call.
}
