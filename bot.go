package botty

import (
	"fmt"

	"github.com/gillepool/botty/brain"
	"github.com/gillepool/botty/storage"
)

type Bot struct {
	Name    string
	Brain   *brain.Brain
	Storage *storage.Storage
}

func New(name string) *Bot {
	brain := brain.NewBrain()
	store := storage.NewStorage()
	return &Bot{
		Name:    name,
		Brain:   brain,
		Storage: store,
	}
}

func (b *Bot) Run() error {
	if len(b.Brain.RegistrationErrs) > 0 {
		return fmt.Errorf("invalid event handlers: %w", b.Brain.RegistrationErrs)
	}

	b.Brain.HandleEvents()
	return nil
}

func main() {
	bot := New("Botty")
	bot.Run()
}
