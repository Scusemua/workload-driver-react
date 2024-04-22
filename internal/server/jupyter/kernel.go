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
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	JavascriptISOString = "2006-01-02T15:04:05.999Z07:00"
	kernelServiceApi    = "api/kernels"

	KernelConnectionInit KernelConnectionStatus = "initializing" // When the KernelConnection struct is first created.
	KernelConnecting     KernelConnectionStatus = "connecting"   // When we are creating the kernel websocket.
	KernelConnected      KernelConnectionStatus = "connected"    // Once we've connected.
	KernelDisconnected   KernelConnectionStatus = "disconnected" // We're not connected to the kernel, but we're unsure if it is dead or not.
	KernelDead           KernelConnectionStatus = "dead"         // Kernel is dead. We're not connected.
)

var (
	ErrWebsocketAlreadySetup   = errors.New("the kernel connection's websocket has already been setup")
	ErrWebsocketCreationFailed = errors.New("creation of websocket connection to kernel has failed")
	ErrKernelNotFound          = errors.New("received HTTP 404 status when requesting info for kernel")
	ErrNetworkIssue            = errors.New("received HTTP 503 or HTTP 424 in response to request")
	ErrUnexpectedFailure       = errors.New("the request could not be completed for some unexpected reason")
	ErrKernelIsDead            = errors.New("kernel is dead")
)

type KernelConnectionStatus string

type KernelConnection struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	// Register callbacks for responses to particular messages.
	responseChannels map[string]chan KernelMessage

	// How many messages we've sent. Used when creating message IDs.
	messageCount int

	connectionStatus KernelConnectionStatus

	kernelId                      string
	jupyterServerAddress          string
	jupyterSessionId              string
	clientId                      string
	webSocket                     *websocket.Conn
	originalWebsocketCloseHandler func(int, string) error
	model                         *jupyterKernel
}

func NewKernelConnection(kernelId string, jupyterSessionId string, atom *zap.AtomicLevel, jupyterServerAddress string) (*KernelConnection, error) {
	conn := &KernelConnection{
		clientId:             jupyterSessionId,
		jupyterSessionId:     jupyterSessionId,
		kernelId:             kernelId,
		atom:                 atom,
		jupyterServerAddress: jupyterServerAddress,
		messageCount:         0,
		connectionStatus:     KernelConnectionInit,
		responseChannels:     make(map[string]chan KernelMessage),
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	conn.logger = zap.New(core, zap.Development())

	conn.sugaredLogger = conn.logger.Sugar()

	err := conn.setupWebsocket(conn.jupyterServerAddress)
	if err != nil {
		conn.logger.Error("Failed to setup websocket for new kernel.", zap.Error(err))
		return nil, err
	}

	return conn, nil
}

// Send an `execute_request` message.
//
// #### Notes
// See [Messaging in Jupyter](https://jupyter-client.readthedocs.io/en/latest/messaging.html#execute).
//
// Future `onReply` is called with the `execute_reply` content when the
// shell reply is received and validated. The future will resolve when
// this message is received and the `idle` iopub status is received.
//
// Arguments:
// - code (string): The code to execute.
// - silent (bool): Whether to execute the code as quietly as possible. The default is `false`.
// - storeHistory (bool): Whether to store history of the execution. The default `true` if silent is False. It is forced to  `false ` if silent is `true`.
// - userExpressions (map[string]interface{}): A mapping of names to expressions to be evaluated in the kernel's interactive namespace.
// - allowStdin (bool): Whether to allow stdin requests. The default is `true`.
// - stopOnError (bool): Whether to the abort execution queue on an error. The default is `false`.
func (conn *KernelConnection) RequestExecute(code string, silent bool, storeHistory bool, userExpressions map[string]interface{}, allowStdin bool, stopOnError bool) error {
	content := &executeRequestKernelMessageContent{
		Code:            code,
		Silent:          silent,
		StoreHistory:    storeHistory,
		UserExpressions: userExpressions,
		AllowStdin:      allowStdin,
		StopOnError:     stopOnError,
	}

	header := &KernelMessageHeader{
		Date:        time.Now().Format(time.RFC3339),
		MessageId:   uuid.NewString(),
		MessageType: "execute_request",
		Session:     conn.kernelId,
		Username:    conn.clientId,
		Version:     VERSION,
	}

	message := &executeRequestKernelMessage{
		Header:       header,
		Channel:      "shell",
		Content:      content,
		Buffers:      make([]byte, 0),
		Metadata:     make(map[string]interface{}),
		ParentHeader: nil,
	}

	err := conn.sendMessage(message)
	if err != nil {
		conn.logger.Error("Error while writing 'execute_request' message.", zap.String("kernel-id", conn.kernelId), zap.Error(err))
		return err
	}

	return nil
}

func (conn *KernelConnection) ConnectionStatus() KernelConnectionStatus {
	return conn.connectionStatus
}

func (conn *KernelConnection) RequestKernelInfo() (KernelMessage, error) {
	var message KernelMessage = conn.createKernelMessage("kernel_info_request", conn.jupyterSessionId, "", ShellChannel)
	var messageId string = message.GetHeader().MessageId
	responseChan := make(chan KernelMessage)
	conn.responseChannels[messageId] = responseChan

	conn.logger.Debug("Sending 'request-info' message now.", zap.String("message", message.String()))

	err := conn.sendMessage(message)
	if err != nil {
		return nil, err
	}

	timeout := time.Second * time.Duration(5)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		time.Sleep(timeout)
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ErrRequestTimedOut %w : %s", ErrRequestTimedOut, ctx.Err())
		case resp := <-responseChan:
			{
				conn.logger.Debug("Received response to 'request-info' request.", zap.String("response", resp.String()))
				return resp, nil
			}
		default:
			{
				time.Sleep(time.Millisecond * 250)
			}
		}
	}
}

// Interrupt a kernel.
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
func (conn *KernelConnection) InterruptKernel() error {
	if conn.connectionStatus == KernelDead {
		// Cannot interrupt a dead kernel.
		return ErrKernelIsDead
	}

	var requestBody map[string]interface{} = make(map[string]interface{})
	requestBody["kernel_id"] = conn.kernelId

	requestBodyEncoded, err := json.Marshal(requestBody)
	if err != nil {
		conn.logger.Error("Failed to marshal request body for kernel interruption request", zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s/api/kernels/%s/interrupt", conn.jupyterServerAddress, conn.kernelId)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBodyEncoded))

	if err != nil {
		conn.logger.Error("Failed to create HTTP request for kernel interruption.", zap.String("url", url), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		conn.logger.Error("Error while issuing HTTP request to interrupt kernel.", zap.String("url", url), zap.Error(err))
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

// Listen for messages from the kernel.
func (conn *KernelConnection) serveMessages() {
	for {
		var kernelMessage KernelMessage
		err := conn.webSocket.ReadJSON(&kernelMessage)

		if err != nil {
			conn.logger.Error("Websocket::Read error.", zap.Error(err))

			if errors.Is(err, &websocket.CloseError{}) {
				return
			}

			continue
		}

		conn.logger.Debug("Received message from kernel.", zap.Any("message", kernelMessage.String()))

		if responseChannel, ok := conn.responseChannels[kernelMessage.GetParentHeader().MessageId]; ok {
			conn.logger.Debug("Found response channel for message.", zap.String("message-id", kernelMessage.GetParentHeader().MessageId))
			responseChannel <- kernelMessage
		} else {
			conn.logger.Debug("Could not find response channel for message.", zap.String("message-id", kernelMessage.GetParentHeader().MessageId))
		}
	}
}

func (conn *KernelConnection) createKernelMessage(messageType string, sessionId string, username string, channel KernelSocketChannel) KernelMessage {
	header := &KernelMessageHeader{
		Date:        time.Now().UTC().Format(JavascriptISOString),
		MessageId:   conn.getNextMessageId(),
		MessageType: messageType,
		Session:     sessionId,
		Username:    username,
		Version:     VERSION,
	}

	message := &baseKernelMessage{
		Channel:      channel,
		Header:       header,
		Content:      make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		Buffers:      make([]byte, 0),
		ParentHeader: &KernelMessageHeader{},
	}

	return message
}

func (conn *KernelConnection) getNextMessageId() string {
	messageId := fmt.Sprintf("%s_%d_%d", conn.jupyterSessionId, os.Getpid(), conn.messageCount)
	conn.messageCount += 1
	return messageId
}

func (conn *KernelConnection) updateConnectionStatus(status KernelConnectionStatus) {
	if conn.connectionStatus == status {
		return
	}

	conn.connectionStatus = status

	// Send a kernel info request to make sure we send at least one
	// message to get kernel status back. Always request kernel info
	// first, to get kernel status back and ensure iopub is fully
	// established. If we are restarting, this message will skip the queue
	// and be sent immediately.
	if conn.connectionStatus == KernelConnected {
		st := time.Now()

		num_tries := 0
		max_num_tries := 3

		var statusMessage KernelMessage
		var err error

		for num_tries <= max_num_tries {
			time.Sleep(time.Duration(1.5*float64(num_tries)) * time.Second) // Will be 0 initially.
			statusMessage, err = conn.RequestKernelInfo()
			if err != nil {
				num_tries += 1
				continue
			} else {
				break
			}
		}

		if err != nil {
			conn.logger.Error("We've supposedly connected, but the 'request-info' request FAILED.", zap.Error(err), zap.Duration("time-elapsed", time.Since(st)))
		} else {
			conn.logger.Debug("Successfully retrieved kernel info on connected-status-changed.", zap.String("kernel-info", statusMessage.String()), zap.Duration("time-elapsed", time.Since(st)))
		}
	}
}

// Side-effect: updates the KernelConnection's `webSocket` field.
func (conn *KernelConnection) setupWebsocket(jupyterServerAddress string) error {
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
	url := partialUrl + "/" + fmt.Sprintf("channels?session_id=%s", url.PathEscape(conn.clientId))

	conn.sugaredLogger.Debugf("Created full kernel websocket URL: '%s'", url)

	// ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel()
	st := time.Now()
	// ws, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		conn.logger.Error("Failed to dial kernel websocket.", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("ErrWebsocketCreationFailed %w : %s", ErrWebsocketCreationFailed, err.Error())
	}

	conn.logger.Debug("Successfully connected to the kernel.", zap.Duration("time-taken-to-connect", time.Since(st)), zap.String("kernel-id", conn.kernelId))
	conn.webSocket = ws

	go conn.serveMessages()

	// Setup the close handler, which automatically tries to reconnect.
	if conn.originalWebsocketCloseHandler == nil {
		handler := conn.webSocket.CloseHandler()
		conn.originalWebsocketCloseHandler = handler
	}
	conn.webSocket.SetCloseHandler(conn.websocketClosed)
	conn.model, err = conn.getKernelModel()

	if err != nil {
		if errors.Is(err, ErrNetworkIssue) {
			conn.updateConnectionStatus(KernelDisconnected)
			return fmt.Errorf("ErrWebsocketCreationFailed %w : %s", ErrWebsocketCreationFailed, err.Error())
		} else {
			conn.updateConnectionStatus(KernelDead)
			return fmt.Errorf("ErrWebsocketCreationFailed %w : %s", ErrWebsocketCreationFailed, err.Error())
		}
	}

	conn.updateConnectionStatus(KernelConnected)
	return nil
}

func (conn *KernelConnection) websocketClosed(code int, text string) error {
	if conn.originalWebsocketCloseHandler == nil {
		panic("Original websocket close-handler is not set.")
	}

	// Try to get the model.
	model, err := conn.getKernelModel()
	if err != nil {
		if errors.Is(err, ErrNetworkIssue) && conn.reconnect() {
			// If it was a network error and we were able to reconnect, then exit the 'websocket closed' handler.
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

func (conn *KernelConnection) reconnect() bool {
	num_tries := 0
	max_num_tries := 5

	for num_tries < max_num_tries {
		err := conn.setupWebsocket(conn.jupyterServerAddress)
		if err != nil {
			if errors.Is(err, ErrNetworkIssue) && (num_tries+1) <= max_num_tries {
				num_tries += 1
				sleepInterval := time.Second * time.Duration(2*num_tries)
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

func (conn *KernelConnection) getKernelModel() (*jupyterKernel, error) {
	url := fmt.Sprintf("http://%s/api/kernels/%s", conn.jupyterServerAddress, conn.kernelId)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		conn.logger.Error("Error encountered while creating HTTP request to get model for kernel.", zap.String("kernel-id", conn.kernelId), zap.String("url", url), zap.Error(err))
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		conn.logger.Error("Received error while requesting model for kernel.", zap.String("kernel-id", conn.kernelId), zap.String("url", url), zap.Error(err))
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
		conn.logger.Error("Failed to unmarshal JSON response when requesting model for new kernel.", zap.String("kernel-id", conn.kernelId), zap.String("url", url), zap.Error(err))
		return nil, err
	}

	conn.logger.Debug("Successfully retrieved model for kernel.", zap.String("model", model.String()))
	return model, nil
}

func (conn *KernelConnection) sendMessage(message KernelMessage) error {
	conn.logger.Debug("Writing message of type `kernel_info_request` now.", zap.String("message-id", message.GetHeader().MessageId))
	err := conn.webSocket.WriteJSON(message)
	if err != nil {
		conn.logger.Error("Error while writing 'request-info' message.", zap.Error(err))
		return err
	}

	return nil
}
