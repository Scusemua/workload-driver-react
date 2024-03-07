package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/domain"
	"go.uber.org/zap"
)

type BaseHandler struct {
	http.Handler

	logger *zap.Logger
	opts   *config.Configuration

	BackendHttpGetHandler domain.BackendHttpGetHandler
}

func newBaseHandler(opts *config.Configuration) *BaseHandler {
	handler := &BaseHandler{
		opts: opts,
	}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.BackendHttpGetHandler = handler

	return handler
}

func (h *BaseHandler) PrimaryHttpHandler() domain.BackendHttpGetHandler {
	return h.BackendHttpGetHandler
}

// Write an error back to the client.
func (h *BaseHandler) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	c.JSON(http.StatusInternalServerError, msg)
}

func (h *BaseHandler) HandleRequest(c *gin.Context) {
	h.BackendHttpGetHandler.HandleRequest(c)
}
