package jupyter

import (
	"encoding/json"
	"errors"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrAlreadyConnectedToKernel = errors.New("session is already connected to its kernel")
)

type jupyterSession struct {
	LocalSessionId   string           `json:"-"`
	JupyterSessionId string           `json:"id"`
	Path             string           `json:"path"`
	Name             string           `json:"name"`
	SessionType      string           `json:"type"`
	JupyterKernel    *jupyterKernel   `json:"kernel"`
	JupyterNotebook  *jupyterNotebook `json:"notebook"`

	SessionConnection *SessionConnection `json:"-"`
}

func newJupyterSessionForRequest(sessionName string, path string, sessionType string, kernelSpecName string) *jupyterSession {
	jupyterKernel := newJupyterKernel("", kernelSpecName)

	return &jupyterSession{
		JupyterSessionId: "",
		Path:             path,
		Name:             sessionName,
		SessionType:      sessionType,
		JupyterKernel:    jupyterKernel,
	}
}

func (s *jupyterSession) String() string {
	out, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type jupyterKernel struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	LastActivity   string `json:"last_activity"`
	ExecutionState string `json:"execution_state"`
	Connections    int    `json:"connections"`
}

func newJupyterKernel(id string, name string) *jupyterKernel {
	return &jupyterKernel{
		Id:   id,
		Name: name,
	}
}

type jupyterNotebook struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type Session struct {
	id        string // The ID of the session that the user/trace data supplied.
	jupyterId string // The session ID generated by Jupyter.
	kernelId  string // The ID of the kernel associated with the Session.
}

func NewSession(id string, jupyterId string, kernelId string) *Session {
	sess := &Session{
		id:        id,
		jupyterId: jupyterId,
		kernelId:  kernelId,
	}

	return sess
}

type SessionConnection struct {
	model  *jupyterSession
	kernel *KernelConnection

	jupyterServerAddress string

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel
}

func NewSessionConnection(model *jupyterSession, jupyterServerAddress string, atom *zap.AtomicLevel) *SessionConnection {
	conn := &SessionConnection{
		model:                model,
		jupyterServerAddress: jupyterServerAddress,
		atom:                 atom,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	conn.logger = zap.New(core, zap.Development())

	conn.sugaredLogger = conn.logger.Sugar()

	err := conn.connectToKernel()
	if err != nil {
		panic(err)
	}

	conn.logger.Debug("Successfully connected to kernel.", zap.String("kernel-id", model.JupyterKernel.Id))

	return conn
}

// Side-effect: set the `kernel` field of the SessionConnection.
func (conn *SessionConnection) connectToKernel() error {
	if conn.kernel != nil {
		return ErrAlreadyConnectedToKernel
	}

	conn.kernel = NewKernelConnection(conn.model.JupyterKernel, conn.jupyterServerAddress, conn.atom)

	return nil
}
