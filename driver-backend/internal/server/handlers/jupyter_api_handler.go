package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

const (
	// Jupyter Server HTTP API endpoint for retrieving the list of kernel specs.
	kernelSpecJupyterServerEndpoint = "/api/kernelspecs"
)

type JupyterAPIHandler struct {
	jupyterServerAddress string // IP of the Jupyter Server.
	//spoofKernelSpecs     bool   // Determines whether we return real or fake data.

	logger *zap.Logger
}

func NewJupyterAPIHandler(opts *domain.Configuration) domain.JupyterApiHttpHandler {
	handler := &JupyterAPIHandler{
		jupyterServerAddress: opts.JupyterServerAddress,
		// spoofKernelSpecs:     opts.SpoofKernelSpecs,
	}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.logger.Info("Creating server-side JupyterAPIHandler.")

	return handler
}

// WriteError writes an error back to the client.
func (h *JupyterAPIHandler) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	c.JSON(http.StatusInternalServerError, msg)
}

func (h *JupyterAPIHandler) issueHttpRequest(target string) ([]byte, error) {
	resp, err := http.Get(target)
	if err != nil {
		h.logger.Error("Failed to complete HTTP GET request.", zap.String("error-message", err.Error()), zap.String("URL", target))
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Failed to read response from HTTP GET request.", zap.Error(err), zap.String("URL", target))
		return nil, err
	}

	return body, nil
}

func (h *JupyterAPIHandler) doSpoofKernelSpecs() []*domain.KernelSpec {
	// Distributed kernel.
	distributedKernel := &domain.KernelSpec{
		Name:          "distributed",
		DisplayName:   "Distributed Python3",
		Language:      "python3",
		InterruptMode: "signal",
		ArgV:          []string{"/opt/conda/bin/python3", "-m", "distributed_notebook.kernel", "-f", "{connection_file}", "--debug", "--IPKernelApp.outstream_class=distributed_notebook.kernel.iostream.OutStream"},
		KernelProvisioner: &domain.KernelProvisioner{
			Name:    "gateway-provisioner",
			Gateway: "gateway:8080",
		},
	}

	// Standard Python3 kernel.
	python3Kernel := &domain.KernelSpec{
		Name:          "python3",
		DisplayName:   "Python 3 (ipykernel)",
		Language:      "python",
		InterruptMode: "signal",
		ArgV:          []string{"N/A"},
	}

	// Made-up kernel.
	aiKernel := &domain.KernelSpec{
		Name:          "ai-kernel",
		DisplayName:   "AI-Powered Kernel",
		Language:      "all of them",
		InterruptMode: "impossible",
		ArgV:          []string{"N/A"},
	}

	return []*domain.KernelSpec{distributedKernel, python3Kernel, aiKernel}
}

// Retrieve the kernel specs by issuing an HTTP request to the Jupyter Server.
func (h *JupyterAPIHandler) getKernelSpecsFromJupyter() []*domain.KernelSpec {
	target := h.jupyterServerAddress + kernelSpecJupyterServerEndpoint

	body, err := h.issueHttpRequest(target)
	if err != nil {
		return nil
	}

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil
	}

	// TODO(Ben): Handle errors here gracefully.
	kernelSpecsJson := response["kernelspecs"].(map[string]interface{})
	h.logger.Debug(fmt.Sprintf("Retrieved %d kernel spec(s) from Jupyter Server.", len(kernelSpecsJson)), zap.Any("kernel-specs", kernelSpecsJson))

	kernelSpecs := make([]*domain.KernelSpec, 0, len(kernelSpecsJson))

	for specName, spec := range kernelSpecsJson {
		// TODO(Ben): Handle errors here gracefully.
		var specDefinition map[string]interface{} = spec.(map[string]interface{})["spec"].(map[string]interface{})

		kernelSpec := &domain.KernelSpec{
			Name:          specName,
			DisplayName:   specDefinition["display_name"].(string),
			Language:      specDefinition["language"].(string),
			InterruptMode: specDefinition["interrupt_mode"].(string),
		}

		var specMetadata map[string]interface{} = specDefinition["metadata"].(map[string]interface{})

		if val, ok := specMetadata["kernel_provisioner"]; ok {
			// TODO(Ben): Handle errors here gracefully.
			var kernelProvisioner map[string]interface{} = val.(map[string]interface{})

			kernelSpec.KernelProvisioner = &domain.KernelProvisioner{
				// TODO(Ben): Handle errors here gracefully.
				Name: kernelProvisioner["provisioner_name"].(string),
				// TODO(Ben): Handle errors here gracefully.
				Gateway: kernelProvisioner["config"].(map[string]interface{})["gateway"].(string),
				Valid:   true,
			}
		} else {
			kernelSpec.KernelProvisioner = &domain.KernelProvisioner{
				Name:    "",
				Gateway: "",
				Valid:   false,
			}
		}

		kernelSpecs = append(kernelSpecs, kernelSpec)
	}

	return kernelSpecs
}

// HandleCreateKernelRequest handles an HTTP POST request to create a new jupyter kernel.
func (h *JupyterAPIHandler) HandleCreateKernelRequest(*gin.Context) {

}

// HandleGetKernelSpecRequest handles an HTTP GET request to get the jupyter kernel specs.
func (h *JupyterAPIHandler) HandleGetKernelSpecRequest(c *gin.Context) {
	var kernelSpecs []*domain.KernelSpec

	// If we're spoofing the cluster, then just return some made up kernel specs for testing/debugging purposes.
	//if h.spoofKernelSpecs {
	//	h.logger.Info("Spoofing Jupyter kernel specs now.")
	//	kernelSpecs = h.doSpoofKernelSpecs()
	//} else {
	//	h.logger.Info("Retrieving Jupyter kernel specs from the Jupyter Server now.", zap.String("jupyter-server-ip", h.jupyterServerAddress))
	//	kernelSpecs = h.getKernelSpecsFromJupyter()
	//
	//	if kernelSpecs == nil {
	//		// Write error back to front-end.
	//		h.logger.Error("Failed to retrieve list of kernel specs from Jupyter Server.")
	//		h.WriteError(c, "Failed to retrieve list of kernel specs from Jupyter Server.")
	//		return
	//	}
	//}

	h.logger.Info("Sending kernel specs back to client now.", zap.Any("kernel-specs", kernelSpecs))
	c.JSON(http.StatusOK, kernelSpecs)
}
