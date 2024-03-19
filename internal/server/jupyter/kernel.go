package jupyter

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	JavascriptISOString = "2006-01-02T15:04:05.999Z07:00"
	kernelServiceApi    = "api/kernels"
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

func NewKernelConnection(model *jupyterKernel, clientId string, jupyterServerAddress string, atom *zap.AtomicLevel) *KernelConnection {
	conn := &KernelConnection{
		clientId: clientId,
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
	url := partialUrl + "/" + fmt.Sprintf("channels?session_id=%s", url.PathEscape(conn.clientId))

	conn.sugaredLogger.Debugf("Created full kernel websocket URL: '%s'", url)

	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		conn.logger.Error("Failed to dial kernel websocket.", zap.String("url", url), zap.Error(err))
		return err
	}

	conn.webSocket = ws
	return nil
}

func (conn *KernelConnection) requestKernelInfo(sessionId string, username string, messageId string) error {
	var message *KernelMessage = &KernelMessage{
		Header: &KernelMessageHeader{
			Date:        time.Now().UTC().Format(JavascriptISOString),
			MessageId:   messageId,
			MessageType: "kernel_info_request",
			Session:     sessionId,
			Username:    username,
			Version:     "5.2",
		},
		Content:  make(map[string]interface{}),
		Metadata: make(map[string]interface{}),
	}

	conn.logger.Debug("Writing message of type `kernel_info_request` now.", zap.String("message-id", messageId))
	err := conn.webSocket.WriteJSON(message)
	if err != nil {
		conn.logger.Error("Error while writing 'RequestKernelInfo' message.", zap.Error(err))
		return err
	}

	return nil
}
