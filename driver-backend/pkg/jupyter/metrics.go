package jupyter

// MetricsConsumer defines an interface used by Jupyter components to publish Prometheus kernelMetricsManager.
type MetricsConsumer interface {
	// ObserveJupyterSessionCreationLatency records the latency of creating a Jupyter session
	// during the execution of a particular workload, as identified by the given workload ID.
	ObserveJupyterSessionCreationLatency(latencyMilliseconds int64, workloadId string)

	// ObserveJupyterSessionTerminationLatency records the latency of terminating a Jupyter session
	// during the execution of a particular workload, as identified by the given workload ID.
	ObserveJupyterSessionTerminationLatency(latencyMilliseconds int64, workloadId string)

	// ObserveJupyterExecuteRequestE2ELatency records the end-to-end latency of an "execute_request" message
	// during the execution of a particular workload, as identified by the given workload ID.
	ObserveJupyterExecuteRequestE2ELatency(latencyMilliseconds int64, workloadId string)

	// AddJupyterRequestExecuteTime records the time taken to process an "execute_request" for the total, aggregate,
	// cumulative time spent processing "execute_request" messages.
	AddJupyterRequestExecuteTime(latencyMilliseconds int64, kernelId string, workloadId string)
}
