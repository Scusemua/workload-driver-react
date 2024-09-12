package jupyter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ZapSessionIDKey       = "session_id"
	WorkloadIdMetadataKey = "workload-id"
)

var (
	ErrCreateFileBadRequest    = errors.New("bad request when trying to create a file")
	ErrCreateSessionBadRequest = errors.New("bad request when trying to create a new session")
	ErrStopKernelBadRequest    = errors.New("bad request when trying to stop a kernel")

	ErrCreateFileUnknownFailure    = errors.New("the 'create file' operation failed for an unknown or unexpected reason")
	ErrCreateSessionUnknownFailure = errors.New("the 'create session' operation failed for an unknown or unexpected reason")

	ErrNoActiveConnection = errors.New("no active connection to target kernel exists")

	ErrInvalidSessionName = errors.New("invalid session ID specified")
)

type kernelManagerMetricsImpl struct {
	NumFilesCreated       int `json:"num-files-created"`
	NumKernelsCreated     int `json:"num-kernels-created"`
	NumSessionsCreated    int `json:"num-sessions-created"`
	NumKernelsTerminated  int `json:"num-kernels-terminated"`
	NumSessionsTerminated int `json:"num-sessions-terminated"`
}

func (m *kernelManagerMetricsImpl) FileCreated() {
	m.NumFilesCreated += 1
}

func (m *kernelManagerMetricsImpl) KernelCreated() {
	m.NumKernelsCreated += 1
}

// SessionCreated records that a Session was created.
// It also updates the Prometheus metric for the latency of session-creation operations.
func (m *kernelManagerMetricsImpl) SessionCreated(workloadId string, latency time.Duration) {
	m.NumSessionsCreated += 1

	metrics.PrometheusMetricsWrapperInstance.JupyterSessionCreationLatency.
		With(prometheus.Labels{"workload_id": workloadId}).
		Observe(latency.Seconds())
}

//func (m *kernelManagerMetricsImpl) KernelTerminated() {
//	m.NumKernelsTerminated += 1
//}

// SessionTerminated records that a session has been terminated.
// It also updates the Prometheus metric for the latency of session-terminated operations.
func (m *kernelManagerMetricsImpl) SessionTerminated(workloadId string, latency time.Duration) {
	m.NumSessionsTerminated += 1

	metrics.PrometheusMetricsWrapperInstance.JupyterSessionTerminationLatency.
		With(prometheus.Labels{"workload_id": workloadId}).
		Observe(latency.Seconds())
}

type BasicKernelSessionManager struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	jupyterServerAddress             string                        // Address of the Jupyter Server.
	metrics                          *kernelManagerMetricsImpl     // Maintains some metrics.
	localSessionIdToKernelId         map[string]string             // Map from "local" (provided by us) Session IDs to Kernel IDs. We provide the Session IDs, while Jupyter provides the Kernel IDs.
	kernelIdToLocalSessionId         map[string]string             // Map from Kernel IDs to Jupyter Session IDs. We provide the Session IDs, while Jupyter provides the Kernel IDs.
	localSessionIdToJupyterSessionId map[string]string             // Map from "local" (provided by us) Session IDs to the Jupyter-provided Session IDs.
	kernelIdToJupyterSessionId       map[string]string             // Map from Kernel IDs to "local" Session IDs. Jupyter provides both the Session IDs and the Kernel IDs.
	sessionMap                       map[string]*SessionConnection // Map from Session ID to Session. The keys are the Session IDs supplied by us/the trace data.
	metadata                         map[string]string             // Metadata is miscellaneous metadata attached to the BasicKernelSessionManager that is mostly used for metrics
	adjustSessionNames               bool                          // If true, ensure all session names are 36 characters in length. For now, this should be true. Setting it to false causes problems for some reason...

	metadataMutex sync.Mutex
}

func NewKernelSessionManager(jupyterServerAddress string, adjustSessionNames bool, atom *zap.AtomicLevel) *BasicKernelSessionManager {
	manager := &BasicKernelSessionManager{
		jupyterServerAddress:             jupyterServerAddress,
		metrics:                          &kernelManagerMetricsImpl{},
		localSessionIdToKernelId:         make(map[string]string),
		localSessionIdToJupyterSessionId: make(map[string]string),
		kernelIdToJupyterSessionId:       make(map[string]string),
		kernelIdToLocalSessionId:         make(map[string]string),
		sessionMap:                       make(map[string]*SessionConnection),
		metadata:                         make(map[string]string),
		adjustSessionNames:               adjustSessionNames,
		atom:                             atom,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, manager.atom)
	manager.logger = zap.New(core, zap.Development())

	manager.sugaredLogger = manager.logger.Sugar()

	return manager
}

// AddMetadata attaches some metadata to the BasicKernelSessionManager.
//
// All metadata should be added when the BasicKernelSessionManager is created, as
// the BasicKernelSessionManager adds all metadata in its metadata dictionary to the
// metadata dictionary of any SessionConnection and KernelConnection instances that it
// creates. Metadata added to the BasicKernelSessionManager after a SessionConnection or
// KernelConnection is created will not be added to any existing SessionConnection or
// KernelConnection instances.
//
// This particular implementation of AddMetadata is thread-safe.
func (m *BasicKernelSessionManager) AddMetadata(key, value string) {
	m.metadataMutex.Lock()
	defer m.metadataMutex.Unlock()

	m.metadata[key] = value
}

// GetMetadata retrieves a piece of metadata that may be attached to the KernelSessionManager.
//
// If there is metadata with the given key attached to the KernelSessionManager, then that metadata
// is returned, along with a boolean equal to true.
//
// If there is no metadata attached to the KernelSessionManager at the given key, then the empty
// string is returned, along with a boolean equal to false.
//
// This particular implementation of GetMetadata is thread-safe.
func (m *BasicKernelSessionManager) GetMetadata(key string) (string, bool) {
	m.metadataMutex.Lock()
	defer m.metadataMutex.Unlock()

	value, ok := m.metadata[key]
	if !ok {
		m.logger.Warn("Could not find metadata with specified key attached to BasicKernelSessionManager.", zap.String("key", key))
	}
	return value, ok
}

// CreateSession creates a new session.
func (m *BasicKernelSessionManager) CreateSession(sessionId string, path string, sessionType string, kernelSpecName string, resourceSpec *ResourceSpec) (*SessionConnection, error) {
	if m.adjustSessionNames {
		if len(sessionId) < 36 {
			generatedUuid := uuid.NewString()
			sessionId = strings.ToLower(sessionId) + "-" + generatedUuid[0:36-(len(sessionId)+1)]
		} else if len(sessionId) > 36 {
			return nil, fmt.Errorf("%w: specified session ID \"%s\" is too long (max length is 36 characters when the KernelSessionManager has been configured to adjust names)", ErrInvalidSessionName, sessionId)
		}
	}

	requestBody := newJupyterSessionForRequest(sessionId, path, sessionType, kernelSpecName, resourceSpec)

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

	sentAt := time.Now()
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
			if err := json.Unmarshal(body, &jupyterSession); err != nil {
				m.logger.Error("Failed to decode Jupyter Session from JSON.", zap.Error(err))
				return nil, err
			}

			m.logger.Debug("Received 'Created' when creating session", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.String(ZapSessionIDKey, sessionId))

			var kernelId = jupyterSession.JupyterKernel.Id
			var jupyterSessionId = jupyterSession.JupyterSessionId

			// Update all of our many mappings...
			m.localSessionIdToKernelId[sessionId] = kernelId
			m.localSessionIdToJupyterSessionId[sessionId] = jupyterSessionId
			m.kernelIdToJupyterSessionId[kernelId] = jupyterSessionId
			m.kernelIdToLocalSessionId[kernelId] = sessionId

			st := time.Now()
			// Connect to the Session and to the associated kernel.
			sessionConnection, err = NewSessionConnection(jupyterSession, "", m.jupyterServerAddress, m.atom)
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
			m.logger.Error("Received HTTP 400 'Bad Request' when creating session", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", body))
			return nil, fmt.Errorf("ErrCreateSessionBadRequest %w : %s", ErrCreateSessionBadRequest, string(body))
		}
	default:
		var responseJson map[string]interface{}
		if err := json.Unmarshal(body, &responseJson); err != nil {
			m.logger.Error("Failed to decode JSON response with unexpected HTTP status code.", zap.Int("http-status-code", http.StatusBadRequest), zap.Error(err))
		}

		m.logger.Warn("Unexpected response status code when creating a new session.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", body), zap.String("request-args", requestBody.String()), zap.String("response-body", string(body)))
		if message, ok := responseJson["message"]; ok {
			return nil, fmt.Errorf("ErrCreateSessionUnknownFailure %w: %s", ErrCreateSessionUnknownFailure, message)
		} else if reason, ok := responseJson["reason"]; ok {
			return nil, fmt.Errorf("ErrCreateSessionUnknownFailure %w: %s", ErrCreateSessionUnknownFailure, reason)
		} else {
			return nil, fmt.Errorf("ErrCreateSessionUnknownFailure %w: %s", ErrCreateSessionUnknownFailure, string(body))
		}
	}

	workloadId, _ := m.GetMetadata(WorkloadIdMetadataKey)
	m.metrics.SessionCreated(workloadId, time.Since(sentAt))
	// TODO(Ben): Does this also create a new kernel?
	return sessionConnection, nil
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
func (m *BasicKernelSessionManager) InterruptKernel(sessionId string) error {
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
	if conn.ConnectionStatus() == KernelDead {
		// Cannot interrupt a dead kernel.
		return ErrKernelIsDead
	}

	var requestBody = make(map[string]interface{})
	requestBody["kernel_id"] = conn.KernelId()

	requestBodyEncoded, err := json.Marshal(requestBody)
	if err != nil {
		m.logger.Error("Failed to marshal request body for kernel interruption request", zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s/api/kernels/%s/interrupt", conn.JupyterServerAddress(), conn.KernelId())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBodyEncoded))

	if err != nil {
		m.logger.Error("Failed to create HTTP request for kernel interruption.", zap.String("url", url), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Error while issuing HTTP request to interrupt kernel.", zap.String("url", url), zap.Error(err))
		return err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logger.Error("Failed to read response to interrupting kernel.", zap.Error(err))
		return err
	}

	m.logger.Debug("Received response to interruption request.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("response", data))
	return nil
}

func (m *BasicKernelSessionManager) CreateFile(path string) error {
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

func (m *BasicKernelSessionManager) StopKernel(id string) error {
	url := fmt.Sprintf("http://%s/api/sessions/%s", m.jupyterServerAddress, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		m.logger.Error("Failed to create DeleteSession request while stopping kernel.", zap.String(ZapSessionIDKey, id), zap.Error(err))
		return err
	}

	client := &http.Client{}

	sentAt := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when stopping session.", zap.String(ZapSessionIDKey, id), zap.String("url", url), zap.Error(err))
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusNoContent: // https://jupyter-server.readthedocs.io/en/latest/developers/rest-api.html#delete--api-kernels-kernel_id
		{
			m.logger.Debug("Received 'OK'", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
		}
	case http.StatusBadRequest:
		{
			m.logger.Error("Received HTTP 400 'Bad Request'", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
			return fmt.Errorf("ErrStopKernelBadRequest %w : %s", ErrStopKernelBadRequest, string(body))
		}
	default:
		m.logger.Warn("Unexpected response status code when stopping session.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
	}

	workloadId, _ := m.GetMetadata(WorkloadIdMetadataKey)
	m.metrics.SessionTerminated(workloadId, time.Since(sentAt))
	// TODO(Ben): Does this also terminate the kernel?
	return nil
}

func (m *BasicKernelSessionManager) GetMetrics() KernelManagerMetrics {
	return m.metrics
}

// ConnectTo connects to an existing kernel.
//
// @param kernelId - The ID of the target kernel.
//
// @param sessionId - The ID of the session associated with the target kernel.
//
// @returns a WebSocket-backed connection to the kernel.
func (m *BasicKernelSessionManager) ConnectTo(kernelId string, sessionId string, username string) (KernelConnection, error) {
	m.logger.Debug("Connecting to kernel now.", zap.String("kernel_id", kernelId), zap.String("session_id", sessionId))
	conn, err := NewKernelConnection(kernelId, sessionId, username, m.jupyterServerAddress, m.atom)
	if err != nil {
		m.logger.Error("Failed to connect to kernel.", zap.String("kernel_id", kernelId), zap.String("session_id", sessionId))
	} else {
		m.logger.Debug("Successfully connected to kernel.", zap.String("kernel_id", kernelId), zap.String("session_id", sessionId))

		// Add all the BasicKernelSessionManager's metadata to the new KernelConnection.
		m.metadataMutex.Lock()
		defer m.metadataMutex.Unlock()
		for key, value := range m.metadata {
			conn.AddMetadata(key, value)
		}
	}

	// On success, conn will be non-nil and err will be nil.
	// If there is an error, then err will be non-nil and connection will be nil.
	return conn, err
}
