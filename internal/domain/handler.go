package domain

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type BackendHttpGetHandler interface {
	// Write an error back to the client.
	WriteError(*gin.Context, string)

	// Handle a message/request from the front-end.
	HandleRequest(*gin.Context)

	// Return the request handler responsible for handling a majority of requests.
	PrimaryHttpHandler() BackendHttpGetHandler
}

type BackendHttpGetPatchHandler interface {
	BackendHttpGetHandler

	// Handle a message/request from the front-end.
	HandlePatchRequest(*gin.Context)
}

type BackendHttpPostHandler interface {
	BackendHttpGetHandler
}

type EnableDisableNodeRequest struct {
	NodeName string `json:"node_name"`
	Enable   bool   `json:"enable"` // If true, enable the node. Otherwise, disable the node.
}

func (r *EnableDisableNodeRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type JupyterApiHttpHandler interface {
	// Handle an HTTP GET request to get the jupyter kernel specs.
	HandleGetKernelSpecRequest(*gin.Context)

	// Handle an HTTP POST request to create a new jupyter kernel.
	HandleCreateKernelRequest(*gin.Context)

	// Write an error back to the client.
	WriteError(*gin.Context, string)
}
