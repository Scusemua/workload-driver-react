package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"net/http"
)

type SchedulingPolicyHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewSchedulingPolicyHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *SchedulingPolicyHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &SchedulingPolicyHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side SchedulingPolicyHttpHandler.")

	return handler
}

func (h *SchedulingPolicyHttpHandler) HandleRequest(c *gin.Context) {
	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))
		return
	}

	c.String(http.StatusOK, h.grpcClient.schedulingPolicy)
}
