package workload

import (
	"fmt"
	"sync"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type workloadManagerImpl struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	workloadDrivers *orderedmap.OrderedMap[string, domain.WorkloadDriver] // Map from workload ID to the associated driver.
	workloadsMap    *orderedmap.OrderedMap[string, domain.Workload]       // Map from workload ID to workload
	workloads       []domain.Workload                                     // Slice of workloads. Same contents as the map, but in slice form.
	workloadMutex   sync.Mutex                                            // Synchronizes access to the workload drivers and the workloads themselves (both the map and the slice).
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

// Return the workload driver associated with the given workload ID.
// If there is no driver associated with the provided workload ID, then nil is returned.
func (m *workloadManagerImpl) GetWorkloadDriver(workloadId string) domain.WorkloadDriver {
	m.workloadMutex.Lock()
	defer m.workloadMutex.Unlock()

	return m.workloadDrivers.GetOrDefault(workloadId, nil)
}

// Toggle debug logging on or off (depending on the value of the 'enabled' parameter) for the specified workload.
// If there is no workload with the specified ID, then an error is returned.
//
// If successful, then this returns the updated workload.
func (m *workloadManagerImpl) ToggleDebugLogging(workloadId string, enabled bool) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	workloadDriver.LockDriver()
	defer workloadDriver.UnlockDriver()

	updatedWorkload := workloadDriver.ToggleDebugLogging(enabled)
	return updatedWorkload, nil
}

// Start the workload with the specified ID.
// The workload must have already been registered.
//
// If successful, then this returns the updated workload.
// If there is no workload with the specified ID, then an error is returned.
// Likewise, if the specified workload is either already-running or has already been stopped, then an error is returned.
func (m *workloadManagerImpl) StartWorkload(workloadId string) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	workloadDriver.LockDriver()
	defer workloadDriver.UnlockDriver()

	// Start the workload.
	// This sets the "start time" and transitions the workload to the "running" state.
	err := workloadDriver.StartWorkload()
	if err != nil {
		return nil, err
	}

	go workloadDriver.ProcessWorkload(nil) // &wg
	go workloadDriver.DriveWorkload(nil)   // &wg

	workload := workloadDriver.GetWorkload()
	workload.UpdateTimeElapsed()

	m.logger.Debug("Started workload.", zap.String("workload-id", workloadId), zap.Any("workload-source", workload.GetWorkloadSource()))

	return workload, nil
}

// Stop the workload with the specified ID.
// The workload must have already been registered and should be actively-running.
//
// If successful, then this returns the updated workload.
// If there is no workload with the specified ID, or the specified workload is not actively-running, then an error is returned.
func (m *workloadManagerImpl) StopWorkload(workloadId string) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	workloadDriver.LockDriver()
	defer workloadDriver.UnlockDriver()

	// Start the workload.
	// This sets the "start time" and transitions the workload to the "running" state.
	err := workloadDriver.StopWorkload()
	if err != nil {
		return nil, err
	}

	return workloadDriver.GetWorkload(), nil
}
