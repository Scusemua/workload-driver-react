package driver

import (
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

// The Workload Driver consumes events from the Workload Generator and takes action accordingly.
type WorkloadDriver struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	opts     *domain.WorkloadConfig
	doneChan chan struct{}
}

func NewWorkloadDriver(opts *domain.WorkloadConfig) *WorkloadDriver {
	driver := &WorkloadDriver{
		opts:     opts,
		doneChan: make(chan struct{}),
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	driver.logger = logger
	driver.sugaredLogger = logger.Sugar()

	return driver
}

func (d *WorkloadDriver) DriveSimulation() {
	panic("Not implemented.")
}

func (d *WorkloadDriver) DoneChan() chan struct{} {
	return d.doneChan
}

func (d *WorkloadDriver) SubmitEvent(evt domain.Event) {
	panic("Not implemented.")
}
