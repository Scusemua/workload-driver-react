package generator

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zhangjyr/gocsv"
	"go.uber.org/zap"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

var (
	ErrNoPathSpecified = errors.New("no path specified")
)

type DriverEvent string

func (evt DriverEvent) String() string {
	return string(evt)
}

// Some "special" events.
const (
	EventError      DriverEvent = "error"
	EventTickHolder DriverEvent = "tickholder"
	EventNoMore     DriverEvent = "nomore"
)

type NewDriver func(int, ...func(TraceDriver)) TraceDriver

// TraceDriver is the interface that wraps the basic methods of a workload_generator.
type TraceDriver interface {
	domain.EventSource
	Id() int
	Drive(context.Context, ...string)
	// DriveWithSlice is similar to BaseDriver::DriveSync, except Record structs are consumed from the given channel.
	// This is primarily used for testing. We can feed specific records to the `recordChannel` argument,
	// prompting the TraceDriver to generate particular events, which we can then use for testing.
	//
	// The `stopChan` is used to tell the goroutine executing this function to return/exit.
	// DriveWithChannel(context.Context, chan Record, chan struct{}) error
	// Given a pre-populated slice of records, process them one after another. The chan is used to indicate that the TraceDriver is done processing the slice.
	//
	// The error chan is used to report errors back to the main goroutine, as this is generally called in its own goroutine,
	// so the returned error will not make it to the main goroutine.
	DriveWithSlice(context.Context, []Record, chan struct{}, chan<- error) error
	// OnEvent() <-chan *Event
	// EventSource() chan<- *Event

	String() string
	Setup(context.Context) error
	GetRecord() Record
	HandleRecord(context.Context, Record)
	Teardown(context.Context)
}

type BaseDriver struct {
	// TraceDriver specifies the implementation struct.
	// Call TraceDriver.Func() if the BaseDriver provides a default implementation.
	TraceDriver

	log      *zap.Logger
	sugarLog *zap.SugaredLogger

	id              int
	ReadingInterval time.Duration
	RecordProvider  RecordProvider

	LastTimestamp time.Time

	events    chan *domain.Event
	eventBuff domain.EventHeap
	lastEvent *domain.Event

	// Synchronizes concurrent access to the maps containing maximum utilization values.
	MaxesMutex sync.RWMutex
	// Maximum utilization values for each Session across each training event that the Session processes.
	TrainingMaxes map[string][]float64
	// Maximum utilization values for each Session across its entire lifetime.
	SessionMaxes map[string]float64
	// The (maximum) number of GPUs used by the Session during its lifetime.
	SessionNumGPUs map[string]int
	// The number of GPUs used by the Session during each training event that the Session processes.
	TrainingNumGPUs map[string][]int

	TrainingIndices map[string]int

	// Map from Session ID to a bool indicating whether the Session is currently training.
	SessionIsCurrentlyTraining map[string]bool

	MaxSessionOutputPath  string
	MaxTrainingOutputPath string

	MaxPerGpuSessionOutputPath  string
	MaxPerGpuTrainingOutputPath string

	ExecutionMode int

	Rand *rand.Rand

	DriverType string
}

func NewBaseDriver(id int) *BaseDriver {
	driver := &BaseDriver{
		id:                         id,
		events:                     make(chan *domain.Event),
		eventBuff:                  make(domain.EventHeap, 0, 1000),
		SessionMaxes:               make(map[string]float64),
		SessionNumGPUs:             make(map[string]int),
		TrainingMaxes:              make(map[string][]float64),
		TrainingNumGPUs:            make(map[string][]int),
		SessionIsCurrentlyTraining: make(map[string]bool),
		TrainingIndices:            make(map[string]int),
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	driver.log = logger
	driver.sugarLog = logger.Sugar()

	return driver
}

// TrainingStarted is called in pre-run mode when the Synthesizer encounters a training-started event.
// Sets the value in the latest training max slot to 0.
func (d *BaseDriver) TrainingStarted(podId string) {
	d.MaxesMutex.Lock()
	defer d.MaxesMutex.Unlock()

	if alreadyTraining, ok := d.SessionIsCurrentlyTraining[podId]; ok && alreadyTraining {
		panic(fmt.Sprintf("BaseDriver::TrainingStarted called for already-training Session: Session %s", podId))
	}

	d.SessionIsCurrentlyTraining[podId] = true

	if d.DriverType == "GPU" {
		gpuDriver := d.TraceDriver.(*GPUDriver)

		maxes := gpuDriver.PerGpuTrainingMaxes[podId]
		maxes = append(maxes, []float64{0, 0, 0, 0, 0, 0, 0, 0})
		gpuDriver.PerGpuTrainingMaxes[podId] = maxes
	}

	idx, ok := d.TrainingIndices[podId]
	if !ok {
		idx = 0
	}

	d.TrainingIndices[podId] = idx + 1

	// log.Info("Session %s training #%d", podId, heapIndex+1)

	// podTrainingMaxes, ok := d.TrainingMaxes[podId]

	// if !ok {
	// 	log.Error(fmt.Sprintf("[%s] Expected to find set of training maxes for pod \"%s\"", d.DriverType, podId))
	// 	panic(fmt.Sprintf("Expected to find set of training maxes for pod \"%s\"", podId))
	// }

	// n := len(podTrainingMaxes)

	// d.MaxesMutex.RUnlock()
	// d.MaxesMutex.Lock()
	// defer d.MaxesMutex.Unlock()

	// // Set to 0 so that we start recording them now.
	// podTrainingMaxes[n-1] = 0.0
	// d.TrainingMaxes[podId] = podTrainingMaxes
}

// TrainingEnded is called in pre-run mode when the Synthesizer encounters a training-stopped event.
// Prepares the next slot in the training maxes by appending to the list a new value of -1.
func (d *BaseDriver) TrainingEnded(podId string) {
	d.MaxesMutex.Lock()
	defer d.MaxesMutex.Unlock()

	d.SessionIsCurrentlyTraining[podId] = false

	// Get the training maxes for the Session whose training just ended.
	podTrainingMaxes, ok := d.TrainingMaxes[podId]
	if !ok {
		logger.Error(fmt.Sprintf("[%s] Expected to find set of training maxes for pod \"%s\"", d.DriverType, podId))
		panic(fmt.Sprintf("Expected to find set of training maxes for pod \"%s\"", podId))
	}

	// Put a 0 into the maxes list for the next training event.
	podTrainingMaxes = append(podTrainingMaxes, 0.0)
	d.TrainingMaxes[podId] = podTrainingMaxes

	// If we're a GPU driver, then we also need to update the number of GPUs and the per-GPU-device list.
	if d.DriverType == "GPU" {
		podTrainingNumGPUs, ok2 := d.TrainingNumGPUs[podId]

		if !ok2 {
			logger.Error(fmt.Sprintf("[%s] Expected to find set of training numGPU values for pod \"%s\"", d.DriverType, podId))
			panic(fmt.Sprintf("Expected to find set of training numGPU values for pod \"%s\"", podId))
		}

		// Put a 0 for the next training event.
		podTrainingNumGPUs = append(podTrainingNumGPUs, 0)
		d.TrainingNumGPUs[podId] = podTrainingNumGPUs

		// Commented-out: we do this in BaseDriver::TrainingStarted() now.
		// // Now update the per-GPU-device field.
		// maxes := d.TraceDriver.(*GPUDriver).PerGpuTrainingMaxes[podId]
		// maxes = append(maxes, []float64{0, 0, 0, 0, 0, 0, 0, 0})
		// d.TraceDriver.(*GPUDriver).PerGpuTrainingMaxes[podId] = maxes
	}
}

func (d *BaseDriver) IsDriver() bool {
	return true
}

func (d *BaseDriver) Id() int {
	return d.id
}

func (d *BaseDriver) IsLastShouldContinue() bool {
	return true
}

func (d *BaseDriver) DriveWithSlice(ctx context.Context, records []Record, doneChan chan struct{}, errorChan chan<- error) error {
	if err := d.Setup(ctx); err != nil {
		d.log.Error("Failed to setup driver.", zap.Error(err))
		errorChan <- err
		return err
	}
	defer d.Teardown(ctx)

	d.sugarLog.Debugf("There are %d record(s) to process.", len(records))

	for _, record := range records {
		// sugarLog.Debugf("Handling record %d/%d: %v.", i+1, len(records), record)
		d.HandleRecord(ctx, record)
	}

	d.sugarLog.Debugf("Finished processing all %d record(s).", len(records))
	err := d.TriggerEvent(ctx, &domain.Event{
		EventSource:         d,
		OriginalEventSource: d,
		Name:                EventNoMore,
		SessionId:           "N/A",
		ID:                  uuid.New().String(),
	})
	if err != nil {
		d.sugarLog.Warnf("Error while triggering events: %v", err)
	}

	doneChan <- struct{}{}

	return context.Canceled
}

func (d *BaseDriver) DriveSync(ctx context.Context, mfPaths ...string) error {
	if len(mfPaths) == 0 {
		return ErrNoPathSpecified
	}

	if err := d.Setup(ctx); err != nil {
		return err
	}
	defer d.Teardown(ctx)

	manifest, err := os.Open(mfPaths[0])
	if err != nil {
		return err
	}
	defer func(manifest *os.File) {
		err := manifest.Close()
		if err != nil {
			log.Printf("[ERROR] Failed to close manifest: %v\n", err)
		}
	}(manifest)

	defaultTime := time.Time{}

	for i := 0; true; {
		reader := gocsv.NewSimpleDecoderFromCSVReader(csv.NewReader(manifest))
		record := d.GetRecord()

		ctxCSV, err := gocsv.UnmarshalDecoderWithContext(ctx, reader, record) // use clean context to start read a file.
		lineNo := 2
		for err != io.EOF {
			if err == nil {
				d.HandleRecord(ctx, record)
			} else {
				sugarLog.Warnf("Unable to parse csv on line %d(%s): %v", lineNo, mfPaths[i], err)
			}

			select {
			case <-ctx.Done():
				// TODO(Ben): I don't think we need this? It's covered by the ctxCSV.Done(), isn't it?
				// I don't know if it is because ctxCSV is created with a *copy* of ctx as its parent.
				// Nothing need to be done.
				return context.Canceled
			case <-ctxCSV.Done():
				// Nothing need to be done.
				return context.Canceled
			// case <-time.After(time.Second):
			default:
			}

			record = d.GetRecord()
			ctxCSV, err = gocsv.UnmarshalDecoderWithContext(ctxCSV, reader, record)
			lineNo++

			if d.LastTimestamp != defaultTime && record.GetTS().After(d.LastTimestamp) {
				sugarLog.Warnf("Encountered record with Timestamp %v: %v", record.GetTS(), record)
				sugarLog.Warnf("TraceDriver's `LastTimestamp` is %v. Finished parsing file \"%s\".", d.LastTimestamp, mfPaths[i])
				break
			}
		}

		i++
		if i >= len(mfPaths) {
			break
		}

		nextManifest, openErr := os.Open(mfPaths[i])
		if openErr != nil {
			return openErr
		}
		err = manifest.Close()
		if err != nil {
			d.sugarLog.Warnf("Error while closing manifest: %v", err)
		}
		manifest = nextManifest
	}

	return nil
}

func (d *BaseDriver) Drive(ctx context.Context, mfPaths ...string) {
	// Drain events
	for len(d.events) > 0 {
		<-d.events
	}

	if err := d.DriveSync(ctx, mfPaths...); err != nil {
		triggerErr := d.TriggerError(ctx, err) // If context is done, trigger context.Canceled error will fail anyway.
		if triggerErr != nil {
			sugarLog.Errorf("Failed on notify error: %v, reason: %v", err, triggerErr)
		}
		return
	}

	if err := d.TriggerEvent(ctx, &domain.Event{
		EventSource:         d,
		OriginalEventSource: d,
		Name:                EventNoMore,
		ID:                  uuid.New().String(),
		SessionId:           "N/A",
	}); err != nil {
		d.sugarLog.Warnf("Error while triggering events: %v", err)
	}

	if err := d.flushEvents(); err != nil {
		d.sugarLog.Warnf("Error while triggering events: %v", err)
	}
}

func (d *BaseDriver) Trigger(ctx context.Context, Name domain.EventName, rec Record) error {
	return d.TriggerEvent(ctx, &domain.Event{
		EventSource:         d.TraceDriver,
		OriginalEventSource: d.TraceDriver,
		Name:                Name,
		Data:                rec,
		Timestamp:           rec.GetTS(),
		OriginalTimestamp:   rec.GetTS(),
		ID:                  uuid.New().String(),
	})
}

func (d *BaseDriver) TriggerError(ctx context.Context, e error) error {
	err := d.TriggerEvent(ctx, &domain.Event{
		EventSource:         d.TraceDriver,
		OriginalEventSource: d.TraceDriver,
		Name:                EventError,
		SessionId:           "N/A",
		Data:                e,
		ID:                  uuid.New().String(),
	})
	if err != nil {
		return err
	}

	return d.flushEvents()
}

// TriggerEvent buffers events of same Timestamp and call FlushEvent if the Timestamp changes.
// The buffer and flush design allows objects' status being updated to the timetick before any event
// of the timetick being triggered.
func (d *BaseDriver) TriggerEvent(_ context.Context, evt *domain.Event) error {
	if len(d.eventBuff) > 0 && evt.Timestamp != d.eventBuff[len(d.eventBuff)-1].Timestamp {
		err := d.flushEvents()
		if err != nil {
			return err
		}
	}

	// Assign random Timestamp offset to simulate stochastic behavior
	// Use negative for new readings to be successfully read.
	// Events triggered in a row will maitain a strict order
	if len(d.eventBuff) > 0 && evt.Data == d.eventBuff[len(d.eventBuff)-1].Data { // If there are already buffered events and the Data of the last event in the buffer is the same as this event...
		evt.OrderSeq = d.eventBuff[len(d.eventBuff)-1].OrderSeq + 1 // Set to one greater than the event at the end of the event buffer.
	} else {
		// Minus(event happened before the reading) some random ns
		evt.OrderSeq = evt.Timestamp.UnixNano() - d.Rand.Int63n(d.ReadingInterval.Nanoseconds()-int64(time.Second)) // Skip 1 second for separation
	}

	d.eventBuff = append(d.eventBuff, evt)
	d.lastEvent = evt
	return nil
}

func (d *BaseDriver) SetId(id int) {
	d.id = id
}

func (d *BaseDriver) FlushEvents(ctx context.Context, Timestamp time.Time) error {
	if len(d.eventBuff) == 0 && (d.lastEvent == nil || Timestamp != d.lastEvent.Timestamp) {
		err := d.TriggerEvent(ctx, &domain.Event{
			EventSource:         d.TraceDriver,
			OriginalEventSource: d.TraceDriver,
			Name:                EventTickHolder,
			SessionId:           "N/A",
			Timestamp:           Timestamp,
			ID:                  uuid.New().String(),
		})

		if err != nil {
			d.sugarLog.Errorf("Error while triggering events: %v", err)
			return err
		}
	}

	return d.flushEvents()
}

func (d *BaseDriver) flushEvents() error {
	if len(d.eventBuff) == 0 {
		return nil
	}

	// Sort stochastic Timestamp in order
	sort.Sort(d.eventBuff)

	for _, evt := range d.eventBuff {
		d.events <- evt
	}
	d.eventBuff = d.eventBuff[:0]
	return nil
}

func (d *BaseDriver) OnEvent() <-chan *domain.Event {
	return d.events
}
