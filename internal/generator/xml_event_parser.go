package generator

import (
	"container/heap"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var (
	re = regexp.MustCompile(`(\d+)$`)
)

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
	// Length of a single simulation tick.
	tick int64

	// The start time for the event sequence as the number of seconds.
	startingSeconds int64

	cpuRecords []*CPURecord
	gpuRecords []*GPURecord
	memRecords []*Memory

	log      *zap.Logger
	sugarLog *zap.SugaredLogger

	// Map from SessionID to PodID
	podMap map[string]int

	targetFilePath string

	// Optional map that specifies how many GPUs a particular Session should have.
	numGpuMap map[string]int
}

func NewXMLEventParser(tickSeconds int64, startingSeconds int64, targetFilePath string, atom *zap.AtomicLevel) *XMLEventParser {
	return NewXMLEventParserWithMap(tickSeconds, startingSeconds, targetFilePath, make(map[string]int), atom)
}

func NewXMLEventParserWithMap(tickSeconds int64, startingSeconds int64, targetFilePath string, numGpuMap map[string]int, atom *zap.AtomicLevel) *XMLEventParser {
	p := &XMLEventParser{
		tick:            tickSeconds, // time.Second * time.Duration(tickSeconds),
		gpuRecords:      make([]*GPURecord, 0),
		cpuRecords:      make([]*CPURecord, 0),
		memRecords:      make([]*Memory, 0),
		podMap:          make(map[string]int, 0),
		startingSeconds: startingSeconds,
		targetFilePath:  targetFilePath,
		numGpuMap:       numGpuMap,
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	p.log = logger
	p.sugarLog = logger.Sugar()

	return p
}

func (p *XMLEventParser) readXMLFile() MXFile {
	p.sugarLog.Debugf("Parsing XML file for events: \"%s\"", p.targetFilePath)

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

	p.sugarLog.Debugf("mxFile: %v\n", mxFile)

	defer xmlFile.Close()

	return mxFile
}

func extractNumberAtEnd(s string) (int, error) {
	// Find the substring that matches the pattern
	match := re.FindStringSubmatch(s)

	// If there's no match, return an error
	if len(match) == 0 {
		return 0, fmt.Errorf("no number found at the end of the string")
	}

	// Convert the matched string to an integer
	number, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("error converting string to integer: %v", err)
	}

	return number, nil
}

func (p *XMLEventParser) RegisterSession(sessionId string) int {
	idx, err := extractNumberAtEnd(sessionId)
	if err != nil {
		panic(err)
	}

	podIdx := idx - 1
	p.podMap[sessionId] = podIdx

	return podIdx
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
		var (
			objectsForSession []MXObject
			ok                bool
			sessionId         string
		)

		sessionId = object.Session

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
			p.sugarLog.Debug("Processing MXObject for Session %s: %v", sessionId, mxObject)
			duration := mxObject.DurationInTicks()
			startTime := mxObject.StartingTick()
			gpuIndex := mxObject.GPUIndex
			gpu := mxObject.GPU
			cpu := mxObject.CPU
			mem := mxObject.Memory

			// Always make the last three records deactivated (i.e., utilization of 0.)
			// If the event is already idle, then that doesn't change anything.
			// If the event is a training event, then it will end at the proper time, as it
			// takes three consecutive idle readings to deactivate/stop training.

			// Active records, which may also be idle depending on what kind of event this is.
			trainingStopsAt := startTime + duration

			var stopIssuingNonIdleRecordsAt int
			if duration > 1 {
				stopIssuingNonIdleRecordsAt = trainingStopsAt - 2
			} else {
				stopIssuingNonIdleRecordsAt = trainingStopsAt
			}

			for i := startTime; i < stopIssuingNonIdleRecordsAt; i++ {
				var numGPUs int
				var ok bool

				if numGPUs, ok = p.numGpuMap[sessionId]; !ok {
					numGPUs = 1
				} else {
					p.sugarLog.Debug("Found entry in NumGpuMap for session %s.", sessionId)
				}

				for j := 0; j < numGPUs; j++ {
					gpuRecord := &GPURecord{
						Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
						Value:     gpu,
						Pod:       sessionId,
						PodIdx:    podIdx,
						GPUIdx:    fmt.Sprintf("%d", gpuIndex+j),
					}

					gpuRecords = append(gpuRecords, gpuRecord)

					p.sugarLog.Debug("New active rec: %v", gpuRecord)
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
				last_i_value = i
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

				p.sugarLog.Debug("Generated idle GPU record for Session %s: %v", sessionId, gpuRecord)

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
		for i := last_i_value + 1; i < last_i_value+4; i++ {
			gpuRecord := &GPURecord{
				Timestamp: UnixTime(time.Unix(p.startingSeconds+(int64(i)*p.tick), 0)),
				Value:     0,
				Pod:       sessionId,
				PodIdx:    podIdx,
				GPUIdx:    fmt.Sprintf("%d", lastGpuIndex),
			}

			p.sugarLog.Debug("Generated idle [termination] GPU record for Session %s: %v", sessionId, gpuRecord)

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

		p.sugarLog.Debug("There are %d GPU records for Session %s.", len(gpuRecords), sessionId)
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

type recordWrapper struct {
	rec    Record
	sessId string
}

func newRecordWrapper(rec Record, sessId string) *recordWrapper {
	return &recordWrapper{
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
		heap.Push(&recHeap, newRecordWrapper(records[0], sess))

		finalResultSize += len(records)
	}

	for len(result) < finalResultSize {
		nextRecordWrapper := recHeap.Peek()
		record := nextRecordWrapper.rec
		result = append(result, record)
		p.sugarLog.Debugf("Merged record %d/%d into master list: %v", len(result), finalResultSize, record)
		sessionId := nextRecordWrapper.sessId
		nextIdx := indices[sessionId]
		indices[sessionId] = nextIdx + 1

		if nextIdx >= len(arrays[sessionId]) {
			// Done with records for this Session.
			heap.Pop(&recHeap)
			p.sugarLog.Debugf("Exhausted all records for Session %s", sessionId)
			p.sugarLog.Debugf("Records accumulated so far: %d", len(result))
			continue
		}

		recHeap[0] = newRecordWrapper(arrays[sessionId][nextIdx], sessionId)
		heap.Fix(&recHeap, 0)
	}

	return result
}

type recordHeap []*recordWrapper

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
	*h = append(*h, x.(*recordWrapper))
}

func (h *recordHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ret := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return ret
}

func (h recordHeap) Peek() *recordWrapper {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}
