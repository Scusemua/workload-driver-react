package domain

import "encoding/json"

type KubernetesNode struct {
	NodeId             string             `json:"NodeId"`
	Pods               []*KubernetesPod   `json:"Pods"`
	Age                string             `json:"Age"` // Convert the time.Duration to a string
	IP                 string             `json:"IP"`
	AllocatedResources map[string]float64 `json:"AllocatedResources"`
	CapacityResources  map[string]float64 `json:"CapacityResources"`
	Enabled            bool               `json:"Enabled"`
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
