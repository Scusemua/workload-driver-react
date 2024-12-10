package workload

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/zhangjyr/gocsv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ClusterStatisticsRefresher func(update bool, clear bool) (*ClusterStatistics, error)

type BasicWorkloadManager struct {
	atom          *zap.AtomicLevel
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	configuration            *domain.Configuration                                // Server-wide configuration.
	pushGoroutineActive      atomic.Int32                                         // Indicates whether there is already a goroutine serving the "push" routine, which pushes updated workload data to the frontend.
	pushUpdateInterval       time.Duration                                        // The interval at which we push updates to the workloads to the frontend.
	workloadWebsocketHandler *WebsocketHandler                                    // Workload WebSocket handler. Accepts and processes WebSocket requests related to workloads.
	workloadDrivers          *orderedmap.OrderedMap[string, *BasicWorkloadDriver] // Map from workload ID to the associated driver.
	workloadsMap             *orderedmap.OrderedMap[string, domain.Workload]      // Map from workload ID to workload
	workloads                []domain.Workload                                    // Slice of workloads. Same contents as the map, but in slice form.
	mu                       sync.Mutex                                           // Synchronizes access to the workload drivers and the workloads themselves (both the map and the slice).
	workloadStartedChan      chan string                                          // Channel of workload IDs. When a workload is started, its ID is submitted to this channel.
	callbackProvider         CallbackProvider                                     // callbackProvider provides a number of functions required by the WorkloadManager, WorkloadDriver instances, or Workload instances themselves.
}

func init() {
	jsonpatch.SupportNegativeIndices = false

	gocsv.SetHeaderNormalizer(func(s string) string {
		if v, ok := headerRegister.Load(s); ok {
			if v, ok := v.(CSVHeaderProvider); ok {
				return v.MarshalCSVHeader(s)
			}
		}
		return s
	})
}

// CallbackProvider provides a number of functions required by the WorkloadManager, WorkloadDriver instances,
// or Workload instances themselves.
type CallbackProvider interface {
	RefreshAndClearClusterStatistics(update bool, clear bool) (*ClusterStatistics, error)

	// HandleCriticalWorkloadError is a callback passed to WorkloadDrivers (via the WorkloadManager).
	// If a critical error occurs during the execution of the workload, then this handler is called.
	HandleCriticalWorkloadError(workloadId string, err error)

	// HandleWorkloadError is a callback passed to WorkloadDrivers (via the WorkloadManager).
	// If a non-critical error occurs during the execution of the workload, then this handler is called.
	HandleWorkloadError(workloadId string, err error)

	// SendNotification is an RPC handler that is called by the Cluster Scheduler to send notifications to the frontend.
	// We also call it directly to send our own notifications to the frontend.
	SendNotification(notification *proto.Notification)

	// GetSchedulingPolicy returns the configured scheduling policy along with a flag indicating whether the returned
	// policy name is valid.
	GetSchedulingPolicy() (string, bool)
}

func NewWorkloadManager(configuration *domain.Configuration, atom *zap.AtomicLevel, callbackProvider CallbackProvider) *BasicWorkloadManager {
	manager := &BasicWorkloadManager{
		atom:                atom,
		configuration:       configuration,
		workloadDrivers:     orderedmap.NewOrderedMap[string, *BasicWorkloadDriver](),
		workloadsMap:        orderedmap.NewOrderedMap[string, domain.Workload](),
		workloads:           make([]domain.Workload, 0),
		workloadStartedChan: make(chan string, 4),
		pushUpdateInterval:  time.Second * time.Duration(configuration.PushUpdateInterval),
		//onCriticalError:          provider.HandleCriticalWorkloadError,
		//onNonCriticalError:       provider.HandleWorkloadError,
		//notifyCallback:           provider.SendNotification,
		//refreshClusterStatistics: provider.RefreshAndClearClusterStatistics,
		//getSchedulingPolicy:      provider.GetSchedulingPolicy,
		callbackProvider: callbackProvider,
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

	manager.workloadWebsocketHandler = NewWebsocketHandler(configuration, manager, manager.workloadStartedChan, atom)
	manager.pushGoroutineActive.Store(0)

	return manager
}

// GetWorkloadWebsocketHandler returns a function that can handle WebSocket requests for workload operations.
//
// This simply returns the handler function of the WebsocketHandler struct of the WorkloadManager.
func (m *BasicWorkloadManager) GetWorkloadWebsocketHandler() gin.HandlerFunc {
	if m.pushGoroutineActive.CompareAndSwap(0, 1) {
		go m.serverPushRoutine()
	}

	return m.workloadWebsocketHandler.serveWorkloadWebsocket
}

// GetWorkloads returns a slice containing all currently-registered workloads (at the time that the method is called).
// The workloads within this slice should not be modified by the caller.
func (m *BasicWorkloadManager) GetWorkloads() []domain.Workload {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.workloads
}

// GetActiveWorkloads returns a map from Workload ID to Workload struct containing workloads that are active when the method is called.
func (m *BasicWorkloadManager) GetActiveWorkloads() map[string]domain.Workload {
	m.mu.Lock()
	defer m.mu.Unlock()

	activeWorkloads := make(map[string]domain.Workload)

	for _, workload := range m.workloads {
		if workload.IsInProgress() {
			activeWorkloads[workload.GetId()] = workload
		}
	}

	return activeWorkloads
}

// GetWorkloadDriver returns the workload driver associated with the given workload ID.
// If there is no driver associated with the provided workload ID, then nil is returned.
func (m *BasicWorkloadManager) GetWorkloadDriver(workloadId string) *BasicWorkloadDriver {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.workloadDrivers.GetOrDefault(workloadId, nil)
}

// ToggleDebugLogging toggles debug logging on or off (depending on the value of the 'enabled' parameter) for the specified workload.
// If there is no workload with the specified ID, then an error is returned.
//
// If successful, then this returns the updated workload.
func (m *BasicWorkloadManager) ToggleDebugLogging(workloadId string, enabled bool) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	updatedWorkload := workloadDriver.ToggleDebugLogging(enabled)
	return updatedWorkload, nil
}

// RegisterOnCriticalErrorHandler registers a critical error handler for the specified workload.
//
// If the specified workload cannot be found, then an error is returned.
//
// If there is already a critical handler error registered for the particular workload, then the existing
// critical error handler is overwritten.
func (m *BasicWorkloadManager) RegisterOnCriticalErrorHandler(workloadId string, handler domain.WorkloadErrorHandler) error {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		m.logger.Error("Cannot register critical error handler for workload as that workload has not yet been "+
			"registered with the Workload Manager.", zap.String("workload_id", workloadId))
		return fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	workloadDriver.onCriticalErrorOccurred = handler
	return nil
}

// RegisterOnNonCriticalErrorHandler registers a non-critical error handler for the specified workload.
//
// If the specified workload cannot be found, then an error is returned.
//
// If there is already a non-critical handler error registered for the particular workload, then the existing
// non-critical error handler is overwritten.
func (m *BasicWorkloadManager) RegisterOnNonCriticalErrorHandler(workloadId string, handler domain.WorkloadErrorHandler) error {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		m.logger.Error("Cannot register critical error handler for workload as that workload has not yet been "+
			"registered with the Workload Manager.", zap.String("workload_id", workloadId))
		return fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	workloadDriver.onNonCriticalErrorOccurred = handler
	return nil
}

// StartWorkload starts the workload with the specified ID.
// The workload must have already been registered.
//
// If successful, then this returns the updated workload.
// If there is no workload with the specified ID, then an error is returned.
// Likewise, if the specified workload is either already-running or has already been stopped, then an error is returned.
func (m *BasicWorkloadManager) StartWorkload(workloadId string) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		m.logger.Error("Cannot register critical error handler for workload as that workload has not yet been "+
			"registered with the Workload Manager.", zap.String("workload_id", workloadId))
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	// Start the workload.
	// This sets the "start time" and transitions the workload to the "running" state.
	err := workloadDriver.StartWorkload()
	if err != nil {
		return nil, err
	}

	go workloadDriver.ProcessWorkloadEvents() // &wg
	go workloadDriver.DriveWorkload()         // &wg

	workload := workloadDriver.GetWorkload()
	workload.UpdateTimeElapsed()

	m.logger.Debug("Started workload.",
		zap.String("workload_id", workloadId),
		zap.String("workload_name", workload.WorkloadName()))

	return workload, nil
}

// StopWorkload stops the workload with the specified ID.
// The workload must have already been registered and should be actively-running.
//
// If successful, then this returns the updated workload.
// If there is no workload with the specified ID, or the specified workload is not actively-running, then an error is returned.
func (m *BasicWorkloadManager) StopWorkload(workloadId string) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		m.logger.Error("Could not find workload driver with specified workload ID.", zap.String("workload_id", workloadId))
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	// Stop the workload.
	err := workloadDriver.StopWorkload()
	if err != nil {
		return nil, err
	}

	return workloadDriver.GetWorkload(), nil
}

// PauseWorkload attempts to pause the specified workload.
//
// Whatever tick is currently being processed will process to its end. Then, the next tick will not be
// processed until the workload is unpaused.
func (m *BasicWorkloadManager) PauseWorkload(workloadId string) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		m.logger.Error("Could not find workload driver with specified workload ID.", zap.String("workload_id", workloadId))
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	// Pause the workload.
	err := workloadDriver.PauseWorkload()
	if err != nil {
		return nil, err
	}

	return workloadDriver.GetWorkload(), nil
}

// UnpauseWorkload attempts to pause the specified workload.
func (m *BasicWorkloadManager) UnpauseWorkload(workloadId string) (domain.Workload, error) {
	workloadDriver := m.GetWorkloadDriver(workloadId)
	if workloadDriver == nil {
		m.logger.Error("Could not find workload driver with specified workload ID.", zap.String("workload_id", workloadId))
		return nil, fmt.Errorf("%w: \"%s\"", domain.ErrWorkloadNotFound, workloadId)
	}

	// Unpause the workload.
	err := workloadDriver.UnpauseWorkload()
	if err != nil {
		return nil, err
	}

	return workloadDriver.GetWorkload(), nil
}

// RegisterWorkload registers a new workload.
func (m *BasicWorkloadManager) RegisterWorkload(request *domain.WorkloadRegistrationRequest, ws domain.ConcurrentWebSocket) (domain.Workload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new workload driver.
	workloadDriver := NewBasicWorkloadDriver(m.configuration, true, request.TimescaleAdjustmentFactor,
		ws, m.atom, m.callbackProvider)

	// Register a new workload with the workload driver.
	workload, err := workloadDriver.RegisterWorkload(request)
	if err != nil {
		m.logger.Error("Failed to create and register new workload.", zap.Any("workload-registration-request", request), zap.Error(err))
		return nil, err
	}

	// Update our internal state and perform the necessary bookkeeping.
	workloadId := workload.GetId()
	m.workloads = append(m.workloads, workload)
	m.workloadsMap.Set(workloadId, workload)
	m.workloadDrivers.Set(workloadId, workloadDriver)

	m.logger.Debug("Successfully registered workload with Workload Manager.",
		zap.String("workload_id", workloadId),
		zap.String("workload_name", workload.WorkloadName()),
		zap.String("workload", workload.String()))

	return workload, err
}

// Push an update to the frontend.
// patchPayload is a JSON PATCH, and fullPayload is the full, encoded workload state.
func (m *BasicWorkloadManager) pushWorkloadUpdate(payload []byte) error {
	errs := m.workloadWebsocketHandler.broadcastToWorkloadWebsockets(payload)

	if len(errs) >= 1 {
		return errors.Join(errs...)
	}

	return nil
}

// Used to push updates about active workloads to the frontend.
func (m *BasicWorkloadManager) serverPushRoutine( /* doneChan chan struct{} */ ) {
	activeWorkloads := m.GetActiveWorkloads()

	// Function that continuously pulls workload IDs out of the 'workloadStartedChan' until there are none left.
	// This returns the number of new workloads detected.
	checkForNewActiveWorkloads := func() int {
		var numNewActiveWorkloads = 0

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

	// Cache the previous "workload state" so that we can just create a patch from
	// the current workload state and send the patch, which should be much smaller.
	previousWorkloadsEncoded := make(map[string][]byte)

	// We'll loop until the underlying WebSocket connection is terminated.
	for {
		// Check for any newly-registered workloads before pushing an update.
		checkForNewActiveWorkloads()

		// If we have any active workloads, then we'll push some updates to the front-end for the active workloads.
		if len(activeWorkloads) > 0 {
			// Keep track of any workloads that are no longer active.
			// We'll push one more update for these workloads and then stop pushing updates for them,
			// as the state/data of non-active workl	oads does not change.
			noLongerActivelyRunning := make([]string, 0)
			activeWorkloadsSlice := make([]domain.Workload, 0)

			m.mu.Lock()
			// Iterate over all the active workloads.
			for _, workload := range activeWorkloads {
				// If the workload is no longer active, then make a note to remove it after this next update.
				// (We need to include it in the update so the frontend knows it's no longer active.)
				if !workload.IsInProgress() {
					// This workload is no longer active. We'll push it to the frontend one last time,
					// and then we'll stop pushing updates for this workload.
					noLongerActivelyRunning = append(noLongerActivelyRunning, workload.GetId())
				}

				// Update this field.
				workload.UpdateTimeElapsed()
				activeWorkloadsSlice = append(activeWorkloadsSlice, workload)
			}
			m.mu.Unlock()

			// Create a message to push to the frontend.
			var msgId = uuid.NewString()
			responseBuilder := newResponseBuilder(msgId, OpPushedWorkloadUpdate)

			allWorkloadsEncodedSizeBytes := 0
			for _, workload := range activeWorkloadsSlice {
				workloadEncoded, err := json.Marshal(workload)
				if err != nil {
					panic(err)
				}

				prevEncoding, loaded := previousWorkloadsEncoded[workload.GetId()]
				if loaded {
					patch, err := jsonpatch.CreateMergePatch(prevEncoding, workloadEncoded)
					if err != nil {
						m.logger.Error("Failed to create merge patch for workload.", zap.Any("workload", workload), zap.Error(err))
						responseBuilder.AddModifiedWorkload(workload)
					} else {
						//m.logger.Debug("Creating patch for workload.", zap.ByteString("patch", patch))
						responseBuilder.AddModifiedWorkloadAsPatch(patch, workload.GetId())
					}
				} else {
					responseBuilder.AddModifiedWorkload(workload)
				}

				previousWorkloadsEncoded[workload.GetId()] = workloadEncoded
				allWorkloadsEncodedSizeBytes += len(workloadEncoded)
			}

			responseEncoded, err := responseBuilder.BuildResponse().Encode()
			if err != nil {
				panic(err)
			}

			// Send an update to the frontend.
			// TODO: Only push updates if something meaningful has changed.
			// TODO: This is written as if it supports multiple clients, but if a new client comes in after this routine has started, then it won't work.
			// Specifically, this doesn't consider what the state of the client is, so it's just sending out whatever payload, either a JSON PATCH
			// or not, regardless of what the client already has.
			//
			// New clients need to receive the full workload, so maybe we should pass both to pushWorkloadUpdate, and some logic in there
			// will determine if the client needs the full workload or just a patch.
			if err := m.pushWorkloadUpdate(responseEncoded); err != nil && !errors.Is(err, websocket.ErrCloseSent) && !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				m.logger.Error("Failed to push workload update to frontend.", zap.Error(err))
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
