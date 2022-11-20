package events

type InitEvent struct{}

type ShutdownEvent struct{}

// The ReceiveMessageEvent is emitted by an Adapter when the Bot sees
// a new message from the chat.
type ReceiveMessageEvent struct {
	ID       string // The ID of the message, identifying it at least uniquely within the Channel
	Text     string // The message text.
	AuthorID string // A string identifying the author of the message on the adapter.
	Channel  string // The channel over which the message was received.

	// A message may optionally also contain additional information that was
	// received by the Adapter
	Data interface{}
}
