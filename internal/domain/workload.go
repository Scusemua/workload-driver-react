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
	WorkloadReady      WorkloadState = iota // Workload is registered and ready to be started.
	WorkloadRunning    WorkloadState = 1    // Workload is actively running/in-progress.
	WorkloadFinished   WorkloadState = 2    // Workload stopped naturally/successfully after processing all events.
	WorkloadErred      WorkloadState = 3    // Workload stopped due to an error.
	WorkloadTerminated WorkloadState = 4    // Workload stopped because it was explicitly terminated early/premature.

	CsvWorkloadPresetType WorkloadPresetType = "CSV"
	XmlWorkloadPresetType WorkloadPresetType = "XML"
)

type WorkloadGenerator interface {
	GenerateWorkload(EventConsumer, *Workload, WorkloadPreset, *WorkloadRegistrationRequest) error // Start generating the workload.
	StopGeneratingWorkload()                                                                       // Stop generating the workload prematurely.
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
	MessageId         string      `json:"msg_id"`
	NewWorkloads      []*Workload `json:"new_workloads"`
	ModifiedWorkloads []*Workload `json:"modified_workloads"`
	DeletedWorkloads  []*Workload `json:"deleted_workloads"`
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
	Key  string `name:"key" yaml:"key" json:"key" description:"Key for code-use only (i.e., we don't intend to display this to the user for the most part)."` // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	Seed int64  `name:"seed" yaml:"seed" json:"seed" description:"RNG seed for the workload."`
	// By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'.
	// This will lead to many sessions reserving fewer GPUs than when this property is disabled (default).
	AdjustGpuReservations bool   `name:"adjust_gpu_reservations" json:"adjust_gpu_reservations" description:"By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'. This will lead to many sessions reserving fewer GPUs than when this property is disabled (default)."`
	WorkloadName          string `name:"name" json:"name" yaml:"name" description:"Non-unique identifier of the workload created/specified by the user when launching the workload."`
	DebugLogging          bool   `name:"debug_logging" json:"debug_logging" yaml:"debug_logging" description:"Flag indicating whether debug-level logging should be enabled."`
}

func (r *WorkloadRegistrationRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type WorkloadState int

type Workload struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	WorkloadState       WorkloadState  `json:"workload_state"`
	WorkloadPreset      WorkloadPreset `json:"workload_preset"`
	WorkloadPresetName  string         `json:"workload_preset_name"`
	WorkloadPresetKey   string         `json:"workload_preset_key"`
	DebugLoggingEnabled bool           `json:"debug_logging_enabled"`
	ErrorMessage        string         `json:"error_message"`

	Seed int64 `json:"seed"`

	RegisteredTime     time.Time `json:"registered_time"`
	StartTime          time.Time `json:"start_time"`
	TimeElasped        string    `json:"time_elapsed"` // Computed at the time that the data is requested by the user.
	NumTasksExecuted   int64     `json:"num_tasks_executed"`
	NumEventsProcessed int64     `json:"num_events_processed"`
	NumSessionsCreated int64     `json:"num_sessions_created"`
	NumActiveSessions  int64     `json:"num_active_sessions"`
	NumActiveTrainings int64     `json:"num_active_trainings"`
}

// Return true if the workload stopped because it was explicitly terminated early/premature.
func (w *Workload) IsTerminated() bool {
	return w.WorkloadState == WorkloadTerminated
}

// Return true if the workload is registered and ready to be started.
func (w *Workload) IsReady() bool {
	return w.WorkloadState == WorkloadReady
}

// Return true if the workload stopped due to an error.
func (w *Workload) IsErred() bool {
	return w.WorkloadState == WorkloadErred
}

// Return true if the workload is actively running/in-progress.
func (w *Workload) IsRunning() bool {
	return w.WorkloadState == WorkloadRunning
}

// Return true if the workload stopped naturally/successfully after processing all events.
func (w *Workload) IsFinished() bool {
	return w.WorkloadState == WorkloadFinished
}

func (w *Workload) String() string {
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

func (p *WorkloadPreset) MarshalJSON() ([]byte, error) {
	if p.IsCsv() {
		return json.Marshal(p.CsvWorkloadPreset)
	} else if p.IsXml() {
		return json.Marshal(p.XmlWorkloadPreset)
	} else {
		panic(fmt.Sprintf("WorkloadPreset is of invalid type: %v", p.PresetType))
	}
}

func (p *WorkloadPreset) Key() string {
	if p.IsCsv() {
		return p.CsvWorkloadPreset.Key
	} else if p.IsXml() {
		return p.XmlWorkloadPreset.Key
	} else {
		panic(fmt.Sprintf("WorkloadPreset is of invalid type: %v", p.PresetType))
	}
}

func (p *WorkloadPreset) Name() string {
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
	XmlFilePath string `json:"-" yaml:"xml_file" description:"File path to the XML file definining the workload's tasks."` // File path to the XML file definining the workload's tasks.
	SvgFilePath string `json:"-" yaml:"svg_file" description:"File path to SVG file for rendering the events."`
	SvgContent  string `json:"svg_content" description:"The contents of the SVG file."`
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
func LoadWorkloadPresetsFromFile(filepath string) ([]WorkloadPreset, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open or read workload presets file: %v\n", err)
		return nil, err
	}

	workloadPresets := make([]WorkloadPreset, 0)
	err = yaml.Unmarshal(file, &workloadPresets)

	if err != nil {
		fmt.Printf("[ERROR] Failed to unmarshal workload presets: %v\n", err)
		return nil, err
	}

	return workloadPresets, nil
}
