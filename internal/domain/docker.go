package domain

import (
	"encoding/json"
	"time"
)

// DockerContainer represents a Container within the distributed cluster that will be running on a ClusterNode.
type DockerContainer struct {
	// ContainerName refers to the name of the DockerContainer.
	ContainerName string `json:"ContainerName"`

	// ContainerPhase returns to the lifestyle phase of the DockerContainer.
	ContainerPhase string `json:"ContainerPhase"`

	// ContainerAge refers to the age of the DockerContainer.
	// The value is created by converting a time.Duration to a string.
	ContainerAge string `json:"ContainerAge"`

	// ContainerIP refers to the IP or network address of the DockerContainer.
	ContainerIP string `json:"ContainerIP"`

	// Valid is a flag used to determine if the DockerContainer struct was sent/received correctly over the network.
	Valid bool `json:"Valid"`
}

// NewDockerContainer constructs and returns a pointer to a DockerContainer struct.
func NewDockerContainer() *DockerContainer {
	container := &DockerContainer{}

	return container
}

func (dc *DockerContainer) GetContainerType() ContainerType {
	return ContainerTypeDockerContainer
}

func (dc *DockerContainer) GetValidNodeType() NodeType {
	return DockerSwarmNodeType
}

func (dc *DockerContainer) GetName() string {
	return dc.ContainerName
}

func (dc *DockerContainer) GetState() string {
	return dc.ContainerPhase
}

func (dc *DockerContainer) GetAge() string {
	return dc.ContainerAge
}

func (dc *DockerContainer) GetIp() string {
	return dc.ContainerIP
}

func (dc *DockerContainer) IsValid() bool {
	return dc.Valid
}

func (dc *DockerContainer) String() string {
	out, err := json.Marshal(dc)
	if err != nil {
		panic(err)
	}

	return string(out)
}

// DockerSwarmNode represents a node within a Docker Swarm cluster.
type DockerSwarmNode struct {
	// Type is the NodeType of the DockerSwarmNode, which will necessarily be DockerSwarmNodeType.
	Type NodeType `json:"type"`

	// NodeID is the unique ID of the node.
	NodeId string `json:"nodeId"`

	// Containers is a slice of DockerContainer instances that are currently scheduled on this DockerSwarmNode.
	Containers ContainerList `json:"Containers"`

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

func (d DockerSwarmNode) GetNodeType() NodeType {
	return DockerSwarmNodeType
}

func (d DockerSwarmNode) GetValidContainerType() ContainerType {
	return ContainerTypeDockerContainer
}

func (d DockerSwarmNode) GetContainers() ContainerList {
	return d.Containers
}

func (d DockerSwarmNode) GetNodeId() string {
	return d.NodeId
}

func (d DockerSwarmNode) GetAge() string {
	return d.Age.String()
}

func (d DockerSwarmNode) GetIp() string {
	return d.IP
}

func (d DockerSwarmNode) GetAllocatedResources() map[string]float64 {
	return d.AllocatedResources
}

func (d DockerSwarmNode) GetResourceCapacities() map[string]float64 {
	return d.CapacityResources
}

func (d DockerSwarmNode) IsEnabled() bool {
	return d.Enabled
}

func (d DockerSwarmNode) String() string {
	out, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	return string(out)
}
