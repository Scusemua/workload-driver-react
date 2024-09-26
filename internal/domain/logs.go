package domain

import (
	"encoding/json"
	"go.uber.org/zap"
)

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

// LogErrorWithoutStacktrace calls the Error method of the given zap.Logger after configuring the logger
// to only add stack traces on panic-level messages.
func LogErrorWithoutStacktrace(logger *zap.Logger, msg string, fields ...zap.Field) {
	logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).Error(msg, fields...)
}

// LogSugaredErrorWithoutStacktrace calls the Error method of the given zap.SugaredLogger after configuring the logger
// to only add stack traces on panic-level messages.
func LogSugaredErrorWithoutStacktrace(logger *zap.SugaredLogger, args ...interface{}) {
	logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).Error(args...)
}
