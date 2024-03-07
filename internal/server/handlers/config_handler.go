package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/domain"
	"go.uber.org/zap"
)

type ConfigHttpHandler struct {
	*BaseHandler
}

func NewConfigHttpHandler(opts *config.Configuration) domain.BackendHttpGetHandler {
	handler := &ConfigHttpHandler{
		BaseHandler: newBaseHandler(opts),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info(fmt.Sprintf("Creating server-side ConfigHttpHandler.\nOptions: %s", opts))

	return handler
}

func (h *ConfigHttpHandler) HandleRequest(c *gin.Context) {
	h.logger.Info("Sending config back to client now.", zap.Any("config", h.opts))
	c.JSON(http.StatusOK, h.opts)
}