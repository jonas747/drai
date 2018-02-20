package drai

import (
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"sync"
	"time"
)

// Instance represents a currently loaded and running app
type Instance struct {
	sync.RWMutex

	// Immutable fields
	// Do nott modify these or you will encounter race conditions
	Session   *discordgo.Session
	Engine    *Engine
	App       App
	ChannelID string
	GuildID   string
	Actions   []*Action

	// Will only react to these users
	UserIDs       []string
	AllowAllUsers bool

	IdleTimeout time.Duration
	LastAction  time.Time
}

func (i *Instance) handleReactionAdd(s *discordgo.Session, ra *discordgo.MessageReactionAdd) {
	i.RLock()

	if !i.AllowAllUsers {
		// Check user filter
		found := false
		for _, v := range i.UserIDs {
			if v == ra.UserID {
				found = true
				break
			}
		}

		if !found {
			i.RUnlock()
			return
		}
	}
	var action *Action

	// Find the corresponding action
	for _, a := range i.Actions {
		if a.MessageID == ra.MessageID && a.Emoji == ra.Emoji.Name {
			// Bingo!
			action = a
			break
		}
	}

	i.RUnlock()

	if action == nil {
		return
	}

	// Upgrade the lock, and call the callback if found
	i.Lock()
	err := i.App.HandleAction(ra.UserID, action)
	i.Unlock()

	if err != nil {
		logrus.WithError(err).Error("Error running action callback")
	}
}

// AddActions registers a set of of actions on the message, adding the reactions aswell
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) AddActions(actions ...*Action) error {
	i.Actions = append(i.Actions, actions...)

	// Add the reactions
	for _, v := range actions {
		err := i.Session.MessageReactionAdd(i.ChannelID, v.MessageID, v.Emoji)
		if err != nil {
			return errors.WithMessage(err, "AddActions")
		}
	}

	return nil
}

// RemoveActions unregisters a set of of actions on the message, clearing the reactions aswell
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (inst *Instance) RemoveActions(actions ...*Action) {
	for i := 0; i < len(inst.Actions); i++ {
		for _, a := range actions {
			elem := inst.Actions[i]
			if elem != a {
				continue
			}

			inst.Actions = append(inst.Actions[:i], inst.Actions[i+1:]...)

			// Remove the reactions
			// TODO: Remove all users reactions
			inst.Session.MessageReactionRemove(inst.ChannelID, elem.MessageID, elem.Emoji, "@me")
		}
	}
}

// ClearActions unregisters all actions on the message, clearing the reactions aswell
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) ClearActions() {
	for _, a := range i.Actions {
		// Remove the reactions
		i.Session.MessageReactionsRemoveAll(i.ChannelID, a.MessageID)
	}

	i.Actions = nil // Maybe also remove the reactions?
}

// AddUsers adds the specified users to the whitelist
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) AddUsers(UserIDs []string) {
	i.UserIDs = append(i.UserIDs, UserIDs...)
}

// RemoveUsers removes the specified users from the whitelist
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) RemoveUsers(UserIDs []string) {
	for j := 0; j < len(i.Actions); j++ {
		for _, uid := range UserIDs {
			elem := i.UserIDs[j]
			if elem == uid {
				i.UserIDs = append(i.UserIDs[:j], i.UserIDs[j+1:]...)
			}
		}
	}
}

// ClearUsers clears the whitelist
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) ClearUsers() {
	i.UserIDs = nil
}

// Exit ends a App.
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (inst *Instance) Exit() {
	inst.Engine.Lock()

	for i, v := range inst.Engine.CurrentInstances {
		if v == inst {
			inst.Engine.CurrentInstances = append(inst.Engine.CurrentInstances[:i], inst.Engine.CurrentInstances[i+1:]...)
		}
	}

	inst.Engine.Unlock()

	inst.App.Exit(inst)
}

type App interface {
	// Initialize and Start your app here
	// This is not called when loading from a serialized state
	Start(instance *Instance) error

	// Perform cleanup here
	Exit(instance *Instance) error

	HandleAction(userID string, action *Action) error

	// Your app should be able to load from this serialized format after say a restart for example
	SerializeState() ([]byte, error)

	// Initalize the app here from a serialized state
	LoadState(*Instance, []byte) error
}
