package domain

// EventQueueReceiver is an interface that is wrapped by the EventQueue interface.
// Entities that only need to be able to deposit events into the queue receive/use values of this type.
// This is because they have no reason to access the rest of the EventQueue API.
type EventQueueReceiver interface {
	EnqueueEvent(*Event)
}
