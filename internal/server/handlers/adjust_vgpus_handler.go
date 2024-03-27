package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
)

type AdjustVirtualGpusHandler struct {
	*GrpcClient
}

func NewAdjustVirtualGpusHandler(opts *domain.Configuration) domain.BackendHttpGetPatchHandler {
	handler := &AdjustVirtualGpusHandler{
		GrpcClient: NewGrpcClient(opts, !opts.SpoofKernels),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side AdjustVirtualGpusHandler.")

	return handler
}

func (h *AdjustVirtualGpusHandler) HandleRequest(c *gin.Context) {
	panic("Not implemented.")
}

func (h *AdjustVirtualGpusHandler) HandlePatchRequest(c *gin.Context) {
	if h.opts.SpoofKernels {
		// Do nothing.
		return
	}

	var setVirtualGPUsRequest *gateway.SetVirtualGPUsRequest
	if err := c.BindJSON(&setVirtualGPUsRequest); err != nil {
		h.logger.Error("Failed to extract and/or unmarshal SetVirtualGPUsRequest request from request body.")

		c.JSON(http.StatusBadRequest, &domain.ErrorMessage{
			Description:  "Failed to extract SetVirtualGPUsRequest request from request body.",
			ErrorMessage: err.Error(),
			Valid:        true,
		})
	}

	h.logger.Info("Received SetVirtualGPUsRequest request.", zap.String("target-node", setVirtualGPUsRequest.KubernetesNodeName), zap.Int32("vGPUs", setVirtualGPUsRequest.Value))

	resp, err := h.rpcClient.SetTotalVirtualGPUs(context.TODO(), setVirtualGPUsRequest)
	if err != nil {
		h.logger.Error("An error occurred while changing virtual GPUs on node.", zap.String("target-node", setVirtualGPUsRequest.KubernetesNodeName), zap.Int32("vGPUs", setVirtualGPUsRequest.Value), zap.Error(err))

		c.JSON(http.StatusNotModified, &domain.ErrorMessage{
			Description:  "An error occurred while changing virtual GPUs on node",
			ErrorMessage: err.Error(),
			Valid:        true,
		})
	} else {
		h.logger.Info("Successfully changed the virtual GPUs available on node.", zap.String("target-node", setVirtualGPUsRequest.KubernetesNodeName), zap.Int32("vGPUs", setVirtualGPUsRequest.Value), zap.Any("response", resp))
		c.JSON(http.StatusOK, resp)
	}
}
