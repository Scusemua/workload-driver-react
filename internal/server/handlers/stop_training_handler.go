package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

type StopTrainingHandler struct {
	*BaseHandler
}

func NewStopTrainingHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	handler := &StopTrainingHandler{
		BaseHandler: newBaseHandler(opts),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side StopTrainingHandler.")

	return handler
}

func (h *StopTrainingHandler) HandleRequest(c *gin.Context) {

}
