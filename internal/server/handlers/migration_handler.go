package handlers

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
)

type MigrationHttpHandler struct {
	*BaseGRPCHandler
}

func NewMigrationHttpHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	handler := &MigrationHttpHandler{
		BaseGRPCHandler: newBaseGRPCHandler(opts, !opts.SpoofKernels),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info(fmt.Sprintf("Creating server-side MigrationHttpHandler.\nOptions: %s", opts))

	return handler
}

func (h *MigrationHttpHandler) HandleRequest(c *gin.Context) {
	var migrationRequest *gateway.MigrationRequest
	if err := c.BindJSON(&migrationRequest); err != nil {
		h.logger.Error("Failed to extract and/or unmarshal migration request from request body.")

		c.JSON(400, &domain.ErrorMessage{
			Description:  "Failed to extract migration request from request body.",
			ErrorMessage: err.Error(),
			Valid:        true,
		})
	}

	resp, err := h.rpcClient.MigrateKernelReplica(context.TODO(), migrationRequest)
	if err != nil {
		h.logger.Error("An error occurred while triggering or performing the kernel replica migration.", zap.String("kernelID", migrationRequest.TargetReplica.KernelId), zap.Int32("replicaID", migrationRequest.TargetReplica.ReplicaId), zap.String("target-node", migrationRequest.GetTargetNodeId()), zap.Error(err))

		c.JSON(500, &domain.ErrorMessage{
			Description:  "An error occurred while triggering or performing the kernel replica migration.",
			ErrorMessage: err.Error(),
			Valid:        true,
		})
	} else {
		h.logger.Info("Successfully triggered kernel replica migration.", zap.String("kernelID", migrationRequest.TargetReplica.KernelId), zap.Int32("replicaID", migrationRequest.TargetReplica.ReplicaId), zap.String("target-node", migrationRequest.GetTargetNodeId()), zap.Any("response", resp))
	}
}
