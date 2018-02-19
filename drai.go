package drai

import (
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"sync"
)

type Engine struct {
	sync.RWMutex

	// Currently running and active application instances
	CurrentInstances []*Instance

	StorageBackend StorageBackend
}

func NewEngine() *Engine {
	return &Engine{}
}

// StartApp starts the specified application, returns an error if the app failed to start
func (e *Engine) StartApp(session *discordgo.Session, app App, guildID, channelID string) (*Instance, error) {
	instance := &Instance{
		App:       app,
		ChannelID: channelID,
		GuildID:   guildID,
		Engine:    e,
		Session:   session,
	}

	err := instance.App.Start(instance)
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
		if instance.ChannelID == ra.ChannelID {
			go instance.handleReactionAdd(s, ra)
		}
	}

	e.RUnlock()
}

func (e *Engine) StopAndSaveStates() error {
	e.Lock()
	if e.StorageBackend == nil {
		e.StorageBackend = &FSStorageBackend{Path: "drai_apps.json"}
		logrus.Warn("No storage backend specified, using default fs backend: drai_apps.json")
	}

	err := e.StorageBackend.SaveApps(e.CurrentInstances)
	if err != nil {
		return err
	}

	e.Unlock()

	return nil
}

func (e *Engine) RestoreApps(session *discordgo.Session) error {
	e.Lock()
	if e.StorageBackend == nil {
		e.StorageBackend = &FSStorageBackend{Path: "drai_apps.json"}
		logrus.Warn("No storage backend specified, using default fs backend: drai_apps.json")
	}

	apps, err := e.StorageBackend.LoadApps(e, session)
	if err != nil {
		return err
	}

	e.CurrentInstances = apps

	e.Unlock()

	return nil
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
