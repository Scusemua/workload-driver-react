package domain

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
)

const (
	// Base of the API endpoint.
	BASE_API_GROUP_ENDPOINT = "/api"

	// Used internally (by the frontend) to get the current kubernetes nodes from the backend.
	KUBERNETES_NODES_ENDPOINT = "/nodes"

	// Used internally (by the frontend) to get the system config from the backend.
	SYSTEM_CONFIG_ENDPOINT = "/config"

	// Used internally (by the frontend) to trigger kernel replica migrations.
	MIGRATION_ENDPOINT = "/migrate"

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

// Used to pass errors back to another window.
type ErrorHandler interface {
	HandleError(error, string)
}

type WorkloadDriver interface {
	// Return true if we're connected to the Cluster Gateway.
	ConnectedToGateway() bool

	KernelSpecProvider() KernelSpecProvider // Return the entity responsible for providing the up-to-date list of Jupyter kernel specs.
	KernelProvider() KernelProvider         // Return the entity responsible for providing the up-to-date list of Jupyter kernels.
	NodeProvider() NodeProvider             // Return the entity responsible for providing the up-to-date list of Kubernetes nodes.

	// Tell the Cluster Gateway to migrate a particular replica.
	MigrateKernelReplica(*gateway.MigrationRequest) error
	DialGatewayGRPC(string) error // Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
}

type WorkloadDriverOptions struct {
	HttpPort int `name:"http_port" description:"Port that the server will listen on." json:"http_port"`
}

type ResourceProvider[resource any] interface {
	Count() int32          // Number of currently-active resources.
	Resources() []resource // List of currently-active resources.
	RefreshResources()     // Manually/explicitly refresh the set of active resources from the Cluster Gateway.
	Start(string) error    // Start querying for resources periodically.

	RefreshOccurred()                                   // Called automatically when a refresh occurred; informs the subscribers.
	QueryResources()                                    // Call in its own goroutine; polls for resources.
	SubscribeToRefreshes(string, func([]resource) bool) // Subscribe to Kernel refreshes.
	UnsubscribeFromRefreshes(string)                    // Unsubscribe from Kernel refreshes.
	DialGatewayGRPC(string) error                       // Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
}

type KernelProvider interface {
	ResourceProvider[*gateway.DistributedJupyterKernel]
}

type KernelSpecProvider interface {
	ResourceProvider[*KernelSpec]
}

type NodeProvider interface {
	ResourceProvider[*KubernetesNode]
}

type KubernetesNode struct {
	NodeId          string           `json:"NodeId"`
	Pods            []*KubernetesPod `json:"Pods"`
	Age             string           `json:"Age"` // Convert the time.Duration to a string
	IP              string           `json:"IP"`
	CapacityCPU     float64          `json:"CapacityCPU"`
	CapacityMemory  float64          `json:"CapacityMemory"`
	CapacityGPUs    float64          `json:"CapacityGPUs"`
	CapacityVGPUs   float64          `json:"CapacityVGPUs"`
	AllocatedCPU    float64          `json:"AllocatedCPU"`
	AllocatedMemory float64          `json:"AllocatedMemory"`
	AllocatedGPUs   float64          `json:"AllocatedGPUs"`
	AllocatedVGPUs  float64          `json:"AllocatedVGPUs"`
}

func (kn *KubernetesNode) String() string {
	out, err := json.Marshal(kn)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type KubernetesPod struct {
	PodName  string `json:"PodName"`
	PodPhase string `json:"PodPhase"`
	PodAge   string `json:"PodAge"` // Convert the time.Duration to a string
	PodIP    string `json:"PodIP"`

	Valid bool `json:"Valid"` // Used to determine if the struct was sent/received correctly over the network.
}

func (kp *KubernetesPod) String() string {
	out, err := json.Marshal(kp)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type ErrorMessage struct {
	Description  string `json:"Description"`  // Provides additional context for what occurred; written by us.
	ErrorMessage string `json:"ErrorMessage"` // The value returned by err.Error() for whatever error occurred.
	Valid        bool   `json:"Valid"`        // Used to determine if the struct was sent/received correctly over the network.
}

func (m *ErrorMessage) String() string {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type BackendHttpGetHandler interface {
	// Write an error back to the client.
	WriteError(*gin.Context, string)

	// Handle a message/request from the front-end.
	HandleRequest(*gin.Context)

	// Return the request handler responsible for handling a majority of requests.
	PrimaryHttpHandler() BackendHttpGetHandler
}

type JupyterApiHttpHandler interface {
	// Handle an HTTP GET request to get the jupyter kernel specs.
	HandleGetKernelSpecRequest(*gin.Context)

	// Handle an HTTP POST request to create a new jupyter kernel.
	HandleCreateKernelRequest(*gin.Context)

	// Write an error back to the client.
	WriteError(*gin.Context, string)
}

type BackendHttpGRPCHandler interface {
	BackendHttpGetHandler

	// Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success.
	DialGatewayGRPC(string) error
}

type KernelSpec struct {
	Name              string             `json:"name"`
	DisplayName       string             `json:"displayName"`
	Language          string             `json:"language"`
	InterruptMode     string             `json:"interruptMode"`
	KernelProvisioner *KernelProvisioner `json:"kernelProvisioner"`
	ArgV              []string           `json:"argV"`
}

func (ks *KernelSpec) String() string {
	out, err := json.Marshal(ks)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type KernelProvisioner struct {
	Name    string `json:"name"`
	Gateway string `json:"display_name"`
	Valid   bool   `json:"valid"`
}

func (kp *KernelProvisioner) String() string {
	out, err := json.Marshal(kp)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type WorkloadPreset struct {
	Name        string   `json:"name"`        // Human-readable name for this particular workload preset.
	Description string   `json:"description"` // Human-readable description of the workload.
	Key         string   `json:"key"`         // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	Months      []string `json:"months"`      // The months of data used by the workload.
}