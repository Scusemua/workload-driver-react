package domain

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type WorkloadGenerator interface {
	GenerateWorkload(EventConsumer, *WorkloadPreset, *WorkloadRequest) error
}

type WorkloadRequest struct {
	Key  string `name:"key" yaml:"key" json:"key" description:"Key for code-use only (i.e., we don't intend to display this to the user for the most part)."` // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	Seed int    `name:"seed" yaml:"seed" json:"seed" description:"RNG seed for the workload."`
	// By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'.
	// This will lead to many sessions reserving fewer GPUs than when this property is disabled (default).
	AdjustGpuReservations bool   `name:"adjust_gpu_reservations" json:"adjust_gpu_reservations" description:"By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'. This will lead to many sessions reserving fewer GPUs than when this property is disabled (default)."`
	WorkloadName          string `name:"name" json:"name" yaml:"name" description:"Non-unique identifier of the workload created/specified by the user when launching the workload."`
}

type WorkloadPreset struct {
	Name        string   `name:"name" yaml:"name" json:"name" description:"Human-readable name for this particular workload preset."`                                   // Human-readable name for this particular workload preset.
	Description string   `name:"description" yaml:"description" json:"description" description:"Human-readable description of the workload."`                           // Human-readable description of the workload.
	Key         string   `name:"key"  yaml:"key" json:"key" description:"Key for code-use only (i.e., we don't intend to display this to the user for the most part)."` // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	Months      []string `name:"months" yaml:"months" json:"months" description:"The months of data used by the workload."`                                             // The months of data used by the workload.

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

func (p *WorkloadPreset) String() string {
	return fmt.Sprintf("WorkloadPreset[Name=%s,Key=%s,Months=%s]", p.Name, p.Key, p.Months)
}

// Read a yaml file containing one or more WorkloadPreset definitions.
// Return a list of *WorkloadPreset containing the definitions from the file.
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
