package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/server/config"
	"go.uber.org/zap"
)

type ConfigHttpHandler struct {
	*BaseHandler
}

func NewConfigHttpHandler(opts *config.Configuration) *ConfigHttpHandler {
	handler := &ConfigHttpHandler{
		BaseHandler: NewBaseHandler(opts),
	}
	handler.BackendHttpHandler = handler

	handler.Logger.Info(fmt.Sprintf("Creating server-side ConfigHttpHandler.\nOptions: %s", opts))

	return handler
}

func (h *ConfigHttpHandler) HandleRequest(c *gin.Context) {
	h.Logger.Info("Sending config back to client now.", zap.Any("config", h.opts))
	c.JSON(http.StatusOK, h.opts)
}
