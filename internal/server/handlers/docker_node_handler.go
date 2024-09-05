package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"net/http"
)

// DockerSwarmNodeHttpHandler handles HTTP GET and HTTP PATCH requests that respectively retrieve and modify
// nodes within the distributed cluster. These nodes are represented by the domain.ClusterNode interface.
//
// DockerSwarmNodeHttpHandler is used as the internal node handler by the NodeHttpHandler struct when the cluster is
// running/deployed in Docker Swarm mode.
//
// DockerSwarmNodeHttpHandler issues gRPC requests to the Cluster Gateway to query and modify nodes within the cluster.
type DockerSwarmNodeHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler

	// The handler will return 0 nodes until this flag is flipped to true.
	nodeTypeRegistered bool
}

// NewDockerSwarmNodeHttpHandler creates a new DockerSwarmNodeHttpHandler struct and return a pointer to it.
func NewDockerSwarmNodeHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler) *DockerSwarmNodeHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &DockerSwarmNodeHttpHandler{
		BaseHandler:        newBaseHandler(opts),
		grpcClient:         grpcClient,
		nodeTypeRegistered: false,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side DockerSwarmNodeHttpHandler.")

	handler.logger.Info("Successfully created server-side DockerSwarmNodeHttpHandler handler.")

	return handler
}

// HandleRequest handles an HTTP GET request, which retrieves a list of the nodes provisioned within the cluster.
func (h *DockerSwarmNodeHttpHandler) HandleRequest(c *gin.Context) {
	if h.grpcClient == nil {
		h.logger.Error("gRPC Client cannot be nil while handling HTTP GET request.")
		c.Status(http.StatusInternalServerError)
		return
	}

	resp, err := h.grpcClient.GetVirtualDockerNodes(context.Background(), &gateway.Void{})
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	nodes := resp.GetNodes()
	h.logger.Debug("Retrieved virtual Docker nodes.", zap.Int("num-nodes", len(nodes)))

	c.JSON(http.StatusOK, nodes)
}

// HandlePatchRequest handles an HTTP PATCH requests, which enable making changes to the nodes,
// such as changing the number of nodes manually, or adjusting the resources available on one or more nodes.
func (h *DockerSwarmNodeHttpHandler) HandlePatchRequest(c *gin.Context) {
	if h.grpcClient == nil {
		h.logger.Error("gRPC Client cannot be nil while handling HTTP GET request.")
		c.Status(http.StatusInternalServerError)
		return
	}

	h.logger.Warn("HTTP PATCH requests are not yet supported by the DockerSwarmNodeHttpHandler")
	c.Status(http.StatusNotImplemented)
	return
}
