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
	ErrRequestTimedOut          = errors.New("the request timed out")
)

// The difference between this and `jupyterSession` is that this struct has a different type for the `JupyterNotebook` field so that it isn't null in the request.
type jupyterSessionReq struct {
	LocalSessionId   string                 `json:"-"`
	JupyterSessionId string                 `json:"id"`
	Path             string                 `json:"path"`
	Name             string                 `json:"name"`
	SessionType      string                 `json:"type"`
	JupyterKernel    *jupyterKernel         `json:"kernel"`
	JupyterNotebook  map[string]interface{} `json:"notebook"`

	SessionConnection *SessionConnection `json:"-"`
}

func newJupyterSessionForRequest(sessionName string, path string, sessionType string, kernelSpecName string) *jupyterSessionReq {
	jupyterKernel := newJupyterKernel(sessionName, kernelSpecName)

	return &jupyterSessionReq{
		JupyterSessionId: sessionName,
		LocalSessionId:   sessionName,
		Path:             path,
		Name:             sessionName,
		SessionType:      sessionType,
		JupyterKernel:    jupyterKernel,
		JupyterNotebook:  make(map[string]interface{}),
	}
}

func (s *jupyterSessionReq) String() string {
	out, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	return string(out)
}

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

func (k *jupyterKernel) String() string {
	out, err := json.Marshal(k)
	if err != nil {
		panic(err)
	}

	return string(out)
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
	kernel KernelConnection

	jupyterServerAddress string

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel
}

// NewSessionConnection creates a new SessionConnection.
//
// We do not return until we've successfully connected to the kernel.
func NewSessionConnection(model *jupyterSession, username string, jupyterServerAddress string, atom *zap.AtomicLevel) (*SessionConnection, error) {
	conn := &SessionConnection{
		model:                model,
		jupyterServerAddress: jupyterServerAddress,
		atom:                 atom,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	conn.logger = zap.New(core, zap.Development())

	conn.sugaredLogger = conn.logger.Sugar()

	err := conn.connectToKernel(username)
	if err != nil && !errors.Is(err, ErrAlreadyConnectedToKernel) {
		return nil, err
	}

	conn.logger.Debug("Successfully connected to kernel (as part of a SessionConnection).", zap.String("kernel-id", model.JupyterKernel.Id), zap.String("session-id", model.JupyterSessionId))

	return conn, err
}

// connectToKernel creates a new WebSocket-backed connection to the kernel associated with this session.
// Side-effect: set the `kernel` field of the SessionConnection.
func (conn *SessionConnection) connectToKernel(username string) error {
	if conn.kernel != nil {
		return ErrAlreadyConnectedToKernel
	}

	var err error
	conn.kernel, err = NewKernelConnection(conn.model.JupyterKernel.Id, conn.model.JupyterSessionId, username, conn.jupyterServerAddress, conn.atom)
	return err // Will be nil if everything went OK.
}

func (conn *SessionConnection) RegisterIoPubHandler(id string, handler IOPubMessageHandler) error {
	return conn.kernel.RegisterIoPubHandler(id, handler)
}

// UnregisterIoPubHandler unregisters a handler/consumer of IOPub messages that was registered under the specified ID.
func (conn *SessionConnection) UnregisterIoPubHandler(id string) error {
	return conn.kernel.UnregisterIoPubHandler(id)
}

func (conn *SessionConnection) Kernel() KernelConnection {
	return conn.kernel
}
