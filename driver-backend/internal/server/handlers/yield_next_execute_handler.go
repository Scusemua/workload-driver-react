package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
)

type YieldNextExecuteHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewYieldNextExecuteHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) domain.BackendHttpPostHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &YieldNextExecuteHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side YieldNextExecuteHandler.")

	handler.logger.Info("Successfully created server-side YieldNextExecuteHandler handler.")

	return handler
}

func (h *YieldNextExecuteHandler) HandleRequest(c *gin.Context) {
	st := time.Now()
	h.logger.Debug("Handling 'yield-next-execute-request' request now.")

	var req map[string]interface{}
	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to bind request to JSON for 'yield-next-execute-request'", zap.Error(err))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	val, ok := req["kernel_id"]
	if !ok {
		h.logger.Error("Request did not contain 'kernelId' entry.")
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	kernelId := val.(string)
	if len(kernelId) == 0 || len(kernelId) > 36 {
		h.sugaredLogger.Errorf("Request contained invalid 'kernelId': \"%s\".", kernelId)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	_, err = h.grpcClient.FailNextExecution(context.TODO(), &gateway.KernelId{
		Id: kernelId,
	})
	if err != nil {
		h.logger.Error("'FailNextExecution' gRPC call encountered an error.", zap.Error(err))
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	h.sugaredLogger.Debugf("Handled 'yield-next-execute-request' request in %v.", time.Since(st))
	c.Status(http.StatusOK)
}
