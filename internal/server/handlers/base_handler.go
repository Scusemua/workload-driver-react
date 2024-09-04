package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

type BaseHandler struct {
	http.Handler

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	opts          *domain.Configuration

	BackendHttpGetHandler domain.BackendHttpGetHandler
}

func newBaseHandler(opts *domain.Configuration) *BaseHandler {
	handler := &BaseHandler{
		opts: opts,
	}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.sugaredLogger = handler.logger.Sugar()

	handler.BackendHttpGetHandler = handler

	return handler
}

func (h *BaseHandler) PrimaryHttpHandler() domain.BackendHttpGetHandler {
	return h.BackendHttpGetHandler
}

// WriteError writes an error back to the client.
func (h *BaseHandler) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not handle request: %s", errorMessage))
}

func (h *BaseHandler) HandleRequest(c *gin.Context) {
	h.BackendHttpGetHandler.HandleRequest(c)
}
