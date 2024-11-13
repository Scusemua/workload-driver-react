package generator

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	NoGPUEvent GPUEvent = "none"
)

func createUtil(reading float64) *GPUUtil {
	util := &GPUUtil{GPUName: AnyGPU}
	util.init(&GPURecord{Timestamp: UnixTime(time.Now()), Value: reading})
	return util
}

func createEventsBuff() []GPUEvent {
	return make([]GPUEvent, 2)
}

func validateEvent(evt GPUEvent, events []GPUEvent) GPUEvent {
	for _, event := range events {
		if evt == event {
			return evt
		}
	}
	return NoGPUEvent
}

var _ = Describe("GPUUtil", func() {
	It("should the GPUUtil be initialized properly", func() {
		util := createUtil(0.0)
		Expect(util.Timestamp).To(Not(Equal(time.Time{})))
		Expect(util.Status).To(Equal(GPUIdle))

		util = createUtil(10.0)
		Expect(util.Timestamp).To(Not(Equal(time.Time{})))
		Expect(util.Status).To(Equal(GPUBusy))
	})

	It("should event 'started' triggered when status turn from stopped to idle", func() {
		// Buffering is skipped, create committed GPUUtil directly.
		util := createUtil(0.0)
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUStarted, events)).To(Equal(EventGPUStarted))
		Expect(util.Status).To(Equal(GPUIdle))
	})

	It("should event 'started' triggered when status turn from stopped to busy", func() {
		// Buffering is skipped, create committed GPUUtil directly.
		util := createUtil(10.0)
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUStarted, events)).To(Equal(EventGPUStarted))
		Expect(util.Status).To(Equal(GPUBusy))
	})

	It("should event 'activated' triggered when status turn from stopped to busy", func() {
		// Buffering is skipped, create committed GPUUtil directly.
		util := createUtil(10.0)
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUActivated, events)).To(Equal(EventGPUActivated))
		Expect(util.Status).To(Equal(GPUBusy))
	})

	It("should event 'activated' triggered when status turn from idle to busy", func() {
		// Buffering is skipped, create committed GPUUtil directly.
		util := createUtil(0.0)
		util.commitAndInit(&GPURecord{Timestamp: UnixTime(time.Now()), Value: 10.0})
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUActivated, events)).To(Equal(EventGPUActivated))
		Expect(util.Status).To(Equal(GPUBusy))
	})

	It("should not event 'deactivated' triggered when status turn from busy to idle within GPUDeactivationDelay", func() {
		// Buffering is skipped, create committed GPUUtil directly.
		util := createUtil(10.0)
		util.commitAndInit(&GPURecord{Timestamp: UnixTime(time.Now()), Value: 0.0})
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUDeactivated, events)).To(Equal(NoGPUEvent))
		Expect(util.Status).To(Equal(GPUIdleDelay))
	})

	It("should not event 'activated' triggered when status turn from idle back to busy within GPUDeactivationDelay", func() {
		// Buffering is skipped, create committed GPUUtil directly.
		util := createUtil(10.0)
		events := createEventsBuff()
		util.commitAndInit(&GPURecord{Timestamp: UnixTime(time.Now()), Value: 0.0})
		events, _ = util.transit(events, false)
		events = events[:0]
		Expect(util.Status).To(Equal(GPUIdleDelay))

		util.commitAndInit(&GPURecord{Timestamp: UnixTime(time.Now()), Value: 10.0})
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUActivated, events)).To(Equal(NoGPUEvent))
		Expect(util.Status).To(Equal(GPUBusy))
	})

	It("should event 'deactivated' triggered when status turn from busy to idle beyond GPUDeactivationDelay", func() {
		// Buffering is skipped, create committed GPUUtil directly.
		util := createUtil(10.0)
		events := createEventsBuff()
		util.commitAndInit(&GPURecord{Timestamp: UnixTime(time.Now()), Value: 0.0})
		for i := 0; i < GPUDeactivationDelay; i++ {
			events, _ = util.transit(events, false)
			events = events[:0]
			Expect(util.Status).To(Equal(GPUIdleDelay))
			util.commitAndInit(&GPURecord{Timestamp: UnixTime(time.Now()), Value: 0.0})
			util.Repeat = i + 1
		}
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUDeactivated, events)).To(Equal(EventGPUDeactivated))
		Expect(util.Status).To(Equal(GPUIdle))
	})

	It("should event 'deactivated' triggered when status turn from busy to stopped", func() {
		// No commitAndReset implemented, so we will have to go buffering way.
		buff := createUtil(10.0)
		buff.commit()
		util := buff.reset(time.Now())
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUDeactivated, events)).To(Equal(EventGPUDeactivated))
		Expect(util.Status).To(Equal(GPUStopped))
	})

	It("should event 'stop' triggered when status turn from busy to stopped", func() {
		// No commitAndReset implemented, so we will have to go buffering way.
		buff := createUtil(10.0)
		buff.commit()
		util := buff.reset(time.Now())
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUStopped, events)).To(Equal(EventGPUStopped))
		Expect(util.Status).To(Equal(GPUStopped))
	})

	It("should event 'stop' triggered when status turn from idle to stopped", func() {
		// No commitAndReset implemented, so we will have to go buffering way.
		buff := createUtil(0.0)
		buff.commit()
		util := buff.reset(time.Now())
		events := createEventsBuff()
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUStopped, events)).To(Equal(EventGPUStopped))
		Expect(util.Status).To(Equal(GPUStopped))
	})

	It("should event 'stop' triggered when status turn from idleDelay to stopped", func() {
		// No commitAndReset implemented, so we will have to go buffering way.
		buff := createUtil(10.0)
		buff.commitAndInit(&GPURecord{Timestamp: UnixTime(time.Now()), Value: 0.0})
		util := buff.commit()
		events := createEventsBuff()
		events, _ = util.transit(events, false)
		events = events[:0]
		Expect(util.Status).To(Equal(GPUIdleDelay))

		util = buff.reset(time.Now())
		events, err := util.transit(events, false)
		Expect(err).To(BeNil())
		Expect(validateEvent(EventGPUStopped, events)).To(Equal(EventGPUStopped))
		Expect(util.Status).To(Equal(GPUStopped))
	})

	// TODO: Add test case for GPUStopDelay
	// TODO: Add test case for GPUStopDelay by force
})
