package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

type WorkloadPresetHttpHandler struct {
	*BaseHandler

	workloadPresetsMap map[string]*domain.WorkloadPreset
	workloadPresets    []*domain.WorkloadPreset
}

func NewWorkloadPresetHttpHandler(opts *domain.Configuration, atom *zap.AtomicLevel) *WorkloadPresetHttpHandler {
	handler := &WorkloadPresetHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side WorkloadPresetHttpHandler.")

	// Load the list of workload presets from the specified file.
	handler.logger.Debug("Loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath))
	presets, err := domain.LoadWorkloadPresetsFromFile(opts.WorkloadPresetsFilepath)
	if err != nil {
		handler.logger.Error("Error encountered while loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath), zap.Error(err))
		presets = make([]*domain.WorkloadPreset, 0)
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
	h.sugaredLogger.Debugf("Returning %d workload preset(s) to user.", len(h.workloadPresets))
	c.JSON(http.StatusOK, h.workloadPresets)
}
