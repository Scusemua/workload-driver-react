package generator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	GPUDeactivationDelay   = 2 // Deactivate after continuous 3 idles
	GPUStopDelay           = 2 // Stop after continuous 3 no reading.
	GPUActivationThreshold = 0 // 2%

	//NoGpu  = "NO_GPU"
	AnyGPU = "ANY_GPU"
)

var (
	ErrUnexpectedGPUStTrans = errors.New("unexpected GPU state transition")
)

type GPUEvent string

func (evt GPUEvent) String() string {
	return string(evt)
}

const (
	EventGPUStarted     GPUEvent = "started"
	EventGPUActivated   GPUEvent = "activated"
	EventGPUDeactivated GPUEvent = "deactivated"
	EventGPUStopped     GPUEvent = "stopped"
	EventGpuUpdateUtil  GPUEvent = "update-util"
)

type GPUStatus int

const (
	GPUStopped GPUStatus = iota
	GPUIdle
	GPUIdleDelay
	GPUBusy
	GPUStopping
)

// GPUUtil is used as a buffer to track overall GPU utilizations with the number of GPU cores used.
// After updated GPU readings of the same timestamp, buffered summary are committed to the "LastUtil".
type GPUUtil struct {
	// prototype keeps track of the original object provisioned by the workload_generator.
	prototype *GPUUtil

	Timestamp time.Time `json:"timestamp"` // Time of the event triggered
	Pod       string    `json:"pod"`
	GPUs      int       `json:"num_gpus"`
	Value     float64   `json:"gpu_utilization"`
	Max       float64   `json:"max_gpu_utilization"`
	Status    GPUStatus `json:"status"`
	GPUName   string    `json:"GPUName"` // TODO(Ben): Right now, we always use 'AnyGPU' for this. Eventually need a way to support a variety of different GPUs. Could do so randomly for now.
	VRamGB    float64   `json:"vram"`

	// Repeat shows how many iterations the status holds.
	Repeat       int       `json:"repeat"`
	RawTimestamp time.Time `json:"rawTimestamp"` // Time traced back to the start of the event (Repeat == 0) for delayed event.

	// LastUtil stores last committed GPU utilization.
	LastUtil *GPUUtil `json:"-"`
}

func (ed *GPUUtil) String() string {
	return fmt.Sprintf("GPUUtil[Pod: %s. GPUs: %d. Util: %.2f%%. VRAM: %.2f GB. Status: %v. Timestamp: %v. RawTS: %v.]",
		ed.Pod, ed.GPUs, ed.Value, ed.VRamGB, ed.Status, ed.Timestamp, ed.RawTimestamp)
}

func (ed *GPUUtil) GetTS() time.Time {
	return ed.Timestamp
}

func (ed *GPUUtil) GetPod() string {
	return ed.Pod
}

func (ed *GPUUtil) Committed() *GPUUtil {
	return ed.prototype.LastUtil
}

func (ed *GPUUtil) DebugInitialize(rec *GPURecord) *GPUUtil {
	return ed.init(rec)
}

// init initiate the GPU utilization with latest GPU reading.
func (ed *GPUUtil) init(rec *GPURecord) *GPUUtil {
	ed.prototype = ed
	ed.Timestamp = rec.Timestamp.Time()
	ed.GPUs = 1
	ed.Value = rec.Value
	ed.VRamGB = rec.VramGb
	if ed.Value > GPUActivationThreshold {
		ed.Status = GPUBusy
	} else {
		ed.Status = GPUIdle
	}
	ed.Repeat = 0
	ed.RawTimestamp = rec.Timestamp.Time()
	return ed
}

func (ed *GPUUtil) DebugUpdate(rec *GPURecord) *GPUUtil {
	return ed.update(rec)
}

func (ed *GPUUtil) update(rec *GPURecord) *GPUUtil {
	// log.Info("Update %s. GPU Index: %s. GPUs: %d -> %d. Utilization: %.2f%% -> %.2f%% (+%.2f%%).", rec.Pod, rec.GPUIdx, ed.GPUs, ed.GPUs+1, ed.Value, ed.Value+rec.Value, rec.Value)

	ed.GPUs++
	ed.Value += rec.Value

	// Just take the largest of the two for now.
	if rec.VramGb > ed.VRamGB {
		ed.VRamGB = rec.VramGb
	}

	// Status will be promoted to GPUBusy only, and keep GPUIdle intact.
	if ed.Value > 0 {
		ed.Status = GPUBusy
	}

	return ed
}

// archive stores current GPU utilization at LastUtil for retrospection.
func (ed *GPUUtil) archive() *GPUUtil {
	if ed.LastUtil == nil {
		ed.LastUtil = ed.snapshot()
	} else {
		*ed.LastUtil = *ed         // Avoid create new object and copy values only.
		ed.LastUtil.LastUtil = nil // Reset the history of archived to nil to avoid loop back reference.
	}
	return ed.LastUtil
}

// commit leverage "archive" to conclude buffered GPU utilization of last sampling tick.
func (ed *GPUUtil) commit() *GPUUtil {
	// By preserving the history first, recovering the history later, and keeping no history in the archive(),
	// we can maintain one level of history only.
	var history *GPUUtil
	if ed.LastUtil != nil {
		// Preserve the history to be recovered later. Reuse archive to avoid creating a new object.
		history = ed.LastUtil.archive()
	}
	committed := ed.archive()    // Leverage "archive" for committing.
	committed.LastUtil = history // Recover the history of the committed utilization.
	if history != nil {
		// Set field "Repeat", GPUIdleDelay is equivalent to GPUIdle
		eqStatus := history.Status
		if eqStatus == GPUIdleDelay {
			eqStatus = GPUIdle
		} else if eqStatus == GPUStopping {
			eqStatus = GPUStopped
		}
		if committed.Status == eqStatus {
			committed.Repeat = history.Repeat + 1
			committed.RawTimestamp = history.RawTimestamp
		}
	}
	return committed
}

// commitAndUpdate concludes buffered GPU utilization of last sampling tick and
// resets the utilization with latest GPU reading.
func (ed *GPUUtil) commitAndInit(rec *GPURecord) (committed *GPUUtil) {
	ed.Max = math.Max(ed.Max, ed.Value)
	committed = ed.commit()
	ed.init(rec)
	return committed
}

// reset concludes GPU utilizations with no actual reading.
func (ed *GPUUtil) reset(time time.Time) *GPUUtil {
	// Reset current tick with dummy reading.
	ed.Timestamp = time
	if ed.LastUtil != nil {
		ed.GPUs = ed.LastUtil.GPUs
	}
	ed.Value = 0
	ed.VRamGB = 0
	ed.Status = GPUStopped
	ed.Repeat = 0
	ed.RawTimestamp = time

	// Commit reset reading.
	if ed.LastUtil == nil || ed.LastUtil.Status != GPUStopped {
		return ed.commit()
	} else {
		return ed.LastUtil
	}
}

func (ed *GPUUtil) snapshot() *GPUUtil {
	ss := *ed
	return &ss
}

func (ed *GPUUtil) DebugCommitAndInit(rec *GPURecord) (committed *GPUUtil) {
	return ed.commitAndInit(rec)
}

func (ed *GPUUtil) transit(evtBuff []GPUEvent, force bool) ([]GPUEvent, error) {
	if commited := ed.Committed(); commited != ed {
		return commited.transit(evtBuff, force)
	}

	lastStatus := GPUStopped
	if ed.LastUtil != nil {
		lastStatus = ed.LastUtil.Status
	}

	// Support the detection of series transitions
	for {
		if lastStatus == ed.Status {
			return evtBuff, nil
		}

		switch lastStatus {
		case GPUStopped:
			if ed.Status == GPUIdle || ed.Status == GPUBusy {
				lastStatus = GPUIdle
				evtBuff = append(evtBuff, EventGPUStarted)
				continue
			}
			return evtBuff, ErrUnexpectedGPUStTrans
		case GPUIdle:
			if ed.Status == GPUBusy {
				lastStatus = GPUBusy
				evtBuff = append(evtBuff, EventGPUActivated)
				continue
			} else if ed.Status == GPUStopped && (force || ed.Repeat == GPUStopDelay) {
				lastStatus = GPUStopped
				evtBuff = append(evtBuff, EventGPUStopped)
				continue
			} else if ed.Status == GPUStopped {
				ed.Status = GPUStopping
				return evtBuff, nil
			}
			return evtBuff, ErrUnexpectedGPUStTrans
		case GPUIdleDelay:
			if ed.Status == GPUIdle && ed.Repeat < GPUDeactivationDelay {
				ed.Status = GPUIdleDelay
				return evtBuff, nil
			} else if ed.Status == GPUIdle || ed.Status == GPUStopped {
				lastStatus = GPUIdle
				evtBuff = append(evtBuff, EventGPUDeactivated)
				continue
			} else if ed.Status == GPUBusy {
				lastStatus = GPUBusy
				continue
			}
			return evtBuff, ErrUnexpectedGPUStTrans
		case GPUBusy:
			// We defer deactivate event by GPUDeactivationDelay
			if (ed.Status == GPUIdle && ed.Repeat == GPUDeactivationDelay) || ed.Status == GPUStopped {
				lastStatus = GPUIdle
				evtBuff = append(evtBuff, EventGPUDeactivated)
				continue
			} else if ed.Status == GPUIdle {
				ed.Status = GPUIdleDelay
				return evtBuff, nil
			}
			return evtBuff, ErrUnexpectedGPUStTrans
		case GPUStopping:
			// We defer stop event by CPUStopDelay
			if ed.Status == GPUStopped && !force && ed.Repeat < GPUStopDelay {
				ed.Status = GPUStopping
				return evtBuff, nil
			} else if ed.Status == GPUStopped {
				lastStatus = GPUStopped
				evtBuff = append(evtBuff, EventGPUStopped)
				continue
			} else {
				// Reading is available
				lastStatus = GPUIdle
				continue
			}
		}
	}
}

type GPURecord struct {
	Timestamp UnixTime `csv:"timestamp" json:"timestamp"`
	PodIdx    int      `csv:"exported_pod" json:"exported_pod"`
	GPUIdx    string   `csv:"gpu" json:"gpu"`
	Value     float64  `csv:"value" json:"value"` // Instance string `csv:"instance"`
	VramGb    float64  `csv:"vram,omitempty" json:"vram"`
	Pod       string   `json:"pod"`
}

func (r *GPURecord) GetTS() time.Time {
	return r.Timestamp.Time()
}

func (r *GPURecord) String() string {
	return fmt.Sprintf("GPURecord[Timestamp=%v, PodIdx=%d, GpuIdx=%s, Value=%.2f, Vram=%.2f, Pod=%s]", r.Timestamp, r.PodIdx, r.GPUIdx, r.Value, r.VramGb, r.Pod)
}

type GPURecordMapper struct {
	Pod string `csv:"key"`
}

func (r *GPURecordMapper) GetTS() time.Time {
	return time.Time{}
}

type GPUDriver struct {
	*BaseDriver

	MapperPath string

	// The maximum utilization achieved by each unique GPU device over the course of the entire simulation.
	PerGpuSessionMaxes map[string][]float64

	// The maximum utilization achieved by each unique GPU device over the course of each individual training event.
	PerGpuTrainingMaxes map[string][][]float64

	podMap    []string
	podMapper *GPURecordMapper
	pods      []*GPUUtil
	lastRead  int64         // unix timestamp in second
	interval  time.Duration // Tick interval detected during driving.
	gcBuff    []*GPUUtil
}

func NewGPUDriver(id int, configs ...func(TraceDriver)) TraceDriver {
	logger.Debug("Creating GPUDriver now.\n")
	drv := &GPUDriver{
		BaseDriver:          NewBaseDriver(id),
		gcBuff:              make([]*GPUUtil, 0, 1000),
		PerGpuSessionMaxes:  make(map[string][]float64),
		PerGpuTrainingMaxes: make(map[string][][]float64),
	}
	drv.TraceDriver = drv
	for _, config := range configs {
		config(drv)
	}
	if drv.RecordProvider == nil {
		drv.RecordProvider = &RecordPool{}
	}
	return drv
}

func (d *GPUDriver) SetPodMap(podMap []string) {
	d.podMap = podMap
}

func (d *GPUDriver) String() string {
	return "GPU"
}

func (d *GPUDriver) Setup(ctx context.Context) error {
	if d.podMapper != nil {
		return nil
	}

	if d.MapperPath == "" {
		d.pods = make([]*GPUUtil, 1000)

		sugarLog.Debugf("%v set up, no mapper loaded", d)
		return nil
	}

	d.podMap = make([]string, 0, 1000)
	d.podMapper = &GPURecordMapper{}
	err := d.DriveSync(context.TODO(), d.MapperPath)
	d.podMapper = nil
	d.pods = make([]*GPUUtil, len(d.podMap))
	sugarLog.Infof("%v set up, mapper loaded, %d entries", d, len(d.podMap))
	return err
}

func (d *GPUDriver) Teardown(ctx context.Context) {
	if d.podMapper != nil {
		return
	}

	d.sugarLog.Debugf("%v tearing down, last read %v", d, d.lastRead)
	if d.lastRead != 0 {
		if err := d.gc(ctx, time.Unix(d.lastRead, 0), false); err != nil {
			d.sugarLog.Warnf("Error while garbage collecting in GPU driver: %v", err)
		}

		if d.interval == time.Duration(0) {
			d.interval = time.Second
		}

		if err := d.gc(ctx, time.Unix(d.lastRead, int64(d.interval)), true); err != nil {
			d.sugarLog.Warnf("Error while garbage collecting in GPU driver: %v", err)
		}
	}
	d.pods = nil
	d.podMap = nil
}

func (d *GPUDriver) GetRecord() Record {
	if d.podMapper != nil {
		return d.podMapper
	}

	r, _ := d.RecordProvider.Get().(*GPURecord)
	if r != nil {
		return r
	}

	return &GPURecord{}
}

func (d *GPUDriver) HandleRecord(ctx context.Context, r Record) {
	if r == d.podMapper {
		d.podMap = append(d.podMap, d.podMapper.Pod)
		return
	}

	defer d.RecordProvider.Recycle(r)

	rec := r.(*GPURecord)
	if d.podMap != nil {
		rec.Pod = d.podMap[rec.PodIdx]
	} else {
		rec.Pod = strconv.Itoa(rec.PodIdx)
	}

	if d.lastRead != 0 && d.lastRead < rec.Timestamp.Time().Unix() {
		ts := time.Unix(d.lastRead, 0)
		d.updateInterval(rec.Timestamp.Time().Sub(ts))
		d.gc(ctx, ts, false)
		d.FlushEvents(ctx, ts)
	}

	d.lastRead = rec.Timestamp.Time().Unix()

	// d.sugarLog.Debugf("GPUDriver is processing record: %v.", rec)

	if d.ExecutionMode == 0 {
		d.updateMaxUtilizationPerGpuDevice(rec)
	}

	gpu, created := d.ensurePod(rec)
	// d.sugarLog.Debugf("Got %v from ensuring pod.", gpu)
	if created {
		gpu.init(rec)
		// d.sugarLog.Debugf("Initializing GPUUtil: %v", gpu)

		if d.ExecutionMode == 0 {
			d.MaxesMutex.RLock()

			// Check if we have a training max recorded for this particular Session.
			if _, ok := d.TrainingMaxes[rec.Pod]; !ok {
				d.MaxesMutex.RUnlock()
				d.MaxesMutex.Lock()
				defer d.MaxesMutex.Unlock()
				podTrainingMaxes := make([]float64, 1)
				podTrainingMaxes[0] = 0
				d.TrainingMaxes[rec.Pod] = podTrainingMaxes

				podTrainingGPUs := make([]int, 1)
				podTrainingGPUs[0] = 0
				d.TrainingNumGPUs[rec.Pod] = podTrainingGPUs
			} else {
				d.MaxesMutex.RUnlock()
			}
		}

		return
	} else if gpu.Timestamp == rec.Timestamp.Time() {
		gpu.update(rec)
		// d.sugarLog.Debugf("Updating GPUUtil: %v", gpu)
		return
	}

	// d.sugarLog.Debugf("Concluding GPUUtil %v", gpu)
	events := make([]GPUEvent, 0, 2) // events buffer
	committed := gpu.commitAndInit(rec)

	if d.ExecutionMode == 0 {
		d.updateMaxUtilization(committed)
	}

	// d.sugarLog.Debugf("Concluded GPUUtil %v", committed)
	events, err := committed.transit(events, false)
	if err != nil {
		sugarLog.Warnf("Error on handling records: %v", err)
	}

	// if len(events) == 0 {
	// 	events = append(events, EventGpuUpdateUtil)
	// }

	// d.sugarLog.Debugf("GPUDriver. Processed record: %v. Committed Status: %v. Triggering %d event(s).", rec, committed.Status, len(events))
	err = d.triggerMulti(ctx, events, committed)
	if err != nil {
		sugarLog.Warnf("Error on triggering events: %v", err)
	}
}

// When executing in the "pre-run" mode, we record the maximum GPU utilization for each session.
// This function compares the latest reading from the trace against the maximum GPU utilization
// we've recorded for the associated session and updates the record if the latest reading is greater.
func (d *GPUDriver) updateMaxUtilization(committed *GPUUtil) {
	// log.Trace("Committed GPU util of %.2f for session %s", committed.Value, committed.Pod)
	// log.Info("[%s] Updating max utilization. Acquiring MaxesMutex lock.", d.DriverType)
	d.MaxesMutex.Lock()
	defer d.MaxesMutex.Unlock()
	currentSessionMax, ok := d.SessionMaxes[committed.Pod]

	if !ok {
		// d.MaxesMutex.RUnlock()
		// d.MaxesMutex.Lock()
		d.SessionMaxes[committed.Pod] = committed.Value
		d.SessionNumGPUs[committed.Pod] = committed.GPUs
		// d.MaxesMutex.Unlock()
		// d.MaxesMutex.RLock()
	} else if committed.Value > currentSessionMax {
		// d.MaxesMutex.RUnlock()
		// d.MaxesMutex.Lock()
		d.SessionMaxes[committed.Pod] = committed.Value
		d.SessionNumGPUs[committed.Pod] = committed.GPUs
		// d.MaxesMutex.Unlock()
		// d.MaxesMutex.RLock()
	}

	currentTrainingMaxes, ok2 := d.TrainingMaxes[committed.Pod]
	currentTrainingGpus, ok3 := d.TrainingNumGPUs[committed.Pod]

	if !ok2 {
		panic(fmt.Sprintf("Expected to find list of training maxes for session \"%s\".", committed.Pod))
	}

	if !ok3 {
		panic(fmt.Sprintf("Expected to find list of number of GPUs for each training event for session \"%s\".", committed.Pod))
	}

	n1 := len(currentTrainingMaxes)
	n2 := len(currentTrainingGpus)

	if n1 != n2 {
		panic(fmt.Sprintf("The number of training maxes (%d) and the number of training NumGPU values (%d) differ for session \"%s\".", n1, n2, committed.Pod))
	}

	n := n1

	// It's set (in the base TraceDriver) to -1 when training ends so that we stop recording.
	// It's set (in the base TraceDriver) to 0 when training begins so that we start recording.
	if d.SessionIsCurrentlyTraining[committed.Pod] && committed.Value > currentTrainingMaxes[n-1] {
		currentTrainingMaxes[n-1] = committed.Value
		currentTrainingGpus[n-1] = committed.GPUs

		d.TrainingMaxes[committed.Pod] = currentTrainingMaxes
		d.TrainingNumGPUs[committed.Pod] = currentTrainingGpus
	}
}

// Update the per-GPU-device max utilizations for the Session and current training event.
func (d *GPUDriver) updateMaxUtilizationPerGpuDevice(rec *GPURecord) {
	gpuIndex, err := strconv.Atoi(rec.GPUIdx)

	if err != nil {
		panic(err)
	}

	// log.Info("Acquiring MaxesMutex for GPUDriver::updateMaxUtilizationPerGpuDevice")
	d.MaxesMutex.Lock()
	defer d.MaxesMutex.Unlock()
	// log.Info("Acquired MaxesMutex for GPUDriver::updateMaxUtilizationPerGpuDevice")

	sessionMaxesPerGpuDevice, ok := d.PerGpuSessionMaxes[rec.Pod]

	if !ok {
		sessionMaxesPerGpuDevice = []float64{0, 0, 0, 0, 0, 0, 0, 0}
		d.PerGpuSessionMaxes[rec.Pod] = sessionMaxesPerGpuDevice
	}

	sessionMax := sessionMaxesPerGpuDevice[gpuIndex]
	if rec.Value > sessionMax {
		sessionMaxesPerGpuDevice[gpuIndex] = rec.Value
		d.PerGpuSessionMaxes[rec.Pod] = sessionMaxesPerGpuDevice
	}

	if d.SessionIsCurrentlyTraining[rec.Pod] {
		trainingMaxesPerGpuDevice, ok := d.PerGpuTrainingMaxes[rec.Pod]

		if !ok {
			trainingMaxesPerGpuDevice = make([][]float64, 0)
			currentTrainingMaxesPerGpuDevice := []float64{0, 0, 0, 0, 0, 0, 0, 0}
			trainingMaxesPerGpuDevice = append(trainingMaxesPerGpuDevice, currentTrainingMaxesPerGpuDevice)
			d.PerGpuTrainingMaxes[rec.Pod] = trainingMaxesPerGpuDevice
		}

		trainingMax := trainingMaxesPerGpuDevice[len(trainingMaxesPerGpuDevice)-1][gpuIndex]
		if rec.Value > trainingMax {
			d.PerGpuTrainingMaxes[rec.Pod][len(trainingMaxesPerGpuDevice)-1][gpuIndex] = rec.Value
		}
	}

	// log.Info("Released MaxesMutex for GPUDriver::updateMaxUtilizationPerGpuDevice")
}

func (d *GPUDriver) ensurePod(rec *GPURecord) (util *GPUUtil, created bool) {
	if cap(d.pods) <= rec.PodIdx {
		pods := make([]*GPUUtil, int(math.Ceil(float64(rec.PodIdx+1)/float64(cap(d.pods))))*cap(d.pods))
		copy(pods[:cap(d.pods)], d.pods)
		d.pods = pods
	}
	if d.pods[rec.PodIdx] == nil {
		created = true
		d.pods[rec.PodIdx] = &GPUUtil{
			Pod:     rec.Pod,
			GPUName: AnyGPU,
		}
	}
	return d.pods[rec.PodIdx], created
}

func (d *GPUDriver) updateInterval(interval time.Duration) {
	if d.interval == time.Duration(0) || d.interval > interval {
		d.interval = interval
	}
}

func (d *GPUDriver) triggerMulti(ctx context.Context, names []GPUEvent, data *GPUUtil) error {
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

// gc handles pods have no reading at specified time.
func (d *GPUDriver) gc(ctx context.Context, ts time.Time, force bool) error {
	d.gcBuff = d.gcBuff[:0]
	events := make([]GPUEvent, 0, 2) // events buffer
	var err error
	for _, pod := range d.pods {
		// if pod != nil && pod.Pod == "trainman-k8s-job-ccdd6414-980b-4a76-8b91-0937740e2b3b-24lx7" {
		// 	log.Debug("check garbageCollect %v %v %v", pod.Timestamp, pod.Status, pod)
		// 	if pod.LastUtil != nil {
		// 		log.Debug("history %v %v %v", pod.Timestamp, pod.Status, pod)
		// 	}
		// }
		// Ignore unseen, read at specified time, or stopped
		if pod == nil || pod.Timestamp == ts || (pod.LastUtil != nil && pod.LastUtil.Status == GPUStopped) {
			continue
		}
		d.gcBuff = append(d.gcBuff, pod)

		// Commit last uncommitted (has data)
		committed := pod.commit()
		events, err = committed.transit(events, force)
		if err != nil {
			sugarLog.Warnf("Error on commiting last readings in garbageCollect: %v, %v", err, committed)
		}
		if err := d.triggerMulti(ctx, events, committed); err != nil {
			return err
		}
		events = events[:0]
	}
	for _, pod := range d.gcBuff {
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
