package event_queue_test

import (
	"errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/event_queue"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("EventQueue Tests", func() {
	atom := zap.NewAtomicLevelAt(zap.DebugLevel)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	It("Can be instantiated correctly", func() {
		queue := event_queue.NewEventQueue(&atom)

		Expect(queue.Len()).To(Equal(0))
	})

	Context("Basic operation", func() {
		var queue *event_queue.EventQueue

		BeforeEach(func() {
			queue = event_queue.NewEventQueue(&atom)
		})

		It("Will correctly handle enqueuing and de-queuing events for a single session", func() {
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(0))

			session1Id := "Session1"
			eventSessionStarted := createEvent(domain.EventSessionStarted, session1Id, 0, time.UnixMilli(0), mockCtrl)
			queue.EnqueueEvent(eventSessionStarted)
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(1))
			Expect(queue.MainEventQueueLength()).To(Equal(1))
			Expect(queue.HasEventsForSession(session1Id)).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(0))).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(1))).To(BeFalse())

			timestamp, err := queue.GetTimestampOfNextReadyEvent()
			Expect(err).ToNot(BeNil())
			Expect(errors.Is(err, event_queue.ErrNoMoreEvents)).To(BeTrue())

			eventSessionReady := createEvent(domain.EventSessionReady, session1Id, 1, time.UnixMilli(1), mockCtrl)
			queue.EnqueueEvent(eventSessionReady)

			Expect(queue.Len()).To(Equal(1))
			Expect(queue.NumSessionQueues()).To(Equal(1))
			Expect(queue.MainEventQueueLength()).To(Equal(1))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForTick(time.UnixMilli(0))).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(1))).To(BeTrue())

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			eventTrainingStarted := createEvent(domain.EventSessionTrainingStarted, session1Id, 2, time.UnixMilli(2), mockCtrl)
			queue.EnqueueEvent(eventTrainingStarted)

			Expect(queue.Len()).To(Equal(2))
			Expect(queue.NumSessionQueues()).To(Equal(1))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForTick(time.UnixMilli(0))).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(1))).To(BeTrue())

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			eventTrainingEnded := createEvent(domain.EventSessionTrainingEnded, session1Id, 3, time.UnixMilli(3), mockCtrl)
			queue.EnqueueEvent(eventTrainingEnded)

			Expect(queue.Len()).To(Equal(3))
			Expect(queue.NumSessionQueues()).To(Equal(1))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForTick(time.UnixMilli(0))).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(1))).To(BeTrue())

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			eventSessionStopped := createEvent(domain.EventSessionStopped, session1Id, 4, time.UnixMilli(4), mockCtrl)
			queue.EnqueueEvent(eventSessionStopped)

			Expect(queue.Len()).To(Equal(4))
			Expect(queue.NumSessionQueues()).To(Equal(1))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForTick(time.UnixMilli(0))).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(1))).To(BeTrue())

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			dequeuedEvent := queue.Pop(time.UnixMilli(1))
			Expect(dequeuedEvent).To(Equal(eventSessionReady))
			Expect(queue.Len()).To(Equal(3))
			Expect(queue.NumSessionQueues()).To(Equal(1))

			dequeuedEvent = queue.Pop(time.UnixMilli(2))
			Expect(dequeuedEvent).To(Equal(eventTrainingStarted))
			Expect(queue.Len()).To(Equal(2))
			Expect(queue.NumSessionQueues()).To(Equal(1))

			dequeuedEvent = queue.Pop(time.UnixMilli(3))
			Expect(dequeuedEvent).To(Equal(eventTrainingEnded))
			Expect(queue.Len()).To(Equal(1))
			Expect(queue.NumSessionQueues()).To(Equal(1))

			dequeuedEvent = queue.Pop(time.UnixMilli(4))
			Expect(dequeuedEvent).To(Equal(eventSessionStopped))
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(1))
		})

		It("Will correctly handle session operations for multiple sessions", func() {
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(0))

			session1Id := "Session1"
			eventSession1Started := createEvent(domain.EventSessionStarted, session1Id, 0, time.UnixMilli(0), mockCtrl)
			queue.EnqueueEvent(eventSession1Started)
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(1))

			session2Id := "Session2"
			eventSession2Started := createEvent(domain.EventSessionStarted, session2Id, 1, time.UnixMilli(0), mockCtrl)
			queue.EnqueueEvent(eventSession2Started)
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(2))
			Expect(queue.MainEventQueueLength()).To(Equal(2))

			eventSession1Ready := createEvent(domain.EventSessionReady, session1Id, 3, time.UnixMilli(2), mockCtrl)
			queue.EnqueueEvent(eventSession1Ready)

			Expect(queue.Len()).To(Equal(1))
			Expect(queue.NumSessionQueues()).To(Equal(2))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())

			timestamp, err := queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(2)))

			nextEvent := queue.Peek(time.UnixMilli(2))
			Expect(nextEvent).To(Equal(eventSession1Ready))

			Expect(queue.Peek(time.UnixMilli(1)) == nil).To(BeTrue())

			eventSession2Ready := createEvent(domain.EventSessionReady, session2Id, 2, time.UnixMilli(1), mockCtrl)
			queue.EnqueueEvent(eventSession2Ready)

			Expect(queue.Len()).To(Equal(2))
			Expect(queue.NumSessionQueues()).To(Equal(2))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForSession(session2Id)).To(BeTrue())

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			nextEvent = queue.Peek(time.UnixMilli(2))
			Expect(nextEvent).To(Equal(eventSession2Ready))

			eventTrainingStartedSess1 := createEvent(domain.EventSessionTrainingStarted, session1Id, 4, time.UnixMilli(3), mockCtrl)
			queue.EnqueueEvent(eventTrainingStartedSess1)

			Expect(queue.Len()).To(Equal(3))
			Expect(queue.NumSessionQueues()).To(Equal(2))

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			nextEvent = queue.Peek(time.UnixMilli(2))
			Expect(nextEvent).To(Equal(eventSession2Ready))

			eventTrainingStartedSess2 := createEvent(domain.EventSessionTrainingStarted, session2Id, 5, time.UnixMilli(4), mockCtrl)
			queue.EnqueueEvent(eventTrainingStartedSess2)
			Expect(queue.Len()).To(Equal(4))

			eventTrainingEndedSess1 := createEvent(domain.EventSessionTrainingEnded, session1Id, 7, time.UnixMilli(6), mockCtrl)
			queue.EnqueueEvent(eventTrainingEndedSess1)
			Expect(queue.Len()).To(Equal(5))

			eventTrainingEndedSess2 := createEvent(domain.EventSessionTrainingEnded, session2Id, 6, time.UnixMilli(5), mockCtrl)
			queue.EnqueueEvent(eventTrainingEndedSess2)
			Expect(queue.Len()).To(Equal(6))

			eventSession1Stopped := createEvent(domain.EventSessionStopped, session1Id, 8, time.UnixMilli(7), mockCtrl)
			queue.EnqueueEvent(eventSession1Stopped)
			Expect(queue.Len()).To(Equal(7))

			eventSession2Stopped := createEvent(domain.EventSessionStopped, session2Id, 9, time.UnixMilli(8), mockCtrl)
			queue.EnqueueEvent(eventSession2Stopped)
			Expect(queue.Len()).To(Equal(8))

			nextEvent = queue.Peek(time.UnixMilli(1))
			Expect(nextEvent).To(Equal(eventSession2Ready))
			Expect(queue.Pop(time.UnixMilli(1))).To(Equal(eventSession2Ready))
			Expect(queue.Len()).To(Equal(7))

			nextEvent = queue.Peek(time.UnixMilli(2))
			Expect(nextEvent).To(Equal(eventSession1Ready))
			Expect(queue.Pop(time.UnixMilli(2))).To(Equal(eventSession1Ready))
			Expect(queue.Len()).To(Equal(6))

			nextEvent = queue.Peek(time.UnixMilli(3))
			Expect(nextEvent).To(Equal(eventTrainingStartedSess1))
			Expect(queue.Pop(time.UnixMilli(3))).To(Equal(eventTrainingStartedSess1))
			Expect(queue.Len()).To(Equal(5))

			nextEvent = queue.Peek(time.UnixMilli(4))
			Expect(nextEvent).To(Equal(eventTrainingStartedSess2))
			Expect(queue.Pop(time.UnixMilli(4))).To(Equal(eventTrainingStartedSess2))
			Expect(queue.Len()).To(Equal(4))

			nextEvent = queue.Peek(time.UnixMilli(5))
			Expect(nextEvent).To(Equal(eventTrainingEndedSess2))
			Expect(queue.Pop(time.UnixMilli(5))).To(Equal(eventTrainingEndedSess2))
			Expect(queue.Len()).To(Equal(3))

			nextEvent = queue.Peek(time.UnixMilli(6))
			Expect(nextEvent).To(Equal(eventTrainingEndedSess1))
			Expect(queue.Pop(time.UnixMilli(6))).To(Equal(eventTrainingEndedSess1))
			Expect(queue.Len()).To(Equal(2))

			nextEvent = queue.Peek(time.UnixMilli(7))
			Expect(nextEvent).To(Equal(eventSession1Stopped))
			Expect(queue.Pop(time.UnixMilli(7))).To(Equal(eventSession1Stopped))
			Expect(queue.Len()).To(Equal(1))

			nextEvent = queue.Peek(time.UnixMilli(8))
			Expect(nextEvent).To(Equal(eventSession2Stopped))
			Expect(queue.Pop(time.UnixMilli(8))).To(Equal(eventSession2Stopped))
			Expect(queue.Len()).To(Equal(0))
		})

		It("Will return an error when delaying an unknown session", func() {
			err := queue.DelaySession("UnknownSession", time.Second*5)
			Expect(errors.Is(err, event_queue.ErrUnregisteredSession)).To(BeTrue())
		})

		It("Will correctly account for delay", func() {
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(0))

			session1Id := "Session1"
			eventSession1Started := createEvent(domain.EventSessionStarted, session1Id, 0, time.UnixMilli(0), mockCtrl)
			queue.EnqueueEvent(eventSession1Started)

			eventSession1Ready := createEvent(domain.EventSessionReady, session1Id, 1, time.UnixMilli(1), mockCtrl)
			queue.EnqueueEvent(eventSession1Ready)

			eventTrainingStartedSess1 := createEvent(domain.EventSessionTrainingStarted, session1Id, 2, time.UnixMilli(2), mockCtrl)
			queue.EnqueueEvent(eventTrainingStartedSess1)

			eventTrainingEndedSess1 := createEvent(domain.EventSessionTrainingEnded, session1Id, 3, time.UnixMilli(3), mockCtrl)
			queue.EnqueueEvent(eventTrainingEndedSess1)

			eventSession1Stopped := createEvent(domain.EventSessionStopped, session1Id, 4, time.UnixMilli(4), mockCtrl)
			queue.EnqueueEvent(eventSession1Stopped)

			session2Id := "Session2"
			eventSession2Started := createEvent(domain.EventSessionStarted, session2Id, 5, time.UnixMilli(10), mockCtrl)
			queue.EnqueueEvent(eventSession2Started)

			eventSession2Ready := createEvent(domain.EventSessionReady, session2Id, 6, time.UnixMilli(11), mockCtrl)
			queue.EnqueueEvent(eventSession2Ready)

			eventTrainingStartedSess2 := createEvent(domain.EventSessionTrainingStarted, session2Id, 7, time.UnixMilli(12), mockCtrl)
			queue.EnqueueEvent(eventTrainingStartedSess2)

			eventTrainingEndedSess2 := createEvent(domain.EventSessionTrainingEnded, session2Id, 8, time.UnixMilli(13), mockCtrl)
			queue.EnqueueEvent(eventTrainingEndedSess2)

			eventSession2Stopped := createEvent(domain.EventSessionStopped, session2Id, 9, time.UnixMilli(14), mockCtrl)
			queue.EnqueueEvent(eventSession2Stopped)

			err := queue.DelaySession(session1Id, time.Millisecond*50)
			Expect(err).To(BeNil())

			By("Returning the events of the non-delayed session first")

			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForSession(session2Id)).To(BeTrue())

			timestamp, err := queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			GinkgoWriter.Printf("GetTimestampOfNextReadyEvent: %v\n", timestamp)
			Expect(timestamp).To(Equal(time.UnixMilli(11)))

			nextEvent := queue.Peek(time.UnixMilli(11))
			Expect(nextEvent).To(Equal(eventSession2Ready))
			Expect(queue.Pop(time.UnixMilli(11))).To(Equal(eventSession2Ready))
			Expect(queue.Len()).To(Equal(7))

			nextEvent = queue.Peek(time.UnixMilli(12))
			Expect(nextEvent).To(Equal(eventTrainingStartedSess2))
			Expect(queue.Pop(time.UnixMilli(12))).To(Equal(eventTrainingStartedSess2))
			Expect(queue.Len()).To(Equal(6))

			nextEvent = queue.Peek(time.UnixMilli(13))
			Expect(nextEvent).To(Equal(eventTrainingEndedSess2))
			Expect(queue.Pop(time.UnixMilli(13))).To(Equal(eventTrainingEndedSess2))
			Expect(queue.Len()).To(Equal(5))

			nextEvent = queue.Peek(time.UnixMilli(14))
			Expect(nextEvent).To(Equal(eventSession2Stopped))
			Expect(queue.Pop(time.UnixMilli(14))).To(Equal(eventSession2Stopped))
			Expect(queue.Len()).To(Equal(4))

			Expect(queue.HasEventsForSession(session2Id)).To(BeFalse())

			By("Returning the events of the delayed session")

			nextEvent = queue.Peek(time.UnixMilli(51))
			Expect(nextEvent).To(Equal(eventSession1Ready))
			Expect(queue.Pop(time.UnixMilli(51))).To(Equal(eventSession1Ready))
			Expect(queue.Len()).To(Equal(3))

			nextEvent = queue.Peek(time.UnixMilli(52))
			Expect(nextEvent).To(Equal(eventTrainingStartedSess1))
			Expect(queue.Pop(time.UnixMilli(52))).To(Equal(eventTrainingStartedSess1))
			Expect(queue.Len()).To(Equal(2))

			nextEvent = queue.Peek(time.UnixMilli(53))
			Expect(nextEvent).To(Equal(eventTrainingEndedSess1))
			Expect(queue.Pop(time.UnixMilli(53))).To(Equal(eventTrainingEndedSess1))
			Expect(queue.Len()).To(Equal(1))

			nextEvent = queue.Peek(time.UnixMilli(54))
			Expect(nextEvent).To(Equal(eventSession1Stopped))
			Expect(queue.Pop(time.UnixMilli(54))).To(Equal(eventSession1Stopped))
			Expect(queue.Len()).To(Equal(0))

			Expect(queue.HasEventsForSession(session1Id)).To(BeFalse())
		})
	})
})
