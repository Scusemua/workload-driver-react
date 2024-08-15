package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

var (
	DefaultWorkloads = []*domain.CsvWorkloadPreset{
		{
			BaseWorkloadPreset: domain.BaseWorkloadPreset{
				Name:        "June - August",
				Description: "Workload based on trace data from June, July, and August.",
				Key:         "jun-aug",
			},
			Months: []string{"jun", "jul", "aug"},
		},
		{
			BaseWorkloadPreset: domain.BaseWorkloadPreset{
				Name:        "July",
				Description: "Workload based on trace data from July.",
				Key:         "jul",
			},
			Months: []string{"jul"},
		},
		{
			BaseWorkloadPreset: domain.BaseWorkloadPreset{
				Name:        "August",
				Description: "Workload based on trace data from August.",
				Key:         "aug",
			},
			Months: []string{"aug"},
		},
	}
)

type WorkloadPresetHttpHandler struct {
	*BaseHandler

	workloadPresetsMap map[string]*domain.WorkloadPreset
	workloadPresets    []*domain.WorkloadPreset
}

func NewWorkloadPresetHttpHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	handler := &WorkloadPresetHttpHandler{
		BaseHandler: newBaseHandler(opts),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side WorkloadPresetHttpHandler.")

	// Load the list of workload presets from the specified file.
	handler.logger.Debug("Loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath))
	presets, err := domain.LoadWorkloadPresetsFromFile(opts.WorkloadPresetsFilepath)
	if err != nil {
		handler.logger.Error("Error encountered while loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath), zap.Error(err))
		panic(err)
	}

	handler.workloadPresets = presets
	handler.workloadPresetsMap = make(map[string]*domain.WorkloadPreset, len(presets))
	for _, preset := range presets {
		handler.workloadPresetsMap[preset.GetKey()] = preset

		handler.logger.Debug("Discovered workload preset.", zap.Any(fmt.Sprintf("preset-%s", preset.GetKey()), preset.String()))
	}

	return handler
}

func (h *WorkloadPresetHttpHandler) HandleRequest(c *gin.Context) {
	// TODO(Ben): We'll have an explicit configuration file that defines the workload presets.
	// This will get parsed by the server, and the result of parsing that file is what will be returned.
	// For now, we'll just return the hard-coded defaults.
	h.sugaredLogger.Debugf("Returning %d workload preset(s) to user.", len(h.workloadPresets))
	c.JSON(http.StatusOK, h.workloadPresets)
}
