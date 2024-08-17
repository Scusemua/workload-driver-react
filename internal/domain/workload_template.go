package domain

import "fmt"

type WorkloadTemplate struct {
	Name     string             `json:"name"`
	Sessions []*WorkloadSession `json:"sessions"`
}

func (t *WorkloadTemplate) String() string {
	return fmt.Sprintf("Template[%s]", t.Name)
}
