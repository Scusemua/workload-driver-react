package generator

import (
	"container/heap"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

var (
	globalCustomEventIndex atomic.Uint64
)

type sessionMetaWrapper struct {
	session *SessionMeta

	gpu       *GPUUtil
	cpu       *CPUUtil
	memBuffer *MemoryUtilBuffer
}

type CustomEventSequencer struct {
	sessions      map[string]*sessionMetaWrapper
	eventHeap     internalEventHeap
	eventConsumer domain.EventConsumer

	log      *zap.Logger
	sugarLog *zap.SugaredLogger

	tickDurationSeconds int64                 // The number of seconds in a tick.
	startingSeconds     int64                 // The start time for the event sequence as the number of seconds.
	podMap              map[string]int        // Map from SessionID to PodID
	waitingEvents       map[string]*eventImpl // The event that will be submitted/enqueued once the next commit happens.

	// sessionEventIndexes is a map from session ID to the current event localIndex for the session.
	// See the localIndex field of eventImpl for a description of what the "event localIndex" is.
	// The current entry for a particular session is the localIndex of the next event to be created.
	// That is, when creating the next event, its localIndex field should be set to the current
	// entry in the sessionEventIndexes map (using the associated session's ID as the key).
	sessionEventIndexes map[string]int
}

func NewCustomEventSequencer(eventConsumer domain.EventConsumer, startingSeconds int64, tickDurationSeconds int64, atom *zap.AtomicLevel) *CustomEventSequencer {
	customEventSequencer := &CustomEventSequencer{
		sessions:            make(map[string]*sessionMetaWrapper),
		sessionEventIndexes: make(map[string]int),
		waitingEvents:       make(map[string]*eventImpl),
		eventHeap:           internalEventHeap(make([]*internalEventHeapElement, 0, 100)),
		podMap:              make(map[string]int),
		eventConsumer:       eventConsumer,
		startingSeconds:     startingSeconds,
		tickDurationSeconds: tickDurationSeconds,
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	customEventSequencer.log = logger
	customEventSequencer.sugarLog = logger.Sugar()

	customEventSequencer.sugarLog.Debugf("Created new CustomEventSequencer with startingSeconds=%d and tickDurationSeconds=%d.", startingSeconds, tickDurationSeconds)

	return customEventSequencer
}

func (s *CustomEventSequencer) SubmitEvents(workloadGenerationCompleteChan chan interface{}) {
	s.sugarLog.Debugf("Submitting events (in a separate goroutine) now.")
	go func() {
		for s.eventHeap.Len() > 0 {
			e := heap.Pop(&s.eventHeap).(*internalEventHeapElement)
			s.eventConsumer.SubmitEvent(e.Event)
			s.sugarLog.Debugf("Submitted event #%d '%s' targeting session '%s' [%v]. EventID=%s.",
				e.Event.SessionSpecificEventIndex(), e.Event.Name(), e.Event.Data().(*SessionMeta).Pod, e.Event.Timestamp(), e.Event.Id())
		}

		workloadGenerationCompleteChan <- struct{}{}
	}()
}

func (s *CustomEventSequencer) RegisterSession(sessionId string, maxCPUs float64, maxMem float64, maxGPUs int, podIdx int) {
	if _, ok := s.sessions[sessionId]; ok {
		panic(fmt.Sprintf("Cannot register session %s; session was same ID already exists!", sessionId))
	}

	session := &SessionMeta{
		Pod:              sessionId,
		MaxSessionCPUs:   maxCPUs,
		MaxSessionMemory: maxMem,
		MaxSessionGPUs:   maxGPUs,
	}

	wrappedSession := &sessionMetaWrapper{
		session: session,
	}

	s.podMap[sessionId] = podIdx
	s.sessions[sessionId] = wrappedSession
	s.sessionEventIndexes[sessionId] = 0

	s.sugarLog.Debugf("Registered session \"%s\". MaxCPUs: %.2f, MaxMemory: %.2f, MaxGPUs: %d.", sessionId, maxCPUs, maxMem, maxGPUs)
}

func (s *CustomEventSequencer) getSessionMeta(sessionId string) *SessionMeta {
	var (
		wrappedSession *sessionMetaWrapper
		ok             bool
	)

	if wrappedSession, ok = s.sessions[sessionId]; !ok {
		panic(fmt.Sprintf("Could not find session with specified ID: \"%s\". Has this session been registered yet?", sessionId))
	}

	return wrappedSession.session
}

func (s *CustomEventSequencer) getWrappedSession(sessionId string) *sessionMetaWrapper {
	var (
		wrappedSession *sessionMetaWrapper
		ok             bool
	)

	if wrappedSession, ok = s.sessions[sessionId]; !ok {
		panic(fmt.Sprintf("Could not find session wrapper with specified ID: \"%s\". Has this session been registered yet?", sessionId))
	}

	return wrappedSession
}

func (s *CustomEventSequencer) stepCpu(sessionId string, timestamp time.Time, cpuUtil float64) {
	wrappedSession := s.getWrappedSession(sessionId)

	cpu := wrappedSession.cpu
	podIdx, ok := s.podMap[sessionId]
	if !ok {
		panic(fmt.Sprintf("Cannot find PodIDX for Session \"%s\"", sessionId))
	}

	record := &CPURecord{
		Timestamp: UnixTime(timestamp),
		Pod:       sessionId,
		PodIdx:    podIdx,
		Value:     cpuUtil,
	}
	committed := cpu.Debug_CommitAndInit(record)
	wrappedSession.session.CPU = committed
}

func (s *CustomEventSequencer) stepGpu(sessionId string, timestamp time.Time, gpuUtil []domain.GpuUtilization) {
	wrappedSession := s.getWrappedSession(sessionId)

	gpu := wrappedSession.gpu
	podIdx, ok := s.podMap[sessionId]
	if !ok {
		panic(fmt.Sprintf("Cannot find PodIDX for Session \"%s\"", sessionId))
	}

	var committed *GPUUtil
	for gpuIdx, gpuUtil := range gpuUtil {
		record := &GPURecord{
			Timestamp: UnixTime(timestamp),
			Pod:       sessionId,
			PodIdx:    podIdx,
			Value:     gpuUtil.Utilization,
			GPUIdx:    fmt.Sprintf("%d", gpuIdx),
		}

		if gpuIdx == 0 {
			committed = gpu.DebugCommitAndInit(record)
		} else {
			gpu.DebugUpdate(record)
		}
	}

	wrappedSession.session.GPU = committed
}

func (s *CustomEventSequencer) stepMemory(sessionId string, timestamp time.Time, memUtil float64) {
	wrappedSession := s.getWrappedSession(sessionId)

	memBuffer := wrappedSession.memBuffer
	podIdx, ok := s.podMap[sessionId]
	if !ok {
		panic(fmt.Sprintf("Cannot find PodIDX for Session \"%s\"", sessionId))
	}

	record := &Memory{
		Timestamp: UnixTime(timestamp),
		Pod:       sessionId,
		PodIdx:    podIdx,
		Value:     memUtil,
	}
	nextUtil := memBuffer.Debug_Init(record)
	currentUtil := memBuffer.Debug_Commit(nextUtil)
	wrappedSession.session.Memory = currentUtil
}

func (s *CustomEventSequencer) AddSessionStartedEvent(sessionId string, tickNumber int, cpuUtil float64, memUtil float64, gpuUtil float64, numGPUs int) {
	sec := s.startingSeconds + (int64(tickNumber) * s.tickDurationSeconds)
	timestamp := time.Unix(sec, 0)
	session := s.getWrappedSession(sessionId)
	podIdx, ok := s.podMap[sessionId]
	if !ok {
		panic(fmt.Sprintf("Cannot find PodIDX for Session \"%s\"", sessionId))
	}

	session.cpu = &CPUUtil{
		Timestamp:     timestamp,
		Pod:           sessionId,
		Value:         cpuUtil,
		Max:           session.session.MaxSessionCPUs,
		MaxTaskCPU:    session.session.MaxSessionCPUs,
		MaxSessionCPU: session.session.MaxSessionCPUs,
		Status:        CPUIdle,
		Repeat:        0,
	}
	session.cpu.Debug_SetPrototypeSelf()
	session.session.CPU = session.cpu

	gpu := &GPUUtil{
		Pod:     sessionId,
		GPUName: AnyGPU,
	}
	gpuRecord := &GPURecord{
		Timestamp: UnixTime(timestamp),
		GPUIdx:    "0",
		Value:     gpuUtil,
		Pod:       sessionId,
		PodIdx:    podIdx,
	}
	gpu.DebugInitialize(gpuRecord)

	for i := 1; i < numGPUs; i++ {
		nextGpuRecord := &GPURecord{
			Timestamp: UnixTime(timestamp),
			GPUIdx:    fmt.Sprintf("%d", i),
			Value:     gpuUtil,
			Pod:       sessionId,
			PodIdx:    podIdx,
		}
		gpu.DebugUpdate(nextGpuRecord)
	}
	// Just commit and init with the same record we used before.
	// We'll overwrite the values later.
	session.gpu = gpu.DebugCommitAndInit(gpuRecord)
	session.session.GPU = session.gpu

	session.memBuffer = &MemoryUtilBuffer{}
	memRecord := &Memory{
		Timestamp: UnixTime(timestamp),
		Pod:       sessionId,
		Value:     memUtil,
		PodIdx:    podIdx,
	}
	nextUtil := session.memBuffer.Debug_Init(memRecord)
	session.memBuffer.Debug_Commit(nextUtil)

	localIndex := s.sessionEventIndexes[sessionId]
	s.sessionEventIndexes[sessionId] = localIndex + 1
	evt := &eventImpl{
		name:                domain.EventSessionReady,
		eventSource:         nil,
		originalEventSource: nil,
		timestamp:           timestamp,
		localIndex:          localIndex,
		id:                  uuid.New().String(),
		globalIndex:         globalCustomEventIndex.Add(1),
	}
	s.waitingEvents[sessionId] = evt

	s.log.Debug("Adding session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", domain.EventSessionReady.String()),
		zap.Time("timestamp", timestamp),
		zap.Int64("second", sec),
		zap.Int("local_index", evt.localIndex),
		zap.Uint64("global_index", evt.globalIndex),
		zap.String("session_id", evt.id))
}

func (s *CustomEventSequencer) AddSessionTerminatedEvent(sessionId string, tickNumber int) {
	sec := s.startingSeconds + (int64(tickNumber) * s.tickDurationSeconds)
	timestamp := time.Unix(sec, 0)
	sessionMeta := s.getSessionMeta(sessionId)

	s.stepCpu(sessionId, timestamp, 0)
	s.stepGpu(sessionId, timestamp, []domain.GpuUtilization{{Utilization: 0}})
	s.stepMemory(sessionId, timestamp, 0)

	s.submitWaitingEvent(sessionMeta)

	// Step again just to commit the 0 util entries that were initialized above.
	s.stepCpu(sessionId, timestamp, 0)
	s.stepGpu(sessionId, timestamp, []domain.GpuUtilization{{Utilization: 0}})
	s.stepMemory(sessionId, timestamp, 0)

	metadata := sessionMeta.Snapshot()
	eventIndex := s.sessionEventIndexes[sessionId]
	s.sessionEventIndexes[sessionId] = eventIndex + 1
	evt := &eventImpl{
		name:                domain.EventSessionStopped,
		eventSource:         nil,
		originalEventSource: nil,
		data:                metadata,
		timestamp:           timestamp,
		localIndex:          eventIndex,
		id:                  uuid.New().String(),
		globalIndex:         globalCustomEventIndex.Add(1),
	}

	heap.Push(&s.eventHeap, &internalEventHeapElement{evt, -1})
	s.sugarLog.Debugf("Added 'stopped' event for Session %s with timestamp %v (sec=%d). EventID=%s.", sessionId, timestamp, sec, evt.id)

	s.log.Debug("Added session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", domain.EventSessionStopped.String()),
		zap.Time("timestamp", timestamp),
		zap.Int64("second", sec),
		zap.Int("local_index", evt.localIndex),
		zap.Uint64("global_index", evt.globalIndex),
		zap.String("event_id", evt.id),
		zap.String("metadata", metadata.String()))
}

func (s *CustomEventSequencer) submitWaitingEvent(sessionMeta *SessionMeta) {
	sessionId := sessionMeta.Pod
	dataForWaitingEvent := sessionMeta.Snapshot()
	s.waitingEvents[sessionId].data = dataForWaitingEvent
	evt := s.waitingEvents[sessionId]
	heap.Push(&s.eventHeap, &internalEventHeapElement{evt, -1})

	s.log.Debug("Adding session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", evt.name.String()),
		zap.Time("timestamp", evt.timestamp),
		zap.Int64("order_seq", evt.orderSeq),
		zap.Int("local_index", evt.localIndex),
		zap.Uint64("global_index", evt.globalIndex),
		zap.String("session_id", evt.id))

	delete(s.waitingEvents, sessionId)
}

// AddTrainingEvent registers a training event for a particular session.
//
// Parameters:
// - sessionId: The target Session's ID
// - duration: The duration that the training should last.
func (s *CustomEventSequencer) AddTrainingEvent(sessionId string, tickNumber int, durationInTicks int, cpuUtil float64, memUtil float64, gpuUtil []domain.GpuUtilization) {
	startSec := s.startingSeconds + (int64(tickNumber) * s.tickDurationSeconds)
	startTime := time.Unix(startSec, 0)
	sessionMeta := s.getSessionMeta(sessionId)

	s.stepCpu(sessionId, startTime, cpuUtil)
	s.stepGpu(sessionId, startTime, gpuUtil)
	s.stepMemory(sessionId, startTime, memUtil)

	s.submitWaitingEvent(sessionMeta)

	endSec := s.startingSeconds + (int64(tickNumber+durationInTicks) * s.tickDurationSeconds)
	endTime := time.Unix(endSec, 0)

	s.stepCpu(sessionId, endTime, 0)
	s.stepGpu(sessionId, endTime, gpuUtil)
	s.stepMemory(sessionId, endTime, 0)

	eventIndex := s.sessionEventIndexes[sessionId]
	metadata := sessionMeta.Snapshot()
	trainingStartedEvent := &eventImpl{
		name:                domain.EventSessionTrainingStarted,
		eventSource:         nil,
		originalEventSource: nil,
		data:                metadata,
		localIndex:          eventIndex,
		timestamp:           startTime,
		id:                  uuid.New().String(),
		globalIndex:         globalCustomEventIndex.Add(1),
	}
	heap.Push(&s.eventHeap, &internalEventHeapElement{trainingStartedEvent, -1})
	s.log.Debug("Added session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", domain.EventSessionTrainingStarted.String()),
		zap.Time("timestamp", startTime),
		zap.Int64("second", startSec),
		zap.Int("local_index", trainingStartedEvent.localIndex),
		zap.Uint64("global_index", trainingStartedEvent.globalIndex),
		zap.String("event_id", trainingStartedEvent.id),
		zap.String("metadata", metadata.String()))

	trainingEndedEvent := &eventImpl{
		name:                domain.EventSessionTrainingEnded,
		eventSource:         nil,
		originalEventSource: nil,
		data:                nil,
		localIndex:          eventIndex + 1,
		timestamp:           endTime,
		id:                  uuid.New().String(),
		globalIndex:         globalCustomEventIndex.Add(1),
	}
	s.waitingEvents[sessionId] = trainingEndedEvent
	s.sessionEventIndexes[sessionId] = eventIndex + 2
}

type internalEventHeap []*internalEventHeapElement

type internalEventHeapElement struct {
	domain.Event

	// heapIndex is the index of the internalEventHeapElement within its containing internalEventHeap.
	// heapIndex is distinct from the domain.Event's SessionSpecificEventIndex and GlobalEventIndex methods.
	// heapIndex is dynamic and will change depending on what position the internalEventHeapElement is in
	// within the containing internalEventHeap, whereas SessionSpecificEventIndex and GlobalEventIndex are/return
	// static values that are set when the underlying domain.Event is first created.
	heapIndex int
}

func (e *internalEventHeapElement) Idx() int {
	return e.heapIndex
}

func (e *internalEventHeapElement) SetIndex(idx int) {
	e.heapIndex = idx
}

func (h internalEventHeap) Len() int {
	return len(h)
}

func (h internalEventHeap) Less(i, j int) bool {
	// We want to ensure that TrainingEnded events are processed before SessionStopped events.
	// So, if the event at localIndex i is a TrainingEnded event while the event at localIndex j is a SessionStopped event,
	// then the event at localIndex i should be processed first.
	if h[i].Timestamp().Equal(h[j].Timestamp()) {
		if h[i].Name() == domain.EventSessionTrainingEnded && h[j].Name() == domain.EventSessionStopped {
			if h[i].SessionSpecificEventIndex() /* training-ended */ > h[j].SessionSpecificEventIndex() /* session-stopped */ {
				// We expect the global localIndex of the training-ended event to be less than that of the session-stopped
				// event, since the training-ended event should have been created prior to the session-stopped event.
				log.Fatalf("Global event indices do not reflect correct ordering of events. "+
					"TrainingEnded: %s. SessionStopped: %s.", h[i].String(), h[j].String())
			}

			return true
		} else if h[j].Name() == domain.EventSessionTrainingEnded && h[i].Name() == domain.EventSessionStopped {
			if h[j].SessionSpecificEventIndex() /* training-ended */ > h[i].SessionSpecificEventIndex() /* session-stopped */ {
				// We expect the global localIndex of the training-ended event to be less than that of the session-stopped
				// event, since the training-ended event should have been created prior to the session-stopped event.
				log.Fatalf("Global event indices do not reflect correct ordering of events. "+
					"TrainingEnded: %s. SessionStopped: %s.", h[j].String(), h[i].String())
			}

			return false
		}

		return h[i].GlobalEventIndex() < h[j].GlobalEventIndex()
	}

	return h[i].Timestamp().Before(h[j].Timestamp())
}

func (h internalEventHeap) Swap(i, j int) {
	// log.Printf("Swap %d, %d (%v, %v) of %d", i, j, h[i], h[j], len(h))
	h[i].SetIndex(j)
	h[j].SetIndex(i)
	h[i], h[j] = h[j], h[i]
}

func (h *internalEventHeap) Push(x interface{}) {
	x.(*internalEventHeapElement).SetIndex(len(*h))
	*h = append(*h, x.(*internalEventHeapElement))
}

func (h *internalEventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h internalEventHeap) Peek() *internalEventHeapElement {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
