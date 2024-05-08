package jupyter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ZapSessionIDKey = "session-id"
)

var (
	ErrCreateFileBadRequest    = errors.New("bad request when trying to create a file")
	ErrCreateSessionBadRequest = errors.New("bad request when trying to create a new session")
	ErrStopKernelBadRequest    = errors.New("bad request when trying to stop a kernel")

	ErrCreateFileUnknownFailure    = errors.New("the 'create file' operation failed for an unknown or unexpected reason")
	ErrCreateSessionUnknownFailure = errors.New("the 'create session' operation failed for an unknown or unexpected reason")

	ErrNoActiveConnection = errors.New("no active connection to target kernel exists")
)

type KernelManagerMetrics struct {
	NumFilesCreated       int `json:"num-files-created"`
	NumKernelsCreated     int `json:"num-kernels-created"`
	NumSessionsCreated    int `json:"num-sessions-created"`
	NumKernelsTerminated  int `json:"num-kernels-terminated"`
	NumSessionsTerminated int `json:"num-sessions-terminated"`
}

func (m *KernelManagerMetrics) FileCreated() {
	m.NumFilesCreated += 1
}

func (m *KernelManagerMetrics) KernelCreated() {
	m.NumKernelsCreated += 1
}

func (m *KernelManagerMetrics) SessionCreated() {
	m.NumSessionsCreated += 1
}

func (m *KernelManagerMetrics) KernelTerminated() {
	m.NumKernelsTerminated += 1
}

func (m *KernelManagerMetrics) SessionTerminated() {
	m.NumSessionsTerminated += 1
}

type KernelSessionManager struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	// Address of the Jupyter Server.
	jupyterServerAddress string

	// Maintains some metrics.
	metrics *KernelManagerMetrics

	localSessionIdToKernelId map[string]string // Map from "local" (provided by us) Session IDs to Kernel IDs. We provide the Session IDs, while Jupyter provides the Kernel IDs.
	kernelIdToLocalSessionId map[string]string // Map from Kernel IDs to Jupyter Session IDs. We provide the Session IDs, while Jupyter provides the Kernel IDs.

	localSessionIdToJupyterSessionId map[string]string // Map from "local" (provided by us) Session IDs to the Jupyter-provided Session IDs.
	jupyterSessionIdLocalSessionIdTo map[string]string

	kernelIdToJupyterSessionId map[string]string // Map from Kernel IDs to "local" Session IDs. Jupyter provides both the Session IDs and the Kernel IDs.
	jupyterSessionIdToKernelId map[string]string

	sessionMap map[string]*SessionConnection // Map from Session ID to Session. The keys are the Session IDs supplied by us/the trace data.
}

func NewKernelManager(opts *domain.Configuration, atom *zap.AtomicLevel) *KernelSessionManager {
	manager := &KernelSessionManager{
		jupyterServerAddress:             opts.JupyterServerAddress,
		metrics:                          &KernelManagerMetrics{},
		localSessionIdToKernelId:         make(map[string]string),
		localSessionIdToJupyterSessionId: make(map[string]string),
		jupyterSessionIdLocalSessionIdTo: make(map[string]string),
		jupyterSessionIdToKernelId:       make(map[string]string),
		kernelIdToJupyterSessionId:       make(map[string]string),
		kernelIdToLocalSessionId:         make(map[string]string),
		sessionMap:                       make(map[string]*SessionConnection),
		atom:                             atom,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, manager.atom)
	manager.logger = zap.New(core, zap.Development())

	manager.sugaredLogger = manager.logger.Sugar()

	return manager
}

// Start a new kernel.
func (m *KernelSessionManager) StartNew(kernelSpec string) error {
	return nil
}

// Create a new session.
func (m *KernelSessionManager) CreateSession(sessionId string, path string, sessionType string, kernelSpecName string) (*SessionConnection, error) {
	if len(sessionId) < 36 {
		generated_uuid := uuid.NewString()
		sessionId = sessionId + "-" + generated_uuid[0:36-(len(sessionId)+1)]
	}

	requestBody := newJupyterSessionForRequest(sessionId, path, sessionType, kernelSpecName)

	requestBodyJson, err := json.Marshal(&requestBody)
	if err != nil {
		m.logger.Error("Error encountered while marshalling payload for CreateSession operation.", zap.Error(err))
		return nil, err
	}

	m.logger.Debug("Issuing 'CREATE-SESSION' request now.", zap.String("request-args", requestBody.String()))

	url := fmt.Sprintf("http://%s/api/sessions", m.jupyterServerAddress)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBodyJson))
	if err != nil {
		m.logger.Error("Error encountered while creating request for CreateFile operation.", zap.String("request-args", requestBody.String()), zap.String("path", path), zap.String("url", url), zap.Error(err))
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when creating new session.", zap.String("request-args", requestBody.String()), zap.String("local-session-id", sessionId), zap.String("url", url), zap.Error(err))
		return nil, err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var sessionConnection *SessionConnection
	switch resp.StatusCode {
	case http.StatusCreated:
		{
			var jupyterSession *jupyterSession
			json.Unmarshal(body, &jupyterSession)
			m.logger.Debug("Received 'Created' when creating session", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.String(ZapSessionIDKey, sessionId))

			var kernelId string = jupyterSession.JupyterKernel.Id
			var jupyterSessionId string = jupyterSession.JupyterSessionId

			// Update all of our many mappings...
			m.localSessionIdToKernelId[sessionId] = kernelId
			m.localSessionIdToJupyterSessionId[sessionId] = jupyterSessionId
			m.jupyterSessionIdLocalSessionIdTo[jupyterSessionId] = sessionId
			m.kernelIdToJupyterSessionId[kernelId] = jupyterSessionId
			m.kernelIdToLocalSessionId[kernelId] = sessionId
			m.jupyterSessionIdToKernelId[jupyterSessionId] = kernelId

			st := time.Now()
			// Connect to the Session and to the associated kernel.
			sessionConnection, err = NewSessionConnection(jupyterSession, m.jupyterServerAddress, m.atom)
			if err != nil {
				m.logger.Error("Could not establish connection to Session.", zap.String(ZapSessionIDKey, sessionId), zap.String("kernel-id", kernelId), zap.Error(err))
				return nil, err
			}
			creationTime := time.Since(st)

			m.sessionMap[sessionId] = sessionConnection

			m.logger.Debug("Successfully created and setup session.", zap.Duration("time-to-create", creationTime), zap.String("local-session-id", sessionId), zap.String("jupyter-session-id", jupyterSessionId), zap.String("kernel-id", kernelId))
		}
	case http.StatusBadRequest:
		{
			var responseJson map[string]interface{}
			json.Unmarshal(body, &responseJson)
			m.logger.Error("Received HTTP 400 'Bad Request' when creating session", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", body))
			return nil, fmt.Errorf("ErrCreateSessionBadRequest %w : %s", ErrCreateSessionBadRequest, string(body))
		}
	default:
		var responseJson map[string]interface{}
		json.Unmarshal(body, &responseJson)
		m.logger.Warn("Unexpected respone status code when creating a new session.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", body), zap.String("request-args", requestBody.String()))

		return nil, fmt.Errorf("ErrCreateSessionUnknownFailure %w: %s", ErrCreateSessionUnknownFailure, string(body))
	}

	m.metrics.SessionCreated()
	// TODO(Ben): Does this also create a new kernel?
	return sessionConnection, nil
}

// Flip the 'run_training_code' flag within the kernel so that it stops executing training code.
func (m *KernelSessionManager) StopRunningTrainingCode(sessionId string) error {
	return nil
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
func (m *KernelSessionManager) InterruptKernel(sessionId string) error {
	sess, ok := m.sessionMap[sessionId]
	if !ok {
		m.logger.Error("Cannot interrupt kernel. Associated kernel/session not found.", zap.String("session_id", sessionId))
		return ErrKernelNotFound
	}

	if sess.kernel == nil {
		m.logger.Error("Cannot interrupt kernel. No active connection to kernel.", zap.String("session_id", sessionId))
		return ErrNoActiveConnection
	}

	conn := sess.kernel
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

func (m *KernelSessionManager) CreateFile(path string) error {
	url := fmt.Sprintf("http://%s/api/contents/%s", m.jupyterServerAddress, path)

	createFileRequest := newCreateFileRequest(path)
	payload, err := json.Marshal(&createFileRequest)
	if err != nil {
		m.logger.Error("Error encountered while marshalling payload for CreateFile operation.", zap.Error(err))
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		m.logger.Error("Error encountered while creating request for CreateFile operation.", zap.String("path", path), zap.String("url", url), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when creating new file.", zap.String("path", path), zap.String("url", url), zap.Error(err))
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusCreated:
		{
			m.logger.Debug("Received 'Created' when creating file", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
		}
	case http.StatusBadRequest:
		{
			m.logger.Error("Received HTTP 400 'Bad Request' when creating file", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
			return fmt.Errorf("ErrCreateFileBadRequest %w : %s", ErrCreateFileBadRequest, string(body))
		}
	case http.StatusNotFound:
		{
			m.logger.Error("Received HTTP 400 'Bad Request' when creating file", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
			return fmt.Errorf("ErrCreateFileBadRequest %w : %s", ErrCreateFileBadRequest, string(body))
		}
	default:
		m.logger.Warn("Unexpected respone status code when creating file.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))

		if resp.StatusCode >= 400 {
			return fmt.Errorf("ErrCreateFileUnknownFailure %w: %s", ErrCreateFileUnknownFailure, string(body))
		}
	}

	m.metrics.FileCreated()
	return nil
}

func (m *KernelSessionManager) StopKernel(id string) error {
	url := fmt.Sprintf("http://%s/api/sessions/%s", m.jupyterServerAddress, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		m.logger.Error("Failed to create DeleteSession request while stopping kernel.", zap.String(ZapSessionIDKey, id), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when stopping session.", zap.String(ZapSessionIDKey, id), zap.String("url", url), zap.Error(err))
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		{
			m.logger.Debug("Received 'OK'", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
		}
	case http.StatusBadRequest:
		{
			m.logger.Error("Received HTTP 400 'Bad Request'", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
			return fmt.Errorf("ErrStopKernelBadRequest %w : %s", ErrStopKernelBadRequest, string(body))
		}
	default:
		m.logger.Warn("Unexpected respone status code when stopping session.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
	}

	m.metrics.SessionTerminated()
	// TODO(Ben): Does this also terminate the kernel?
	return nil
}

func (m *KernelSessionManager) GetMetrics() *KernelManagerMetrics {
	return m.metrics
}
