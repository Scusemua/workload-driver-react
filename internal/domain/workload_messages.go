package domain

import "encoding/json"

type WorkloadWebsocketMessage interface {
	GetOperation() string
	GetMessageId() string
}

type BaseMessage struct {
	Operation string `json:"op"`
	MessageId string `json:"msg_id"`
}

func (m *BaseMessage) GetOperation() string {
	return m.Operation
}

func (m *BaseMessage) GetMessageId() string {
	return m.MessageId
}

type SubscriptionRequest struct {
	*BaseMessage
}

// Given a payload that encodes an arbitrary (i.e., of any type) workload-related WebSocket message, unmarshal and return the message.
func UnmarshalRequestPayload[WorkloadWebsocketMessageType WorkloadWebsocketMessage](encodedMessage []byte) (WorkloadWebsocketMessageType, error) {
	var decodedMessage WorkloadWebsocketMessageType
	err := json.Unmarshal(encodedMessage, &decodedMessage)

	return decodedMessage, err
}

func (r *SubscriptionRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type ToggleDebugLogsRequest struct {
	*BaseMessage
	WorkloadId string `json:"workload_id"`
	Enabled    bool   `json:"enabled"`
}

func (r *ToggleDebugLogsRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type WorkloadResponse struct {
	// MessageIndex      int32      `json:"message_index"`
	MessageId         string     `json:"msg_id"`
	NewWorkloads      []Workload `json:"new_workloads"`
	ModifiedWorkloads []Workload `json:"modified_workloads"`
	DeletedWorkloads  []Workload `json:"deleted_workloads"`
}

// Encode the response to a JSON format.
func (r *WorkloadResponse) Encode() ([]byte, error) {
	return json.Marshal(r)
}

func (r *WorkloadResponse) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

// Wrapper around a WorkloadRegistrationRequest; contains the message ID and operation field.
type WorkloadRegistrationRequestWrapper struct {
	*BaseMessage
	WorkloadRegistrationRequest *WorkloadRegistrationRequest `json:"workloadRegistrationRequest"`
}

func (r *WorkloadRegistrationRequestWrapper) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type StopTrainingRequest struct {
	KernelId  string `json:"kernel_id"`  // The ID of the kernel to target (i.e., to stop training).
	SessionId string `json:"session_id"` // The associated session.
}

// Whether this pauses or unpauses a workload depends on the value of the Operation field.
type PauseUnpauseWorkloadRequest struct {
	*BaseMessage
	WorkloadId string `json:"workload_id"` // ID of the workload to (un)pause.
}

// Request for starting/stopping a workload. Whether this starts or stops a workload depends on the value of the Operation field.
type StartStopWorkloadRequest struct {
	*BaseMessage
	WorkloadId string `json:"workload_id"`
}

func (r *StartStopWorkloadRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

// Request for starting/stopping a workload. Whether this starts or stops a workload depends on the value of the Operation field.
type StartStopWorkloadsRequest struct {
	*BaseMessage
	WorkloadIDs []string `json:"workload_ids"`
}

func (r *StartStopWorkloadsRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type WorkloadRegistrationRequest struct {
	*BaseMessage

	// By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'.
	// This will lead to many sessions reserving fewer GPUs than when this property is disabled (default).
	AdjustGpuReservations     bool              `name:"adjust_gpu_reservations" json:"adjust_gpu_reservations" description:"By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'. This will lead to many sessions reserving fewer GPUs than when this property is disabled (default)."`
	WorkloadName              string            `name:"name" json:"name" yaml:"name" description:"Non-unique identifier of the workload created/specified by the user when launching the workload."`
	DebugLogging              bool              `name:"debug_logging" json:"debug_logging" yaml:"debug_logging" description:"Flag indicating whether debug-level logging should be enabled."`
	Template                  *WorkloadTemplate `json:"workload_template"` // Will be nil if it is a preset-based workload, rather than a template-based workload.
	Type                      string            `name:"type" json:"type"`
	Key                       string            `name:"key" yaml:"key" json:"key" description:"Key for code-use only (i.e., we don't intend to display this to the user for the most part)."` // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	Seed                      int64             `name:"seed" yaml:"seed" json:"seed" description:"RNG seed for the workload."`
	TimescaleAdjustmentFactor float64           `name:"timescale_adjustment_factor" json:"timescale_adjustment_factor" description:"Adjusts how long ticks are simulated for."`
}

func (r *WorkloadRegistrationRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}
