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
	"go.uber.org/zap/zapcore"
)

//const (
//	OneSessionOneTraining string = "1 Session with 1 Training Event"
//)

type BasicWorkloadGenerator struct {
	ctx            context.Context
	cancelFunction context.CancelFunc

	driver domain.WorkloadDriver

	atom *zap.AtomicLevel

	synthesizer *Synthesizer

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	opts *domain.Configuration
}

func NewWorkloadGenerator(opts *domain.Configuration, atom *zap.AtomicLevel, driver domain.WorkloadDriver) *BasicWorkloadGenerator {
	generator := &BasicWorkloadGenerator{
		opts:   opts,
		atom:   atom,
		driver: driver,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	logger := zap.New(core, zap.Development())
	sugaredLogger := logger.Sugar()

	generator.logger = logger
	generator.sugaredLogger = sugaredLogger

	return generator
}

func (g *BasicWorkloadGenerator) startCpuDriverForXml(ctx context.Context, synth *Synthesizer, records []Record, doneChan chan struct{}, errorChan chan<- error) TraceDriver {
	cpuDriver := synth.AddDriverEventSource(NewCPUDriver, func(d TraceDriver) {
		drv := d.(*CPUDriver)
		drv.ReadingInterval = time.Second * time.Duration(60)
		drv.ExecutionMode = 1
		drv.Rand = rand.New(rand.NewSource(g.opts.Seed))
	})
	// TODO(Ben): Do not hardcode the Pod map. a230e335-d964-41fc-833f-ffe4ef931c7d
	cpuDriver.(*CPUDriver).SetPodMap([]string{"test_session1-41fc-833f-ffe4ef931c7d", "test_session2-448f-b21b-3855540d96ec", "test_session3-4e9d-844e-4d7a19fd3bb7"})
	go cpuDriver.DriveWithSlice(ctx, records, doneChan, errorChan)
	return cpuDriver
}

func (g *BasicWorkloadGenerator) startGpuDriverForXml(ctx context.Context, synth *Synthesizer, records []Record, doneChan chan struct{}, errorChan chan<- error) TraceDriver {
	gpuDriver := synth.AddDriverEventSource(NewGPUDriver, func(d TraceDriver) {
		drv := d.(*GPUDriver)
		drv.ReadingInterval = time.Second * time.Duration(60)
		drv.ExecutionMode = 1
		drv.Rand = rand.New(rand.NewSource(g.opts.Seed))
	})
	// TODO(Ben): Do not hardcode the Pod map.
	gpuDriver.(*GPUDriver).SetPodMap([]string{"test_session1-41fc-833f-ffe4ef931c7d", "test_session2-448f-b21b-3855540d96ec", "test_session3-4e9d-844e-4d7a19fd3bb7"})
	go gpuDriver.DriveWithSlice(ctx, records, doneChan, errorChan)
	return gpuDriver
}

func (g *BasicWorkloadGenerator) startMemoryDriver(ctx context.Context, synth *Synthesizer, records []Record, doneChan chan struct{}, errorChan chan<- error) TraceDriver {
	memDriver := synth.AddDriverEventSource(NewMemoryDriver, func(d TraceDriver) {
		drv := d.(*MemoryDriver)
		drv.ReadingInterval = time.Second * time.Duration(60)
		drv.ExecutionMode = 1
		drv.Rand = rand.New(rand.NewSource(g.opts.Seed))
	})
	// TODO(Ben): Do not hardcode the Pod map.
	memDriver.(*MemoryDriver).SetPodMap([]string{"test_session1-41fc-833f-ffe4ef931c7d", "test_session2-448f-b21b-3855540d96ec", "test_session3-4e9d-844e-4d7a19fd3bb7"})
	go memDriver.DriveWithSlice(ctx, records, doneChan, errorChan)
	return memDriver
}

func (g *BasicWorkloadGenerator) StopGeneratingWorkload() {
	g.logger.Debug("Stopping workload generation now.")

	if g.cancelFunction != nil {
		g.cancelFunction()
	}
}

func (g *BasicWorkloadGenerator) generateWorkloadWithCsvPreset(consumer domain.EventConsumer, maxUtilizationConsumer domain.MaxUtilizationConsumer,
	workloadPreset *domain.CsvWorkloadPreset, workloadRegistrationRequest *domain.WorkloadRegistrationRequest) error {

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
			return err
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

	maxUtilizationWrapper := domain.NewMaxUtilizationWrapper(cpuSessionMap, memSessionMap, gpuSessionMap, cpuTaskMap, memTaskMap, gpuTaskMap)
	maxUtilizationConsumer.SetMaxUtilizationWrapper(maxUtilizationWrapper)

	g.synthesizer = NewSynthesizer(g.opts, maxUtilizationWrapper, g.atom)
	// Set the cluster as the EventHandler for the Synthesizer.
	g.synthesizer.SetEventConsumer(consumer)

	g.logger.Debug("Driving GPU now.")

	// Drive GPU trace
	if workloadPreset.GPUTraceFile != "" {
		g.logger.Debug("Adding GPU driver as event source for Synthesizer.")
		gpuDriver := g.synthesizer.AddDriverEventSource(NewGPUDriver, func(d TraceDriver) {
			drv := d.(*GPUDriver)
			drv.MapperPath = workloadPreset.GPUMappingFile
			drv.ReadingInterval = time.Duration(workloadPreset.GPUTraceStep) * time.Second
			drv.SessionMaxes = make(map[string]float64)

			if g.opts.LastTimestamp > 0 {
				drv.LastTimestamp = time.Unix(g.opts.LastTimestamp, 0)
			} else {
				drv.LastTimestamp = time.Time{}
			}
			drv.ExecutionMode = 1
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
		cpuDriver := g.synthesizer.AddDriverEventSource(NewCPUDriver, func(d TraceDriver) {
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

			drv.ExecutionMode = 1
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
		memDriver := g.synthesizer.AddDriverEventSource(NewMemoryDriver, func(d TraceDriver) {
			drv := d.(*MemoryDriver)
			drv.MapperPath = workloadPreset.MemMappingFile
			drv.ReadingInterval = time.Duration(workloadPreset.MemTraceStep) * time.Second
			drv.SessionMaxes = make(map[string]float64)

			if g.opts.LastTimestamp > 0 {
				drv.LastTimestamp = time.Unix(g.opts.LastTimestamp, 0)
			} else {
				drv.LastTimestamp = time.Time{}
			}

			drv.ExecutionMode = 1
			drv.DriverType = "Memory"
			drv.Rand = rand.New(rand.NewSource(workloadRegistrationRequest.Seed))

			g.logger.Debug("Created memory driver.")
		})
		go memDriver.Drive(g.ctx, workloadPreset.NormalizeTracePaths(workloadPreset.MemTraceFile)...)
	}

	g.logger.Debug("Beginning to generate workload now.")

	g.synthesizer.Synthesize(g.ctx, g.opts, consumer.WorkloadEventGeneratorCompleteChan())

	g.logger.Debug("Finished generating CSV workload.", zap.String("workload_id", maxUtilizationConsumer.GetId()))

	return nil
}

func (g *BasicWorkloadGenerator) generateWorkloadWithXmlPreset(consumer domain.EventConsumer, maxUtilizationConsumer domain.MaxUtilizationConsumer,
	workloadPreset *domain.XmlWorkloadPreset) error {

	g.ctx, g.cancelFunction = context.WithCancel(context.Background())
	defer g.cancelFunction()

	g.logger.Debug("Generating workload from XML preset.", zap.String("workload-preset-name", workloadPreset.Name), zap.String("workload-preset-key", workloadPreset.Key))
	g.synthesizer = NewSynthesizer(g.opts, workloadPreset.MaxUtilization, g.atom)
	g.synthesizer.SetEventConsumer(consumer)
	xmlEventParser := NewXMLEventParser(g.opts.TraceStep, 0, workloadPreset.XmlFilePath, g.atom)
	gpuRecords, cpuRecords, _ := xmlEventParser.Parse()

	gpuDoneChan := make(chan struct{})
	cpuDoneChan := make(chan struct{})
	errorChan := make(chan error, 2)
	g.startCpuDriverForXml(g.ctx, g.synthesizer, cpuRecords, cpuDoneChan, errorChan)
	g.startGpuDriverForXml(g.ctx, g.synthesizer, gpuRecords, gpuDoneChan, errorChan)

	go g.synthesizer.Synthesize(g.ctx, g.opts, g.driver.WorkloadEventGeneratorCompleteChan())

	if err := g.waitForCpuGpuDriversToFinish(gpuDoneChan, cpuDoneChan, errorChan); err != nil {
		g.logger.Error("Error encountered by either the CPU or GPU driver.", zap.Error(err))
		return err
	}

	g.logger.Debug("Finished generating XML workload.", zap.String("workload_id", maxUtilizationConsumer.GetId()))
	return nil
}

// Wait for just the CPU and GPU drivers to finish generating events.
func (g *BasicWorkloadGenerator) waitForCpuGpuDriversToFinish(gpuDoneChan chan struct{}, cpuDoneChan chan struct{}, errorChan <-chan error) error {
	gpuDone := false
	cpuDone := false

	for !gpuDone || !cpuDone {
		select {
		case <-gpuDoneChan:
			g.logger.Debug("GPU TraceDriver finished.\n")
			gpuDone = true
		case <-cpuDoneChan:
			g.logger.Debug("CPU TraceDriver finished.\n")
			cpuDone = true
		case err := <-errorChan:
			g.logger.Error("Received error from one of the CPU/GPU drivers.", zap.Error(err))
			return err
		}
	}

	return nil
}

func (g *BasicWorkloadGenerator) GeneratePresetWorkload(consumer domain.EventConsumer, workload domain.MaxUtilizationConsumer,
	workloadPreset *domain.WorkloadPreset, workloadRegistrationRequest *domain.WorkloadRegistrationRequest) error {

	if workload == nil {
		panic("Workload cannot be nil when the workload generator is running.")
	}

	if workloadPreset == nil {
		panic("Workload preset cannot be nil when the workload generator is running for a template-based workload.")
	}

	if workloadRegistrationRequest == nil {
		panic("Workload registration request cannot be nil when the workload generator is running.")
	}

	if workloadPreset.IsCsv() {
		return g.generateWorkloadWithCsvPreset(consumer, workload, &workloadPreset.CsvWorkloadPreset, workloadRegistrationRequest)
	} else {
		return g.generateWorkloadWithXmlPreset(consumer, workload, &workloadPreset.XmlWorkloadPreset)
	}
}

func (g *BasicWorkloadGenerator) GenerateTemplateWorkload(consumer domain.EventConsumer, workloadSessions []*domain.WorkloadTemplateSession,
	workloadRegistrationRequest *domain.WorkloadRegistrationRequest) error {

	if workloadSessions == nil {
		panic("Workload sessions cannot be nil when the workload generator is running for a template-based workload.")
	}

	if workloadRegistrationRequest == nil {
		panic("Workload registration request cannot be nil when the workload generator is running.")
	}

	var cpuSessionMap, memSessionMap = make(map[string]float64), make(map[string]float64)
	var gpuSessionMap = make(map[string]int)
	var cpuTaskMap, memTaskMap = make(map[string][]float64), make(map[string][]float64)
	var gpuTaskMap = make(map[string][]int)
	g.ctx, g.cancelFunction = context.WithCancel(context.Background())
	defer g.cancelFunction()

	// Populate all of the above mappings using the data from the template.
	for _, session := range workloadSessions {
		cpuSessionMap[session.GetId()] = session.GetCurrentResourceRequest().Cpus
		memSessionMap[session.GetId()] = session.GetCurrentResourceRequest().MemoryMB
		gpuSessionMap[session.GetId()] = session.GetCurrentResourceRequest().Gpus

		if session.GetTrainings() == nil {
			panic(fmt.Sprintf("The `Trainings` field of Session %s is nil.", session.GetId()))
		}

		if len(session.GetTrainings()) == 0 {
			// g.sugaredLogger.Warnf("Session %s has no trainings associated with it.", session.GetId())
			continue
		}

		// Iterate over each of the training events for the session to populate the per-training-event maps.
		cpuPerTrainingTask := make([]float64, len(session.GetTrainings()))
		memPerTrainingTask := make([]float64, len(session.GetTrainings()))
		numGpusPerTrainingTask := make([]int, len(session.GetTrainings()))
		for trainingIndex, trainingEvent := range session.GetTrainings() {
			cpuPerTrainingTask[trainingIndex] = trainingEvent.Millicpus
			memPerTrainingTask[trainingIndex] = trainingEvent.MemUsageMB
			numGpusPerTrainingTask[trainingIndex] = len(trainingEvent.GpuUtil)
		}

		cpuTaskMap[session.GetId()] = cpuPerTrainingTask
		memTaskMap[session.GetId()] = memPerTrainingTask
		gpuTaskMap[session.GetId()] = numGpusPerTrainingTask
	}

	sequencer := NewCustomEventSequencer(consumer, 0, 60, g.atom)

	generatorFunc, err := ManySessionsManyTrainingEvents(workloadSessions)

	if err != nil {
		g.logger.Error("Failed to apply template.", zap.Error(err))
		consumer.GetErrorChan() <- err
		return err
	}

	err = generatorFunc(sequencer, g.logger)
	if err != nil {
		g.logger.Error("Error occurred while executing generator function for template-based workload.", zap.Error(err))
		consumer.GetErrorChan() <- err
		return err
	}

	return nil
}

func (g *BasicWorkloadGenerator) getSessionGpuMap(filePath string, adjustGpuReservations bool) (map[string]int, error) {
	gpuSessionMap := make(map[string]int)

	g.sugaredLogger.Debugf("Parsing `MaxSessionGpuFile` \"%v\"\n.", filePath)
	gpuSessionFile, err := os.Open(filePath)
	if err != nil {
		g.sugaredLogger.Errorf(fmt.Sprintf("Failed to open `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err))
		return nil, err
	}
	defer gpuSessionFile.Close()

	var sessions []*SessionMaxGpu
	if err := gocsv.UnmarshalFile(gpuSessionFile, &sessions); err != nil { // Load session data from file
		g.sugaredLogger.Debugf("Failed to parse `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, session := range sessions {
		numGPUsFloat, err := strconv.ParseFloat(session.NumGPUs, 64)
		if err != nil {
			g.sugaredLogger.Debugf("Failed to parse SessionNumGPUs value \"%v\" for Pod %v. Error: %v\n", session.NumGPUs, session.SessionID, err)
			panic(err)
		}

		// Depending on whether we're supposed to convert the number of GPUs using the utilization value -- do so (or don't).
		var numGPUs int
		if adjustGpuReservations {
			maxGpuUtilization, err := strconv.ParseFloat(session.MaxUtilization, 64)
			if err != nil {
				g.sugaredLogger.Debugf("Failed to parse maxGpuUtilization value \"%v\" for Pod %v. Error: %v\n", session.MaxUtilization, session.SessionID, err)
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

func (g *BasicWorkloadGenerator) getSessionMemMap(filePath string) map[string]float64 {
	memorySessionMap := make(map[string]float64)
	g.sugaredLogger.Debugf("Parsing `MaxSessionMemFile` \"%v\"\n.", filePath)
	memSessionFile, err := os.Open(filePath)
	if err != nil {
		g.sugaredLogger.Errorf(fmt.Sprintf("Failed to open `MaxSessionMemFile` \"%v\": %v\n.", filePath, err))
	}
	defer memSessionFile.Close()

	var sessions []*SessionMaxMemory
	if err := gocsv.UnmarshalFile(memSessionFile, &sessions); err != nil { // Load session data from file
		g.sugaredLogger.Debugf("Failed to parse `MaxSessionMemFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxSessionMemFile` \"%v\": %v\n.", filePath, err))
	}
	for _, session := range sessions {
		maxMemory, err := strconv.ParseFloat(session.MaxMemoryBytes, 64)
		if err != nil {
			g.sugaredLogger.Debugf("Failed to parse MaxSessionMem value \"%v\" for Pod %v. Error: %v\n", session.MaxMemoryBytes, session.SessionID, err)
			panic(err)
		}
		maxMem := math.Max(math.Ceil(maxMemory/1.0e9), 1)
		memorySessionMap[session.SessionID] = maxMem
	}

	return memorySessionMap
}

func (g *BasicWorkloadGenerator) getSessionCpuMap(filePath string) map[string]float64 {
	cpuSessionMap := make(map[string]float64)
	g.sugaredLogger.Debugf("Parsing `MaxSessionCpuFile` \"%v\"\n.", filePath)
	MaxSessionCpuFile, err := os.Open(filePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to open `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err))
	}
	defer MaxSessionCpuFile.Close()

	var sessions []*SessionMaxCpu
	if err := gocsv.UnmarshalFile(MaxSessionCpuFile, &sessions); err != nil { // Load session data from file
		g.sugaredLogger.Debugf("Failed to parse `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, session := range sessions {
		// g.sugaredLogger.Debugf("Session[Pod=%v, GPUs=%v, GPUUtilMax=%v, MaxSessionCPU=%v]\n", session.Pod, session.GPUs, session.GPUUtilMax, session.MaxSessionCPU)
		maxCpuUtilPercentage, err := strconv.ParseFloat(session.MaxUtilization, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
		if err != nil {
			g.sugaredLogger.Debugf("Failed to parse MaxSessionCPU value \"%v\" for Pod %v. Error: %v\n", session.MaxUtilization, session.SessionID, err)
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

func (g *BasicWorkloadGenerator) getTrainingTaskCpuMap(filePath string) map[string][]float64 {
	cpuTrainingTaskMap := make(map[string][]float64)
	g.sugaredLogger.Debugf("Parsing `MaxTrainingTaskCpuFile` \"%v\"\n.", filePath)
	MaxTrainingTaskCpuFile, err := os.Open(filePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to open `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err))
	}
	defer MaxTrainingTaskCpuFile.Close()

	var trainingMaxes []*TrainingTaskMaxCpu
	if err := gocsv.UnmarshalFile(MaxTrainingTaskCpuFile, &trainingMaxes); err != nil { // Load session data from file
		g.sugaredLogger.Debugf("Failed to parse `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, trainingMax := range trainingMaxes {
		maxCpuUtilPercentage, err := strconv.ParseFloat(trainingMax.MaxUtilization, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
		if err != nil {
			g.sugaredLogger.Debugf("Failed to parse MaxTaskCPU value \"%v\" for Pod %s, Seq#: %s. Error: %v\n", trainingMax.MaxUtilization, trainingMax.SessionID, trainingMax.TrainingTaskNum, err)
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

func (g *BasicWorkloadGenerator) getTrainingTaskGpuMap(filePath string, adjustGpuReservations bool) map[string][]int {
	gpuTrainingTaskMap := make(map[string][]int)

	g.sugaredLogger.Debugf("Parsing `MaxTrainingTaskGpuFile` \"%v\"\n.", filePath)
	gpuTrainingTaskFile, err := os.Open(filePath)
	if err != nil {
		g.sugaredLogger.Errorf(fmt.Sprintf("Failed to open `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err))
	}
	defer gpuTrainingTaskFile.Close()

	var trainingTasks []*TrainingTaskMaxGpu
	if err := gocsv.UnmarshalFile(gpuTrainingTaskFile, &trainingTasks); err != nil { // Load session data from file
		g.sugaredLogger.Debugf("Failed to parse `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err))
	}
	for _, trainingTask := range trainingTasks {
		numGPUsFloat, err := strconv.ParseFloat(trainingTask.NumGPUs, 64)
		if err != nil {
			g.sugaredLogger.Debugf("Failed to parse TrainingNumGPUs value \"%v\" for Pod %v. Error: %v\n", trainingTask.NumGPUs, trainingTask.SessionID, err)
			panic(err)
		}

		// Depending on whether we're supposed to convert the number of GPUs using the utilization value -- do so (or don't).
		var numGPUs int
		if adjustGpuReservations {
			maxGpuUtilization, err := strconv.ParseFloat(trainingTask.MaxUtilization, 64)
			if err != nil {
				g.sugaredLogger.Debugf("Failed to parse maxGpuUtilization value \"%v\" for Pod %v. Error: %v\n", trainingTask.MaxUtilization, trainingTask.SessionID, err)
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

func (g *BasicWorkloadGenerator) getTrainingTaskMemMap(filePath string) map[string][]float64 {
	memTrainingTaskMap := make(map[string][]float64)
	g.sugaredLogger.Debugf("Parsing `MaxTrainingTaskMemFile` \"%v\"\n.", filePath)
	MaxTrainingTaskMemFile, err := os.Open(filePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to open `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err))
	}
	defer MaxTrainingTaskMemFile.Close()

	var trainingMaxes []*TrainingTaskMemory
	if err := gocsv.UnmarshalFile(MaxTrainingTaskMemFile, &trainingMaxes); err != nil { // Load session data from file
		g.sugaredLogger.Debugf("Failed to parse `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err)
		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err))
	}
	for _, trainingMax := range trainingMaxes {
		maxMemory, err := strconv.ParseFloat(trainingMax.MaxMemoryBytes, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
		if err != nil {
			g.sugaredLogger.Debugf("Failed to parse MaxTaskMemory value \"%v\" for Pod %v, Seq#: %s. Error: %v\n", trainingMax.MaxMemoryBytes, trainingMax.SessionID, trainingMax.TrainingTaskNum, err)
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
