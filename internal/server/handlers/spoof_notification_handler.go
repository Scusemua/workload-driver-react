package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
)

type SpoofedNotificationHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewSpoofedNotificationHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler) domain.BackendHttpGetHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &SpoofedNotificationHttpHandler{
		BaseHandler: newBaseHandler(opts),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side SpoofedNotificationHttpHandler.")

	return handler
}

func (h *SpoofedNotificationHttpHandler) HandleRequest(c *gin.Context) {
	h.grpcClient.SpoofNotifications(context.Background(), &gateway.Void{})

	c.JSON(http.StatusOK, make(map[string]interface{}))
}
