package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/workload"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type WorkloadTemplateHttpHandler struct {
	*BaseHandler

	WorkloadTemplatesMap map[string]*workload.PreloadedWorkloadTemplate
	WorkloadTemplates    []*workload.PreloadedWorkloadTemplate
}

// loadWorkloadTemplatesFromFile reads a yaml file containing one or more domain.PreloadedWorkloadTemplate definitions.
func loadWorkloadTemplatesFromFile(filepath string) ([]*workload.PreloadedWorkloadTemplate, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open or read workload templates file: %v\n", err)
		return nil, err
	}

	workloadTemplates := make([]*workload.PreloadedWorkloadTemplate, 0)
	err = yaml.Unmarshal(file, &workloadTemplates)

	if err != nil {
		fmt.Printf("[ERROR] Failed to unmarshal workload templates: %v\n", err)
		return nil, err
	}

	return workloadTemplates, nil
}

func NewWorkloadTemplateHttpHandler(opts *domain.Configuration, atom *zap.AtomicLevel) *WorkloadTemplateHttpHandler {
	handler := &WorkloadTemplateHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side WorkloadTemplateHttpHandler.")

	// Load the list of workload templates from the specified file.
	handler.logger.Debug("Loading workload templates from file now.", zap.String("filepath", opts.WorkloadTemplatesFilepath))
	templates, err := loadWorkloadTemplatesFromFile(opts.WorkloadTemplatesFilepath)
	if err != nil {
		handler.logger.Error("Error encountered while loading workload templates from file now.", zap.String("filepath", opts.WorkloadTemplatesFilepath), zap.Error(err))
		templates = make([]*workload.PreloadedWorkloadTemplate, 0)
	}

	handler.WorkloadTemplates = templates
	handler.WorkloadTemplatesMap = make(map[string]*workload.PreloadedWorkloadTemplate, len(templates))
	for _, template := range templates {
		handler.WorkloadTemplatesMap[template.Key] = template

		handler.logger.Debug("Discovered workload template.",
			zap.String(fmt.Sprintf("template-%s", template.Key), template.String()))
	}

	return handler
}

func (h *WorkloadTemplateHttpHandler) HandleRequest(c *gin.Context) {
	// Check if any query parameters exist
	if len(c.Request.URL.Query()) == 0 {
		h.logger.Debug("Returning workload template(s) to user.", zap.Int("num_templates", len(h.WorkloadTemplates)))
		c.JSON(http.StatusOK, h.WorkloadTemplates)
		return
	}

	requestedTemplate := c.Query("template")
	if requestedTemplate == "" {
		h.logger.Debug("Returning workload template(s) to user.", zap.Int("num_templates", len(h.WorkloadTemplates)))
		c.JSON(http.StatusOK, h.WorkloadTemplates)
		return
	}

	preloadedTemplate, ok := h.WorkloadTemplatesMap[requestedTemplate]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid workload template specified: \"%s\"", requestedTemplate),
		})
		return
	}

	if preloadedTemplate.IsLarge {
		response := make(map[string]interface{})
		response["preloaded_template"] = preloadedTemplate
		c.JSON(http.StatusOK, response)
		return
	}

	// Open the file
	templateFile, err := os.Open(preloadedTemplate.Filepath)
	if err != nil {
		h.logger.Error("Failed to read preloaded workload template from file.",
			zap.String("template_filepath", preloadedTemplate.Filepath),
			zap.String("template_display_name", preloadedTemplate.DisplayName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Errorf("failed to load requested template: %w", err),
		})
		return
	}
	defer templateFile.Close()

	// Create a JSON decoder
	decoder := json.NewDecoder(templateFile)

	// Decode JSON into the struct
	var loadedTemplate *domain.WorkloadRegistrationRequest
	err = decoder.Decode(&loadedTemplate)
	if err != nil {
		h.logger.Error("Failed to decode preloaded workload template.",
			zap.String("template_filepath", preloadedTemplate.Filepath),
			zap.String("template_display_name", preloadedTemplate.DisplayName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Errorf("failed to decode requested template: %w", err),
		})
		return
	}

	response := make(map[string]interface{})
	response["template"] = loadedTemplate
	c.JSON(http.StatusOK, response)
}
