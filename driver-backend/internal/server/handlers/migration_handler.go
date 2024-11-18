package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
)

type MigrationHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewMigrationHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *MigrationHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &MigrationHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side MigrationHttpHandler.")

	return handler
}

func (h *MigrationHttpHandler) HandleRequest(c *gin.Context) {
	//if h.opts.SpoofKernels {
	//	// Do nothing.
	//	return
	//}

	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))

		h.grpcClient.HandleConnectionError()

		return
	}

	var migrationRequest *gateway.MigrationRequest
	if err := c.BindJSON(&migrationRequest); err != nil {
		h.logger.Error("Failed to extract and/or unmarshal migration request from request body.")

		_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("ErrBadRequest %w: %s", err, err.Error()))
		return
	}

	h.logger.Info("Received migration request.", zap.Int32("replica-smr-id", migrationRequest.TargetReplica.ReplicaId), zap.String("kernel_id", migrationRequest.TargetReplica.KernelId), zap.String("target-node-id", migrationRequest.GetTargetNodeId()))

	resp, err := h.grpcClient.MigrateKernelReplica(context.TODO(), migrationRequest)
	if err != nil {
		domain.LogErrorWithoutStacktrace(h.logger, "An error occurred while triggering or performing the kernel replica migration.", zap.String("kernelID", migrationRequest.TargetReplica.KernelId), zap.Int32("replicaID", migrationRequest.TargetReplica.ReplicaId), zap.String("target-node", migrationRequest.GetTargetNodeId()), zap.Error(err))
		h.grpcClient.HandleConnectionError()

		_ = c.AbortWithError(http.StatusInternalServerError, err)
	} else {
		h.logger.Info("Successfully triggered kernel replica migration.", zap.String("kernelID", migrationRequest.TargetReplica.KernelId), zap.Int32("replicaID", migrationRequest.TargetReplica.ReplicaId), zap.String("target-node", migrationRequest.GetTargetNodeId()), zap.Any("response", resp))
	}
}
