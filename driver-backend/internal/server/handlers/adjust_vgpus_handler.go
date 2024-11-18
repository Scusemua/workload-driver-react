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

type AdjustVirtualGpusHandler struct {
	*BaseHandler

	grpcClient *ClusterDashboardHandler
}

func NewAdjustVirtualGpusHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) domain.BackendHttpGetPatchHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &AdjustVirtualGpusHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side AdjustVirtualGpusHandler.")

	return handler
}

func (h *AdjustVirtualGpusHandler) HandleRequest(c *gin.Context) {
	panic("Not implemented.")
}

func (h *AdjustVirtualGpusHandler) HandlePatchRequest(c *gin.Context) {
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

	resp, err := h.grpcClient.SetTotalVirtualGPUs(context.TODO(), setVirtualGPUsRequest)
	if err != nil {
		domain.LogErrorWithoutStacktrace(h.logger, "An error occurred while changing virtual GPUs on node.",
			zap.String("target-node", setVirtualGPUsRequest.KubernetesNodeName),
			zap.Int32("vGPUs", setVirtualGPUsRequest.Value),
			zap.Error(err))
		h.grpcClient.HandleConnectionError()

		c.AbortWithError(http.StatusNotModified, err)
	} else {
		h.logger.Info("Successfully changed the virtual GPUs available on node.",
			zap.String("target-node", setVirtualGPUsRequest.KubernetesNodeName),
			zap.Int32("vGPUs", setVirtualGPUsRequest.Value),
			zap.Any("response", resp))
		c.JSON(http.StatusOK, resp)
	}
}
