package generator

import (
	"container/heap"
	"context"
	"fmt"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap/zapcore"
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
//const (
//	EventSynthesizerTick SynthesizerEvent = "tick"
//)

type Synthesizer struct {
	Sources        []domain.EventSource
	GenericSources []domain.EventSource // Non-driver EventSources. Added to `Sources` after first TraceDriver-generated event is processed.
	Tick           int64

	log      *zap.Logger
	sugarLog *zap.SugaredLogger

	drivingCPU bool
	drivingGPU bool
	drivingMem bool

	// sessionIdMapping is used if session IDs are longer than 36 characters.
	sessionIdMapping map[string]string

	consumer              domain.EventConsumer
	bufferedEvents        chan domain.Event
	eventsChannel         chan domain.Event
	eventsHeap            domain.EventHeap
	numActiveSources      uint64
	maxUtilizationWrapper *domain.MaxUtilizationWrapper

	executionMode int
	// Per synthsizing fields
	sessions   map[string]*SessionMeta
	lastTickTs int64
	// firstEventTs int64
}

func NewSynthesizer(opts *domain.Configuration, maxUtilizationWrapper *domain.MaxUtilizationWrapper, atom *zap.AtomicLevel) *Synthesizer {
	// if opts.ExecutionMode == 1 {
	if maxUtilizationWrapper.MemSessionMap == nil {
		panic("The Synthesizer's per-session max-memory map should not be nil during a standard (i.e., non-pre-run) simulation.")
	}

	if maxUtilizationWrapper.CpuSessionMap == nil {
		panic("The Synthesizer's per-session max-CPU map should not be nil during a standard (i.e., non-pre-run) simulation.")
	}

	if maxUtilizationWrapper.GpuSessionMap == nil {
		panic("The Synthesizer's per-session max-GPU map should not be nil during a standard (i.e., non-pre-run) simulation.")
	}
	// }

	synthesizer := &Synthesizer{
		// Sources:          make([]TraceDriver, 0, 2),
		bufferedEvents:        make(chan domain.Event),
		eventsHeap:            make(domain.EventHeap, 0, 1),
		eventsChannel:         make(chan domain.Event),
		numActiveSources:      0,
		maxUtilizationWrapper: maxUtilizationWrapper,
		sessionIdMapping:      make(map[string]string),
		executionMode:         1,
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	synthesizer.log = logger
	synthesizer.sugarLog = logger.Sugar()

	return synthesizer
}

// AddEventSource adds a generic EventSource (i.e., not necessarily a TraceDriver).
func (s *Synthesizer) AddEventSource(evtSource domain.EventSource) domain.EventSource {
	if s.GenericSources == nil {
		s.GenericSources = make([]domain.EventSource, 0, 2)
	}

	s.GenericSources = append(s.GenericSources, evtSource)

	return evtSource
}

// AddDriverEventSource adds a TraceDriver as an domain.EventSource to the Synthesizer.
func (s *Synthesizer) AddDriverEventSource(create NewDriver, configs ...func(TraceDriver)) TraceDriver {
	if s.Sources == nil {
		s.Sources = make([]domain.EventSource, 0, 2)
	}

	id := len(s.Sources)
	var eventSource domain.EventSource = create(id, configs...)
	s.Sources = append(s.Sources, eventSource)

	s.log.Debug("Added new event source to driver: %v.", zap.Any("event_source", eventSource))

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
		s.log.Error("Unexpected/unsupported type of event source added to Synthesizer.",
			zap.String("event_source_type", reflect.TypeOf(s.Sources[id]).Elem().String()))
		panic(fmt.Sprintf("Unexpected/unsupported type of event source added to Synthesizer. Type: \"%s\"",
			reflect.TypeOf(s.Sources[id]).Elem().String()))
	}

	return s.Sources[id].(TraceDriver)
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

func (s *Synthesizer) VramSessionMap() map[string]float64 {
	return s.maxUtilizationWrapper.VramSessionMap
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

// handleEventPreprocessMode is used to handle an event by the Synthesizer when executionMode is 0 (i.e., pre-process).
func (s *Synthesizer) handleEventPreprocessMode(evtName domain.SessionEventName, sess *SessionMeta) {
	switch evtName {
	case domain.EventSessionTrainingStarted:
		for _, evtSrc := range s.Sources {
			evtSrc.TrainingStarted(sess.Pod)
		}
	case domain.EventSessionTrainingEnded:
		for _, evtSrc := range s.Sources {
			evtSrc.TrainingEnded(sess.Pod)
		}
	default:
		// Do nothing.
	}
}

func (s *Synthesizer) handleEventStandard(evt *domain.Event, triggeredEventName domain.SessionEventName, sess *SessionMeta) {
	eventData := sess.Snapshot()

	trainingIdx := s.CurrentTrainingNumberMap()[sess.Pod]

	eventData.CurrentTrainingMaxCPUs = s.CpuTrainingTaskMap()[sess.Pod][trainingIdx]
	eventData.CurrentTrainingMaxMemory = s.MemTrainingTaskMap()[sess.Pod][trainingIdx]

	gpuMap := s.GpuTrainingTaskMap()[sess.Pod]
	if len(gpuMap) == 0 {
		if trainingIdx == 0 {
			eventData.CurrentTrainingMaxGPUs = 0
		} else {
			panic(fmt.Sprintf("Training #%d for Session %s, but we have no max GPU training task data for that session.", trainingIdx+1, sess.Pod))
		}
	} else {
		eventData.CurrentTrainingMaxGPUs = s.GpuTrainingTaskMap()[sess.Pod][trainingIdx]
	}

	if triggeredEventName == domain.EventSessionTrainingStarted {
		if len(s.CpuTrainingTaskMap()[sess.Pod]) <= (trainingIdx + 1) {
			s.sugarLog.Warnf("Cannot incr training heapIndex for Session %s. len(CpuTrainingTaskMap): %d. Training localIndex: %d",
				sess.Pod, len(s.CpuTrainingTaskMap()[sess.Pod]), trainingIdx)
		} else if len(s.MemTrainingTaskMap()[sess.Pod]) <= (trainingIdx + 1) {
			s.sugarLog.Warnf("Cannot incr training heapIndex for Session %s. len(MemTrainingTaskMap): %d. Training localIndex: %d",
				sess.Pod, len(s.MemTrainingTaskMap()[sess.Pod]), trainingIdx)
		} else if len(s.GpuTrainingTaskMap()[sess.Pod]) <= (trainingIdx + 1) {
			s.sugarLog.Warnf("Cannot incr training heapIndex for Session %s. len(GpuTrainingTaskMap): %d. Training localIndex: %d",
				sess.Pod, len(s.GpuTrainingTaskMap()[sess.Pod]), trainingIdx)
		} else {
			s.CurrentTrainingNumberMap()[sess.Pod] = trainingIdx + 1
		}
	}

	if len(eventData.Pod) > 36 {
		originalPod := eventData.Pod
		newPod := uuid.NewString()
		eventData.Pod = newPod
		s.sessionIdMapping[originalPod] = newPod
	}

	sessEvt := &domain.Event{
		Name:                triggeredEventName,
		EventSource:         evt.EventSource,
		OriginalEventSource: evt.OriginalEventSource,
		Data:                eventData,
		Timestamp:           eventData.Timestamp,
		ID:                  uuid.New().String(),
		OriginalTimestamp:   eventData.Timestamp,
		SessionId:           eventData.Pod,
	}

	s.consumer.SubmitEvent(sessEvt)
}

func (s *Synthesizer) initSession(evt *domain.Event, podData domain.PodData) *SessionMeta {
	var (
		maxCPUs, maxMem, maxVRAM                  float64
		maxGPUs                                   int
		noCpuEntry, noMemoryEntry, noGpuEntry, ok bool
	)

	if s.executionMode == 1 {
		// CPU is stored in SimulationDriver::CpuSessionMap as the number of vCPUs,
		// which is calculated by rounding-up the maximum utilization achieved by the session.
		if s.CpuSessionMap() != nil {
			if maxCPUs, ok = s.CpuSessionMap()[podData.GetPod()]; !ok {
				s.log.Warn("No data in CPU Session Map for pod.", zap.String("session_id", podData.GetPod()))
				maxCPUs = 0
				noCpuEntry = true
			}
		}

		if s.MemSessionMap() != nil {
			// Memory is stored in SimulationDriver::MemSessionMap as GB values.
			if maxMem, ok = s.MemSessionMap()[podData.GetPod()]; !ok {
				s.log.Warn("No data in Memory Session Map for pod.", zap.String("session_id", podData.GetPod()))
				maxMem = 0
				noMemoryEntry = true
			}
		}

		if s.GpuSessionMap() != nil {
			if maxGPUs, ok = s.GpuSessionMap()[podData.GetPod()]; !ok {
				s.log.Warn("No data in GPU Session Map for pod",
					zap.String("session_id", podData.GetPod()),
					zap.Int("gpu_session_map_length", len(s.GpuSessionMap())))
				maxGPUs = 0
				noGpuEntry = true
			}
		}

		if s.VramSessionMap() != nil {
			if maxVRAM, ok = s.VramSessionMap()[podData.GetPod()]; !ok {
				s.log.Warn("No data in VRAM Session Map for pod.",
					zap.String("session_id", podData.GetPod()),
					zap.Int("gpu_session_map_length", len(s.GpuSessionMap())))
				maxVRAM = 0
				noGpuEntry = true
			}
		}

		if noCpuEntry && noMemoryEntry && noGpuEntry {
			s.log.Warn("The maximum resource values for CPUs, GPU, Memory are all 0 for Session. Skipping.",
				zap.String("session_id", podData.GetPod()))
			return nil
		}
	} else {
		// Default values.
		maxCPUs = 1
		maxMem = 128
		maxGPUs = 1
		maxVRAM = 0.128
	}

	sess := &SessionMeta{
		Pod:              podData.GetPod(),
		MaxSessionCPUs:   maxCPUs,
		MaxSessionMemory: maxMem,
		MaxSessionGPUs:   maxGPUs,
		MaxSessionVRAM:   maxVRAM,
	}
	s.sessions[podData.GetPod()] = sess

	return sess
}

// Handle the latest event (in chronological order) generated by the Workload Generator.
func (s *Synthesizer) transitionAndSubmitEvent(evt *domain.Event) {
	if podData, ok := evt.Data.(domain.PodData); ok {
		sess, initted := s.sessions[podData.GetPod()]

		if evt.Name == EventGpuUpdateUtil && (!initted || sess.Status != SessionStatusTraining) {
			return
		}

		if !initted {
			s.log.Debug("Initializing session.",
				zap.String("event_name", evt.Name.String()),
				zap.String("event_id", evt.ID))
			sess = s.initSession(evt, podData)

			if sess == nil {
				s.log.Warn("Session was not initialized.",
					zap.String("event_name", evt.Name.String()),
					zap.String("event_id", evt.ID))

				return
			}
		}

		triggered, err := sess.Transit(evt)

		if err != nil {
			s.log.Error("Error while transitioning event.",
				zap.String("session_id", evt.SessionID()),
				zap.String("event_name", evt.Name.String()),
				zap.String("event_id", evt.ID),
				zap.Error(err))
			return
		}

		if len(triggered) == 0 {
			//s.log.Debug("Triggered 0 events after transitioning event.",
			//	zap.String("session_id", evt.SessionID()),
			//	zap.String("event_name", evt.Name.String()),
			//	zap.String("event_id", evt.ID),
			//	zap.Error(err))
			return
		}

		//s.log.Debug("Preparing to submit triggered event(s) after transitioning event.",
		//	zap.String("session_id", evt.SessionID()),
		//	zap.String("event_name", evt.Name.String()),
		//	zap.String("event_id", evt.ID),
		//	zap.Int("num_triggered", len(triggered)),
		//	zap.Error(err))

		for _, evtName := range triggered {
			if s.executionMode == 1 {
				s.handleEventStandard(evt, evtName, sess)
			} else {
				s.handleEventPreprocessMode(evtName, sess)
				fmt.Printf("Latest Event Timestamp: %v\x1b[1G", sess.Timestamp)
			}
		}

		return
	}

	//s.log.Warn("Event had unexpected data.",
	//	zap.String("event_name", evt.Name.String()),
	//	zap.String("event_id", evt.ID),
	//	zap.Any("event", evt))
}

func (s *Synthesizer) Synthesize(ctx context.Context, opts *domain.Configuration, workloadGenerationCompleteChan chan interface{}) { // , clusterDoneChan chan struct{}
	simulationStart := time.Now()

	s.Tick = opts.TraceStep

	s.eventsHeap = make(domain.EventHeap, 0, len(s.Sources))
	if s.drivingCPU && s.drivingGPU {
		s.sessions = make(map[string]*SessionMeta, 1000)
	}
	s.lastTickTs = 0
	defer func() {
		s.sessions = nil
	}()

	s.log.Debug("Synthesizing workload now.",
		zap.Int("num_event_sources", len(s.Sources)))

	// Establish the heap
	for i := 0; i < len(s.Sources); i++ {
		heap.Push(&s.eventsHeap, <-s.Sources[i].OnEvent())
	}

	// Exclude empty sources
	startEvt := s.eventsHeap.Peek()
	for startEvt != nil && startEvt.Name == EventNoMore {
		// The source is empty
		src := heap.Pop(&s.eventsHeap)
		s.log.Warn("Removed empty event source.",
			zap.Any("event_source", src))
		startEvt = s.eventsHeap.Peek()
	}

	s.log.Debug("Processing events now.")

	// Looking for recent eventsHeap
	start := time.Now()
	skipped := time.Duration(0)
	// firstEventProcessed := false
	for s.eventsHeap.Len() > 0 {
		evt := s.eventsHeap.Peek()

		switch evt.Name {
		case EventError:
			if evt.Data != context.Canceled {
				s.log.Error(evt.String())
			}
			fallthrough
		case EventNoMore:
			// The source of most recent event is drained or there is an error. Pop the source
			heap.Pop(&s.eventsHeap)
			if evt.Name == EventNoMore {
				s.log.Info("Received EventNoMore.",
					zap.Int("num_sources_left", s.eventsHeap.Len()),
					zap.Any("event", evt))
			} else {
				s.log.Info("Received Event.",
					zap.Int("num_sources_left", s.eventsHeap.Len()),
					zap.String("event_name", evt.Name.String()),
					zap.String("event_id", evt.ID),
					zap.Time("event_timestamp", evt.Timestamp))
			}

			continue
		}

		planned := time.Now()
		timeToNext := evt.Timestamp.Sub(startEvt.Timestamp) - skipped - planned.Sub(start)

		// Skip idle time
		skipped += timeToNext

		//s.log.Debug("Transitioning and submitting event.",
		//	zap.String("event_name", evt.Name.String()),
		//	zap.String("event_id", evt.ID),
		//	zap.Time("event_timestamp", evt.Timestamp))

		// Dispatch event
		s.transitionAndSubmitEvent(evt)

		// Refill event
		select {
		case s.eventsHeap[0] = <-s.Sources[evt.EventSource.Id()].OnEvent():
			// Abnormal eventsHeap (nomore or error) has no timestamp(0), so calling fix will be ok.
			heap.Fix(&s.eventsHeap, 0)
		case <-ctx.Done():
			s.sugarLog.Warn("Synthesizer has been stopped. ctx.Err: %v", ctx.Err())
			return
		}
	}

	s.log.Info("Finished consuming events from drivers. Workload generation is done.",
		zap.Duration("time_elapsed", time.Since(simulationStart)))

	if s.executionMode == 1 {
		workloadGenerationCompleteChan <- struct{}{}
		s.log.Info("Informed the Workload Driver that the generator has finished generating events.")
	}
}
