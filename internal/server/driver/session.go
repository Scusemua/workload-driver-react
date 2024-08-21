package driver

import (
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
)

// TODO: Merge this with the WorkloadSession struct.
type Session struct {
	sessionId       string
	meta            *generator.Session
	resourceRequest *ResourceRequest
	createdAtTime   time.Time
}

func NewSession(id string, meta *generator.Session, resourceRequest *ResourceRequest, createdAtTime time.Time) *Session {
	return &Session{
		sessionId:       id,
		meta:            meta,
		resourceRequest: resourceRequest,
		createdAtTime:   createdAtTime,
	}
}

func (s *Session) Id() string {
	return s.sessionId
}

func (s *Session) Meta() *generator.Session {
	return s.meta
}

func (s *Session) ResourceRequest() *ResourceRequest {
	return s.resourceRequest
}

func (s *Session) CreatedAtTime() time.Time {
	return s.createdAtTime
}
