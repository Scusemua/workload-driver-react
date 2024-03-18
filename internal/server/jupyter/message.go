package jupyter

import "encoding/json"

type KernelMessage struct {
	Date        string `json:"date"`
	MessageId   string `json:"msg_id"`
	MessageType string `json:"msg_type"`
	Session     string `json:"session"`
	Username    string `json:"username"`
	Version     string `json:"version"`
}

func (m *KernelMessage) String() string {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(out)
}
