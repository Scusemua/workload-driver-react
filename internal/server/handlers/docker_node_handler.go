package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	protoNodes := resp.GetNodes()
	h.logger.Debug("Retrieved virtual Docker nodes.", zap.Int("num-nodes", len(protoNodes)))

	var nodes = make([]*domain.VirtualDockerNode, 0, len(protoNodes))
	for _, protoNode := range protoNodes {
		virtualDockerNode := domain.VirtualDockerNodeFromProtoVirtualDockerNode(protoNode)
		nodes = append(nodes, virtualDockerNode)
	}

	h.sugaredLogger.Debugf("Returning %d virtual Docker node(s): %v", len(nodes), nodes)

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

	var req map[string]interface{}
	c.BindJSON(&req)

	targetNumNodesVal, ok := req["target_num_nodes"]
	if !ok {
		h.logger.Error("HTTP PATCH request for /nodes endpoint missing \"target_num_nodes\" entry in JSON payload.")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	_, err := h.grpcClient.SetNumVirtualDockerNodes(context.TODO(), &gateway.SetNumVirtualDockerNodesRequest{
		RequestId:      uuid.NewString(),
		TargetNumNodes: targetNumNodesVal.(int32),
	})

	if err != nil {
		status, ok := status.FromError(err)
		if ok {
			switch status.Code() {
			case codes.Internal:
				{
					c.AbortWithError(http.StatusInternalServerError, status.Err())
				}
			case codes.FailedPrecondition:
				{
					c.AbortWithError(http.StatusBadRequest, status.Err())
				}
			case codes.InvalidArgument:
				{
					c.AbortWithError(http.StatusBadRequest, status.Err())
				}
			default:
				{
					c.AbortWithError(http.StatusInternalServerError, status.Err())
				}
			}
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	c.Status(http.StatusOK)
}
