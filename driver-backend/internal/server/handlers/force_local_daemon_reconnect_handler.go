package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"net/http"
)

type ForceLocalDaemonToReconnectHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewForceLocalDaemonToReconnectHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *ForceLocalDaemonToReconnectHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &ForceLocalDaemonToReconnectHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Created server-side ForceLocalDaemonToReconnectHttpHandler.")

	return handler
}

func (h *ForceLocalDaemonToReconnectHttpHandler) HandleRequest(c *gin.Context) {
	h.logger.Debug("Received new QueryMessage request.")

	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))

		h.grpcClient.HandleConnectionError()

		return
	}

	var req *gateway.ForceLocalDaemonToReconnectRequest

	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to bind JSON for ForceLocalDaemonToReconnect request.", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	h.logger.Debug("Instructing Local Daemon to reconnect to Cluster Gateway.",
		zap.String("ID", req.LocalDaemonId), zap.Bool("delay", req.Delay))

	_, err = h.grpcClient.ForceLocalDaemonToReconnect(context.Background(), req)
	if err != nil {
		h.logger.Error("Failed to instruct local daemon to reconnect message status.",
			zap.String("ID", req.LocalDaemonId), zap.Bool("delay", req.Delay), zap.Error(err))

		c.Status(domain.GRPCStatusToHTTPStatus(err))

		_ = c.Error(err)

		return
	}

	h.logger.Debug("Successfully instructed local daemon to reconnect.",
		zap.String("ID", req.LocalDaemonId), zap.Bool("delay", req.Delay))

	c.Status(http.StatusOK)
}
