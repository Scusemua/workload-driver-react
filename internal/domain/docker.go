package domain

import "time"

// DockerContainer represents a Container within the distributed cluster that will be running on a ClusterNode.
type DockerContainer struct {
}

// NewDockerContainer constructs and returns a pointer to a DockerContainer struct.
func NewDockerContainer() *DockerContainer {
  container := &DockerContainer{}

  return container
}

// DockerSwarmNode represents a node within a Docker Swarm cluster.
type DockerSwarmNode struct {
  // Type is the NodeType of the DockerSwarmNode, which will necessarily be DockerSwarmNodeType.
  Type NodeType `json:"type"`

  // NodeID is the unique ID of the node.
  NodeId string `json:"nodeId"`

  // Containers is a slice of DockerContainer instances that are currently scheduled on this DockerSwarmNode.
  Containers []*DockerContainer `json:"Containers"`

  // Age refers to the length of time that the DockerSwarmNode has existed.
  Age time.Duration `json:"Age"`

  // IP is the network address of the DockerSwarmNode.
  IP string `json:"IP"`

  // AllocatedResources is a map from resource name to a float64 representing the quantity of that resource
  // that is presently allocated to Container instances on the DockerSwarmNode.
  AllocatedResources map[string]float64 `json:"AllocatedResources"`

  // CapacityResources is a map from resource name to a float64 representing the quantity of that resource
  // that is allocatable on the DockerSwarmNode.
  //
  // Quantities stored in the CapacityResources do not change based on active resource allocations.
  // They simply refer to the total amount of resources with which the DockerSwarmNode is configured.
  CapacityResources map[string]float64 `json:"CapacityResources"`

  // Enabled is a flag indicating whether the DockerSwarmNode is currently enabled/allowed to host Container instances.
  Enabled bool `json:"Enabled"`
}

// NewDockerSwarmNode constructs and returns a pointer to a DockerSwarmNode struct.
func NewDockerSwarmNode() *DockerSwarmNode {
  node := &DockerSwarmNode{}

  return node
}
