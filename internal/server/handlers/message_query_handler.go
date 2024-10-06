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

	handler.logger.Info("Created server-side MessageQueryHttpHandler.")

	return handler
}

func (h *MessageQueryHttpHandler) HandleRequest(c *gin.Context) {
	h.logger.Debug("Received new QueryMessage request.")

	var req *gateway.QueryMessageRequest

	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to bind JSON for QueryMessage request.", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	h.logger.Debug("Querying status of Jupyter message.", zap.Object("query_request", req))

	resp, err := h.grpcClient.QueryMessage(context.Background(), req)

	if err != nil {
		h.logger.Error("Failed to query message status.", zap.Error(err))

		_ = c.Error(err)

		c.JSON(domain.GRPCStatusToHTTPStatus(err), &domain.ErrorMessage{
			ErrorMessage: err.Error(),
			Valid:        true,
		})

		return
	}

	h.logger.Debug("Successfully queried message status.", zap.Object("query_result", resp))

	c.JSON(http.StatusOK, resp)
}
