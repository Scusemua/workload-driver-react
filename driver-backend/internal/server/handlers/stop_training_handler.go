package handlers

import (
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/pkg/jupyter"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
)

type StopTrainingHandler struct {
	*BaseHandler

	manager jupyter.KernelSessionManager

	kernelConnections *hashmap.HashMap
}

func NewStopTrainingHandler(opts *domain.Configuration, atom *zap.AtomicLevel) *StopTrainingHandler {
	jupyterAddress := path.Join(opts.InternalJupyterServerAddress, opts.JupyterServerBasePath)

	handler := &StopTrainingHandler{
		BaseHandler:       newBaseHandler(opts, atom),
		manager:           jupyter.NewKernelSessionManager(jupyterAddress, true, atom, metrics.PrometheusMetricsWrapperInstance),
		kernelConnections: hashmap.New(8),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side StopTrainingHandler.")

	return handler
}

func (h *StopTrainingHandler) HandleRequest(c *gin.Context) {
	var (
		req              *domain.StopTrainingRequest
		kernelConnection jupyter.KernelConnection
		err              error
	)

	err = c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to unmarshal StopTrainingRequest.", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	h.logger.Debug("Stopping training for kernel.", zap.String("kernel_id", req.KernelId), zap.String("session_id", req.SessionId))

	kernelConnection, err = h.getKernelConnection(req.KernelId, req.SessionId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}

	h.sugaredLogger.Debugf("Issuing 'stop-training' message to kernel %s now.", req.KernelId)
	err = kernelConnection.StopRunningTrainingCode(false)
	if err != nil {
		h.logger.Error("Failed to stop training.", zap.String("kernel_id", req.KernelId), zap.String("session_id", req.SessionId), zap.String("connection-status", string(kernelConnection.ConnectionStatus())), zap.String("error_message", err.Error()))
		_ = c.AbortWithError(http.StatusInternalServerError, err)

		h.kernelConnections.Del(req.KernelId)
		go func() {
			_ = kernelConnection.Close()
		}()

		return
	}

	h.logger.Debug("Successfully stopped training.", zap.String("kernel_id", req.KernelId), zap.String("session_id", req.SessionId))
}

func (h *StopTrainingHandler) getKernelConnection(kernelId string, sessionId string) (jupyter.KernelConnection, error) {
	var (
		val              interface{}
		ok               bool
		kernelConnection jupyter.KernelConnection
		err              error
	)

	// Check if we already have a cached connection. If not, we'll connect.
	if val, ok = h.kernelConnections.Get(kernelId); !ok {
		h.sugaredLogger.Debugf("No cached connection to kernel %s. Creating new connection now.", kernelId)
		kernelConnection, err = h.manager.ConnectTo(kernelId, sessionId, "")
		if err != nil {
			h.logger.Error("Could not establish connection to kernel in order to stop training.", zap.String("kernel_id", kernelId), zap.String("session_id", sessionId), zap.String("error_message", err.Error()))
			return nil, err
		}
	} else {
		kernelConnection = val.(jupyter.KernelConnection)

		// If the connection is no longer active, then attempt to reconnect.
		if !kernelConnection.Connected() {
			h.sugaredLogger.Debugf("Cached connection to kernel %s is no longer connected. Creating new connection now.", kernelId)
			kernelConnection, err := h.manager.ConnectTo(kernelId, sessionId, "")
			if err != nil {
				h.logger.Error("Could not establish connection to kernel in order to stop training.", zap.String("kernel_id", kernelId), zap.String("session_id", sessionId), zap.String("error_message", err.Error()))
				return nil, err
			} else {
				// Cache the connection.
				h.kernelConnections.Set(kernelId, kernelConnection)
			}
		} else {
			h.sugaredLogger.Debug("Found active, cached connection to kernel %s. Reusing cached connection.", kernelId)
		}
	}

	// On success, err will be nil, and kernelConnection will be non-nil.
	// On error, err will be non-nil, and kernelConnection will be nil.
	return kernelConnection, err
}
