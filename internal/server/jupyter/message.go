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

type KernelMessage struct {
	Header       *KernelMessageHeader   `json:"header"`
	Channel      KernelSocketChannel    `json:"channel"`
	Content      map[string]interface{} `json:"content"`
	Buffers      []byte                 `json:"buffers"`
	Metadata     map[string]interface{} `json:"metadata"`
	ParentHeader *KernelMessageHeader   `json:"parent_header"`
}

type KernelSocketChannel string

func (m *KernelMessage) String() string {
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
