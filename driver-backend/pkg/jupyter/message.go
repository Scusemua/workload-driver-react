package jupyter

import (
	"encoding/json"
)

const (
	ShellChannel   KernelSocketChannel = "shell"
	ControlChannel KernelSocketChannel = "control"
	IOPubChannel   KernelSocketChannel = "iopub"
	StdinChannel   KernelSocketChannel = "stdin"
)

type KernelSocketChannel string

func (s KernelSocketChannel) String() string {
	return string(s)
}

type KernelMessage interface {
	GetHeader() *KernelMessageHeader
	GetChannel() KernelSocketChannel
	GetContent() interface{}
	GetBuffers() [][]byte
	GetMetadata() map[string]interface{}
	GetParentHeader() *KernelMessageHeader
	DecodeContent() (map[string]interface{}, error)

	// AddMetadata adds metadata with the given key and value to the underlying message.
	//
	// If there already exists an entry with the given key, then that entry is silently overwritten.
	AddMetadata(key string, value interface{})

	String() string
}

// ResourceSpec can be passed within a jupyterSessionReq when creating a new Session or Kernel.
type ResourceSpec struct {
	Cpu  int     `json:"cpu"`    // In millicpus (1/1000th CPU core)
	Mem  float64 `json:"memory"` // In MB
	Gpu  int     `json:"gpu"`
	Vram float64 `json:"vram"` // In GB
}

func (s *ResourceSpec) String() string {
	m, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	return string(m)
}

type BaseKernelMessage struct {
	Channel      KernelSocketChannel    `json:"channel"`
	Header       *KernelMessageHeader   `json:"header"`
	ParentHeader *KernelMessageHeader   `json:"parent_header"`
	Metadata     map[string]interface{} `json:"metadata"`
	Content      interface{}            `json:"content"`
	Buffers      [][]byte               `json:"buffers"`
}

func (m *BaseKernelMessage) GetHeader() *KernelMessageHeader {
	return m.Header
}

func (m *BaseKernelMessage) DecodeContent() (map[string]interface{}, error) {
	var content map[string]interface{}
	err := json.Unmarshal(m.Content.([]byte), &content)
	return content, err
}

func (m *BaseKernelMessage) GetChannel() KernelSocketChannel {
	return m.Channel
}

func (m *BaseKernelMessage) GetContent() interface{} {
	return m.Content
}

func (m *BaseKernelMessage) GetBuffers() [][]byte {
	return m.Buffers
}

// AddMetadata adds metadata with the given key and value to the underlying message.
//
// If there already exists an entry with the given key, then that entry is silently overwritten.
func (m *BaseKernelMessage) AddMetadata(key string, value interface{}) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}

	m.Metadata[key] = value
}

func (m *BaseKernelMessage) GetMetadata() map[string]interface{} {
	return m.Metadata
}

func (m *BaseKernelMessage) GetParentHeader() *KernelMessageHeader {
	return m.ParentHeader
}

func (m *BaseKernelMessage) String() string {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type KernelMessageHeader struct {
	Date        string      `json:"date"`
	MessageId   string      `json:"msg_id"`
	MessageType MessageType `json:"msg_type"`
	Session     string      `json:"session"`
	Username    string      `json:"username"`
	Version     string      `json:"version"`
}

func (h *KernelMessageHeader) String() string {
	out, err := json.Marshal(h)
	if err != nil {
		panic(err)
	}

	return string(out)
}
