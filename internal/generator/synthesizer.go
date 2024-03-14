package generator

import (
	"container/heap"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

type SynthesizerEvent string

func (evt SynthesizerEvent) String() string {
	return string(evt)
}

// Because traces are sampling in different intervals, so the Synthesizer needs to "normalize"
// the intervals/time series so that all of the eventsHeap are processed in the correct order.
// This event gives a chance to get the correct timestamp.
const (
	EventSynthesizerTick SynthesizerEvent = "tick"
)

type MaxUtilizationWrapper struct {
	CpuSessionMap map[string]float64 // Maximum CPU utilization achieved during each Session's lifetime.
	MemSessionMap map[string]float64 // Maximum memory used (in gigabytes) during each Session's lifetime.
	GpuSessionMap map[string]int     // Maximum number of GPUs used during each Session's lifetime.

	CurrentTrainingNumberMap map[string]int // Map from SessionID to the current training task number we're on (beginning with 0, then 1, then 2, ..., etc.)

	CpuTaskMap map[string][]float64 // Maximum CPU utilization achieved during each training event for each Session, arranged in chronological order of training events.
	MemTaskMap map[string][]float64 // Maximum memory used (in GB) during each training event for each Session, arranged in chronological order of training events.
	GpuTaskMap map[string][]int     // Maximum number of GPUs used during each training event for each Session, arranged in chronological order of training events.
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

type Synthesizer struct {
	Sources        []domain.EventSource
	GenericSources []domain.EventSource // Non-driver EventSources. Added to `Sources` after first Driver-generated event is processed.
	Tick           int64

	log      *zap.Logger
	sugarLog *zap.SugaredLogger

	drivingCPU bool
	drivingGPU bool
	drivingMem bool

	consumer              domain.EventConsumer
	bufferedEvents        chan domain.Event
	eventsChannel         chan domain.Event
	eventsHeap            SimpleEventHeap
	numActiveSources      uint64
	maxUtilizationWrapper *MaxUtilizationWrapper

	executionMode int
	// Per synthsizing fields
	sessions   map[string]*Session
	lastTickTs int64
	// firstEventTs int64
}

func NewSynthesizer(opts *domain.Configuration, maxUtilizationWrapper *MaxUtilizationWrapper) *Synthesizer {
	if opts.ExecutionMode == 1 {
		if maxUtilizationWrapper.MemSessionMap == nil {
			panic("The Synthesizer's per-session max-memory map should not be nil during a standard (i.e., non-pre-run) simulation.")
		}

		if maxUtilizationWrapper.CpuSessionMap == nil {
			panic("The Synthesizer's per-session max-CPU map should not be nil during a standard (i.e., non-pre-run) simulation.")
		}

		if maxUtilizationWrapper.GpuSessionMap == nil {
			panic("The Synthesizer's per-session max-GPU map should not be nil during a standard (i.e., non-pre-run) simulation.")
		}
	}

	synthesizer := &Synthesizer{
		// Sources:          make([]Driver, 0, 2),
		bufferedEvents:        make(chan domain.Event),
		eventsHeap:            make(SimpleEventHeap, 0, 1),
		eventsChannel:         make(chan domain.Event),
		numActiveSources:      0,
		maxUtilizationWrapper: maxUtilizationWrapper,
		executionMode:         opts.ExecutionMode,
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	synthesizer.log = logger
	synthesizer.sugarLog = logger.Sugar()

	return synthesizer
}

// Add a generic EventSource (i.e., not necessarily a Driver).
func (s *Synthesizer) AddEventSource(evtSource domain.EventSource) domain.EventSource {
	if s.GenericSources == nil {
		s.GenericSources = make([]domain.EventSource, 0, 2)
	}

	s.GenericSources = append(s.GenericSources, evtSource)

	return evtSource
}

// Add a Driver as an domain.EventSource to the Synthesizer.
func (s *Synthesizer) AddDriverEventSource(create NewDriver, configs ...func(Driver)) Driver {
	if s.Sources == nil {
		s.Sources = make([]domain.EventSource, 0, 2)
	}

	id := len(s.Sources)
	var eventSource domain.EventSource = create(id, configs...)
	s.Sources = append(s.Sources, eventSource)

	s.sugarLog.Debug("Added new event source to driver: %v.", eventSource)

	switch reflect.TypeOf(s.Sources[id]).Elem().String() {
	case "generator.CPUDriver":
		s.drivingCPU = true
		s.log.Debug("Synthesizer will be receiving CPU events.")
	case "generator.GPUDriver":
		s.drivingGPU = true
		s.log.Debug("Synthesizer will be receiving GPU events.")
	case "generator.MemoryDriver":
		s.drivingMem = true
		s.log.Debug("Synthesizer will be receiving memory events.")
	default:
		s.sugarLog.Error("Unexpected/unsupported type of event source added to Synthesizer. Type: \"%s\"", reflect.TypeOf(s.Sources[id]).Elem().String())
		panic(fmt.Sprintf("Unexpected/unsupported type of event source added to Synthesizer. Type: \"%s\"", reflect.TypeOf(s.Sources[id]).Elem().String()))
	}

	return s.Sources[id].(Driver)
}

func (s *Synthesizer) SetEventConsumer(c domain.EventConsumer) {
	s.consumer = c
}

func (s *Synthesizer) CpuSessionMap() map[string]float64 {
	return s.maxUtilizationWrapper.CpuSessionMap
}

func (s *Synthesizer) MemSessionMap() map[string]float64 {
	return s.maxUtilizationWrapper.MemSessionMap
}

func (s *Synthesizer) GpuSessionMap() map[string]int {
	return s.maxUtilizationWrapper.GpuSessionMap
}

func (s *Synthesizer) CpuTrainingTaskMap() map[string][]float64 {
	return s.maxUtilizationWrapper.CpuTaskMap
}

func (s *Synthesizer) MemTrainingTaskMap() map[string][]float64 {
	return s.maxUtilizationWrapper.MemTaskMap
}

func (s *Synthesizer) GpuTrainingTaskMap() map[string][]int {
	return s.maxUtilizationWrapper.GpuTaskMap
}

func (s *Synthesizer) CurrentTrainingNumberMap() map[string]int {
	return s.maxUtilizationWrapper.CurrentTrainingNumberMap
}

// Handle the latest event (in chronological order) generated by the Workload Generator.
func (s *Synthesizer) transitionAndSubmitEvent(evt domain.Event) {
	if podData, ok := evt.Data().(domain.PodData); ok {
		sess, initted := s.sessions[podData.GetPod()]

		if evt.Name() == EventGpuUpdateUtil {
			if !initted || sess.Status != SessionTraining {
				return
			}
		}

		if !initted {
			var maxCPUs, maxMem float64
			var maxGPUs int
			var noCpuEntry, noMemoryEntry, noGpuEntry bool

			if s.executionMode == 1 {
				// CPU is stored in SimulationDriver::CpuSessionMap as the number of vCPUs,
				// which is calculated by rounding-up the maximum utilization achieved by the session.
				if s.CpuSessionMap() != nil {
					if maxCPUs, ok = s.CpuSessionMap()[podData.GetPod()]; !ok {
						s.sugarLog.Error("No data in CPU Session Map for pod %s.", podData.GetPod())
						maxCPUs = 0
						noCpuEntry = true
					}
				}

				if s.MemSessionMap() != nil {
					// Memory is stored in SimulationDriver::MemSessionMap as GB values.
					if maxMem, ok = s.MemSessionMap()[podData.GetPod()]; !ok {
						s.sugarLog.Error("No data in Memory Session Map for pod %s.", podData.GetPod())
						maxMem = 0
						noMemoryEntry = true
					}
				}

				if s.GpuSessionMap() != nil {
					if maxGPUs, ok = s.GpuSessionMap()[podData.GetPod()]; !ok {
						s.sugarLog.Error("No data in GPU Session Map for pod %s.", podData.GetPod())
						maxGPUs = 0
						noGpuEntry = true
					}
				}

				if noCpuEntry && noMemoryEntry && noGpuEntry {
					s.sugarLog.Error("The maximum resource values for CPUs, GPU, Memory are all 0 for Session %s. Skipping.", podData.GetPod())
					return
				}
			} else {
				// Default values.
				maxCPUs = 1
				maxMem = 1
				maxGPUs = 1
			}

			sess = &Session{
				Pod:              podData.GetPod(),
				MaxSessionCPUs:   maxCPUs,
				MaxSessionMemory: maxMem,
				MaxSessionGPUs:   maxGPUs,
			}
			s.sessions[podData.GetPod()] = sess
		}
		triggered, err := sess.Transit(evt, false)
		if err != nil {
			s.log.Warn(err.Error())
			return
		} else if len(triggered) == 0 {
			return
		}
		for _, evtName := range triggered {
			if s.executionMode == 1 {
				eventData := sess.Snapshot()

				trainingIdx := s.CurrentTrainingNumberMap()[sess.Pod]

				eventData.CurrentTrainingMaxCPUs = s.CpuTrainingTaskMap()[sess.Pod][trainingIdx]
				eventData.CurrentTrainingMaxMemory = s.MemTrainingTaskMap()[sess.Pod][trainingIdx]
				eventData.CurrentTrainingMaxGPUs = s.GpuTrainingTaskMap()[sess.Pod][trainingIdx]

				if evtName == EventSessionTrainingStarted {
					if len(s.CpuTrainingTaskMap()[sess.Pod]) <= (trainingIdx + 1) {
						s.sugarLog.Warn("Cannot incr training idx for Session %s. len(CpuTrainingTaskMap): %d. Training index: %d", sess.Pod, len(s.CpuTrainingTaskMap()[sess.Pod]), trainingIdx)
					} else if len(s.MemTrainingTaskMap()[sess.Pod]) <= (trainingIdx + 1) {
						s.sugarLog.Warn("Cannot incr training idx for Session %s. len(MemTrainingTaskMap): %d. Training index: %d", sess.Pod, len(s.MemTrainingTaskMap()[sess.Pod]), trainingIdx)
					} else if len(s.GpuTrainingTaskMap()[sess.Pod]) <= (trainingIdx + 1) {
						s.sugarLog.Warn("Cannot incr training idx for Session %s. len(GpuTrainingTaskMap): %d. Training index: %d", sess.Pod, len(s.GpuTrainingTaskMap()[sess.Pod]), trainingIdx)
					} else {
						s.CurrentTrainingNumberMap()[sess.Pod] = trainingIdx + 1
					}
				}

				sessEvt := &eventImpl{name: evtName, eventSource: evt.EventSource(), originalEventSource: evt.OriginalEventSource(), data: eventData, timestamp: sess.Timestamp, id: uuid.New().String()}
				s.sugarLog.Debug("Enqueuing Session-level event targeting pod %s: %s [ts=%v]", sessEvt.Data().(*Session).Pod, evtName, sess.Timestamp)
				s.consumer.SubmitEvent(sessEvt)
			} else {
				switch evtName {
				case EventSessionTrainingStarted:
					for _, evtSrc := range s.Sources {
						evtSrc.TrainingStarted(sess.Pod)
					}
				case EventSessionTrainingEnded:
					for _, evtSrc := range s.Sources {
						evtSrc.TrainingEnded(sess.Pod)
					}
				default:
					// Do nothing.
				}
			}

			if s.executionMode == 0 {
				fmt.Printf("Latest Event Timestamp: %v\x1b[1G", sess.Timestamp)
			}
		}
		return
	}
}

func (s *Synthesizer) Synthesize(ctx context.Context, opts *domain.Configuration, workloadSimulatorDoneChan chan struct{}) { // , clusterDoneChan chan struct{}
	simulation_start := time.Now()

	s.Tick = opts.TraceStep

	s.eventsHeap = make(SimpleEventHeap, 0, len(s.Sources))
	if s.drivingCPU && s.drivingGPU {
		s.sessions = make(map[string]*Session, 1000)
	}
	s.lastTickTs = 0
	defer func() {
		s.sessions = nil
	}()

	numDriversFinished := 0

	// Establish the heap
	for i := 0; i < len(s.Sources); i++ {
		s.sugarLog.Debug("Pushing event source %d '%v' onto events heap.", (i + 1), s.Sources[i])
		heap.Push(&s.eventsHeap, <-s.Sources[i].OnEvent())
	}

	// Exclude empty sources
	startEvt := s.eventsHeap.Peek()
	for startEvt != nil && startEvt.Name() == EventNoMore {
		s.log.Warn(startEvt.String())
		// The source is empty
		heap.Pop(&s.eventsHeap)
		startEvt = s.eventsHeap.Peek()
	}

	// Looking for recent eventsHeap
	start := time.Now()
	skipped := time.Duration(0)
	// firstEventProcessed := false
	for s.eventsHeap.Len() > 0 {
		evt := s.eventsHeap.Peek()
		switch evt.Name() {
		case EventError:
			if evt.Data() != context.Canceled {
				log.Error(evt.String())
			}
			fallthrough
		case EventNoMore:
			// The source of most recent event is drained or there is an error. Pop the source
			heap.Pop(&s.eventsHeap)
			if evt.Name() == EventNoMore {
				sugarLog.Info("%v, %d sources left.", evt, s.eventsHeap.Len())
			} else {
				sugarLog.Debug("%v, %d sources left.", evt, s.eventsHeap.Len())
			}

			switch evt.EventSource().(type) {
			case Driver:
				numDriversFinished += 1
			}
			continue
		}

		planned := time.Now()
		timeToNext := evt.Timestamp().Sub(startEvt.Timestamp()) - skipped - planned.Sub(start)

		// Skip idle time
		skipped += timeToNext

		// Dispatch event
		s.transitionAndSubmitEvent(evt)

		// Refill event
		select {
		case s.eventsHeap[0] = <-s.Sources[evt.EventSource().Id()].OnEvent():
			// Abnormal eventsHeap (nomore or error) has no timestamp(0), so calling fix will be ok.
			heap.Fix(&s.eventsHeap, 0)
		case <-ctx.Done():
			return
		}
	}

	s.sugarLog.Infof("Finished consuming events from drivers. Workload generation is done. Time elapsed: %v.\n\n", time.Since(simulation_start))

	if s.executionMode == 1 {
		workloadSimulatorDoneChan <- struct{}{}
		s.log.Info("Informed the Simulation Driver that the simulation has ended.")
	}
}

type SimpleEventHeap []domain.Event

func (h SimpleEventHeap) Len() int {
	return len(h)
}

func (h SimpleEventHeap) Less(i, j int) bool {
	return h[i].Timestamp().Before(h[j].Timestamp())
}

func (h SimpleEventHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *SimpleEventHeap) Push(x interface{}) {
	*h = append(*h, x.(domain.Event))
}

func (h *SimpleEventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h SimpleEventHeap) Peek() domain.Event {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
