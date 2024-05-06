package generator

import (
	"container/heap"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CustomEventSequencer struct {
	// Length of a single simulation tick.
	tick int64

	// The start time for the event sequence as the number of seconds.
	startingSeconds int64

	counter int64

	cpuRecords []*CPURecord
	gpuRecords []*GPURecord
	memRecords []*Memory

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	// Map from PodIDX to PodID
	podMap     map[string]int
	nextPodIdx int
}

func NewCustomEventSequencer(tickSeconds int64, startingSeconds int64, atom *zap.AtomicLevel) *CustomEventSequencer {
	customEventSequencer := &CustomEventSequencer{
		tick:            tickSeconds, // time.Second * time.Duration(tickSeconds),
		gpuRecords:      make([]*GPURecord, 0),
		cpuRecords:      make([]*CPURecord, 0),
		memRecords:      make([]*Memory, 0),
		podMap:          make(map[string]int, 0),
		startingSeconds: startingSeconds,
		// registeredSessions: make(map[string]struct{}, 0),
		nextPodIdx: 0,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	customEventSequencer.logger = logger
	customEventSequencer.sugaredLogger = logger.Sugar()

	return customEventSequencer
}

// This function should be called once you are done adding records.
// This will sort all of the records by their timestamps so that they are in-order for the drivers to process them.
func (c *CustomEventSequencer) SortRecords() {
	sort.Slice(c.cpuRecords, func(i, j int) bool {
		return c.cpuRecords[i].Timestamp.Time().Before(c.cpuRecords[j].Timestamp.Time())
	})

	sort.Slice(c.gpuRecords, func(i, j int) bool {
		return c.cpuRecords[i].Timestamp.Time().Before(c.cpuRecords[j].Timestamp.Time())
	})

	sort.Slice(c.memRecords, func(i, j int) bool {
		return c.cpuRecords[i].Timestamp.Time().Before(c.cpuRecords[j].Timestamp.Time())
	})
}

func (c *CustomEventSequencer) RegisterSession(sessionId string) int {
	podIdx := c.nextPodIdx
	c.podMap[sessionId] = podIdx
	c.nextPodIdx += 1

	return podIdx
}

// Create and add a GPU record to the internal sequence of records.
// Returns the created GPU record.
func (c *CustomEventSequencer) addGpuRecord(gpuValue float64, sessionId string, gpuIdx string, ts time.Time) *GPURecord {
	var podIdx int
	var ok bool

	if podIdx, ok = c.podMap[sessionId]; !ok {
		podIdx = c.RegisterSession(sessionId)
	}

	gpuRecord := &GPURecord{
		Timestamp: UnixTime(ts),
		Value:     gpuValue,
		Pod:       sessionId,
		PodIdx:    podIdx,
		GPUIdx:    gpuIdx,
	}

	c.gpuRecords = append(c.gpuRecords, gpuRecord)

	return gpuRecord
}

// Create and add a CPU record to the internal sequence of records.
// Returns the created CPU record.
func (c *CustomEventSequencer) addCpuRecord(cpuValue float64, sessionId string, ts time.Time) *CPURecord {
	var podIdx int
	var ok bool

	if podIdx, ok = c.podMap[sessionId]; !ok {
		podIdx = c.RegisterSession(sessionId)
	}

	cpuRecord := &CPURecord{
		// CPU record timestamps are adjusted by 120 (i.e., ts - 120), so we add it to the timestamps of the records that we create for the tests.
		Timestamp: UnixTime(ts.Add(time.Second * time.Duration(120))),
		Value:     cpuValue,
		Pod:       sessionId,
		PodIdx:    podIdx,
	}

	c.cpuRecords = append(c.cpuRecords, cpuRecord)

	return cpuRecord
}

// Add records to the record sequence that will generate a training event.
// The training duration must be at least 3. If it's not, then 3 will be used.
func (c *CustomEventSequencer) AddTrainingEvent(trainingDurationSec int, targetSessionId string, cpuUsage float64) {
	// Training ends after three consecutive empty readings.
	if trainingDurationSec < 3 {
		c.sugaredLogger.Warnf("Specified invalid training duration of %d second(s) for Session %v. Using default, minimum value of 3 seconds instead.", trainingDurationSec, targetSessionId)
		trainingDurationSec = 3
	}
}

// Add records to the record sequence that will generate a training event.
func (c *CustomEventSequencer) AddIdleInterval(idleIntervalDurationSec int64, targetSessionId string) {

}

// Add records to create a new Session.
func (c *CustomEventSequencer) AddSession(sessionId string) {
	for i := 0; i < 3; i++ {
		c.addGpuRecord(0, sessionId, "0", time.Unix(c.startingSeconds+(c.counter*c.tick), 0))

		c.addCpuRecord(0, sessionId, time.Unix(c.startingSeconds+(c.counter*c.tick), 0))

		c.counter += 1
	}
}

type MXFile struct {
	XMLName xml.Name   `xml:"mxfile"`
	Diagram XMLDiagram `xml:"diagram"`
}

func (o MXFile) String() string {
	return fmt.Sprintf("MXFile[Diagram=%v]", o.Diagram)
}

type XMLDiagram struct {
	XMLName    xml.Name     `xml:"diagram"`
	GraphModel MXGraphModel `xml:"mxGraphModel"`
}

func (o XMLDiagram) String() string {
	return fmt.Sprintf("XMLDiagram[GraphModel=%v]", o.GraphModel)
}

type MXGraphModel struct {
	XMLName xml.Name `xml:"mxGraphModel"`
	Root    XMLRoot  `xml:"root"`
}

func (o MXGraphModel) String() string {
	return fmt.Sprintf("MXGraphModel[Root=%v]", o.Root)
}

type XMLRoot struct {
	XMLName xml.Name   `xml:"root"`
	MXCells []MXCell   `xml:"mxCell"`
	Objects []MXObject `xml:"object"`
}

func (o XMLRoot) String() string {
	// return fmt.Sprintf("XMLRoot[Objects(%d)=%v]", len(o.Objects), o.Objects)
	return fmt.Sprintf("XMLRoot[MXCells=%v, Objects=%v]", o.MXCells, o.Objects)
}

type MXCell struct {
	XMLName  xml.Name   `xml:"mxCell"`
	Geometry MXGeometry `xml:"mxGeometry"`
}

func (o MXCell) String() string {
	return fmt.Sprintf("MXCell[Geometry=%v]", o.Geometry)
}

type MXObject struct {
	XMLName  xml.Name `xml:"object"`
	Cell     MXCell   `xml:"mxCell"`
	Label    string   `xml:"label,attr"`
	Session  string   `xml:"Session,attr"`
	CPU      float64  `xml:"CPU,attr"`
	GPU      float64  `xml:"GPU,attr"`
	Memory   float64  `xml:"Memory,attr"`
	GPUIndex int      `xml:"GPUIndex,attr"`
	ID       string   `xml:"id,attr"`
}

func (o MXObject) DurationInTicks() int {
	return o.Cell.Geometry.Width
}

func (o MXObject) StartingTick() int {
	return o.Cell.Geometry.X
}

func (o MXObject) String() string {
	return fmt.Sprintf("MXObject[Session=%s, CPU=%.2f, GPU=%.2f, Memory=%.2f, GPUIndex=%d, StartingTick=%d ticks, Duration=%d ticks]", o.Session, o.CPU, o.GPU, o.Memory, o.GPUIndex, o.StartingTick(), o.DurationInTicks())
}

type MXGeometry struct {
	XMLName xml.Name `xml:"mxGeometry"`
	X       int      `xml:"x,attr"`
	Y       int      `xml:"y,attr"`
	Height  int      `xml:"height,attr"`
	Width   int      `xml:"width,attr"`
}

func (o MXGeometry) String() string {
	return fmt.Sprintf("MXGeometry[X=%d, Y=%d, Height=%d, Width=%d]", o.X, o.Y, o.Height, o.Width)
}

type XMLEventParser struct {
	*CustomEventSequencer

	targetFilePath string

	// Optional map that specifies how many GPUs a particular Session should have.
	numGpuMap map[string]int
}

func NewXMLEventParser(tickSeconds int64, startingSeconds int64, targetFilePath string, atom *zap.AtomicLevel) *XMLEventParser {
	return NewXMLEventParserWithMap(tickSeconds, startingSeconds, targetFilePath, make(map[string]int), atom)
}

func NewXMLEventParserWithMap(tickSeconds int64, startingSeconds int64, targetFilePath string, numGpuMap map[string]int, atom *zap.AtomicLevel) *XMLEventParser {
	customEventSequencer := NewCustomEventSequencer(tickSeconds, startingSeconds+tickSeconds, atom) // Start in the next tick, not the current one, so add `tickSeconds`.

	p := &XMLEventParser{
		customEventSequencer,
		targetFilePath,
		numGpuMap,
	}

	return p
}

func (p *XMLEventParser) readXMLFile() MXFile {
	p.sugaredLogger.Debugf("Parsing XML file for events: \"%s\"", p.targetFilePath)

	xmlFile, err := os.Open(p.targetFilePath)

	if err != nil {
		panic(err)
	}

	byteValue, _ := io.ReadAll(xmlFile)

	var mxFile MXFile
	err = xml.Unmarshal(byteValue, &mxFile)

	if err != nil {
		panic(err)
	}

	defer xmlFile.Close()

	return mxFile
}

func (p *XMLEventParser) Parse() ([]Record, []Record, []Record) {
	mxFile := p.readXMLFile()
	root := mxFile.Diagram.GraphModel.Root
	mxObjects := root.Objects

	// Steps:
	// 1) Separate objects by their Session.
	// 2) Sort objects by their x position.
	// 3) Begin processing the objects, creating records for each session separately.
	// 4) Merge (like in merge sort) all of the events from all of the different sessions.

	// Map from SessionID to slice of MXObject structs.
	objMap := make(map[string][]MXObject)

	// Step 1
	for _, object := range mxObjects {
		var objectsForSession []MXObject
		var ok bool

		sessionId := object.Session

		if objectsForSession, ok = objMap[sessionId]; !ok {
			objectsForSession = make([]MXObject, 0, 1)
		}

		objectsForSession = append(objectsForSession, object)
		objMap[sessionId] = objectsForSession
	}

	// Step 2
	for _, objects := range objMap {
		sort.SliceStable(objects, func(i, j int) bool {
			return objects[i].StartingTick() < objects[j].StartingTick()
		})
	}

	// Step 3
	perSessionCpuRecords := make(map[string][]Record)
	perSessionGpuRecords := make(map[string][]Record)
	perSessionMemRecords := make(map[string][]Record)

	for _, objects := range objMap {
		var cpuRecords []Record
		var gpuRecords []Record
		var memRecords []Record
		var ok bool

		sessionId := objects[0].Session

		if cpuRecords, ok = perSessionCpuRecords[objects[0].Session]; !ok {
			cpuRecords = make([]Record, 0)
		}
		if gpuRecords, ok = perSessionGpuRecords[objects[0].Session]; !ok {
			gpuRecords = make([]Record, 0)
		}
		if memRecords, ok = perSessionMemRecords[objects[0].Session]; !ok {
			memRecords = make([]Record, 0)
		}

		var podIdx int
		if podIdx, ok = p.podMap[sessionId]; !ok {
			podIdx = p.RegisterSession(sessionId)
		}

		var lastGpuIndex int
		var last_i_value int

		for _, mxObject := range objects {
			p.sugaredLogger.Debugf("Processing MXObject for Session %s: %v", sessionId, mxObject)
			duration := mxObject.DurationInTicks()
			startTime := mxObject.StartingTick()
			gpuIndex := mxObject.GPUIndex
			gpu := mxObject.GPU
			cpu := mxObject.CPU
			mem := mxObject.Memory

			// Always make the last three records deactivated (i.e., utilization of 0.)
			// If the event is already idle, then that doesn't change anything.
			// If the event is a training event, then it will end at the proper time, as it
			// Takes three consecutive idle readings to deactivate/stop training.

			// Active records, which may also be idle depending on what kind of event this is.
			trainingStopsAt := startTime + duration
			stopIssuingNonIdleRecordsAt := trainingStopsAt - 2
			for i := startTime; i < stopIssuingNonIdleRecordsAt; i++ {
				var numGPUs int
				var ok bool

				if numGPUs, ok = p.numGpuMap[sessionId]; !ok {
					numGPUs = 1
				} else {
					p.sugaredLogger.Debugf("Found entry in NumGpuMap for session %s.", sessionId)
				}

				// p.sugaredLogger.Debugf("Session %s is supposed to have %d GPUs.", sessionId, numGPUs)

				for j := 0; j < numGPUs; j++ {
					gpuRecord := &GPURecord{
						Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
						Value:     gpu,
						Pod:       sessionId,
						PodIdx:    podIdx,
						GPUIdx:    fmt.Sprintf("%d", gpuIndex+j),
					}

					gpuRecords = append(gpuRecords, gpuRecord)
				}

				cpuRecord := &CPURecord{
					// CPU record timestamps are adjusted by 120 (i.e., ts - 120), so we add it to the timestamps of the records that we create for the tests.
					Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0).Add(time.Second * time.Duration(120))),
					Value:     cpu,
					Pod:       sessionId,
					PodIdx:    podIdx,
				}

				memRecord := &Memory{
					// CPU record timestamps are adjusted by 120 (i.e., ts - 120), so we add it to the timestamps of the records that we create for the tests.
					Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
					Value:     mem,
					Pod:       sessionId,
					PodIdx:    podIdx,
				}

				cpuRecords = append(cpuRecords, cpuRecord)
				memRecords = append(memRecords, memRecord)
			}

			// Idle records.
			for i := stopIssuingNonIdleRecordsAt; i < trainingStopsAt; i++ {
				gpuRecord := &GPURecord{
					Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
					Value:     0,
					Pod:       sessionId,
					PodIdx:    podIdx,
					GPUIdx:    fmt.Sprintf("%d", gpuIndex),
				}

				cpuRecord := &CPURecord{
					// CPU record timestamps are adjusted by 120 (i.e., ts - 120), so we add it to the timestamps of the records that we create for the tests.
					Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0).Add(time.Second * time.Duration(120))),
					Value:     0,
					Pod:       sessionId,
					PodIdx:    podIdx,
				}

				memRecord := &Memory{
					// CPU record timestamps are adjusted by 120 (i.e., ts - 120), so we add it to the timestamps of the records that we create for the tests.
					Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
					Value:     0,
					Pod:       sessionId,
					PodIdx:    podIdx,
				}

				cpuRecords = append(cpuRecords, cpuRecord)
				gpuRecords = append(gpuRecords, gpuRecord)
				memRecords = append(memRecords, memRecord)
				last_i_value = i
			}

			lastGpuIndex = gpuIndex
		}

		// Make sure that the Session terminates if the last event was an active/training event.
		for i := last_i_value + 1; i < last_i_value+3; i++ {
			gpuRecord := &GPURecord{
				Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
				Value:     0,
				Pod:       sessionId,
				PodIdx:    podIdx,
				GPUIdx:    fmt.Sprintf("%d", lastGpuIndex),
			}

			cpuRecord := &CPURecord{
				// CPU record timestamps are adjusted by 120 (i.e., ts - 120), so we add it to the timestamps of the records that we create for the tests.
				Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0).Add(time.Second * time.Duration(120))),
				Value:     0,
				Pod:       sessionId,
				PodIdx:    podIdx,
			}

			memRecord := &Memory{
				// CPU record timestamps are adjusted by 120 (i.e., ts - 120), so we add it to the timestamps of the records that we create for the tests.
				Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
				Value:     0,
				Pod:       sessionId,
				PodIdx:    podIdx,
			}

			cpuRecords = append(cpuRecords, cpuRecord)
			gpuRecords = append(gpuRecords, gpuRecord)
			memRecords = append(memRecords, memRecord)
		}

		perSessionCpuRecords[sessionId] = cpuRecords
		perSessionGpuRecords[sessionId] = gpuRecords
		perSessionMemRecords[sessionId] = memRecords
	}

	for _, objects := range perSessionCpuRecords {
		sort.SliceStable(objects, func(i, j int) bool {
			return objects[i].GetTS().Before(objects[j].GetTS())
		})
	}

	for _, objects := range perSessionGpuRecords {
		sort.SliceStable(objects, func(i, j int) bool {
			return objects[i].GetTS().Before(objects[j].GetTS())
		})
	}

	for _, objects := range perSessionMemRecords {
		sort.SliceStable(objects, func(i, j int) bool {
			return objects[i].GetTS().Before(objects[j].GetTS())
		})
	}

	// Merge all of the sorted arrays into one (for each type of records).
	gpuRecords := p.MergeArrays(perSessionGpuRecords)
	cpuRecords := p.MergeArrays(perSessionCpuRecords)
	memRecords := p.MergeArrays(perSessionMemRecords)

	return gpuRecords, cpuRecords, memRecords
}

type RecordWrapper struct {
	rec    Record
	sessId string
}

func NewRecordWrapper(rec Record, sessId string) *RecordWrapper {
	return &RecordWrapper{
		rec:    rec,
		sessId: sessId,
	}
}

func (p *XMLEventParser) MergeArrays(arrays map[string][]Record) []Record {
	result := make([]Record, 0)
	indices := make(map[string]int)
	recHeap := make(recordHeap, 0)
	finalResultSize := 0

	for sess, records := range arrays {
		indices[sess] = 1
		heap.Push(&recHeap, NewRecordWrapper(records[0], sess))

		finalResultSize += len(records)
	}

	for len(result) < finalResultSize {
		nextRecordWrapper := recHeap.Peek()
		record := nextRecordWrapper.rec
		result = append(result, record)
		// p.sugaredLogger.Trace("Merged record %d/%d into master list: %v", len(result), finalResultSize, record)
		sessionId := nextRecordWrapper.sessId
		nextIdx := indices[sessionId]
		indices[sessionId] = nextIdx + 1

		if nextIdx >= len(arrays[sessionId]) {
			// Done with records for this Session.
			heap.Pop(&recHeap)
			// p.sugaredLogger.Trace("Exhausted all records for Session %s", sessionId)
			// p.sugaredLogger.Trace("Records accumulated so far: %d", len(result))
			continue
		}

		recHeap[0] = NewRecordWrapper(arrays[sessionId][nextIdx], sessionId)
		heap.Fix(&recHeap, 0)
	}

	return result
}

type recordHeap []*RecordWrapper

func (h recordHeap) Len() int {
	return len(h)
}

func (h recordHeap) Less(i, j int) bool {
	return h[i].rec.GetTS().Before(h[j].rec.GetTS())
}

func (h recordHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *recordHeap) Push(x interface{}) {
	*h = append(*h, x.(*RecordWrapper))
}

func (h *recordHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h recordHeap) Peek() *RecordWrapper {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
