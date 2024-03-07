package domain

import "encoding/json"

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
