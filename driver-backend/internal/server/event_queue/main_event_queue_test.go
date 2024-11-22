package event_queue_test

import (
	"container/heap"
	"fmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/event_queue"
	"go.uber.org/mock/gomock"
	"time"
)

var _ = Describe("MainEventQueue Tests", func() {
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

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

			Expect(queue.Len()).To(Equal(1))
			Expect(queue.Peek()).To(Equal(sessionQueue1))

			session2Id := "Session2"
			sessionQueue2 := event_queue.NewSessionEventQueue(session2Id)

			Expect(sessionQueue2.Len()).To(Equal(0))
			Expect(sessionQueue2.SessionId).To(Equal(session2Id))
			Expect(sessionQueue2.HeapIndex).To(Equal(-1))

			heap.Push(&queue, sessionQueue2)

			Expect(queue.Len()).To(Equal(2))
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

			session2Event1 := createEvent(domain.EventSessionStarted, session2Id, 0, time.UnixMilli(0), mockCtrl)
			sessionQueue2.Push(session2Event1)

			heap.Push(&queue, sessionQueue2)

			Expect(queue.Len()).To(Equal(2))
			Expect(queue.Peek()).To(Equal(sessionQueue2))
			Expect(queue.Peek().Len()).To(Equal(1))
		})

		It("Will correctly handle enqueuing and de-queuing SessionEventQueues", func() {
			session1Id := "Session1"
			sessionQueue1 := event_queue.NewSessionEventQueue(session1Id)
			session1Event1 := createEvent(domain.EventSessionStarted, session1Id, 1, time.UnixMilli(1), mockCtrl)
			sessionQueue1.Push(session1Event1)
			Expect(sessionQueue1.Len()).To(Equal(1))

			session2Id := "Session2"
			sessionQueue2 := event_queue.NewSessionEventQueue(session2Id)
			session2Event1 := createEvent(domain.EventSessionStarted, session2Id, 2, time.UnixMilli(2), mockCtrl)
			sessionQueue2.Push(session2Event1)
			Expect(sessionQueue2.Len()).To(Equal(1))

			heap.Push(&queue, sessionQueue2)
			Expect(queue.Len()).To(Equal(1))
			Expect(queue.Peek()).To(Equal(sessionQueue2))

			heap.Push(&queue, sessionQueue1)
			Expect(queue.Len()).To(Equal(2))
			Expect(queue.Peek()).To(Equal(sessionQueue1))

			session3Id := "Session3"
			sessionQueue3 := event_queue.NewSessionEventQueue(session3Id)
			session3Event1 := createEvent(domain.EventSessionStarted, session3Id, 0, time.UnixMilli(0), mockCtrl)
			sessionQueue3.Push(session3Event1)
			Expect(sessionQueue3.Len()).To(Equal(1))

			heap.Push(&queue, sessionQueue3)
			Expect(queue.Len()).To(Equal(3))
			Expect(queue.Peek()).To(Equal(sessionQueue3))
		})

		It("Will be sorted into the correct order when initialized using a reverse-ordered non-empty backing slice", func() {
			sessionIDs := make([]string, 0, 8)
			sessionEventQueues := make([]*event_queue.SessionEventQueue, 0, 8)
			for i := 0; i < 8; i++ {
				sessionId := fmt.Sprintf("Session-%d", i)
				sessionIDs = append(sessionIDs, sessionId)

				sessionEventQueue := event_queue.NewSessionEventQueue(sessionId)

				evt := createEvent(domain.EventSessionStarted, sessionId, uint64(i), time.UnixMilli(int64(i)), mockCtrl)
				sessionEventQueue.Push(evt)

				sessionEventQueues = append(sessionEventQueues, sessionEventQueue)
			}

			for i := 0; i < 8; i++ {
				Expect(sessionEventQueues[i]).ToNot(BeNil())
				Expect(sessionEventQueues[i].SessionId).To(Equal(sessionIDs[i]))

				sessionId := fmt.Sprintf("Session-%d", i)
				Expect(sessionEventQueues[i].SessionId).To(Equal(sessionId))
			}

			queue = event_queue.MainEventQueue{sessionEventQueues[7], sessionEventQueues[6], sessionEventQueues[5], sessionEventQueues[4], sessionEventQueues[3], sessionEventQueues[2], sessionEventQueues[1], sessionEventQueues[0]}

			for i := 0; i < 8; i++ {
				timestamp, _ := queue[i].NextEventTimestamp()
				fmt.Printf("Queue element #%d: \"%s\" (HeapIndex=%d) [%v]\n", i, queue[i].SessionId, queue[i].HeapIndex, timestamp)
			}

			heap.Init(&queue)

			for i := 0; i < 8; i++ {
				timestamp, _ := queue[i].NextEventTimestamp()
				fmt.Printf("Queue element #%d: \"%s\" (HeapIndex=%d) [%v]\n", i, queue[i].SessionId, queue[i].HeapIndex, timestamp)
			}

			Expect(queue.Len()).To(Equal(8))

			for i := 0; i < 8; i++ {
				Expect(queue.Peek().SessionId).To(Equal(sessionIDs[i]))
				Expect(queue.Peek()).To(Equal(sessionEventQueues[i]))

				sessionEventQueue := heap.Pop(&queue).(*event_queue.SessionEventQueue)
				Expect(sessionEventQueue).ToNot(BeNil())
				Expect(sessionEventQueue.SessionId).To(Equal(sessionIDs[i]))
				Expect(sessionEventQueue).To(Equal(sessionEventQueues[i]))
			}
		})
	})
})
