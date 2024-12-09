package workload

import (
	"errors"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/pkg/jupyter"
	"go.uber.org/zap"
	"sync/atomic"
)

var (
	ErrClientNotRunning = errors.New("cannot stop client as it is not running")
)

// Client encapsulates a Session and runs as a dedicated goroutine, processing events for that Session.
type Client struct {
	SessionId        string
	WorkloadId       string
	SessionMeta      domain.SessionMetadata
	WorkloadSession  *domain.WorkloadTemplateSession
	KernelConnection jupyter.KernelConnection
	EventQueue       chan *domain.Event

	logger *zap.Logger

	running atomic.Int32
}

// NewClient creates a new Client struct and returns a pointer to it.
func NewClient(sessionId string, workloadId string) *Client {
	client := &Client{
		SessionId:  sessionId,
		WorkloadId: workloadId,
		EventQueue: make(chan *domain.Event, 1024),
	}

	return client
}

func (c *Client) Stop() error {
	if !c.running.CompareAndSwap(1, 0) {
		return ErrClientNotRunning
	}

	c.logger.Debug("Client has been told to stop running.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.WorkloadId))

	return nil
}

func (c *Client) Run() {
	if !c.running.CompareAndSwap(0, 1) {
		c.logger.Warn("Client is already running.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId))
		return
	}

	for c.running.Load() == 1 {
		// Do something.
	}
}
