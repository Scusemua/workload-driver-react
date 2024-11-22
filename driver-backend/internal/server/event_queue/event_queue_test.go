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

			eventTrainingEnded := createEvent(domain.EventSessionTrainingStarted, session1Id, 2, time.UnixMilli(3), mockCtrl)
			queue.EnqueueEvent(eventTrainingEnded)

			Expect(queue.Len()).To(Equal(3))
			Expect(queue.NumSessionQueues()).To(Equal(1))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForTick(time.UnixMilli(0))).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(1))).To(BeTrue())

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			eventSessionStopped := createEvent(domain.EventSessionStopped, session1Id, 2, time.UnixMilli(4), mockCtrl)
			queue.EnqueueEvent(eventSessionStopped)

			Expect(queue.Len()).To(Equal(4))
			Expect(queue.NumSessionQueues()).To(Equal(1))
			Expect(queue.HasEventsForSession(session1Id)).To(BeTrue())
			Expect(queue.HasEventsForTick(time.UnixMilli(0))).To(BeFalse())
			Expect(queue.HasEventsForTick(time.UnixMilli(1))).To(BeTrue())

			timestamp, err = queue.GetTimestampOfNextReadyEvent()
			Expect(err).To(BeNil())
			Expect(timestamp).To(Equal(time.UnixMilli(1)))

			dequeuedEvent, ok := queue.GetNextEvent(time.UnixMilli(1))
			Expect(ok).To(BeTrue())
			Expect(dequeuedEvent).To(Equal(eventSessionReady))
			Expect(queue.Len()).To(Equal(3))
			Expect(queue.NumSessionQueues()).To(Equal(1))

			dequeuedEvent, ok = queue.GetNextEvent(time.UnixMilli(2))
			Expect(ok).To(BeTrue())
			Expect(dequeuedEvent).To(Equal(eventTrainingStarted))
			Expect(queue.Len()).To(Equal(2))
			Expect(queue.NumSessionQueues()).To(Equal(1))

			dequeuedEvent, ok = queue.GetNextEvent(time.UnixMilli(3))
			Expect(ok).To(BeTrue())
			Expect(dequeuedEvent).To(Equal(eventTrainingEnded))
			Expect(queue.Len()).To(Equal(1))
			Expect(queue.NumSessionQueues()).To(Equal(1))

			dequeuedEvent, ok = queue.GetNextEvent(time.UnixMilli(4))
			Expect(ok).To(BeTrue())
			Expect(dequeuedEvent).To(Equal(eventSessionStopped))
			Expect(queue.Len()).To(Equal(0))
			Expect(queue.NumSessionQueues()).To(Equal(1))
		})
	})
})
