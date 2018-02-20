package drai

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"sync"
	"time"
)

var (
	ErrStopping = errors.New("Engine is stopping")
)

type Engine struct {
	sync.RWMutex

	// Currently running and active application instances
	CurrentInstances []*Instance

	StorageBackend StorageBackend

	Stopped bool
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Run() {
	ticker := time.NewTicker(time.Second * 10)
	for {
		<-ticker.C

		e.RLock()
		runningCop := make([]*Instance, len(e.CurrentInstances))
		copy(runningCop, e.CurrentInstances)
		e.RUnlock()

		for _, v := range runningCop {
			v.RLock()
			if v.IdleTimeout != 0 && time.Now().Sub(v.LastAction) > v.IdleTimeout {
				v.RUnlock()
				go func() {
					v.Lock()
					v.Exit()
					v.Unlock()
				}()
				logrus.Info("App had idle timeout")
				continue
			}
			v.RUnlock()
		}
	}
}

// StartApp starts the specified application, returns an error if the app failed to start
func (e *Engine) StartApp(session *discordgo.Session, app App, guildID, channelID string, idleTimeout time.Duration) (*Instance, error) {
	instance := &Instance{
		App:         app,
		ChannelID:   channelID,
		GuildID:     guildID,
		Engine:      e,
		Session:     session,
		IdleTimeout: idleTimeout,
		LastAction:  time.Now(),
	}

	err := instance.App.Start(instance)
	if err != nil {
		return instance, err
	}

	e.Lock()
	if e.Stopped {
		return nil, ErrStopping
	}

	e.CurrentInstances = append(e.CurrentInstances, instance)
	e.Unlock()

	return instance, nil
}

// HandleMessageReactionAdd is supposed to be added as a discord handler
// it handles incomming Reaction Add events to be further processed
func (e *Engine) HandleMessageReactionAdd(s *discordgo.Session, ra *discordgo.MessageReactionAdd) {
	e.RLock()
	if e.Stopped {
		e.RUnlock()
		return
	}

	for _, instance := range e.CurrentInstances {
		if instance.ChannelID == ra.ChannelID {
			go instance.handleReactionAdd(s, ra)
		}
	}

	e.RUnlock()
}

func (e *Engine) StopAndSaveStates() error {
	e.Lock()
	if e.Stopped {
		e.Unlock()
		return nil
	}

	if e.StorageBackend == nil {
		e.StorageBackend = &FSStorageBackend{Path: "drai_apps.json"}
		logrus.Warn("No storage backend specified, using default fs backend: drai_apps.json")
	}

	e.Stopped = true

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
	UserData

	Emoji     string
	MessageID string

	RemoveReactionOnSuccess      bool
	RemoveReactionNotWhitelisted bool

	// Currently unused
	Name string
}

func (a *Action) Equal(other *Action) bool {
	if a.MessageID == other.MessageID && a.Emoji == other.Emoji {
		return true
	}

	return false
}

type UserData struct {
	M map[string]interface{}
}

func (u *UserData) Set(key string, val interface{}) {
	if u.M == nil {
		u.M = make(map[string]interface{})
	}

	u.M[key] = val
}

func (u *UserData) Get(key string) (interface{}, bool) {
	if u.M == nil {
		u.M = make(map[string]interface{})
	}

	v, ok := u.M[key]
	return v, ok
}

func (u *UserData) Str(key string) (v string, ok bool) {
	if iv, ok2 := u.Get(key); ok2 {
		switch t := iv.(type) {
		case string:
			v = t
			ok = true
		case []byte:
			v = string(t)
			ok = true
		}
	}

	return
}

func (u *UserData) Bool(key string) (v bool, ok bool) {
	if iv, ok2 := u.Get(key); ok2 {
		switch t := iv.(type) {
		case bool:
			v = t
			ok = true
		}
	}

	return
}

func (u *UserData) Int64(key string) (v int64, ok bool) {
	if iv, ok2 := u.Get(key); ok2 {
		switch t := iv.(type) {
		case int:
			v = int64(t)
			ok = true
		case int64:
			v = t
			ok = true
		case float64:
			v = int64(t)
			ok = true
		case float32:
			v = int64(t)
			ok = true
		}
	}

	return
}
func (u *UserData) Int(key string) (v int, ok bool) {
	if iv, ok2 := u.Int64(key); ok2 {
		v = int(iv)
		ok = true
	}

	return
}
