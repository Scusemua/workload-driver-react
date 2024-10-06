package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"net/http"
)

type MessageQueryHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewMessageQueryHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler) domain.BackendHttpGetHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &MessageQueryHttpHandler{
		BaseHandler: newBaseHandler(opts),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side MessageQueryHttpHandler.")

	return handler
}

func (h *MessageQueryHttpHandler) HandleRequest(c *gin.Context) {
	var req *gateway.QueryMessageRequest

	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to bind JSON for QueryMessage request.", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	resp, err := h.grpcClient.QueryMessage(context.Background(), req)

	if err != nil {
		h.logger.Error("Failed to query message status.", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	h.logger.Debug("Successfully queried message status.", zap.Object("query_result", resp))

	c.JSON(http.StatusOK, resp)
}
