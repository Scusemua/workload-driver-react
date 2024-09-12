package metrics

import (
	"github.com/mattn/go-colorable"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	PrometheusMetricsWrapperInstance *PrometheusMetricsWrapper
)

func init() {
	atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	PrometheusMetricsWrapperInstance, _ = NewPrometheusMetricsWrapper(&atom)
}

// PrometheusMetricsWrapper is a simple wrapper around several Prometheus metrics.
type PrometheusMetricsWrapper struct {
	logger *zap.Logger

	WorkloadTrainingEventsCompleted *prometheus.CounterVec
	WorkloadEventsProcessed         *prometheus.CounterVec
	WorkloadTotalNumSessions        *prometheus.CounterVec

	WorkloadTrainingEventDuration  *prometheus.HistogramVec
	WorkloadSessionLifetimeSeconds *prometheus.HistogramVec

	// JupyterSessionCreationLatency is a metric tracking the latency between when
	// the network request to create a new Session is first sent and when the response
	// is received, indicating that the new Session has been created successfully.
	//
	// The latency is observed from the Golang-based Jupyter client, and the units
	// of the metric are seconds.
	JupyterSessionCreationLatency *prometheus.HistogramVec
	// JupyterSessionTerminationLatency is a metric tracking the latency between when
	// the HTTP request to terminate a Session is sent and when the response is received,
	// indicating that the Session has been successfully terminated.
	//
	// The latency is observed from the Golang-based Jupyter client, and the units
	// of the metric are seconds.
	JupyterSessionTerminationLatency *prometheus.HistogramVec

	// JupyterExecuteRequestEndToEndLatency is the end-to-end latency observed when sending
	// "execute_request" messages.
	//
	// The latency is observed from the Golang-based Jupyter client, and the units
	// of the metric are seconds.
	JupyterExecuteRequestEndToEndLatency *prometheus.HistogramVec

	WorkloadActiveTrainingSessions *prometheus.GaugeVec
	WorkloadActiveNumSessions      *prometheus.GaugeVec
}

// NewPrometheusMetricsWrapper creates a new PrometheusMetricsWrapper struct and returns a pointer to it.
// NewPrometheusMetricsWrapper initializes creates and registers all the metrics encapsulated by the
// PrometheusMetricsWrapper struct after creating the struct.
func NewPrometheusMetricsWrapper(atom *zap.AtomicLevel) (*PrometheusMetricsWrapper, []error) {
	// Counter metrics.
	metricsWrapper := &PrometheusMetricsWrapper{
		WorkloadTrainingEventsCompleted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "workload_training_events_completed_total",
		}, []string{"workload_id"}),
		WorkloadEventsProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "workload_events_processed_total",
		}, []string{"workload_id"}),
		WorkloadTotalNumSessions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "sessions_created_total",
		}, []string{"workload_id"}),

		// Histogram metrics.
		WorkloadTrainingEventDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "training_duration_seconds",
		}, []string{"workload_id", "session_id"}),
		WorkloadSessionLifetimeSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "session_lifetime_seconds",
		}, []string{"workload_id"}),

		JupyterSessionCreationLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "jupyter",
			Name:      "session_creation_latency_seconds",
		}, []string{"workload_id"}),
		JupyterSessionTerminationLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "jupyter",
			Name:      "session_termination_latency_seconds",
		}, []string{"workload_id"}),
		JupyterExecuteRequestEndToEndLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "jupyter",
			Name:      "execute_request_e2e_latency_seconds",
		}, []string{"workload_id"}),

		// Gauge metrics.
		WorkloadActiveNumSessions: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "active_sessions",
		}, []string{"workload_id"}),
		WorkloadActiveTrainingSessions: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "active_trainings",
		}, []string{"workload_id"}),
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	errs := make([]error, 0)

	if err := prometheus.Register(metricsWrapper.WorkloadTrainingEventsCompleted); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadTrainingEventsCompleted"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.WorkloadEventsProcessed); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadEventsProcessed"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.WorkloadTotalNumSessions); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadTotalNumSessions"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.WorkloadTrainingEventDuration); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadTrainingEventDuration"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.WorkloadSessionLifetimeSeconds); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadSessionLifetimeSeconds"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.JupyterSessionCreationLatency); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "JupyterSessionCreationLatency"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.JupyterExecuteRequestEndToEndLatency); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "JupyterExecuteRequestEndToEndLatency"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.WorkloadActiveNumSessions); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadActiveNumSessions"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.WorkloadActiveTrainingSessions); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadActiveTrainingSessions"), zap.Error(err))
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return metricsWrapper, errs
	} else {
		return metricsWrapper, nil
	}
}
