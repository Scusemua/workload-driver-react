package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

var (
	DefaultWorkloads = []*domain.WorkloadPreset{
		{
			Name:        "June - August",
			Description: "Workload based on trace data from June, July, and August.",
			Key:         "jun-aug",
			Months:      []string{"jun", "jul", "aug"},
		},
		{
			Name:        "July",
			Description: "Workload based on trace data from July.",
			Key:         "jul",
			Months:      []string{"jul"},
		},
		{
			Name:        "August",
			Description: "Workload based on trace data from August.",
			Key:         "aug",
			Months:      []string{"aug"},
		},
	}
)

type WorkloadPresetHttpHandler struct {
	*BaseHandler
}

func NewWorkloadPresetHttpHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	handler := &WorkloadPresetHttpHandler{
		BaseHandler: newBaseHandler(opts),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side WorkloadPresetHttpHandler.")

	return handler
}

func (h *WorkloadPresetHttpHandler) HandleRequest(c *gin.Context) {
	// TODO(Ben): We'll have an explicit configuration file that defines the workload presets.
	// This will get parsed by the server, and the result of parsing that file is what will be returned.
	// For now, we'll just return the hard-coded defaults.
	h.logger.Debug("Returning hard-coded default workload presets.")
	c.JSON(200, DefaultWorkloads)
}
