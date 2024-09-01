package jupyter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	JavascriptISOString = "2006-01-02T15:04:05.999Z07:00"
	kernelServiceApi    = "api/kernels"

	KernelConnectionInit KernelConnectionStatus = "initializing" // When the BasicKernelConnection struct is first created.
	KernelConnecting     KernelConnectionStatus = "connecting"   // When we are creating the kernel websocket.
	KernelConnected      KernelConnectionStatus = "connected"    // Once we've connected.
	KernelDisconnected   KernelConnectionStatus = "disconnected" // We're not connected to the kernel, but we're unsure if it is dead or not.
	KernelDead           KernelConnectionStatus = "dead"         // Kernel is dead. We're not connected.

	ExecuteRequest          MessageType = "execute_request"
	KernelInfoRequest       MessageType = "kernel_info_request"
	StopRunningTrainingCode MessageType = "stop_running_training_code_request"
	DummyMessage            MessageType = "dummy_message_request"
	AckMessage              MessageType = "ACK"
	CommCloseMessage        MessageType = "comm_close"
)

var (
	ErrWebsocketAlreadySetup   = errors.New("the kernel connection's websocket has already been setup")
	ErrWebsocketCreationFailed = errors.New("creation of websocket connection to kernel has failed")
	ErrKernelNotFound          = errors.New("received HTTP 404 status when requesting info for kernel")
	ErrNetworkIssue            = errors.New("received HTTP 503 or HTTP 424 in response to request")
	ErrUnexpectedFailure       = errors.New("the request could not be completed for some unexpected reason")
	ErrKernelIsDead            = errors.New("kernel is dead")
	ErrNotConnected            = errors.New("kernel is not connected")
	ErrCantAckNotRegistered    = errors.New("cannot ACK message as registration for associated channel has not yet completed")
)

type MessageType string

func (t MessageType) String() string {
	return string(t)
}

// If the message type is not of the form "{action}_request" or "{action}_reply", then this will panic.
func (t MessageType) getBaseMessageType() string {
	if strings.HasSuffix(t.String(), "request") {
		return t.String()[0 : len(t.String())-7]
	} else if strings.HasSuffix(t.String(), "reply") {
		return t.String()[0 : len(t.String())-5]
	}

	panic(fmt.Sprintf("Invalid message type: \"%s\"", t))
}

type KernelConnectionStatus string

type BasicKernelConnection struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	// TODO: The response delivery mechanism is wrong. For one, control/shell messages should receive messages on that channel.
	// But if we receive an IOPub message with a parent as a shell message, then the IOPub message is considered to be the response.
	// So, we need to look at parent header request ID, channel type, AND we also need to check the message type.
	// The request is in the form <action>_request and the response is <action>_reply.

	// Register callbacks for responses to particular messages.
	//
	// For now, we only support responses for SHELL and CONTROL messages.
	//
	// Keys for this channel are generated by the 'getResponseChannelKeyX' functions defined in "internal/server/jupyter/utils.go".
	// See the documentation of those functions for additional details.
	responseChannels map[string]chan KernelMessage

	// IOPub message handlers.
	iopubMessageHandlers map[string]IOPubMessageHandler

	messageCount                  int                     // How many messages we've sent. Used when creating message IDs.
	connectionStatus              KernelConnectionStatus  // Connection status with the remote kernel.
	kernelId                      string                  // ID of the associated kernel
	jupyterServerAddress          string                  // Jupyter server IP address
	clientId                      string                  // Jupyter client ID
	username                      string                  // Jupyter username
	webSocket                     *websocket.Conn         // The websocket that is connected to Jupyter
	originalWebsocketCloseHandler func(int, string) error // The original close handler method of the websocket; we replace this with our own, and we call the original from ours.
	model                         *jupyterKernel          // Jupyter kernel model.
	kernelStdout                  []string                // STDOUT history from the kernel, as extracted from IOPub messages.
	kernelStderr                  []string                // STDERR history from the kernel, as extracted from IOPub messages.
	registeredShell               bool                    // True if we've successfully registered our shell channel as a Golang frontend.
	registeredControl             bool                    // True if we've successfully registered our control channel as a Golang frontend.

	// Gorilla Websockets support 1 concurrent reader and 1 concurrent writer on the same websocket.
	// What this means is that we can read from the websocket with one goroutine while we write to the websocket with another goroutine.
	// However, we cannot have > 1 goroutines reading at the same time, nor can we have > 1 goroutines write at the same time.
	// So, we have two locks: one for reading, and one for writing.

	rlock             sync.Mutex // Synchronizes read operations on the websocket.
	wlock             sync.Mutex // Synchronizes write operations on the websocket.
	iopubHandlerMutex sync.Mutex // Synchronizes access to state related to the IOPub message handlers.
}

func NewKernelConnection(kernelId string, clientId string, username string, jupyterServerAddress string, atom *zap.AtomicLevel) (*BasicKernelConnection, error) {
	if len(clientId) == 0 {
		clientId = uuid.NewString()
	}

	conn := &BasicKernelConnection{
		clientId:             clientId,
		kernelId:             kernelId,
		username:             username,
		atom:                 atom,
		jupyterServerAddress: jupyterServerAddress,
		messageCount:         0,
		connectionStatus:     KernelConnectionInit,
		responseChannels:     make(map[string]chan KernelMessage),
		registeredShell:      false,
		registeredControl:    false,
		kernelStdout:         make([]string, 0),
		kernelStderr:         make([]string, 0),
		iopubMessageHandlers: make(map[string]IOPubMessageHandler),
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	conn.logger = logger
	conn.sugaredLogger = logger.Sugar()

	// core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	// conn.logger = zap.New(core, zap.Development())
	// conn.sugaredLogger = conn.logger.Sugar()

	err := conn.setupWebsocket(conn.jupyterServerAddress)
	if err != nil {
		conn.logger.Error("Failed to setup websocket for new kernel.", zap.Error(err))
		return nil, err
	}

	return conn, nil
}

// Stdout returns the slice of stdout messages received by the BasicKernelConnection.
func (conn *BasicKernelConnection) Stdout() []string {
	return conn.kernelStdout
}

// Stderr returns the slice of stderr messages received by the BasicKernelConnection.
func (conn *BasicKernelConnection) Stderr() []string {
	return conn.kernelStderr
}

func (conn *BasicKernelConnection) waitForResponseWithTimeout(responseChan chan KernelMessage, timeoutInterval time.Duration, messageType MessageType) (KernelMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	conn.sugaredLogger.Debugf("Waiting for response to \"%s\" message from kernel %s.", messageType, conn.kernelId)
	select {
	case <-ctx.Done():
		{
			err := ctx.Err()
			conn.sugaredLogger.Errorf("Failed to receive response to \"%s\" message from kernel %s before timing out (interval=%v): %v", messageType, conn.kernelId, timeoutInterval, err)
			return nil, err
		}
	case resp := <-responseChan:
		{
			conn.sugaredLogger.Debugf("Successfully received response to \"%s\" message from kernel %s.", messageType, conn.kernelId)
			return resp, nil
		}
	}
}

// RegisterIoPubHandler registers a handler/consumer of IOPub messages under a specific ID.
func (conn *BasicKernelConnection) RegisterIoPubHandler(id string, handler IOPubMessageHandler) error {
	conn.iopubHandlerMutex.Lock()
	defer conn.iopubHandlerMutex.Unlock()

	if _, ok := conn.iopubMessageHandlers[id]; ok {
		conn.logger.Error("Could not register IOPub message handler.", zap.String("id", id), zap.Error(ErrHandlerAlreadyExists))
		return ErrHandlerAlreadyExists
	}

	conn.iopubMessageHandlers[id] = handler
	conn.logger.Debug("Registered IOPub message handler.", zap.String("id", id))
	return nil
}

// UnregisterIoPubHandler unregisters a handler/consumer of IOPub messages that was registered under the specified ID.
func (conn *BasicKernelConnection) UnregisterIoPubHandler(id string) error {
	conn.iopubHandlerMutex.Lock()
	defer conn.iopubHandlerMutex.Unlock()

	if _, ok := conn.iopubMessageHandlers[id]; !ok {
		conn.logger.Error("Could not unregister IOPub message handler.", zap.String("id", id), zap.Error(ErrNoHandlerFound))
		return ErrNoHandlerFound
	}

	delete(conn.iopubMessageHandlers, id)
	conn.logger.Debug("Unregistered IOPub message handler.", zap.String("id", id))
	return nil
}

func (conn *BasicKernelConnection) SendDummyMessage(channel KernelSocketChannel, content interface{}, waitForResponse bool) (KernelMessage, error) {
	message, responseChan := conn.createKernelMessage(DummyMessage, channel, content)
	err := conn.sendMessage(message)
	if err != nil {
		conn.logger.Error("Error while writing `dummy_message` message.", zap.String("kernel-id", conn.kernelId), zap.Error(err))
		return nil, err
	}

	if waitForResponse {
		return conn.waitForResponseWithTimeout(responseChan, time.Second*5, DummyMessage)
	} else {
		return nil, nil
	}
}

// StopRunningTrainingCode sends a 'stop_running_training_code_request' message.
func (conn *BasicKernelConnection) StopRunningTrainingCode(waitForResponse bool) error {
	message, responseChan := conn.createKernelMessage(StopRunningTrainingCode, ControlChannel, nil)

	err := conn.sendMessage(message)
	if err != nil {
		conn.logger.Error("Error while writing 'stop_running_training_code_request' message.", zap.String("kernel-id", conn.kernelId), zap.Error(err))
		return err
	}

	if waitForResponse {
		_, err := conn.waitForResponseWithTimeout(responseChan, time.Second*10, StopRunningTrainingCode)

		if err != nil {
			conn.logger.Warn("Sending 'dummy' control request to see if we receive a response, seeing as our 'stop_running_training_code_request' request timed-out...", zap.String("kernel-id", conn.kernelId))
			dummyResp, dummyErr := conn.SendDummyMessage(ControlChannel, nil, true)

			if dummyErr != nil {
				conn.logger.Error("'dummy_message' request failed as well (in addition to the failed 'stop_running_training_code_request' request).'", zap.String("kernel-id", conn.kernelId), zap.Error(dummyErr))
			} else {
				conn.logger.Warn("Successfully received response to 'dummy' request.", zap.Any("dummy-response", dummyResp))
			}

			return err // Return the original error.
		}

		// This will be nil if we successfully received a response.
		return err
	}

	return nil
}

// sendAck sends an ACK to the Jupyter Server (and subsequently the Cluster Gateway).
// It returns the address of the Jupyter Server associated with this kernel.
func (conn *BasicKernelConnection) sendAck(msg KernelMessage, channel KernelSocketChannel) error {
	conn.logger.Debug("Attempting to ACK message.", zap.String("message-id", msg.GetHeader().MessageId), zap.String("channel", string(msg.GetChannel())), zap.String("kernel-id", conn.kernelId))

	if channel != ShellChannel && channel != ControlChannel {
		conn.sugaredLogger.Warnf("Cannot ACK message of type \"%s\"...", channel)
	}

	if (channel == ShellChannel && !conn.registeredShell) || (channel == ControlChannel && !conn.registeredControl) {
		conn.sugaredLogger.Warnf("Cannot ACK '%s' '%s' message '%s' as %s channel registration has not yet completed.", channel, msg.GetHeader().MessageType, msg.GetHeader().MessageId, channel)
		return fmt.Errorf("%w: %s", ErrCantAckNotRegistered, channel)
	}

	var content = make(map[string]interface{})
	content["sender-identity"] = fmt.Sprintf("GoJupyter-%s", conn.kernelId)

	ackMessage, _ := conn.createKernelMessage(AckMessage, channel, content)
	ackMessage.(*baseKernelMessage).ParentHeader = msg.GetParentHeader()

	firstPart := fmt.Sprintf(LightBlueStyle.Render("Sending ACK for %v \"%v\""), channel, msg.GetParentHeader().MessageType)
	secondPart := fmt.Sprintf("(MsgId=%v)", LightPurpleStyle.Render(msg.GetParentHeader().MessageId))
	thirdPart := fmt.Sprintf(LightBlueStyle.Render("message: %v"), ackMessage)
	conn.sugaredLogger.Debugf("%s %s %s", firstPart, secondPart, thirdPart)

	err := conn.sendMessage(ackMessage)
	if err != nil {
		conn.logger.Error("Error while writing 'ACK' message.", zap.String("kernel-id", conn.kernelId), zap.Error(err))
		return err
	}

	return nil
}

// JupyterServerAddress returns the address of the Jupyter Server associated with this kernel.
func (conn *BasicKernelConnection) JupyterServerAddress() string {
	return conn.jupyterServerAddress
}

// Connected returns true if the connection is currently active.
func (conn *BasicKernelConnection) Connected() bool {
	return conn.connectionStatus == KernelConnected
}

// ConnectionStatus returns the connection status of the kernel.
func (conn *BasicKernelConnection) ConnectionStatus() KernelConnectionStatus {
	return conn.connectionStatus
}

// KernelId returns the ID of the kernel itself.
func (conn *BasicKernelConnection) KernelId() string {
	return conn.kernelId
}

// RequestExecute sends an `execute_request` message.
//
// #### Notes
// See [Messaging in Jupyter](https://jupyter-client.readthedocs.io/en/latest/messaging.html#execute).
//
// Future `onReply` is called with the `execute_reply` content when the shell reply is received and validated.
// The future will resolve when this message is received and the `idle` iopub status is received.
//
// Arguments:
// - code (string): The code to execute.
// - silent (bool): Whether to execute the code as quietly as possible. The default is `false`.
// - storeHistory (bool): Whether to store history of the execution. The default `true` if silent is False. It is forced to  `false ` if silent is `true`.
// - userExpressions (map[string]interface{}): A mapping of names to expressions to be evaluated in the kernel's interactive namespace.
// - allowStdin (bool): Whether to allow stdin requests. The default is `true`.
// - stopOnError (bool): Whether to the abort execution queue on an error. The default is `false`.
// - waitForResponse (bool): Whether to wait for a response from the kernel, or just return immediately.
func (conn *BasicKernelConnection) RequestExecute(code string, silent bool, storeHistory bool, userExpressions map[string]interface{}, allowStdin bool, stopOnError bool, waitForResponse bool) error {
	content := &executeRequestKernelMessageContent{
		Code:            code,
		Silent:          silent,
		StoreHistory:    storeHistory,
		UserExpressions: userExpressions,
		AllowStdin:      allowStdin,
		StopOnError:     stopOnError,
	}

	message, responseChan := conn.createKernelMessage(ExecuteRequest, ShellChannel, content)

	err := conn.sendMessage(message)
	if err != nil {
		conn.logger.Error("Error while writing 'execute_request' message.", zap.String("kernel-id", conn.kernelId), zap.Error(err))
		return err
	}

	if waitForResponse {
		// Wait without a timeout for the response.
		// Code executions can take an arbitrary amount of time.
		response := <-responseChan
		conn.sugaredLogger.Debugf("Received response to `execute_request` message %s: %v", message.GetHeader().MessageId, response)
	}

	return nil
}

func (conn *BasicKernelConnection) RequestKernelInfo() (KernelMessage, error) {
	content := make(map[string]interface{})
	content["sender-id"] = fmt.Sprintf("GoJupyter-%s", conn.kernelId)

	message, responseChan := conn.createKernelMessage(KernelInfoRequest, ShellChannel, content)

	conn.logger.Debug("Sending 'request-info' message now.", zap.String("message-id", message.GetHeader().MessageId), zap.String("kernel_id", conn.kernelId), zap.String("session", message.GetHeader().Session), zap.String("message", message.String()))

	err := conn.sendMessage(message)
	if err != nil {
		return nil, err
	}

	timeout := time.Second * time.Duration(5)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("ErrRequestTimedOut %w : %s", ErrRequestTimedOut, ctx.Err())
	case resp := <-responseChan:
		{
			conn.logger.Debug("Received response to 'request-info' request.", zap.String("response", resp.String()))
			return resp, nil
		}
	}
}

// InterruptKernel interrupts a kernel.
//
// #### Notes
// Uses the [Jupyter Server API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/jupyter-server/jupyter_server/main/jupyter_server/services/api/api.yaml#!/kernels).
//
// The promise is fulfilled on a valid response and rejected otherwise.
//
// It is assumed that the API call does not mutate the kernel id or name.
//
// The promise will be rejected if the kernel status is `Dead` or if the
// request fails or the response is invalid.
func (conn *BasicKernelConnection) InterruptKernel() error {
	if conn.connectionStatus == KernelDead {
		// Cannot interrupt a dead kernel.
		return ErrKernelIsDead
	}

	conn.logger.Debug("Attempting to Interrupt kernel.", zap.String("kernel_id", conn.kernelId))

	var requestBody = make(map[string]interface{})
	requestBody["kernel_id"] = conn.kernelId

	requestBodyEncoded, err := json.Marshal(requestBody)
	if err != nil {
		conn.logger.Error("Failed to marshal request body for kernel interruption request", zap.Error(err))
		return err
	}

	endpoint := fmt.Sprintf("%s/api/kernels/%s/interrupt", conn.jupyterServerAddress, conn.kernelId)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(requestBodyEncoded))

	if err != nil {
		conn.logger.Error("Failed to create HTTP request for kernel interruption.", zap.String("url", endpoint), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		conn.logger.Error("Error while issuing HTTP request to interrupt kernel.", zap.String("url", endpoint), zap.Error(err))
		return err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		conn.logger.Error("Failed to read response to interrupting kernel.", zap.Error(err))
		return err
	}

	conn.logger.Debug("Received response to interruption request.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("response", data))
	return nil
}

// Close the connection to the kernel.
func (conn *BasicKernelConnection) Close() error {
	message, _ := conn.createKernelMessage(CommCloseMessage, ShellChannel, nil)
	err := conn.sendMessage(message)

	if err != nil {
		conn.logger.Error("Failed to send 'comm_closed' message to kernel.", zap.String("kernel_id", conn.kernelId), zap.String("error-message", err.Error()))
	}

	conn.logger.Warn("Closing WebSocket connection to kernel now.", zap.String("kernel_id", conn.kernelId))
	err = conn.webSocket.Close()

	if err != nil {
		conn.logger.Error("Error while closing WebSocket connection to kernel.", zap.String("kernel_id", conn.kernelId), zap.Error(err))
	}

	return err // Will be nil on success.
}

func (conn *BasicKernelConnection) ClientId() string {
	return conn.clientId
}

func (conn *BasicKernelConnection) Username() string {
	return conn.username
}

// Listen for messages from the kernel.
func (conn *BasicKernelConnection) serveMessages() {
	for {
		var kernelMessage *baseKernelMessage
		conn.rlock.Lock()
		err := conn.webSocket.ReadJSON(&kernelMessage)
		conn.rlock.Unlock()

		if err != nil {
			if errors.Is(err, &websocket.CloseError{}) {
				conn.logger.Warn("Websocket::CloseError.", zap.Error(err))
				return
			}

			conn.logger.Error("Websocket::Read error.", zap.Error(err))

			var rawJsonMap map[string]interface{}
			conn.rlock.Lock()
			err = conn.webSocket.ReadJSON(&rawJsonMap)
			conn.rlock.Unlock()
			if err != nil {
				conn.logger.Error("Websocket::Read error. Failed to unmarshal JSON message into raw key-value map.", zap.Error(err))
			} else {
				conn.logger.Error("Unmarshalled JSON message into raw key-value map.")
				for k, v := range rawJsonMap {
					conn.sugaredLogger.Errorf("%s: %v", k, v)
				}
			}

			continue
		}

		// We send ACKs for Shell and Control messages.
		// We will also attempt to pair the message with its original request.
		if kernelMessage.Channel == ShellChannel || kernelMessage.Channel == ControlChannel {
			conn.sugaredLogger.Debugf("Received %s \"%s\" message '%s' from kernel %s: %v", kernelMessage.Channel, kernelMessage.Header.MessageType, kernelMessage.Header.MessageId, conn.kernelId, kernelMessage)

			// Commented-out; for now, we're not ACK-ing anything.
			// We do this in another goroutine so as not to block this message-receiver goroutine.
			// go conn.sendAck(kernelMessage, kernelMessage.Channel)

			responseChannelKey := getResponseChannelKeyFromReply(kernelMessage)
			if responseChannel, ok := conn.responseChannels[responseChannelKey]; ok {
				conn.logger.Debug("Found response channel for websocket message.", zap.String("request-message-id", kernelMessage.GetParentHeader().MessageId), zap.String("response-message-id", kernelMessage.GetHeader().MessageId), zap.String("message-type", string(kernelMessage.Header.MessageType)), zap.String("channel", kernelMessage.Channel.String()), zap.String("response-channel-key", responseChannelKey), zap.String("kernel-id", conn.kernelId))
				responseChannel <- kernelMessage
				conn.logger.Debug("Response delivered (via channel) for websocket message.", zap.String("request-message-id", kernelMessage.GetParentHeader().MessageId), zap.String("response-message-id", kernelMessage.GetHeader().MessageId))
			} else {
				conn.logger.Warn("Could not find response channel associated with message.", zap.String("request-message-id", kernelMessage.GetParentHeader().MessageId), zap.String("response-message-id", kernelMessage.GetHeader().MessageId), zap.String("message-type", string(kernelMessage.Header.MessageType)), zap.String("channel", kernelMessage.Channel.String()), zap.String("response-channel-key", responseChannelKey), zap.String("kernel-id", conn.kernelId))
			}
		} else {
			// For messages that are not Shell or Control, we do not actually log the message. Too much output. (IOPub messages generate a lot of output.)
			conn.sugaredLogger.Debugf("Received %s \"%s\" message '%s' from kernel %s.", kernelMessage.Channel, kernelMessage.Header.MessageType, kernelMessage.Header.MessageId, conn.kernelId)

			if kernelMessage.Channel == IOPubChannel {
				// TODO: Make it so we can query/view all of the output generated by a Session via the Workload Driver console/frontend.
				conn.handleIOPubMessage(kernelMessage)
			}
		}
	}
}

func (conn *BasicKernelConnection) handleIOPubMessage(kernelMessage KernelMessage) {
	conn.iopubHandlerMutex.Lock()
	defer conn.iopubHandlerMutex.Unlock()

	// If there are no handlers registered, then just invoke the default IOPub message handler.
	if len(conn.iopubMessageHandlers) == 0 {
		go conn.defaultHandleIOPubMessage(kernelMessage)
		return
	}

	// Otherwise, invoke the message handlers.
	for _, handler := range conn.iopubMessageHandlers {
		go handler(conn, kernelMessage)
	}
}

// defaultHandleIOPubMessage provides a default handler for IOPub messages.
// This extracts stream IOPub messages and stores them within the kernel connection struct.
//
// Important: this will be called in its own goroutine.
//
// Also, this function does not match the definition of the 'IOPubMessageHandler' type.
func (conn *BasicKernelConnection) defaultHandleIOPubMessage(kernelMessage KernelMessage) {
	// We just want to extract the output from 'stream' IOPub messages.
	// We don't care about non-stream-type IOPub messages here, so we'll just return.
	if kernelMessage.GetHeader().MessageType != "stream" {
		return
	}

	content := kernelMessage.GetContent().(map[string]interface{})

	var (
		stream string
		text   string
		ok     bool
	)

	stream, ok = content["name"].(string)
	if !ok {
		conn.logger.Warn("Content of IOPub message did not contain an entry with key \"name\" and value of type string.", zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("kernel-id", conn.kernelId))
		return
	}

	text, ok = content["text"].(string)
	if !ok {
		conn.logger.Warn("Content of IOPub message did not contain an entry with key \"text\" and value of type string.", zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("kernel-id", conn.kernelId))
		return
	}

	switch stream {
	case "stdout":
		{
			conn.kernelStdout = append(conn.kernelStdout, text)
		}
	case "stderr":
		{
			conn.kernelStderr = append(conn.kernelStdout, text)
		}
	default:
		conn.logger.Error("Unknown or unsupported stream found in IOPub message.", zap.String("stream", stream), zap.String("kernel-id", conn.kernelId), zap.Any("message", kernelMessage))
	}

	return
}

func (conn *BasicKernelConnection) createKernelMessage(messageType MessageType, channel KernelSocketChannel, content interface{}) (KernelMessage, chan KernelMessage) {
	messageId := conn.getNextMessageId()
	header := &KernelMessageHeader{
		Date:        time.Now().UTC().Format(JavascriptISOString),
		MessageId:   messageId,
		MessageType: messageType,
		Session:     conn.clientId,
		Username:    conn.clientId,
		Version:     VERSION,
	}

	if content == nil {
		content = make(map[string]interface{})
	}

	metadata := make(map[string]interface{})
	metadata["kernel-id"] = conn.kernelId

	message := &baseKernelMessage{
		Channel:      channel,
		Header:       header,
		Content:      content,
		Metadata:     metadata,
		Buffers:      make([]byte, 0),
		ParentHeader: &KernelMessageHeader{},
	}

	var responseChannel chan KernelMessage
	if channel == ShellChannel || channel == ControlChannel {
		// We create a buffered channel so that the 'message-receiver' goroutine cannot get blocked trying to put
		// a result into a response channelfor which the receiver is not actively listening/waiting for said response.
		responseChannel = make(chan KernelMessage, 1)
		responseChannelKey := getResponseChannelKeyFromRequest(message)
		conn.responseChannels[responseChannelKey] = responseChannel

		conn.sugaredLogger.Debugf("Stored response channel for %s \"%s\" message under key \"%s\" for kernel %s.", channel, messageType, responseChannelKey, conn.kernelId)
	}

	return message, responseChannel /* Will be nil for messages that are not either Shell or Control */
}

func (conn *BasicKernelConnection) getNextMessageId() string {
	messageId := fmt.Sprintf("%s_%d_%d", conn.clientId, os.Getpid(), conn.messageCount)
	conn.messageCount += 1
	return messageId
}

func (conn *BasicKernelConnection) updateConnectionStatus(status KernelConnectionStatus) {
	if conn.connectionStatus == status {
		return
	}

	conn.connectionStatus = status

	// Send a kernel info request to make sure we send at least one
	// message to get kernel status back. Always request kernel info
	// first, to get kernel status back and ensure iopub is fully
	// established. If we are restarting, this message will skip the queue
	// and be sent immediately.
	success := false
	maxNumTries := 5
	if conn.connectionStatus == KernelConnected {
		conn.logger.Debug("Connection status is being updated to 'connected'. Attempting to retrieve kernel info.", zap.String("kernel_id", conn.kernelId))
		st := time.Now()

		numTries := 0

		var statusMessage KernelMessage
		var err error

		for numTries <= maxNumTries {
			statusMessage, err = conn.RequestKernelInfo()
			if err != nil {
				numTries += 1
				conn.sugaredLogger.Errorf("Attempt %d/%d to request-info from kernel %s FAILED. Error: %s", numTries, maxNumTries, conn.kernelId, err)
				time.Sleep(time.Duration(1.5*float64(numTries)) * time.Second)
				continue
			} else {
				success = true
				conn.logger.Debug("Successfully retrieved kernel info on connected-status-changed.", zap.String("kernel-info", statusMessage.String()), zap.Duration("time-elapsed", time.Since(st)))
				break
			}
		}

		if !success {
			conn.sugaredLogger.Errorf("Failed to successfully 'request-info' from kernel %s after %d attempts.", conn.kernelId, maxNumTries)
			conn.connectionStatus = KernelDisconnected
		}
	}

	conn.sugaredLogger.Debugf("Kernel %s connection status set to '%s'", conn.kernelId, conn.connectionStatus)
}

// setupWebsocket sets up the WebSocket connection to the Jupyter Server.
// Side-effect: updates the BasicKernelConnection's `webSocket` field.
func (conn *BasicKernelConnection) setupWebsocket(jupyterServerAddress string) error {
	if conn.webSocket != nil {
		return ErrWebsocketAlreadySetup
	}

	conn.updateConnectionStatus(KernelConnecting)

	wsUrl := "ws://" + jupyterServerAddress
	idUrl := url.PathEscape(conn.kernelId)

	partialUrl, err := url.JoinPath(wsUrl, kernelServiceApi, idUrl)
	if err != nil {
		conn.logger.Error("Error when creating partial URL.", zap.String("wsUrl", wsUrl), zap.String("kernelServiceApi", kernelServiceApi), zap.String("idUrl", idUrl), zap.Error(err))
		return fmt.Errorf("ErrWebsocketCreationFailed %w : %s", ErrWebsocketCreationFailed, err.Error())
	}

	conn.sugaredLogger.Debugf("Created partial kernel websocket URL: '%s'", partialUrl)
	endpoint := partialUrl + "/" + fmt.Sprintf("channels?session_id=%s", url.PathEscape(conn.clientId))

	conn.sugaredLogger.Debugf("Created full kernel websocket URL: '%s'", endpoint)

	st := time.Now()

	ws, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		conn.logger.Error("Failed to dial kernel websocket.", zap.String("endpoint", endpoint), zap.String("kernel-id", conn.kernelId), zap.Error(err))
		return fmt.Errorf("ErrWebsocketCreationFailed %w : %s", ErrWebsocketCreationFailed, err.Error())
	}

	conn.logger.Debug("Successfully connected to the kernel.", zap.Duration("time-taken-to-connect", time.Since(st)), zap.String("kernel-id", conn.kernelId))
	conn.webSocket = ws

	go conn.serveMessages()

	// Set up the close handler, which automatically tries to reconnect.
	if conn.originalWebsocketCloseHandler == nil {
		handler := conn.webSocket.CloseHandler()
		conn.originalWebsocketCloseHandler = handler
	}
	conn.webSocket.SetCloseHandler(conn.websocketClosed)

	conn.updateConnectionStatus(KernelConnected)

	// Skip for now... we may or may not need this.
	// The registration idea was so we could figure out a way to add support for ACKs between the Cluster Gateway and the Golang Jupyter frontends.
	// conn.registerAsGolangFrontend()

	return nil
}

// Register with the Cluster Gateway, informing it that we're a Golang frontend and that it should expect us to ACK messages.
// This is noteworthy insofar as Jupyter frontends typically do not ACK messages.
// We don't want to lose any messages, though, so we tell the Cluster Gateway that we WILL be ACKing messages.
// func (conn *BasicKernelConnection) registerAsGolangFrontend() error {
// 	content := make(map[string]interface{}, 0)
// 	content["sender-id"] = fmt.Sprintf("GoJupyter-%s", conn.kernelId)

// 	send_golang_frontend_registration_msg := func(message KernelMessage, responseChan chan KernelMessage) error {
// 		conn.logger.Debug("Sending 'golang_frontend_registration_request' message now.", zap.String("message-id", message.GetHeader().MessageId), zap.String("kernel_id", conn.kernelId), zap.String("session", message.GetHeader().Session), zap.String("message", message.String()))

// 		return conn.sendMessage(message)
// 		// err := conn.sendMessage(message)
// 		// if err != nil {
// 		// 	return err
// 		// }

// 		// timeout := time.Second * time.Duration(10)
// 		// ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 		// defer cancel()

// 		// select {
// 		// case <-ctx.Done():
// 		// 	return fmt.Errorf("%w : %s", ErrRequestTimedOut, ctx.Err())
// 		// case resp := <-responseChan:
// 		// 	{
// 		// 		conn.logger.Debug("Received response to 'golang_frontend_registration_request' request.", zap.String("response", resp.String()))
// 		// 		return nil
// 		// 	}
// 		// }
// 	}

// 	shellMessage, shellResponseChan := conn.createKernelMessage(GolangFrontendRegistrationRequest, ShellChannel, content)
// 	if err := send_golang_frontend_registration_msg(shellMessage, shellResponseChan); err != nil {
// 		conn.logger.Error("Failed to send shell 'golang_frontend_registration_request' message.", zap.Error(err))
// 		return err
// 	} else {
// 		conn.registeredShell = true
// 	}

// 	controlMessage, controlResponseChan := conn.createKernelMessage(GolangFrontendRegistrationRequest, ControlChannel, content)
// 	if err := send_golang_frontend_registration_msg(controlMessage, controlResponseChan); err != nil {
// 		conn.logger.Error("Failed to send control 'golang_frontend_registration_request' message.", zap.Error(err))
// 		return err
// 	} else {
// 		conn.registeredControl = true
// 	}

// 	return nil
// }

func (conn *BasicKernelConnection) websocketClosed(code int, text string) error {
	if conn.originalWebsocketCloseHandler == nil {
		panic("Original websocket close-handler is not set.")
	}

	conn.sugaredLogger.Warnf("WebSocket::Closed handler called for kernel %s.", conn.kernelId)

	// Try to get the model.
	model, err := conn.getKernelModel()
	if err != nil {
		if errors.Is(err, ErrNetworkIssue) && conn.reconnect() {
			// If it was a network error, and we were able to reconnect, then exit the 'websocket closed' handler.
			return nil
		}

		// If it was not a network error, or it was, but we failed to reconnect, then call the original 'websocket closed' handler.
		conn.updateConnectionStatus(KernelDead)
		return conn.originalWebsocketCloseHandler(code, text)
	}

	// If we get the model and the execution state is dead, then we terminate.
	// If we get the model and the execution state is NOT dead, then we try to reconnect.
	conn.model = model
	if model.ExecutionState == string(KernelDead) {
		// Kernel is dead. Call the original 'websocket closed' handler.
		conn.updateConnectionStatus(KernelDead)
		return conn.originalWebsocketCloseHandler(code, text)
	} else {
		success := conn.reconnect()

		// If we reconnected, then just return. If we failed to reconnect, call the original 'websocket closed' handler.
		if success {
			return nil
		} else {
			return conn.originalWebsocketCloseHandler(code, text)
		}
	}
}

func (conn *BasicKernelConnection) reconnect() bool {
	numTries := 0
	maxNumTries := 5

	for numTries < maxNumTries {
		err := conn.setupWebsocket(conn.jupyterServerAddress)
		if err != nil {
			if errors.Is(err, ErrNetworkIssue) && (numTries+1) <= maxNumTries {
				numTries += 1
				sleepInterval := time.Second * time.Duration(2*numTries)
				conn.logger.Error("Network error encountered while trying to reconnect to kernel.", zap.String("kernel-id", conn.kernelId), zap.Error(err), zap.Duration("next-sleep-interval", sleepInterval))
				conn.updateConnectionStatus(KernelDisconnected)
				time.Sleep(sleepInterval)
				continue
			}

			conn.logger.Error("Connection to kernel is dead.", zap.String("kernel-id", conn.kernelId))
			conn.updateConnectionStatus(KernelDead)
			return false
		} else {
			return true
		}
	}

	return false
}

func (conn *BasicKernelConnection) getKernelModel() (*jupyterKernel, error) {
	conn.logger.Debug("Retrieving kernel model via HTTP Rest API.", zap.String("kernel-id", conn.kernelId))

	endpoint := fmt.Sprintf("http://%s/api/kernels/%s", conn.jupyterServerAddress, conn.kernelId)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		conn.logger.Error("Error encountered while creating HTTP request to get model for kernel.", zap.String("kernel-id", conn.kernelId), zap.String("endpoint", endpoint), zap.Error(err))
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		conn.logger.Error("Received error while requesting model for kernel.", zap.String("kernel-id", conn.kernelId), zap.String("endpoint", endpoint), zap.Error(err))
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		conn.logger.Error("Received HTTP 404 when retrieving model for kernel.", zap.String("kernel-id", conn.kernelId))
		return nil, ErrKernelNotFound
	} else if resp.StatusCode == http.StatusServiceUnavailable /* 503 */ || resp.StatusCode == http.StatusFailedDependency /* 424 */ {
		// Network errors. We should retry.
		return nil, ErrNetworkIssue
	} else if resp.StatusCode != http.StatusOK {
		conn.logger.Error("Kernel died unexpectedly.", zap.String("kernel-id", conn.kernelId), zap.Int("http-status-code", resp.StatusCode), zap.String("http-status", resp.Status))
		conn.updateConnectionStatus(KernelDead)

		return nil, fmt.Errorf("ErrUnexpectedFailure %w : HTTP %d -- %s", ErrUnexpectedFailure, resp.StatusCode, resp.Status)
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var model *jupyterKernel

	err = json.Unmarshal(body, &model)
	if err != nil {
		conn.logger.Error("Failed to unmarshal JSON response when requesting model for new kernel.", zap.String("kernel-id", conn.kernelId), zap.String("endpoint", endpoint), zap.Error(err))
		return nil, err
	}

	conn.logger.Debug("Successfully retrieved model for kernel.", zap.String("model", model.String()))
	return model, nil
}

func (conn *BasicKernelConnection) sendMessage(message KernelMessage) error {
	if conn.connectionStatus == KernelDead {
		conn.logger.Error("Cannot send message. Kernel is dead.", zap.String("kernel_id", conn.kernelId))
		return ErrKernelIsDead
	}

	if conn.connectionStatus == KernelConnected {
		conn.sugaredLogger.Debugf("Writing %s message (ID=%s) of type '%s' now to kernel %s.", message.GetChannel(), message.GetHeader().MessageId, message.GetHeader().MessageType, conn.kernelId)
		conn.wlock.Lock()
		err := conn.webSocket.WriteJSON(message)
		conn.wlock.Unlock()
		if err != nil {
			conn.sugaredLogger.Errorf("Error while writing %s message (ID=%s) of type '%s' now to kernel %s. Error: %v", message.GetChannel(), message.GetHeader().MessageId, message.GetHeader().MessageType, conn.kernelId, zap.Error(err))
			return err
		}
	} else {
		conn.sugaredLogger.Errorf("Could not send %s message (ID=%s) of type '%s' now to kernel %s. Kernel is not connected.", message.GetChannel(), message.GetHeader().MessageId, message.GetHeader().MessageType, conn.kernelId)
		return ErrNotConnected
	}

	conn.sugaredLogger.Debugf("Successfully sent %s message %s of type %s to kernel %s.", message.GetChannel(), message.GetHeader().MessageId, message.GetHeader().MessageType, conn.kernelId)
	return nil
}
