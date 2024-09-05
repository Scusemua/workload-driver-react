package domain

import (
	"encoding/json"
)

// KubernetesNode is a struct defining the relevant information of a Kubernetes nodes.
// We parse the data returned by the Kubernetes API to construct KubernetesNode structs.
//
// The KubernetesNode struct is a concrete implementation of the ClusterNode interface.
type KubernetesNode struct {
	NodeId             string                   `json:"NodeId"`
	Pods               ContainerList            `json:"PodsOrContainers"`
	Age                string                   `json:"Age"` // Convert the time.Duration to a string
	IP                 string                   `json:"IP"`
	AllocatedResources map[ResourceName]float64 `json:"AllocatedResources"`
	CapacityResources  map[ResourceName]float64 `json:"CapacityResources"`
	Enabled            bool                     `json:"Enabled"`
}

func (kn *KubernetesNode) GetNodeType() NodeType {
	return KubernetesNodeType
}

func (kn *KubernetesNode) GetValidContainerType() ContainerType {
	return ContainerTypePod
}

func (kn *KubernetesNode) GetContainers() ContainerList {
	return kn.Pods
}

func (kn *KubernetesNode) GetNodeId() string {
	return kn.NodeId
}

func (kn *KubernetesNode) GetAge() string {
	return kn.Age
}

func (kn *KubernetesNode) GetIp() string {
	return kn.IP
}

func (kn *KubernetesNode) GetAllocatedResources() map[ResourceName]float64 {
	return kn.AllocatedResources
}

func (kn *KubernetesNode) GetResourceCapacities() map[ResourceName]float64 {
	return kn.CapacityResources
}

func (kn *KubernetesNode) IsEnabled() bool {
	return kn.Enabled
}

func (kn *KubernetesNode) String() string {
	out, err := json.Marshal(kn)
	if err != nil {
		panic(err)
	}

	return string(out)
}

// KubernetesPod is a struct defining the relevant information of a Kubernetes Pod.
// We parse the data returned by the Kubernetes API to construct KubernetesPod structs.
//
// The KubernetesPod struct is a concrete implementation of the Container interface.
type KubernetesPod struct {
	// PodName refers to the name of the KubernetesPod.
	PodName string `json:"Name"`

	// PodPhase returns to the [Kubernetes phase] of the KubernetesPod.
	//
	// [Kubernetes phase]: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-phase
	PodPhase string `json:"Phase"`

	// PodAge refers to the age of the KubernetesPod.
	// The value is created by converting a time.Duration to a string.
	PodAge string `json:"Age"`

	// PodIP refers to the IP or network address of the KubernetesPod.
	PodIP string `json:"IP"`

	// Valid is a flag used to determine if the KubernetesPod struct was sent/received correctly over the network.
	Valid bool `json:"Valid"`
}

func (kp *KubernetesPod) GetContainerType() ContainerType {
	return ContainerTypePod
}

func (kp *KubernetesPod) GetValidNodeType() NodeType {
	return KubernetesNodeType
}

func (kp *KubernetesPod) GetName() string {
	return kp.PodName
}

func (kp *KubernetesPod) GetState() string {
	return kp.PodPhase
}

func (kp *KubernetesPod) GetAge() string {
	return kp.PodAge
}

func (kp *KubernetesPod) GetIp() string {
	return kp.PodIP
}

func (kp *KubernetesPod) IsValid() bool {
	return kp.Valid
}

func (kp *KubernetesPod) String() string {
	out, err := json.Marshal(kp)
	if err != nil {
		panic(err)
	}

	return string(out)
}
