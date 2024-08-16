package domain

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	WorkloadReady      WorkloadState = iota // workloadImpl is registered and ready to be started.
	WorkloadRunning    WorkloadState = 1    // workloadImpl is actively running/in-progress.
	WorkloadFinished   WorkloadState = 2    // workloadImpl stopped naturally/successfully after processing all events.
	WorkloadErred      WorkloadState = 3    // workloadImpl stopped due to an error.
	WorkloadTerminated WorkloadState = 4    // workloadImpl stopped because it was explicitly terminated early/premature.

	CsvWorkloadPresetType WorkloadPresetType = "CSV"
	XmlWorkloadPresetType WorkloadPresetType = "XML"

	UnspecifiedWorkload WorkloadType = "UnspecifiedWorkloadType" // Default value, before it is set.
	PresetWorkload      WorkloadType = "WorkloadFromPreset"
	TemplateWorkload    WorkloadType = "WorkloadFromTemplate"
	TraceWorkload       WorkloadType = "WorkloadFromTrace"
)

type WorkloadGenerator interface {
	GeneratePresetWorkload(EventConsumer, Workload, *WorkloadPreset, *WorkloadRegistrationRequest) error     // Start generating the workload.
	GenerateTemplateWorkload(EventConsumer, Workload, *WorkloadTemplate, *WorkloadRegistrationRequest) error // Start generating the workload.
	StopGeneratingWorkload()                                                                                 // Stop generating the workload prematurely.
}

type BaseMessage struct {
	Operation string `json:"op"`
	MessageId string `json:"msg_id"`
}

type SubscriptionRequest struct {
	*BaseMessage
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
	MessageId         string     `json:"msg_id"`
	NewWorkloads      []Workload `json:"new_workloads"`
	ModifiedWorkloads []Workload `json:"modified_workloads"`
	DeletedWorkloads  []Workload `json:"deleted_workloads"`
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

type WorkloadState int

// Workloads can be of several different types, namely 'preset' and 'template' and possibly 'trace'.
// Have not fully committed to making 'trace' a separate type from 'preset'.
//
// Workloads of type 'preset' are static in their definition, whereas workloads of type 'template'
// have properties that the user can specify and change before submitting the workload for registration.
type WorkloadType string

type Workload interface {
	// Return true if the workload stopped because it was explicitly terminated early/premature.
	IsTerminated() bool
	// Return true if the workload is registered and ready to be started.
	IsReady() bool
	// Return true if the workload stopped due to an error.
	IsErred() bool
	// Return true if the workload is actively running/in-progress.
	IsRunning() bool
	// Return true if the workload finished in any capacity (i.e., either successfully or due to an error).
	IsFinished() bool
	// Return true if the workload stopped naturally/successfully after processing all events.
	DidCompleteSuccessfully() bool
	// To String.
	String() string
	// Return the unique ID of the workload.
	GetId() string
	// Return the name of the workload.
	// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
	WorkloadName() string
	// Return the current state of the workload.
	GetWorkloadState() WorkloadState
	// Return the time elapsed, which is computed at the time that data is requested by the user.
	GetTimeElasped() time.Duration
	// Return the time elapsed as a string, which is computed at the time that data is requested by the user.
	GetTimeElaspedAsString() string
	// Update the time elapsed.
	SetTimeElasped(time.Duration)
	// Instruct the Workload to recompute its 'time elapsed' field.
	UpdateTimeElapsed()
	// Return the number of events processed by the workload.
	GetNumEventsProcessed() int64
	// Return the time that the workload was started.
	GetStartTime() time.Time
	// Get the time at which the workload finished.
	// If the workload hasn't finished yet, the returned boolean will be false.
	// If the workload has finished, then the returned boolean will be true.
	GetEndTime() (time.Time, bool)
	// Return the time that the workload was registered.
	GetRegisteredTime() time.Time
	// Return the workload's seed.
	GetSeed() int64
	// Set the workload's seed. Can only be performed once. If attempted again, this will panic.
	SetSeed(seed int64)
	// Get the error message associated with the workload.
	// If the workload is not in an ERROR state, then this returns the empty string and false.
	// If the workload is in an ERROR state, then the boolean returned will be true.
	GetErrorMessage() (string, bool)
	// Set the error message for the workload.
	SetErrorMessage(string)
	// Return a flag indicating whether debug logging is enabled.
	IsDebugLoggingEnabled() bool
	// Enable or disable debug logging for the workload.
	SetDebugLoggingEnabled(enabled bool)
	// Set the state of the workload.
	SetWorkloadState(state WorkloadState)
	// Start the Workload.
	StartWorkload()
	// Mark the workload as having completed successfully.
	SetWorkloadCompleted()
	// Called after an event is processed for the Workload.
	// Just updates some internal metrics.
	ProcessedEvent()
	// Called when a Session is created for/in the Workload.
	// Just updates some internal metrics.
	SessionCreated()
	// Called when a Session is stopped for/in the Workload.
	// Just updates some internal metrics.
	SessionStopped()
	// Called when a training starts during/in the workload.
	// Just updates some internal metrics.
	TrainingStarted()
	// Called when a training stops during/in the workload.
	// Just updates some internal metrics.
	TrainingStopped()
	// Get the type of workload (TRACE, PRESET, or TEMPLATE).
	GetWorkloadType() WorkloadType
	// Return true if this workload was created using a preset.
	IsPresetWorkload() bool
	// Return true if this workload was created using a template.
	IsTemplateWorkload() bool
	// Return true if this workload was created using the trace data.
	IsTraceWorkload() bool
	// If this is a preset workload, return the name of the preset.
	// If this is a trace workload, return the trace information.
	// If this is a template workload, return the template information.
	GetWorkloadSource() interface{}
	// Get the workload's Timescale Adjustment Factor, which effects the
	// timescale at which tickets are replayed/"simulated".
	GetTimescaleAdjustmentFactor() float64
}

type workloadImpl struct {
	Id                        string        `json:"id"`
	Name                      string        `json:"name"`
	WorkloadState             WorkloadState `json:"workload_state"`
	DebugLoggingEnabled       bool          `json:"debug_logging_enabled"`
	ErrorMessage              string        `json:"error_message"`
	Seed                      int64         `json:"seed"`
	seedSet                   bool
	RegisteredTime            time.Time     `json:"registered_time"`
	StartTime                 time.Time     `json:"start_time"`
	EndTime                   time.Time     `json:"end_time"`
	WorkloadDuration          time.Duration `json:"workload_duration"` // The total time that the workload executed for. This is only set once the workload has completed.
	TimeElasped               time.Duration `json:"time_elapsed"`      // Computed at the time that the data is requested by the user. This is the time elapsed SO far.
	NumTasksExecuted          int64         `json:"num_tasks_executed"`
	NumEventsProcessed        int64         `json:"num_events_processed"`
	NumSessionsCreated        int64         `json:"num_sessions_created"`
	NumActiveSessions         int64         `json:"num_active_sessions"`
	NumActiveTrainings        int64         `json:"num_active_trainings"`
	TimescaleAdjustmentFactor float64       `json:"timescale_adjustment_factor"`
	WorkloadType              WorkloadType  `json:"workload_type"`

	// workloadSource interface{} `json:"-"`

	// This is basically the child struct.
	// So, if this is a preset workload, then this is the WorkloadFromPreset struct.
	// We use this so we can delegate certain method calls to the child/derived struct.
	workload Workload `json:"-"`
}

// Get the type of workload (TRACE, PRESET, or TEMPLATE).
func (w *workloadImpl) GetWorkloadType() WorkloadType {
	return w.WorkloadType
}

// Return true if this workload was created using a preset.
func (w *workloadImpl) IsPresetWorkload() bool {
	return w.WorkloadType == PresetWorkload
}

// Return true if this workload was created using a template.
func (w *workloadImpl) IsTemplateWorkload() bool {
	return w.WorkloadType == TemplateWorkload
}

// Return true if this workload was created using the trace data.
func (w *workloadImpl) IsTraceWorkload() bool {
	return w.WorkloadType == TraceWorkload
}

// If this is a preset workload, return the name of the preset.
// If this is a trace workload, return the trace information.
// If this is a template workload, return the template information.
func (w *workloadImpl) GetWorkloadSource() interface{} {
	return w.workload.GetWorkloadSource()
}

func (w *workloadImpl) StartWorkload() {
	w.WorkloadState = WorkloadRunning
	w.StartTime = time.Now()
}

func (w *workloadImpl) GetTimescaleAdjustmentFactor() float64 {
	return w.TimescaleAdjustmentFactor
}

// Mark the workload as having completed successfully.
func (w *workloadImpl) SetWorkloadCompleted() {
	w.WorkloadState = WorkloadFinished
	w.EndTime = time.Now()
	w.WorkloadDuration = time.Since(w.StartTime)
}

// Get the error message associated with the workload.
// If the workload is not in an ERROR state, then this returns the empty string and false.
// If the workload is in an ERROR state, then the boolean returned will be true.
func (w *workloadImpl) GetErrorMessage() (string, bool) {
	if w.WorkloadState == WorkloadErred {
		return w.ErrorMessage, true
	}

	return "", false
}

// Set the error message for the workload.
func (w *workloadImpl) SetErrorMessage(errorMessage string) {
	w.ErrorMessage = errorMessage
}

// Return a flag indicating whether debug logging is enabled.
func (w *workloadImpl) IsDebugLoggingEnabled() bool {
	return w.DebugLoggingEnabled
}

// Enable or disable debug logging for the workload.
func (w *workloadImpl) SetDebugLoggingEnabled(enabled bool) {
	w.DebugLoggingEnabled = enabled
}

// Set the workload's seed. Can only be performed once. If attempted again, this will panic.
func (w *workloadImpl) SetSeed(seed int64) {
	if w.seedSet {
		panic(fmt.Sprintf("Workload seed has already been set to value %d", w.Seed))
	}

	w.Seed = seed
	w.seedSet = true
}

// Return the workload's seed.
func (w *workloadImpl) GetSeed() int64 {
	return w.Seed
}

// Return the current state of the workload.
func (w *workloadImpl) GetWorkloadState() WorkloadState {
	return w.WorkloadState
}

// Set the state of the workload.
func (w *workloadImpl) SetWorkloadState(state WorkloadState) {
	w.WorkloadState = state
}

// Return the time that the workload was started.
func (w *workloadImpl) GetStartTime() time.Time {
	return w.StartTime
}

// Get the time at which the workload finished.
// If the workload hasn't finished yet, the returned boolean will be false.
// If the workload has finished, then the returned boolean will be true.
func (w *workloadImpl) GetEndTime() (time.Time, bool) {
	if w.IsFinished() {
		return w.EndTime, true
	}

	return time.Time{}, false
}

// Return the time that the workload was registered.
func (w *workloadImpl) GetRegisteredTime() time.Time {
	return w.RegisteredTime
}

// Return the time elapsed, which is computed at the time that data is requested by the user.
func (w *workloadImpl) GetTimeElasped() time.Duration {
	return w.TimeElasped
}

// Return the time elapsed as a string, which is computed at the time that data is requested by the user.
func (w *workloadImpl) GetTimeElaspedAsString() string {
	return w.TimeElasped.String()
}

// Update the time elapsed.
func (w *workloadImpl) SetTimeElasped(timeElapsed time.Duration) {
	w.TimeElasped = timeElapsed
}

// Instruct the Workload to recompute its 'time elapsed' field.
func (w *workloadImpl) UpdateTimeElapsed() {
	w.TimeElasped = time.Since(w.StartTime)
}

// Return the number of events processed by the workload.
func (w *workloadImpl) GetNumEventsProcessed() int64 {
	return w.NumEventsProcessed
}

// Return the name of the workload.
// The name is not necessarily unique and is meant to be descriptive, whereas the ID is unique.
func (w *workloadImpl) WorkloadName() string {
	return w.Name
}

// Called after an event is processed for the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) ProcessedEvent() {
	w.NumEventsProcessed += 1
}

// Called when a Session is created for/in the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) SessionCreated() {
	w.NumActiveSessions += 1
	w.NumSessionsCreated += 1
}

// Called when a Session is stopped for/in the Workload.
// Just updates some internal metrics.
func (w *workloadImpl) SessionStopped() {
	w.NumActiveSessions -= 1
}

// Called when a training starts during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStarted() {
	w.NumActiveTrainings += 1
}

// Called when a training stops during/in the workload.
// Just updates some internal metrics.
func (w *workloadImpl) TrainingStopped() {
	w.NumTasksExecuted += 1
	w.NumActiveTrainings -= 1
}

// Return the unique ID of the workload.
func (w *workloadImpl) GetId() string {
	return w.Id
}

// Return true if the workload stopped because it was explicitly terminated early/premature.
func (w *workloadImpl) IsTerminated() bool {
	return w.WorkloadState == WorkloadTerminated
}

// Return true if the workload is registered and ready to be started.
func (w *workloadImpl) IsReady() bool {
	return w.WorkloadState == WorkloadReady
}

// Return true if the workload stopped due to an error.
func (w *workloadImpl) IsErred() bool {
	return w.WorkloadState == WorkloadErred
}

// Return true if the workload is actively running/in-progress.
func (w *workloadImpl) IsRunning() bool {
	return w.WorkloadState == WorkloadRunning
}

// Return true if the workload stopped naturally/successfully after processing all events.
func (w *workloadImpl) IsFinished() bool {
	return w.IsErred() || w.DidCompleteSuccessfully()
}

// Return true if the workload stopped naturally/successfully after processing all events.
func (w *workloadImpl) DidCompleteSuccessfully() bool {
	return w.WorkloadState == WorkloadFinished
}

func (w *workloadImpl) String() string {
	out, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type WorkloadPresetType string

type BaseWorkloadPreset struct {
	Name        string             `name:"name" yaml:"name" json:"name" description:"Human-readable name for this particular workload preset."`                                   // Human-readable name for this particular workload preset.
	Description string             `name:"description" yaml:"description" json:"description" description:"Human-readable description of the workload."`                           // Human-readable description of the workload.
	Key         string             `name:"key"  yaml:"key" json:"key" description:"Key for code-use only (i.e., we don't intend to display this to the user for the most part)."` // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	PresetType  WorkloadPresetType `name:"preset_type" yaml:"preset_type" json:"preset_type" description:"The type of workload preset. Could be CSV or XML."`
}

type WorkloadPreset struct {
	PresetType WorkloadPresetType `name:"preset_type" yaml:"preset_type" json:"preset_type" description:"The type of workload preset. Could be CSV or XML."`
	CsvWorkloadPreset
	XmlWorkloadPreset
}

type WorkloadTemplate struct {
	Name     string             `json:"name"`
	Sessions []*WorkloadSession `json:"sessions"`
}

func (t *WorkloadTemplate) String() string {
	return fmt.Sprintf("Template[%s]", t.Name)
}

type WorkloadFromTemplate struct {
	*workloadImpl

	Template *WorkloadTemplate `json:"workload_template"`
}

func (w *WorkloadFromTemplate) GetWorkloadSource() interface{} {
	return w.Template
}

type WorkloadFromPreset struct {
	*workloadImpl

	WorkloadPreset     *WorkloadPreset `json:"workload_preset"`
	WorkloadPresetName string          `json:"workload_preset_name"`
	WorkloadPresetKey  string          `json:"workload_preset_key"`
}

func (w *WorkloadFromPreset) GetWorkloadSource() interface{} {
	return w.WorkloadPreset
}

func NewWorkload(id string, workloadName string, seed int64, debugLoggingEnabled bool, timescaleAdjustmentFactor float64) Workload {
	return &workloadImpl{
		Id:                        id, // Same ID as the driver.
		Name:                      workloadName,
		WorkloadState:             WorkloadReady,
		TimeElasped:               time.Duration(0),
		Seed:                      seed,
		RegisteredTime:            time.Now(),
		NumTasksExecuted:          0,
		NumEventsProcessed:        0,
		NumSessionsCreated:        0,
		NumActiveSessions:         0,
		NumActiveTrainings:        0,
		DebugLoggingEnabled:       debugLoggingEnabled,
		TimescaleAdjustmentFactor: timescaleAdjustmentFactor,
		WorkloadType:              UnspecifiedWorkload,
	}
}

func NewWorkloadFromPreset(baseWorkload Workload, workloadPreset *WorkloadPreset) *WorkloadFromPreset {
	if workloadPreset == nil {
		panic("Workload preset cannot be nil when creating a new workload from a preset.")
	}

	if baseWorkload == nil {
		panic("Base workload cannot be nil when creating a new workload.")
	}

	var (
		baseWorkloadImpl *workloadImpl
		ok               bool
	)
	if baseWorkloadImpl, ok = baseWorkload.(*workloadImpl); !ok {
		panic("The provided workload is not a base workload, or it is not a pointer type.")
	}

	workload_from_preset := &WorkloadFromPreset{
		workloadImpl:       baseWorkloadImpl,
		WorkloadPreset:     workloadPreset,
		WorkloadPresetName: workloadPreset.GetName(),
		WorkloadPresetKey:  workloadPreset.GetKey(),
	}

	baseWorkloadImpl.WorkloadType = PresetWorkload
	baseWorkloadImpl.workload = workload_from_preset

	return workload_from_preset
}

func NewWorkloadFromTemplate(baseWorkload Workload, workloadTemplate *WorkloadTemplate) *WorkloadFromTemplate {
	if workloadTemplate == nil {
		panic("Workload template cannot be nil when creating a new workload from a template.")
	}

	if baseWorkload == nil {
		panic("Base workload cannot be nil when creating a new workload.")
	}

	var (
		baseWorkloadImpl *workloadImpl
		ok               bool
	)
	if baseWorkloadImpl, ok = baseWorkload.(*workloadImpl); !ok {
		panic("The provided workload is not a base workload, or it is not a pointer type.")
	}

	workload_from_template := &WorkloadFromTemplate{
		workloadImpl: baseWorkloadImpl,
		Template:     workloadTemplate,
	}

	baseWorkloadImpl.WorkloadType = TemplateWorkload
	baseWorkloadImpl.workload = workload_from_template

	return workload_from_template
}

func (p *WorkloadPreset) MarshalJSON() ([]byte, error) {
	if p.IsCsv() {
		return json.Marshal(p.CsvWorkloadPreset)
	} else if p.IsXml() {
		return json.Marshal(p.XmlWorkloadPreset)
	} else {
		panic(fmt.Sprintf("WorkloadPreset is of invalid type: %v", p.PresetType))
	}
}

func (p *WorkloadPreset) GetKey() string {
	if p.IsCsv() {
		return p.CsvWorkloadPreset.Key
	} else if p.IsXml() {
		return p.XmlWorkloadPreset.Key
	} else {
		panic(fmt.Sprintf("WorkloadPreset is of invalid type: %v", p.PresetType))
	}
}

func (p *WorkloadPreset) GetName() string {
	if p.IsCsv() {
		return p.CsvWorkloadPreset.Name
	} else if p.IsXml() {
		return p.XmlWorkloadPreset.Name
	} else {
		panic(fmt.Sprintf("WorkloadPreset is of invalid type: %v", p.PresetType))
	}
}

func (p *WorkloadPreset) Description() string {
	if p.IsCsv() {
		return p.CsvWorkloadPreset.Description
	} else if p.IsXml() {
		return p.XmlWorkloadPreset.Description
	} else {
		panic(fmt.Sprintf("WorkloadPreset is of invalid type: %v", p.PresetType))
	}
}

func (p *WorkloadPreset) String() string {
	if p.IsCsv() {
		return p.CsvWorkloadPreset.String()
	} else if p.IsXml() {
		return p.XmlWorkloadPreset.String()
	} else {
		panic(fmt.Sprintf("WorkloadPreset is of invalid type: %v", p.PresetType))
	}
}

func (p *WorkloadPreset) IsCsv() bool {
	return p.PresetType == CsvWorkloadPresetType
}

func (p *WorkloadPreset) IsXml() bool {
	return p.PresetType == XmlWorkloadPresetType
}

func (p *WorkloadPreset) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var basePreset BaseWorkloadPreset
	err := unmarshal(&basePreset)
	if err != nil {
		log.Fatalf("Failed to unmarshal workload preset: %v", err)
	}

	if basePreset.PresetType == CsvWorkloadPresetType {
		var csvPreset CsvWorkloadPreset
		err := unmarshal(&csvPreset)
		if err != nil {
			log.Fatalf("Failed to unmarshal XML preset: %v\n", err)
		}

		csvPreset.BaseWorkloadPreset = basePreset
		p.PresetType = CsvWorkloadPresetType
		p.CsvWorkloadPreset = csvPreset
		if err != nil {
			log.Fatalf("Failed to unmarshal CSV workload preset: %v\n", err)
		}
		log.Printf("Unmarshaled CSV workload preset (1): \"%v\"\n", csvPreset)
		log.Printf("Unmarshaled CSV workload preset (2): \"%v\"\n", p.CsvWorkloadPreset)
		log.Printf("Unmarshaled CSV workload preset (3): \"%s\"\n", p.CsvWorkloadPreset.Name)
	} else if basePreset.PresetType == XmlWorkloadPresetType {
		var xmlPreset XmlWorkloadPreset
		err := unmarshal(&xmlPreset)
		if err != nil {
			log.Fatalf("Failed to unmarshal XML preset: %v\n", err)
		}

		xmlPreset.BaseWorkloadPreset = basePreset

		// Some presets may not have an associated SVG file.
		if xmlPreset.SvgFilePath != "" {
			err = xmlPreset.LoadSvgContent()
			if err != nil {
				log.Printf("[ERROR] Could not load SVG content for XML preset %s from file \"%s\": %v\n", xmlPreset.GetName(), xmlPreset.SvgFilePath, err)
			} else {
				log.Printf("Successfully loaded SVG content for for XML preset %s from file \"%s\"\n", xmlPreset.GetName(), xmlPreset.SvgFilePath)
			}
		}

		cpuSessionMap := make(map[string]float64)
		cpuSessionMap["test_session1-41fc-833f-ffe4ef931c7d"] = 2
		cpuSessionMap["test_session2-448f-b21b-3855540d96ec"] = 2
		memSessionMap := make(map[string]float64)
		memSessionMap["test_session1-41fc-833f-ffe4ef931c7d"] = 4
		memSessionMap["test_session2-448f-b21b-3855540d96ec"] = 4
		gpuSessionMap := make(map[string]int)
		gpuSessionMap["test_session1-41fc-833f-ffe4ef931c7d"] = 1
		gpuSessionMap["test_session2-448f-b21b-3855540d96ec"] = 1

		cpuTaskMap := make(map[string][]float64)
		cpuTaskMap["test_session1-41fc-833f-ffe4ef931c7d"] = []float64{2, 2, 2, 2, 2, 2, 2, 2}
		cpuTaskMap["test_session2-448f-b21b-3855540d96ec"] = []float64{2, 2, 2, 2, 2, 2, 2, 2}
		memTaskMap := make(map[string][]float64)
		memTaskMap["test_session1-41fc-833f-ffe4ef931c7d"] = []float64{4, 4, 4, 4, 4, 4, 4, 4}
		memTaskMap["test_session2-448f-b21b-3855540d96ec"] = []float64{4, 4, 4, 4, 4, 4, 4, 4}
		gpuTaskMap := make(map[string][]int)
		gpuTaskMap["test_session1-41fc-833f-ffe4ef931c7d"] = []int{1, 1, 1, 1, 1, 1, 1, 1}
		gpuTaskMap["test_session2-448f-b21b-3855540d96ec"] = []int{1, 1, 1, 1, 1, 1, 1, 1}
		xmlPreset.MaxUtilization = NewMaxUtilizationWrapper(cpuSessionMap, memSessionMap, gpuSessionMap, cpuTaskMap, memTaskMap, gpuTaskMap)

		p.PresetType = XmlWorkloadPresetType
		p.XmlWorkloadPreset = xmlPreset

		if err != nil {
			log.Fatalf("Failed to unmarshal CSV workload preset: %v\n", err)
		}
		log.Printf("Unmarshaled XML workload preset \"%s\"\n", p.XmlWorkloadPreset.Name)
	} else {
		log.Fatalf("Unsupported workload preset type: %v", basePreset.PresetType)
	}

	return nil
}

func (p *BaseWorkloadPreset) GetName() string {
	return p.Name
}

func (p *BaseWorkloadPreset) GetDescription() string {
	return p.Description
}

func (p *BaseWorkloadPreset) GetKey() string {
	return p.Key
}

func (p *BaseWorkloadPreset) GetPresetType() WorkloadPresetType {
	return p.PresetType
}

func (p *BaseWorkloadPreset) String() string {
	return fmt.Sprintf("BaseWorkloadPreset[Name=%s,Key=%s,PresetType=%s]", p.Name, p.Key, p.PresetType)
}

type XmlWorkloadPreset struct {
	BaseWorkloadPreset
	XmlFilePath    string                 `json:"-" yaml:"xml_file" description:"File path to the XML file definining the workload's tasks."` // File path to the XML file definining the workload's tasks.
	SvgFilePath    string                 `json:"-" yaml:"svg_file" description:"File path to SVG file for rendering the events."`
	SvgContent     string                 `json:"svg_content" description:"The contents of the SVG file."`
	MaxUtilization *MaxUtilizationWrapper `json:"max_utilization" yaml:"max_utilization" description:"Max utilizations of the events contained within the preset."`
}

func NewMaxUtilizationWrapper(cpuSessionMap map[string]float64, memSessionMap map[string]float64, gpuSessionMap map[string]int, cpuTaskMap map[string][]float64, memTaskMap map[string][]float64, gpuTaskMap map[string][]int) *MaxUtilizationWrapper {
	maxUtilizationWrapper := &MaxUtilizationWrapper{
		MemSessionMap:            memSessionMap,
		CpuSessionMap:            cpuSessionMap,
		GpuSessionMap:            gpuSessionMap,
		CpuTaskMap:               cpuTaskMap,
		MemTaskMap:               memTaskMap,
		GpuTaskMap:               gpuTaskMap,
		CurrentTrainingNumberMap: make(map[string]int),
	}

	// Initialize the entries for all the tasks in the CurrentTrainingNumberMap.
	for key := range maxUtilizationWrapper.CpuTaskMap {
		maxUtilizationWrapper.CurrentTrainingNumberMap[key] = 0
	}

	return maxUtilizationWrapper
}

func (p *XmlWorkloadPreset) LoadSvgContent() error {
	fileContents, err := os.ReadFile(p.SvgFilePath)
	if err != nil {
		return err
	}

	p.SvgContent = string(fileContents)
	return nil
}

func (p *XmlWorkloadPreset) GetName() string {
	return p.Name
}

func (p *XmlWorkloadPreset) GetDescription() string {
	return p.Description
}

func (p *XmlWorkloadPreset) GetKey() string {
	return p.Key
}

func (p *XmlWorkloadPreset) GetXmlFilePath() string {
	return p.XmlFilePath
}

func (p *XmlWorkloadPreset) String() string {
	return fmt.Sprintf("CsvWorkloadPreset[Name=%s,Key=%s,XmlFile=%s]", p.Name, p.Key, p.XmlFilePath)
}

type CsvWorkloadPreset struct {
	BaseWorkloadPreset
	Months            []string `name:"months" yaml:"months" json:"months" description:"The months of data used by the workload."` // The months of data used by the workload.
	MonthsDescription string   `name:"months_description" yaml:"months_description" json:"months_description" description:"Formatted, human-readable text of the form (StartMonth) - (EndMonth) or (Month) if there is only one month included in the trace."`

	/* The following fields are not sent to the client. They're just used by the server. */

	GPUTraceFile      string `yaml:"gputrace" json:"-" name:"gputrace" description:"File path of GPU utilization trace."`
	GPUTraceStep      int64  `yaml:"gputrace-step" json:"-" name:"gputrace-step" description:"Interval, in seconds, of two consecutive trace readings of GPU."`
	GPUMappingFile    string `yaml:"gpumap" json:"-" name:"gpumap" description:"File path of GPU idx/pod map."`
	MaxSessionCpuFile string `yaml:"max-session-cpu-file" json:"-" name:"max-session-cpu-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the lifetime-maximum CPU utilization of each session."`        // File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max CPU used by each session.
	MaxSessionMemFile string `yaml:"max-session-mem-file" json:"-" name:"max-session-mem-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the lifetime-maximum memory (in bytes) used by each session."` // // File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max memory (in bytes) used by each session.
	MaxSessionGpuFile string `yaml:"max-session-gpu-file" json:"-" name:"max-session-gpu-file" desciption:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the lifetime-maximum GPU utilization of each session."`         // File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max CPU used by each session.
	MaxTaskCpuFile    string `yaml:"max-task-cpu-file" json:"-" name:"max-task-cpu-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max CPU utilization achieved within each individual training task."`
	MaxTaskMemFile    string `yaml:"max-task-mem-file" json:"-" name:"max-task-mem-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max memory (in bytes) used within each individual training task."`
	MaxTaskGpuFile    string `yaml:"max-task-gpu-file" json:"-" name:"max-task-gpu-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max GPU utilization achieved within each individual training task."`
	CPUTraceFile      string `yaml:"cputrace" json:"-" name:"cputrace" description:"File path of CPU counter trace."`
	CPUTraceStep      int64  `yaml:"cputrace-step" json:"-" name:"cputrace-step" description:"Interval, in seconds, of two consecutive trace readings of CPU."`
	CPUMappingFile    string `yaml:"cpumap" json:"-" name:"cpumap" description:"File path of CPU idx/pod map."`
	CPUDowntime       string `yaml:"cpudown" json:"-" name:"cpudown" description:"CPU trace downtime."`
	MemTraceFile      string `yaml:"memtrace" json:"-" name:"memtrace" description:"File path of memory usage trace."`
	MemTraceStep      int64  `yaml:"memtrace-step" json:"-" name:"memtrace-step" description:"Interval, in seconds, of two consecutive trace readings of memory."`
	MemMappingFile    string `yaml:"memmap" json:"-" name:"memmap" description:"File path of memory idx/pod map."`
	FromMonth         string `yaml:"from-month" json:"-" name:"from-month" description:"Month the trace starts if the path of trace file contains placeholder."`
	ToMonth           string `yaml:"to-month" json:"-" name:"to-month" description:"Month the trace ends if the path of trace file contains placeholder."`
}

func (p *CsvWorkloadPreset) GetName() string {
	return p.Name
}

func (p *CsvWorkloadPreset) GetDescription() string {
	return p.Description
}

func (p *CsvWorkloadPreset) GetKey() string {
	return p.Key
}

func (p *CsvWorkloadPreset) String() string {
	return fmt.Sprintf("CsvWorkloadPreset[Name=%s,Key=%s,Months=%s]", p.Name, p.Key, p.Months)
}

func (p *CsvWorkloadPreset) NormalizeTracePaths(path string) []string {
	if p.FromMonth == "" {
		return []string{path}
	}

	paths := make([]string, 0, len(Months))
	fromMonth := 0
	// Match the start month
	if p.FromMonth != "" {
		for i := 0; i < len(Months); i++ {
			if Months[i] == p.FromMonth {
				fromMonth = i
			}
		}
	}
	// Match the end month
	for i := 0; i < len(Months); i++ {
		idx := (fromMonth + i) % len(Months)
		paths = append(paths, fmt.Sprintf(path, Months[idx]))
		if Months[idx] == p.ToMonth {
			return paths
		}
	}
	return paths
}

func (p *CsvWorkloadPreset) NormalizeDowntime(downtime string) []int64 {
	if downtime == "" {
		return nil
	}

	startEnds := strings.Split(downtime, ",")
	downtimes := make([]int64, len(startEnds))
	for i, startEnd := range startEnds {
		downtimes[i], _ = strconv.ParseInt(startEnd, 10, 64)
	}
	return downtimes
}

// Read a yaml file containing one or more CsvWorkloadPreset definitions.
// Return a list of *CsvWorkloadPreset containing the definitions from the file.
//
// Returns an error if an error occurred. In this case, the returned slice will be nil.
// If no error occurred and the slice was read/created successfully, then the returned error will be nil.
func LoadWorkloadPresetsFromFile(filepath string) ([]*WorkloadPreset, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open or read workload presets file: %v\n", err)
		return nil, err
	}

	workloadPresets := make([]*WorkloadPreset, 0)
	err = yaml.Unmarshal(file, &workloadPresets)

	if err != nil {
		fmt.Printf("[ERROR] Failed to unmarshal workload presets: %v\n", err)
		return nil, err
	}

	return workloadPresets, nil
}

type MaxUtilizationWrapper struct {
	CpuSessionMap map[string]float64 `json:"cpu-session-map" yaml:"cpu-session-map"` // Maximum CPU utilization achieved during each Session's lifetime.
	MemSessionMap map[string]float64 `json:"mem-session-map" yaml:"mem-session-map"` // Maximum memory used (in gigabytes) during each Session's lifetime.
	GpuSessionMap map[string]int     `json:"gpu-session-map" yaml:"hpu-session-map"` // Maximum number of GPUs used during each Session's lifetime.

	CurrentTrainingNumberMap map[string]int // Map from SessionID to the current training task number we're on (beginning with 0, then 1, then 2, ..., etc.)

	CpuTaskMap map[string][]float64 `json:"cpu-task-map" yaml:"cpu-task-map"` // Maximum CPU utilization achieved during each training event for each Session, arranged in chronological order of training events.
	MemTaskMap map[string][]float64 `json:"mem-task-map" yaml:"mem-task-map"` // Maximum memory used (in GB) during each training event for each Session, arranged in chronological order of training events.
	GpuTaskMap map[string][]int     `json:"gpu-task-map" yaml:"gpu-task-map"` // Maximum number of GPUs used during each training event for each Session, arranged in chronological order of training events.
}

// Corresponds to the `Session` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type WorkloadSession struct {
	Id          string          `json:"id"`
	MaxCPUs     float64         `json:"maxCPUs"`
	MaxMemoryGB float64         `json:"maxMemoryGB"`
	MaxNumGPUs  int             `json:"maxNumGPUs"`
	StartTick   int             `json:"startTick"`
	StopTick    int             `json:"stopTick"`
	Trainings   []TrainingEvent `json:"trainings"`
}

// Corresponds to the `TrainingEvent` struct defined in `web/app/Data/workloadImpl.tsx`.
// Used by the frontend when submitting workloads created from templates (as opposed to presets).
type TrainingEvent struct {
	SessionId       string    `json:"sessionId"`
	TrainingId      string    `json:"trainingId"`
	CpuUtil         float64   `json:"cpuUtil"`
	MemUsageGB      float64   `json:"memUsageGB"`
	GpuUtil         []float64 `json:"gpuUtil"`
	StartTick       int       `json:"startTick"`
	DurationInTicks int       `json:"durationInTicks"`
}

func (e *TrainingEvent) NumGPUs() int {
	return len(e.GpuUtil)
}
