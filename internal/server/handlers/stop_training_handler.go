package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/jupyter"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type StopTrainingHandler struct {
	*BaseHandler

	manager jupyter.KernelSessionManager

	kernelConnections *hashmap.HashMap
}

func NewStopTrainingHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	handler := &StopTrainingHandler{
		BaseHandler:       newBaseHandler(opts),
		manager:           jupyter.NewKernelSessionManager(opts, &atom),
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
		val              interface{}
		ok               bool
		err              error
	)

	err = c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to unmarshal StopTrainingRequest.", zap.Error(err))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	h.logger.Debug("Stopping training for kernel.", zap.String("kernel_id", req.KernelId), zap.String("session_id", req.SessionId))

	if val, ok = h.kernelConnections.Get(req.KernelId); !ok {
		h.sugaredLogger.Debugf("No cached connection to kernel %s. Creating new connection now.", req.KernelId)
		kernelConnection, err = h.connectToKernel(req.KernelId, req.SessionId)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else {
		kernelConnection = val.(jupyter.KernelConnection)

		// If the connection is no longer active, then attempt to reconnect.
		if !kernelConnection.Connected() {
			h.sugaredLogger.Debugf("Cached connection to kernel %s is no longer connected. Creating new connection now.", req.KernelId)
			kernelConnection, err = h.connectToKernel(req.KernelId, req.SessionId)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		} else {
			h.sugaredLogger.Debug("Found active, cached connection to kernel %s. Reusing cached connection.", req.KernelId)
		}
	}

	h.sugaredLogger.Debugf("Issuing 'stop-training' message to kernel %s now.", req.KernelId)
	err = kernelConnection.StopRunningTrainingCode(true)
	if err != nil {
		h.logger.Error("Failed to stop training.", zap.String("kernel_id", req.KernelId), zap.String("session_id", req.SessionId), zap.Error(err))
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	h.logger.Debug("Successfully stopped training.", zap.String("kernel_id", req.KernelId), zap.String("session_id", req.SessionId))
}

func (h *StopTrainingHandler) connectToKernel(kernelId string, sessionId string) (jupyter.KernelConnection, error) {
	kernelConnection, err := h.manager.ConnectTo(kernelId, sessionId, "")
	if err != nil {
		h.logger.Error("Could not establish connection to kernel in order to stop training.", zap.String("kernel_id", kernelId), zap.String("session_id", sessionId), zap.Error(err))
	} else {
		h.kernelConnections.Set(kernelId, kernelConnection)
	}

	// On success, err will be nil, and kernelConnection will be non-nil.
	// On error, err will be non-nil, and kernelConnection will be nil.
	return kernelConnection, err
}
