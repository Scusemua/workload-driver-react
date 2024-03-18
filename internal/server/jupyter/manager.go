package jupyter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type createSessionArgs struct {
	Id          string                  `json:"id"`
	Path        string                  `json:"path"`
	Name        string                  `json:"name"`
	SessionType string                  `json:"type"`
	KernelArgs  createSessionKernelArgs `json:"kernel"`
}

type createSessionKernelArgs struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	// connections     int    `json:"connections"`
	// execution_state string `json:"execution_state"`
	// last_activity   string `json:"last_activity"`
}

func newCreateSessionArgs(id string, path string, name string, sessionType string) *createSessionArgs {
	kernelArgs := newCreateSessionKernelArgs(id, name)

	return &createSessionArgs{
		Id:          id,
		Path:        path,
		Name:        name,
		SessionType: sessionType,
		KernelArgs:  *kernelArgs,
	}
}

func newCreateSessionKernelArgs(id string, name string) *createSessionKernelArgs {
	return &createSessionKernelArgs{
		Id:   id,
		Name: name,
	}
}

type KernelManagerMetrics struct {
	NumFilesCreated       int `json:"num-files-created"`
	NumKernelsCreated     int `json:"num-kernels-created`
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

	jupyterServerAddress string

	metrics *KernelManagerMetrics
}

func NewKernelManager(opts *domain.Configuration, atom *zap.AtomicLevel) *KernelManager {
	manager := &KernelManager{
		jupyterServerAddress: opts.JupyterServerAddress,
		metrics:              &KernelManagerMetrics{},
	}

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	manager.logger = zap.New(core)

	manager.sugaredLogger = manager.logger.Sugar()

	return manager
}

// Start a new kernel.
func (m *KernelManager) StartNew(kernelSpec string) error {
	return nil
}

// Create a new session.
func (m *KernelManager) CreateSession(id string, name string, path string, sessionType string) error {
	// First, create the notebook file.
	err := m.CreateFile(path)
	if err != nil {
		m.logger.Error("Failed to create notebook file while creating new session.", zap.String("session-id", id), zap.Error(err))
		return err
	}

	requestBody := newCreateSessionArgs(id, path, name, sessionType)

	requestBodyJson, err := json.Marshal(&requestBody)
	if err != nil {
		m.logger.Error("Error encountered while marshalling payload for CreateSession operation.", zap.Error(err))
	}

	url := fmt.Sprintf("%s/api/sessions", m.jupyterServerAddress)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyJson))
	if err != nil {
		m.logger.Error("Error encountered while creating request for CreateFile operation.", zap.String("path", path), zap.String("url", url), zap.Error(err))
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("Received error when creating new session.", zap.String("session-id", id), zap.String("url", url), zap.Error(err))
		return err
	}

	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response Body:", string(body))

	m.metrics.SessionCreated()
	// TODO(Ben): Does this also create a new kernel?
	return nil
}

func (m *KernelManager) CreateFile(path string) error {
	url := fmt.Sprintf("%s/api/contents/%s", m.jupyterServerAddress, path)
	var payload = []byte(fmt.Sprintf(`"{"path":"%s"}`, path))
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

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response Body:", string(body))

	m.metrics.FileCreated()
	return nil
}

func (m *KernelManager) StopKernel(id string) error {
	url := fmt.Sprintf("%s/api/sessions/%s", m.jupyterServerAddress, id)
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

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response Body:", string(body))

	m.metrics.SessionTerminated()
	// TODO(Ben): Does this also terminate the kernel?
	return nil
}

func (m *KernelManager) GetMetrics() *KernelManagerMetrics {
	return m.metrics
}
