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

type sessionMetaWrapper struct {
	session *SessionMeta

	gpu       *GPUUtil
	cpu       *CPUUtil
	memBuffer *MemoryUtilBuffer
}

type CustomEventSequencer struct {
	sessions      map[string]*sessionMetaWrapper
	eventHeap     domain.EventHeap
	eventConsumer domain.EventConsumer

	log      *zap.Logger
	sugarLog *zap.SugaredLogger

	tickDurationSeconds int64                    // The number of seconds in a tick.
	startingSeconds     int64                    // The start time for the event sequence as the number of seconds.
	podMap              map[string]int           // Map from SessionID to PodID
	waitingEvents       map[string]*domain.Event // The event that will be submitted/enqueued once the next commit happens.

	// sessionEventIndexes is a map from session ID to the current event localIndex for the session.
	// See the localIndex field of domain.Event for a description of what the "event localIndex" is.
	// The current entry for a particular session is the localIndex of the next event to be created.
	// That is, when creating the next event, its localIndex field should be set to the current
	// entry in the sessionEventIndexes map (using the associated session's ID as the key).
	sessionEventIndexes map[string]int
}

func NewCustomEventSequencer(eventConsumer domain.EventConsumer, startingSeconds int64, tickDurationSeconds int64, atom *zap.AtomicLevel) *CustomEventSequencer {
	customEventSequencer := &CustomEventSequencer{
		sessions:            make(map[string]*sessionMetaWrapper),
		sessionEventIndexes: make(map[string]int),
		waitingEvents:       make(map[string]*domain.Event),
		eventHeap:           make(domain.EventHeap, 0, 100),
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
			e := heap.Pop(&s.eventHeap).(*domain.Event)
			s.eventConsumer.SubmitEvent(e)
			s.sugarLog.Debugf("Submitted event #%d '%s' targeting session '%s' [%v]. EventID=%s.",
				e.SessionSpecificEventIndex(), e.Name, e.Data.(*SessionMeta).Pod, e.Timestamp, e.Id())
		}

		workloadGenerationCompleteChan <- struct{}{}
	}()
}

func (s *CustomEventSequencer) RegisterSession(sessionId string, maxCPUs float64, maxMem float64, maxGPUs int, maxVRAM float64, podIdx int) {
	if _, ok := s.sessions[sessionId]; ok {
		panic(fmt.Sprintf("Cannot register session %s; session was same ID already exists!", sessionId))
	}

	session := &SessionMeta{
		Pod:              sessionId,
		MaxSessionCPUs:   maxCPUs,
		MaxSessionMemory: maxMem,
		MaxSessionGPUs:   maxGPUs,
		MaxSessionVRAM:   maxVRAM,
	}

	wrappedSession := &sessionMetaWrapper{
		session: session,
	}

	s.podMap[sessionId] = podIdx
	s.sessions[sessionId] = wrappedSession
	s.sessionEventIndexes[sessionId] = 0

	s.sugarLog.Debugf("Registered session \"%s\". MaxCPUs: %.2f, MaxMemory: %.2f, MaxGPUs: %d, MaxVRAM: %.2f", sessionId, maxCPUs, maxMem, maxGPUs, maxVRAM)
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
	committed := cpu.DebugCommitAndInit(record)
	wrappedSession.session.CPU = committed
}

func (s *CustomEventSequencer) stepGpu(sessionId string, timestamp time.Time, gpuUtil []domain.GpuUtilization, vramGb float64) {
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
			VramGb:    vramGb,
		}

		if gpuIdx == 0 {
			committed = gpu.DebugCommitAndInit(record)
		} else {
			gpu.DebugUpdate(record)
		}
	}

	wrappedSession.session.GPU = committed

	if committed != nil {
		wrappedSession.session.VRAM = committed.VRamGB
	}
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
	evt := &domain.Event{
		Name:                domain.EventSessionReady,
		EventSource:         nil,
		OriginalEventSource: nil,
		Timestamp:           timestamp,
		OriginalTimestamp:   timestamp,
		LocalIndex:          localIndex,
		ID:                  uuid.New().String(),
		GlobalIndex:         globalCustomEventIndex.Add(1),
		HeapIndex:           -1,
	}
	s.waitingEvents[sessionId] = evt

	s.log.Debug("Adding session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", domain.EventSessionReady.String()),
		zap.Time("timestamp", timestamp),
		zap.Int64("second", sec),
		zap.Int("local_index", evt.LocalIndex),
		zap.Uint64("global_index", evt.GlobalIndex),
		zap.String("session_id", evt.ID))
}

func (s *CustomEventSequencer) AddSessionTerminatedEvent(sessionId string, tickNumber int) {
	sec := s.startingSeconds + (int64(tickNumber) * s.tickDurationSeconds)
	timestamp := time.Unix(sec, 0)
	sessionMeta := s.getSessionMeta(sessionId)

	s.stepCpu(sessionId, timestamp, 0)
	s.stepGpu(sessionId, timestamp, []domain.GpuUtilization{{Utilization: 0}}, 0)
	s.stepMemory(sessionId, timestamp, 0)

	s.submitWaitingEvent(sessionMeta)

	// Step again just to commit the 0 util entries that were initialized above.
	s.stepCpu(sessionId, timestamp, 0)
	s.stepGpu(sessionId, timestamp, []domain.GpuUtilization{{Utilization: 0}}, 0)
	s.stepMemory(sessionId, timestamp, 0)

	metadata := sessionMeta.Snapshot()
	eventIndex := s.sessionEventIndexes[sessionId]
	s.sessionEventIndexes[sessionId] = eventIndex + 1
	evt := &domain.Event{
		Name:                domain.EventSessionStopped,
		EventSource:         nil,
		OriginalEventSource: nil,
		Data:                metadata,
		Timestamp:           timestamp,
		OriginalTimestamp:   timestamp,
		LocalIndex:          eventIndex,
		ID:                  uuid.New().String(),
		GlobalIndex:         globalCustomEventIndex.Add(1),
		HeapIndex:           -1,
	}

	heap.Push(&s.eventHeap, evt)
	s.sugarLog.Debugf("Added 'stopped' event for Session %s with timestamp %v (sec=%d). EventID=%s.", sessionId, timestamp, sec, evt.ID)

	s.log.Debug("Added session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", domain.EventSessionStopped.String()),
		zap.Time("timestamp", timestamp),
		zap.Int64("second", sec),
		zap.Int("local_index", evt.LocalIndex),
		zap.Uint64("global_index", evt.GlobalIndex),
		zap.String("event_id", evt.ID),
		zap.String("metadata", metadata.String()))
}

func (s *CustomEventSequencer) submitWaitingEvent(sessionMeta *SessionMeta) {
	sessionId := sessionMeta.Pod
	dataForWaitingEvent := sessionMeta.Snapshot()
	s.waitingEvents[sessionId].Data = dataForWaitingEvent
	evt := s.waitingEvents[sessionId]
	heap.Push(&s.eventHeap, evt)

	s.log.Debug("Adding session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", evt.Name.String()),
		zap.Time("timestamp", evt.Timestamp),
		zap.Int64("order_seq", evt.OrderSeq),
		zap.Int("local_index", evt.LocalIndex),
		zap.Uint64("global_index", evt.GlobalIndex),
		zap.String("session_id", evt.ID))

	delete(s.waitingEvents, sessionId)
}

// gpuUtilizationValuesAboveZero returns the number of entries in the slice of domain.GpuUtilization structs
// such that the Utilization field of the domain.GpuUtilization is > 0.
func gpuUtilizationValuesAboveZero(gpuUtil []domain.GpuUtilization) int {
	num := 0
	for _, util := range gpuUtil {
		if util.Utilization > 0 {
			num += 1
		}
	}

	return num
}

// AddTrainingEvent registers a training event for a particular session.
//
// Parameters:
// - sessionId: The target Session's ID
// - duration: The duration that the training should last.
func (s *CustomEventSequencer) AddTrainingEvent(sessionId string, tickNumber int, durationInTicks int, cpuUtil float64, memUtil float64, gpuUtil []domain.GpuUtilization, vramUsageGB float64) {
	startSec := s.startingSeconds + (int64(tickNumber) * s.tickDurationSeconds)
	startTime := time.Unix(startSec, 0)
	sessionMeta := s.getSessionMeta(sessionId)

	s.stepCpu(sessionId, startTime, cpuUtil)
	s.stepGpu(sessionId, startTime, gpuUtil, vramUsageGB)
	s.stepMemory(sessionId, startTime, memUtil)

	sessionMeta.CurrentTrainingMaxCPUs = cpuUtil
	sessionMeta.CurrentTrainingMaxMemory = memUtil
	sessionMeta.CurrentTrainingMaxGPUs = gpuUtilizationValuesAboveZero(gpuUtil)
	sessionMeta.CurrentTrainingMaxVRAM = vramUsageGB

	s.submitWaitingEvent(sessionMeta)

	endSec := s.startingSeconds + (int64(tickNumber+durationInTicks) * s.tickDurationSeconds)
	endTime := time.Unix(endSec, 0)

	s.stepCpu(sessionId, endTime, 0)
	s.stepGpu(sessionId, endTime, gpuUtil, vramUsageGB)
	s.stepMemory(sessionId, endTime, 0)

	eventIndex := s.sessionEventIndexes[sessionId]
	metadata := sessionMeta.Snapshot()
	trainingStartedEvent := &domain.Event{
		Name:                domain.EventSessionTrainingStarted,
		EventSource:         nil,
		OriginalEventSource: nil,
		Data:                metadata,
		LocalIndex:          eventIndex,
		OriginalTimestamp:   startTime,
		Timestamp:           startTime,
		ID:                  uuid.New().String(),
		GlobalIndex:         globalCustomEventIndex.Add(1),
		HeapIndex:           -1,
	}
	heap.Push(&s.eventHeap, trainingStartedEvent)
	s.log.Debug("Added session event.",
		zap.String("session_id", sessionId),
		zap.String("event_name", domain.EventSessionTrainingStarted.String()),
		zap.Time("timestamp", startTime),
		zap.Int64("second", startSec),
		zap.Int("local_index", trainingStartedEvent.LocalIndex),
		zap.Uint64("global_index", trainingStartedEvent.GlobalIndex),
		zap.String("event_id", trainingStartedEvent.ID),
		zap.String("metadata", metadata.String()))

	trainingEndedEvent := &domain.Event{
		Name:                domain.EventSessionTrainingEnded,
		EventSource:         nil,
		OriginalEventSource: nil,
		Data:                nil,
		LocalIndex:          eventIndex + 1,
		Timestamp:           endTime,
		OriginalTimestamp:   endTime,
		ID:                  uuid.New().String(),
		GlobalIndex:         globalCustomEventIndex.Add(1),
	}
	s.waitingEvents[sessionId] = trainingEndedEvent
	s.sessionEventIndexes[sessionId] = eventIndex + 2
}
