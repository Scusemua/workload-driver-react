package domain

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type BackendHttpGetHandler interface {
	// WriteError writes an error back to the client.
	WriteError(*gin.Context, string)

	// HandleRequest handles a message/request from the front-end.
	HandleRequest(*gin.Context)

	// PrimaryHttpHandler returns the request handler responsible for handling a majority of requests.
	PrimaryHttpHandler() BackendHttpGetHandler
}

type BackendHttpGetPatchHandler interface {
	BackendHttpGetHandler

	// HandlePatchRequest handles a message/request from the front-end.
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
	// HandleGetKernelSpecRequest handles an HTTP GET request to get the jupyter kernel specs.
	HandleGetKernelSpecRequest(*gin.Context)

	// HandleCreateKernelRequest handles an HTTP POST request to create a new jupyter kernel.
	HandleCreateKernelRequest(*gin.Context)

	// WriteError writes an error back to the client.
	WriteError(*gin.Context, string)
}
