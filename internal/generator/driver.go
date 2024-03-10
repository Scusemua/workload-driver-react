package generator

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zhangjyr/gocsv"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
)

var (
	ErrClosed          = errors.New("closed")
	ErrDriverClosed    = errors.New("driver closed")
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

type NewDriver func(int, ...func(Driver)) Driver

// Driver is the interface that wraps the basic methods of a workload_generator.
type Driver interface {
	domain.EventSource
	Id() int
	Drive(context.Context, ...string)
	// Similar to BaseDriver::DriveSync, except Record structs are consumed from the given channel.
	// This is primarily used for testing. We can feed specific records to the `recordChannel` argument,
	// prompting the Driver to generate particular events, which we can then use for testing.
	//
	// The `stopChan` is used to tell the goroutine executing this function to return/exit.
	// DriveWithChannel(context.Context, chan Record, chan struct{}) error
	// Given a pre-populated slice of records, process them one after another. The chan is used to indicate that the Driver is done processing the slice.
	DriveWithSlice(context.Context, []Record, chan struct{}) error
	// OnEvent() <-chan *Event
	// EventSource() chan<- *Event

	// For overwriten
	String() string
	Setup(context.Context) error
	GetRecord() Record
	HandleRecord(context.Context, Record)
	Teardown(context.Context)
}

// QuerableDriver is a driver that optimized for metric query.
// type QuerableDriver interface {
// 	Driver

// Sync notifies the driver that all data queried later should be synced to the time
// specified by the event. Sync returns the event if triggered, call repeatedly until
// EventSynced is returned.
// 	Sync(*Event) *Event
// }

type BaseDriver struct {
	// Driver specifies the implementation struct.
	// Call Driver.Func() if the BaseDriver provides a default implementation.
	Driver

	id              int
	ReadingInterval time.Duration
	RecordProvider  RecordProvider

	LastTimestamp time.Time

	events    chan domain.Event
	eventBuff EventBuff
	lastEvent domain.Event

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

	// Map from Session ID to a bool indicating whether or not the Session is currently training.
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
	return &BaseDriver{
		id:                         id,
		events:                     make(chan domain.Event),
		eventBuff:                  make(EventBuff, 0, 1000),
		SessionMaxes:               make(map[string]float64),
		SessionNumGPUs:             make(map[string]int),
		TrainingMaxes:              make(map[string][]float64),
		TrainingNumGPUs:            make(map[string][]int),
		SessionIsCurrentlyTraining: make(map[string]bool),
		TrainingIndices:            make(map[string]int),
	}
}

// Called in pre-run mode when the Synthesizer encounters a training-started event.
// Sets the value in the latest training max slot to 0.
func (d *BaseDriver) TrainingStarted(podId string) {
	d.MaxesMutex.Lock()
	defer d.MaxesMutex.Unlock()

	if alreadyTraining, ok := d.SessionIsCurrentlyTraining[podId]; ok && alreadyTraining {
		panic(fmt.Sprintf("BaseDriver::TrainingStarted called for already-training Session: Session %s", podId))
	}

	d.SessionIsCurrentlyTraining[podId] = true

	if d.DriverType == "GPU" {
		gpuDriver := d.Driver.(*GPUDriver)

		maxes := gpuDriver.PerGpuTrainingMaxes[podId]
		maxes = append(maxes, []float64{0, 0, 0, 0, 0, 0, 0, 0})
		gpuDriver.PerGpuTrainingMaxes[podId] = maxes
	}

	idx, ok := d.TrainingIndices[podId]
	if !ok {
		idx = 0
	}

	d.TrainingIndices[podId] = idx + 1

	// log.Info("Session %s training #%d", podId, idx+1)

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

// Called in pre-run mode when the Synthesizer encounters a training-stopped event.
// Prepares the next slot in the training maxes by appending to the list a new value of -1.
func (d *BaseDriver) TrainingEnded(podId string) {
	d.MaxesMutex.Lock()
	defer d.MaxesMutex.Unlock()

	d.SessionIsCurrentlyTraining[podId] = false

	// Get the training maxes for the Session whose training just ended.
	podTrainingMaxes, ok := d.TrainingMaxes[podId]
	if !ok {
		log.Error(fmt.Sprintf("[%s] Expected to find set of training maxes for pod \"%s\"", d.DriverType, podId))
		panic(fmt.Sprintf("Expected to find set of training maxes for pod \"%s\"", podId))
	}

	// Put a 0 into the maxes list for the next training event.
	podTrainingMaxes = append(podTrainingMaxes, 0.0)
	d.TrainingMaxes[podId] = podTrainingMaxes

	// If we're a GPU driver, then we also need to update the number of GPUs and the per-GPU-device list.
	if d.DriverType == "GPU" {
		podTrainingNumGPUs, ok2 := d.TrainingNumGPUs[podId]

		if !ok2 {
			log.Error(fmt.Sprintf("[%s] Expected to find set of training numGPU values for pod \"%s\"", d.DriverType, podId))
			panic(fmt.Sprintf("Expected to find set of training numGPU values for pod \"%s\"", podId))
		}

		// Put a 0 for the next training event.
		podTrainingNumGPUs = append(podTrainingNumGPUs, 0)
		d.TrainingNumGPUs[podId] = podTrainingNumGPUs

		// Commented-out: we do this in BaseDriver::TrainingStarted() now.
		// // Now update the per-GPU-device field.
		// maxes := d.Driver.(*GPUDriver).PerGpuTrainingMaxes[podId]
		// maxes = append(maxes, []float64{0, 0, 0, 0, 0, 0, 0, 0})
		// d.Driver.(*GPUDriver).PerGpuTrainingMaxes[podId] = maxes
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

func (d *BaseDriver) DriveWithSlice(ctx context.Context, records []Record, doneChan chan struct{}) error {
	if err := d.Setup(ctx); err != nil {
		return err
	}
	defer d.Teardown(ctx)

	// sugarLog.Debug("There are %d record(s) to process.", len(records))

	for _, record := range records {
		// sugarLog.Debug("Handling record %d/%d: %v.", i+1, len(records), record)
		d.HandleRecord(ctx, record)
	}

	// sugarLog.Debug("Received empty struct on \"Stop Channel\". Exiting now. Processed a total of %d record(s).", len(records))
	d.TriggerEvent(ctx, &eventImpl{
		eventSource:         d,
		originalEventSource: d,
		name:                EventNoMore,
		id:                  uuid.New().String(),
	})

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
	defer manifest.Close()

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
				sugarLog.Warn("Unable to parse csv on line %d(%s): %v", lineNo, mfPaths[i], err)
			}

			select {
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
				sugarLog.Warn("Encountered record with timestamp %v: %v", record.GetTS(), record)
				sugarLog.Warn("Driver's `LastTimestamp` is %v. Finished parsing file \"%s\".", d.LastTimestamp, mfPaths[i])
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
		manifest.Close()
		manifest = nextManifest
	}

	// if d.ExecutionMode == 0 {
	// 	sugarLog.Info("Driver %v is writing its max data to file. Number of records to write: %d.", d.String(), len(d.SessionMaxes))
	// 	file, err := os.Create(d.MaxSessionOutputPath)

	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	if d.DriverType == "GPU" {
	// 		file.WriteString("session_id,max_gpu_utilization,num_gpus\n")
	// 	} else if d.DriverType == "CPU" {
	// 		file.WriteString("session_id,max_cpu_utilization\n")
	// 	} else {
	// 		file.WriteString("session_id,max_memory_bytes\n")
	// 	}

	// 	// log.Info("[%s] Acquiring MaxesMutex lock.", d.DriverType)
	// 	d.MaxesMutex.RLock()
	// 	for pod, val := range d.SessionMaxes {
	// 		if d.DriverType == "GPU" {
	// 			numGPUs := d.SessionNumGPUs[pod]
	// 			_, err = file.WriteString(fmt.Sprintf("%s,%.2f,%d\n", pod, val, numGPUs))
	// 			if err != nil {
	// 				panic(err)
	// 			}
	// 		} else if d.DriverType == "CPU" {
	// 			_, err = file.WriteString(fmt.Sprintf("%s,%.17f\n", pod, val))
	// 			if err != nil {
	// 				panic(err)
	// 			}
	// 		} else {
	// 			_, err = file.WriteString(fmt.Sprintf("%s,%.2f\n", pod, val))
	// 			if err != nil {
	// 				panic(err)
	// 			}
	// 		}
	// 	}
	// 	d.MaxesMutex.RUnlock()

	// 	err = file.Close()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	file_train, err := os.Create(d.MaxTrainingOutputPath)

	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	if d.DriverType == "GPU" {
	// 		file_train.WriteString("session_id,seq,max_gpu_utilization,num_gpus\n")
	// 	} else if d.DriverType == "CPU" {
	// 		file_train.WriteString("session_id,seq,max_cpu_utilization\n")
	// 	} else {
	// 		file_train.WriteString("session_id,seq,max_mem_bytes\n")
	// 	}

	// 	for pod, maxes := range d.TrainingMaxes {
	// 		for training_event_seq_num, max_util := range maxes {
	// 			if d.DriverType == "GPU" {
	// 				numGPUs := d.TrainingNumGPUs[pod]
	// 				_, err = file_train.WriteString(fmt.Sprintf("%s,%d,%.2f,%d\n", pod, training_event_seq_num, max_util, numGPUs[training_event_seq_num]))
	// 				if err != nil {
	// 					panic(err)
	// 				}
	// 			} else if d.DriverType == "CPU" {
	// 				_, err = file_train.WriteString(fmt.Sprintf("%s,%d,%.17f\n", pod, training_event_seq_num, max_util))
	// 				if err != nil {
	// 					panic(err)
	// 				}
	// 			} else {
	// 				_, err = file_train.WriteString(fmt.Sprintf("%s,%d,%.2f\n", pod, training_event_seq_num, max_util))
	// 				if err != nil {
	// 					panic(err)
	// 				}
	// 			}
	// 		}
	// 	}

	// 	if d.DriverType == "GPU" {
	// 		var max_gpu_sess_file, max_gpu_train_file *os.File
	// 		var err error

	// 		// First, per-Session GPU device maxes.
	// 		max_gpu_sess_file, err = os.Create(d.MaxPerGpuSessionOutputPath)
	// 		if err != nil {
	// 			panic(fmt.Sprintf("Failed to create MaxPerGpuSession file at path \"%s\"", d.MaxPerGpuSessionOutputPath))
	// 		}

	// 		max_gpu_sess_file.WriteString("session_id,sum_gpus,gpu0,gpu1,gpu2,gpu3,gpu4,gpu5,gpu6,gpu7\n")

	// 		gpuDriver := d.Driver.(*GPUDriver)

	// 		for pod, vals := range gpuDriver.PerGpuSessionMaxes {
	// 			max_gpu_sess_file.WriteString(fmt.Sprintf("%s,%f,%f,%f,%f,%f,%f,%f,%f,%f\n", pod, sum(vals), vals[0], vals[1], vals[2], vals[3], vals[4], vals[5], vals[6], vals[7]))
	// 		}

	// 		err = max_gpu_sess_file.Close()
	// 		if err != nil {
	// 			panic(fmt.Sprintf("Failed to close MaxPerGpuSession file at path \"%s\"", d.MaxPerGpuSessionOutputPath))
	// 		}

	// 		// Second, per-Session per-Training event GPU device maxes.
	// 		max_gpu_train_file, err = os.Create(d.MaxPerGpuTrainingOutputPath)
	// 		if err != nil {
	// 			panic(fmt.Sprintf("Failed to create MaxPerGpuTraining file at path \"%s\"", d.MaxPerGpuTrainingOutputPath))
	// 		}

	// 		max_gpu_train_file.WriteString("session_id,seq,sum_gpus,gpu0,gpu1,gpu2,gpu3,gpu4,gpu5,gpu6,gpu7\n")

	// 		for pod, maxesForEachTraining := range gpuDriver.PerGpuTrainingMaxes {
	// 			for training_event_seq_num, vals := range maxesForEachTraining {
	// 				max_gpu_train_file.WriteString(fmt.Sprintf("%s,%d,%f,%f,%f,%f,%f,%f,%f,%f,%f\n", pod, training_event_seq_num, sum(vals), vals[0], vals[1], vals[2], vals[3], vals[4], vals[5], vals[6], vals[7]))
	// 			}
	// 		}

	// 		err = max_gpu_train_file.Close()
	// 		if err != nil {
	// 			panic(fmt.Sprintf("Failed to close MaxPerGpuTraining file at path \"%s\"", d.MaxPerGpuSessionOutputPath))
	// 		}
	// 	}

	// 	err = file_train.Close()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	return nil
}

func sum(vals []float64) (sum float64) {
	for _, val := range vals {
		sum += val
	}

	return sum
}

func (d *BaseDriver) Drive(ctx context.Context, mfPaths ...string) {
	// Drain events
	for len(d.events) > 0 {
		<-d.events
	}

	if err := d.DriveSync(ctx, mfPaths...); err != nil {
		triggerErr := d.TriggerError(ctx, err) // If context is done, trigger context.Canceled error will fail anyway.
		if triggerErr != nil {
			sugarLog.Error("Failed on notify error: %v, reason: %v", err, triggerErr)
		}
		return
	}

	d.TriggerEvent(ctx, &eventImpl{
		eventSource:         d,
		originalEventSource: d,
		name:                EventNoMore,
		id:                  uuid.New().String(),
	})
	d.flushEvents()
}

func (d *BaseDriver) Trigger(ctx context.Context, name domain.EventName, rec Record) error {
	return d.TriggerEvent(ctx, &eventImpl{
		eventSource:         d.Driver,
		originalEventSource: d.Driver,
		name:                name,
		data:                rec,
		timestamp:           rec.GetTS(),
		id:                  uuid.New().String(),
	})
}

func (d *BaseDriver) TriggerError(ctx context.Context, e error) error {
	err := d.TriggerEvent(ctx, &eventImpl{
		eventSource:         d.Driver,
		originalEventSource: d.Driver,
		name:                EventError,
		data:                e,
		id:                  uuid.New().String(),
	})
	if err != nil {
		return err
	}

	return d.flushEvents()
}

// TriggerEvent buffers events of same timestamp and call FlushEvent if the timestamp changes.
// The buffer and flush design allows objects' status being updated to the timetick before any event
// of the timetick being triggered.
func (d *BaseDriver) TriggerEvent(ctx context.Context, evt domain.Event) error {
	// if evt.Name != EventTickHolder {
	// 	log.Debug("Triggering driver event: %v", evt)
	// }
	if len(d.eventBuff) > 0 && evt.Timestamp() != d.eventBuff[len(d.eventBuff)-1].Timestamp() {
		err := d.flushEvents()
		if err != nil {
			return err
		}
	}

	// Assign random timestamp offset to simulate stochastic behavior
	// Use negative for new readings to be successfully read.
	// Events triggered in a row will maitain a strict order
	if len(d.eventBuff) > 0 && evt.Data() == d.eventBuff[len(d.eventBuff)-1].Data() { // If there are already buffered events and the data of the last event in the buffer is the same as this event...
		evt.SetOrderSeq(d.eventBuff[len(d.eventBuff)-1].OrderSeq() + 1) // Set to one greater than the event at the end of the event buffer.
	} else {
		// Minus(event happened before the reading) some random ns
		evt.SetOrderSeq(evt.Timestamp().UnixNano() - d.Rand.Int63n(d.ReadingInterval.Nanoseconds()-int64(time.Second))) // Skip 1 second for separation
	}

	d.eventBuff = append(d.eventBuff, evt)
	d.lastEvent = evt
	return nil
}

func (d *BaseDriver) SetId(id int) {
	d.id = id
}

func (d *BaseDriver) FlushEvents(ctx context.Context, timestamp time.Time) error {
	if len(d.eventBuff) == 0 && (d.lastEvent == nil || timestamp != d.lastEvent.Timestamp()) {
		d.TriggerEvent(ctx, &eventImpl{
			eventSource:         d.Driver,
			originalEventSource: d.Driver,
			name:                EventTickHolder,
			timestamp:           timestamp,
			id:                  uuid.New().String(),
		})
	}

	return d.flushEvents()
}

func (d *BaseDriver) flushEvents() error {
	if len(d.eventBuff) == 0 {
		return nil
	}

	// Sort stochastic timestamp in order
	sort.Sort(d.eventBuff)

	for _, evt := range d.eventBuff {
		d.events <- evt
	}
	d.eventBuff = d.eventBuff[:0]
	return nil
}

func (d *BaseDriver) OnEvent() <-chan domain.Event {
	return d.events
}
