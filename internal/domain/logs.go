package domain

import "encoding/json"

type GetLogsRequest struct {
	Op        string `json:"op"`
	MessageId string `json:"msg_id"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Follow    bool   `json:"follow"`
}

func (r *GetLogsRequest) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(out)
}
