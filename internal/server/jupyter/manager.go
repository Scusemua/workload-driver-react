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

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrCreateFileBadRequest    = errors.New("bad request when trying to create a file")
	ErrCreateSessionBadRequest = errors.New("bad request when trying to create a new session")
	ErrStopKernelBadRequest    = errors.New("bad request when trying to stop a kernel")

	ErrCreateFileUnknownFailure    = errors.New("the 'create file' operation failed for an unknown or unexpected reason")
	ErrCreateSessionUnknownFailure = errors.New("the 'create session' operation failed for an unknown or unexpected reason")
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

type KernelManager struct {
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

func NewKernelManager(opts *domain.Configuration, atom *zap.AtomicLevel) *KernelManager {
	manager := &KernelManager{
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
func (m *KernelManager) StartNew(kernelSpec string) error {
	return nil
}

// Create a new session.
func (m *KernelManager) CreateSession(localSessionId string, sessionName string, path string, sessionType string, kernelSpecName string) error {
	// First, create the notebook file.
	// err := m.CreateFile(filepath.Join("./", path))
	// if err != nil {
	// 	m.logger.Error("Failed to create notebook file while creating new session.", zap.String("session-id", id), zap.Error(err))
	// 	return err
	// }

	requestBody := newJupyterSessionForRequest(sessionName, path, sessionType, kernelSpecName)

	requestBodyJson, err := json.Marshal(&requestBody)
	if err != nil {
		m.logger.Error("Error encountered while marshalling payload for CreateSession operation.", zap.Error(err))
		return err
	}

	m.logger.Debug("Issuing 'CREATE-SESSION' request now.", zap.String("request-args", requestBody.String()))

	url := fmt.Sprintf("http://%s/api/sessions", m.jupyterServerAddress)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyJson))
	if err != nil {
		m.logger.Error("Error encountered while creating request for CreateFile operation.", zap.String("request-args", requestBody.String()), zap.String("path", path), zap.String("url", url), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when creating new session.", zap.String("request-args", requestBody.String()), zap.String("local-session-id", localSessionId), zap.String("url", url), zap.Error(err))
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusCreated:
		{
			var jupyterSession *jupyterSession
			json.Unmarshal(body, &jupyterSession)
			m.logger.Debug("Received 'Created' when creating session", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", jupyterSession.String()))

			var kernelId string = jupyterSession.JupyterKernel.Id
			var jupyterSessionId string = jupyterSession.JupyterSessionId

			// Update all of our many mappings...
			m.localSessionIdToKernelId[localSessionId] = kernelId
			m.localSessionIdToJupyterSessionId[localSessionId] = jupyterSessionId
			m.jupyterSessionIdLocalSessionIdTo[jupyterSessionId] = localSessionId
			m.kernelIdToJupyterSessionId[kernelId] = jupyterSessionId
			m.kernelIdToLocalSessionId[kernelId] = localSessionId
			m.jupyterSessionIdToKernelId[jupyterSessionId] = kernelId

			// Connect to the Session and to the associated kernel.
			sessionConnection := NewSessionConnection(jupyterSession, m.jupyterServerAddress, m.atom)

			m.sessionMap[localSessionId] = sessionConnection

			m.logger.Debug("Successfully created and setup session.", zap.String("local-session-id", localSessionId), zap.String("jupyter-session-id", jupyterSessionId), zap.String("kernel-id", kernelId))
			m.logger.Debug("Issuing 'RequestKernelInfo' now.", zap.String("local-session-id", localSessionId), zap.String("jupyter-session-id", jupyterSessionId), zap.String("kernel-id", kernelId))

			numTries := 0
			maxNumTries := 3

			for numTries < maxNumTries {
				m.logger.Debug("Requesting kernel info now", zap.Int("attempt", numTries+1), zap.String("local-session-id", localSessionId), zap.String("jupyter-session-id", jupyterSessionId), zap.String("kernel-id", kernelId))
				resp, err := sessionConnection.RequestKernelInfo()

				if err != nil {
					m.logger.Error("Error when issuing `RequestKernelInfo`", zap.String("local-session-id", localSessionId), zap.String("jupyter-session-id", jupyterSessionId), zap.String("kernel-id", kernelId), zap.Error(err))

					numTries += 1

					if numTries < maxNumTries {
						time.Sleep(time.Second * time.Duration(numTries))
					} else {
						return err
					}
				} else {
					m.logger.Debug("Received response for `RequestKernelInfo`", zap.String("local-session-id", localSessionId), zap.String("jupyter-session-id", jupyterSessionId), zap.String("kernel-id", kernelId), zap.String("response", resp.String()))
				}
			}
		}
	case http.StatusBadRequest:
		{
			var responseJson map[string]interface{}
			json.Unmarshal(body, &responseJson)
			m.logger.Error("Received HTTP 400 'Bad Request' when creating session", zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", body))
			return fmt.Errorf("ErrCreateSessionBadRequest %w : %s", ErrCreateSessionBadRequest, string(body))
		}
	default:
		var responseJson map[string]interface{}
		json.Unmarshal(body, &responseJson)
		m.logger.Warn("Unexpected respone status code when creating a new session.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.Any("headers", resp.Header), zap.Any("body", body), zap.String("request-args", requestBody.String()))

		return fmt.Errorf("ErrCreateSessionUnknownFailure %w: %s", ErrCreateSessionUnknownFailure, string(body))
	}

	m.metrics.SessionCreated()
	// TODO(Ben): Does this also create a new kernel?
	return nil
}

func (m *KernelManager) CreateFile(path string) error {
	url := fmt.Sprintf("http://%s/api/contents/%s", m.jupyterServerAddress, path)

	createFileRequest := newCreateFileRequest(path)
	payload, err := json.Marshal(&createFileRequest)
	if err != nil {
		m.logger.Error("Error encountered while marshalling payload for CreateFile operation.", zap.Error(err))
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
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

func (m *KernelManager) StopKernel(id string) error {
	url := fmt.Sprintf("http://%s/api/sessions/%s", m.jupyterServerAddress, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		m.logger.Error("Failed to create DeleteSession request while stopping kernel.", zap.String("session-id", id), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when stopping session.", zap.String("session-id", id), zap.String("url", url), zap.Error(err))
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

func (m *KernelManager) GetMetrics() *KernelManagerMetrics {
	return m.metrics
}
