package jupyter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	kernelServiceApi = "api/kernels"
)

var (
	ErrWebsocketAlreadySetup = errors.New("the kernel connection's websocket has already been setup")
)

type KernelConnection struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	clientId  string
	webSocket *websocket.Conn
	model     *jupyterKernel
}

func NewKernelConnection(model *jupyterKernel, jupyterServerAddress string, atom *zap.AtomicLevel) *KernelConnection {
	conn := &KernelConnection{
		clientId: uuid.NewString(),
		model:    model,
		atom:     atom,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	conn.logger = zap.New(core, zap.Development())

	conn.sugaredLogger = conn.logger.Sugar()

	err := conn.setupWebsocket(jupyterServerAddress)
	if err != nil {
		panic(err)
	}

	go conn.serveMessages()

	return conn
}

// Side-effect: updates the KernelConnection's `webSocket` field.
func (conn *KernelConnection) setupWebsocket(jupyterServerAddress string) error {
	if conn.webSocket != nil {
		return ErrWebsocketAlreadySetup
	}

	wsUrl := "ws://" + jupyterServerAddress
	idUrl := url.PathEscape(conn.model.Id)

	partialUrl, err := url.JoinPath(wsUrl, kernelServiceApi, idUrl)
	if err != nil {
		conn.logger.Error("Error when creating partial URL.", zap.String("wsUrl", wsUrl), zap.String("kernelServiceApi", kernelServiceApi), zap.String("idUrl", idUrl), zap.Error(err))
		return err
	}

	conn.sugaredLogger.Debugf("Created partial kernel websocket URL: '%s'", partialUrl)
	url, err := url.JoinPath(
		partialUrl,
		fmt.Sprintf("channels?session_id=%s", url.PathEscape(conn.clientId)))
	if err != nil {
		conn.logger.Error("Error when creating full URL.", zap.String("partialUrl", partialUrl), zap.String("wsUrl", wsUrl), zap.String("kernelServiceApi", kernelServiceApi), zap.String("idUrl", idUrl), zap.Error(err))
		return err
	}

	conn.sugaredLogger.Debugf("Created full kernel websocket URL: '%s'", url)

	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		conn.logger.Error("Failed to dial kernel websocket.", zap.String("url", url), zap.Error(err))
		return err
	}

	conn.webSocket = ws
	return nil
}

func (conn *KernelConnection) serveMessages() {
	for {
		_, message, err := conn.webSocket.ReadMessage()
		if err != nil {
			conn.logger.Error("Websocket::Read error.", zap.Error(err))
		}

		var kernelMessage *KernelMessage
		err = json.Unmarshal(message, &kernelMessage)
		if err != nil {
			conn.logger.Error("Error while unmarshalling message from kernel.", zap.Any("message", message), zap.Error(err))
		}

		conn.logger.Debug("Received message from kernel.", zap.Any("message", kernelMessage.String()))
	}
}
