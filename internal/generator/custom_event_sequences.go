package generator

import (
	"errors"
	"fmt"
)

/*
 * This file contains a collection of various "custom event sequences."
 * These are similar to the XML-based event sequences; however, they are not defined using XML.
 * Instead, they are defined programmatically in Golang, using the API of the Custom Event Sequencer.
 */

var (
	ErrInvalidConfiguration error = errors.New("Invalid configuration specified")
)

// Defines a function that, when called and passed a pointer to a CustomEventSequencer,
// will use the CustomEventSequencer API to create an executable workload trace.
type SequencerFunction func(sequencer *CustomEventSequencer)

// Utility/helper struct to specify arguments of a Session that should be registered with a CustomEventSequencer.
type SessionRegistrationArguments struct {
	SessionID   string
	MaxCPUs     float64
	MaxMemoryGB float64
	MaxGPUs     int
}

func NewSessionRegistrationArguments(sessionId string, maxCPUs float64, maxMemoryGB float64, maxGPUs int) *SessionRegistrationArguments {
	return &SessionRegistrationArguments{
		SessionID:   sessionId,
		MaxGPUs:     maxGPUs,
		MaxCPUs:     maxCPUs,
		MaxMemoryGB: maxMemoryGB,
	}
}

// Utility/helper struct to specify the resource utilization during a training event/task.
type TrainingResourceUtilizationArgs struct {
	CpuUtilization float64 // CPU utilization; to be in the interval [0, 100]
	MemoryUsageGB  float64 // Memory utilization in gigabytes; must be >= 0
	GpuUtilization float64 // Aggregate GPU utilization, such that 100 = 1 GPU, and (e.g.,) 800 = 8 GPUs. Should be within interval [0, 100]
}

func NewTrainingResourceUtilizationArgs(cpuUtil float64, memUsageGb float64, gpuUtil float64) *TrainingResourceUtilizationArgs {
	return &TrainingResourceUtilizationArgs{
		CpuUtilization: cpuUtil,
		MemoryUsageGB:  memUsageGb,
		GpuUtilization: gpuUtil,
	}
}

func ValidateSessionArguments(sessionArgs *SessionRegistrationArguments) error {
	if sessionArgs.MaxCPUs < 0 {
		return fmt.Errorf("%w: invalid maximum number of CPUs specified (%f). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, sessionArgs.MaxCPUs)
	}

	if sessionArgs.MaxGPUs < 0 {
		return fmt.Errorf("%w: invalid maximum number of GPUs specified (%d). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, sessionArgs.MaxGPUs)
	}

	if sessionArgs.MaxMemoryGB < 0 {
		return fmt.Errorf("%w: invalid maximum memory usage (in GB) specified (%f). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, sessionArgs.MaxMemoryGB)
	}

	if len(sessionArgs.SessionID) == 0 {
		return fmt.Errorf("%w: invalid session ID specified (\"%s\"). The Session ID cannot be the empty string", ErrInvalidConfiguration, sessionArgs.SessionID)
	}

	return nil
}

func ValidateSessionArgumentsAgainstTrainingArguments(sessionArgs *SessionRegistrationArguments, trainingArgs *TrainingResourceUtilizationArgs) error {
	if sessionArgs.MaxCPUs < trainingArgs.CpuUtilization {
		return fmt.Errorf("%w: incompatible max CPUs (%f) and training CPU utilization (%f) specified. Training CPU utilization cannot exceed maximum session CPUs", ErrInvalidConfiguration, sessionArgs.MaxCPUs, trainingArgs.CpuUtilization)
	}

	if sessionArgs.MaxGPUs < int(trainingArgs.GpuUtilization) { // TODO: This is wrong based on how we're specifying max GPUs and training GPU utilization
		return fmt.Errorf("%w: incompatible max GPUs (%d) and training GPU utilization (%f) specified. Training GPU utilization cannot exceed maximum session GPUs", ErrInvalidConfiguration, sessionArgs.MaxGPUs, trainingArgs.GpuUtilization)
	}

	if sessionArgs.MaxMemoryGB < trainingArgs.MemoryUsageGB {
		return fmt.Errorf("%w: incompatible max memory usage (%f GB) and training memory usage (%f GB) specified. Training memory usage cannot exceed maximum session memory usage", ErrInvalidConfiguration, sessionArgs.MaxMemoryGB, trainingArgs.MemoryUsageGB)
	}

	return nil
}

// Create a training sequence involving a single Session that trains just once.
//
// The following quantites are configurable and are to be passed as arguments to this function (in this order):
// - session start time (>= 0)
// - training start time (> 'session start time')
// - training duration (> 0)
// - session terminate time (> 'training start time' + 'training duration')
// - the number of GPUs to use while training (> 0)
//
// This will return nil and an ErrInvalidConfiguration error if the arguments are invalid.
func SingleSessionSingleTraining(sessionArgs *SessionRegistrationArguments, sessionStartTick int, startTrainingTick int, trainingDurationInTicks int, sessionTerminateTick int, trainingArgs *TrainingResourceUtilizationArgs) (SequencerFunction, error) {
	if err := ValidateSessionArguments(sessionArgs); err != nil {
		return nil, err
	}

	if err := ValidateSessionArgumentsAgainstTrainingArguments(sessionArgs, trainingArgs); err != nil {
		return nil, err
	}

	// Validate `sessionStartTick` argument.
	if sessionStartTick < 0 {
		return nil, fmt.Errorf("%w: invalid starting tick specified: %d. Must be greater than or equal to 0", ErrInvalidConfiguration, sessionStartTick)
	}

	// Validate `sessionStartTick` and `startTrainingTick` arguments.
	if sessionStartTick > startTrainingTick {
		return nil, fmt.Errorf("%w: specified 'start-tick' (%d) occurs after specified 'start-training' tick (%d). Session must be started before it may begin training", ErrInvalidConfiguration, sessionStartTick, startTrainingTick)
	}

	// Validate `sessionStartTick` and `sessionTerminateTick` arguments.
	if sessionStartTick > sessionTerminateTick {
		return nil, fmt.Errorf("%w: specified 'start-tick' (%d) occurs after specified 'start-terminated' tick (%d). Session must be started before it may be terminated", ErrInvalidConfiguration, sessionStartTick, sessionTerminateTick)
	}

	// Validate `startTrainingTick` and `sessionTerminateTick` arguments.
	if startTrainingTick > sessionTerminateTick {
		return nil, fmt.Errorf("%w: specified 'start-training' (%d) occurs after specified 'start-terminated' tick (%d). Session cannot start training after it has been terminated", ErrInvalidConfiguration, startTrainingTick, sessionTerminateTick)
	}

	if trainingDurationInTicks <= 0 {
		return nil, fmt.Errorf("%w: invalid training duration specified: %d ticks. Must be strictly greater than 0", ErrInvalidConfiguration, trainingDurationInTicks)
	}

	// Validate `startTrainingTick`, `trainingDurationInTicks`, and `sessionTerminateTick` arguments.
	trainingEndTick := startTrainingTick + trainingDurationInTicks
	if trainingEndTick > sessionTerminateTick {
		panic(fmt.Sprintf("session instructed to begin training during tick %d for a total of %d tick(s), which means that training would end during tick %d; however, Session has been instructed to terminate during tick %d. Session must complete training before it can be terminated", startTrainingTick, trainingDurationInTicks, trainingEndTick, sessionTerminateTick))
	}

	return func(sequencer *CustomEventSequencer) {
		sequencer.RegisterSession(sessionArgs.SessionID, sessionArgs.MaxCPUs, sessionArgs.MaxMemoryGB, sessionArgs.MaxGPUs, 0)

		sequencer.AddSessionStartedEvent(sessionArgs.SessionID, sessionStartTick, 0, 0, 0, 1)
		sequencer.AddTrainingEvent(sessionArgs.SessionID, startTrainingTick, trainingDurationInTicks, trainingArgs.CpuUtilization, trainingArgs.MemoryUsageGB, trainingArgs.GpuUtilization, 1) // TODO: Fix GPU util/num GPU specified here.
		sequencer.AddSessionTerminatedEvent(sessionArgs.SessionID, sessionTerminateTick)

		sequencer.SubmitEvents()
	}, nil
}
