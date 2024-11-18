package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

type ConfigHttpHandler struct {
	*BaseHandler
}

func NewConfigHttpHandler(opts *domain.Configuration, atom *zap.AtomicLevel) *ConfigHttpHandler {
	handler := &ConfigHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side ConfigHttpHandler.")

	return handler
}

func (h *ConfigHttpHandler) HandleRequest(c *gin.Context) {
	h.logger.Info("Sending config back to client now.", zap.Any("config", h.opts))
	c.JSON(http.StatusOK, h.opts)
}
