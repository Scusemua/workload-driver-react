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
	GetBuffers() []byte
	GetMetadata() map[string]interface{}
	GetParentHeader() *KernelMessageHeader
	DecodeContent() (map[string]interface{}, error)

	String() string
}

// ResourceSpec can be passed within a jupyterSessionReq when creating a new Session or Kernel.
type ResourceSpec struct {
	Cpu int     `json:"cpu"`    // In millicpus (1/1000th CPU core)
	Mem float64 `json:"memory"` // In MB
	Gpu int     `json:"gpu"`
}

type baseKernelMessage struct {
	Channel      KernelSocketChannel    `json:"channel"`
	Header       *KernelMessageHeader   `json:"header"`
	ParentHeader *KernelMessageHeader   `json:"parent_header"`
	Metadata     map[string]interface{} `json:"metadata"`
	Content      interface{}            `json:"content"`
	Buffers      []byte                 `json:"buffers"`
}

func (m *baseKernelMessage) GetHeader() *KernelMessageHeader {
	return m.Header
}

func (m *baseKernelMessage) DecodeContent() (map[string]interface{}, error) {
	var content map[string]interface{}
	err := json.Unmarshal(m.Content.([]byte), &content)
	return content, err
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

func (m *baseKernelMessage) String() string {
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

// type KernelMessageBuilder interface {
// 	WithChannel(channel KernelSocketChannel) KernelMessageBuilder
// 	WithContent(content interface{}) KernelMessageBuilder
// 	WithMessageType(messageType string) KernelMessageBuilder
// 	BuildMessage() KernelMessage
// }

// type kernelMessageBuilderImpl struct {
// 	channel      KernelSocketChannel
// 	header       *KernelMessageHeader
// 	parentHeader *KernelMessageHeader
// 	metadata     map[string]interface{}
// 	content      interface{}
// 	buffers      []byte

// 	// For header.
// 	messageId   string
// 	session     string
// 	username    string
// 	version     string
// 	messageType string
// }

// func newKernelMessageBuilder(messageId string) KernelMessageBuilder {
// 	return &kernelMessageBuilderImpl{}
// }

// func (b *kernelMessageBuilderImpl) WithChannel(channel KernelSocketChannel) *kernelMessageBuilderImpl {
// 	b.channel = channel
// 	return b
// }

// // This doesn't do any sort of type checking on the content.
// // Content should probably just be a map[string]interface{} for a majority of cases.
// func (b *kernelMessageBuilderImpl) WithContent(content interface{}) *kernelMessageBuilderImpl {
// 	b.content = content
// 	return b
// }

// func (b *kernelMessageBuilderImpl) WithMessageType(messageType string) *kernelMessageBuilderImpl {
// 	b.messageType = messageType
// 	return b
// }

// func (b *kernelMessageBuilderImpl) BuildMessage() KernelMessage {
// 	date := time.Now().UTC().Format(JavascriptISOString)

// 	header := &KernelMessageHeader{
// 		Date:        date,
// 		MessageId:   b.messageId,
// 		MessageType: b.messageType,
// 		Session:     b.session,
// 		Username:    b.username,
// 		Version:     b.version,
// 	}

// 	if b.content == nil {
// 		b.content = make(map[string]interface{})
// 	}

// 	metadata := make(map[string]interface{})

// 	message := &baseKernelMessage{
// 		Channel:      b.channel,
// 		Header:       header,
// 		Content:      b.content,
// 		Metadata:     metadata,
// 		Buffers:      make([]byte, 0),
// 		ParentHeader: &KernelMessageHeader{},
// 	}

// 	responseChannel := make(chan KernelMessage)
// 	conn.responseChannels[messageId] = responseChannel
// }
