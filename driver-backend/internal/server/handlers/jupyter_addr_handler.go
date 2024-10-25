package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"net/http"
)

type JupyterAddressHttpHandler struct {
	*BaseHandler

	frontendJupyterAddress string
}

func NewJupyterAddressHttpHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	handler := &JupyterAddressHttpHandler{
		BaseHandler:            newBaseHandler(opts),
		frontendJupyterAddress: opts.FrontendJupyterServerAddress,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Created server-side JupyterAddressHttpHandler.")

	return handler
}

func (h *JupyterAddressHttpHandler) HandleRequest(c *gin.Context) {
	response := make(map[string]interface{})
	response["jupyter_address"] = h.frontendJupyterAddress
	c.JSON(http.StatusOK, response)
}
