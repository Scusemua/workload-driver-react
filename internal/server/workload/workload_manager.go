package workload

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type workloadManagerImpl struct {
	atom          *zap.AtomicLevel
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	configuration            *domain.Configuration                                 // Server-wide configuration.
	pushGoroutineActive      atomic.Int32                                          // Indicates whether there is already a goroutine serving the "push" routine, which pushes updated workload data to the frontend.
	pushUpdateInterval       time.Duration                                         // The interval at which we push updates to the workloads to the frontend.
	workloadWebsocketHandler *WorkloadWebsocketHandler                             // Workload WebSocket handler. Accepts and processes WebSocket requests related to workloads.
	workloadDrivers          *orderedmap.OrderedMap[string, domain.WorkloadDriver] // Map from workload ID to the associated driver.
	workloadsMap             *orderedmap.OrderedMap[string, domain.Workload]       // Map from workload ID to workload
	workloads                []domain.Workload                                     // Slice of workloads. Same contents as the map, but in slice form.
	mu                       sync.Mutex                                            // Synchronizes access to the workload drivers and the workloads themselves (both the map and the slice).
	workloadStartedChan      chan string                                           // Channel of workload IDs. When a workload is started, its ID is submitted to this channel.
}

func NewWorkloadManager(configuration *domain.Configuration, atom *zap.AtomicLevel) domain.WorkloadManager {
	manager := &workloadManagerImpl{
		atom:                atom,
		configuration:       configuration,
		workloadDrivers:     orderedmap.NewOrderedMap[string, domain.WorkloadDriver](),
		workloadsMap:        orderedmap.NewOrderedMap[string, domain.Workload](),
		workloads:           make([]domain.Workload, 0),
		workloadStartedChan: make(chan string, 4),
		pushUpdateInterval:  time.Second * time.Duration(configuration.PushUpdateInterval),
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	manager.logger = logger
	manager.sugaredLogger = logger.Sugar()

	manager.workloadWebsocketHandler = NewWorkloadWebsocketHandler(configuration, manager, manager.workloadStartedChan, atom)
	manager.pushGoroutineActive.Store(0)

	return manager
}

// Return a function that can handle WebSocket requests for workload operations.
//
// This simply returns the handler function of the WorkloadWebsocketHandler struct of the WorkloadManager.
func (m *workloadManagerImpl) GetWorkloadWebsocketHandler() gin.HandlerFunc {
	if m.pushGoroutineActive.CompareAndSwap(0, 1) {
		go m.serverPushRoutine()
	}

	return m.workloadWebsocketHandler.serveWorkloadWebsocket
}

// Lock all workload drivers.
// This locks each individual driver as well as the top-level workloads mutex.
// It does not release any locks.
func (m *workloadManagerImpl) lockWorkloadDrivers() {
	m.mu.Lock()

	for el := m.workloadDrivers.Front(); el != nil; el = el.Next() {
		el.Value.LockDriver()
	}
}

// Unlock all workload drivers in the reverse order that they were locked.
//
// If the 'releaseMainLock' parameter is true, then this will also unlock the workloadManagerImpl::mu.
//
// IMPORTANT: This must be called while the workloadManagerImpl::mu is held.
func (m *workloadManagerImpl) unlockWorkloadDrivers(releaseMainLock bool) {
	if releaseMainLock {
		defer m.mu.Unlock()
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

// Return a map from Workload ID to Workload struct containing workloads that are active when the method is called.
func (m *workloadManagerImpl) GetActiveWorkloads() map[string]domain.Workload {
	m.lockWorkloadDrivers()
	defer m.unlockWorkloadDrivers(true)

	activeWorkloads := make(map[string]domain.Workload)

	for _, workload := range m.workloads {
		if workload.IsRunning() {
			activeWorkloads[workload.GetId()] = workload
		}
	}

	return activeWorkloads
}

// Return the workload driver associated with the given workload ID.
// If there is no driver associated with the provided workload ID, then nil is returned.
func (m *workloadManagerImpl) GetWorkloadDriver(workloadId string) domain.WorkloadDriver {
	m.mu.Lock()
	defer m.mu.Unlock()

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
		m.sugaredLogger.Errorf("Cannot start workload \"%s\" as it has not yet been reigstered with the Workload Manager.", workloadId)
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
		m.logger.Error("Could not find workload driver with specified workload ID.", zap.String("workload-id", workloadId))
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	workloadDriver.LockDriver()
	defer workloadDriver.UnlockDriver()

	// Stop the workload.
	err := workloadDriver.StopWorkload()
	if err != nil {
		return nil, err
	}

	return workloadDriver.GetWorkload(), nil
}

// Register a new workload.
func (m *workloadManagerImpl) RegisterWorkload(request *domain.WorkloadRegistrationRequest, ws domain.ConcurrentWebSocket) (domain.Workload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new workload driver.
	workloadDriver := NewWorkloadDriver(m.configuration, true, request.TimescaleAdjustmentFactor, ws, m.atom)

	// Register a new workload with the workload driver.
	workload, err := workloadDriver.RegisterWorkload(request)
	if err != nil {
		m.logger.Error("Failed to create and register new workload.", zap.Any("workload-registration-request", request), zap.Error(err))
		return nil, err
	}

	// Update our internal state and perform the necessary book-keeping.
	workloadId := workload.GetId()
	m.workloads = append(m.workloads, workload)
	m.workloadsMap.Set(workloadId, workload)
	m.workloadDrivers.Set(workloadId, workloadDriver)

	m.sugaredLogger.Debugf("Successfully registered workload \"%s\" with Workload Manager.", workloadId)

	return workload, err
}

// Push an update to the frontend.
func (m *workloadManagerImpl) pushWorkloadUpdate(payload []byte) error {
	errs := m.workloadWebsocketHandler.broadcastToWorkloadWebsockets(payload)

	if len(errs) >= 1 {
		return errors.Join(errs...)
	}

	return nil
}

// Used to push updates about active workloads to the frontend.
func (m *workloadManagerImpl) serverPushRoutine( /* doneChan chan struct{} */ ) {
	activeWorkloads := m.GetActiveWorkloads()

	// Function that continuously pulls workload IDs out of the 'workloadStartedChan' until there are none left.
	// This returns the number of new workloads detected.
	checkForNewActiveWorkloads := func() int {
		var numNewActiveWorkloads int = 0

		for { // Keep pulling workload IDs out of the 'workloadStartedChan' until there are none left.
			select {
			case id := <-m.workloadStartedChan:
				{
					m.mu.Lock()

					// Add the newly-registered workload to the active workloads map.
					var ok bool
					activeWorkloads[id], ok = m.workloadsMap.Get(id)
					if !ok {
						panic(fmt.Sprintf("Failed to find supposedly-active workload \"%s\"", id))
					}

					m.mu.Unlock()

					numNewActiveWorkloads += 1
				}
			// case <-doneChan:
			// 	{
			// 		return
			// 	}
			default:
				// There are no more IDs in the 'workload started' channel, so we can return.
				return numNewActiveWorkloads
			}
		}
	}

	// We'll loop until the underlying WebSocket connection is terminated.
	for {
		// Check for any newly-registered workloads before pushing an update.
		checkForNewActiveWorkloads()

		// If we have any active workloads, then we'll push some updates to the front-end for the active workloads.
		if len(activeWorkloads) > 0 {
			// Keep track of any workloads that are no longer active.
			// We'll push one more update for these workloads and then stop pushing updates for them,
			// as the state/data of non-active workoads does not change.
			noLongerActivelyRunning := make([]string, 0)

			m.mu.Lock()
			// Iterate over all the active workloads.
			for _, workload := range activeWorkloads {
				// If the workload is no longer active, then make a note to remove it after this next update.
				// (We need to include it in the update so the frontend knows it's no longer active.)
				if !workload.IsRunning() {
					// This workload is no longer active. We'll push it to the frontend one last time,
					// and then we'll stop pushing updates for this workload.
					noLongerActivelyRunning = append(noLongerActivelyRunning, workload.GetId())
				}

				// Lock the workloads' drivers while we marshal the workloads to JSON.
				associatedDriver, _ := m.workloadDrivers.Get(workload.GetId())
				associatedDriver.LockDriver()

				// Update this field.
				workload.UpdateTimeElapsed()
			}
			m.mu.Unlock()

			// Create a message to push to the frontend.
			var msgId string = uuid.NewString()

			// Get a slice of all of the workloads in the 'activeWorkloads' map.
			activeWorkloadsSlice := getMapValues[string, domain.Workload](activeWorkloads)

			// Build a message containing the slice of workloads as its contents.
			responseBuilder := newResponseBuilder(msgId)
			response := responseBuilder.WithModifiedWorkloads(activeWorkloadsSlice).BuildResponse()

			// Encode the message. If we fail to encode the response, then we panic, because that shouldn't happen.
			payload, err := response.Encode()
			if err != nil {
				m.logger.Error("Error while marshalling message payload.", zap.Error(err))
				panic(err)
			}

			// Unlock all of the workload drivers.
			m.mu.Lock()
			for _, workload := range activeWorkloads {
				associatedDriver, _ := m.workloadDrivers.Get(workload.GetId())
				associatedDriver.UnlockDriver()
			}
			m.mu.Unlock()

			// Send an update to the frontend.
			// TODO: Only push updates if something meaningful has changed.
			if err = m.pushWorkloadUpdate(payload); err != nil {
				m.logger.Error("Failed to push workload update to frontend.", zap.Error(err))
			} else {
				m.logger.Debug("Successfully pushed 'Active Workloads' update to frontend.", zap.String("message-id", msgId))
			}

			// Remove workloads that are now inactive from the map.
			// Their data isn't going to change again, so we don't need to keep pushing them to the frontend.
			for _, id := range noLongerActivelyRunning {
				delete(activeWorkloads, id)
			}
		}

		// Sleep for the configured amount of time, and then we'll go again.
		time.Sleep(m.pushUpdateInterval)
	}
}
