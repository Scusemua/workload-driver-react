package jupyter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
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

	// RemoteStorageDefinitionMetadataKey is used to register a proto.RemoteStorageDefinition with a
	// KernelSessionManager so that the proto.RemoteStorageDefinition can be embedded in the metadata
	// of "execute_request" and "yield_request" messages to instruct the kernel how to simulate
	// remote storage reads and writes.
	RemoteStorageDefinitionMetadataKey = "remote_storage_definition"
)

var (
	ErrCreateFileBadRequest    = errors.New("bad request when trying to create a file")
	ErrCreateSessionBadRequest = errors.New("bad request when trying to create a new session")
	ErrStopKernelBadRequest    = errors.New("bad request when trying to stop a kernel")
	ErrStopKernelNotFound      = errors.New("jupyter server could not find specified kernel during stop-kernel operation")

	ErrCreateFileUnknownFailure    = errors.New("the 'create file' operation failed for an unknown or unexpected reason")
	ErrCreateSessionUnknownFailure = errors.New("the 'create session' operation failed for an unknown or unexpected reason")

	ErrNoActiveConnection = errors.New("no active connection to target kernel exists")

	ErrInvalidSessionName = errors.New("invalid session ID specified")
)

type KernelMetricsManager struct {
	NumFilesCreated       int `json:"num-files-created"`
	NumKernelsCreated     int `json:"num-kernels-created"`
	NumSessionsCreated    int `json:"num-sessions-created"`
	NumKernelsTerminated  int `json:"num-kernels-terminated"`
	NumSessionsTerminated int `json:"num-sessions-terminated"`

	// metricsConsumer is used to publish metrics to Prometheus.
	metricsConsumer MetricsConsumer
}

func (m *KernelMetricsManager) FileCreated() {
	m.NumFilesCreated += 1
}

func (m *KernelMetricsManager) KernelCreated() {
	m.NumKernelsCreated += 1
}

// SessionCreated records that a Session was created.
// It also updates the Prometheus metric for the latency of session-creation operations.
func (m *KernelMetricsManager) SessionCreated(latency time.Duration, workloadId string) {
	m.NumSessionsCreated += 1

	if m.metricsConsumer != nil {
		m.metricsConsumer.ObserveJupyterSessionCreationLatency(latency.Milliseconds(), workloadId)
	}
}

//func (m *KernelMetricsManager) KernelTerminated() {
//	m.NumKernelsTerminated += 1
//}

// SessionTerminated records that a session has been terminated.
// It also updates the Prometheus metric for the latency of session-terminated operations.
func (m *KernelMetricsManager) SessionTerminated(latency time.Duration, workloadId string) {
	m.NumSessionsTerminated += 1

	if m.metricsConsumer != nil {
		m.metricsConsumer.ObserveJupyterSessionTerminationLatency(latency.Milliseconds(), workloadId)
	}
}

type BasicKernelSessionManager struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	jupyterServerAddress             string                        // Address of the Jupyter Server.
	kernelMetricsManager             *KernelMetricsManager         // Maintains some metrics for the kernel.
	localSessionIdToKernelId         map[string]string             // Map from "local" (provided by us) Session IDs to Kernel IDs. We provide the Session IDs, while Jupyter provides the Kernel IDs.
	kernelIdToLocalSessionId         map[string]string             // Map from Kernel IDs to Jupyter Session IDs. We provide the Session IDs, while Jupyter provides the Kernel IDs.
	localSessionIdToJupyterSessionId map[string]string             // Map from "local" (provided by us) Session IDs to the Jupyter-provided Session IDs.
	kernelIdToJupyterSessionId       map[string]string             // Map from Kernel IDs to "local" Session IDs. Jupyter provides both the Session IDs and the Kernel IDs.
	sessionMap                       map[string]*SessionConnection // Map from Session ID to Session. The keys are the Session IDs supplied by us/the trace data.
	metadata                         map[string]interface{}        // Metadata is miscellaneous metadata attached to the BasicKernelSessionManager that is mostly used for kernelMetricsManager
	metadataMutex                    sync.Mutex                    // Synchronizes access to the metadata map.
	adjustSessionNames               bool                          // If true, ensure all session names are 36 characters in length. For now, this should be true. Setting it to false causes problems for some reason...

	// Invoked in a new goroutine when an error occurs.
	onError ErrorHandler

	mu sync.Mutex
}

func NewKernelSessionManager(jupyterServerAddress string, adjustSessionNames bool, atom *zap.AtomicLevel, metricsConsumer MetricsConsumer) *BasicKernelSessionManager {
	manager := &BasicKernelSessionManager{
		jupyterServerAddress:             jupyterServerAddress,
		kernelMetricsManager:             &KernelMetricsManager{metricsConsumer: metricsConsumer},
		localSessionIdToKernelId:         make(map[string]string),
		localSessionIdToJupyterSessionId: make(map[string]string),
		kernelIdToJupyterSessionId:       make(map[string]string),
		kernelIdToLocalSessionId:         make(map[string]string),
		sessionMap:                       make(map[string]*SessionConnection),
		metadata:                         make(map[string]interface{}),
		adjustSessionNames:               adjustSessionNames,
		atom:                             atom,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, manager.atom)
	manager.logger = zap.New(core, zap.Development())

	manager.sugaredLogger = manager.logger.Sugar()

	return manager
}

// RegisterOnErrorHandler registers an error handler to be called if the kernel manager encounters an error.
// The error handler is invoked in a new goroutine.
//
// If there is already an existing error handler, then it is overwritten.
func (m *BasicKernelSessionManager) RegisterOnErrorHandler(handler ErrorHandler) {
	m.onError = handler
}

func (m *BasicKernelSessionManager) tryCallErrorHandler(kernelId string, sessionId string, err error) {
	if strings.Contains(err.Error(), "insufficient hosts available") {
		return
	}

	if m.onError != nil {
		go m.onError(kernelId, sessionId, err)
	}
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
func (m *BasicKernelSessionManager) AddMetadata(key string, value interface{}) {
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
func (m *BasicKernelSessionManager) GetMetadata(key string) (interface{}, bool) {
	m.metadataMutex.Lock()
	defer m.metadataMutex.Unlock()

	value, ok := m.metadata[key]
	if !ok {
		m.logger.Warn("Could not find metadata with specified key attached to BasicKernelSessionManager.", zap.String("key", key))
	}
	return value, ok
}

// CreateSession creates a new session.
//
// This is thread-safe.
func (m *BasicKernelSessionManager) CreateSession(sessionId string, sessionPath string, sessionType string,
	kernelSpecName string, resourceSpec *ResourceSpec) (*SessionConnection, error) {

	workloadId, loadedWorkloadIdFromMetadata := m.GetMetadata(WorkloadIdMetadataKey)

	if m.adjustSessionNames {
		if len(sessionId) < 36 {
			generatedUuid := uuid.NewString()
			sessionId = strings.ToLower(sessionId) + "-" + generatedUuid[0:36-(len(sessionId)+1)]
		} else if len(sessionId) > 36 {
			return nil, fmt.Errorf("%w: specified session ID \"%s\" is too long (max length is 36 characters when the KernelSessionManager has been configured to adjust names)", ErrInvalidSessionName, sessionId)
		}
	}

	var requestBody *jupyterSessionReq
	if loadedWorkloadIdFromMetadata {
		requestBody = newJupyterSessionForRequest(sessionId, sessionPath, sessionType, kernelSpecName, resourceSpec, workloadId.(string))
	} else {
		requestBody = newJupyterSessionForRequest(sessionId, sessionPath, sessionType, kernelSpecName, resourceSpec, "N/A")
	}

	requestBodyJson, err := json.Marshal(&requestBody)
	if err != nil {
		m.logger.Error("Error encountered while marshalling payload for CreateSession operation.", zap.Error(err))
		m.tryCallErrorHandler("", sessionId, err)
		return nil, err
	}

	address := path.Join(m.jupyterServerAddress, "/api/sessions")
	url := fmt.Sprintf("http://%s", address)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBodyJson))
	if err != nil {
		m.logger.Error("Error encountered while creating request for CreateFile operation.", zap.String("request-args", requestBody.String()), zap.String("sessionPath", sessionPath), zap.String("url", url), zap.Error(err))
		m.tryCallErrorHandler("", sessionId, err)
		return nil, err
	}

	m.logger.Debug("Issuing 'CREATE-SESSION' request now.", zap.String("request-args", requestBody.String()), zap.String("request-url", url))

	sentAt := time.Now()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when creating new session.", zap.String("request-args", requestBody.String()), zap.String("local-session-id", sessionId), zap.String("url", url), zap.Error(err))
		m.tryCallErrorHandler("", sessionId, err)
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
				m.tryCallErrorHandler("", sessionId, err)
				return nil, err
			}

			m.logger.Debug("Received 'Created' when creating session", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.String(ZapSessionIDKey, sessionId))

			var kernelId = jupyterSession.JupyterKernel.Id
			var jupyterSessionId = jupyterSession.JupyterSessionId

			m.mu.Lock()
			// Update all of our many mappings...
			m.localSessionIdToKernelId[sessionId] = kernelId
			m.localSessionIdToJupyterSessionId[sessionId] = jupyterSessionId
			m.kernelIdToJupyterSessionId[kernelId] = jupyterSessionId
			m.kernelIdToLocalSessionId[kernelId] = sessionId
			m.mu.Unlock()

			st := time.Now()
			// Connect to the Session and to the associated kernel.
			sessionConnection, err = NewSessionConnection(jupyterSession, "", m.jupyterServerAddress, m.atom, m.kernelMetricsManager.metricsConsumer, func(err error) {
				m.tryCallErrorHandler(kernelId, sessionId, err)
			})
			if err != nil {
				m.logger.Error("Could not establish connection to Session.", zap.String(ZapSessionIDKey, sessionId), zap.String("kernel_id", kernelId), zap.Error(err))
				m.tryCallErrorHandler("", sessionId, err)
				return nil, err
			}
			creationTime := time.Since(st)

			m.mu.Lock()
			m.sessionMap[sessionId] = sessionConnection
			m.mu.Unlock()

			m.logger.Debug("Successfully created and setup session.", zap.Duration("time-to-create", creationTime), zap.String("local-session-id", sessionId), zap.String("jupyter-session-id", jupyterSessionId), zap.String("kernel_id", kernelId))
		}
	case http.StatusBadRequest:
		{
			m.logger.Error("Received HTTP 400 'Bad Request' when creating session", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", body))
			m.tryCallErrorHandler("", sessionId, err)
			return nil, fmt.Errorf("ErrCreateSessionBadRequest %w : %s", ErrCreateSessionBadRequest, string(body))
		}
	case http.StatusInternalServerError:
		{
			m.logger.Warn("Failed to create session due to HTTP 500 Internal Server Error.",
				zap.String("status", resp.Status),
				zap.Any("headers", resp.Header),
				zap.String("body", string(body)))

			return nil, fmt.Errorf(string(body))
		}
	default:
		var responseJson map[string]interface{}
		if err := json.Unmarshal(body, &responseJson); err != nil {
			m.logger.Error("Failed to decode JSON response with unexpected HTTP status code.", zap.Int("http-status-code", http.StatusBadRequest), zap.Error(err))
		}

		m.logger.Warn("Unexpected response status code when creating a new session.",
			zap.Int("status-code", resp.StatusCode),
			zap.String("status", resp.Status),
			zap.Any("headers", resp.Header),
			zap.Any("body", body),
			zap.String("request-args", requestBody.String()),
			zap.String("response-body", string(body)),
			zap.Any("response-json", responseJson))

		if message, ok := responseJson["message"]; ok {
			err = fmt.Errorf("%w: HTTP %d %s - %s",
				ErrCreateSessionUnknownFailure, resp.StatusCode, resp.Status, message)

			m.tryCallErrorHandler("", sessionId, err)

			return nil, err
		}

		if reason, ok := responseJson["reason"]; ok {
			err = fmt.Errorf("%w: HTTP %d %s - %s",
				ErrCreateSessionUnknownFailure, resp.StatusCode, resp.Status, reason)

			m.tryCallErrorHandler("", sessionId, err)

			return nil, err
		}

		err = fmt.Errorf("%w: HTTP %d %s - %s",
			ErrCreateSessionUnknownFailure, resp.StatusCode, resp.Status, string(body))

		m.tryCallErrorHandler("", sessionId, err)

		return nil, err
	}

	if loadedWorkloadIdFromMetadata {
		m.mu.Lock()
		m.kernelMetricsManager.SessionCreated(time.Since(sentAt), workloadId.(string))
		m.mu.Unlock()

		err := sessionConnection.AddMetadata(WorkloadIdMetadataKey, workloadId.(string), true)
		if err != nil {
			m.logger.Error("Error while adding metadata to session connection.",
				zap.String(ZapSessionIDKey, sessionId),
				zap.String("metadata_key", WorkloadIdMetadataKey),
				zap.String("metadata_value", workloadId.(string)),
				zap.String("workload_id", workloadId.(string)))
		}
	} else {
		m.logger.Warn("Could not load WorkloadID metadata from KernelSessionManager while creating session.",
			zap.String("session_id", sessionId),
			zap.Int("num_metadata_entries", len(m.metadata)))
	}

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
		return fmt.Errorf("%w: no connection to kernel \"%s\" (session ID = \"%s\")",
			ErrKernelIsDead, conn.KernelId(), sessionId)
	}

	var requestBody = make(map[string]interface{})
	kernelId := conn.KernelId()
	requestBody["kernel_id"] = kernelId

	requestBodyEncoded, err := json.Marshal(requestBody)
	if err != nil {
		m.logger.Error("Failed to marshal request body for kernel interruption request", zap.Error(err))
		m.tryCallErrorHandler(kernelId, sessionId, err)
		return err
	}

	url := fmt.Sprintf("%s/api/kernels/%s/interrupt", conn.JupyterServerAddress(), conn.KernelId())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBodyEncoded))

	if err != nil {
		m.logger.Error("Failed to create HTTP request for kernel interruption.", zap.String("url", url), zap.Error(err))
		m.tryCallErrorHandler(kernelId, sessionId, err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Error while issuing HTTP request to interrupt kernel.", zap.String("url", url), zap.Error(err))
		m.tryCallErrorHandler(kernelId, sessionId, err)
		return err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logger.Error("Failed to read response to interrupting kernel.", zap.Error(err))
		m.tryCallErrorHandler(kernelId, sessionId, err)
		return err
	}

	m.logger.Debug("Received response to interruption request.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("response", data))
	return nil
}

func (m *BasicKernelSessionManager) CreateFile(target string) error {
	pathSuffix := fmt.Sprintf("/api/contents/%s", target)
	address := path.Join(m.jupyterServerAddress, pathSuffix)
	url := fmt.Sprintf("http://%s", address)

	createFileRequest := newCreateFileRequest(target)
	payload, err := json.Marshal(&createFileRequest)
	if err != nil {
		m.logger.Error("Error encountered while marshalling payload for CreateFile operation.", zap.Error(err))
		m.tryCallErrorHandler("", "", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		m.logger.Error("Error encountered while creating request for CreateFile operation.", zap.String("target", target), zap.String("url", url), zap.Error(err))
		m.tryCallErrorHandler("", "", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when creating new file.", zap.String("target", target), zap.String("url", url), zap.Error(err))
		m.tryCallErrorHandler("", "", err)
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
			err = fmt.Errorf("ErrCreateFileBadRequest %w : %s", ErrCreateFileBadRequest, string(body))
			m.tryCallErrorHandler("", "", err)
			return err
		}
	case http.StatusNotFound:
		{
			m.logger.Error("Received HTTP 400 'Bad Request' when creating file", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
			err = fmt.Errorf("ErrCreateFileBadRequest %w : %s", ErrCreateFileBadRequest, string(body))
			m.tryCallErrorHandler("", "", err)
			return err
		}
	default:
		m.logger.Warn("Unexpected respone status code when creating file.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))

		if resp.StatusCode >= 400 {
			err = fmt.Errorf("ErrCreateFileUnknownFailure %w: %s", ErrCreateFileUnknownFailure, string(body))
			m.tryCallErrorHandler("", "", err)
			return err
		}
	}

	m.kernelMetricsManager.FileCreated()
	return nil
}

func (m *BasicKernelSessionManager) StopKernel(id string) error {
	pathSuffix := fmt.Sprintf("/api/sessions/%s", id)
	address := path.Join(m.jupyterServerAddress, pathSuffix)
	url := fmt.Sprintf("http://%s", address)

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
			return fmt.Errorf("%w : %s", ErrStopKernelBadRequest, string(body))
		}
	case http.StatusNotFound:
		{
			m.logger.Error("Received HTTP 404 'Bad Request'", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
			return fmt.Errorf("%w: \"%s\" (%s)", ErrStopKernelNotFound, id, string(body))
		}
	default:
		m.logger.Warn("Unexpected response status code when stopping session.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", string(body)))
	}

	workloadId, loaded := m.GetMetadata(WorkloadIdMetadataKey)

	if loaded {
		m.kernelMetricsManager.SessionTerminated(time.Since(sentAt), workloadId.(string))
	} else {
		m.logger.Warn("Could not load WorkloadId from KernelSessionManager metadata while stopping kernel.",
			zap.String("kernel_id", id),
			zap.Int("num_metadata_entries", len(m.metadata)))
	}

	// TODO(Ben): Does this also terminate the kernel?
	return nil
}

func (m *BasicKernelSessionManager) GetMetrics() KernelManagerMetrics {
	return m.kernelMetricsManager
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
	conn, err := NewKernelConnection(kernelId, sessionId, username, m.jupyterServerAddress, m.atom, m.kernelMetricsManager.metricsConsumer, func(err error) { m.tryCallErrorHandler(kernelId, sessionId, err) })
	if err != nil {
		m.logger.Error("Failed to connect to kernel.",
			zap.String("kernel_id", kernelId), zap.String("session_id", sessionId))
		m.tryCallErrorHandler(kernelId, sessionId, err)
		return nil, err
	}

	m.logger.Debug("Successfully connected to kernel.",
		zap.String("kernel_id", kernelId), zap.String("session_id", sessionId))

	// Add all the BasicKernelSessionManager's metadata to the new KernelConnection.
	m.metadataMutex.Lock()
	defer m.metadataMutex.Unlock()
	for key, value := range m.metadata {
		m.logger.Debug("Adding metadata to kernel.", zap.String("kernel_id", kernelId),
			zap.String("metadata_key", key), zap.Any("metadata_value", value))
		conn.AddMetadata(key, value)
	}

	// On success, conn will be non-nil and err will be nil.
	// If there is an error, then err will be non-nil and connection will be nil.
	return conn, nil
}
