package domain

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const (
	CsvWorkloadPresetType WorkloadPresetType = "CSV"
	XmlWorkloadPresetType WorkloadPresetType = "XML"
)

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
		// log.Printf("Unmarshaled CSV workload preset (1): \"%v\"\n", csvPreset)
		// log.Printf("Unmarshaled CSV workload preset (2): \"%v\"\n", p.CsvWorkloadPreset)
		// log.Printf("Unmarshaled CSV workload preset (3): \"%s\"\n", p.CsvWorkloadPreset.Name)
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
			}
			// else {
			// 	log.Printf("Successfully loaded SVG content for for XML preset %s from file \"%s\"\n", xmlPreset.GetName(), xmlPreset.SvgFilePath)
			// }
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
		// log.Printf("Unmarshaled XML workload preset \"%s\"\n", p.XmlWorkloadPreset.Name)
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
