package driver

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
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

// Return true if the workload has completed; otherwise, return false.
func (d *WorkloadDriver) IsWorkloadComplete() bool {
	return d.workloadComplete.Load()
}

// This should be called from its own goroutine.
func (d *WorkloadDriver) DriveWorkload() {
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
