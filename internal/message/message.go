package message

import (
	"context"
	"fmt"

	"github.com/gillepool/botty/internal/adapter"
)

// A Message is automatically created from a ReceiveMessageEvent and then passed
// to the RespondFunc that was registered via Bot.Respond(…) or Bot.RespondRegex(…)
// when the message matches the regular expression of the handler.
type Message struct {
	Context  context.Context
	ID       string // The ID of the message, identifying it at least uniquely within the Channel
	Text     string
	AuthorID string
	Channel  string
	Matches  []string    // contains all sub matches of the regular expression that matched the Text
	Data     interface{} // corresponds to the ReceiveMessageEvent.Data field

	Adapter adapter.Adapter
}

func (msg *Message) Respond(text string, args ...interface{}) {
	_ = msg.RespondE(text, args...)
}

func (msg *Message) RespondE(text string, args ...interface{}) error {
	if len(args) > 0 {
		text = fmt.Sprintf(text, args...)
	}

	return msg.Adapter.Send(text, msg.Channel)
}
