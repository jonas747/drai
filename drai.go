package drai

import (
	"github.com/bwmarrin/discordgo"
	"sync"
)

type Engine struct {
	sync.RWMutex

	// Currently running and active application instances
	CurrentInstances []*Instance
}

func NewEngine() *Engine {
	return &Engine{}
}

// StartApp starts the specified application, returns an error if the app failed to start
func (e *Engine) StartApp(session *discordgo.Session, app App, guildID, channelID string) (*Instance, error) {
	instance := &Instance{
		app:       app,
		channelID: channelID,
		guildID:   guildID,
		engine:    e,
		session:   session,
	}

	err := instance.App().Start(instance)
	if err != nil {
		return instance, err
	}

	e.Lock()
	e.CurrentInstances = append(e.CurrentInstances, instance)
	e.Unlock()

	return instance, nil
}

// HandleMessageReactionAdd is supposed to be added as a discord handler
// it handles incomming Reaction Add events to be further processed
func (e *Engine) HandleMessageReactionAdd(s *discordgo.Session, ra *discordgo.MessageReactionAdd) {
	e.RLock()

	for _, instance := range e.CurrentInstances {
		if instance.ChannelID() == ra.ChannelID {
			go instance.handleReactionAdd(s, ra)
		}
	}

	e.RUnlock()
}

// Action represents a registered action for apps
type Action struct {
	Emoji     string
	Callback  func(string, *Action) error
	MessageID string

	RemoveReactionOnSuccess      bool
	RemoveReactionNotWhitelisted bool

	// Currently unused
	Name string

	// But whatever you want here
	UserData interface{}
}
