package handlers

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

type BaseHandler struct {
	http.Handler

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	opts *domain.Configuration

	BackendHttpGetHandler domain.BackendHttpGetHandler
}

func newBaseHandler(opts *domain.Configuration, atom *zap.AtomicLevel) *BaseHandler {
	handler := &BaseHandler{
		opts: opts,
		atom: atom,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	handler.logger = zap.New(core, zap.Development())

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
