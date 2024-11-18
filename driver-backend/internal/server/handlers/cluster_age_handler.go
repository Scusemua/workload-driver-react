package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type ClusterAgeHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewClusterAgeHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *ClusterAgeHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &ClusterAgeHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side ClusterAgeHttpHandler.")

	return handler
}

func (h *ClusterAgeHttpHandler) HandleRequest(c *gin.Context) {
	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))
		return
	}

	age, err := h.grpcClient.ClusterAge(context.Background(), &gateway.Void{})
	if err != nil {
		h.logger.Error("Failed to retrieve Cluster age.", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	createdAt := time.UnixMilli(age.Age)
	h.logger.Debug("Successfully retrieved Cluster age.", zap.Time("cluster_created-at", createdAt),
		zap.Duration("cluster_age", time.Since(createdAt)))

	c.String(http.StatusOK, "%d", age.Age)
}
