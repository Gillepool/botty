package adapter

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gillepool/botty/internal/brain"
	"github.com/gillepool/botty/internal/events"
	"go.uber.org/zap"
)

type DiscordAdapter struct {
	Client *discordgo.Session
	Prefix string
	Author string
	logger *zap.Logger
	events chan discordEvent
}

type discordEvent struct {
	Message discordgo.Message
}

// NewCLIAdapter creates a new CLIAdapter. The caller must call Close
// to make the CLIAdapter stop reading messages and emitting events.
func NewDiscordAdapter(name, token string, logger *zap.Logger) (*DiscordAdapter, error) {
	// Create a new Discord session using the provided bot token.
	client, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	events := make(chan discordEvent)

	discordAdapter := &DiscordAdapter{
		Client: client,
		Prefix: fmt.Sprintf("%s > ", name),
		Author: "Daniel",
		logger: logger,
		events: events,
	}

	// In this example, we only care about receiving message events.
	discordAdapter.Client.Identify.Intents = discordgo.IntentsGuildMessages

	discordAdapter.Client.Open()

	discordAdapter.Client.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		events <- discordEvent{
			Message: *m.Message,
		}
	})

	return discordAdapter, nil
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.

func (a *DiscordAdapter) handleDiscordEvents(b *brain.Brain) {
	for msg := range a.events {
		a.handleMessageEvent(msg.Message, b)
	}
}

func (a *DiscordAdapter) handleMessageEvent(msg discordgo.Message, brain *brain.Brain) {
	brain.Emit(events.ReceiveMessageEvent{
		Text:     msg.Content,
		Channel:  msg.ChannelID,
		ID:       msg.ID,
		AuthorID: msg.Author.Username,
		Data:     msg,
	})
}

func (a *DiscordAdapter) RegisterAt(brain *brain.Brain) {
	go a.handleDiscordEvents(brain)
}

// Send implemenation sends all text messages to given ChannelID
func (a *DiscordAdapter) Send(text, channelID string) error {
	a.logger.Info("Sending message to channel", zap.String("text", text))
	_, err := a.Client.ChannelMessageSend(channelID, text)
	return err
}

// Close makes the CLIAdapter stop emitting any new events or printing any output.
// Calling this function more than once will result in an error.
func (a *DiscordAdapter) Close() error {
	return nil
}
