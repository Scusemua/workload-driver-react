package driver

import (
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

// The Workload Driver consumes events from the Workload Generator and takes action accordingly.
type WorkloadDriver struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
}

func NewWorkloadDriver() *WorkloadDriver {
	driver := &WorkloadDriver{}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	driver.logger = logger
	driver.sugaredLogger = logger.Sugar()

	return driver
}

func (d *WorkloadDriver) SubmitEvent(evt domain.Event) {

}
