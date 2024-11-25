package generator

import (
	"errors"
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

/*
 * This file contains a collection of various "custom event sequences."
 * These are similar to the XML-based event sequences; however, they are not defined using XML.
 * Instead, they are defined programmatically in Golang, using the API of the Custom Event Sequencer.
 */

var (
	ErrInvalidConfiguration = errors.New("invalid configuration specified")
)

// SequencerFunction defines a function that, when called and passed a pointer to a CustomEventSequencer,
// will use the CustomEventSequencer API to create an executable workload trace.
type SequencerFunction func(sequencer *CustomEventSequencer, logger *zap.Logger) error

// SessionArguments is a utility/helper struct to specify arguments of a Session that should be registered with a CustomEventSequencer.
type SessionArguments struct {
	Id               string
	MaxCPUs          float64
	MaxMemoryGB      float64
	MaxGPUs          int
	SessionStartTick int
	StopTick         int
}

func NewSessionArguments(sessionId string, maxCPUs float64, maxMemoryGB float64, maxGPUs int, startTick int, stopTick int) *SessionArguments {
	return &SessionArguments{
		Id:               sessionId,
		MaxGPUs:          maxGPUs,
		MaxCPUs:          maxCPUs,
		MaxMemoryGB:      maxMemoryGB,
		SessionStartTick: startTick,
		StopTick:         stopTick,
	}
}

// TrainingResourceUtilizationArgs is a utility/helper struct to specify the resource utilization during a training event/task.
type TrainingResourceUtilizationArgs struct {
	CpuUtilization float64   // CPU utilization; to be in the interval [0, 100]
	MemoryUsageGB  float64   // Memory utilization in gigabytes; must be >= 0
	GpuUtilization []float64 // Aggregate GPU utilization, such that 100 = 1 GPU, and (e.g.,) 800 = 8 GPUs. Should be within interval [0, 100]

	// Used by the 'WithGpuUtilization' API to set the GPU utilization for a specific GPU.
	// Each time the 'WithGpuUtilization' function is called, this value is incremented,
	// so that the next call to WithGpuUtilization will update the GPU utilization for the next GPU.
	gpuIndex int
}

func NewTrainingResourceUtilizationArgs(cpuUtil float64, memUsageGb float64, numGPUs int) *TrainingResourceUtilizationArgs {
	if numGPUs < 0 {
		panic(fmt.Sprintf("Invalid specified number of GPUs: %d. Number of GPUs must be greater than or equal to 0.", numGPUs))
	}

	return &TrainingResourceUtilizationArgs{
		CpuUtilization: cpuUtil,
		MemoryUsageGB:  memUsageGb,
		GpuUtilization: make([]float64, 0, numGPUs),
		gpuIndex:       0,
	}
}

func (a *TrainingResourceUtilizationArgs) NumGPUs() int {
	return cap(a.GpuUtilization)
}

// WithGpuUtilization sets the GPU utilization of the next GPU that has not already been specified via the 'WithGpuUtilization' function.
//
// The first call to 'WithGpuUtilization' will set the GPU utilization of GPU 0.
// The second call to 'WithGpuUtilization' will set the GPU utilization of GPU 1.
// This continues until all GPUs have been specified.
//
// If 'WithGpuUtilization' is called again after specifying the utilization of all GPUs, then the function will panic.
//
// This modifies the 'TrainingResourceUtilizationArgs' struct on which it was called in-place; it also returns the TrainingResourceUtilizationArgs struct.
func (a *TrainingResourceUtilizationArgs) WithGpuUtilization(gpuUtil float64) *TrainingResourceUtilizationArgs {
	if gpuUtil < 0 {
		panic(fmt.Sprintf("Invalid GPU utilization specified: %f. Value must be greater than or equal to 0.", gpuUtil))
	}

	if a.gpuIndex >= a.NumGPUs() {
		panic(fmt.Sprintf("The GPU utilization of all GPUs (total of %d) has already been specified.", a.NumGPUs()))
	}

	a.GpuUtilization[a.gpuIndex] = gpuUtil
	a.gpuIndex += 1

	return a
}

// WithGpuUtilizationForSpecificGpu sets the GPU utilization of a specific GPU (identified by its localIndex, which should range from 0 to NUM_GPUS - 1).
//
// This modifies the 'TrainingResourceUtilizationArgs' struct on which it was called in-place; it also returns the TrainingResourceUtilizationArgs struct.
func (a *TrainingResourceUtilizationArgs) WithGpuUtilizationForSpecificGpu(gpuIndex int, gpuUtil float64) *TrainingResourceUtilizationArgs {
	if gpuIndex < 0 || gpuIndex > a.NumGPUs() {
		panic(fmt.Sprintf("Invalid GPU localIndex specified: %d. Value must be greater than or equal to 0 and less than the total number of GPUs (%d).", gpuIndex, cap(a.GpuUtilization)))
	}

	if gpuUtil < 0 {
		panic(fmt.Sprintf("Invalid GPU utilization specified: %f. Value must be greater than or equal to 0.", gpuUtil))
	}

	a.GpuUtilization[gpuIndex] = gpuUtil

	return a
}

func validateSession(session *domain.WorkloadTemplateSession) error {
	if session == nil {
		panic("Session should not be nil.")
	}

	if session.GetTrainings() == nil {
		panic("Session's `Trainings` field should not be nil.")
	}

	if session.GetMaxResourceRequest().Cpus < 0 {
		return fmt.Errorf("%w: invalid maximum number of CPUs specified (%f). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, session.GetMaxResourceRequest().Cpus)
	}

	if session.GetMaxResourceRequest().Gpus < 0 {
		return fmt.Errorf("%w: invalid maximum number of GPUs specified (%d). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, session.GetMaxResourceRequest().Gpus)
	}

	if session.GetMaxResourceRequest().MemoryMB < 0 {
		return fmt.Errorf("%w: invalid maximum memory usage (in MB) specified (%f). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, session.GetMaxResourceRequest().MemoryMB)
	}

	// Validate `session.GetStartTick()`
	if session.GetStartTick() < 0 {
		return fmt.Errorf("%w: invalid starting tick specified: %d. Must be greater than or equal to 0", ErrInvalidConfiguration, session.GetStartTick())
	}

	if len(session.GetId()) == 0 {
		return fmt.Errorf("%w: invalid session ID specified (\"%s\"). The Session ID cannot be the empty string", ErrInvalidConfiguration, session.GetId())
	}

	// Validate `session.GetStartTick()` and `sessionTerminateTick` arguments
	if session.GetStartTick() > session.GetStopTick() {
		return fmt.Errorf("%w: specified 'start-tick' (%d) occurs after specified 'start-terminated' tick (%d). Session must be started before it may be terminated", ErrInvalidConfiguration, session.GetStartTick(), session.GetStopTick())
	}

	// If the session trains at least once, then verify that the first and last training are within the bounds of the session.
	if len(session.GetTrainings()) >= 1 {
		firstTraining := session.GetTrainings()[0]
		startTrainingTick := firstTraining.StartTick

		if session.GetStartTick() > startTrainingTick {
			return fmt.Errorf("%w: specified 'start-tick' (%d) occurs after specified 'start-training' tick (%d) for session's first training event. Session must be started before it may begin training", ErrInvalidConfiguration, session.GetStartTick(), startTrainingTick)
		}

		// Validate `startTrainingTick` and `sessionTerminateTick` arguments
		if startTrainingTick > session.GetStopTick() {
			return fmt.Errorf("%w: specified 'start-training' (%d) occurs after specified 'start-terminated' tick (%d) for session's first training event. Session cannot start training after it has been terminated", ErrInvalidConfiguration, startTrainingTick, session.GetStopTick())
		}

		if (startTrainingTick + firstTraining.DurationInTicks) > session.GetStopTick() {
			return fmt.Errorf("%w: session's first training would conclude after the session is supposed to terminate [SessionStart: %d, TrainingStart: %d, TrainingDuration: %d, SessionEnd: %d]", ErrInvalidConfiguration, session.GetStartTick(), startTrainingTick, firstTraining.DurationInTicks, session.GetStopTick())
		}
	}

	// Also check the last training, if there's at least 2 training events.
	if len(session.GetTrainings()) >= 2 {
		lastTraining := session.GetTrainings()[len(session.GetTrainings())-1]
		startTrainingTick := lastTraining.StartTick

		if session.GetStartTick() > startTrainingTick {
			return fmt.Errorf("%w: specified 'start-tick' (%d) occurs after specified 'start-training' tick (%d) for session's final training event. Session must be started before it may begin training", ErrInvalidConfiguration, session.GetStartTick(), startTrainingTick)
		}

		// Validate `startTrainingTick` and `sessionTerminateTick` arguments
		if startTrainingTick > session.GetStopTick() {
			return fmt.Errorf("%w: specified 'start-training' (%d) occurs after specified 'start-terminated' tick (%d) for session's final training event. Session cannot start training after it has been terminated", ErrInvalidConfiguration, startTrainingTick, session.GetStopTick())
		}

		if (startTrainingTick + lastTraining.DurationInTicks) > session.GetStopTick() {
			return fmt.Errorf("%w: session's final training would conclude after the session is supposed to terminate [SessionStart: %d, TrainingStart: %d, TrainingDuration: %d, SessionEnd: %d]", ErrInvalidConfiguration, session.GetStartTick(), startTrainingTick, lastTraining.DurationInTicks, session.GetStopTick())
		}
	}

	return nil
}

func validateSessionArgumentsAgainstTrainingArguments(session *domain.WorkloadTemplateSession) error {
	if session == nil {
		panic("Session cannot be nil.")
	}

	if session.GetTrainings() == nil {
		panic("Session's `Trainings` field cannot be nil.")
	}

	for _, trainingEvent := range session.GetTrainings() {
		if session.GetMaxResourceRequest().Cpus < trainingEvent.Millicpus {
			return fmt.Errorf("%w: incompatible max CPUs (%f) and training CPU utilization (%f) specified. Training CPU utilization cannot exceed maximum session CPUs", ErrInvalidConfiguration, session.GetMaxResourceRequest().Cpus, trainingEvent.Millicpus)
		}

		if session.GetMaxResourceRequest().Gpus < trainingEvent.NumGPUs() {
			return fmt.Errorf("%w: incompatible max GPUs (%d) and training GPU utilization (%d) specified. Training GPU utilization cannot exceed maximum session GPUs", ErrInvalidConfiguration, session.GetMaxResourceRequest().Gpus, trainingEvent.NumGPUs())
		}

		if session.GetMaxResourceRequest().MemoryMB < trainingEvent.MemUsageMB {
			return fmt.Errorf("%w: incompatible max memory usage (%f MB) and training memory usage (%f GB) specified. Training memory usage cannot exceed maximum session memory usage", ErrInvalidConfiguration, session.GetMaxResourceRequest().MemoryMB, trainingEvent.MemUsageMB)
		}
	}

	return nil
}

// ManySessionsManyTrainingEvents is the default "generator function" to produce a workload from a template.
// There used to be a SingleSessionSingleTraining function, but it was removed as its use was eclipsed by
// the ManySessionsManyTrainingEvents function.
func ManySessionsManyTrainingEvents(sessions []*domain.WorkloadTemplateSession) (SequencerFunction, error) {
	if sessions == nil {
		panic("Session arguments cannot be nil.")
	}

	if len(sessions) == 0 {
		panic(fmt.Sprintf("Sessions has unexpected length: %d", len(sessions)))
	}

	var approximateFinalTick int64 = 0
	for _, session := range sessions {
		if err := validateSession(session); err != nil {
			return nil, err
		}

		if err := validateSessionArgumentsAgainstTrainingArguments(session); err != nil {
			return nil, err
		}

		if int64(session.StopTick) > approximateFinalTick {
			approximateFinalTick = int64(session.StopTick)
		}

		if len(session.GetTrainings()) == 0 {
			continue
		}

		trainingEvent := session.GetTrainings()[0]
		if trainingEvent.DurationInTicks <= 0 {
			return nil, fmt.Errorf("%w: invalid training duration specified: %d ticks. Must be strictly greater than 0", ErrInvalidConfiguration, trainingEvent.DurationInTicks)
		}

		finalTrainingEvent := session.GetTrainings()[len(session.GetTrainings())-1]
		if int64(finalTrainingEvent.StartTick+finalTrainingEvent.DurationInTicks) > approximateFinalTick {
			approximateFinalTick = int64(finalTrainingEvent.StartTick + finalTrainingEvent.DurationInTicks)
		}
	}

	return func(sequencer *CustomEventSequencer, log *zap.Logger) error {
		seenSessions := make(map[string]struct{})

		for _, session := range sessions {
			if _, ok := seenSessions[session.GetId()]; ok {
				log.Error("We've already added events for Session.", zap.String("session_id", session.Id))
				panic("Duplicate Session.")
			}
			seenSessions[session.GetId()] = struct{}{}

			sequencer.RegisterSession(session.GetId(), session.GetMaxResourceRequest().Cpus, session.GetMaxResourceRequest().MemoryMB, session.GetMaxResourceRequest().Gpus, session.GetMaxResourceRequest().VRAM, 0)
			sequencer.AddSessionStartedEvent(session.GetId(), session.GetStartTick(), 0, 0, 0, 1)

			for _, trainingEvent := range session.GetTrainings() {
				sequencer.AddTrainingEvent(session.GetId(), trainingEvent.StartTick, trainingEvent.DurationInTicks, trainingEvent.Millicpus, trainingEvent.MemUsageMB, trainingEvent.GpuUtil, trainingEvent.VRamUsageGB)
			}

			sequencer.AddSessionTerminatedEvent(session.GetId(), session.GetStopTick())
		}

		sequencer.eventConsumer.RegisterApproximateFinalTick(approximateFinalTick)

		sequencer.SubmitEvents(sequencer.eventConsumer.WorkloadEventGeneratorCompleteChan())
		return nil
	}, nil
}
