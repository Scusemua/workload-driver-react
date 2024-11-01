package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
)

type PanicHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewPanicHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler) *PanicHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &PanicHttpHandler{
		BaseHandler: newBaseHandler(opts),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side PanicHttpHandler.")

	return handler
}

func (h *PanicHttpHandler) HandleRequest(c *gin.Context) {
	h.grpcClient.InducePanic(context.Background(), &gateway.Void{})

	c.JSON(http.StatusOK, make(map[string]interface{}))
}
