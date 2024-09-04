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

type Container interface {
	// GetContainerType returns the ContainerType of the ClusterNode.
	GetContainerType() ContainerType

	// GetValidNodeType returns the NodeType of the ClusterNode instances onto which
	// the Container is permitted to be scheduled.
	GetValidNodeType() NodeType
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
}
