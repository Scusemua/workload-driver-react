package generator

import (
	"errors"
	"fmt"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
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
type SequencerFunction func(sequencer *CustomEventSequencer) error

// Utility/helper struct to specify arguments of a Session that should be registered with a CustomEventSequencer.
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

// Utility/helper struct to specify the resource utilization during a training event/task.
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

// Set the GPU utilization of the next GPU that has not already been specified via the 'WithGpuUtilization' function.
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

// Set the GPU utilization of a specific GPU (identified by its index, which should range from 0 to NUM_GPUS - 1).
//
// This modifies the 'TrainingResourceUtilizationArgs' struct on which it was called in-place; it also returns the TrainingResourceUtilizationArgs struct.
func (a *TrainingResourceUtilizationArgs) WithGpuUtilizationForSpecificGpu(gpuIndex int, gpuUtil float64) *TrainingResourceUtilizationArgs {
	if gpuIndex < 0 || gpuIndex > a.NumGPUs() {
		panic(fmt.Sprintf("Invalid GPU index specified: %d. Value must be greater than or equal to 0 and less than the total number of GPUs (%d).", gpuIndex, cap(a.GpuUtilization)))
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

	if session.GetResourceRequest().Cpus < 0 {
		return fmt.Errorf("%w: invalid maximum number of CPUs specified (%f). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, session.GetResourceRequest().Cpus)
	}

	if session.GetResourceRequest().Gpus < 0 {
		return fmt.Errorf("%w: invalid maximum number of GPUs specified (%d). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, session.GetResourceRequest().Gpus)
	}

	if session.GetResourceRequest().MemoryGB < 0 {
		return fmt.Errorf("%w: invalid maximum memory usage (in GB) specified (%f). Quantity must be greater than or equal to 0", ErrInvalidConfiguration, session.GetResourceRequest().MemoryGB)
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
		if session.GetResourceRequest().Cpus < trainingEvent.CpuUtil {
			return fmt.Errorf("%w: incompatible max CPUs (%f) and training CPU utilization (%f) specified. Training CPU utilization cannot exceed maximum session CPUs", ErrInvalidConfiguration, session.GetResourceRequest().Cpus, trainingEvent.CpuUtil)
		}

		if session.GetResourceRequest().Gpus < trainingEvent.NumGPUs() {
			return fmt.Errorf("%w: incompatible max GPUs (%d) and training GPU utilization (%d) specified. Training GPU utilization cannot exceed maximum session GPUs", ErrInvalidConfiguration, session.GetResourceRequest().Gpus, trainingEvent.NumGPUs())
		}

		if session.GetResourceRequest().MemoryGB < trainingEvent.MemUsageGB {
			return fmt.Errorf("%w: incompatible max memory usage (%f GB) and training memory usage (%f GB) specified. Training memory usage cannot exceed maximum session memory usage", ErrInvalidConfiguration, session.GetResourceRequest().MemoryGB, trainingEvent.MemUsageGB)
		}
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
func SingleSessionSingleTraining(sessions []*domain.WorkloadTemplateSession) (SequencerFunction, error) {
	if sessions == nil {
		panic("Session arguments cannot be nil.")
	}

	if len(sessions) != 1 {
		panic(fmt.Sprintf("Sessions has unexpected length: %d", len(sessions)))
	}

	var session *domain.WorkloadTemplateSession = sessions[0]
	if err := validateSession(session); err != nil {
		return nil, err
	}

	if err := validateSessionArgumentsAgainstTrainingArguments(session); err != nil {
		return nil, err
	}

	if len(session.GetTrainings()) != 1 {
		return nil, fmt.Errorf("%w: session has illegal number of training events for this particular template (%d, expected 1)", ErrInvalidConfiguration, len(session.GetTrainings()))
	}

	trainingEvent := session.GetTrainings()[0]
	if trainingEvent.DurationInTicks <= 0 {
		return nil, fmt.Errorf("%w: invalid training duration specified: %d ticks. Must be strictly greater than 0", ErrInvalidConfiguration, trainingEvent.DurationInTicks)
	}

	return func(sequencer *CustomEventSequencer) error {
		sequencer.RegisterSession(session.GetId(), session.GetResourceRequest().Cpus, session.GetResourceRequest().MemoryGB, session.GetResourceRequest().Gpus, 0)

		trainingEvent := session.GetTrainings()[0]

		sequencer.AddSessionStartedEvent(session.GetId(), session.GetStartTick(), 0, 0, 0, 1)
		sequencer.AddTrainingEvent(session.GetId(), trainingEvent.StartTick, trainingEvent.DurationInTicks, trainingEvent.CpuUtil, trainingEvent.MemUsageGB, trainingEvent.GpuUtil) // TODO: Fix GPU util/num GPU specified here.
		sequencer.AddSessionTerminatedEvent(session.GetId(), session.GetStopTick())

		sequencer.SubmitEvents()

		return nil
	}, nil
}
