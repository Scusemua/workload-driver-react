package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/server/domain"
	"go.uber.org/zap"
)

type BaseHandler struct {
	http.Handler

	Logger *zap.Logger
	opts   *config.Configuration

	BackendHttpHandler domain.BackendHttpHandler
}

func NewBaseHandler(opts *config.Configuration) *BaseHandler {
	handler := &BaseHandler{
		opts: opts,
	}

	var err error
	handler.Logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.BackendHttpHandler = handler

	return handler
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
	h.BackendHttpHandler.HandleRequest(c)
}
