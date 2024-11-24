package event_queue_test

import (
	"container/heap"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/mock_domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/event_queue"
	"go.uber.org/mock/gomock"
	"time"
)

var _ = Describe("SessionEventQueue Tests", func() {
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	It("Can be instantiated correctly", func() {
		sessionId := uuid.NewString()
		queue := event_queue.NewSessionEventQueue(sessionId)

		Expect(queue.Len()).To(Equal(0))
		Expect(queue.SessionId).To(Equal(sessionId))
		Expect(queue.HeapIndex).To(Equal(-1))
	})

	It("Will correctly enqueue and dequeue a single event", func() {
		sessionId := uuid.NewString()
		queue := event_queue.NewSessionEventQueue(sessionId)
		Expect(queue.Len()).To(Equal(0))

		data := mock_domain.NewMockSessionMetadata(mockCtrl)
		data.EXPECT().GetPod().AnyTimes().Return(sessionId)

		evt := &domain.Event{
			Name:        domain.EventSessionReady,
			GlobalIndex: 0,
			LocalIndex:  0,
			ID:          uuid.NewString(),
			SessionId:   sessionId,
			Timestamp:   time.Now(),
			Data:        data,
		}

		queue.Push(evt)

		Expect(queue.Len()).To(Equal(1))
		Expect(queue.Peek()).To(Equal(evt))

		poppedEvent := queue.Pop()
		Expect(queue.Len()).To(Equal(0))
		Expect(poppedEvent).To(Equal(evt))
	})

	It("Will correctly enqueue and dequeue a multiple events", func() {
		sessionId := uuid.NewString()
		queue := event_queue.NewSessionEventQueue(sessionId)
		Expect(queue.Len()).To(Equal(0))

		evt1 := createEvent(domain.EventSessionReady, sessionId, 0, time.UnixMilli(0), mockCtrl)
		evt2 := createEvent(domain.EventSessionTrainingStarted, sessionId, 1, time.UnixMilli(1), mockCtrl)
		evt3 := createEvent(domain.EventSessionTrainingEnded, sessionId, 2, time.UnixMilli(2), mockCtrl)
		evt4 := createEvent(domain.EventSessionStopped, sessionId, 3, time.UnixMilli(3), mockCtrl)

		By("Correctly pushing events onto its internal queue")

		queue.Push(evt4)
		Expect(queue.Len()).To(Equal(1))
		Expect(queue.Peek()).To(Equal(evt4))

		queue.Push(evt3)
		Expect(queue.Len()).To(Equal(2))
		Expect(queue.Peek()).To(Equal(evt3))

		queue.Push(evt2)
		Expect(queue.Len()).To(Equal(3))
		Expect(queue.Peek()).To(Equal(evt2))

		queue.Push(evt1)
		Expect(queue.Len()).To(Equal(4))
		Expect(queue.Peek()).To(Equal(evt1))

		By("Correctly popping events off of its internal queue")

		evt := queue.Pop()
		Expect(queue.Len()).To(Equal(3))
		Expect(evt).To(Equal(evt1))
		Expect(queue.Peek()).To(Equal(evt2))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(2))
		Expect(evt).To(Equal(evt2))
		Expect(queue.Peek()).To(Equal(evt3))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt3))
		Expect(queue.Peek()).To(Equal(evt4))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt4))
		Expect(queue.Peek() == nil).To(BeTrue())
	})

	It("Will panic if pushing event for wrong session into queue", func() {
		sessionId := uuid.NewString()
		queue := event_queue.NewSessionEventQueue(sessionId)
		Expect(queue.Len()).To(Equal(0))

		evt := createEvent(domain.EventSessionTrainingStarted, "WrongSessionId", 0, time.UnixMilli(0), mockCtrl)

		pushAndPanic := func() {
			queue.Push(evt)
		}

		Expect(pushAndPanic).To(Panic())
	})

	It("Will correctly break ties using the global indices of events", func() {
		sessionId := uuid.NewString()
		queue := event_queue.NewSessionEventQueue(sessionId)
		Expect(queue.Len()).To(Equal(0))

		evt1 := createEvent(domain.EventSessionTrainingStarted, sessionId, 0, time.UnixMilli(0), mockCtrl)
		evt2 := createEvent(domain.EventSessionTrainingEnded, sessionId, 1, time.UnixMilli(0), mockCtrl)

		Expect(evt1.Timestamp).To(Equal(evt2.Timestamp))
		Expect(evt1.OriginalTimestamp).To(Equal(evt2.OriginalTimestamp))

		queue.Push(evt2)
		Expect(queue.Len()).To(Equal(1))
		Expect(queue.Peek()).To(Equal(evt2))

		queue.Push(evt1)
		Expect(queue.Len()).To(Equal(2))
		Expect(queue.Peek()).To(Equal(evt1))

		evt := queue.Pop()
		Expect(queue.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt1))
		Expect(queue.Peek()).To(Equal(evt2))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt2))
		Expect(queue.Peek() == nil).To(BeTrue())

		// Note that these are not in the order you'd inspect just based on their names.
		evt3 := createEvent(domain.EventSessionTrainingStarted, sessionId, 3, time.UnixMilli(0), mockCtrl)
		evt4 := createEvent(domain.EventSessionTrainingEnded, sessionId, 2, time.UnixMilli(0), mockCtrl)

		queue.Push(evt3)
		Expect(queue.Len()).To(Equal(1))
		Expect(queue.Peek()).To(Equal(evt3))

		queue.Push(evt4)
		Expect(queue.Len()).To(Equal(2))
		Expect(queue.Peek()).To(Equal(evt4))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt4))
		Expect(queue.Peek()).To(Equal(evt3))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt3))
		Expect(queue.Peek() == nil).To(BeTrue())
	})

	It("Will correctly break ties by placing 'training-ended' events before 'session-stopped' events", func() {
		sessionId := uuid.NewString()
		queue := event_queue.NewSessionEventQueue(sessionId)
		Expect(queue.Len()).To(Equal(0))

		// Note that these were assigned the same indices.
		evt1 := createEvent(domain.EventSessionStopped, sessionId, 1, time.UnixMilli(0), mockCtrl)
		evt2 := createEvent(domain.EventSessionTrainingEnded, sessionId, 0, time.UnixMilli(0), mockCtrl)

		By("Doing so when we push the 'training-ended' event onto the queue first.")

		queue.Push(evt2)
		Expect(queue.Len()).To(Equal(1))
		Expect(queue.Peek()).To(Equal(evt2))

		queue.Push(evt1)
		Expect(queue.Len()).To(Equal(2))
		Expect(queue.Peek()).To(Equal(evt2))

		evt := queue.Pop()
		Expect(queue.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt2))
		Expect(queue.Peek()).To(Equal(evt1))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt1))
		Expect(queue.Peek() == nil).To(BeTrue())

		By("Doing so when we push the 'session-stopped' event onto the queue first.")
		queue = event_queue.NewSessionEventQueue(sessionId)
		Expect(queue.Len()).To(Equal(0))

		evt2 = createEvent(domain.EventSessionStopped, sessionId, 1, time.UnixMilli(0), mockCtrl)
		evt1 = createEvent(domain.EventSessionTrainingEnded, sessionId, 0, time.UnixMilli(0), mockCtrl)

		queue.Push(evt2)
		Expect(queue.Len()).To(Equal(1))
		Expect(queue.Peek()).To(Equal(evt2))

		queue.Push(evt1)
		Expect(queue.Len()).To(Equal(2))
		Expect(queue.Peek()).To(Equal(evt1))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt1))
		Expect(queue.Peek()).To(Equal(evt2))

		evt = queue.Pop()
		Expect(queue.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt2))
		Expect(queue.Peek() == nil).To(BeTrue())
	})

	It("Will panic when the event's indices are in an impossible configuration", func() {
		sessionId := uuid.NewString()
		queue := event_queue.NewSessionEventQueue(sessionId)
		Expect(queue.Len()).To(Equal(0))

		// Note that these were assigned the same indices.
		evt1 := createEvent(domain.EventSessionStopped, sessionId, 0, time.UnixMilli(0), mockCtrl)
		evt2 := createEvent(domain.EventSessionTrainingEnded, sessionId, 1, time.UnixMilli(0), mockCtrl)

		By("Doing so when we push the 'training-ended' event onto the queue first.")

		queue.Push(evt2)
		Expect(queue.Len()).To(Equal(1))
		Expect(queue.Peek()).To(Equal(evt2))

		pushAndCausePanic := func() {
			queue.Push(evt1)
		}

		Expect(pushAndCausePanic).To(Panic())
	})

	It("Will be sorted into the correct order when initialized using a non-empty backing slice", func() {
		sessionId := uuid.NewString()

		evt1 := createEvent(domain.EventSessionReady, sessionId, 0, time.UnixMilli(0), mockCtrl)
		evt2 := createEvent(domain.EventSessionTrainingStarted, sessionId, 1, time.UnixMilli(1), mockCtrl)
		evt3 := createEvent(domain.EventSessionTrainingEnded, sessionId, 2, time.UnixMilli(2), mockCtrl)
		evt4 := createEvent(domain.EventSessionTrainingStarted, sessionId, 3, time.UnixMilli(3), mockCtrl)
		evt5 := createEvent(domain.EventSessionTrainingEnded, sessionId, 4, time.UnixMilli(4), mockCtrl)
		evt6 := createEvent(domain.EventSessionTrainingStarted, sessionId, 5, time.UnixMilli(5), mockCtrl)
		evt7 := createEvent(domain.EventSessionTrainingEnded, sessionId, 6, time.UnixMilli(6), mockCtrl)
		evt8 := createEvent(domain.EventSessionStopped, sessionId, 7, time.UnixMilli(7), mockCtrl)

		queue := &event_queue.SessionEventQueue{
			SessionId:     sessionId,
			InternalQueue: domain.EventHeap{evt8, evt2, evt7, evt6, evt5, evt3, evt4, evt1},
			HeapIndex:     -1,
		}

		heap.Init(&queue.InternalQueue)

		Expect(queue.Len()).To(Equal(8))
		Expect(queue.Peek()).To(Equal(evt1))

		Expect(queue.Pop()).To(Equal(evt1))
		Expect(queue.Peek()).To(Equal(evt2))
		Expect(queue.Len()).To(Equal(7))

		Expect(queue.Pop()).To(Equal(evt2))
		Expect(queue.Peek()).To(Equal(evt3))
		Expect(queue.Len()).To(Equal(6))

		Expect(queue.Pop()).To(Equal(evt3))
		Expect(queue.Peek()).To(Equal(evt4))
		Expect(queue.Len()).To(Equal(5))

		Expect(queue.Pop()).To(Equal(evt4))
		Expect(queue.Peek()).To(Equal(evt5))
		Expect(queue.Len()).To(Equal(4))

		Expect(queue.Pop()).To(Equal(evt5))
		Expect(queue.Peek()).To(Equal(evt6))
		Expect(queue.Len()).To(Equal(3))

		Expect(queue.Pop()).To(Equal(evt6))
		Expect(queue.Peek()).To(Equal(evt7))
		Expect(queue.Len()).To(Equal(2))

		Expect(queue.Pop()).To(Equal(evt7))
		Expect(queue.Peek()).To(Equal(evt8))
		Expect(queue.Len()).To(Equal(1))

		Expect(queue.Pop()).To(Equal(evt8))
		Expect(queue.Peek() == nil).To(BeTrue())
		Expect(queue.Len()).To(Equal(0))
	})

	It("Will be sorted into the correct order when initialized using a reverse-ordered non-empty backing slice", func() {
		sessionId := uuid.NewString()

		evt1 := createEvent(domain.EventSessionReady, sessionId, 0, time.UnixMilli(0), mockCtrl)
		evt2 := createEvent(domain.EventSessionTrainingStarted, sessionId, 1, time.UnixMilli(1), mockCtrl)
		evt3 := createEvent(domain.EventSessionTrainingEnded, sessionId, 2, time.UnixMilli(2), mockCtrl)
		evt4 := createEvent(domain.EventSessionTrainingStarted, sessionId, 3, time.UnixMilli(3), mockCtrl)
		evt5 := createEvent(domain.EventSessionTrainingEnded, sessionId, 4, time.UnixMilli(4), mockCtrl)
		evt6 := createEvent(domain.EventSessionTrainingStarted, sessionId, 5, time.UnixMilli(5), mockCtrl)
		evt7 := createEvent(domain.EventSessionTrainingEnded, sessionId, 6, time.UnixMilli(6), mockCtrl)
		evt8 := createEvent(domain.EventSessionStopped, sessionId, 7, time.UnixMilli(7), mockCtrl)

		queue := &event_queue.SessionEventQueue{
			SessionId:     sessionId,
			InternalQueue: domain.EventHeap{evt8, evt7, evt6, evt5, evt4, evt3, evt2, evt1},
			HeapIndex:     -1,
		}

		heap.Init(&queue.InternalQueue)

		Expect(queue.Len()).To(Equal(8))
		Expect(queue.Peek()).To(Equal(evt1))

		Expect(queue.Pop()).To(Equal(evt1))
		Expect(queue.Peek()).To(Equal(evt2))
		Expect(queue.Len()).To(Equal(7))

		Expect(queue.Pop()).To(Equal(evt2))
		Expect(queue.Peek()).To(Equal(evt3))
		Expect(queue.Len()).To(Equal(6))

		Expect(queue.Pop()).To(Equal(evt3))
		Expect(queue.Peek()).To(Equal(evt4))
		Expect(queue.Len()).To(Equal(5))

		Expect(queue.Pop()).To(Equal(evt4))
		Expect(queue.Peek()).To(Equal(evt5))
		Expect(queue.Len()).To(Equal(4))

		Expect(queue.Pop()).To(Equal(evt5))
		Expect(queue.Peek()).To(Equal(evt6))
		Expect(queue.Len()).To(Equal(3))

		Expect(queue.Pop()).To(Equal(evt6))
		Expect(queue.Peek()).To(Equal(evt7))
		Expect(queue.Len()).To(Equal(2))

		Expect(queue.Pop()).To(Equal(evt7))
		Expect(queue.Peek()).To(Equal(evt8))
		Expect(queue.Len()).To(Equal(1))

		Expect(queue.Pop()).To(Equal(evt8))
		Expect(queue.Peek() == nil).To(BeTrue())
		Expect(queue.Len()).To(Equal(0))
	})
})
