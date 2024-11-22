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

var _ = Describe("MainEventQueue Tests", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	createEvent := func(name domain.EventName, sessionId string, index uint64, timestamp time.Time) *domain.Event {
		data := mock_domain.NewMockSessionMetadata(mockCtrl)
		data.EXPECT().GetPod().AnyTimes().Return(sessionId)

		return &domain.Event{
			Name:        name,
			GlobalIndex: index,
			LocalIndex:  int(index),
			ID:          uuid.NewString(),
			Timestamp:   timestamp,
		}
	}

	Expect(createEvent).ToNot(BeNil())

	It("Can be instantiated correctly", func() {
		queue := make(event_queue.MainEventQueue, 0)

		Expect(queue).ToNot(BeNil())
		Expect(queue.Len()).To(Equal(0))
	})

	Context("Basic operation", func() {
		var queue event_queue.MainEventQueue

		BeforeEach(func() {
			queue = make(event_queue.MainEventQueue, 0)
		})

		It("Will correctly handle a single empty SessionEventQueue", func() {
			sessionId := uuid.NewString()
			sessionQueue := event_queue.NewSessionEventQueue(sessionId)

			Expect(sessionQueue.Len()).To(Equal(0))
			Expect(sessionQueue.SessionId).To(Equal(sessionId))
			Expect(sessionQueue.HeapIndex).To(Equal(-1))

			heap.Push(&queue, sessionQueue)

			Expect(queue.Len()).To(Equal(1))
			Expect(queue.Peek()).To(Equal(sessionQueue))
		})

		It("Will correctly handle two empty SessionEventQueues", func() {
			session1Id := "Session1"
			sessionQueue1 := event_queue.NewSessionEventQueue(session1Id)

			Expect(sessionQueue1.Len()).To(Equal(0))
			Expect(sessionQueue1.SessionId).To(Equal(session1Id))
			Expect(sessionQueue1.HeapIndex).To(Equal(-1))

			heap.Push(&queue, sessionQueue1)

			GinkgoWriter.Println("Pushed first session queue.")

			Expect(queue.Len()).To(Equal(1))
			Expect(queue.Peek()).To(Equal(sessionQueue1))

			session2Id := "Session2"
			sessionQueue2 := event_queue.NewSessionEventQueue(session2Id)

			Expect(sessionQueue2.Len()).To(Equal(0))
			Expect(sessionQueue2.SessionId).To(Equal(session2Id))
			Expect(sessionQueue2.HeapIndex).To(Equal(-1))

			heap.Push(&queue, sessionQueue2)

			GinkgoWriter.Println("Pushed first second queue.")

			Expect(queue.Len()).To(Equal(2))

			Expect(queue.Peek()).To(Equal(sessionQueue1))

			sessionQueue := queue.Pop().(*event_queue.SessionEventQueue)
			Expect(sessionQueue).ToNot(BeNil())
			Expect(sessionQueue.SessionId).To(Equal(session1Id))
			Expect(sessionQueue).To(Equal(sessionQueue1))
			Expect(queue.Len()).To(Equal(1))

			Expect(queue.Peek()).To(Equal(sessionQueue2))

			sessionQueue = queue.Pop().(*event_queue.SessionEventQueue)
			Expect(sessionQueue).ToNot(BeNil())
			Expect(sessionQueue.SessionId).To(Equal(session2Id))
			Expect(sessionQueue).To(Equal(sessionQueue2))
			Expect(queue.Len()).To(Equal(0))
		})

		It("Will position an empty SessionEventQueue behind a non-empty SessionEventQueue", func() {
			session1Id := "Session1"
			sessionQueue1 := event_queue.NewSessionEventQueue(session1Id)

			heap.Push(&queue, sessionQueue1)

			Expect(queue.Len()).To(Equal(1))
			Expect(queue.Peek()).To(Equal(sessionQueue1))
			Expect(queue.Peek().Len()).To(Equal(0))

			session2Id := "Session2"
			sessionQueue2 := event_queue.NewSessionEventQueue(session2Id)

			session2Event1 := createEvent(domain.EventSessionStarted, session2Id, 0, time.UnixMilli(0))
			sessionQueue2.Push(session2Event1)

			heap.Push(&queue, sessionQueue2)

			Expect(queue.Len()).To(Equal(1))
			Expect(queue.Peek()).To(Equal(sessionQueue2))
			Expect(queue.Peek().Len()).To(Equal(1))
		})
	})
})
