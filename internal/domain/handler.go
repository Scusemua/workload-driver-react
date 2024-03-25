package domain

import "github.com/gin-gonic/gin"

type BackendHttpGetHandler interface {
	// Write an error back to the client.
	WriteError(*gin.Context, string)

	// Handle a message/request from the front-end.
	HandleRequest(*gin.Context)

	// Return the request handler responsible for handling a majority of requests.
	PrimaryHttpHandler() BackendHttpGetHandler
}

type BackendHttpGetPostHandler interface {
	BackendHttpGetHandler

	// Handle a message/request from the front-end.
	HandlePostRequest(*gin.Context)
}

type EnableDisableNodeRequest struct {
	NodeName string `json:"node_name"`
	Enable   bool   `json:"enable"` // If true, enable the node. Otherwise, disable the node.
}

type JupyterApiHttpHandler interface {
	// Handle an HTTP GET request to get the jupyter kernel specs.
	HandleGetKernelSpecRequest(*gin.Context)

	// Handle an HTTP POST request to create a new jupyter kernel.
	HandleCreateKernelRequest(*gin.Context)

	// Write an error back to the client.
	WriteError(*gin.Context, string)
}

type BackendHttpGRPCHandler interface {
	BackendHttpGetHandler

	// Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success.
	DialGatewayGRPC(string) error
}
