package domain

import (
	"encoding/json"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
)

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

// UnmarshalRequestPayload -- given a payload that encodes an arbitrary (i.e., of any type) workload-related WebSocket
// message, unmarshal and return the message.
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

type PatchedWorkload struct {
	Patch      string `json:"patch"`
	WorkloadId string `json:"workloadId"`
}

type WorkloadResponse struct {
	// MessageIndex      int32      `json:"message_index"`
	Operation         string             `json:"op"`                 // The operation of the original request.
	Status            string             `json:"status"`             // OK or ERROR.
	MessageId         string             `json:"msg_id"`             // Unique ID of the message.
	NewWorkloads      []Workload         `json:"new_workloads"`      // Workloads that are newly-created.
	ModifiedWorkloads []Workload         `json:"modified_workloads"` // Modified workloads sent in their entirety.
	PatchedWorkloads  []*PatchedWorkload `json:"patched_workloads"`  // Modified workloads sent as JSON merge patches.
	DeletedWorkloads  []Workload         `json:"deleted_workloads"`  // Workloads that are being deleted.
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

// WorkloadRegistrationRequestWrapper is a wrapper around a WorkloadRegistrationRequest.
// WorkloadRegistrationRequestWrapper contains the message ID and operation field.
type WorkloadRegistrationRequestWrapper struct {
	*BaseMessage
	WorkloadRegistrationRequest *WorkloadRegistrationRequest `json:"workload_registration_request"`
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

// PauseUnpauseWorkloadRequest is a request for pausing and un-pausing a workload.
// PauseUnpauseWorkloadRequest this pauses or unpauses a workload depends on the value of the Operation field.
type PauseUnpauseWorkloadRequest struct {
	*BaseMessage
	WorkloadId string `json:"workload_id"` // ID of the workload to (un)pause.
}

// GetSpecificWorkloadRequest is used to request the latest version of a specific workload.
type GetSpecificWorkloadRequest struct {
	*BaseMessage
	WorkloadId string `json:"workload_id"`
}

func (r *GetSpecificWorkloadRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

// StartStopWorkloadRequest is a request for starting/stopping a workload. Whether this starts or stops a
// workload depends on the value of the Operation field.
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

// StartStopWorkloadsRequest is a request for starting/stopping a workload.
// Whether this starts or stops a workload depends on the value of the Operation field.
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
	// By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'.
	// This will lead to many sessions reserving fewer GPUs than when this property is disabled (default).
	AdjustGpuReservations     bool                           `name:"adjust_gpu_reservations" json:"adjust_gpu_reservations" description:"By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'. This will lead to many sessions reserving fewer GPUs than when this property is disabled (default)."`
	WorkloadName              string                         `name:"name" json:"name" yaml:"name" description:"Non-unique identifier of the workload created/specified by the user when launching the workload."`
	DebugLogging              bool                           `name:"debug_logging" json:"debug_logging" yaml:"debug_logging" description:"Flag indicating whether debug-level logging should be enabled."`
	Sessions                  []*WorkloadTemplateSession     `name:"sessions" json:"sessions" yaml:"sessions" description:"The sessions defined by the template. These are used to construct the workload."`
	TemplateFilePath          string                         `name:"template_file_path" json:"template_file_path" yaml:"template_file_path" description:"Path to .JSON file containing a large template that comes pre-loaded on the server."`
	Type                      string                         `name:"type" json:"type"`
	Key                       string                         `name:"key" yaml:"key" json:"key" description:"Key for code-use only (i.e., we don't intend to display this to the user for the most part)."` // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	Seed                      int64                          `name:"seed" yaml:"seed" json:"seed" description:"RNG seed for the workload."`
	TimescaleAdjustmentFactor float64                        `name:"timescale_adjustment_factor" json:"timescale_adjustment_factor" description:"Adjusts how long ticks are simulated for."`
	RemoteStorageDefinition   *proto.RemoteStorageDefinition `name:"remote_storage_definition" json:"remote_storage_definition" yaml:"remote_storage_definition" mapstructure:"remote_storage_definition" description:"Defines a simulated remote storage to be used during the workload."`

	// SessionsSamplePercentage is the percent of sessions from a CSV workload for which we'll actually process events.
	//
	// If SessionsSamplePercentage is set to 1.0, then all sessions will be processed.
	//
	// SessionsSamplePercentage must be > 0.
	SessionsSamplePercentage float64 `name:"sessions_sample_percentage" json:"sessions_sample_percentage" yaml:"sessions_sample_percentage"`
}

func (r *WorkloadRegistrationRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}
