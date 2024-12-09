package workload

import (
	"errors"
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/pkg/jupyter"
	"go.uber.org/zap"
	"sync/atomic"
)

var (
	ErrClientNotRunning  = errors.New("cannot stop client as it is not running")
	ErrInvalidFirstEvent = errors.New("client received invalid first event")
)

type ClientBuilder struct {
	id              string
	workloadId      string
	sessionMeta     string
	workloadSession *domain.WorkloadTemplateSession
}

// Client encapsulates a Session and runs as a dedicated goroutine, processing events for that Session.
type Client struct {
	SessionId           string
	WorkloadId          string
	SessionMeta         domain.SessionMetadata
	WorkloadSession     *domain.WorkloadTemplateSession
	KernelConnection    jupyter.KernelConnection
	SessionConnection   *jupyter.SessionConnection
	KernelManager       jupyter.KernelSessionManager
	SchedulingPolicy    string
	EventQueue          chan *domain.Event
	MaximumResourceSpec *domain.ResourceRequest

	logger *zap.Logger

	running atomic.Int32
}

// NewClient creates a new Client struct and returns a pointer to it.
func NewClient(sessionId string, workloadId string, sessionReadyEvent *domain.Event, sessionMeta domain.SessionMetadata) *Client {
	client := &Client{
		SessionId:   sessionId,
		WorkloadId:  workloadId,
		EventQueue:  make(chan *domain.Event, 1024),
		SessionMeta: sessionMeta,
		MaximumResourceSpec: &domain.ResourceRequest{
			Cpus:     sessionMeta.GetMaxSessionCPUs(),
			MemoryMB: sessionMeta.GetMaxSessionMemory(),
			Gpus:     sessionMeta.GetMaxSessionGPUs(),
			VRAM:     sessionMeta.GetMaxSessionVRAM(),
		},
	}

	client.EventQueue <- sessionReadyEvent

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

// initialize creates the associated kernel and connects to it.
func (c *Client) initialize() error {
	evt := <-c.EventQueue

	if evt.Name != domain.EventSessionReady {
		c.logger.Error("Received unexpected event for first event.",
			zap.String("session_id", c.SessionId),
			zap.String("event_name", evt.Name.String()),
			zap.String("event", evt.String()),
			zap.String("workload_id", c.WorkloadId))

		return fmt.Errorf("%w: \"%s\"", ErrInvalidFirstEvent, evt.Name.String())
	}

	var initialResourceRequest *jupyter.ResourceSpec
	if c.SchedulingPolicy == "static" || c.SchedulingPolicy == "dynamic-v3" || c.SchedulingPolicy == "dynamic-v4" {
		// Try to get the first training event of the session, and just reserve those resources.
		firstTrainingEvent := c.WorkloadSession.Trainings[0]

		if firstTrainingEvent != nil {
			initialResourceRequest = &jupyter.ResourceSpec{
				Cpu:  int(firstTrainingEvent.Millicpus),
				Mem:  firstTrainingEvent.MemUsageMB,
				Gpu:  firstTrainingEvent.NumGPUs(),
				Vram: firstTrainingEvent.VRamUsageGB,
			}
		} else {
			c.logger.Warn("Could not find first training event of session.",
				zap.String("workload_id", c.WorkloadId),
				zap.String("session_id", c.SessionId))
		}
	} else {
		initialResourceRequest = &jupyter.ResourceSpec{
			Cpu:  int(c.MaximumResourceSpec.Cpus),
			Mem:  c.MaximumResourceSpec.MemoryMB,
			Gpu:  c.MaximumResourceSpec.Gpus,
			Vram: c.MaximumResourceSpec.VRAM,
		}
	}

	sessionConnection, err := c.KernelManager.CreateSession(
		c.SessionId, fmt.Sprintf("%s.ipynb", c.SessionId),
		"notebook", "distributed", initialResourceRequest)
	if err != nil {
		c.logger.Warn("Failed to create session.",
			zap.String("workload_id", c.WorkloadId),
			zap.String(ZapInternalSessionIDKey, c.SessionId),
			zap.Error(err))

		return err
	}

	c.SessionConnection = sessionConnection

	return nil
}

func (c *Client) Run() {
	if !c.running.CompareAndSwap(0, 1) {
		c.logger.Warn("Client is already running.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId))
		return
	}

	err := c.initialize()
	if err != nil {
		c.logger.Error("Failed to initialize client.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId),
			zap.Error(err))

		c.running.Store(0)
		return
	}

	for c.running.Load() == 1 {
		// Do something.
	}
}
