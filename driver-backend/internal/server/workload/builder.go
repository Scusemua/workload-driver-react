package workload

import (
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/zhangjyr/hashmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Builder is the builder for the Workload struct.
type Builder struct {
	id                        string
	workloadName              string
	seed                      int64
	debugLoggingEnabled       bool
	timescaleAdjustmentFactor float64
	sessionsSamplePercentage  float64
	remoteStorageDefinition   *proto.RemoteStorageDefinition
	atom                      *zap.AtomicLevel
}

// NewBuilder creates a new Builder instance.
func NewBuilder(atom *zap.AtomicLevel) *Builder {
	return &Builder{
		atom:                      atom,
		seed:                      -1,
		debugLoggingEnabled:       true,
		sessionsSamplePercentage:  1.0,
		timescaleAdjustmentFactor: 1.0,
	}
}

// SetID sets the ID for the workload.
func (b *Builder) SetID(id string) *Builder {
	b.id = id
	return b
}

// SetWorkloadName sets the name for the workload.
func (b *Builder) SetWorkloadName(workloadName string) *Builder {
	b.workloadName = workloadName
	return b
}

// SetSeed sets the seed value for the workload.
func (b *Builder) SetSeed(seed int64) *Builder {
	b.seed = seed
	return b
}

// EnableDebugLogging enables or disables debug logging.
func (b *Builder) EnableDebugLogging(enabled bool) *Builder {
	b.debugLoggingEnabled = enabled
	return b
}

// SetTimescaleAdjustmentFactor sets the timescale adjustment factor.
func (b *Builder) SetTimescaleAdjustmentFactor(factor float64) *Builder {
	b.timescaleAdjustmentFactor = factor
	return b
}

// SetSessionsSamplePercentage sets the sessions sample percentage.
func (b *Builder) SetSessionsSamplePercentage(percentage float64) *Builder {
	b.sessionsSamplePercentage = percentage
	return b
}

// SetRemoteStorageDefinition sets the remote storage definition.
func (b *Builder) SetRemoteStorageDefinition(def *proto.RemoteStorageDefinition) *Builder {
	b.remoteStorageDefinition = def
	return b
}

// Build creates a Workload instance with the specified values.
func (b *Builder) Build() *BasicWorkload {
	workload := &BasicWorkload{
		Id:                        b.id, // Same ID as the driver.
		Name:                      b.workloadName,
		Seed:                      b.seed,
		DebugLoggingEnabled:       b.debugLoggingEnabled,
		TimescaleAdjustmentFactor: b.timescaleAdjustmentFactor,
		WorkloadType:              UnspecifiedWorkload,
		atom:                      b.atom,
		sessionsMap:               hashmap.New(32),
		trainingStartedTimes:      hashmap.New(32),
		SumTickDurationsMillis:    0,
		TickDurationsMillis:       make([]int64, 0),
		RemoteStorageDefinition:   b.remoteStorageDefinition,
		SampledSessions:           make(map[string]interface{}),
		UnsampledSessions:         make(map[string]interface{}),
		Statistics:                NewStatistics(b.sessionsSamplePercentage),
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), b.atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	workload.logger = logger
	workload.sugaredLogger = logger.Sugar()

	return workload
}
