package jupyter

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type KernelConnection struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
}

func NewKernelConnection(atom *zap.AtomicLevel) *KernelConnection {
	conn := &KernelConnection{}

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, atom)
	conn.logger = zap.New(core)

	conn.sugaredLogger = conn.logger.Sugar()

	return conn
}
