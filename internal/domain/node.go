package domain

const (
	// KubernetesNodeType is a NodeType that refers to a node within a Kubernetes cluster.
	KubernetesNodeType NodeType = "KubernetesNode"

	// DockerSwarmNodeType is a NodeType that refers to a node within a Kubernetes cluster.
	DockerSwarmNodeType NodeType = "DockerSwarmNode"

	// ContainerTypePod instances are Container instances deployed atop a ClusterNode with NodeType equal to KubernetesNodeType .
	ContainerTypePod ContainerType = "Pod"

	// ContainerTypeDockerContainer instances are Container instances deployed atop a ClusterNode with NodeType equal to DockerSwarmNodeType .
	ContainerTypeDockerContainer ContainerType = "DockerContainer"
)

// NodeType defines the "type" of a node. Nodes may either belong to a Kubernetes cluster or a Docker swarm.
type NodeType string

// ContainerType defines the "type" of a Container that is running on a ClusterNode.
// Container instances deployed on a Kubernetes cluster are "Pods", whereas Container Instances running on
// a Docker swarm node are "Docker containers".
type ContainerType string

// ContainerList defines a slice of Container instances (specifically, instances of some concrete implementation of
// the Container interface).
type ContainerList []Container

// Container defines the generic, platform-agnostic interface for Container instances
// (which are either Docker containers or Kubernetes pods).
type Container interface {
	// GetContainerType returns the ContainerType of the ClusterNode.
	GetContainerType() ContainerType

	// GetValidNodeType returns the NodeType of the ClusterNode instances onto which
	// the Container is permitted to be scheduled.
	GetValidNodeType() NodeType

	// GetName returns the name of the Container.
	GetName() string

	// GetState returns the lifecycle state of the Container.
	GetState() string

	// GetAge returns the age of the Container.
	// The value is created by converting a time.Duration to a string.
	GetAge() string

	// GetIp returns the IP or network address of the Container.
	GetIp() string

	// IsValid returns a flag that used to determine if the Container struct was sent/received correctly over the network.
	IsValid() bool

	// String returns a string representation of the Container suitable for logging.
	String() string
}

// ClusterNode defines an abstract node within the distributed cluster.
// A ClusterNode may be a Kubernetes node or a Docker Swarm node.
type ClusterNode interface {
	// GetNodeType returns the NodeType of the ClusterNode.
	// The NodeType of the ClusterNode must be either KubernetesNodeType or DockerSwarmNodeType.
	GetNodeType() NodeType

	// GetValidContainerType returns the ContainerType of Container instances that may be
	// scheduled onto the ClusterNode.
	GetValidContainerType() ContainerType

	// GetContainers returns a slice of Container instances that represents the Container instances
	// currently scheduled on the ClusterNode.
	GetContainers() ContainerList

	// GetNodeId returns the unique ID of the node.
	GetNodeId() string

	// GetAge returns a string created from a time.Duration.
	// The string represents/indicates the length of time that the ClusterNode has existed.
	GetAge() string

	// GetIp returns the network address of the ClusterNode.
	GetIp() string

	// GetAllocatedResources returns a map from resource name to a float64 representing the quantity of that resource
	// that is presently allocated to Container instances on the ClusterNode.
	GetAllocatedResources() map[string]float64

	// GetResourceCapacities returns is a map from resource name to a float64 representing the quantity of that resource
	// that is allocatable on the ClusterNode.
	//
	// Quantities stored in the CapacityResources do not change based on active resource allocations.
	// They simply refer to the total amount of resources with which the ClusterNode is configured.
	GetResourceCapacities() map[string]float64

	// IsEnabled returns a bool indicating whether the ClusterNode is currently enabled.
	// When a ClusterNode is enabled, it is permitted to host Container instances.
	// When a ClusterNode is disabled, it is not permitted to host Container instances.
	IsEnabled() bool

	// String returns a string representation of the ClusterNode suitable for logging.
	String() string
}
