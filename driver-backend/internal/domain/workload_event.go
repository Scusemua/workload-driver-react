package domain

import (
	"encoding/json"
	"time"
)

type WorkloadEvent struct {
	Index                 int    `json:"idx"`                     // Index of the event relative to the workload in which the event occurred. The first event to occur has index 0.
	Id                    string `json:"id"`                      // Unique ID of the event.
	Name                  string `json:"name"`                    // The name of the event.
	Session               string `json:"session"`                 // The ID of the session targeted by the event.
	Timestamp             string `json:"timestamp"`               // The timestamp specified by the trace data/template/preset.
	ProcessedAt           string `json:"processed_at"`            // The real-world clocktime at which the event was processed.
	SimProcessedAt        string `json:"sim_processed_at"`        // The simulation clocktime at which the event was processed. May differ from the 'Timestamp' field if there were delays.
	ProcessedSuccessfully bool   `json:"processed_successfully"`  // True if the event was processed without error.
	ErrorMessage          string `json:"error_message,omitempty"` // Error message from the error that caused the event to not be processed successfully.
}

// NewEmptyWorkloadEvent returns an "empty" workload event -- with none of its fields populated.
// This is intended to be used with the WorkloadEvent::WithX functions, as
// another means of constructing WorkloadEvent structs.
//
// Note: WorkloadEvent::ProcessedSuccessfully field is initialized to true.
// It can be set to false explicitly via WorkloadEvent::WithProcessedStatus,
// by passing a non-nil error to WorkloadEvent::WithError,
// or by passing a non-empty string to WorkloadEvent::WithErrorMessage.
func NewEmptyWorkloadEvent() *WorkloadEvent {
	return &WorkloadEvent{
		ProcessedSuccessfully: true,
	}
}

func NewWorkloadEvent(idx int, id string, name string, session string, timestamp string, processedAt string, simulationProcessedAt string, processedSuccessfully bool, err error) *WorkloadEvent {
	event := &WorkloadEvent{
		Index:                 idx,
		Id:                    id,
		Name:                  name,
		Session:               session,
		Timestamp:             timestamp,
		ProcessedAt:           processedAt,
		ProcessedSuccessfully: processedSuccessfully,
		SimProcessedAt:        simulationProcessedAt,
	}

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return event
}

// WithIndex should be used with caution; the workload implementation should be the only entity that uses this function.
func (evt *WorkloadEvent) WithIndex(eventIndex int) *WorkloadEvent {
	evt.Index = eventIndex
	return evt
}

func (evt *WorkloadEvent) WithEventId(eventId string) *WorkloadEvent {
	evt.Id = eventId
	return evt
}

func (evt *WorkloadEvent) WithSessionId(sessionId string) *WorkloadEvent {
	evt.Session = sessionId
	return evt
}

func (evt *WorkloadEvent) WithEventName(name NamedEvent) *WorkloadEvent {
	evt.Name = name.String()
	return evt
}

func (evt *WorkloadEvent) WithEventNameString(name string) *WorkloadEvent {
	evt.Name = name
	return evt
}

func (evt *WorkloadEvent) WithEventTimestamp(eventTimestamp time.Time) *WorkloadEvent {
	evt.Timestamp = eventTimestamp.String()
	return evt
}

func (evt *WorkloadEvent) WithEventTimestampAsString(eventTimestamp string) *WorkloadEvent {
	evt.Timestamp = eventTimestamp
	return evt
}

func (evt *WorkloadEvent) WithProcessedAtTime(processedAt time.Time) *WorkloadEvent {
	evt.ProcessedAt = processedAt.String()
	return evt
}

func (evt *WorkloadEvent) WithProcessedAtTimeAsString(processedAt string) *WorkloadEvent {
	evt.ProcessedAt = processedAt
	return evt
}

func (evt *WorkloadEvent) WithSimProcessedAtTime(simulationProcessedAt time.Time) *WorkloadEvent {
	evt.SimProcessedAt = simulationProcessedAt.String()
	return evt
}

func (evt *WorkloadEvent) WithSimProcessedAtTimeAsString(simulationProcessedAt string) *WorkloadEvent {
	evt.SimProcessedAt = simulationProcessedAt
	return evt
}

func (evt *WorkloadEvent) WithProcessedStatus(success bool) *WorkloadEvent {
	evt.ProcessedSuccessfully = success
	return evt
}

// WithError conditionally sets the 'ErrorMessage' field of the WorkloadEvent struct if the error argument is non-nil.
//
// Note: If the event is non-nil, then this also updates the WorkloadEvent::ProcessedSuccessfully field, setting it to false.
// You can manually flip it back to true, if desired, by calling the WorkloadEvent::WithProcessedStatus method and passing true.
func (evt *WorkloadEvent) WithError(err error) *WorkloadEvent {
	if err != nil {
		evt.ErrorMessage = err.Error()
		evt.ProcessedSuccessfully = false
	}

	return evt
}

// WithErrorMessage sets the error message of the WorkloadEvent.
//
// Note: If the error message is non-empty (i.e., has length >= 1), then the ProcessedSuccessfully field is automatically set to false.
// You can manually flip it back to true, if desired, by calling the WorkloadEvent::WithProcessedStatus method and passing true.
func (evt *WorkloadEvent) WithErrorMessage(errorMessage string) *WorkloadEvent {
	evt.ErrorMessage = errorMessage

	if len(errorMessage) >= 1 {
		evt.ProcessedSuccessfully = false
	}

	return evt
}

func (evt *WorkloadEvent) String() string {
	out, err := json.Marshal(evt)
	if err != nil {
		panic(err)
	}

	return string(out)
}
