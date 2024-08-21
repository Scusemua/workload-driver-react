package workload

import (
	"sync"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type workloadManagerImpl struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	workloadDrivers    *orderedmap.OrderedMap[string, domain.WorkloadDriver] // Map from workload ID to the associated driver.
	workloadsMap       *orderedmap.OrderedMap[string, domain.Workload]       // Map from workload ID to workload
	workloads          []domain.Workload                                     // Slice of workloads. Same contents as the map, but in slice form.
	pushUpdateInterval time.Duration                                         // The interval at which we push updates to the workloads to the frontend.
	subscribers        map[string]domain.ConcurrentWebSocket                 // Websockets that have submitted a workload and thus will want updates for that workload.
	workloadMutex      sync.Mutex                                            // Synchronizes access to the workload drivers and the workloads themselves (both the map and the slice).
}

func NewWorkloadManager(atom *zap.AtomicLevel) domain.WorkloadManager {
	manager := &workloadManagerImpl{}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	manager.logger = logger
	manager.sugaredLogger = logger.Sugar()

	return manager
}

// Lock all workload drivers.
// This locks each individual driver as well as the top-level workloads mutex.
// It does not release any locks.
func (m *workloadManagerImpl) lockWorkloadDrivers() {
	m.workloadMutex.Lock()

	for el := m.workloadDrivers.Front(); el != nil; el = el.Next() {
		el.Value.LockDriver()
	}
}

// Unlock all workload drivers in the reverse order that they were locked.
//
// If the 'releaseMainLock' parameter is true, then this will also unlock the workloadManagerImpl::workloadMutex.
//
// IMPORTANT: This must be called while the workloadManagerImpl::workloadMutex is held.
func (m *workloadManagerImpl) unlockWorkloadDrivers(releaseMainLock bool) {
	if releaseMainLock {
		defer m.workloadMutex.Unlock()
	}

	for el := m.workloadDrivers.Back(); el != nil; el = el.Prev() {
		el.Value.UnlockDriver()
	}
}

// Return a slice containing all currently-registered workloads (at the time that the method is called).
// The workloads within this slice should not be modified by the caller.
func (m *workloadManagerImpl) GetWorkloads() []domain.Workload {
	m.lockWorkloadDrivers()
	defer m.unlockWorkloadDrivers(true)

	return m.workloads
}
