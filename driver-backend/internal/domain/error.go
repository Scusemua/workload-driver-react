package domain

import (
	"encoding/json"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

const (
	ResponseStatusError string = "ERROR"
	ResponseStatusOK    string = "OK"
)

var (
	ErrUnknownSession = errors.New("received 'training-started' or 'training-ended' event for unknown session")
)

// ErrorHandler is used to pass errors back to another window.
type ErrorHandler interface {
	HandleError(error, string)
}

type ErrorMessage struct {
	Description  string `json:"Description"`  // Provides additional context for what occurred; written by us.
	ErrorMessage string `json:"ErrorMessage"` // The value returned by err.Error() for whatever error occurred.
	Valid        bool   `json:"Valid"`        // Used to determine if the struct was sent/received correctly over the network.
	Operation    string `json:"op"`           // The original operation of the request to which this error is being sent as a response.
	Status       string `json:"status"`       // ERROR.
	MessageId    string `json:"msg_id"`       // Corresponding MessageId, if applicable (such as when sending/receiving JSON WebSocket messages).
}

func (m *ErrorMessage) Encode() []byte {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return out
}

func (m *ErrorMessage) String() string {
	out := m.Encode()
	return string(out)
}

// GRPCStatusToHTTPStatus converts a gRPC error to the corresponding HTTP status code
func GRPCStatusToHTTPStatus(err error) int {
	// Get the gRPC status from the error
	st, ok := status.FromError(err)
	if !ok {
		// If it's not a gRPC error, return internal server error as default
		return http.StatusInternalServerError
	}

	// Switch on the gRPC status code and map to corresponding HTTP status code
	switch st.Code() {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusRequestedRangeNotSatisfiable
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
