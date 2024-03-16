package generator

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/zhangjyr/gocsv"
	"go.uber.org/zap"
)

type workloadGeneratorImpl struct {
	ctx            context.Context
	cancelFunction context.CancelFunc

	synthesizer *Synthesizer

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	opts *domain.Configuration
}

func NewWorkloadGenerator(opts *domain.Configuration) domain.WorkloadGenerator {
	generator := &workloadGeneratorImpl{
		opts: opts,
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	generator.logger = logger
	generator.sugaredLogger = logger.Sugar()

	return generator
}

func (g *workloadGeneratorImpl) StopGeneratingWorkload() {
	g.logger.Debug("Stopping workload generation now.")
	g.cancelFunction()
}

func (g *workloadGeneratorImpl) GenerateWorkload(workloadDriver domain.EventConsumer, workload *domain.Workload, workloadPreset *domain.WorkloadPreset, workloadRegistrationRequest *domain.WorkloadRegistrationRequest) error {
	var cpuSessionMap, memSessionMap map[string]float64 = nil, nil
	var gpuSessionMap map[string]int = nil
	var cpuTaskMap, memTaskMap map[string][]float64 = nil, nil
	var gpuTaskMap map[string][]int = nil
	g.ctx, g.cancelFunction = context.WithCancel(context.Background())
	defer g.cancelFunction()

	// Per-Session max CPU.
	if workloadPreset.MaxSessionCpuFile != "" {
		cpuSessionMap = g.getSessionCpuMap(workloadPreset.MaxSessionCpuFile)
	}

	// Per-Session max memory.
	if workloadPreset.MaxSessionMemFile != "" {
		memSessionMap = g.getSessionMemMap(workloadPreset.MaxSessionMemFile)
	}

	// Per-Session max number of GPUs.
	if workloadPreset.MaxSessionGpuFile != "" {
		var err error
		gpuSessionMap, err = g.getSessionGpuMap(workloadPreset.MaxSessionGpuFile, workloadRegistrationRequest.AdjustGpuReservations)

		if err != nil {
			panic(err)
		}
	}

	// Per-training-event max CPU.
	if workloadPreset.MaxTaskCpuFile != "" {
		cpuTaskMap = g.getTrainingTaskCpuMap(workloadPreset.MaxTaskCpuFile)
	}

	// Per-training-event max memory.
	if workloadPreset.MaxTaskMemFile != "" {
		memTaskMap = g.getTrainingTaskMemMap(workloadPreset.MaxTaskMemFile)
	}

	// Per-training-event max number of GPUs.
	if workloadPreset.MaxTaskGpuFile != "" {
		gpuTaskMap = g.getTrainingTaskGpuMap(workloadPreset.MaxTaskGpuFile, workloadRegistrationRequest.AdjustGpuReservations)
	}

	maxUtilizationWrapper := NewMaxUtilizationWrapper(cpuSessionMap, memSessionMap, gpuSessionMap, cpuTaskMap, memTaskMap, gpuTaskMap)
	g.synthesizer = NewSynthesizer(g.opts, maxUtilizationWrapper)
	// Set the cluster as the EventHandler for the Synthesizer.
	g.synthesizer.SetEventConsumer(workloadDriver)

	g.logger.Debug("Driving GPU now.")

	// Drive GPU trace
	if workloadPreset.GPUTraceFile != "" {
		g.logger.Debug("Adding GPU driver as event source for Synthesizer.")
		gpuDriver := g.synthesizer.AddDriverEventSource(NewGPUDriver, func(d Driver) {
			drv := d.(*GPUDriver)
			drv.MapperPath = workloadPreset.GPUMappingFile
			drv.ReadingInterval = time.Duration(workloadPreset.GPUTraceStep) * time.Second
			drv.SessionMaxes = make(map[string]float64)

			if g.opts.LastTimestamp > 0 {
				drv.LastTimestamp = time.Unix(g.opts.LastTimestamp, 0)
			} else {
				drv.LastTimestamp = time.Time{}
			}
			drv.ExecutionMode = 1 // g.opts.ExecutionMode
			drv.DriverType = "GPU"
			drv.Rand = rand.New(rand.NewSource(workloadRegistrationRequest.Seed))

			g.logger.Debug("Created GPU driver.")
		})
		go gpuDriver.Drive(g.ctx, workloadPreset.NormalizeTracePaths(workloadPreset.GPUTraceFile)...)
	}

	g.logger.Debug("Driving CPU now.")

	// Drive CPU trace
	if workloadPreset.CPUTraceFile != "" {
		g.logger.Debug("Adding CPU driver as event source for Synthesizer.")
		cpuDriver := g.synthesizer.AddDriverEventSource(NewCPUDriver, func(d Driver) {
			drv := d.(*CPUDriver)
			drv.MapperPath = workloadPreset.CPUMappingFile
			drv.Downtimes = workloadPreset.NormalizeDowntime(workloadPreset.CPUDowntime)
			drv.ReadingInterval = time.Duration(workloadPreset.CPUTraceStep) * time.Second
			drv.SessionMaxes = make(map[string]float64)

			if g.opts.LastTimestamp > 0 {
				drv.LastTimestamp = time.Unix(g.opts.LastTimestamp, 0)
			} else {
				drv.LastTimestamp = time.Time{}
			}

			// drv.MaxSessionOutputPath = filepath.Join(outputSubdirectoryPath, g.opts.FromMonth+"-"+g.opts.ToMonth+"-session_cpu_maxes.csv")
			// g.sugaredLogger.Info("Set 'MaxSessionOutputPath' for CPU driver to \"%s\"", drv.MaxSessionOutputPath)
			// drv.MaxTrainingOutputPath = filepath.Join(outputSubdirectoryPath, g.opts.FromMonth+"-"+g.opts.ToMonth+"-training_cpu_maxes.csv")
			// g.sugaredLogger.Info("Set 'MaxTrainingOutputPath' for memory driver to \"%s\"", drv.MaxTrainingOutputPath)
			drv.ExecutionMode = 1 // g.opts.ExecutionMode
			drv.DriverType = "CPU"
			drv.Rand = rand.New(rand.NewSource(workloadRegistrationRequest.Seed))

			g.logger.Debug("Created CPU driver.")
		})
		go cpuDriver.Drive(g.ctx, workloadPreset.NormalizeTracePaths(workloadPreset.CPUTraceFile)...)
	}

	g.logger.Debug("Driving memory now.")

	// Drive memory trace
	if workloadPreset.MemTraceFile != "" {
		g.logger.Debug("Adding memory driver as event source for Synthesizer.")
		SessionReadyExpects |= SessionMemReady
		memDriver := g.synthesizer.AddDriverEventSource(NewMemoryDriver, func(d Driver) {
			drv := d.(*MemoryDriver)
			drv.MapperPath = workloadPreset.MemMappingFile
			// No downtime(s) for memory trace?
			// drv.Downtimes = g.opts.NormalizeDowntime(workload.MemDowntime)
			drv.ReadingInterval = time.Duration(workloadPreset.MemTraceStep) * time.Second
			drv.SessionMaxes = make(map[string]float64)

			if g.opts.LastTimestamp > 0 {
				drv.LastTimestamp = time.Unix(g.opts.LastTimestamp, 0)
			} else {
				drv.LastTimestamp = time.Time{}
			}

			// drv.MaxSessionOutputPath = filepath.Join(outputSubdirectoryPath, g.opts.FromMonth+"-"+g.opts.ToMonth+"-session_mem_maxes.csv")
			// g.sugaredLogger.Info("Set 'MaxSessionOutputPath' for memory driver to \"%s\"", drv.MaxSessionOutputPath)
			// drv.MaxTrainingOutputPath = filepath.Join(outputSubdirectoryPath, g.opts.FromMonth+"-"+g.opts.ToMonth+"-training_mem_maxes.csv")
			// g.sugaredLogger.Info("Set 'MaxTrainingOutputPath' for memory driver to \"%s\"", drv.MaxTrainingOutputPath)
			drv.ExecutionMode = 1 // g.opts.ExecutionMode
			drv.DriverType = "Memory"
			drv.Rand = rand.New(rand.NewSource(workloadRegistrationRequest.Seed))

			g.logger.Debug("Created memory driver.")
		})
		go memDriver.Drive(g.ctx, workloadPreset.NormalizeTracePaths(workloadPreset.MemTraceFile)...)
	}

	g.logger.Debug("Beginning to generate workload now.")

	g.synthesizer.Synthesize(g.ctx, g.opts, workloadDriver.DoneChan())

	return nil
}

func (g *workloadGeneratorImpl) getSessionGpuMap(filePath string, adjustGpuReservations bool) (map[string]int, error) {
	gpuSessionMap := make(map[string]int)

	fmt.Printf("Parsing `MaxSessionGpuFile` \"%v\"\n.", filePath)
	gpuSessionFile, err := os.Open(filePath)
	defer gpuSessionFile.Close()
	if err != nil {
		g.sugaredLogger.Errorf(fmt.Sprintf("Failed to open `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err))
		return nil, err
	}

	sessions := []*SessionMaxGpu{}
	if err := gocsv.UnmarshalFile(gpuSessionFile, &sessions); err != nil { // Load session data from file
		fmt.Printf("Failed to parse `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, session := range sessions {
		numGPUsFloat, err := strconv.ParseFloat(session.NumGPUs, 64)
		if err != nil {
			fmt.Printf("Failed to parse SessionNumGPUs value \"%v\" for Pod %v. Error: %v\n", session.NumGPUs, session.SessionID, err)
			panic(err)
		}

		// Depending on whether or not we're supposed to convert the number of GPUs using the utilization value -- do so (or don't).
		var numGPUs int
		if adjustGpuReservations {
			maxGpuUtilization, err := strconv.ParseFloat(session.MaxUtilization, 64)
			if err != nil {
				fmt.Printf("Failed to parse maxGpuUtilization value \"%v\" for Pod %v. Error: %v\n", session.MaxUtilization, session.SessionID, err)
				panic(err)
			}

			// Round up to the nearest whole number.
			numGPUs = int(RoundUp(maxGpuUtilization/100.0, 1.0))
			numGPUs = MinInt(numGPUs, int(numGPUsFloat))
			numGPUs = MaxInt(numGPUs, 1) // If numGPUs is set to 0 because maxGpuUtilization is 0, then default to 1, as we charge users based on number of GPUs.

			// Validate that the number we obtained is "reasonable" (i.e., it isn't negative, nor is it greater than the maximum possible number of GPUs a session is supposed to have used).
			if numGPUs < 0 || numGPUs > 8 {
				g.sugaredLogger.Errorf("Obtained unexpected value of %d when converting #GPUs value. NumGPUs: %.0f. Max GPU Utilization: %.2f. Session: %s.\n", numGPUs, numGPUsFloat, maxGpuUtilization, session)
			}
		} else {
			numGPUs = int(numGPUsFloat)
		}

		gpuSessionMap[session.SessionID] = numGPUs
	}

	return gpuSessionMap, nil
}

func (g *workloadGeneratorImpl) getSessionMemMap(filePath string) map[string]float64 {
	memorySessionMap := make(map[string]float64)
	fmt.Printf("Parsing `MaxSessionMemFile` \"%v\"\n.", filePath)
	memSessionFile, err := os.Open(filePath)
	defer memSessionFile.Close()
	if err != nil {
		g.sugaredLogger.Errorf(fmt.Sprintf("Failed to open `MaxSessionMemFile` \"%v\": %v\n.", filePath, err))
	}

	sessions := []*SessionMaxMemory{}
	if err := gocsv.UnmarshalFile(memSessionFile, &sessions); err != nil { // Load session data from file
		fmt.Printf("Failed to parse `MaxSessionMemFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxSessionMemFile` \"%v\": %v\n.", filePath, err))
	}
	for _, session := range sessions {
		maxMemory, err := strconv.ParseFloat(session.MaxMemoryBytes, 64)
		if err != nil {
			fmt.Printf("Failed to parse MaxSessionMem value \"%v\" for Pod %v. Error: %v\n", session.MaxMemoryBytes, session.SessionID, err)
			panic(err)
		}
		maxMem := math.Max(math.Ceil(maxMemory/1.0e9), 1)
		memorySessionMap[session.SessionID] = maxMem
	}

	return memorySessionMap
}

func (g *workloadGeneratorImpl) getSessionCpuMap(filePath string) map[string]float64 {
	cpuSessionMap := make(map[string]float64)
	fmt.Printf("Parsing `MaxSessionCpuFile` \"%v\"\n.", filePath)
	MaxSessionCpuFile, err := os.Open(filePath)
	defer MaxSessionCpuFile.Close()
	if err != nil {
		panic(fmt.Sprintf("Failed to open `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err))
	}

	sessions := []*SessionMaxCpu{}
	if err := gocsv.UnmarshalFile(MaxSessionCpuFile, &sessions); err != nil { // Load session data from file
		fmt.Printf("Failed to parse `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, session := range sessions {
		// fmt.Printf("Session[Pod=%v, GPUs=%v, GPUUtilMax=%v, MaxSessionCPU=%v]\n", session.Pod, session.GPUs, session.GPUUtilMax, session.MaxSessionCPU)
		maxCpuUtilPercentage, err := strconv.ParseFloat(session.MaxUtilization, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
		if err != nil {
			fmt.Printf("Failed to parse MaxSessionCPU value \"%v\" for Pod %v. Error: %v\n", session.MaxUtilization, session.SessionID, err)
			panic(err)
		}

		// Round up to nearest multiple of 100% for the utilization.
		// For example, 8.3% utilization goes to 100%.
		// 103.57% utilization goes to 200%.
		// Then divide that value by 100 to get the number of vCPUs required.
		maxCpu := math.Max(RoundUp(maxCpuUtilPercentage, 100)/100.0, 1)
		cpuSessionMap[session.SessionID] = maxCpu
	}

	return cpuSessionMap
}

func (g *workloadGeneratorImpl) getTrainingTaskCpuMap(filePath string) map[string][]float64 {
	cpuTrainingTaskMap := make(map[string][]float64)
	fmt.Printf("Parsing `MaxTrainingTaskCpuFile` \"%v\"\n.", filePath)
	MaxTrainingTaskCpuFile, err := os.Open(filePath)
	defer MaxTrainingTaskCpuFile.Close()
	if err != nil {
		panic(fmt.Sprintf("Failed to open `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err))
	}

	trainingMaxes := []*TrainingTaskMaxCpu{}
	if err := gocsv.UnmarshalFile(MaxTrainingTaskCpuFile, &trainingMaxes); err != nil { // Load session data from file
		fmt.Printf("Failed to parse `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, trainingMax := range trainingMaxes {
		maxCpuUtilPercentage, err := strconv.ParseFloat(trainingMax.MaxUtilization, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
		if err != nil {
			fmt.Printf("Failed to parse MaxTaskCPU value \"%v\" for Pod %s, Seq#: %s. Error: %v\n", trainingMax.MaxUtilization, trainingMax.SessionID, trainingMax.TrainingTaskNum, err)
			panic(err)
		}

		// Round up to nearest multiple of 100% for the utilization.
		// For example, 8.3% utilization goes to 100%.
		// 103.57% utilization goes to 200%.
		// Then divide that value by 100 to get the number of vCPUs required.
		maxCpu := math.Max(RoundUp(maxCpuUtilPercentage, 100)/100.0, 1)

		maxCpuUtilizations, ok := cpuTrainingTaskMap[trainingMax.SessionID]

		if !ok {
			maxCpuUtilizations = make([]float64, 0)
		}

		maxCpuUtilizations = append(maxCpuUtilizations, maxCpu)
		cpuTrainingTaskMap[trainingMax.SessionID] = maxCpuUtilizations
	}

	return cpuTrainingTaskMap
}

func (g *workloadGeneratorImpl) getTrainingTaskGpuMap(filePath string, adjustGpuReservations bool) map[string][]int {
	gpuTrainingTaskMap := make(map[string][]int)

	fmt.Printf("Parsing `MaxTrainingTaskGpuFile` \"%v\"\n.", filePath)
	gpuTrainingTaskFile, err := os.Open(filePath)
	defer gpuTrainingTaskFile.Close()
	if err != nil {
		g.sugaredLogger.Errorf(fmt.Sprintf("Failed to open `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err))
	}

	trainingTasks := []*TrainingTaskMaxGpu{}
	if err := gocsv.UnmarshalFile(gpuTrainingTaskFile, &trainingTasks); err != nil { // Load session data from file
		fmt.Printf("Failed to parse `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, trainingTask := range trainingTasks {
		numGPUsFloat, err := strconv.ParseFloat(trainingTask.NumGPUs, 64)
		if err != nil {
			fmt.Printf("Failed to parse TrainingNumGPUs value \"%v\" for Pod %v. Error: %v\n", trainingTask.NumGPUs, trainingTask.SessionID, err)
			panic(err)
		}

		// Depending on whether or not we're supposed to convert the number of GPUs using the utilization value -- do so (or don't).
		var numGPUs int
		if adjustGpuReservations {
			maxGpuUtilization, err := strconv.ParseFloat(trainingTask.MaxUtilization, 64)
			if err != nil {
				fmt.Printf("Failed to parse maxGpuUtilization value \"%v\" for Pod %v. Error: %v\n", trainingTask.MaxUtilization, trainingTask.SessionID, err)
				panic(err)
			}

			// Round up to the nearest whole number.
			numGPUs = int(RoundUp(maxGpuUtilization/100.0, 1.0))
			numGPUs = MinInt(numGPUs, int(numGPUsFloat))
			numGPUs = MaxInt(numGPUs, 1) // If numGPUs is set to 0 because maxGpuUtilization is 0, then default to 1, as we charge users based on number of GPUs.

			// Validate that the number we obtained is "reasonable" (i.e., it isn't negative, nor is it greater than the maximum possible number of GPUs a session is supposed to have used).
			if numGPUs < 0 || numGPUs > 8 {
				g.sugaredLogger.Errorf("Obtained unexpected value of %d when converting #GPUs value. NumGPUs: %.0f. Max GPU Utilization: %.2f. TrainingTask: %s.\n", numGPUs, numGPUsFloat, maxGpuUtilization, trainingTask)
			}
		} else {
			numGPUs = int(numGPUsFloat)
		}

		numGPUsList, ok := gpuTrainingTaskMap[trainingTask.SessionID]

		if !ok {
			numGPUsList = make([]int, 0)
		}

		numGPUsList = append(numGPUsList, numGPUs)
		gpuTrainingTaskMap[trainingTask.SessionID] = numGPUsList
	}

	return gpuTrainingTaskMap
}

func (g *workloadGeneratorImpl) getTrainingTaskMemMap(filePath string) map[string][]float64 {
	memTrainingTaskMap := make(map[string][]float64)
	fmt.Printf("Parsing `MaxTrainingTaskMemFile` \"%v\"\n.", filePath)
	MaxTrainingTaskMemFile, err := os.Open(filePath)
	defer MaxTrainingTaskMemFile.Close()
	if err != nil {
		panic(fmt.Sprintf("Failed to open `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err))
	}

	trainingMaxes := []*TrainingTaskMemory{}
	if err := gocsv.UnmarshalFile(MaxTrainingTaskMemFile, &trainingMaxes); err != nil { // Load session data from file
		fmt.Printf("Failed to parse `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err))
	}
	for _, trainingMax := range trainingMaxes {
		maxMemory, err := strconv.ParseFloat(trainingMax.MaxMemoryBytes, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
		if err != nil {
			fmt.Printf("Failed to parse MaxTaskMemory value \"%v\" for Pod %v, Seq#: %s. Error: %v\n", trainingMax.MaxMemoryBytes, trainingMax.SessionID, trainingMax.TrainingTaskNum, err)
			panic(err)
		}

		maxMem := math.Max(math.Ceil(maxMemory/1.0e9), 1)

		maxMemoryUsages, ok := memTrainingTaskMap[trainingMax.SessionID]

		if !ok {
			maxMemoryUsages = make([]float64, 0)
		}

		maxMemoryUsages = append(maxMemoryUsages, maxMem)
		memTrainingTaskMap[trainingMax.SessionID] = maxMemoryUsages
	}

	return memTrainingTaskMap
}
