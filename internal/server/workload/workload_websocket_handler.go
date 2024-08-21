package workload

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/concurrent_websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	OpGetWorkloads            string = "get_workloads"
	OpRegisterWorkloads       string = "register_workload"
	OpStartWorkload           string = "start_workload"
	OpStopWorkload            string = "stop_workload"
	OpStopWorkloads           string = "stop_workloads"
	OpPauseWorkload           string = "pause_workload"
	OpUnpauseWorkload         string = "unpause_workload"
	OpWorkloadToggleDebugLogs string = "toggle_debug_logs"
	OpWorkloadSubscribe       string = "subscribe"
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

type websocketRequestHandler func(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error)

type WorkloadWebsocketHandler struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	configuration        *domain.Configuration                 // The system/server configuration. This is passed to workload drivers when we create them during workload registration.
	workloadManager      domain.WorkloadManager                // Provides access to all of the workloads.
	workloadMessageIndex atomic.Int32                          // Monotonically increasing index assigned to each outgoing workload message.
	handlers             map[string]websocketRequestHandler    // A map from operation ID to the associated request handler.
	subscribers          map[string]domain.ConcurrentWebSocket // Websockets that have submitted a workload and thus will want updates for that workload.
	expectedOriginPort   int                                   // The origin port expected for incoming WebSocket connections.
	workloadStartedChan  chan<- string                         // Channel of workload IDs. When a workload is started, its ID is submitted to this channel.
}

func NewWorkloadWebsocketHandler(configuration *domain.Configuration, workloadManager domain.WorkloadManager, workloadStartedChan chan<- string, atom *zap.AtomicLevel) *WorkloadWebsocketHandler {
	handler := &WorkloadWebsocketHandler{
		configuration:       configuration,
		workloadManager:     workloadManager,
		atom:                atom,
		handlers:            make(map[string]websocketRequestHandler),
		subscribers:         make(map[string]domain.ConcurrentWebSocket),
		workloadStartedChan: workloadStartedChan,
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
	h.handlers[OpGetWorkloads] = h.handleGetWorkloads
	h.handlers[OpRegisterWorkloads] = h.handleRegisterWorkload
	h.handlers[OpStartWorkload] = h.handleStartWorkload
	h.handlers[OpStopWorkload] = h.handleStopWorkload
	h.handlers[OpStopWorkloads] = h.handleStopWorkloads
	h.handlers[OpPauseWorkload] = h.handlePauseWorkload
	h.handlers[OpUnpauseWorkload] = h.handleUnpauseWorkload
	h.handlers[OpWorkloadToggleDebugLogs] = h.handleToggleDebugLogs
	h.handlers[OpWorkloadSubscribe] = h.handleSubscriptionRequest
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

	return concurrent_websocket.NewConcurrentWebSocket(conn), nil
}

// Offload a workload-related WebSocket request to the appropriate request handler.
//
// Return the message ID (or an empty string if the message ID could not be extracted), the encoded response payload generated by the handler,
// and any errors encountered either while unpacking the message or while the handler processed the message.
func (h *WorkloadWebsocketHandler) dispatchRequest(message []byte, ws domain.ConcurrentWebSocket) (string, []byte, error) {
	var request map[string]interface{}
	if err := json.Unmarshal(message, &request); err != nil {
		h.logger.Error("Error while unmarshalling data message from workload-related websocket.", zap.Error(err), zap.ByteString("message-bytes", message), zap.String("message-string", string(message)))
		return "", nil, err
	}

	h.sugaredLogger.Debugf("Received workload-related WebSocket message: %v", request)

	var (
		opVal    interface{}
		msgIdVal interface{}
		ok       bool
	)

	if msgIdVal, ok = request["msg_id"]; !ok {
		h.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.Binary("message", message))
		return "", nil, ErrMissingMessageId
	}
	msgId := msgIdVal.(string)

	if opVal, ok = request["op"]; !ok {
		h.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.String("msg_id", msgId), zap.Binary("message", message))
		return msgId, nil, ErrMissingOp
	}

	opId := opVal.(string)
	handler, ok := h.handlers[opId]
	if !ok {
		h.logger.Error("Invalid workload-related WebSocket operation requested.", zap.String("operation-id", opId))
		return msgId, nil, fmt.Errorf("%w: \"%s\"", ErrInvalidOperation, opId)
	}

	responsePayload, err := handler(msgId, message, ws)
	return msgId, responsePayload, err
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
		Description:  desciption,
		Valid:        true,
	}
}

// Write a message to the given websocket.
func (h *WorkloadWebsocketHandler) sendMessage(ws domain.ConcurrentWebSocket, payload []byte) error {
	if payload == nil {
		panic("Payload should not be nil when sending a WebSocket message.")
	}

	return ws.WriteMessage(websocket.BinaryMessage, payload)
}

// Given the response and the error returned by a handler, create and return an encoded message to be sent as a response.
//
// If the error is non-nil, then an error message will be created, regardless of the value of the provided response.
// If both the error and the response are nil, then this method will return nil.
func (h *WorkloadWebsocketHandler) getResponsePayload(response []byte, err error) []byte {
	var payload []byte = response // If response is nil, then the payload is nil at this point.
	if err != nil {
		// Error was non-nil, so we'll send back an error message.
		// Overwrite the value of the 'payload' variable with an encoded error message.
		errorMessage := h.generateErrorPayload(err, "")
		payload = errorMessage.Encode()
	}

	return payload
}

// Upgrade the HTTP connection to a WebSocket connection.
// Then, serve requests sent by the remote WebSocket.
func (h *WorkloadWebsocketHandler) serveWorkloadWebsocket(c *gin.Context) {
	h.logger.Debug("Handling workload-related websocket connection")

	ws, err := h.upgradeConnectionToWebsocket(c)
	if err != nil {
		h.logger.Error("Failed to update HTTP connection to WebSocket connection.", zap.Error(err))
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Process messages until the remote client disconnects or an irrecoverable error occurs.
	for {
		// Read the next message from the WebSocket.
		// ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
		// It will block until a message is received and read.
		_, message, err := ws.ReadMessage()
		if err != nil {
			h.logger.Error("Error while reading message from websocket.", zap.Error(err))
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// Handle the request.
		msgId, response, err := h.dispatchRequest(message, ws)

		// Create and encode a response.
		payload := h.getResponsePayload(response, err)

		// If the encoded response is nil, then we won't be sending anything back.
		if payload == nil {
			h.logger.Debug("Not sending response for WebSocket message.", zap.String("msg_id", msgId), zap.Any("message", message))
			continue
		}

		// The encoded response is non-nil, so we'll send it back to the remote client.
		if err = h.sendMessage(ws, response); err != nil {
			h.logger.Error("Failed to write WebSocket response.", zap.String("msg_id", msgId), zap.Any("response", response), zap.Error(err))
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

// Return the currently-registered workloads.
func (h *WorkloadWebsocketHandler) handleGetWorkloads(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	workloads := h.workloadManager.GetWorkloads()

	responseBuilder := newResponseBuilder(msgId)
	response := responseBuilder.WithModifiedWorkloads(workloads).BuildResponse()
	return response.Encode()
}

// Add a websocket to the subscribers field. This is used for workload-related communication.
func (h *WorkloadWebsocketHandler) handleSubscriptionRequest(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	h.subscribers[ws.RemoteAddr().String()] = ws
	return h.handleGetWorkloads(msgId, message, ws)
}

// Remove a websocket from the subscribers field.
func (h *WorkloadWebsocketHandler) removeSubscription(ws domain.ConcurrentWebSocket) {
	if ws.RemoteAddr() != nil {
		h.logger.Debug("Removing subscription for WebSocket.", zap.String("remote-address", ws.RemoteAddr().String()))
		delete(h.subscribers, ws.RemoteAddr().String())
	}
}

// Handle a request to toggle debug logging on/off for a particular workload.
func (h *WorkloadWebsocketHandler) handleToggleDebugLogs(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	req, err := domain.UnmarshalRequestPayload[*domain.ToggleDebugLogsRequest](message)
	if err != nil {
		h.logger.Error("Failed to unmarshal ToggleDebugLogsRequest.", zap.Error(err))
		return nil, err
	}

	modifiedWorkload, err := h.workloadManager.ToggleDebugLogging(req.WorkloadId, req.Enabled)
	if err != nil {
		return nil, err
	}

	// TODO: Consider broadcasting the response?
	responseBuilder := newResponseBuilder(msgId)
	response := responseBuilder.WithModifiedWorkload(modifiedWorkload).BuildResponse()
	return response.Encode()
}

// Handle a request to start a particular workload.
func (h *WorkloadWebsocketHandler) handleStartWorkload(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	req, err := domain.UnmarshalRequestPayload[*domain.StartStopWorkloadRequest](message)
	if err != nil {
		h.logger.Error("Failed to unmarshal StartStopWorkloadRequest.", zap.Error(err))
		return nil, err
	}

	// We're in the 'start workload' handler, but 'StartStopWorkloadRequest' messages can specify an operation ID of either 'start_workload' or 'stop_workload'.
	// So, we're just performing a quick sanity check here to verify that the request did indeed instruct us to start a workload, rather than stop a workload.
	if req.Operation != OpStartWorkload {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	h.logger.Debug("Starting workload.", zap.String("workload-id", req.WorkloadId))

	startedWorkload, err := h.workloadManager.StartWorkload(req.MessageId)
	if err != nil {
		return nil, err
	}

	// Notify the server-push goutine that the workload has started.
	h.workloadStartedChan <- req.WorkloadId

	// TODO: Consider broadcasting the response?
	startedWorkload.UpdateTimeElapsed()
	responseBuilder := newResponseBuilder(msgId)
	response := responseBuilder.WithModifiedWorkload(startedWorkload).BuildResponse()
	return response.Encode()
}

// Handle a request to stop a particular workload.
func (h *WorkloadWebsocketHandler) handleStopWorkload(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	req, err := domain.UnmarshalRequestPayload[*domain.StartStopWorkloadRequest](message)
	if err != nil {
		h.logger.Error("Failed to unmarshal StartStopWorkloadRequest.", zap.Error(err))
		return nil, err
	}

	// We're in the 'start workload' handler, but 'StartStopWorkloadRequest' messages can specify an operation ID of either 'start_workload' or 'stop_workload'.
	// So, we're just performing a quick sanity check here to verify that the request did indeed instruct us to start a workload, rather than stop a workload.
	if req.Operation != OpStopWorkload {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	h.logger.Debug("Stopping workload.", zap.String("workload-id", req.WorkloadId))

	stoppedWorkload, err := h.workloadManager.StopWorkload(req.MessageId)
	if err != nil {
		return nil, err
	}

	// TODO: Consider broadcasting the response?
	stoppedWorkload.UpdateTimeElapsed()
	responseBuilder := newResponseBuilder(msgId)
	response := responseBuilder.WithModifiedWorkload(stoppedWorkload).BuildResponse()
	return response.Encode()
}

// Handle a request to stop 1 or more active workloads.
//
// If one or more of the specified workloads are not stoppable (i.e., they either do not exist, or they're not actively running),
// then this will return an error. However, this will stop all valid workloads specified within the request before returning said error.
func (h *WorkloadWebsocketHandler) handleStopWorkloads(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	req, err := domain.UnmarshalRequestPayload[*domain.StartStopWorkloadsRequest](message)
	if err != nil {
		h.logger.Error("Failed to unmarshal StartStopWorkloadsRequest.", zap.Error(err))
		return nil, err
	}

	if req.Operation != OpStopWorkloads {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadsRequest: \"%s\"", req.Operation))
	}

	// Create a slice for all of the workloads that were stopped.
	// Optimistically pre-allocate enough slots for every workload specified in the request to be successfully stopped.
	// It should usually work, as the frontend generally prevents users for submitting invalid requests.
	// It would only "go wrong" if the frontend's state is out of sync, which should be very uncommon.
	var stoppedWorkloads []domain.Workload = make([]domain.Workload, 0, len(req.WorkloadIDs))

	// Errors accumulated while stopping the workloads specified in the request.
	var errors []error = make([]error, 0)

	for _, workloadId := range req.WorkloadIDs {
		h.logger.Debug("Stopping workload.", zap.String("workload-id", workloadId))

		stoppedWorkload, err := h.workloadManager.StopWorkload(req.MessageId)
		if err != nil {
			h.logger.Error("Failed to stop workload.", zap.String("workload-id", workloadId), zap.Error(err))
			errors = append(errors, err)
			continue
		}

		stoppedWorkload.UpdateTimeElapsed()
		stoppedWorkloads = append(stoppedWorkloads, stoppedWorkload)
	}

	responseBuilder := newResponseBuilder(msgId)
	response := responseBuilder.WithModifiedWorkloads(stoppedWorkloads).BuildResponse()
	return response.Encode()
}

// Handle a request to pause (i.e., temporarily suspend/halt the execution of) an actively-running workload.
//
// This is presently not supported/implemented.
func (h *WorkloadWebsocketHandler) handlePauseWorkload(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	req, err := domain.UnmarshalRequestPayload[*domain.PauseUnpauseWorkloadRequest](message)
	if err != nil {
		h.logger.Error("Failed to unmarshal PauseUnpauseWorkloadRequest.", zap.Error(err))
		return nil, err
	}

	if req.Operation != OpPauseWorkload {
		panic(fmt.Sprintf("Unexpected operation field in PauseUnpauseWorkloadRequest: \"%s\"", req.Operation))
	}

	panic("Not implemented yet.")
}

// Handle a request to unpause (i.e., resume the execution of) a active workload that has previously been paused.
//
// This is presently not supported/implemented.
func (h *WorkloadWebsocketHandler) handleUnpauseWorkload(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	req, err := domain.UnmarshalRequestPayload[*domain.PauseUnpauseWorkloadRequest](message)
	if err != nil {
		h.logger.Error("Failed to unmarshal PauseUnpauseWorkloadRequest.", zap.Error(err))
		return nil, err
	}

	if req.Operation != OpUnpauseWorkload {
		panic(fmt.Sprintf("Unexpected operation field in PauseUnpauseWorkloadRequest: \"%s\"", req.Operation))
	}

	panic("Not implemented yet.")
}

// Handle a request to register a new workload.
// This does not start the workload; that is a separate operation.
func (h *WorkloadWebsocketHandler) handleRegisterWorkload(msgId string, message []byte, ws domain.ConcurrentWebSocket) ([]byte, error) {
	req, err := domain.UnmarshalRequestPayload[*domain.WorkloadRegistrationRequest](message)
	if err != nil {
		h.logger.Error("Failed to unmarshal WorkloadRegistrationRequest.", zap.Error(err))
		return nil, err
	}

	workload, err := h.workloadManager.RegisterWorkload(req, ws)

	responseBuilder := newResponseBuilder(msgId)
	response := responseBuilder.WithNewWorkload(workload).BuildResponse()
	return response.Encode()
}

// Send a binary websocket message to all workload websockets (contained in the 'subscribers' field of the serverImpl struct).
func (h *WorkloadWebsocketHandler) broadcastToWorkloadWebsockets(payload []byte) []error {
	errors := make([]error, 0)

	toRemove := make([]domain.ConcurrentWebSocket, 0)

	for _, ws := range h.subscribers {
		err := ws.WriteMessage(websocket.BinaryMessage, payload)
		if err != nil {
			h.logger.Error("Error while broadcasting websocket message.", zap.Error(err))
			errors = append(errors, err)

			if _, ok := err.(*websocket.CloseError); ok || err == websocket.ErrCloseSent {
				toRemove = append(toRemove, ws)
			}
		}
	}

	for _, ws := range toRemove {
		h.removeSubscription(ws)
	}

	return errors
}
