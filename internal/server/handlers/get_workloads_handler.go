package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

type GetWorkloadsHttpHandler struct {
	*BaseHandler

	workloadPresetsMap map[string]*domain.WorkloadPreset
	workloadPresets    []*domain.WorkloadPreset

	workloadsMap map[string]*domain.Workload // Map from workload ID to workload
	workloads    []*domain.Workload
}

func NewGetWorkloadsHttpHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	handler := &GetWorkloadsHttpHandler{
		BaseHandler:  newBaseHandler(opts),
		workloadsMap: make(map[string]*domain.Workload),
		workloads:    make([]*domain.Workload, 0, 2),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side GetWorkloadsHttpHandler.")

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
		handler.workloadPresetsMap[preset.Key] = preset
	}

	return handler
}

func (h *GetWorkloadsHttpHandler) HandleRequest(c *gin.Context) {
	// TODO(Ben): We'll have an explicit configuration file that defines the workload presets.
	// This will get parsed by the server, and the result of parsing that file is what will be returned.
	// For now, we'll just return the hard-coded defaults.
	h.sugaredLogger.Debugf("Returning %d workloads to user.", len(h.workloads))
	c.JSON(http.StatusOK, h.workloads)
}
