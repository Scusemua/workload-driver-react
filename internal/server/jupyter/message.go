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

type KernelMessage interface {
	GetHeader() *KernelMessageHeader
	GetChannel() KernelSocketChannel
	GetContent() interface{}
	GetBuffers() []byte
	GetMetadata() map[string]interface{}
	GetParentHeader() *KernelMessageHeader

	String() string
}

type baseKernelMessage struct {
	Header       *KernelMessageHeader   `json:"header"`
	Channel      KernelSocketChannel    `json:"channel"`
	Content      interface{}            `json:"content"`
	Buffers      []byte                 `json:"buffers"`
	Metadata     map[string]interface{} `json:"metadata"`
	ParentHeader *KernelMessageHeader   `json:"parent_header"`
}

func (m *baseKernelMessage) GetHeader() *KernelMessageHeader {
	return m.Header
}

func (m *baseKernelMessage) GetChannel() KernelSocketChannel {
	return m.Channel
}

func (m *baseKernelMessage) GetContent() interface{} {
	return m.Content
}

func (m *baseKernelMessage) GetBuffers() []byte {
	return m.Buffers
}

func (m *baseKernelMessage) GetMetadata() map[string]interface{} {
	return m.Metadata
}

func (m *baseKernelMessage) GetParentHeader() *KernelMessageHeader {
	return m.ParentHeader
}

type KernelSocketChannel string

func (m *baseKernelMessage) String() string {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type KernelMessageHeader struct {
	Date        string `json:"date"`
	MessageId   string `json:"msg_id"`
	MessageType string `json:"msg_type"`
	Session     string `json:"session"`
	Username    string `json:"username"`
	Version     string `json:"version"`
}

func (h *KernelMessageHeader) String() string {
	out, err := json.Marshal(h)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type executeRequestKernelMessageContent struct {
	Code            string                 `json:"code"`             // The code to execute.
	Silent          bool                   `json:"silent"`           // Whether to execute the code as quietly as possible. The default is `false`.
	StoreHistory    bool                   `json:"store_history"`    // Whether to store history of the execution. The default `true` if silent is False. It is forced to  `false ` if silent is `true`.
	UserExpressions map[string]interface{} `json:"user_expressions"` // A mapping of names to expressions to be evaluated in the kernel's interactive namespace.
	AllowStdin      bool                   `json:"allow_stdin"`      // Whether to allow stdin requests. The default is `true`.
	StopOnError     bool                   `json:"stop_on_error"`    // Whether to the abort execution queue on an error. The default is `false`.
}

func (m *executeRequestKernelMessageContent) String() string {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(out)
}
