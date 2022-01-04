package botty

import "github.com/gillepool/botty/reactions"

type InitEvent struct{}

type ShutdownEvent struct{}

// The ReceiveMessageEvent is typically emitted by an Adapter when the Bot sees
// a new message from the chat.
type ReceiveMessageEvent struct {
	ID       string // The ID of the message, identifying it at least uniquely within the Channel
	Text     string // The message text.
	AuthorID string // A string identifying the author of the message on the adapter.
	Channel  string // The channel over which the message was received.

	// A message may optionally also contain additional information that was
	// received by the Adapter (e.g. with the slack adapter this may be the
	// *slack.MessageEvent. Each Adapter implementation should document if and
	// what information is available here, if any at all.
	Data interface{}
}

// An Event may be emitted by a chat Adapter to indicate that a message
// received a reaction.
type Event struct {
	Reaction  reactions.Reaction
	MessageID string
	Channel   string
	AuthorID  string
}
