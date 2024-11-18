package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// VariablesHttpHandler returns variables that are used by Grafana to generate data visualizations/dashboards.
type VariablesHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler

	// cachedNodeIds is the last response received from the GetLocalDaemonNodeIDs gRPC function.
	// We use this to return values to queries if future requests to GetLocalDaemonNodeIDs time-out.
	cachedNodeIds []string
}

func NewVariablesHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *VariablesHttpHandler {
	if opts == nil {
		panic("opts cannot be nil.")
	}

	if grpcClient == nil {
		panic("grpcClient cannot be nil.")
	}

	handler := &VariablesHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
	}

	handler.BackendHttpGetHandler = handler
	handler.logger.Info("Successfully created server-side VariablesHttpHandler handler.")

	return handler
}

// getLocalDaemonIDs issues a gRPC call to the Cluster Gateway to retrieve a string
// slice containing the host IDs of all the active Local Daemons in the Cluster.
func (h *VariablesHttpHandler) getLocalDaemonIDs() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := h.grpcClient.GetLocalDaemonNodeIDs(ctx, &proto.Void{})
	if err != nil {
		return nil, err
	}

	// Update the cached response.
	h.cachedNodeIds = resp.HostIds
	return resp.HostIds, nil
}

func (h *VariablesHttpHandler) HandleRequest(c *gin.Context) {
	variable := c.Param("variable_name")
	h.logger.Debug("Received query for variable.", zap.String("variable", variable))

	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")
		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))

		h.grpcClient.HandleConnectionError()

		return
	}

	response := make(map[string]interface{})
	switch variable {
	case "num_nodes":
		{
			localDaemonIds, err := h.getLocalDaemonIDs()
			if err != nil {
				// If we have a cached response available, then we'll return that along with an error.
				// The status code will still indicate that an error occurred, however.
				if h.cachedNodeIds != nil {
					// Return the cached response.
					response["num_nodes"] = len(h.cachedNodeIds)
					// Include the error in the response.
					response["error"] = err.Error()
					c.AbortWithStatusJSON(http.StatusInternalServerError, response)
				} else {
					// We don't have an old response cached, so just abort with an error.
					_ = c.AbortWithError(http.StatusInternalServerError, err)
				}
				return
			}
			response["num_nodes"] = len(localDaemonIds)
		}
	case "local_daemon_ids":
		{
			localDaemonIds, err := h.getLocalDaemonIDs()
			if err != nil {
				// If we have a cached response available, then we'll return that along with an error.
				// The status code will still indicate that an error occurred, however.
				if h.cachedNodeIds != nil {
					// Return the cached response.
					response["local_daemon_ids"] = h.cachedNodeIds
					// Include the error in the response.
					response["error"] = err.Error()
					c.AbortWithStatusJSON(http.StatusInternalServerError, response)
				} else {
					// We don't have an old response cached, so just abort with an error.
					_ = c.AbortWithError(http.StatusInternalServerError, err)
				}
				return
			}
			response["local_daemon_ids"] = localDaemonIds
		}
	case "default":
		{
			h.logger.Error("Received variable query for unknown variable.", zap.String("variable", variable))
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unknown or unsupported variable: \"%s\"", variable))
			return
		}
	}

	c.JSON(http.StatusOK, response)
}
