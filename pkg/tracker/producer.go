package tracker

// Producer - this interface is exposed by Producer types.
// Such types are responsible for writing messages to the messages channel.
type Producer interface {
	// Name for the producer
	Name() string
	// Start starts goroutines which poll or connect for messages and
	// writes them to the messages channel
	Start()
	// Stop sends the stop signal to the goroutines and waits for them
	// to finish.
	Stop()
}
