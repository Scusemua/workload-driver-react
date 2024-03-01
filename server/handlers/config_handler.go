package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/server/domain"
	"go.uber.org/zap"
)

type ConfigHttpHandler struct {
	*BaseHandler
}

func NewConfigHttpHandler(opts *config.Configuration) domain.BackendHttpHandler {
	handler := &ConfigHttpHandler{
		BaseHandler: newBaseHandler(opts),
	}
	handler.BackendHttpHandler = handler

	handler.logger.Info(fmt.Sprintf("Creating server-side ConfigHttpHandler.\nOptions: %s", opts))

	return handler
}

func (h *ConfigHttpHandler) HandleRequest(c *gin.Context) {
	h.logger.Info("Sending config back to client now.", zap.Any("config", h.opts))
	c.JSON(http.StatusOK, h.opts)
}
