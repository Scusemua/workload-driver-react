package domain

import "time"

type EventName interface {
	String() string
}

type PodData interface {
	GetPod() string
}

type EventSource interface {
	OnEvent() <-chan Event
	String() string
	Id() int
	SetId(int)
	IsDriver() bool
	// If the EventSource is the last one, should the simulation continue?
	// For Drivers, the answer is yes. For non-Drivers, it may depend.
	// For example, if the BufferedService is the last one, then the simulation
	// should end once all buffered events have been retriggered.
	IsLastShouldContinue() bool
	// Called in pre-run mode when the Synthesizer encounters a training-started event.
	// Sets the value in the latest training max slot to 0.
	TrainingStarted(string)
	// Called in pre-run mode when the Synthesizer encounters a training-stopped event.
	// Prepares the next slot in the training maxes by appending to the list a new value of -1.
	TrainingEnded(string)
}

type EventConsumer interface {
	SubmitEvent(Event) // Give an event to the EventConsumer so that it may be processed.
	DoneChan() chan struct{}
}

type Event interface {
	EventSource() EventSource
	OriginalEventSource() EventSource
	Name() EventName
	Data() interface{}
	Timestamp() time.Time
	Id() string
	OrderSeq() int64 // OrderSeq is essentially timestamp of event, but randomized to make behavior stochastic.
	SetOrderSeq(int64)
	String() string
}
