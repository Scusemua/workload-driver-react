package generator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap"
)

const (
	CPUDeactivationDelay   = 2                  // Deactivate after continous 3 idles
	CPUStopDelay           = 9                  // Stop after continous 10 missing readings
	CPURateOffset          = -120 * time.Second // Readings delay due to prometheus rate() extrapolation.
	CPUActivationThreshold = 1                  // rate(reading[5m])
)

var (
	ErrUnexpectedCPUStTrans = errors.New("unexpected CPU state transition")
	logger, _               = zap.NewDevelopment()
	sugarLog                = logger.Sugar()
)

type CPUEvent string

func (evt CPUEvent) String() string {
	return string(evt)
}

const (
	EventCPUStarted     CPUEvent = "started"
	EventCPUActivated   CPUEvent = "activated"
	EventCPUDeactivated CPUEvent = "deactivated"
	EventCPUStopped     CPUEvent = "stopped"
)

type CPUStatus int

const (
	CPUStopped CPUStatus = iota
	CPUIdle
	CPUIdleDelay
	CPUBusy
	CPUStopping
)

// CPUUtil is used as a buffer to track overall CPU utilizations with the number of CPU cores used.
// After updated CPU readings of the same timestamp, buffered summary are committed to the "LastUtil".
type CPUUtil struct {
	// prototype keeps track of the original object provisioned by the generator.
	prototype *CPUUtil

	Timestamp      time.Time   `json:"timestamp"`
	Pod            string      `json:"pod"`
	Value          float64     `json:"cpu_utilization"`
	Max            float64     `json:"max"`
	Status         CPUStatus   `json:"status"`
	CurrentSession SessionMeta `json:"currentSession"`

	MaxTaskCPU    float64 `json:"maxTaskCPU"`
	MaxSessionCPU float64 `json:"maxSessionCPU"`

	// Repeat shows how many iterations the status holds.
	Repeat int `json:"repeat"`

	// LastUtil stores last committed CPU utilization.
	LastUtil *CPUUtil `json:"-"`
}

func (ed *CPUUtil) Debug_SetPrototypeSelf() {
	ed.prototype = ed
}

func (ed *CPUUtil) String() string {
	return fmt.Sprintf("Pod: %s, %.2f%%%%", ed.Pod, ed.Value)
}

func (ed *CPUUtil) GetTS() time.Time {
	return ed.Timestamp
}

func (ed *CPUUtil) GetPod() string {
	return ed.Pod
}

func (ed *CPUUtil) Committed() *CPUUtil {
	if ed.prototype == nil {
		logger.Error("CPUUtil::Committed cannot return as `prototype` field is null for CPUUtil struct.", zap.Any("cpuutil", ed))
		panic("Error")
	}
	return ed.prototype
}

// init initiate the CPU utilization with latest CPU reading.
func (ed *CPUUtil) init(rec *CPURecord) *CPUUtil {
	ed.prototype = ed
	ed.Timestamp = rec.Timestamp.Time()
	ed.Value = rec.Value
	if ed.Value > CPUActivationThreshold {
		ed.Status = CPUBusy
	} else {
		ed.Status = CPUIdle
	}
	ed.Repeat = 0
	if ed.LastUtil != nil {
		// Set field "Repeat", CPUIdleDelay is equivalent to GPUIdel
		eqStatus := ed.LastUtil.Status
		if eqStatus == CPUIdleDelay {
			eqStatus = CPUIdle
		}
		if ed.Status == eqStatus {
			ed.Repeat = ed.LastUtil.Repeat + 1
		}
	}
	return ed
}

// archive stores current CPU utilization at LastUtil for retrospection.
func (ed *CPUUtil) archive() *CPUUtil {
	if ed.LastUtil == nil {
		ed.LastUtil = ed.snapshot()
	} else {
		*ed.LastUtil = *ed
		ed.LastUtil.LastUtil = nil // Reset the history of archived to nil to avoid loop back reference.
	}
	return ed.LastUtil
}

// commit sets latest CPU reading.
func (ed *CPUUtil) commit(rec *CPURecord) (committed *CPUUtil) {
	// preserve history required by transition.
	ed.archive()
	committed = ed.init(rec)
	committed.Max = math.Max(committed.Max, committed.Value)
	return committed
}

// reset sets CPU utilization with no actual reading.
func (ed *CPUUtil) reset(time time.Time) *CPUUtil {
	// preserve history required by transition.
	ed.archive()

	// Reset current tick with dummy reading.
	ed.Timestamp = time
	ed.Value = 0.0
	ed.Status = CPUStopped
	ed.Repeat = 0
	// Set field "Repeat", CPUIdleDelay is equivalent to CPUStopped
	if ed.LastUtil != nil && (ed.LastUtil.Status == CPUStopping || ed.LastUtil.Status == CPUStopped) {
		ed.Repeat = ed.LastUtil.Repeat + 1
	}
	return ed
}

func (ed *CPUUtil) DebugCommitAndInit(rec *CPURecord) (committed *CPUUtil) {
	return ed.commit(rec)
}

func (ed *CPUUtil) snapshot() *CPUUtil {
	ss := *ed
	return &ss
}

func (ed *CPUUtil) transit(evtBuff []CPUEvent, force bool) ([]CPUEvent, error) {
	lastStatus := CPUStopped
	if ed.LastUtil != nil {
		lastStatus = ed.LastUtil.Status
	}

	// logger.Debug("Transitioning CPU Status. Last Status: %v.", lastStatus)

	// Support the detection of series transitions
	for {
		if lastStatus == ed.Status {
			return evtBuff, nil
		}

		switch lastStatus {
		case CPUStopped:
			if ed.Status == CPUIdle || ed.Status == CPUBusy {
				lastStatus = CPUIdle
				evtBuff = append(evtBuff, EventCPUStarted)
				continue
			}
			return evtBuff, ErrUnexpectedCPUStTrans
		case CPUIdle:
			if ed.Status == CPUBusy {
				lastStatus = CPUBusy
				evtBuff = append(evtBuff, EventCPUActivated)
				continue
			} else if ed.Status == CPUStopped && (force || ed.Repeat == CPUStopDelay) {
				lastStatus = CPUStopped
				evtBuff = append(evtBuff, EventCPUStopped)
				continue
			} else if ed.Status == CPUStopped {
				ed.Status = CPUStopping
				return evtBuff, nil
			}
			return evtBuff, ErrUnexpectedCPUStTrans
		case CPUIdleDelay:
			if ed.Status == CPUIdle && ed.Repeat < CPUDeactivationDelay {
				ed.Status = CPUIdleDelay
				return evtBuff, nil
			} else if ed.Status == CPUIdle || ed.Status == CPUStopped {
				lastStatus = CPUIdle
				evtBuff = append(evtBuff, EventCPUDeactivated)
				continue
			} else if ed.Status == CPUBusy {
				lastStatus = CPUBusy
				continue
			}
			return evtBuff, ErrUnexpectedCPUStTrans
		case CPUBusy:
			// We defer deactvate event by CPUDeactivationDelay
			if (ed.Status == CPUIdle && ed.Repeat == CPUDeactivationDelay) || ed.Status == CPUStopped {
				lastStatus = CPUIdle
				evtBuff = append(evtBuff, EventCPUDeactivated)
				continue
			} else if ed.Status == CPUIdle {
				ed.Status = CPUIdleDelay
				return evtBuff, nil
			}
			return evtBuff, ErrUnexpectedCPUStTrans
		case CPUStopping:
			// We defer stop event by CPUStopDelay
			if ed.Status == CPUStopped && !force && ed.Repeat < CPUStopDelay {
				ed.Status = CPUStopping
				return evtBuff, nil
			} else if ed.Status == CPUStopped {
				lastStatus = CPUStopped
				evtBuff = append(evtBuff, EventCPUStopped)
				continue
			} else {
				// Reading is available
				lastStatus = CPUIdle
				continue
			}
		}
	}
}

type CPURecord struct {
	Timestamp UnixTime `csv:"timestamp"`
	PodIdx    int      `csv:"pod"`
	Value     float64  `csv:"value"`
	Pod       string
}

func (r *CPURecord) String() string {
	return fmt.Sprintf("CPURecord[Timestamp=%v, PodIdx=%d, Value=%.2f, Pod=%s]", r.Timestamp, r.PodIdx, r.Value, r.Pod)
}

func (r *CPURecord) GetTS() time.Time {
	return r.Timestamp.Time()
}

type CPURecordMapper struct {
	Pod string `csv:"key"`
}

func (r *CPURecordMapper) GetTS() time.Time {
	return time.Time{}
}

type CPUDriver struct {
	*BaseDriver

	MapperPath string
	Downtimes  []int64

	podMap    []string
	podMapper *CPURecordMapper
	pods      []*CPUUtil
	lastRead  int64         // unix timestamp in second
	interval  time.Duration // Tick interval detected during driving.
	down      int           // Indicate trace server was down and no reading during downtime. Odds indicates down.
}

func NewCPUDriver(id int, configs ...func(TraceDriver)) TraceDriver {
	logger.Debug("Creating CPUDriver now.\n")
	drv := &CPUDriver{BaseDriver: NewBaseDriver(id)}
	drv.TraceDriver = drv
	for _, config := range configs {
		config(drv)
	}
	if drv.RecordProvider == nil {
		drv.RecordProvider = &RecordPool{}
	}
	return drv
}

func (d *CPUDriver) SetPodMap(podMap []string) {
	d.podMap = podMap
}

func (d *CPUDriver) String() string {
	return "CPU"
}

func (d *CPUDriver) IsDown() bool {
	return d.down%2 == 1
}

func (d *CPUDriver) Setup(ctx context.Context) error {
	if d.podMapper != nil {
		return nil
	}

	if d.MapperPath == "" {
		d.pods = make([]*CPUUtil, 1000)
		logger.Debug(fmt.Sprintf("%v set up, no mapper loaded", d))
		return nil
	}

	d.podMap = make([]string, 0, 1000)
	d.podMapper = &CPURecordMapper{}
	err := d.DriveSync(context.TODO(), d.MapperPath)
	d.podMapper = nil
	d.pods = make([]*CPUUtil, len(d.podMap))
	logger.Info(fmt.Sprintf("%v set up, mapper loaded, %d entries", d, len(d.podMap)))
	return err
}

func (d *CPUDriver) Teardown(ctx context.Context) {
	if d.podMapper != nil {
		return
	}

	sugarLog.Debugf("%v tearing down, last read %v", d, d.lastRead)
	if d.lastRead != 0 {
		err := d.garbageCollect(ctx, time.Unix(d.lastRead, 0), false)
		if err != nil {
			d.log.Error("Error while performing garbage collection.", zap.Time("timestamp", time.Unix(d.lastRead, int64(d.interval))), zap.Error(err))
			return
		}
		if d.interval == time.Duration(0) {
			d.interval = time.Second
		}
		err = d.garbageCollect(ctx, time.Unix(d.lastRead, int64(d.interval)), true)
		if err != nil {
			d.log.Error("Error while performing garbage collection.", zap.Time("timestamp", time.Unix(d.lastRead, int64(d.interval))), zap.Error(err))
			return
		}
	}
	d.pods = nil
	d.podMap = nil
}

func (d *CPUDriver) GetRecord() Record {
	if d.podMapper != nil {
		return d.podMapper
	}

	r, _ := d.RecordProvider.Get().(*CPURecord)
	if r != nil {
		return r
	}

	return &CPURecord{}
}

func (d *CPUDriver) HandleRecord(ctx context.Context, r Record) {
	if r == d.podMapper {
		d.podMap = append(d.podMap, d.podMapper.Pod)
		return
	}

	defer d.RecordProvider.Recycle(r)

	rec := r.(*CPURecord)
	rec.Timestamp = UnixTime(rec.Timestamp.Time().Add(CPURateOffset))
	if d.podMap != nil {
		rec.Pod = d.podMap[rec.PodIdx]
	} else {
		rec.Pod = strconv.Itoa(rec.PodIdx)
	}
	// rec.Value = rec.Value * 100.0 // Convert utilization to percentage.

	if d.lastRead != 0 && d.lastRead < rec.Timestamp.Time().Unix() {
		ts := time.Unix(d.lastRead, 0)
		interval := rec.Timestamp.Time().Sub(ts)
		down := d.IsDown()
		if d.validateTick(rec.Timestamp.Time(), interval) {
			if down {
				sugarLog.Warnf("Detected CPU trace server resumed since %v, resume garbageCollect...", rec.Timestamp.Time())
			}
			err := d.garbageCollect(ctx, ts, false)
			if err != nil {
				d.log.Error("Error while performing garbage collection.", zap.Time("timestamp", ts), zap.Error(err))
			}
			err = d.FlushEvents(ctx, ts)
			if err != nil {
				d.log.Error("Error while flushing events.", zap.Time("timestamp", ts), zap.Error(err))
			}
		} else if !down {
			sugarLog.Warnf("Detected CPU trace server down since %v, start to skip garbageCollect...", rec.Timestamp.Time())
		}
	}
	d.lastRead = rec.Timestamp.Time().Unix()

	// sugarLog.Debugf("Handling CPU record: %v.", rec)

	cpu, _ := d.ensurePod(rec)

	if d.ExecutionMode == 0 {
		d.MaxesMutex.Lock()
		if _, ok := d.TrainingMaxes[rec.Pod]; !ok {
			podTrainingMaxes := make([]float64, 1)
			podTrainingMaxes[0] = 0
			d.TrainingMaxes[rec.Pod] = podTrainingMaxes
		}
		d.MaxesMutex.Unlock()
	}

	// logger.Trace("Got %v from ensuring pod.", cpu)
	events := make([]CPUEvent, 0, 2) // events buffer
	committed := cpu.commit(rec)

	d.updateSessionMaxCPU(committed)

	// If LastUtil is non-null, then we'll compare against the `MaxTaskCPU` and `MaxSessionCPU`
	// fields of the CPUUtil struct stored in the LastUtil field. Otherwise, we'll just set
	// `MaxTaskCPU` and `MaxSessionCPU` to `committed.Max`.
	if committed.LastUtil != nil {
		if committed.LastUtil.MaxTaskCPU > committed.Max {
			// The `MaxTaskCPU` field of `LastUtil` is larger than `committed.Max`,
			// so set `committed.MaxTaskCPU` to `LastUtil.MaxTaskCPU`.
			committed.MaxTaskCPU = committed.LastUtil.MaxTaskCPU
		} else {
			committed.MaxTaskCPU = committed.Max
		}

		if committed.LastUtil.MaxSessionCPU > committed.Max {
			// The `MaxSessionCPU` field of `LastUtil` is larger than `committed.Max`,
			// so set `committed.MaxSessionCPU` to `LastUtil.MaxSessionCPU`.
			committed.MaxSessionCPU = committed.LastUtil.MaxSessionCPU
		} else {
			committed.MaxSessionCPU = committed.Max
		}
	} else {
		committed.MaxTaskCPU = committed.Max
		committed.MaxSessionCPU = committed.Max
	}

	events, err := committed.transit(events, false)
	// logger.Debug("Transitioned CPU Status. New Status: %v.", committed.Status)
	if err != nil {
		sugarLog.Warnf("Error on handling records: %v", err)
	}
	err = d.triggerMulti(ctx, events, committed)
	if err != nil {
		d.log.Error("Error after multi-triggering events.", zap.Any("events", events), zap.Error(err))
	}

	// logger.Debug("Finished processing CPU record: %v.", rec)
}

// When executing in the "pre-run" mode, we record the maximum CPU utilization for each session.
// This function compares the latest reading from the trace against the maximum CPU utilization
// we've recorded for the associated session and updates the record if the latest reading is greater.
func (d *CPUDriver) updateSessionMaxCPU(committed *CPUUtil) {
	// logger.Info("[%s] Acquiring MaxesMutex lock.", d.DriverType)
	if d.ExecutionMode == 0 {
		// sugarLog.Debugf("Committed CPU util of %0.4f for session %s", committed.Value, committed.Pod)

		d.MaxesMutex.RLock()
		// d.MaxesMutex.Lock()
		// defer d.MaxesMutex.Unlock()
		currentMax, ok := d.SessionMaxes[committed.Pod]

		if !ok {
			d.MaxesMutex.RUnlock()
			d.MaxesMutex.Lock()
			d.SessionMaxes[committed.Pod] = committed.Value

			currentTrainingMaxes, ok2 := d.TrainingMaxes[committed.Pod]

			if !ok2 {
				panic(fmt.Sprintf("Expected to find list of training maxes for session \"%s\".", committed.Pod))
			}

			n := len(currentTrainingMaxes)
			if d.SessionIsCurrentlyTraining[committed.Pod] {
				currentTrainingMaxes[n-1] = committed.Value
				d.TrainingMaxes[committed.Pod] = currentTrainingMaxes
			}

			d.MaxesMutex.Unlock()
			return
		}

		if committed.Value > currentMax {
			d.MaxesMutex.RUnlock()
			d.MaxesMutex.Lock()
			d.SessionMaxes[committed.Pod] = committed.Value
			d.MaxesMutex.Unlock()
			d.MaxesMutex.RLock()
		}

		currentTrainingMaxes, ok2 := d.TrainingMaxes[committed.Pod]

		if !ok2 {
			panic(fmt.Sprintf("Expected to find list of training maxes for session \"%s\".", committed.Pod))
		}

		n := len(currentTrainingMaxes)
		if d.SessionIsCurrentlyTraining[committed.Pod] && committed.Value > currentTrainingMaxes[n-1] {
			d.MaxesMutex.RUnlock()
			d.MaxesMutex.Lock()
			currentTrainingMaxes[n-1] = committed.Value
			d.TrainingMaxes[committed.Pod] = currentTrainingMaxes
			d.MaxesMutex.Unlock()
		} else {
			d.MaxesMutex.RUnlock()
		}
	}
}

func (d *CPUDriver) ensurePod(rec *CPURecord) (util *CPUUtil, created bool) {
	if cap(d.pods) <= rec.PodIdx {
		pods := make([]*CPUUtil, int(math.Ceil(float64(rec.PodIdx+1)/float64(cap(d.pods))))*cap(d.pods))
		copy(pods[:cap(d.pods)], d.pods)
		d.pods = pods
	}
	if d.pods[rec.PodIdx] == nil {
		created = true
		d.pods[rec.PodIdx] = &CPUUtil{
			Pod:           rec.Pod,
			MaxTaskCPU:    0,
			MaxSessionCPU: 0,
		}
	}
	return d.pods[rec.PodIdx], created
}

func (d *CPUDriver) validateTick(ts time.Time, interval time.Duration) bool {
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

func (d *CPUDriver) triggerMulti(ctx context.Context, names []CPUEvent, data *CPUUtil) error {
	if len(names) == 0 {
		return nil
	}
	data = data.snapshot()
	for _, name := range names {
		if err := d.Trigger(ctx, name, data); err != nil {
			return err
		}
	}
	return nil
}

// garbageCollect handles pods have no reading at specified time.
func (d *CPUDriver) garbageCollect(ctx context.Context, ts time.Time, force bool) error {
	events := make([]CPUEvent, 0, 2) // events buffer
	var err error
	for _, pod := range d.pods {
		// Ignore unseen, read at specified time, or stopped
		if pod == nil || pod.Timestamp == ts || pod.Status == CPUStopped {
			continue
		}

		// Reset readings
		committed := pod.reset(ts)
		events, err = committed.transit(events, force)
		if err != nil {
			sugarLog.Warnf("Error on commiting last readings in garbageCollect: %v, %v", err, committed)
		}
		if err := d.triggerMulti(ctx, events, committed); err != nil {
			return err
		}
		events = events[:0]
	}
	return nil
}
