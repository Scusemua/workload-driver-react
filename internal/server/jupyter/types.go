package jupyter

type KernelConnection interface {
	// Get the connection status of the kernel.
	ConnectionStatus() KernelConnectionStatus

	// Return true if the connection is currently active.
	Connected() bool

	// Return the ID of the kernel itself.
	KernelId() string

	ClientId() string

	Username() string

	// Return the address of the Jupyter Server associated with this kernel.
	JupyterServerAddress() string

	// Send an `execute_request` message.
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
	RequestExecute(code string, silent bool, storeHistory bool, userExpressions map[string]interface{}, allowStdin bool, stopOnError bool, waitForResponse bool) error

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
	InterruptKernel() error

	// Send a `stop_running_training_code` message.
	StopRunningTrainingCode(waitForResponse bool) error
}

type KernelSessionManager interface {
	// Create a new session.
	CreateSession(sessionId string, path string, sessionType string, kernelSpecName string) (*SessionConnection, error)

	// Start a new kernel.
	StartNewKernel(kernelSpec string) error

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
	InterruptKernel(sessionId string) error

	CreateFile(path string) error

	StopKernel(id string) error

	GetMetrics() KernelManagerMetrics

	/**
	 * Connect to an existing kernel.
	 *
	 * @param kernelId - The ID of the target kernel.
	 * @param sessionId - The ID of the session associated with the target kernel.
	 * @param username - The username to use when connecting to the kernel.
	 *
	 * @returns A promise that resolves with the new kernel instance.
	 */
	ConnectTo(kernelId string, sessionId string, username string) (KernelConnection, error)
}

type KernelManagerMetrics interface {
	FileCreated()       // Record that a file has been created.
	KernelCreated()     // Record that a kernel has been created.
	SessionCreated()    // Record that a session has been created.
	KernelTerminated()  // Record that a kernel has been terminated.
	SessionTerminated() // Record that a session has been terminated.
}
