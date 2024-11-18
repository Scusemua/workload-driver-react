package domain

import (
	"encoding/json"
	"fmt"
	"github.com/icza/gox/timex"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"time"
)

// DockerContainer represents a Container within the distributed cluster that will be running on a ClusterNode.
type DockerContainer struct {
	// ContainerName refers to the name of the DockerContainer.
	ContainerName string `json:"Name"`

	// ContainerPhase returns to the lifestyle phase of the DockerContainer.
	ContainerPhase string `json:"Phase"`

	// ContainerAge refers to the age of the DockerContainer.
	// The value is created by converting a time.Duration to a string.
	ContainerAge string `json:"Age"`

	// ContainerIP refers to the IP or network address of the DockerContainer.
	ContainerIP string `json:"IP"`

	// Valid is a flag used to determine if the DockerContainer struct was sent/received correctly over the network.
	Valid bool `json:"Valid"`

	// Type encodes the ContainerType, which for DockerContainer structs will always be ContainerTypeDockerContainer.
	Type ContainerType `json:"Type"`
}

// NewDockerContainer constructs and returns a pointer to a DockerContainer struct.
func NewDockerContainer() *DockerContainer {
	container := &DockerContainer{
		Type: ContainerTypeDockerContainer,
	}

	return container
}

func (dc *DockerContainer) GetContainerType() ContainerType {
	return ContainerTypeDockerContainer
}

func (dc *DockerContainer) GetValidNodeType() NodeType {
	return VirtualDockerNodeType
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

// VirtualDockerNode represents a node within a Docker Swarm cluster.
type VirtualDockerNode struct {
	// Type is the NodeType of the VirtualDockerNode, which will necessarily be VirtualDockerNodeType.
	Type NodeType `json:"NodeType"`

	// NodeID is the unique ID of the node.
	NodeId string `json:"NodeId"`

	// NodeName is the name of the node, which is (typically) distinct from its ID.
	NodeName string `json:"NodeName"`

	// Containers is a slice of DockerContainer instances that are currently scheduled on this VirtualDockerNode.
	Containers ContainerList `json:"PodsOrContainers"`

	// Age refers to the length of time that the VirtualDockerNode has existed.
	Age string `json:"Age"`

	// CreatedAt is the unix milliseconds at which the corresponding VirtualDockerNode was created.
	// (Not when the struct was created, but the actual cluster node represented by the VirtualDockerNode struct.)
	CreatedAt int64 `json:"CreatedAt"`

	age time.Duration

	// IP is the network address of the VirtualDockerNode.
	IP string `json:"IP"`

	// AllocatedResources is a map from resource name to a float64 representing the quantity of that resource
	// that is presently allocated to Container instances on the VirtualDockerNode.
	AllocatedResources map[ResourceName]float64 `json:"AllocatedResources"`

	// IdleResources is a map from resource name to a float64 representing the quantity of that resource
	// that is not actively committed to any particular Container instance on the VirtualDockerNode.
	IdleResources map[ResourceName]float64 `json:"IdleResources"`

	// PendingResources is a map from resource name to a float64 representing the quantity of that resource
	// that is subscribed by, but not allocated to, a Container instance on the VirtualDockerNode.
	PendingResources map[ResourceName]float64 `json:"PendingResources"`

	// CapacityResources is a map from resource name to a float64 representing the quantity of that resource
	// that is allocatable on the VirtualDockerNode.
	//
	// Quantities stored in the CapacityResources do not change based on active resource allocations.
	// They simply refer to the total amount of resources with which the VirtualDockerNode is configured.
	CapacityResources map[ResourceName]float64 `json:"CapacityResources"`

	// Enabled is a flag indicating whether the VirtualDockerNode is currently enabled/allowed to host Container instances.
	Enabled bool `json:"Enabled"`
}

// DockerContainerFromProtoDockerContainer constructs a new DockerContainer struct from the data
// encoded in a proto.DockerContainer struct and returns a pointer to the new DockerContainer struct.
func DockerContainerFromProtoDockerContainer(protoContainer *proto.DockerContainer) *DockerContainer {
	return &DockerContainer{
		ContainerName:  protoContainer.GetContainerName(),
		ContainerPhase: protoContainer.GetContainerStatus(),
		ContainerAge:   protoContainer.GetContainerAge(),
		ContainerIP:    protoContainer.GetContainerIp(),
		Valid:          protoContainer.GetValid(),
		Type:           ContainerTypeDockerContainer,
	}
}

// VirtualDockerNodeFromProtoVirtualDockerNode constructs a new VirtualDockerNode struct from the data
// encoded in a proto.VirtualDockerNode struct and returns a pointer to the new VirtualDockerNode struct.
func VirtualDockerNodeFromProtoVirtualDockerNode(protoNode *proto.VirtualDockerNode) *VirtualDockerNode {
	containers := make(ContainerList, 0, len(protoNode.Containers))
	for _, protoContainer := range protoNode.Containers {
		containers = append(containers, DockerContainerFromProtoDockerContainer(protoContainer))
	}

	allocatedResources := make(map[ResourceName]float64)
	capacityResources := make(map[ResourceName]float64)
	idleResources := make(map[ResourceName]float64)
	pendingResources := make(map[ResourceName]float64)

	allocatedResources[CpuResource] = float64(protoNode.AllocatedCpu)
	pendingResources[CpuResource] = float64(protoNode.PendingCpu)
	idleResources[CpuResource] = float64(protoNode.SpecCpu) - float64(protoNode.AllocatedCpu)
	capacityResources[CpuResource] = float64(protoNode.SpecCpu)

	allocatedResources[MemoryResource] = float64(protoNode.AllocatedMemory)
	pendingResources[MemoryResource] = float64(protoNode.PendingMemory)
	idleResources[MemoryResource] = float64(protoNode.SpecMemory) - float64(protoNode.AllocatedMemory)
	capacityResources[MemoryResource] = float64(protoNode.SpecMemory)

	allocatedResources[GpuResource] = float64(protoNode.AllocatedGpu)
	pendingResources[GpuResource] = float64(protoNode.PendingGpu)
	idleResources[GpuResource] = float64(protoNode.SpecGpu) - float64(protoNode.AllocatedGpu)
	capacityResources[GpuResource] = float64(protoNode.SpecGpu)

	allocatedResources[VirtualGpuResource] = float64(protoNode.AllocatedGpu)
	pendingResources[VirtualGpuResource] = float64(protoNode.PendingGpu)
	idleResources[VirtualGpuResource] = float64(protoNode.SpecGpu) - float64(protoNode.AllocatedGpu)
	capacityResources[VirtualGpuResource] = float64(protoNode.SpecGpu)

	allocatedResources[VRAMResource] = float64(protoNode.AllocatedVRAM)
	pendingResources[VRAMResource] = float64(protoNode.PendingVRAM)
	idleResources[VRAMResource] = float64(protoNode.SpecVRAM) - float64(protoNode.AllocatedVRAM)
	capacityResources[VRAMResource] = float64(protoNode.SpecVRAM)

	return &VirtualDockerNode{
		NodeId:             protoNode.NodeId,
		NodeName:           protoNode.NodeName,
		Type:               VirtualDockerNodeType,
		IP:                 protoNode.Address,
		CreatedAt:          protoNode.CreatedAt.Seconds,
		Age:                fmt.Sprintf("%v", timex.Round(time.Now().Sub(protoNode.CreatedAt.AsTime()), 3)),
		age:                time.Now().Sub(protoNode.CreatedAt.AsTime()),
		Enabled:            protoNode.Enabled,
		Containers:         containers,
		AllocatedResources: allocatedResources,
		CapacityResources:  capacityResources,
		PendingResources:   pendingResources,
		IdleResources:      idleResources,
	}
}

// NewVirtualDockerNode constructs and returns a pointer to a VirtualDockerNode struct.
func NewVirtualDockerNode() *VirtualDockerNode {
	node := &VirtualDockerNode{}

	return node
}

func (d *VirtualDockerNode) GetNodeType() NodeType {
	return VirtualDockerNodeType
}

func (d *VirtualDockerNode) GetValidContainerType() ContainerType {
	return ContainerTypeDockerContainer
}

func (d *VirtualDockerNode) GetContainers() ContainerList {
	return d.Containers
}

func (d *VirtualDockerNode) GetNodeId() string {
	return d.NodeId
}

func (d *VirtualDockerNode) GetAge() string {
	return d.Age
}

func (d *VirtualDockerNode) GetAgeAsDuration() time.Duration {
	return d.age
}

func (d *VirtualDockerNode) GetIp() string {
	return d.IP
}

func (d *VirtualDockerNode) GetAllocatedResources() map[ResourceName]float64 {
	return d.AllocatedResources
}

func (d *VirtualDockerNode) GetResourceCapacities() map[ResourceName]float64 {
	return d.CapacityResources
}

func (d *VirtualDockerNode) GetIdleResources() map[ResourceName]float64 {
	return d.IdleResources
}

func (d *VirtualDockerNode) IsEnabled() bool {
	return d.Enabled
}

func (d *VirtualDockerNode) String() string {
	out, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	return string(out)
}
