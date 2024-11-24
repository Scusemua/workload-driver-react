package domain_test

import (
	"container/heap"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/scusemua/workload-driver-react/m/v2/internal/mock_domain"
	"go.uber.org/mock/gomock"
	"time"

	. "github.com/onsi/gomega"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

var _ = Describe("EventHeap Tests", func() {
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
			SessionId:   sessionId,
			ID:          uuid.NewString(),
			Timestamp:   timestamp,
		}
	}

	It("Will correctly enqueue and dequeue a multiple events", func() {
		sessionId := uuid.NewString()
		eventHeap := make(domain.EventHeap, 0)

		evt1 := createEvent(domain.EventSessionReady, sessionId, 0, time.UnixMilli(0))
		evt2 := createEvent(domain.EventSessionTrainingStarted, sessionId, 1, time.UnixMilli(1))
		evt3 := createEvent(domain.EventSessionTrainingEnded, sessionId, 2, time.UnixMilli(2))
		evt4 := createEvent(domain.EventSessionStopped, sessionId, 3, time.UnixMilli(3))

		By("Correctly pushing events onto its internal queue")

		heap.Push(&eventHeap, evt4)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(eventHeap.Peek()).To(Equal(evt4))

		heap.Push(&eventHeap, evt3)
		Expect(eventHeap.Len()).To(Equal(2))
		Expect(eventHeap.Peek()).To(Equal(evt3))

		heap.Push(&eventHeap, evt2)
		Expect(eventHeap.Len()).To(Equal(3))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		heap.Push(&eventHeap, evt1)
		Expect(eventHeap.Len()).To(Equal(4))
		Expect(eventHeap.Peek()).To(Equal(evt1))

		By("Correctly popping events off of its internal queue")

		evt := heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(3))
		Expect(evt).To(Equal(evt1))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(2))
		Expect(evt).To(Equal(evt2))
		Expect(eventHeap.Peek()).To(Equal(evt3))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt3))
		Expect(eventHeap.Peek()).To(Equal(evt4))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt4))
		Expect(eventHeap.Peek() == nil).To(BeTrue())
	})

	It("Will correctly break ties using the global indices of events", func() {
		sessionId := uuid.NewString()
		eventHeap := make(domain.EventHeap, 0)
		Expect(eventHeap.Len()).To(Equal(0))

		evt1 := createEvent(domain.EventSessionTrainingStarted, sessionId, 0, time.UnixMilli(0))
		evt2 := createEvent(domain.EventSessionTrainingEnded, sessionId, 1, time.UnixMilli(0))

		Expect(evt1.Timestamp).To(Equal(evt2.Timestamp))
		Expect(evt1.OriginalTimestamp).To(Equal(evt2.OriginalTimestamp))

		heap.Push(&eventHeap, evt2)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		heap.Push(&eventHeap, evt1)
		Expect(eventHeap.Len()).To(Equal(2))
		Expect(eventHeap.Peek()).To(Equal(evt1))

		evt := heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt1))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt2))
		Expect(eventHeap.Peek() == nil).To(BeTrue())

		// Note that these are not in the order you'd inspect just based on their names.
		evt3 := createEvent(domain.EventSessionTrainingStarted, sessionId, 3, time.UnixMilli(0))
		evt4 := createEvent(domain.EventSessionTrainingEnded, sessionId, 2, time.UnixMilli(0))

		heap.Push(&eventHeap, evt3)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(eventHeap.Peek()).To(Equal(evt3))

		heap.Push(&eventHeap, evt4)
		Expect(eventHeap.Len()).To(Equal(2))
		Expect(eventHeap.Peek()).To(Equal(evt4))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt4))
		Expect(eventHeap.Peek()).To(Equal(evt3))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt3))
		Expect(eventHeap.Peek() == nil).To(BeTrue())
	})

	It("Will correctly break ties by placing 'training-ended' events before 'session-stopped' events", func() {
		sessionId := uuid.NewString()
		eventHeap := make(domain.EventHeap, 0)
		Expect(eventHeap.Len()).To(Equal(0))

		// Note that these were assigned the same indices.
		evt1 := createEvent(domain.EventSessionStopped, sessionId, 1, time.UnixMilli(0))
		evt2 := createEvent(domain.EventSessionTrainingEnded, sessionId, 0, time.UnixMilli(0))

		By("Doing so when we push the 'training-ended' event onto the queue first.")

		heap.Push(&eventHeap, evt2)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		heap.Push(&eventHeap, evt1)
		Expect(eventHeap.Len()).To(Equal(2))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		evt := heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt2))
		Expect(eventHeap.Peek()).To(Equal(evt1))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt1))
		Expect(eventHeap.Peek() == nil).To(BeTrue())

		By("Doing so when we push the 'session-stopped' event onto the queue first.")
		eventHeap = make(domain.EventHeap, 0)
		Expect(eventHeap.Len()).To(Equal(0))

		evt2 = createEvent(domain.EventSessionStopped, sessionId, 1, time.UnixMilli(0))
		evt1 = createEvent(domain.EventSessionTrainingEnded, sessionId, 0, time.UnixMilli(0))

		heap.Push(&eventHeap, evt2)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		heap.Push(&eventHeap, evt1)
		Expect(eventHeap.Len()).To(Equal(2))
		Expect(eventHeap.Peek()).To(Equal(evt1))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(1))
		Expect(evt).To(Equal(evt1))
		Expect(eventHeap.Peek()).To(Equal(evt2))

		evt = heap.Pop(&eventHeap)
		Expect(eventHeap.Len()).To(Equal(0))
		Expect(evt).To(Equal(evt2))
		Expect(eventHeap.Peek() == nil).To(BeTrue())
	})
})
