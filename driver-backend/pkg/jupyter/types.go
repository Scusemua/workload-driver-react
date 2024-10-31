package jupyter

import (
	"errors"
	"time"
)

var (
	ErrNoHandlerFound       = errors.New("no handler found registered under the specified ID")
	ErrHandlerAlreadyExists = errors.New("there is already a handler registered under the specified ID")
)

// IOPubMessageHandler defines a message handler for IOPub messages sent by a Jupyter kernel to us.
// Important: an IOPubMessageHandler must be thread-safe insofar as it will be called from its own goroutine.
//
// It can return an arbitrary value.
type IOPubMessageHandler func(conn KernelConnection, kernelMessage KernelMessage) interface{}

type KernelConnection interface {
	// ConnectionStatus returns the connection status of the kernel.
	ConnectionStatus() KernelConnectionStatus

	// Connected returns true if the connection is currently active.
	Connected() bool

	// KernelId returns the ID of the kernel itself.
	KernelId() string

	ClientId() string

	Username() string

	// Stdout returns the slice of stdout messages received by the BasicKernelConnection.
	Stdout() []string

	// Stderr returns the slice of stderr messages received by the BasicKernelConnection.
	Stderr() []string

	// JupyterServerAddress returns the address of the Jupyter Server associated with this kernel.
	JupyterServerAddress() string

	// RequestExecute sends an `execute_request` message.
	//
	// #### Notes
	// See [Messaging in Jupyter](https://jupyter-client.readthedocs.io/en/latest/messaging.html#execute).
	//
	// Future `onReply` is called with the `execute_reply` content when the shell reply is received and validated.
	// The future will resolve when this message is received and the `idle` IOPub status is received.
	//
	// Arguments:
	// - code (string): The code to execute.
	// - silent (bool): Whether to execute the code as quietly as possible. The default is `false`.
	// - storeHistory (bool): Whether to store history of the execution. The default `true` if silent is False. It is forced to  `false ` if silent is `true`.
	// - userExpressions (map[string]interface{}): A mapping of names to expressions to be evaluated in the kernel's interactive namespace.
	// - allowStdin (bool): Whether to allow stdin requests. The default is `true`.
	// - stopOnError (bool): Whether to the abort execution queue on an error. The default is `false`.
	// - waitForResponse (bool): Whether to wait for a response from the kernel, or just return immediately.
	RequestExecute(code string, silent bool, storeHistory bool, userExpressions map[string]interface{}, allowStdin bool, stopOnError bool, waitForResponse bool) error

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
	InterruptKernel() error

	// StopRunningTrainingCode sends a 'stop_running_training_code_request' message.
	StopRunningTrainingCode(waitForResponse bool) error

	// Close the connection to the kernel.
	Close() error

	SetOnError(func(err error))

	// RegisterIoPubHandler registers a handler/consumer of IOPub messages under a specific ID.
	RegisterIoPubHandler(id string, handler IOPubMessageHandler) error

	// UnregisterIoPubHandler unregisters a handler/consumer of IOPub messages that was registered under the specified ID.
	UnregisterIoPubHandler(id string) error

	// AddMetadata attaches some metadata to the KernelConnection.
	// This metadata is primarily used for attaching labels to Prometheus kernelMetricsManager.
	AddMetadata(key, value string)

	// GetMetadata retrieves a piece of metadata that may be attached to the KernelConnection.
	// This metadata is primarily used for attaching labels to Prometheus kernelMetricsManager.
	GetMetadata(key string) (string, bool)
}

type ErrorHandler func(sessionId string, kernelId string, err error)

type KernelSessionManager interface {
	// CreateSession creates a new session.
	//
	// This is thread-safe.
	CreateSession(sessionId string, path string, sessionType string, kernelSpecName string, resourceSpec *ResourceSpec) (*SessionConnection, error)

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
	InterruptKernel(sessionId string) error

	CreateFile(path string) error

	StopKernel(id string) error

	GetMetrics() KernelManagerMetrics

	// ConnectTo connect to an existing kernel.
	//
	// @param kernelId - The ID of the target kernel.
	//
	// @param sessionId - The ID of the session associated with the target kernel.
	//
	// @param username - The username to use when connecting to the kernel.
	//
	// @returns A promise that resolves with the new kernel instance.
	ConnectTo(kernelId string, sessionId string, username string) (KernelConnection, error)

	// AddMetadata attaches some metadata to the KernelSessionManager.
	//
	// All metadata should be added when the KernelSessionManager is created, as
	// the KernelSessionManager adds all metadata in its metadata dictionary to the
	// metadata dictionary of any SessionConnection and KernelConnection instances that it
	// creates. Metadata added to the KernelSessionManager after a SessionConnection or
	// KernelConnection is created will not be added to any existing SessionConnection or
	// KernelConnection instances.
	AddMetadata(key, value string)

	// GetMetadata retrieves a piece of metadata that may be attached to the KernelSessionManager.
	//
	// If there is metadata with the given key attached to the KernelSessionManager, then that metadata
	// is returned, along with a boolean equal to true.
	//
	// If there is no metadata attached to the KernelSessionManager at the given key, then the empty
	// string is returned, along with a boolean equal to false.
	GetMetadata(key string) (string, bool)

	// RegisterOnErrorHandler registers an error handler to be called if the kernel manager encounters an error.
	//
	// If there is already an existing error handler, then it is overwritten.
	RegisterOnErrorHandler(handler ErrorHandler)
}

type KernelManagerMetrics interface {
	// FileCreated records that a file has been created.
	FileCreated()

	// KernelCreated records that a kernel has been created.
	KernelCreated()

	// SessionCreated records that a Session was created.
	// It also updates the Prometheus metric for the latency of session-creation operations.
	SessionCreated(latency time.Duration, workloadId string)

	// KernelTerminated records that a kernel has been terminated.
	//KernelTerminated()

	// SessionTerminated records that a session has been terminated.
	// It also updates the Prometheus metric for the latency of session-terminated operations.
	SessionTerminated(latency time.Duration, workloadId string)
}
