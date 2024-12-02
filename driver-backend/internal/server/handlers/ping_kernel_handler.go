package handlers

import (
	"context"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
)

type PingKernelHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewPingKernelHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *PingKernelHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &PingKernelHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side PingKernelHttpHandler.")

	return handler
}

func (h *PingKernelHttpHandler) HandleRequest(c *gin.Context) {

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read request body.", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	var req proto.PingInstruction
	err = protojson.Unmarshal(jsonData, &req)
	if err != nil {
		h.logger.Error("Failed to unmarshal request body to PingInstruction.",
			zap.ByteString("body", jsonData),
			zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	resp, err := h.grpcClient.PingKernel(context.Background(), &req)
	if err != nil {
		h.logger.Error("Error while pinging kernel", zap.String("kernel_id", req.KernelId), zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if resp.Success {
		h.logger.Debug("Successfully pinged kernel.", zap.String("kernel_id", req.KernelId), zap.Array("request_traces", proto.RequestTraceArr(resp.RequestTraces)))
		c.JSON(http.StatusOK, resp)
	} else {
		h.logger.Debug("Failed to ping one or more replicas of kernel.", zap.String("kernel_id", req.KernelId))
		c.Status(http.StatusInternalServerError)
	}
}
