package driver

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
	"go.uber.org/zap"
)

// The Workload Driver consumes events from the Workload Generator and takes action accordingly.
type WorkloadDriver struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	workloadStartTime time.Time         // The time at which the workload began.
	workloadEndTime   time.Time         // The time at which the workload completed.
	workloadComplete  atomic.Bool       // This is set to true when the workload completes.
	eventChan         chan domain.Event // Receives events from the Synthesizer.

	workloadPresets map[string]*domain.WorkloadPreset

	opts     *domain.Configuration
	doneChan chan struct{}
}

func NewWorkloadDriver(opts *domain.Configuration) *WorkloadDriver {
	driver := &WorkloadDriver{
		opts:      opts,
		doneChan:  make(chan struct{}),
		eventChan: make(chan domain.Event),
	}

	driver.workloadComplete.Store(false)

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

func (d *WorkloadDriver) HandleRequest(c *gin.Context) {
	d.logger.Info("WorkloadDriver is handling HTTP request.")

	var workloadRequest *domain.WorkloadRequest
	if err := c.BindJSON(&workloadRequest); err != nil {
		d.logger.Error("Failed to extract and/or unmarshal workload request from request body.")

		c.JSON(http.StatusBadRequest, &domain.ErrorMessage{
			Description:  "Failed to extract workload request from request body.",
			ErrorMessage: err.Error(),
			Valid:        true,
		})
		return
	}

	d.sugaredLogger.Debugf("User is requesting the execution of workload '%s'", workloadRequest.Key)

	if workloadPreset, ok := d.workloadPresets[workloadRequest.Key]; !ok {
		d.logger.Error("Could not find workload preset with specified key.", zap.String("key", workloadRequest.Key))

		c.JSON(http.StatusBadRequest, &domain.ErrorMessage{
			Description:  "Could not find workload preset with specified key.",
			ErrorMessage: "Could not find workload preset with specified key.",
			Valid:        true,
		})
		return
	} else {
		go d.DriveWorkload(workloadPreset, workloadRequest)
	}
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

// Return the request handler responsible for handling a majority of requests.
func (d *WorkloadDriver) PrimaryHttpHandler() domain.BackendHttpGetHandler {
	return d
}

// Return true if the workload has completed; otherwise, return false.
func (d *WorkloadDriver) IsWorkloadComplete() bool {
	return d.workloadComplete.Load()
}

// This should be called from its own goroutine.
func (d *WorkloadDriver) DriveWorkload(workloadPreset *domain.WorkloadPreset, workloadRequest *domain.WorkloadRequest) {
	d.logger.Debug("Starting workload.", zap.Any("workload-preset", workloadPreset), zap.Any("workload-request", workloadRequest))

	workloadGenerator := generator.NewWorkloadGenerator(d.opts)
	go workloadGenerator.GenerateWorkload(d, workloadPreset, workloadRequest)

	d.workloadStartTime = time.Now()
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
				d.workloadComplete.Store(true)

				d.logger.Info("The Workload Generator has finished generating events.")
				d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workloadStartTime)), zap.Any("workload-start-time", d.workloadStartTime), zap.Any("workload-end-time", d.workloadEndTime))

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
