package generator

import (
	"container/heap"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

var (
	globalCustomEventIndex atomic.Uint64
)

type sessionWrapper struct {
	session *Session

	gpu       *GPUUtil
	cpu       *CPUUtil
	memBuffer *MemoryUtilBuffer
}

type CustomEventSequencer struct {
	sessions  map[string]*sessionWrapper
	eventHeap internalEventHeap
	driver    domain.WorkloadDriver
	eqs       domain.EventQueueService

	log      *zap.Logger
	sugarLog *zap.SugaredLogger

	tickDurationSeconds int64                 // The number of seconds in a tick.
	startingSeconds     int64                 // The start time for the event sequence as the number of seconds.
	podMap              map[string]int        // Map from SessionID to PodID
	waitingEvents       map[string]*eventImpl // The event that will be submitted/enqueued once the next commit happens.
}

func NewCustomEventSequencer(driver domain.WorkloadDriver, eqs domain.EventQueueService, startingSeconds int64, tickDurationSeconds int64, atom *zap.AtomicLevel) *CustomEventSequencer {
	customEventSequencer := &CustomEventSequencer{
		sessions:            make(map[string]*sessionWrapper),
		waitingEvents:       make(map[string]*eventImpl),
		eventHeap:           internalEventHeap(make([]*internalEventHeapElement, 0, 100)),
		podMap:              make(map[string]int),
		driver:              driver,
		eqs:                 eqs,
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

func (s *CustomEventSequencer) SubmitEvents() {
	s.sugarLog.Debugf("Submitting events (in a separate goroutine) now.")
	go func() {
		for s.eventHeap.Len() > 0 {
			e := heap.Pop(&s.eventHeap).(*internalEventHeapElement)
			s.driver.SubmitEvent(e.Event)
			s.sugarLog.Debugf("Submitted event '%s' targeting session '%s' [%v]", e.Event.Name(), e.Event.Data().(*Session).Pod, e.Event.Timestamp())
		}
	}()
}

func (s *CustomEventSequencer) RegisterSession(sessionId string, maxCPUs float64, maxMem float64, maxGPUs int, podIdx int) {
	if _, ok := s.sessions[sessionId]; ok {
		panic(fmt.Sprintf("Cannot register session %s; session was same ID already exists!", sessionId))
	}

	session := &Session{
		Pod:              sessionId,
		MaxSessionCPUs:   maxCPUs,
		MaxSessionMemory: maxMem,
		MaxSessionGPUs:   maxGPUs,
	}

	wrappedSession := &sessionWrapper{
		session: session,
	}

	s.podMap[sessionId] = podIdx

	s.sessions[sessionId] = wrappedSession

	s.sugarLog.Debugf("Registered session \"%s\". MaxCPUs: %.2f, MaxMemory: %.2f, MaxGPUs: %d.", sessionId, maxCPUs, maxMem, maxGPUs)
}

func (s *CustomEventSequencer) getSession(sessionId string) *Session {
	var (
		wrappedSession *sessionWrapper
		ok             bool
	)

	if wrappedSession, ok = s.sessions[sessionId]; !ok {
		panic(fmt.Sprintf("Could not find session with specified ID: \"%s\". Has this session been registered yet?", sessionId))
	}

	return wrappedSession.session
}

func (s *CustomEventSequencer) getWrappedSession(sessionId string) *sessionWrapper {
	var (
		wrappedSession *sessionWrapper
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

func (s *CustomEventSequencer) stepGpu(sessionId string, timestamp time.Time, gpuUtil float64, numGPUs int) {
	wrappedSession := s.getWrappedSession(sessionId)

	gpu := wrappedSession.gpu
	podIdx, ok := s.podMap[sessionId]
	if !ok {
		panic(fmt.Sprintf("Cannot find PodIDX for Session \"%s\"", sessionId))
	}

	record := &GPURecord{
		Timestamp: UnixTime(timestamp),
		Pod:       sessionId,
		PodIdx:    podIdx,
		Value:     gpuUtil,
		GPUIdx:    "0",
	}
	committed := gpu.Debug_CommitAndInit(record)
	for i := 1; i < numGPUs; i++ {
		record := &GPURecord{
			Timestamp: UnixTime(timestamp),
			Pod:       sessionId,
			PodIdx:    podIdx,
			Value:     gpuUtil,
			GPUIdx:    fmt.Sprintf("%d", i),
		}
		gpu.Debug_Update(record)
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
	gpu.Debug_Initialize(gpuRecord)

	for i := 1; i < numGPUs; i++ {
		nextGpuRecord := &GPURecord{
			Timestamp: UnixTime(timestamp),
			GPUIdx:    fmt.Sprintf("%d", i),
			Value:     gpuUtil,
			Pod:       sessionId,
			PodIdx:    podIdx,
		}
		gpu.Debug_Update(nextGpuRecord)
	}
	// Just commit and init with the same record we used before.
	// We'll overwrite the values later.
	session.gpu = gpu.Debug_CommitAndInit(gpuRecord)
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

	evt := &eventImpl{
		name:                domain.EventSessionReady,
		eventSource:         nil,
		originalEventSource: nil,
		// Data:                data,
		timestamp: timestamp,
		id:        uuid.New().String(),
	}
	s.waitingEvents[sessionId] = evt

	// s.sugarLog.Debugf("Adding 'Session-Started' event for Session %s with timestamp %v (sec=%d).", sessionId, timestamp, sec)
}

func (s *CustomEventSequencer) AddSessionTerminatedEvent(sessionId string, tickNumber int) {
	sec := s.startingSeconds + (int64(tickNumber) * s.tickDurationSeconds)
	timestamp := time.Unix(sec, 0)
	session := s.getSession(sessionId)

	s.stepCpu(sessionId, timestamp, 0)
	s.stepGpu(sessionId, timestamp, 0, 1)
	s.stepMemory(sessionId, timestamp, 0)

	s.submitWaitingEvent(session)

	// Step again just to commit the 0 util entries that were initialized above.
	s.stepCpu(sessionId, timestamp, 0)
	s.stepGpu(sessionId, timestamp, 0, 1)
	s.stepMemory(sessionId, timestamp, 0)

	data := session.Snapshot()
	evt := &eventImpl{
		name:                domain.EventSessionStopped,
		eventSource:         nil,
		originalEventSource: nil,
		data:                data,
		timestamp:           timestamp,
		id:                  uuid.New().String(),
	}

	heap.Push(&s.eventHeap, &internalEventHeapElement{evt, -1, globalCustomEventIndex.Add(1)})
	s.sugarLog.Debugf("Added 'stopped' event for Session %s with timestamp %v (sec=%d).", sessionId, timestamp, sec)
}

func (s *CustomEventSequencer) submitWaitingEvent(session *Session) {
	sessionId := session.Pod
	dataForWaitingEvent := session.Snapshot()
	s.waitingEvents[sessionId].data = dataForWaitingEvent
	heap.Push(&s.eventHeap, &internalEventHeapElement{s.waitingEvents[sessionId], -1, globalCustomEventIndex.Add(1)})
	s.sugarLog.Debugf("Added '%s' event for Session %s with timestamp %v.", s.waitingEvents[sessionId].Name, sessionId, s.waitingEvents[sessionId].Timestamp)
	delete(s.waitingEvents, sessionId)
}

// Register a training event for a particular session.
//
// Parameters:
// - sessionId: The target Session's ID
// - duration: The duration that the training should last.
func (s *CustomEventSequencer) AddTrainingEvent(sessionId string, tickNumber int, durationInTicks int, cpuUtil float64, memUtil float64, gpuUtil float64, numGPUs int) {
	startSec := s.startingSeconds + (int64(tickNumber) * s.tickDurationSeconds)
	startTime := time.Unix(startSec, 0)
	session := s.getSession(sessionId)

	s.stepCpu(sessionId, startTime, cpuUtil)
	s.stepGpu(sessionId, startTime, gpuUtil, numGPUs)
	s.stepMemory(sessionId, startTime, memUtil)

	s.submitWaitingEvent(session)

	endSec := s.startingSeconds + (int64(tickNumber+durationInTicks) * s.tickDurationSeconds)
	endTime := time.Unix(endSec, 0)

	s.stepCpu(sessionId, endTime, 0)
	s.stepGpu(sessionId, endTime, 0, numGPUs)
	s.stepMemory(sessionId, endTime, 0)

	trainingStartedEvent := &eventImpl{
		name:                domain.EventSessionTrainingStarted,
		eventSource:         nil,
		originalEventSource: nil,
		data:                session.Snapshot(),
		timestamp:           startTime,
		id:                  uuid.New().String(),
	}
	heap.Push(&s.eventHeap, &internalEventHeapElement{trainingStartedEvent, -1, globalCustomEventIndex.Add(1)})
	s.sugarLog.Debugf("Added 'training-started' event for Session %s with timestamp %v (sec=%d).", sessionId, startTime, startSec)

	trainingEndedEvent := &eventImpl{
		name:                domain.EventSessionTrainingEnded,
		eventSource:         nil,
		originalEventSource: nil,
		data:                nil,
		timestamp:           endTime,
		id:                  uuid.New().String(),
	}
	s.waitingEvents[sessionId] = trainingEndedEvent
}

type internalEventHeap []*internalEventHeapElement

type internalEventHeapElement struct {
	domain.Event
	idx         int
	globalIndex uint64
}

func (e *internalEventHeapElement) Idx() int {
	return e.idx
}

func (e *internalEventHeapElement) SetIndex(idx int) {
	e.idx = idx
}

func (h internalEventHeap) Len() int {
	return len(h)
}

func (h internalEventHeap) Less(i, j int) bool {
	if h[i].Timestamp().Equal(h[j].Timestamp()) {
		// We want to ensure that TrainingEnded events are processed before SessionStopped events.
		// So, if event i is TrainingEnded and event j is SessionStopped, then event i should be processed first.
		if h[i].Name() == domain.EventSessionTrainingEnded && h[j].Name() == domain.EventSessionStopped {
			return true
		} else if h[j].Name() == domain.EventSessionTrainingEnded && h[i].Name() == domain.EventSessionStopped {
			return false
		}

		// Stable ordering for events with equal timestamps.
		return h[i].globalIndex < h[j].globalIndex
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
