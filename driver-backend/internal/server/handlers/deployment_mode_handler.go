package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"net/http"
)

type DeploymentModeHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewDeploymentModeHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *DeploymentModeHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &DeploymentModeHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side DeploymentModeHttpHandler.")

	return handler
}

func (h *DeploymentModeHttpHandler) HandleRequest(c *gin.Context) {
	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))
		return
	}

	c.String(http.StatusOK, h.grpcClient.deploymentMode)
}
