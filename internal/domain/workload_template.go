package domain

import "fmt"

type WorkloadTemplate struct {
	Name     string                     `json:"name"`
	Sessions []*WorkloadTemplateSession `json:"sessions"`
}

func (t *WorkloadTemplate) String() string {
	return fmt.Sprintf("Template[%s]", t.Name)
}

func (t *WorkloadTemplate) GetSessions() []*WorkloadTemplateSession {
	return t.Sessions
}
