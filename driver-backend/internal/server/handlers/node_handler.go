package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"net/http"
)

type NodeHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler

	// Once we know what type of nodes are available within the cluster (Docker Swarm nodes or Kubernetes nodes),
	// we create an internal domain.BackendHttpGetPatchHandler for our own private use.
	//
	// The internal domain.BackendHttpGetPatchHandler will be of type KubeNodeHttpHandler or DockerSwarmNodeHttpHandler.
	internalNodeHandler domain.BackendHttpGetPatchHandler

	// The handler will return 0 nodes until this flag is flipped to true.
	nodeTypeRegistered bool

	// The type of nodes available within the cluster (either domain.KubernetesNodeType or domain.DockerSwarmNodeType).
	nodeType domain.NodeType
}

func NewNodeHttpHandler(
	opts *domain.Configuration, atom *zap.AtomicLevel) *NodeHttpHandler {
	if opts == nil {
		panic("opts cannot be nil.")
	}

	handler := &NodeHttpHandler{
		BaseHandler:        newBaseHandler(opts, atom),
		nodeTypeRegistered: false,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Successfully created server-side NodeHttpHandler handler.")

	return handler
}

// HandleRequest handles an HTTP GET request, which retrieves a list of the nodes provisioned within the cluster.
//
// In the case of NodeHttpHandler's HandleRequest method, the request is offloaded to the internal node handler
// if it has been configured and created.
//
// If the internal node creator has not been set up yet, then HandleRequest returns HTTP 503 "Service Unavailable".
func (h *NodeHttpHandler) HandleRequest(c *gin.Context) {
	if !h.nodeTypeRegistered {
		h.logger.Warn(
			"Node type has not been configured yet. Returning HTTP 503 \"Service Unavailable\" for HTTP GET request.")
		c.Status(http.StatusServiceUnavailable)
		return
	}

	h.logger.Debug("Handing off HTTP GET request for Cluster Nodes to internal handler.")
	// Forward the request to the internal node handler.
	h.internalNodeHandler.HandleRequest(c)
}

// HandlePatchRequest handles an HTTP PATCH requests, which enable making changes to the nodes,
// such as changing the number of nodes manually, or adjusting the resources available on one or more nodes.
//
// In the case of NodeHttpHandler's HandlePatchRequest method, the request is offloaded to the internal node handler
// if it has been configured and created.
//
// If the internal node creator has not been set up yet, then HandlePatchRequest returns HTTP 503 "Service Unavailable".
func (h *NodeHttpHandler) HandlePatchRequest(c *gin.Context) {
	if !h.nodeTypeRegistered {
		h.logger.Warn(
			"Node type has not been configured yet. Returning HTTP 503 \"Service Unavailable\" for HTTP PATCH request.")
		c.Status(http.StatusServiceUnavailable)
		return
	}

	h.logger.Debug("Handing off HTTP PATCH request for Cluster Nodes to internal handler.")
	// Forward the request to the internal node handler.
	h.internalNodeHandler.HandlePatchRequest(c)
}

// AssignNodeType informs the NodeHttpHandler of the type of nodes available within the distributed cluster.
// This prompts the NodeHttpHandler to create its internal node handler accordingly.
//
// If the internal node handler has already been created, then this replaces the previous node handler with a new one
// of the specified type.
//
// If no internal node handler had been configured prior to this call, then false is returned.
//
// If there was already a node handler configured prior to this call, then one of two things will happen.
// (i) If the existing node handler is of the specified type, then nothing happens, and false is returned.
// (ii) If the existing node handler is NOT of the specified type, then it is replaced with a new node handler of the
// correct type, and true is returned.
func (h *NodeHttpHandler) AssignNodeType(nodeType domain.NodeType, grpcClient *ClusterDashboardHandler) bool {
	var overwroteExistingHandlerOfDifferentType = false

	// Check if we've already registered an internal node handler.
	if h.nodeTypeRegistered {
		h.logger.Warn(
			"Instructed to assign node type for NodeHttpHandler; however, node type has already been configured.",
			zap.String("existing-node-type", string(h.nodeType)), zap.String("new-node-type", string(nodeType)))

		// If the type differs, then set overwroteExistingHandlerOfDifferentType to true.
		if h.nodeType != nodeType {
			overwroteExistingHandlerOfDifferentType = true
		}
	}

	// Record that we've now registered the node type.
	h.nodeTypeRegistered = true
	h.nodeType = nodeType
	h.grpcClient = grpcClient

	// Create and assign the correct type of node handler for our internal node handler.
	switch nodeType {
	case domain.VirtualDockerNodeType:
		{
			h.createDockerSwarmNodeHandler()
		}
	case domain.DockerSwarmNodeType:
		{
			h.createDockerSwarmNodeHandler()
		}
	case domain.KubernetesNodeType:
		{
			h.createKubeNodeHandler()
		}
	}

	return overwroteExistingHandlerOfDifferentType
}

// createKubeNodeHandler creates a KubeNodeHttpHandler and assigns it to our internalNodeHandler.
//
// This panics if nodeTypeRegistered is false or if nodeType is not equal to domain.KubernetesNodeType.
func (h *NodeHttpHandler) createKubeNodeHandler() {
	if !h.nodeTypeRegistered {
		panic("cannot create Kubernetes Node Handler until the node type has officially been registered")
	}

	if h.nodeType != domain.KubernetesNodeType {
		panic(fmt.Sprintf("cannot create Kubernetes Node Handler; our node type is incompatible: \"%s\"", h.nodeType))
	}

	h.internalNodeHandler = NewKubeNodeHttpHandler(h.opts, h.grpcClient, h.atom)

	h.logger.Debug("Created and assigned KubeNodeHttpHandler as internal handler of NodeHttpHandler.")
}

// createDockerSwarmNodeHandler creates a DockerSwarmNodeHttpHandler and assigns it to our internalNodeHandler.
//
// This panics if nodeTypeRegistered is false or if nodeType is not equal to domain.DockerSwarmNodeType.
func (h *NodeHttpHandler) createDockerSwarmNodeHandler() {
	if !h.nodeTypeRegistered {
		panic("cannot create Docker Swarm Node Handler until the node type has officially been registered")
	}

	if h.nodeType != domain.DockerSwarmNodeType && h.nodeType != domain.VirtualDockerNodeType {
		panic(fmt.Sprintf("cannot create Docker Swarm Node Handler; our node type is incompatible: \"%s\"", h.nodeType))
	}

	h.internalNodeHandler = NewDockerSwarmNodeHttpHandler(h.opts, h.grpcClient, h.atom)

	h.logger.Debug("Created and assigned DockerSwarmNodeHttpHandler as internal handler of NodeHttpHandler.")
}
