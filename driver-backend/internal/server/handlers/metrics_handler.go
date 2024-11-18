package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"go.uber.org/zap"
	"net/http"
)

type MetricsHttpHandler struct {
	*BaseHandler
}

// metricsRequest encapsulates an HTTP request sent by the frontend to share/post/upload a Prometheus metric.
type metricsRequest struct {
	Name     string                 `json:"name"`
	Value    float64                `json:"value"`
	Metadata map[string]interface{} `json:"metadata"`
}

func NewMetricsHttpHandler(opts *domain.Configuration, atom *zap.AtomicLevel) *MetricsHttpHandler {
	handler := &MetricsHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Successfully created server-side MetricsHttpHandler handler.")

	return handler
}

func (h *MetricsHttpHandler) HandleRequest(c *gin.Context) {
	c.Status(http.StatusNotFound)
}

func (h *MetricsHttpHandler) HandlePatchRequest(c *gin.Context) {
	var req *metricsRequest
	if err := c.BindJSON(&req); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	switch req.Name {
	case "distributed_cluster_jupyter_session_creation_latency_seconds":
		{
			metrics.PrometheusMetricsWrapperInstance.JupyterSessionCreationLatencyMilliseconds.
				With(prometheus.Labels{"workload_id": "no_workload"}).
				Observe(req.Value)
			break
		}
	case "distributed_cluster_jupyter_execute_request_e2e_latency_seconds":
		{
			metrics.PrometheusMetricsWrapperInstance.JupyterExecuteRequestEndToEndLatencyMilliseconds.
				With(prometheus.Labels{"workload_id": "no_workload"}).
				Observe(req.Value)
			break
		}
	case "distributed_cluster_jupyter_session_termination_latency_seconds":
		{
			metrics.PrometheusMetricsWrapperInstance.JupyterSessionTerminationLatencyMilliseconds.
				With(prometheus.Labels{"workload_id": "no_workload"}).
				Observe(req.Value)
			break
		}
	default:
		{
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unknown or unsupported Prometheus metric: \"%s\"", req.Name))
			return
		}
	}

	h.logger.Debug("Metric posted.",
		zap.String("metric_name", req.Name),
		zap.Float64("metric_value", req.Value))

	c.Status(http.StatusOK)
}
