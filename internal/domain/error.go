package domain

import "encoding/json"

// Used to pass errors back to another window.
type ErrorHandler interface {
	HandleError(error, string)
}

type ErrorMessage struct {
	Description  string `json:"Description"`  // Provides additional context for what occurred; written by us.
	ErrorMessage string `json:"ErrorMessage"` // The value returned by err.Error() for whatever error occurred.
	Valid        bool   `json:"Valid"`        // Used to determine if the struct was sent/received correctly over the network.
}

func (m *ErrorMessage) String() string {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(out)
}
