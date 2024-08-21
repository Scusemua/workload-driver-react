package workload

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/driver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrMissingMessageId = errors.New("WebSocket message did not contain a top-level \"msg_id\" field")
	ErrMissingOp        = errors.New("WebSocket message did not contain a top-level \"op\" field")
	ErrInvalidOperation = errors.New("Invalid workload-related WebSocket operation requested")

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type websocketRequestHandler func(message []byte) ([]byte, error)

type WorkloadWebsocketHandler struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	workloadManager      domain.WorkloadManager             // Provides access to all of the workloads.
	workloadMessageIndex atomic.Int32                       // Monotonically increasing index assigned to each outgoing workload message.
	handlers             map[string]websocketRequestHandler // A map from operation ID to the associated request handler.
	expectedOriginPort   int                                // The origin port expected for incoming WebSocket connections.
}

func NewWorkloadWebsocketHandler(workloadManager domain.WorkloadManager, atom *zap.AtomicLevel) *WorkloadWebsocketHandler {
	handler := &WorkloadWebsocketHandler{
		workloadManager: workloadManager,
		handlers:        make(map[string]websocketRequestHandler),
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	handler.logger = logger
	handler.sugaredLogger = logger.Sugar()
	handler.setupRequestHandlers()

	return handler
}

func (h *WorkloadWebsocketHandler) setupRequestHandlers() {
	h.handlers["get_workloads"] = h.handleGetWorkloads
	h.handlers["register_workload"] = h.handleRegisterWorkload
	h.handlers["start_workload"] = h.handleStartWorkload
	h.handlers["stop_workload"] = h.handleStopWorkload
	h.handlers["stop_workloads"] = h.handleStopWorkloads
	h.handlers["pause_workload"] = h.handlePauseWorkload
	h.handlers["unpause_workload"] = h.handleUnpauseWorkload
	h.handlers["toggle_debug_logs"] = h.handleToggleDebugLogs
	h.handlers["subscribe"] = h.handleSubscriptionRequest
}

// Upgrade the given HTTP connection to a Websocket connection.
// It is the responsibility of the caller to close the websocket when they're done with it.
func (h *WorkloadWebsocketHandler) upgradeConnectionToWebsocket(c *gin.Context) (domain.ConcurrentWebSocket, error) {
	expectedOriginV1 := fmt.Sprintf("http://127.0.0.1:%d", h.expectedOriginPort)
	expectedOriginV2 := fmt.Sprintf("http://localhost:%d", h.expectedOriginPort)
	h.logger.Debug("Handling websocket origin.", zap.String("request-origin", c.Request.Header.Get("Origin")), zap.String("request-host", c.Request.Host), zap.String("request-uri", c.Request.RequestURI), zap.String("expected-origin-v1", expectedOriginV1), zap.String("expected-origin-v2", expectedOriginV2))

	upgrader.CheckOrigin = func(r *http.Request) bool {
		if r.Header.Get("Origin") == expectedOriginV1 || r.Header.Get("Origin") == expectedOriginV2 {
			return true
		}

		h.sugaredLogger.Errorf("Unexpected origin: %v", r.Header.Get("Origin"))
		return false
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection to Websocket.", zap.Error(err))
		return nil, err
	}

	return domain.NewConcurrentWebSocket(conn), nil
}

// Offload a workload-related WebSocket request to the appropriate request handler.
func (h *WorkloadWebsocketHandler) dispatchRequest(message []byte) ([]byte, error) {
	var request map[string]interface{}
	if err := json.Unmarshal(message, &request); err != nil {
		h.logger.Error("Error while unmarshalling data message from workload-related websocket.", zap.Error(err), zap.ByteString("message-bytes", message), zap.String("message-string", string(message)))
		return nil, err
	}

	h.sugaredLogger.Debugf("Received workload-related WebSocket message: %v", request)

	var (
		opVal interface{}
		ok    bool
	)
	if opVal, ok = request["op"]; !ok {
		h.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.Binary("message", message))
		return nil, ErrMissingOp
	}

	opId := opVal.(string)
	handler, ok := h.handlers[opId]
	if !ok {
		h.logger.Error("Invalid workload-related WebSocket operation requested.", zap.String("operation-id", opId))
		return nil, fmt.Errorf("%w: \"%s\"", ErrInvalidOperation, opId)
	}

	return handler(message)
}

// Create and return an ErrorMessage wrapping the given error.
// The error parameter must not be nil.
//
// Arguments:
// - err (error): The error for which we're generating an error payload.
// - description(string): Optional text that may provide additional context or information concerning what went wrong. This is to be written by us.
func (h *WorkloadWebsocketHandler) generateErrorPayload(err error, desciption string) *domain.ErrorMessage {
	if err == nil {
		panic("The provided error should not be nil when generating an error payload.")
	}

	return &domain.ErrorMessage{
		ErrorMessage: err.Error(),
		Description:  extraText,
		Valid:        true,
	}
}

func (h *WorkloadWebsocketHandler) writeResponse(ws domain.ConcurrentWebSocket, response []byte, err error) error {
	var payload []byte

	// If the error is non-nil, then we send back an error message.
	// Otherwise, we send the provided payload.
	if err != nil {
		errorMessage := h.generateErrorPayload(err, "")
		payload = errorMessage.Encode()
	} else {
		payload = response
	}

	return ws.WriteMessage(websocket.BinaryMessage, payload)
}

// Upgrade the HTTP connection to a WebSocket connection.
// Then, serve requests sent by the remote WebSocket.
func (h *WorkloadWebsocketHandler) serveWorkloadWebsocket(c *gin.Context) error {
	h.logger.Debug("Handling workload-related websocket connection")

	ws, err := h.upgradeConnectionToWebsocket(c)
	if err != nil {
		h.logger.Error("Failed to update HTTP connection to WebSocket connection.", zap.Error(err))
		c.AbortWithError(http.StatusInternalServerError, err)
		return err
	}

	// Used to notify the server-push goroutine that a new workload has been registered.
	workloadStartedChan := make(chan string)
	doneChan := make(chan struct{})
	go h.serverPushRoutine(workloadStartedChan, doneChan)

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			h.logger.Error("Error while reading message from websocket.", zap.Error(err))
			return err
		}

		response, err := h.dispatchRequest(message)
		if err = h.writeResponse(ws, response, err); err != nil {
			h.logger.Error("Failed to write WebSocket response.", zap.Any("response", response), zap.Error(err))
			c.AbortWithError(http.StatusInternalServerError, err)
			return err
		}
	}

	return nil
}

func (h *WorkloadWebsocketHandler) handleGetWorkloads(msgId string, conn domain.ConcurrentWebSocket, broadcastToWorkloadWebsockets bool) ([]byte, error) {
	// h.driversMutex.RLock()
	// for el := h.workloadDrivers.Front(); el != nil; el = el.Next() {
	// 	el.Value.LockDriver()
	// }

	// payload, err := json.Marshal(h.createWorkloadResponseMessage(msgId, nil, h.workloads, nil))

	// if err != nil {
	// 	h.logger.Error("Error while marshalling message payload.", zap.Error(err))
	// 	panic(err)
	// }

	// for el := h.workloadDrivers.Front(); el != nil; el = el.Next() {
	// 	el.Value.UnlockDriver()
	// }
	// h.driversMutex.RUnlock()

	// h.sugaredLogger.Debugf("Returning %d workloads to user.", len(h.workloads))

	// if broadcastToWorkloadWebsockets {
	// 	h.broadcastToWorkloadWebsockets(payload)
	// }
	// if conn != nil {
	// 	conn.WriteMessage(websocket.BinaryMessage, payload)
	// }

	// h.logger.Debug("Wrote response for GET_WORKLOADS to frontend.", zap.String("message-id", msgId))

	workloads := h.workloadManager.GetWorkloads()

	responseBuilder := h.newResponseBuilder()
	response := responseBuilder.WithModifiedWorkloads(workloads).BuildResponse()
	return response.Encode()
}

// Used to push updates about active workloads to the frontend.
func (h *WorkloadWebsocketHandler) serverPushRoutine(workloadStartedChan chan string, doneChan chan struct{}) {
	// Keep track of the active workloads.
	activeWorkloads := make(map[string]domain.Workload)

	// Add all active workloads to the map.
	for _, workload := range h.workloads {
		if workload.IsRunning() {
			activeWorkloads[workload.GetId()] = workload
		}
	}

	// We'll loop forever, unless the connection is terminated.
	for {
		// If we have any active workloads, then we'll push some updates to the front-end for the active workloads.
		if len(activeWorkloads) > 0 {
			toRemove := make([]string, 0)
			updatedWorkloads := make([]domain.Workload, 0)

			h.driversMutex.RLock()
			// Iterate over all the active workloads.
			for _, workload := range activeWorkloads {
				// If the workload is no longer active, then make a note to remove it after this next update.
				// (We need to include it in the update so the frontend knows it's no longer active.)
				if !workload.IsRunning() {
					toRemove = append(toRemove, workload.GetId())
				}

				associatedDriver, _ := h.workloadDrivers.Get(workload.GetId())
				associatedDriver.LockDriver()

				workload.UpdateTimeElapsed() // Update this field.

				// Lock the workloads' drivers while we marshal the workloads to JSON.
				updatedWorkloads = append(updatedWorkloads, workload)
			}
			h.driversMutex.RUnlock()

			var msgId string = uuid.NewString()
			payload, err := json.Marshal(h.createWorkloadResponseMessage(msgId, nil, updatedWorkloads, nil))
			// 	&domain.WorkloadResponse{
			// 	MessageId:         msgId,
			// 	ModifiedWorkloads: updatedWorkloads,
			// 	MessageIndex:      h.workloadMessageIndex.Add(1),
			// })

			if err != nil {
				h.logger.Error("Error while marshalling message payload.", zap.Error(err))
				panic(err)
			}

			h.driversMutex.RLock()
			for _, workload := range updatedWorkloads {
				associatedDriver, _ := h.workloadDrivers.Get(workload.GetId())
				associatedDriver.UnlockDriver()
			}
			h.driversMutex.RUnlock()

			// Send an update to the frontend.
			h.broadcastToWorkloadWebsockets(payload)

			// TODO: Only push updates if something meaningful has changed.
			h.logger.Debug("Pushed 'Active Workloads' update to frontend.", zap.String("message-id", msgId))

			// Remove workloads that are now inactive from the map.
			for _, id := range toRemove {
				delete(activeWorkloads, id)
			}
		}

		// In case there are a bunch of notifications in the 'workload started channel', consume all of them before breaking out.
		var done bool = false
		for !done {
			// Do stuff.
			select {
			case id := <-workloadStartedChan:
				{
					h.workloadsMutex.RLock()
					// Add the newly-registered workload to the active workloads map.
					activeWorkloads[id], _ = h.workloadsMap.Get(id)
					h.workloadsMutex.RUnlock()
				}
			case <-doneChan:
				{
					return
				}
			default:
				// Do nothing.
				time.Sleep(time.Second * 2)
				done = true // No more notifications right now. We'll process what we have.
			}
		}
	}
}

// Add a websocket to the subscribers field. This is used for workload-related communication.
func (h *WorkloadWebsocketHandler) handleSubscriptionRequest(req *domain.SubscriptionRequest, conn domain.ConcurrentWebSocket) {
	h.subscribers[conn.RemoteAddr().String()] = conn
	h.handleGetWorkloads(req.MessageId, conn, false)
}

// Remove a websocket from the subscribers field.
func (h *WorkloadWebsocketHandler) removeSubscription(conn domain.ConcurrentWebSocket) {
	if conn.RemoteAddr() != nil {
		h.logger.Debug("Removing subscription for WebSocket.", zap.String("remote-address", conn.RemoteAddr().String()))
		delete(h.subscribers, conn.RemoteAddr().String())
	}
}

// Send a binary websocket message to all workload websockets (contained in the 'subscribers' field of the serverImpl struct).
func (h *WorkloadWebsocketHandler) broadcastToWorkloadWebsockets(payload []byte) []error {
	errors := make([]error, 0)

	toRemove := make([]domain.ConcurrentWebSocket, 0)

	for _, conn := range h.subscribers {
		err := conn.WriteMessage(websocket.BinaryMessage, payload)
		if err != nil {
			h.logger.Error("Error while broadcasting websocket message.", zap.Error(err))
			errors = append(errors, err)

			if _, ok := err.(*websocket.CloseError); ok || err == websocket.ErrCloseSent {
				toRemove = append(toRemove, conn)
			}
		}
	}

	for _, conn := range toRemove {
		h.removeSubscription(conn)
	}

	return errors
}

func (h *WorkloadWebsocketHandler) newResponseBuilder() *responseBuilder {
	return newResponseBuilder(h.workloadMessageIndex.Add(1))
}

// Create and return a *domain.WorkloadResponse struct.
// We use this function so we can increment the message index field.
func (h *WorkloadWebsocketHandler) createWorkloadResponseMessage(id string, new []domain.Workload, modified []domain.Workload, deleted []domain.Workload) *domain.WorkloadResponse {
	return &domain.WorkloadResponse{
		MessageId:         id,
		NewWorkloads:      new,
		ModifiedWorkloads: modified,
		DeletedWorkloads:  deleted,
		// MessageIndex:      h.workloadMessageIndex.Add(1),
	}
}

func (h *WorkloadWebsocketHandler) handleToggleDebugLogs(req *domain.ToggleDebugLogsRequest) {
	h.driversMutex.RLock()
	driver, _ := h.workloadDrivers.Get(req.WorkloadId)
	h.driversMutex.RUnlock()

	if driver != nil {
		workload := driver.ToggleDebugLogging(req.Enabled)

		driver.LockDriver()
		payload, err := json.Marshal(h.createWorkloadResponseMessage(req.MessageId, nil, []domain.Workload{workload}, nil))
		// &domain.WorkloadResponse{
		// 	MessageId:         req.MessageId,
		// 	ModifiedWorkloads: []domain.Workload{workload},
		// })
		driver.UnlockDriver()

		if err != nil {
			h.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		h.broadcastToWorkloadWebsockets(payload)

		h.logger.Debug("Wrote response for TOGGLE_DEBUG_LOGS to frontend.", zap.String("message-id", req.MessageId))
	} else {
		h.sugaredLogger.Errorf("Could not find driver associated with workload ID=%s", req.WorkloadId)
	}
}

func (h *WorkloadWebsocketHandler) handleStartWorkload(req *domain.StartStopWorkloadRequest, workloadStartedChan chan string) {
	if req.Operation != "start_workload" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	h.logger.Debug("Starting workload.", zap.String("workload-id", req.WorkloadId))

	h.driversMutex.RLock()
	workloadDriver, ok := h.workloadDrivers.Get(req.WorkloadId)
	h.driversMutex.RUnlock()

	if ok {
		// var wg sync.WaitGroup
		// wg.Add(1)

		// Start the workload.
		// This sets the "start time" and transitions the workload to the "running" state.
		workloadDriver.GetWorkload().StartWorkload()

		go workloadDriver.ProcessWorkload(nil) // &wg
		go workloadDriver.DriveWorkload(nil)   // &wg

		h.workloadsMutex.RLock()
		workload, _ := h.workloadsMap.Get(req.WorkloadId)
		workload.UpdateTimeElapsed()
		h.workloadsMutex.RUnlock()

		h.logger.Debug("Started workload.", zap.String("workload-id", req.WorkloadId), zap.Any("workload-source", workload.GetWorkloadSource()))

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(h.createWorkloadResponseMessage(req.MessageId, nil, []domain.Workload{workload}, nil))
		// 	&domain.WorkloadResponse{
		// 	MessageId:         req.MessageId,
		// 	ModifiedWorkloads: []domain.Workload{workload},
		// }
		workloadDriver.UnlockDriver()

		if err != nil {
			h.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		h.broadcastToWorkloadWebsockets(payload)

		h.logger.Debug("Wrote response for START_WORKLOAD to frontend.", zap.String("message-id", req.MessageId), zap.String("workload-id", workloadDriver.ID()))

		// Notify the server-push goutine that the workload has started.
		workloadStartedChan <- req.WorkloadId
	} else {
		h.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", req.WorkloadId))
	}
}

func (h *WorkloadWebsocketHandler) handlePauseWorkload(req *domain.PauseUnpauseWorkloadRequest) {
	panic("Not implemented yet.")
}

func (h *WorkloadWebsocketHandler) handleUnpauseWorkload(req *domain.PauseUnpauseWorkloadRequest) {
	panic("Not implemented yet.")
}

func (h *WorkloadWebsocketHandler) handleStopWorkloads(req *domain.StartStopWorkloadsRequest) {
	if req.Operation != "stop_workloads" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	var updatedWorkloads []domain.Workload = make([]domain.Workload, 0, len(req.WorkloadIDs))

	for _, workloadID := range req.WorkloadIDs {
		h.logger.Debug("Stopping workload.", zap.String("workload-id", workloadID))

		h.driversMutex.RLock()
		workloadDriver, ok := h.workloadDrivers.Get(workloadID)
		h.driversMutex.RUnlock()

		if ok {
			err := workloadDriver.StopWorkload()
			if err != nil {
				h.logger.Error("Error encountered when trying to stop workload.", zap.String("workload-id", workloadID), zap.Error(err))
			} else {
				workload := workloadDriver.GetWorkload()
				// workload.TimeElasped = time.Since(workload.StartTime).String()
				workload.UpdateTimeElapsed()

				h.logger.Debug("Stopped workload.", zap.String("workload-id", workloadID), zap.Any("workload-source", workload.GetWorkloadSource()))
				updatedWorkloads = append(updatedWorkloads, workload)
			}
		} else {
			h.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", workloadID))
		}
	}

	// Lock the workload's driver while we marshal the workload to JSON.
	payload, err := json.Marshal(h.createWorkloadResponseMessage(uuid.NewString(), nil, updatedWorkloads, nil))
	// 	&domain.WorkloadResponse{
	// 	MessageId:         msgId,
	// 	ModifiedWorkloads: updatedWorkloads,
	// }

	if err != nil {
		h.logger.Error("Error while marshalling message payload.", zap.Error(err))
		panic(err)
	}

	h.driversMutex.RLock()
	for _, workload := range updatedWorkloads {
		associatedDriver, _ := h.workloadDrivers.Get(workload.GetId())
		associatedDriver.UnlockDriver()
	}
	h.driversMutex.RUnlock()

	h.broadcastToWorkloadWebsockets(payload)

	h.logger.Debug("Wrote response for STOP_WORKLOADS to frontend.", zap.String("message-id", req.MessageId), zap.Int("requested-num-workloads-stopped", len(req.WorkloadIDs)), zap.Int("actual-num-workloads-stopped", len(updatedWorkloads)))
}

func (h *WorkloadWebsocketHandler) handleStopWorkload(req *domain.StartStopWorkloadRequest) {
	if req.Operation != "stop_workload" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	h.logger.Debug("Stopping workload.", zap.String("workload-id", req.WorkloadId))

	h.driversMutex.RLock()
	workloadDriver, ok := h.workloadDrivers.Get(req.WorkloadId)
	h.driversMutex.RUnlock()

	if ok {
		err := workloadDriver.StopWorkload()
		if err != nil {
			h.logger.Error("Error encountered when trying to stop workload.", zap.String("workload-id", req.WorkloadId), zap.Error(err))
		} else {
			workload := workloadDriver.GetWorkload()
			// workload.TimeElasped = time.Since(workload.StartTime).String()
			workload.UpdateTimeElapsed()

			h.logger.Debug("Stopped workload.", zap.String("workload-id", req.WorkloadId), zap.Any("workload-source", workload.GetWorkloadSource()))
		}

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(h.createWorkloadResponseMessage(req.MessageId, nil, []domain.Workload{workloadDriver.GetWorkload()}, nil))
		// 	&domain.WorkloadResponse{
		// 	MessageId:         req.MessageId,
		// 	ModifiedWorkloads: []domain.Workload{workloadDriver.GetWorkload()},
		// }
		workloadDriver.UnlockDriver()

		if err != nil {
			h.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		h.broadcastToWorkloadWebsockets(payload)

		h.logger.Debug("Wrote response for STOP_WORKLOAD to frontend.", zap.String("message-id", req.MessageId), zap.String("workload-id", req.WorkloadId))
	} else {
		h.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", req.WorkloadId))
	}
}

func (h *WorkloadWebsocketHandler) handleRegisterWorkload(request *domain.WorkloadRegistrationRequest, msgId string, websocket domain.ConcurrentWebSocket) {
	workloadDriver := driver.NewWorkloadDriver(h.opts, true, request.TimescaleAdjustmentFactor, websocket, h.atom)

	workload, _ := workloadDriver.RegisterWorkload(request)

	if workload != nil {
		h.workloadsMutex.Lock()
		h.workloads = append(h.workloads, workload)
		h.workloadsMap.Set(workload.GetId(), workload)
		h.workloadsMutex.Unlock()

		h.driversMutex.Lock()
		h.workloadDrivers.Set(workload.GetId(), workloadDriver)
		h.driversMutex.Unlock()

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(h.createWorkloadResponseMessage(msgId, []domain.Workload{workload}, nil, nil))
		// 	&domain.WorkloadResponse{
		// 	MessageId:    msgId,
		// 	NewWorkloads: []domain.Workload{workload},
		// }
		workloadDriver.UnlockDriver()

		if err != nil {
			h.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		h.broadcastToWorkloadWebsockets(payload)

		h.logger.Debug("Wrote response for REGISTER_WORKLOAD to frontend.", zap.String("message-id", msgId), zap.Any("workload-source", workload.GetWorkloadSource()), zap.Any("workload-id", workload.GetId()))
	} else {
		h.logger.Error("Workload registration did not return a Workload object...")
	}
}
