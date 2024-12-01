package handlers

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/workload"
	"go.uber.org/zap"
	"net/http"
)

type ClusterStatisticsHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewClusterStatisticsHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *ClusterStatisticsHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &ClusterStatisticsHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side ClusterStatisticsHttpHandler.")

	return handler
}

// HandleDeleteRequest clears the Cluster Statistics.
func (h *ClusterStatisticsHttpHandler) HandleDeleteRequest(c *gin.Context) {
	h.logger.Debug("Clearing cluster statistics as instructed by HTTP DELETE request.")
	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))
		return
	}

	resp, err := h.grpcClient.ClearClusterStatistics(context.Background(), &gateway.Void{})
	if err != nil {
		h.logger.Error("Failed to clear Cluster Statistics while handling associated HTTP DELETE request.", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var clusterStatistics *workload.ClusterStatistics

	buffer := bytes.NewBuffer(resp.SerializedClusterStatistics)
	decoder := gob.NewDecoder(buffer)

	err = decoder.Decode(&clusterStatistics)
	if err != nil {
		h.logger.Error("Failed to decode Cluster Statistics while handling associated HTTP DELETE request.", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, clusterStatistics)
}

func (h *ClusterStatisticsHttpHandler) HandleRequest(c *gin.Context) {
	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))
		return
	}

	var request map[string]bool
	err := c.BindJSON(&request)
	if err != nil {
		h.logger.Error("Failed to bind JSON for ClusterStatistics request.", zap.Error(err))
	}
	update, loaded := request["update"]

	requestId := uuid.NewString()
	h.logger.Debug("Retrieving cluster statistics as instructed by HTTP GET request.",
		zap.String("request_id", requestId),
		zap.Bool("update", update && loaded))

	resp, err := h.grpcClient.ClusterStatistics(context.Background(), &gateway.ClusterStatisticsRequest{
		RequestId:   requestId,
		UpdateFirst: update && loaded,
	})
	if err != nil {
		h.logger.Error("Failed to retrieve Cluster Statistics while handling associated HTTP GET request.", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var clusterStatistics *workload.ClusterStatistics

	buffer := bytes.NewBuffer(resp.SerializedClusterStatistics)
	decoder := gob.NewDecoder(buffer)

	err = decoder.Decode(&clusterStatistics)
	if err != nil {
		h.logger.Error("Failed to decode Cluster Statistics while handling associated HTTP GET request.", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, clusterStatistics)
}
