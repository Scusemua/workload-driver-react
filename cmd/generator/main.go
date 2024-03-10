package main

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"log"
// 	"math"
// 	"math/rand"
// 	_ "net/http"
// 	"os"
// 	"os/signal"
// 	"path/filepath"
// 	"runtime/debug"
// 	_ "runtime/debug"

// 	_ "runtime/pprof"
// 	"strconv"
// 	"strings"
// 	"sync"
// 	"syscall"
// 	"time"

// 	"github.com/fatih/color"
// 	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
// 	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
// 	"github.com/scusemua/workload-driver-react/m/v2/internal/server/driver"
// 	"github.com/zhangjyr/gocsv"
// 	"go.uber.org/zap"

// 	_ "net/http/pprof"
// )

// type NotUsed struct {
// 	Name string
// }

// // Pod,CPUUtilMax,GPUs,GPUUtilMax,MaxSessionCPU,MemRate(T-1),MemRate,MemRate(T+1)
// // Deprecated.
// type SessionData struct {
// 	Pod              string  `csv:"Pod"`
// 	CPUUtilMax       string  `csv:"CPUUtilMax"`
// 	GPUs             string  `csv:"GPUs"`
// 	GPUUtilMax       string  `csv:"GPUUtilMax"`
// 	MaxSessionCPU    string  `csv:"MaxSessionCPU"`
// 	MaxSessionMemory string  `csv:"MaxSessionMemory"`
// 	NotUsed1         NotUsed `csv:"-"`
// 	NotUsed2         NotUsed `csv:"-"`
// 	NotUsed3         NotUsed `csv:"-"`
// }

// // Pod,Seq,Start,Elapsed,MaxTaskCPU,Mem(T-1),Mem,Mem(T+1),LastStartMem(T-1),LastStartMem,LastStartMem(T+1)
// // Deprecated.
// type TaskData struct {
// 	Pod        string  `csv:"Pod"`
// 	Seq        string  `csv:"Seq"`
// 	Start      string  `csv:"Start"`
// 	Elapsed    string  `csv:"Elapsed"`
// 	MaxTaskCPU string  `csv:"MaxTaskCPU"`
// 	NotUsed1   NotUsed `csv:"-"`
// 	NotUsed2   NotUsed `csv:"-"`
// 	NotUsed3   NotUsed `csv:"-"`
// 	NotUsed4   NotUsed `csv:"-"`
// 	NotUsed5   NotUsed `csv:"-"`
// 	NotUsed6   NotUsed `csv:"-"`
// }

// // Used to unmarshall CSV file containing max session CPU.
// // The file is in format task_id,max_cpu
// type SessionMaxCpu struct {
// 	SessionID      string `csv:"session_id"`          // The session's ID.
// 	MaxUtilization string `csv:"max_cpu_utilization"` // Maximum CPU utilization of the session.
// }

// // Used to unmarshall CSV file containing max session memory.
// // The file is in format task_id,max_memory
// type SessionMaxMemory struct {
// 	SessionID      string `csv:"session_id"`       // The session's ID.
// 	MaxMemoryBytes string `csv:"max_memory_bytes"` // Maximum memory (in bytes) used by the session.
// }

// // Used to unmarshall CSV file containing max session GPUs.
// // The file is in format session_id,max_gpu_utilization,num_gpus
// type SessionMaxGpu struct {
// 	SessionID      string `csv:"session_id"`          // The session's ID.
// 	MaxUtilization string `csv:"max_gpu_utilization"` // Maximum GPU utilization of the session.
// 	NumGPUs        string `csv:"num_gpus"`            // Number of GPUs used by the session. We may convert this to another value (by multiplying it by `MaxUtilization`), if configured to do so.
// }

// // Used to unmarshall CSV file containing max session CPU.
// // The file is in format task_id,max_cpu
// type TrainingTaskMaxCpu struct {
// 	SessionID       string `csv:"session_id"`          // The session's ID.
// 	TrainingTaskNum string `csv:"seq"`                 // ID representing the chronological order of the training event (0 is first, 1 is second, etc.)
// 	MaxUtilization  string `csv:"max_cpu_utilization"` // Maximum CPU utilization of the session.
// }

// // Used to unmarshall CSV file containing max session memory.
// // The file is in format task_id,max_memory
// type TrainingTaskMemory struct {
// 	SessionID       string `csv:"session_id"`    // The session's ID.
// 	TrainingTaskNum string `csv:"seq"`           // ID representing the chronological order of the training event (0 is first, 1 is second, etc.)
// 	MaxMemoryBytes  string `csv:"max_mem_bytes"` // Maximum memory (in bytes) used by the session.
// }

// // Used to unmarshall CSV file containing max session GPUs.
// // The file is in format session_id,max_gpu_utilization,num_gpus
// type TrainingTaskMaxGpu struct {
// 	SessionID       string `csv:"session_id"`          // The session's ID.
// 	TrainingTaskNum string `csv:"seq"`                 // ID representing the chronological order of the training event (0 is first, 1 is second, etc.)
// 	MaxUtilization  string `csv:"max_gpu_utilization"` // Maximum GPU utilization of the session.
// 	NumGPUs         string `csv:"num_gpus"`            // Number of GPUs used by the session. We may convert this to another value (by multiplying it by `MaxUtilization`), if configured to do so.
// }

// // In the mapping of tasks to their max CPU, the key is the Pod and the Seq,
// // since there are multiple tasks per pod. The exact format of this key is
// // as follows: "<Pod_ID>-<Seq_Number>"
// func (td TaskData) GetTaskMapKey() string {
// 	return fmt.Sprintf("%s-%s", td.Pod, td.Seq)
// }

// func main() {
// 	start_time := time.Now()
// 	logger, err := zap.NewDevelopment()
// 	sugaredLogger := logger.Sugar()
// 	if err != nil {
// 		panic(err)
// 	}

// 	var logOutputFile *os.File
// 	defer func() {
// 		if r := recover(); r != nil {
// 			fmt.Printf("%v\n", r)
// 			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
// 		}

// 		if logOutputFile != nil {
// 			logOutputFile.Close()
// 		}
// 	}()
// 	rand.Seed(time.Now().Unix())

// 	sig := make(chan os.Signal, 1)
// 	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT)

// 	var wg sync.WaitGroup

// 	// Assign some default values for certain configuration parameters.
// 	options := &domain.Configuration{
// 		TraceStep:                         60,
// 		MaxTaskDurationSec:                300, // 5 minutes.
// 		ResourceCreditCPU:                 1,
// 		ResourceCreditGPU:                 1,
// 		ServerfulUserCostMultiplier:       "1.0",
// 		ServerfulProviderCostMultiplier:   "1.0",
// 		ServerlessUserCostMultiplier:      "1.0",
// 		ServerlessProviderCostMultiplier:  "1.0",
// 		ResourceCreditMemMB:               1024,
// 		OutputSessions:                    "", // By default, don't profile sessions.
// 		InitialCreditBalance:              100,
// 		ContinueUntilExplicitlyTerminated: false,
// 		ResourceCreditCostInUSD:           0.0999, // You can purchase 100 compute units for Google Collab for $9.99, so each compute unit is roughly $0.10. For now, we assume the same pricing model.
// 		AdjustGpuReservations:             false,  // Default to false.
// 		ServerlessInstanceTypeSelector:    "smallest-gpu",
// 		ScalingInterval:                   40,
// 		ScalingOutEnaled:                  true,
// 		ScalingBufferSize:                 4,
// 	}

// 	if options.YAML != "" {
// 		log.Printf("YAML configuration file: \"%s\"\n", options.YAML)
// 	}

// 	domain.Config = options
// 	options.CheckUsage()

// 	outputSubdir := time.Now().Format("01-02-2006 15:04:05")
// 	outputSubdir = strings.ReplaceAll(outputSubdir, ":", "-")
// 	outputSubdirectoryPath := filepath.Join("./output/", outputSubdir)
// 	log.Printf("Output subdirectory for this simulation: \"%s\"\n", outputSubdirectoryPath)
// 	err = os.MkdirAll(outputSubdirectoryPath, os.ModePerm)

// 	if options.YAML != "" {
// 		log.Printf("YAML configuration file: \"%s\"\n", options.YAML)

// 		// Copy configuration files. First, the main YAML configuration file.
// 		srcFile, err := os.Open(options.YAML)
// 		if err != nil {
// 			log.Fatalf("Could not open source YAML configuration file \"%s\" for copying. Error: %v\n", options.YAML, err)
// 		}

// 		// Create the destination file for writing
// 		mainConfigCopyFilePath := filepath.Join(outputSubdirectoryPath, filepath.Base(options.YAML))
// 		dstFile, err := os.Create(mainConfigCopyFilePath)
// 		if err != nil {
// 			log.Fatalf("Could not create destination YAML configuration file \"%s\" for copying. Error: %v\n", mainConfigCopyFilePath, err)
// 		}

// 		_, err = io.Copy(dstFile, srcFile)
// 		if err != nil {
// 			log.Fatalf("Failed to copy file \"%s\" to destination \"%s\". Error: %v\n", srcFile.Name(), dstFile.Name(), err)
// 		}
// 	}

// 	if options.ExecutionMode == 0 {
// 		fmt.Printf("%s%s%s\n", color.CyanString("Running in "), color.RedString("PREPROCESSING"), color.CyanString(" mode."))
// 	} else {
// 		fmt.Printf("%s%s%s\n", color.CyanString("Running in "), color.RedString("STANDARD"), color.CyanString(" mode."))
// 	}

// 	// if options.CpuProfileFile != "" {
// 	// 	cpuProfileFullPath := filepath.Join(outputSubdirectoryPath, options.CpuProfileFile)
// 	// 	f, err := os.Create(cpuProfileFullPath)
// 	// 	if err != nil {
// 	// 		log.Printf("Could not create CPU profile \"%s\"", cpuProfileFullPath)
// 	// 		panic(err)
// 	// 	}
// 	// 	defer f.Close() // error handling omitted for example
// 	// 	if err := pprof.StartCPUProfile(f); err != nil {
// 	// 		log.Fatal("Could not start CPU profile: ", err)
// 	// 	}
// 	// 	defer pprof.StopCPUProfile()
// 	// }

// 	if err != nil {
// 		panic(err)
// 	}

// 	if options.LogOutputFile != "" {
// 		logOutputPath := filepath.Join(outputSubdirectoryPath, options.LogOutputFile)
// 		fmt.Printf("Will output logs to \"%s\"\n.", logOutputPath)
// 		logOutputFile, err = os.OpenFile(logOutputPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
// 		if err != nil {
// 			log.Fatalf("error opening file: %v", err)
// 		}
// 		defer logOutputFile.Close()

// 		log.SetOutput(logOutputFile)
// 	}

// 	if options.Seed == 0 {
// 		options.Seed = time.Now().UnixNano()
// 	}
// 	defer finalize(options.Seed)

// 	////////////////////////////////////////////////////////////////////////
// 	// Load per-Session maximum CPU, GPU, and memory utilizations.        //
// 	// Load per-training-event maximum CPU, GPU, and memory utilizations. //
// 	////////////////////////////////////////////////////////////////////////
// 	var cpuSessionMap, memSessionMap map[string]float64 = nil, nil
// 	var gpuSessionMap map[string]int = nil
// 	var cpuTaskMap, memTaskMap map[string][]float64 = nil, nil
// 	var gpuTaskMap map[string][]int = nil
// 	if options.ExecutionMode == 1 {
// 		// Per-Session max CPU.
// 		if options.MaxSessionCpuFile != "" {
// 			cpuSessionMap = getSessionCpuMap(options.MaxSessionCpuFile)
// 		}

// 		// Per-Session max memory.
// 		if options.MaxSessionMemFile != "" {
// 			memSessionMap = getSessionMemMap(options.MaxSessionMemFile)
// 		}

// 		// Per-Session max number of GPUs.
// 		if options.MaxSessionGpuFile != "" {
// 			gpuSessionMap = getSessionGpuMap(options.MaxSessionGpuFile, options.AdjustGpuReservations)
// 		}

// 		// Per-training-event max CPU.
// 		if options.MaxTaskCpuFile != "" {
// 			cpuTaskMap = getTrainingTaskCpuMap(options.MaxTaskCpuFile)
// 		}

// 		// Per-training-event max memory.
// 		if options.MaxTaskMemFile != "" {
// 			memTaskMap = getTrainingTaskMemMap(options.MaxTaskMemFile)
// 		}

// 		// Per-training-event max number of GPUs.
// 		if options.MaxTaskGpuFile != "" {
// 			gpuTaskMap = getTrainingTaskGpuMap(options.MaxTaskGpuFile, options.AdjustGpuReservations)
// 		}
// 	}

// 	// eventQueueService := NewEventQueueService(options)

// 	maxUtilizationWrapper := generator.NewMaxUtilizationWrapper(cpuSessionMap, memSessionMap, gpuSessionMap, cpuTaskMap, memTaskMap, gpuTaskMap)
// 	synth := generator.NewSynthesizer(options, maxUtilizationWrapper)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	go func() {
// 		<-sig
// 		logger.Info("Received signal, aborting...")
// 		cancel()
// 		os.Exit(0)
// 	}()

// 	logger.Debug("Driving GPU now.")

// 	// Drive GPU trace
// 	if options.GPUTraceFile != "" {
// 		logger.Debug("Adding GPU driver as event source for Synthesizer.")
// 		gpuDriver := synth.AddDriverEventSource(generator.NewGPUDriver, func(d generator.Driver) {
// 			drv := d.(*generator.GPUDriver)
// 			drv.MapperPath = options.GPUMappingFile
// 			drv.ReadingInterval = time.Duration(options.GPUTraceStep) * time.Second
// 			drv.SessionMaxes = make(map[string]float64)

// 			if options.LastTimestamp > 0 {
// 				drv.LastTimestamp = time.Unix(options.LastTimestamp, 0)
// 			} else {
// 				drv.LastTimestamp = time.Time{}
// 			}
// 			drv.ExecutionMode = options.ExecutionMode
// 			drv.DriverType = "GPU"
// 			drv.Rand = rand.New(rand.NewSource(options.Seed))

// 			if drv.ExecutionMode == 0 {
// 				drv.MaxSessionOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-session_gpu_maxes.csv")
// 				sugaredLogger.Info("Set 'MaxSessionOutputPath' for GPU driver to \"%s\"", drv.MaxSessionOutputPath)

// 				drv.MaxTrainingOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-training_gpu_maxes.csv")
// 				sugaredLogger.Info("Set 'MaxTrainingOutputPath' for memory driver to \"%s\"", drv.MaxTrainingOutputPath)

// 				drv.MaxPerGpuSessionOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-session_per_gpu_device_maxes.csv")
// 				sugaredLogger.Info("Set 'MaxPerGpuSessionOutputPath' for GPU driver to \"%s\"", drv.MaxPerGpuSessionOutputPath)

// 				drv.MaxPerGpuTrainingOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-training_per_gpu_device_maxes.csv")
// 				sugaredLogger.Info("Set 'MaxPerGpuTrainingOutputPath' for GPU driver to \"%s\"", drv.MaxPerGpuTrainingOutputPath)
// 			}

// 			logger.Debug("Created GPU driver.")
// 		})
// 		go gpuDriver.Drive(ctx, options.NormalizeTracePaths(options.GPUTraceFile)...)
// 	}

// 	logger.Debug("Driving CPU now.")

// 	// Drive CPU trace
// 	if options.CPUTraceFile != "" {
// 		logger.Debug("Adding CPU driver as event source for Synthesizer.")
// 		cpuDriver := synth.AddDriverEventSource(generator.NewCPUDriver, func(d generator.Driver) {
// 			drv := d.(*generator.CPUDriver)
// 			drv.MapperPath = options.CPUMappingFile
// 			drv.Downtimes = options.NormalizeDowntime(options.CPUDowntime)
// 			drv.ReadingInterval = time.Duration(options.CPUTraceStep) * time.Second
// 			drv.SessionMaxes = make(map[string]float64)

// 			if options.LastTimestamp > 0 {
// 				drv.LastTimestamp = time.Unix(options.LastTimestamp, 0)
// 			} else {
// 				drv.LastTimestamp = time.Time{}
// 			}

// 			drv.MaxSessionOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-session_cpu_maxes.csv")
// 			sugaredLogger.Info("Set 'MaxSessionOutputPath' for CPU driver to \"%s\"", drv.MaxSessionOutputPath)
// 			drv.MaxTrainingOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-training_cpu_maxes.csv")
// 			sugaredLogger.Info("Set 'MaxTrainingOutputPath' for memory driver to \"%s\"", drv.MaxTrainingOutputPath)
// 			drv.ExecutionMode = options.ExecutionMode
// 			drv.DriverType = "CPU"
// 			drv.Rand = rand.New(rand.NewSource(options.Seed))

// 			logger.Debug("Created CPU driver.")
// 		})
// 		go cpuDriver.Drive(ctx, options.NormalizeTracePaths(options.CPUTraceFile)...)
// 	}

// 	logger.Debug("Driving memory now.")

// 	// Drive memory trace
// 	if options.MemTraceFile != "" {
// 		logger.Debug("Adding memory driver as event source for Synthesizer.")
// 		generator.SessionReadyExpects |= generator.SessionMemReady
// 		memDriver := synth.AddDriverEventSource(generator.NewMemoryDriver, func(d generator.Driver) {
// 			drv := d.(*generator.MemoryDriver)
// 			drv.MapperPath = options.MemMappingFile
// 			// No downtime(s) for memory trace?
// 			// drv.Downtimes = options.NormalizeDowntime(options.MemDowntime)
// 			drv.ReadingInterval = time.Duration(options.MemTraceStep) * time.Second
// 			drv.SessionMaxes = make(map[string]float64)

// 			if options.LastTimestamp > 0 {
// 				drv.LastTimestamp = time.Unix(options.LastTimestamp, 0)
// 			} else {
// 				drv.LastTimestamp = time.Time{}
// 			}

// 			drv.MaxSessionOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-session_mem_maxes.csv")
// 			sugaredLogger.Info("Set 'MaxSessionOutputPath' for memory driver to \"%s\"", drv.MaxSessionOutputPath)
// 			drv.MaxTrainingOutputPath = filepath.Join(outputSubdirectoryPath, options.FromMonth+"-"+options.ToMonth+"-training_mem_maxes.csv")
// 			sugaredLogger.Info("Set 'MaxTrainingOutputPath' for memory driver to \"%s\"", drv.MaxTrainingOutputPath)
// 			drv.ExecutionMode = options.ExecutionMode
// 			drv.DriverType = "Memory"
// 			drv.Rand = rand.New(rand.NewSource(options.Seed))

// 			logger.Debug("Created memory driver.")
// 		})
// 		go memDriver.Drive(ctx, options.NormalizeTracePaths(options.MemTraceFile)...)
// 	}

// 	// bufferedEventService := entity.NewSimpleBufferedEventService(&options, len(synth.Sources))
// 	// Start to handle clock ticks
// 	// go bufferedEventService.ServeTick()

// 	// Synthesize events.
// 	// cluster := NewCluster(&options, cpuSessionMap, memSessionMap, gpuSessionMap, eventQueueService, &wg, outputSubdirectoryPath)
// 	// if options.ExecutionMode == 1 {
// 	// 	eventQueueService.Initialize(cluster)
// 	// 	defer cluster.Close()
// 	// 	wg.Add(1)
// 	// 	go cluster.ServeTicks()
// 	// }

// 	simulationDriver := driver.NewWorkloadDriver(options) // eventQueueService

// 	if options.ExecutionMode == 1 {
// 		go simulationDriver.DriveWorkload()

// 		// Set the cluster as the EventHandler for the Synthesizer.
// 		synth.SetEventConsumer(simulationDriver)
// 	}

// 	// Start synthesizing.
// 	synth.Synthesize(ctx, options, simulationDriver.DoneChan()) // , cluster.DoneChan()

// 	// log.Printf("Waiting for Cluster to finish before simulation can exit. There is/are %d session event(s) enqueued right now.\n", eventQueueService.Len())
// 	wg.Wait()

// 	log.Printf("Simulation complete. Time elapsed: %v\n.", time.Since(start_time))
// }

// func RoundUp(x, unit float64) float64 {
// 	rounded := math.Round(x/unit) * unit

// 	if rounded < x*unit {
// 		return rounded + unit
// 	}

// 	return rounded
// }

// // Note: if x == y, return x.
// func MinInt(x int, y int) int {
// 	if x <= y {
// 		return x
// 	}

// 	return y
// }

// // Note: if x == y, return x.
// func MaxInt(x int, y int) int {
// 	if x >= y {
// 		return x
// 	}

// 	return y
// }

// func getSessionGpuMap(filePath string, adjustGpuReservations bool) map[string]int {
// 	gpuSessionMap := make(map[string]int)

// 	fmt.Printf("Parsing `MaxSessionGpuFile` \"%v\"\n.", filePath)
// 	gpuSessionFile, err := os.Open(filePath)
// 	defer gpuSessionFile.Close()
// 	if err != nil {
// 		log.Fatalf(fmt.Sprintf("Failed to open `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err))
// 	}

// 	sessions := []*SessionMaxGpu{}
// 	if err := gocsv.UnmarshalFile(gpuSessionFile, &sessions); err != nil { // Load session data from file
// 		fmt.Printf("Failed to parse `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err)
// 		panic(fmt.Sprintf("Failed to parse `MaxSessionGpuFile` \"%v\": %v\n.", filePath, err))
// 	}
// 	for _, session := range sessions {
// 		numGPUsFloat, err := strconv.ParseFloat(session.NumGPUs, 64)
// 		if err != nil {
// 			fmt.Printf("Failed to parse SessionNumGPUs value \"%v\" for Pod %v. Error: %v\n", session.NumGPUs, session.SessionID, err)
// 			panic(err)
// 		}

// 		// Depending on whether or not we're supposed to convert the number of GPUs using the utilization value -- do so (or don't).
// 		var numGPUs int
// 		if adjustGpuReservations {
// 			maxGpuUtilization, err := strconv.ParseFloat(session.MaxUtilization, 64)
// 			if err != nil {
// 				fmt.Printf("Failed to parse maxGpuUtilization value \"%v\" for Pod %v. Error: %v\n", session.MaxUtilization, session.SessionID, err)
// 				panic(err)
// 			}

// 			// Round up to the nearest whole number.
// 			numGPUs = int(RoundUp(maxGpuUtilization/100.0, 1.0))
// 			numGPUs = MinInt(numGPUs, int(numGPUsFloat))
// 			numGPUs = MaxInt(numGPUs, 1) // If numGPUs is set to 0 because maxGpuUtilization is 0, then default to 1, as we charge users based on number of GPUs.

// 			// Validate that the number we obtained is "reasonable" (i.e., it isn't negative, nor is it greater than the maximum possible number of GPUs a session is supposed to have used).
// 			if numGPUs < 0 || numGPUs > 8 {
// 				log.Fatalf("Obtained unexpected value of %d when converting #GPUs value. NumGPUs: %.0f. Max GPU Utilization: %.2f. Session: %s.\n", numGPUs, numGPUsFloat, maxGpuUtilization, session)
// 			}
// 		} else {
// 			numGPUs = int(numGPUsFloat)
// 		}

// 		gpuSessionMap[session.SessionID] = numGPUs
// 	}

// 	return gpuSessionMap
// }

// func getSessionMemMap(filePath string) map[string]float64 {
// 	memorySessionMap := make(map[string]float64)
// 	fmt.Printf("Parsing `MaxSessionMemFile` \"%v\"\n.", filePath)
// 	memSessionFile, err := os.Open(filePath)
// 	defer memSessionFile.Close()
// 	if err != nil {
// 		log.Fatalf(fmt.Sprintf("Failed to open `MaxSessionMemFile` \"%v\": %v\n.", filePath, err))
// 	}

// 	sessions := []*SessionMaxMemory{}
// 	if err := gocsv.UnmarshalFile(memSessionFile, &sessions); err != nil { // Load session data from file
// 		fmt.Printf("Failed to parse `MaxSessionMemFile` \"%v\": %v\n.", filePath, err)
// 		panic(fmt.Sprintf("Failed to parse `MaxSessionMemFile` \"%v\": %v\n.", filePath, err))
// 	}
// 	for _, session := range sessions {
// 		maxMemory, err := strconv.ParseFloat(session.MaxMemoryBytes, 64)
// 		if err != nil {
// 			fmt.Printf("Failed to parse MaxSessionMem value \"%v\" for Pod %v. Error: %v\n", session.MaxMemoryBytes, session.SessionID, err)
// 			panic(err)
// 		}
// 		maxMem := math.Max(math.Ceil(maxMemory/1.0e9), 1)
// 		memorySessionMap[session.SessionID] = maxMem
// 	}

// 	return memorySessionMap
// }

// func getSessionCpuMap(filePath string) map[string]float64 {
// 	cpuSessionMap := make(map[string]float64)
// 	fmt.Printf("Parsing `MaxSessionCpuFile` \"%v\"\n.", filePath)
// 	MaxSessionCpuFile, err := os.Open(filePath)
// 	defer MaxSessionCpuFile.Close()
// 	if err != nil {
// 		panic(fmt.Sprintf("Failed to open `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err))
// 	}

// 	sessions := []*SessionMaxCpu{}
// 	if err := gocsv.UnmarshalFile(MaxSessionCpuFile, &sessions); err != nil { // Load session data from file
// 		fmt.Printf("Failed to parse `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err)
// 		panic(fmt.Sprintf("Failed to parse `MaxSessionCpuFile` \"%v\": %v\n.", filePath, err))
// 	}
// 	for _, session := range sessions {
// 		// fmt.Printf("Session[Pod=%v, GPUs=%v, GPUUtilMax=%v, MaxSessionCPU=%v]\n", session.Pod, session.GPUs, session.GPUUtilMax, session.MaxSessionCPU)
// 		maxCpuUtilPercentage, err := strconv.ParseFloat(session.MaxUtilization, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
// 		if err != nil {
// 			fmt.Printf("Failed to parse MaxSessionCPU value \"%v\" for Pod %v. Error: %v\n", session.MaxUtilization, session.SessionID, err)
// 			panic(err)
// 		}

// 		// Round up to nearest multiple of 100% for the utilization.
// 		// For example, 8.3% utilization goes to 100%.
// 		// 103.57% utilization goes to 200%.
// 		// Then divide that value by 100 to get the number of vCPUs required.
// 		maxCpu := math.Max(RoundUp(maxCpuUtilPercentage, 100)/100.0, 1)
// 		cpuSessionMap[session.SessionID] = maxCpu
// 	}

// 	return cpuSessionMap
// }

// func getTrainingTaskCpuMap(filePath string) map[string][]float64 {
// 	cpuTrainingTaskMap := make(map[string][]float64)
// 	fmt.Printf("Parsing `MaxTrainingTaskCpuFile` \"%v\"\n.", filePath)
// 	MaxTrainingTaskCpuFile, err := os.Open(filePath)
// 	defer MaxTrainingTaskCpuFile.Close()
// 	if err != nil {
// 		panic(fmt.Sprintf("Failed to open `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err))
// 	}

// 	trainingMaxes := []*TrainingTaskMaxCpu{}
// 	if err := gocsv.UnmarshalFile(MaxTrainingTaskCpuFile, &trainingMaxes); err != nil { // Load session data from file
// 		fmt.Printf("Failed to parse `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err)
// 		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskCpuFile` \"%v\": %v\n.", filePath, err))
// 	}
// 	for _, trainingMax := range trainingMaxes {
// 		maxCpuUtilPercentage, err := strconv.ParseFloat(trainingMax.MaxUtilization, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
// 		if err != nil {
// 			fmt.Printf("Failed to parse MaxTaskCPU value \"%v\" for Pod %s, Seq#: %s. Error: %v\n", trainingMax.MaxUtilization, trainingMax.SessionID, trainingMax.TrainingTaskNum, err)
// 			panic(err)
// 		}

// 		// Round up to nearest multiple of 100% for the utilization.
// 		// For example, 8.3% utilization goes to 100%.
// 		// 103.57% utilization goes to 200%.
// 		// Then divide that value by 100 to get the number of vCPUs required.
// 		maxCpu := math.Max(RoundUp(maxCpuUtilPercentage, 100)/100.0, 1)

// 		maxCpuUtilizations, ok := cpuTrainingTaskMap[trainingMax.SessionID]

// 		if !ok {
// 			maxCpuUtilizations = make([]float64, 0)
// 		}

// 		maxCpuUtilizations = append(maxCpuUtilizations, maxCpu)
// 		cpuTrainingTaskMap[trainingMax.SessionID] = maxCpuUtilizations
// 	}

// 	return cpuTrainingTaskMap
// }

// func getTrainingTaskGpuMap(filePath string, adjustGpuReservations bool) map[string][]int {
// 	gpuTrainingTaskMap := make(map[string][]int)

// 	fmt.Printf("Parsing `MaxTrainingTaskGpuFile` \"%v\"\n.", filePath)
// 	gpuTrainingTaskFile, err := os.Open(filePath)
// 	defer gpuTrainingTaskFile.Close()
// 	if err != nil {
// 		log.Fatalf(fmt.Sprintf("Failed to open `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err))
// 	}

// 	trainingTasks := []*TrainingTaskMaxGpu{}
// 	if err := gocsv.UnmarshalFile(gpuTrainingTaskFile, &trainingTasks); err != nil { // Load session data from file
// 		fmt.Printf("Failed to parse `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err)
// 		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskGpuFile` \"%v\": %v\n.", filePath, err))
// 	}
// 	for _, trainingTask := range trainingTasks {
// 		numGPUsFloat, err := strconv.ParseFloat(trainingTask.NumGPUs, 64)
// 		if err != nil {
// 			fmt.Printf("Failed to parse TrainingNumGPUs value \"%v\" for Pod %v. Error: %v\n", trainingTask.NumGPUs, trainingTask.SessionID, err)
// 			panic(err)
// 		}

// 		// Depending on whether or not we're supposed to convert the number of GPUs using the utilization value -- do so (or don't).
// 		var numGPUs int
// 		if adjustGpuReservations {
// 			maxGpuUtilization, err := strconv.ParseFloat(trainingTask.MaxUtilization, 64)
// 			if err != nil {
// 				fmt.Printf("Failed to parse maxGpuUtilization value \"%v\" for Pod %v. Error: %v\n", trainingTask.MaxUtilization, trainingTask.SessionID, err)
// 				panic(err)
// 			}

// 			// Round up to the nearest whole number.
// 			numGPUs = int(RoundUp(maxGpuUtilization/100.0, 1.0))
// 			numGPUs = MinInt(numGPUs, int(numGPUsFloat))
// 			numGPUs = MaxInt(numGPUs, 1) // If numGPUs is set to 0 because maxGpuUtilization is 0, then default to 1, as we charge users based on number of GPUs.

// 			// Validate that the number we obtained is "reasonable" (i.e., it isn't negative, nor is it greater than the maximum possible number of GPUs a session is supposed to have used).
// 			if numGPUs < 0 || numGPUs > 8 {
// 				log.Fatalf("Obtained unexpected value of %d when converting #GPUs value. NumGPUs: %.0f. Max GPU Utilization: %.2f. TrainingTask: %s.\n", numGPUs, numGPUsFloat, maxGpuUtilization, trainingTask)
// 			}
// 		} else {
// 			numGPUs = int(numGPUsFloat)
// 		}

// 		numGPUsList, ok := gpuTrainingTaskMap[trainingTask.SessionID]

// 		if !ok {
// 			numGPUsList = make([]int, 0)
// 		}

// 		numGPUsList = append(numGPUsList, numGPUs)
// 		gpuTrainingTaskMap[trainingTask.SessionID] = numGPUsList
// 	}

// 	return gpuTrainingTaskMap
// }

// func getTrainingTaskMemMap(filePath string) map[string][]float64 {
// 	memTrainingTaskMap := make(map[string][]float64)
// 	fmt.Printf("Parsing `MaxTrainingTaskMemFile` \"%v\"\n.", filePath)
// 	MaxTrainingTaskMemFile, err := os.Open(filePath)
// 	defer MaxTrainingTaskMemFile.Close()
// 	if err != nil {
// 		panic(fmt.Sprintf("Failed to open `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err))
// 	}

// 	trainingMaxes := []*TrainingTaskMemory{}
// 	if err := gocsv.UnmarshalFile(MaxTrainingTaskMemFile, &trainingMaxes); err != nil { // Load session data from file
// 		fmt.Printf("Failed to parse `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err)
// 		panic(fmt.Sprintf("Failed to parse `MaxTrainingTaskMemFile` \"%v\": %v\n.", filePath, err))
// 	}
// 	for _, trainingMax := range trainingMaxes {
// 		maxMemory, err := strconv.ParseFloat(trainingMax.MaxMemoryBytes, 64) // The values are 0 - 100+, where 1 vCPU is 100% util, 2 vCPU is 200% util, etc.
// 		if err != nil {
// 			fmt.Printf("Failed to parse MaxTaskMemory value \"%v\" for Pod %v, Seq#: %s. Error: %v\n", trainingMax.MaxMemoryBytes, trainingMax.SessionID, trainingMax.TrainingTaskNum, err)
// 			panic(err)
// 		}

// 		maxMem := math.Max(math.Ceil(maxMemory/1.0e9), 1)

// 		maxMemoryUsages, ok := memTrainingTaskMap[trainingMax.SessionID]

// 		if !ok {
// 			maxMemoryUsages = make([]float64, 0)
// 		}

// 		maxMemoryUsages = append(maxMemoryUsages, maxMem)
// 		memTrainingTaskMap[trainingMax.SessionID] = maxMemoryUsages
// 	}

// 	return memTrainingTaskMap
// }

// func finalize(seed int64) {
// 	fmt.Printf("Simulation ended. Reproduce with option \"-seed %d\".\n", seed)
// }
