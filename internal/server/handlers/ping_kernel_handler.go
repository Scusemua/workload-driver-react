package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
)

type PingKernelHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewPingKernelHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler) domain.BackendHttpGetHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &PingKernelHttpHandler{
		BaseHandler: newBaseHandler(opts),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side PingKernelHttpHandler.")

	return handler
}

func (h *PingKernelHttpHandler) HandleRequest(c *gin.Context) {
	var req *gateway.PingInstruction
	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to bind request to data type.", zap.Error(err))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	resp, err := h.grpcClient.PingKernel(context.Background(), req)
	if err != nil {
		h.logger.Error("Error while pinging kernel", zap.String("kernel-id", req.KernelId), zap.Error(err))
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if resp.Success {
		h.logger.Debug("Successfully pinged kernel.", zap.String("kernel-id", req.KernelId))
		c.Status(http.StatusOK)
	} else {
		h.logger.Debug("Failed to ping one or more replicas of kernel.", zap.String("kernel-id", req.KernelId))
		c.Status(http.StatusInternalServerError)
	}
}
