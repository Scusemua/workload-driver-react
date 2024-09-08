package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"net/http"
	"path"
	"time"
)

// VariablesHttpHandler returns variables that are used by Grafana to generate data visualizations/dashboards.
type VariablesHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler
}

func NewVariablesHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler) *VariablesHttpHandler {
	if opts == nil {
		panic("opts cannot be nil.")
	}

	if grpcClient == nil {
		panic("grpcClient cannot be nil.")
	}

	handler := &VariablesHttpHandler{
		BaseHandler: newBaseHandler(opts),
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

	return resp.HostIds, nil
}

func (h *VariablesHttpHandler) HandleRequest(c *gin.Context) {
	variable := path.Base(c.Request.RequestURI)
	h.logger.Debug("Received query for variable.", zap.String("variable", variable))

	response := make(map[string]interface{})
	switch variable {
	case "num_nodes":
		{
			localDaemonIds, err := h.getLocalDaemonIDs()
			if err != nil {
				_ = c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
			response["num_nodes"] = len(localDaemonIds)
		}
	case "local_daemon_ids":
		{
			localDaemonIds, err := h.getLocalDaemonIDs()
			if err != nil {
				_ = c.AbortWithError(http.StatusInternalServerError, err)
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
