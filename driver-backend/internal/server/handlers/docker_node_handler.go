package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
func NewDockerSwarmNodeHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *DockerSwarmNodeHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &DockerSwarmNodeHttpHandler{
		BaseHandler:        newBaseHandler(opts, atom),
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

	h.logger.Debug("Serving HTTP GET request for Docker Swarm nodes.")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, err := h.grpcClient.GetVirtualDockerNodes(ctx, &gateway.Void{})
	if err != nil {
		h.logger.Error("Failed to retrieve virtual Docker nodes from Cluster Gateway.", zap.Error(err))
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

	h.sugaredLogger.Debugf("Returning %d virtual Docker node(s): %v", len(protoNodes), nodes)

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
	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to parse JSON included with HTTP PATCH request for number of cluster nodes.", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	targetNumNodesVal, ok := req["target_num_nodes"]
	if !ok {
		h.logger.Error("HTTP PATCH request for /nodes endpoint missing \"target_num_nodes\" entry in JSON payload.")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	operationVal, loaded := req["op"]
	if !loaded {
		err = fmt.Errorf("invalid request: missing \"op\" field")
		h.logger.Error("HTTP PATCH request for virtual docker nodes failed due to an invalid argument.", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	operation := operationVal.(string)

	h.logger.Debug("Preparing to adjust number of nodes.", zap.String("op", operation), zap.Int("nodes", int(targetNumNodesVal.(float64))))

	switch operation {
	case "set_nodes":
		{
			targetNumNodes := int32(targetNumNodesVal.(float64))
			h.logger.Debug("Issuing 'SetNumClusterNodes' request.", zap.Int32("target_num_nodes", targetNumNodes))
			_, err = h.grpcClient.SetNumClusterNodes(context.TODO(), &gateway.SetNumClusterNodesRequest{
				RequestId:      uuid.NewString(),
				TargetNumNodes: targetNumNodes,
			})
			break
		}
	case "add_nodes":
		{
			targetNumNodes := int32(targetNumNodesVal.(float64))
			h.logger.Debug("Issuing 'AddClusterNodes' request.", zap.Int32("target_num_nodes", targetNumNodes))
			_, err = h.grpcClient.AddClusterNodes(context.TODO(), &gateway.AddClusterNodesRequest{
				RequestId: uuid.NewString(),
				NumNodes:  targetNumNodes,
			})
			break
		}
	case "remove_nodes":
		{
			targetNumNodes := int32(targetNumNodesVal.(float64))
			h.logger.Debug("Issuing 'RemoveClusterNodes' request.", zap.Int32("target_num_nodes", targetNumNodes))
			_, err = h.grpcClient.RemoveClusterNodes(context.TODO(), &gateway.RemoveClusterNodesRequest{
				RequestId:        uuid.NewString(),
				NumNodesToRemove: targetNumNodes,
			})
			break
		}
	default:
		{
			err = fmt.Errorf("invalid request: \"op\" field has unknown or unexpected value: \"%s\"", operation)
			h.logger.Error("HTTP PATCH request for virtual docker nodes failed due to an invalid argument.", zap.Error(err))
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	if err != nil {
		operationStatus, ok := status.FromError(err)
		if ok {
			switch operationStatus.Code() {
			case codes.Internal:
				{
					h.logger.Error("HTTP PATCH request for virtual docker nodes failed due to an internal error.", zap.Error(operationStatus.Err()))
					_ = c.AbortWithError(http.StatusInternalServerError, operationStatus.Err())
				}
			case codes.FailedPrecondition:
				{
					h.logger.Error("HTTP PATCH request for virtual docker nodes failed due to a failed precondition.", zap.Error(operationStatus.Err()))
					_ = c.AbortWithError(http.StatusBadRequest, operationStatus.Err())
				}
			case codes.InvalidArgument:
				{
					h.logger.Error("HTTP PATCH request for virtual docker nodes failed due to an invalid argument.", zap.Error(operationStatus.Err()))
					_ = c.AbortWithError(http.StatusBadRequest, operationStatus.Err())
				}
			default:
				{
					h.logger.Error("HTTP PATCH request for virtual docker nodes failed due to an unknown/unexpected error.", zap.Error(operationStatus.Err()))
					_ = c.AbortWithError(http.StatusInternalServerError, operationStatus.Err())
				}
			}
		} else {
			h.logger.Error("HTTP PATCH request for virtual docker nodes failed due to an unknown/unexpected error.", zap.Error(err))
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	h.logger.Debug("Requested scale operation has completed successfully.",
		zap.String("op", operation), zap.Int("nodes", int(targetNumNodesVal.(float64))))
	c.Status(http.StatusOK)
}
