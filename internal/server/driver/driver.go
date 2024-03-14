package driver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
	"go.uber.org/zap"
)

// The Workload Driver consumes events from the Workload Generator and takes action accordingly.
type WorkloadDriver struct {
	id            string
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	workloadEndTime time.Time         // The time at which the workload completed.
	eventChan       chan domain.Event // Receives events from the Synthesizer.

	workloadPresets map[string]*domain.WorkloadPreset
	workloadPreset  *domain.WorkloadPreset
	workloadRequest *domain.WorkloadRequest
	workload        *domain.Workload

	opts     *domain.Configuration
	doneChan chan struct{}
}

func NewWorkloadDriver(opts *domain.Configuration) *WorkloadDriver {
	driver := &WorkloadDriver{
		id:        uuid.NewString(),
		opts:      opts,
		doneChan:  make(chan struct{}),
		eventChan: make(chan domain.Event),
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	driver.logger = logger
	driver.sugaredLogger = logger.Sugar()

	// Load the list of workload presets from the specified file.
	driver.logger.Debug("Loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath))
	presets, err := domain.LoadWorkloadPresetsFromFile(opts.WorkloadPresetsFilepath)
	if err != nil {
		driver.logger.Error("Error encountered while loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath), zap.Error(err))
		panic(err)
	}

	driver.workloadPresets = make(map[string]*domain.WorkloadPreset, len(presets))
	for _, preset := range presets {
		driver.workloadPresets[preset.Key] = preset

		driver.logger.Debug("Discovered preset.", zap.Any(fmt.Sprintf("preset-%s", preset.Key), preset.String()))
	}

	return driver
}

func (d *WorkloadDriver) StartWorkload() {
	go d.DriveWorkload(d.workloadPreset, d.workloadRequest)
}

func (d *WorkloadDriver) GetWorkload() *domain.Workload {
	return d.workload
}

func (d *WorkloadDriver) GetWorkloadPreset() *domain.WorkloadPreset {
	return d.workloadPreset
}

func (d *WorkloadDriver) GetWorkloadRequest() *domain.WorkloadRequest {
	return d.workloadRequest
}

// Returns nil if the workload could not be registered.
func (d *WorkloadDriver) RegisterWorkload(c *gin.Context) *domain.Workload {
	d.logger.Info("WorkloadDriver is handling HTTP request.")

	var workloadRequest *domain.WorkloadRequest
	if err := c.BindJSON(&workloadRequest); err != nil {
		d.logger.Error("Failed to extract and/or unmarshal workload request from request body.")

		c.JSON(http.StatusBadRequest, &domain.ErrorMessage{
			Description:  "Failed to extract workload request from request body.",
			ErrorMessage: err.Error(),
			Valid:        true,
		})
		return nil
	}

	d.workloadRequest = workloadRequest
	d.sugaredLogger.Debugf("User is requesting the execution of workload '%s'", workloadRequest.Key)

	var ok bool
	if d.workloadPreset, ok = d.workloadPresets[workloadRequest.Key]; !ok {
		d.logger.Error("Could not find workload preset with specified key.", zap.String("key", workloadRequest.Key))

		c.JSON(http.StatusBadRequest, &domain.ErrorMessage{
			Description:  "Could not find workload preset with specified key.",
			ErrorMessage: "Could not find workload preset with specified key.",
			Valid:        true,
		})
		return nil
	}

	d.workload = &domain.Workload{
		ID:                 d.id, // Same ID as the driver.
		Name:               workloadRequest.WorkloadName,
		Started:            false,
		Finished:           false,
		WorkloadPreset:     d.workloadPreset,
		WorkloadPresetName: d.workloadPreset.Name,
		WorkloadPresetKey:  d.workloadPreset.Key,
		NumTasksExecuted:   0,
	}
	return d.workload
}

// Write an error back to the client.
func (d *WorkloadDriver) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	c.JSON(http.StatusInternalServerError, msg)
}

// Return true if the workload has completed; otherwise, return false.
func (d *WorkloadDriver) IsWorkloadComplete() bool {
	return d.workload.Finished
}

// This should be called from its own goroutine.
func (d *WorkloadDriver) DriveWorkload(workloadPreset *domain.WorkloadPreset, workloadRequest *domain.WorkloadRequest) {
	d.logger.Debug("Starting workload.", zap.Any("workload-preset", workloadPreset), zap.Any("workload-request", workloadRequest))

	workloadGenerator := generator.NewWorkloadGenerator(d.opts)
	go workloadGenerator.GenerateWorkload(d, workloadPreset, workloadRequest)

	d.workload.StartTime = time.Now()
	d.workload.Started = true

	d.logger.Info("The Workload Driver has started running.")

	for {
		select {
		case evt := <-d.eventChan:
			{
				d.logger.Debug("Received event.", zap.Any("event", evt))
			}
		case <-d.doneChan:
			{
				d.workloadEndTime = time.Now()
				d.workload.Finished = true

				d.logger.Info("The Workload Generator has finished generating events.")
				d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.StartTime)), zap.Any("workload-start-time", d.workload.StartTime), zap.Any("workload-end-time", d.workloadEndTime))

				return
			}
		}
	}
}

// Return the Workload Driver's "done" channel, which is used to signal that the simulation is complete.
func (d *WorkloadDriver) DoneChan() chan struct{} {
	return d.doneChan
}

// Submit an event to the Workload Driver for processing.
func (d *WorkloadDriver) SubmitEvent(evt domain.Event) {
	d.eventChan <- evt
}
