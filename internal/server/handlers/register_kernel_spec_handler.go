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

type RegisterKernelResourceSpecHandler struct {
	*GrpcClient
}

func NewRegisterKernelResourceSpecHandler(opts *domain.Configuration) domain.BackendHttpPostHandler {
	handler := &RegisterKernelResourceSpecHandler{
		GrpcClient: NewGrpcClient(opts, !opts.SpoofKernels),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side RegisterKernelResourceSpecHandler.")

	return handler
}

func (h *RegisterKernelResourceSpecHandler) HandleRequest(c *gin.Context) {
	if h.opts.SpoofKernels {
		// Do nothing.
		return
	}

	var resourceSpecRegistration *gateway.ResourceSpecRegistration
	if err := c.BindJSON(&resourceSpecRegistration); err != nil {
		h.logger.Error("Failed to extract and/or unmarshal ResourceSpecRegistration request from request body.")

		// Failed to extract ResourceSpecRegistration request from request body.
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	h.logger.Info("Received ResourceSpecRegistration request.", zap.String("target-kernel", resourceSpecRegistration.KernelId), zap.Any("resource-spec", resourceSpecRegistration.ResourceSpec))

	resp, err := h.rpcClient.RegisterKernelResourceSpec(context.TODO(), resourceSpecRegistration)
	if err != nil {
		h.logger.Error("An error occurred while changing virtual GPUs on node.", zap.String("target-kernel", resourceSpecRegistration.KernelId), zap.Any("resource-spec", resourceSpecRegistration.ResourceSpec), zap.Error(err))

		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("an error occurred while changing virtual GPUs on node: %w : %s", err, err.Error()))
		return
	} else {
		h.logger.Info("Successfully changed the virtual GPUs available on node.", zap.String("target-kernel", resourceSpecRegistration.KernelId), zap.Any("resource-spec", resourceSpecRegistration.ResourceSpec), zap.Any("response", resp))
		c.JSON(http.StatusOK, resp)
	}
}
