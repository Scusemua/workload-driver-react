package generator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"
)

const (
	MemoryActivationThreshold = 0 // 0 to disable activation events
	MemoryDeactivationDelay   = 2 // Deactivate after continous 3 idles
	MemoryStopDelay           = 9 // Stop after continous 10 missing readings
)

var (
	ErrUnexpectedMemoryStTrans = errors.New("unexpected Memory state transition")
)

type MemoryEvent string

func (evt MemoryEvent) String() string {
	return string(evt)
}

const (
	EventMemoryStarted     MemoryEvent = "started"
	EventMemoryActivated   MemoryEvent = "activated"
	EventMemoryDeactivated MemoryEvent = "deactivated"
	EventMemoryStopped     MemoryEvent = "stopped"
)

type MemoryStatus int

const (
	MemoryStopped MemoryStatus = iota
	MemoryIdle
	MemoryIdleDelay
	MemoryBusy
	MemoryStopping
)

// MemoryUtil is used as a buffer to track overall memory utilizations.
// After updated memory readings of the same timestamp, buffered summary are committed to the "LastUtil".
type MemoryUtil struct {
	Timestamp time.Time `json:"timestamp"`
	Pod       string    `json:"pod"`
	Value     float64   `json:"value"`
	Max       float64   `json:"max"`

	// Status shows the current status of the memory.
	Status MemoryStatus `json:"status"`

	// LastUtil stores last committed memory utilization.
	LastUtil *MemoryUtil `json:"lastUtil"`

	NextUtil *MemoryUtil `json:"nextUtil"`

	MaxSessionMemory float64
	MaxTaskMemory    float64

	// Repeat shows how many iterations the status holds.
	Repeat int
}

func (ed *MemoryUtil) String() string {
	return fmt.Sprintf("MemoryUtil[Pod: %s, Value: %.2f%%%%, Max: %.2f%%%%]", ed.Pod, ed.Value, ed.Max)
}

func (ed *MemoryUtil) GetTS() time.Time {
	return ed.Timestamp
}

func (ed *MemoryUtil) GetPod() string {
	return ed.Pod
}

type MemoryUtilBuffer struct {
	mutex sync.Mutex

	// stores look ahead reading
	prototype *MemoryUtil

	// stores current reading
	current *MemoryUtil
}

func (ed *MemoryUtilBuffer) GetTS() time.Time {
	util := ed.Committed()
	if util == nil {
		return time.Time{}
	} else {
		return util.Timestamp
	}
}

func (ed *MemoryUtilBuffer) GetPod() string {
	util := ed.Committed()
	if util == nil {
		return ""
	} else {
		return util.Pod
	}
}

func (ed *MemoryUtilBuffer) Committed() *MemoryUtil {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	return ed.current
}

// Lookup returns the memory utilization of the given timestamp.
func (ed *MemoryUtilBuffer) Lookup(ts time.Time) (util *MemoryUtil) {
	for util = ed.Committed(); util != nil; util = util.LastUtil {
		if util.Timestamp.Sub(ts) <= 0 {
			break
		}
	}
	// if util != nil {
	// 	log.Debug("MemoryUtilBuffer.Lookup for %v, returned %v: %v, current %v:%v", ts, util.Timestamp, util, ed.current.Timestamp, ed.current)
	// }
	return
}

func (ed *MemoryUtilBuffer) Debug_Init(rec *Memory) *MemoryUtil {
	return ed.init(rec)
}

// init initiate the memory utilization with latest memory reading.
func (ed *MemoryUtilBuffer) init(rec *Memory) *MemoryUtil {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	nextUtil := MemoryUtil{
		MaxSessionMemory: 0,
		MaxTaskMemory:    0,
	}

	// ed.prototype = ed.current
	nextUtil.Timestamp = rec.Timestamp.Time()
	nextUtil.Pod = rec.Pod
	nextUtil.Value = rec.Value
	nextUtil.Max = rec.Value

	// For now, this always evaluates to 'false'; however, this could change if we
	// change the value of the 'MemoryActivationThreshold' constant.
	if MemoryActivationThreshold > 0 && nextUtil.Value > MemoryActivationThreshold {
		nextUtil.Status = MemoryBusy
	} else {
		nextUtil.Status = MemoryIdle
	}
	return &nextUtil
}

func (ed *MemoryUtilBuffer) Debug_Commit(rec *MemoryUtil) *MemoryUtil {
	return ed.commit(rec)
}

// commit buffered memory readin as "current"
func (ed *MemoryUtilBuffer) commit(rec *MemoryUtil) *MemoryUtil {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	// Add rec into the linked list.
	rec.LastUtil = ed.prototype

	// Prepare for transiting lookahead reading (prototype) to current reading.
	if ed.prototype != nil {
		ed.prototype.NextUtil = rec
		// Because only the status of current reading is touched by transit(),
		// we move the Repeat intialization from init() to here.
		ed.prototype.Repeat = 0
		if ed.current != nil {
			// Set field "Repeat", MemoryIdleDelay is equivalent to GPUIdel
			eqStatus := ed.current.Status
			if eqStatus == MemoryIdleDelay {
				eqStatus = MemoryIdle
			} else if eqStatus == MemoryStopping {
				eqStatus = MemoryStopped
			}
			if ed.prototype.Status == eqStatus {
				ed.prototype.Repeat = ed.current.Repeat + 1
			}
		}
		ed.current = ed.prototype
	}

	ed.prototype = rec

	// log.Debug("<Current: %v>", ed.current)
	// log.Debug("<Prototype: %v>", ed.prototype)
	// log.Debug("<Rec: %v>\n", rec)

	return ed.current
}

// reset sets memory utilization with no actual reading.
func (ed *MemoryUtilBuffer) reset(time time.Time) *MemoryUtil {
	empty := &MemoryUtil{
		Timestamp:        time,
		Value:            0.0,
		Max:              0.0,
		Repeat:           0,
		Status:           MemoryStopped,
		MaxSessionMemory: 0,
		MaxTaskMemory:    0,
	}
	if ed.prototype != nil {
		ed.mutex.Lock()
		empty.Pod = ed.prototype.Pod
		ed.mutex.Unlock()
	}

	return ed.commit(empty)
}

// transit transits the status of current reading.
// If MemoryActivationThreshold == 0, activation events are disabled.
// We can always generate EventMemoryStarted and EventMemoryStopped events.
func (ed *MemoryUtilBuffer) transit(evtBuff []MemoryEvent, force bool) ([]MemoryEvent, error) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	current := ed.current

	lastStatus := MemoryStopped
	if current != nil && current.LastUtil != nil {
		lastStatus = current.LastUtil.Status
	}

	// Support the detection of series transitions
	for {
		if lastStatus == current.Status {
			return evtBuff, nil
		}

		switch lastStatus {
		case MemoryStopped:
			if current.Status == MemoryIdle || current.Status == MemoryBusy {
				lastStatus = MemoryIdle
				evtBuff = append(evtBuff, EventMemoryStarted)
				continue
			}
			return evtBuff, ErrUnexpectedMemoryStTrans
		case MemoryIdle:
			if current.Status == MemoryBusy {
				lastStatus = MemoryBusy
				evtBuff = append(evtBuff, EventMemoryActivated)
				continue
			} else if current.Status == MemoryStopped && (force || current.Repeat == MemoryStopDelay) {
				lastStatus = MemoryStopped
				evtBuff = append(evtBuff, EventMemoryStopped)
				continue
			} else if current.Status == MemoryStopped {
				current.Status = MemoryStopping
				return evtBuff, nil
			}
			return evtBuff, ErrUnexpectedMemoryStTrans
		case MemoryIdleDelay:
			if current.Status == MemoryIdle && current.Repeat < MemoryDeactivationDelay {
				current.Status = MemoryIdleDelay
				return evtBuff, nil
			} else if current.Status == MemoryIdle || current.Status == MemoryStopped {
				lastStatus = MemoryIdle
				evtBuff = append(evtBuff, EventMemoryDeactivated)
				continue
			} else if current.Status == MemoryBusy {
				lastStatus = MemoryBusy
				continue
			}
			return evtBuff, ErrUnexpectedMemoryStTrans
		case MemoryBusy:
			// We defer deactvate event by MemoryDeactivationDelay
			if (current.Status == MemoryIdle && current.Repeat == MemoryDeactivationDelay) || current.Status == MemoryStopped {
				lastStatus = MemoryIdle
				evtBuff = append(evtBuff, EventMemoryDeactivated)
				continue
			} else if current.Status == MemoryIdle {
				current.Status = MemoryIdleDelay
				return evtBuff, nil
			}
			return evtBuff, ErrUnexpectedMemoryStTrans
		case MemoryStopping:
			// We defer stop event by MemoryStopDelay
			if current.Status == MemoryStopped && !force && current.Repeat < MemoryStopDelay {
				current.Status = MemoryStopping
				return evtBuff, nil
			} else if current.Status == MemoryStopped {
				lastStatus = MemoryStopped
				evtBuff = append(evtBuff, EventMemoryStopped)
				continue
			} else {
				// Reading is available
				lastStatus = MemoryIdle
				continue
			}
		}
	}
}

type Memory struct {
	Timestamp UnixTime `csv:"timestamp"`
	PodIdx    int      `csv:"pod"`
	Value     float64  `csv:"value"`
	Pod       string
}

func (r *Memory) GetTS() time.Time {
	return r.Timestamp.Time()
}

type MemoryMapper struct {
	Pod string `csv:"key"`
}

func (r *MemoryMapper) GetTS() time.Time {
	return time.Time{}
}

type MemoryDriver struct {
	*BaseDriver

	MapperPath string
	Downtimes  []int64

	MemBuffer     *MemoryUtilBuffer
	LastCommitted *MemoryUtil

	podMap    []string
	podMapper *MemoryMapper
	pods      []*MemoryUtilBuffer // []*MemoryUtil
	lastRead  int64               // unix timestamp in second
	interval  time.Duration       // Tick interval detected during driving.
	down      int                 // Indicate trace server was down and no reading during downtime. Odds indicates down.
}

func NewMemoryDriver(id int, configs ...func(TraceDriver)) TraceDriver {
	logger.Debug("Creating MemoryDriver now.\n")
	drv := &MemoryDriver{BaseDriver: NewBaseDriver(id)}
	drv.TraceDriver = drv
	for _, config := range configs {
		config(drv)
	}
	if drv.RecordProvider == nil {
		drv.RecordProvider = &RecordPool{}
	}
	return drv
}

func (d *MemoryDriver) String() string {
	return "Memory"
}

func (d *MemoryDriver) IsDown() bool {
	return d.down%2 == 1
}

func (d *MemoryDriver) Setup(ctx context.Context) error {
	if d.podMapper != nil {
		return nil
	}

	if d.MapperPath == "" {
		d.pods = make([]*MemoryUtilBuffer, 1000)
		sugarLog.Debugf("%v set up, no mapper loaded", d)
		return nil
	}

	d.podMap = make([]string, 0, 1000)
	d.podMapper = &MemoryMapper{}
	err := d.DriveSync(context.TODO(), d.MapperPath)
	d.podMapper = nil
	d.pods = make([]*MemoryUtilBuffer, len(d.podMap))
	sugarLog.Infof("%v set up, mapper loaded, %d entries", d, len(d.podMap))
	return err
}

func (d *MemoryDriver) SetPodMap(podMap []string) {
	d.podMap = podMap
}

func (d *MemoryDriver) Teardown(ctx context.Context) {
	if d.podMapper != nil {
		return
	}

	sugarLog.Debugf("%v tearing down, last read %v", d, d.lastRead)
	if d.lastRead != 0 {
		d.gc(ctx, time.Unix(d.lastRead, 0), false)
		if d.interval == time.Duration(0) {
			d.interval = time.Second
		}
		d.gc(ctx, time.Unix(d.lastRead, int64(d.interval)), true)
	}
	d.pods = nil
	d.podMap = nil
}

func (d *MemoryDriver) GetRecord() Record {
	if d.podMapper != nil {
		return d.podMapper
	}

	r, _ := d.RecordProvider.Get().(*Memory)
	if r != nil {
		return r
	}

	return &Memory{}
}

func (d *MemoryDriver) HandleRecord(ctx context.Context, r Record) {
	if r == d.podMapper {
		d.podMap = append(d.podMap, d.podMapper.Pod)
		return
	}

	defer d.RecordProvider.Recycle(r)

	rec := r.(*Memory)
	rec.Timestamp = UnixTime(rec.Timestamp.Time())
	if d.podMap != nil {
		rec.Pod = d.podMap[rec.PodIdx]
	} else {
		rec.Pod = strconv.Itoa(rec.PodIdx)
	}
	// rec.Value = rec.Value * 100.0 // Normalize utilization to percentage.

	if d.lastRead != 0 && d.lastRead < rec.Timestamp.Time().Unix() {
		ts := time.Unix(d.lastRead, 0)
		interval := rec.Timestamp.Time().Sub(ts)
		down := d.IsDown()
		if d.validateTick(rec.Timestamp.Time(), interval) {
			if down {
				sugarLog.Warnf("Detected memory trace server resumed since %v, resume garbageCollect...", rec.Timestamp.Time())
			}
			d.gc(ctx, ts, false)
			// Current implementation looks ahead of one interval and only generates events with timestamp of lastRead.Timestamp - interval.
			d.FlushEvents(ctx, ts.Add(-interval))
		} else if !down {
			sugarLog.Warnf("Detected memory trace server down since %v, start to skip garbageCollect...", rec.Timestamp.Time())
		}
	}
	d.lastRead = rec.Timestamp.Time().Unix()

	if d.ExecutionMode == 0 {
		d.MaxesMutex.Lock()
		if _, ok := d.TrainingMaxes[rec.Pod]; !ok {
			podTrainingMaxes := make([]float64, 1)
			podTrainingMaxes[0] = -1
			d.TrainingMaxes[rec.Pod] = podTrainingMaxes
		}
		d.MaxesMutex.Unlock()
	}

	memoryBuffer, _ := d.ensurePod(rec)
	// log.Trace("got %v", memoryBuffer)
	events := make([]MemoryEvent, 0, 2) // events buffer

	nextUtil := memoryBuffer.init(rec)
	currentUtil := memoryBuffer.commit(nextUtil)

	d.updateSessionMaxMemory(currentUtil)

	//log.Debug("Next MemoryUtil: %v", nextUtil)
	// log.Debug("Current MemoryUtil: %v\n", currentUtil)
	if currentUtil != nil {
		// If LastUtil is non-null, then we'll compare against the `MaxTaskMemory` and `MaxSessionMemory`
		// fields of the CPUUtil struct stored in the LastUtil field. Otherwise, we'll just set
		// `MaxTaskMemory` and `MaxSessionMemory` to `committed.Max`.
		if currentUtil.LastUtil != nil {
			if currentUtil.LastUtil.MaxTaskMemory > currentUtil.Value {
				// The `MaxTaskMemory` field of `LastUtil` is larger than `committed.Value`,
				// so set `committed.MaxTaskMemory` to `LastUtil.MaxTaskMemory`.
				currentUtil.MaxTaskMemory = currentUtil.LastUtil.MaxTaskMemory
			} else {
				currentUtil.MaxTaskMemory = currentUtil.Value
			}

			if currentUtil.LastUtil.MaxSessionMemory > currentUtil.Value {
				// The `MaxSessionMemory` field of `LastUtil` is larger than `committed.Value`,
				// so set `committed.MaxSessionMemory` to `LastUtil.MaxSessionMemory`.
				currentUtil.MaxSessionMemory = currentUtil.LastUtil.MaxSessionMemory
			} else {
				currentUtil.MaxSessionMemory = currentUtil.Value
			}
		} else {
			currentUtil.MaxTaskMemory = currentUtil.Value
			currentUtil.MaxSessionMemory = currentUtil.Value
		}

		events, err := memoryBuffer.transit(events, false)
		if err != nil {
			sugarLog.Warnf("Error on handling records: %v", err)
		}
		d.triggerMulti(ctx, events, memoryBuffer)
	}
}

// When executing in the "pre-run" mode, we record the maximum memory used for each session.
// This function compares the latest reading from the trace against the maximum memory utilization
// we've recorded for the associated session and updates the record if the latest reading is greater.
func (d *MemoryDriver) updateSessionMaxMemory(committed *MemoryUtil) {
	if d.ExecutionMode == 0 && committed != nil {
		// log.Trace("Committed memory usage of %.2f bytes for session %s", committed.Value, committed.Pod)
		d.MaxesMutex.RLock()
		current_max, ok := d.SessionMaxes[committed.Pod]

		if !ok {
			d.MaxesMutex.RUnlock()
			d.MaxesMutex.Lock()
			d.SessionMaxes[committed.Pod] = committed.Value

			current_training_maxes, ok2 := d.TrainingMaxes[committed.Pod]

			if !ok2 {
				panic(fmt.Sprintf("Expected to find list of training maxes for session \"%s\".", committed.Pod))
			}

			n := len(current_training_maxes)
			if d.SessionIsCurrentlyTraining[committed.Pod] {
				current_training_maxes[n-1] = committed.Value
				d.TrainingMaxes[committed.Pod] = current_training_maxes
			}

			d.MaxesMutex.Unlock()
			return
		}

		if committed.Value > current_max {
			d.MaxesMutex.RUnlock()
			d.MaxesMutex.Lock()
			d.SessionMaxes[committed.Pod] = committed.Value
			d.MaxesMutex.Unlock()
			d.MaxesMutex.RLock()
		}

		current_training_maxes, ok2 := d.TrainingMaxes[committed.Pod]

		if !ok2 {
			panic(fmt.Sprintf("Expected to find list of training maxes for session \"%s\".", committed.Pod))
		}

		n := len(current_training_maxes)
		if d.SessionIsCurrentlyTraining[committed.Pod] && committed.Value > current_training_maxes[n-1] {
			d.MaxesMutex.RUnlock()
			d.MaxesMutex.Lock()
			current_training_maxes[n-1] = committed.Value
			d.TrainingMaxes[committed.Pod] = current_training_maxes
			d.MaxesMutex.Unlock()
		} else {
			d.MaxesMutex.RUnlock()
		}
	}
}

func (d *MemoryDriver) ensurePod(rec *Memory) (buf *MemoryUtilBuffer, created bool) {
	if cap(d.pods) <= rec.PodIdx {
		// pods := make([]*MemoryUtil, int(math.Ceil(float64(rec.PodIdx+1)/float64(cap(d.pods))))*cap(d.pods))
		pods := make([]*MemoryUtilBuffer, int(math.Ceil(float64(rec.PodIdx+1)/float64(cap(d.pods))))*cap(d.pods))
		copy(pods[:cap(d.pods)], d.pods)
		d.pods = pods
	}
	if d.pods[rec.PodIdx] == nil {
		created = true
		d.pods[rec.PodIdx] = &MemoryUtilBuffer{}
	}
	return d.pods[rec.PodIdx], created
}

func (d *MemoryDriver) validateTick(ts time.Time, interval time.Duration) bool {
	// Calibrate downtime.
	for d.down < len(d.Downtimes) && ts.Unix() >= d.Downtimes[d.down] {
		d.down++
	}

	// Is the trace server down?
	if d.IsDown() {
		return false
	}

	if d.interval == time.Duration(0) || interval <= d.interval {
		d.interval = interval
	}
	return true
}

func (d *MemoryDriver) triggerMulti(ctx context.Context, names []MemoryEvent, data *MemoryUtilBuffer) error {
	if len(names) == 0 {
		return nil
	}
	for _, name := range names {
		if err := d.Trigger(ctx, name, data); err != nil {
			return err
		}
	}
	return nil
}

// gc handles pods have no reading at specified time.
func (d *MemoryDriver) gc(ctx context.Context, ts time.Time, force bool) error {
	events := make([]MemoryEvent, 0, 2) // events buffer
	var err error
	for _, pod := range d.pods {
		// Ignore unseen, read at specified time, or stopped
		if pod == nil || pod.prototype == nil || pod.prototype.Timestamp == ts || pod.prototype.Status == MemoryStopped {
			continue
		}

		// Reset readings
		committed := pod.reset(ts)
		// log.Trace("fill reading %d, %v", pod.prototype.Timestamp.Unix(), pod.prototype)

		events, err = pod.transit(events, force)
		if err != nil {
			sugarLog.Warnf("Error on commiting last readings in garbageCollect: %v, %v", err, committed)
		}
		if err := d.triggerMulti(ctx, events, pod); err != nil {
			return err
		}
		events = events[:0]
	}
	return nil
}
