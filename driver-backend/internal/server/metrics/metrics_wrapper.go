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

	WorkloadTrainingEventDurationMilliseconds *prometheus.HistogramVec
	WorkloadSessionLifetimeSeconds            *prometheus.HistogramVec

	// SessionDelayedDueToResourceContention counts the number of times a Session is delayed due to resource
	// contention when attempting to create its container.
	SessionDelayedDueToResourceContention *prometheus.CounterVec

	// JupyterSessionCreationLatencyMilliseconds is a metric tracking the latency between when
	// the network request to create a new Session is first sent and when the response
	// is received, indicating that the new Session has been created successfully.
	//
	// The latency is observed from the Golang-based Jupyter client, and the units
	// of the metric are seconds.
	JupyterSessionCreationLatencyMilliseconds *prometheus.HistogramVec
	// JupyterSessionTerminationLatencyMilliseconds is a metric tracking the latency between when
	// the HTTP request to terminate a Session is sent and when the response is received,
	// indicating that the Session has been successfully terminated.
	//
	// The latency is observed from the Golang-based Jupyter client, and the units
	// of the metric are seconds.
	JupyterSessionTerminationLatencyMilliseconds *prometheus.HistogramVec

	// JupyterExecuteRequestEndToEndLatencyMilliseconds is the end-to-end latency observed when sending
	// "execute_request" messages.
	//
	// The latency is observed from the Golang-based Jupyter client, and the units
	// of the metric are seconds.
	JupyterExecuteRequestEndToEndLatencyMilliseconds *prometheus.HistogramVec

	// JupyterRequestExecuteTime is a gauge that tracks the time spent actively executing user-code.
	// This is from the perspective of Jupyter clients.
	JupyterRequestExecuteTime *prometheus.GaugeVec

	WorkloadActiveTrainingSessions *prometheus.GaugeVec
	WorkloadActiveNumSessions      *prometheus.GaugeVec

	// JupyterTimeSpentIdle is the amount of time that actively-provisioned Sessions spend not actually executing code.
	// This is from the perspective of Jupyter clients.
	//JupyterTimeSpentIdle *prometheus.GaugeVec
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

		SessionDelayedDueToResourceContention: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "session_delayed_resource_contention",
		}, []string{"workload_id", "session_id"}),

		// Histogram metrics.
		WorkloadTrainingEventDurationMilliseconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "training_duration_milliseconds",
			Buckets: []float64{10 /* 10 ms */, 1e3 /* 1 sec */, 5e3 /* 5 sec */, 10e3 /* 10 sec */, 20e3, /* 20 sec */
				30e3 /* 30 sec */, 60e3 /* 1 min */, 300e3 /* 5 min */, 600e3 /* 10 min */, 1.0e6 /* 30 min */, 3.6e6, /* 1hr */
				7.2e6 /* 2 hr */, 6e7 /* 1,000 min, or 16.66hr */, 6e8 /* 10,000, or 166.66hr */, 6e9 /* 100,000 min, or 1,666.66hr */},
		}, []string{"workload_id", "session_id"}),
		WorkloadSessionLifetimeSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "session_lifetime_seconds",
			Buckets: []float64{60 /* 1 min */, 600 /* 10 min */, 1800 /* 30 min */, 3600, /* 1hr */
				21600 /* 6 hr */, 43200 /* 12 hr */, 86400 /* 24 hr */, 259200 /* 72 hr */, 6.048e5, /* 1 week */
				1.21e6 /* 2 weeks */, 1.814e6 /* 3 weeks */, 1.051e7 /* 1 month */},
		}, []string{"workload_id"}),

		JupyterSessionCreationLatencyMilliseconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "jupyter",
			Name:      "session_creation_latency_milliseconds",
			Buckets:   []float64{1, 10, 30, 75, 150, 250, 500, 1000, 2000, 5000, 10e3, 20e3, 45e3, 90e3, 300e3},
		}, []string{"workload_id"}),
		JupyterSessionTerminationLatencyMilliseconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "jupyter",
			Name:      "session_termination_latency_milliseconds",
			Buckets:   []float64{1, 10, 30, 75, 150, 250, 500, 1000, 2000, 5000, 10e3, 20e3, 45e3, 90e3, 300e3},
		}, []string{"workload_id"}),
		JupyterExecuteRequestEndToEndLatencyMilliseconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "distributed_cluster",
			Subsystem: "jupyter",
			Name:      "execute_request_e2e_latency_milliseconds",
			Buckets: []float64{10 /* 10 ms */, 100, 250, 500, 750, 1e3 /* 1 sec */, 5e3 /* 5 sec */, 10e3 /* 10 sec */, 20e3, /* 20 sec */
				30e3 /* 30 sec */, 60e3 /* 1 min */, 300e3 /* 5 min */, 600e3 /* 10 min */, 1.0e6 /* 30 min */, 3.6e6, /* 1hr */
				7.2e6 /* 2 hr */, 6e7 /* 1,000 min, or 16.66hr */, 6e8 /* 10,000, or 166.66hr */, 6e9 /* 100,000 min, or 1,666.66hr */},
		}, []string{"workload_id"}),

		// Gauge metrics.
		WorkloadActiveNumSessions: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "active_workload_sessions",
			Help:      "Number of actively-running kernels",
		}, []string{"workload_id"}),
		WorkloadActiveTrainingSessions: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "distributed_cluster",
			Subsystem: "workload_driver",
			Name:      "active_trainings",
		}, []string{"workload_id"}),
		JupyterRequestExecuteTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "distributed_cluster",
			Subsystem: "jupyter",
			Name:      "execute_request_active_seconds",
			Help:      "The time, in seconds, that Jupyter clients spend waiting for an \"execute_reply\" response to their \"execute_request\" requests. Includes total training time and all overheads.",
		}, []string{"workload_id", "kernel_id"}),
		//JupyterTimeSpentIdle: prometheus.NewGaugeVec(prometheus.GaugeOpts{
		//	Namespace: "distributed_cluster",
		//	Subsystem: "jupyter",
		//	Name:      "active_trainings",
		//}, []string{"workload_id", "session_id"}),
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

	if err := prometheus.Register(metricsWrapper.WorkloadTrainingEventDurationMilliseconds); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadTrainingEventDurationMilliseconds"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.WorkloadSessionLifetimeSeconds); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "WorkloadSessionLifetimeSeconds"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.JupyterSessionCreationLatencyMilliseconds); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "JupyterSessionCreationLatencyMilliseconds"), zap.Error(err))
		errs = append(errs, err)
	}

	if err := prometheus.Register(metricsWrapper.JupyterExecuteRequestEndToEndLatencyMilliseconds); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "JupyterExecuteRequestEndToEndLatencyMilliseconds"), zap.Error(err))
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

	if err := prometheus.Register(metricsWrapper.JupyterRequestExecuteTime); err != nil {
		metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "JupyterRequestExecuteTime"), zap.Error(err))
		errs = append(errs, err)
	}

	//if err := prometheus.Register(metricsWrapper.JupyterTimeSpentIdle); err != nil {
	//	metricsWrapper.logger.Error("Failed to register Prometheus metric.", zap.String("metric", "JupyterTimeSpentIdle"), zap.Error(err))
	//	errs = append(errs, err)
	//}

	if len(errs) > 0 {
		return metricsWrapper, errs
	} else {
		return metricsWrapper, nil
	}
}

// ObserveJupyterSessionCreationLatency records the latency of creating a Jupyter session
// during the execution of a particular workload, as identified by the given workload ID.
func (m *PrometheusMetricsWrapper) ObserveJupyterSessionCreationLatency(latencyMilliseconds int64, workloadId string) {
	m.JupyterSessionCreationLatencyMilliseconds.
		With(prometheus.Labels{"workload_id": workloadId}).
		Observe(float64(latencyMilliseconds))
}

// ObserveJupyterSessionTerminationLatency records the latency of terminating a Jupyter session
// during the execution of a particular workload, as identified by the given workload ID.
func (m *PrometheusMetricsWrapper) ObserveJupyterSessionTerminationLatency(latencyMilliseconds int64, workloadId string) {
	m.JupyterSessionTerminationLatencyMilliseconds.
		With(prometheus.Labels{"workload_id": workloadId}).
		Observe(float64(latencyMilliseconds))
}

// ObserveJupyterExecuteRequestE2ELatency records the end-to-end latency of an "execute_request" message
// during the execution of a particular workload, as identified by the given workload ID.
func (m *PrometheusMetricsWrapper) ObserveJupyterExecuteRequestE2ELatency(latencyMilliseconds int64, workloadId string) {
	m.JupyterExecuteRequestEndToEndLatencyMilliseconds.
		With(prometheus.Labels{"workload_id": workloadId}).
		Observe(float64(latencyMilliseconds))

}

// AddJupyterRequestExecuteTime records the time taken to process an "execute_request" for the total, aggregate,
// cumulative time spent processing "execute_request" messages.
func (m *PrometheusMetricsWrapper) AddJupyterRequestExecuteTime(latencyMilliseconds int64, kernelId string, workloadId string) {
	m.JupyterRequestExecuteTime.
		With(prometheus.Labels{"workload_id": workloadId, "kernel_id": kernelId}).
		Add(float64(latencyMilliseconds)) // Add another second.
}
