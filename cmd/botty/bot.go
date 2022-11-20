package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gillepool/botty/internal/adapter"
	"github.com/gillepool/botty/internal/brain"
	"github.com/gillepool/botty/internal/events"
	"github.com/gillepool/botty/internal/message"
	"github.com/gillepool/botty/internal/storage"
	"github.com/gillepool/botty/pkg/logger"
	"go.uber.org/zap"
)

type Bot struct {
	Name    string
	Adapter adapter.Adapter
	Brain   *brain.Brain
	Storage *storage.Storage
	Logger  *zap.Logger
}

func New(name string) *Bot {
	logger := logger.NewLogger()
	store := storage.NewStorage(logger.Named("Storage"))

	// Should read the address from config instead
	storage.NewRedisStorage(storage.Config{
		Addr: os.Getenv("redis_addr"),
	})
	brain := brain.NewBrain(logger.Named("Brain"))

	adapter, _ := adapter.NewDiscordAdapter("Daniel", os.Getenv("discord_token"), logger.Named("Discord"))

	logger.Info("Storaged used: ", zap.Any("Storage", store))
	return &Bot{
		Name:    name,
		Brain:   brain,
		Adapter: adapter,
		Storage: store,
		Logger:  logger,
	}
}

func (b *Bot) Respond(msg string, fun func(message.Message) error) {
	b.Logger.Info("Response to", zap.String("message", msg))
	expr := "^" + msg + "$"
	b.RespondRegexp(expr, fun)
}

func (b *Bot) RespondRegexp(expr string, fun func(message.Message) error) {
	if expr == "" {
		return
	}

	if expr[0] == '^' {
		if !strings.HasPrefix(expr, "^(?i)") {
			expr = "^(?i)" + expr[1:]
		}
	} else {
		if !strings.HasPrefix(expr, "(?i)") {
			expr = "^(?i)" + expr
		}
	}

	regex, err := regexp.Compile(expr)
	if err != nil {
		caller := "someone"
		err = fmt.Errorf("%s: %w", caller, err)
		b.Brain.RegistrationErrs = append(b.Brain.RegistrationErrs, err)
		return
	}

	b.Brain.RegisterHandler(func(ctx context.Context, evt events.ReceiveMessageEvent) error {
		matches := regex.FindStringSubmatch(evt.Text)
		if len(matches) == 0 {
			return nil
		}

		brain.FinishEventContent(ctx)

		return fun(message.Message{
			Context:  ctx,
			ID:       evt.ID,
			Text:     evt.Text,
			AuthorID: evt.AuthorID,
			Data:     evt.Data,
			Channel:  evt.Channel,
			Matches:  matches[1:],
			Adapter:  b.Adapter,
		})
	})
}

type ExampleBot struct {
	*Bot
}

func (b *Bot) Run() error {
	fmt.Println("Run")
	if len(b.Brain.RegistrationErrs) > 0 {
		return fmt.Errorf("invalid event handlers: %w", b.Brain.RegistrationErrs)
	}

	b.Adapter.RegisterAt(b.Brain)

	b.Logger.Info("Initialize bot", zap.String("name", b.Name))
	b.Brain.HandleEvents()

	b.Logger.Info("Close storage on shutdown", zap.String("name", b.Name))
	err := b.Storage.Close()
	if err != nil {
		b.Logger.Info("Error while closing memory", zap.Error(err))
	}
	return nil
}

func main() {
	bot := &ExampleBot{
		Bot: New("Botty"),
	}

	bot.Respond("remember (.+) is (.+)", bot.Remember)
	bot.Respond("what is (.+)", bot.WhatIs)
	bot.Run()
}

func (b *ExampleBot) Remember(msg message.Message) error {
	b.Logger.Info("Remember command")
	key, value := msg.Matches[0], msg.Matches[1]
	msg.Respond("Ok I'll remember %s is %s", key, value)
	return b.Storage.Set(key, value)
}

func (b *ExampleBot) WhatIs(msg message.Message) error {
	key := msg.Matches[0]

	key = strings.TrimSuffix(key, "\r")

	var value string
	ok, err := b.Storage.Get(key, &value)
	if err != nil {
		return err
	}

	if ok {
		msg.Respond("%s is %s", key, value)
	} else {
		msg.Respond("Could not found %q stored", key)
	}
	return nil
}
