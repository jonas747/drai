package drai

import (
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"sync"
)

// Instance represents a currently loaded and running app
type Instance struct {
	sync.RWMutex

	// Immutable fields
	session   *discordgo.Session
	engine    *Engine
	app       App
	channelID string
	guildID   string

	actions []*Action

	// Will only react to these users
	userIDs       []string
	AllowAllUsers bool
}

// Accessors for the immutable fields
func (i *Instance) Session() *discordgo.Session { return i.session }
func (i *Instance) Engine() *Engine             { return i.engine }
func (i *Instance) App() App                    { return i.app }
func (i *Instance) ChannelID() string           { return i.channelID }
func (i *Instance) GuildID() string             { return i.guildID }

func (i *Instance) handleReactionAdd(s *discordgo.Session, ra *discordgo.MessageReactionAdd) {
	i.RLock()

	if !i.AllowAllUsers {
		// Check user filter
		found := false
		for _, v := range i.userIDs {
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
	for _, a := range i.actions {
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
	err := action.Callback(ra.UserID, action)
	i.Unlock()

	if err != nil {
		logrus.WithError(err).Error("Error running action callback")
	}
}

// AddActions registers a set of of actions on the message, adding the reactions aswell
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) AddActions(actions ...*Action) error {
	i.actions = append(i.actions, actions...)

	// Add the reactions
	for _, v := range actions {
		err := i.session.MessageReactionAdd(i.channelID, v.MessageID, v.Emoji)
		if err != nil {
			return errors.WithMessage(err, "AddActions")
		}
	}

	return nil
}

// RemoveActions unregisters a set of of actions on the message, clearing the reactions aswell
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (inst *Instance) RemoveActions(actions ...*Action) {
	for i := 0; i < len(inst.actions); i++ {
		for _, a := range actions {
			elem := inst.actions[i]
			if elem != a {
				continue
			}

			inst.actions = append(inst.actions[:i], inst.actions[i+1:]...)

			// Remove the reactions
			// TODO: Remove all users reactions
			inst.session.MessageReactionRemove(inst.channelID, elem.MessageID, elem.Emoji, "@me")
		}
	}
}

// ClearActions unregisters all actions on the message, clearing the reactions aswell
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) ClearActions() {
	for _, a := range i.actions {
		// Remove the reactions
		i.session.MessageReactionsRemoveAll(i.channelID, a.MessageID)
	}

	i.actions = nil // Maybe also remove the reactions?
}

// AddUsers adds the specified users to the whitelist
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) AddUsers(userIDs []string) {
	i.userIDs = append(i.userIDs, userIDs...)
}

// RemoveUsers removes the specified users from the whitelist
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) RemoveUsers(userIDs []string) {
	for j := 0; j < len(i.actions); j++ {
		for _, uid := range userIDs {
			elem := i.userIDs[j]
			if elem == uid {
				i.userIDs = append(i.userIDs[:j], i.userIDs[j+1:]...)
			}
		}
	}
}

// ClearUsers clears the whitelist
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (i *Instance) ClearUsers() {
	i.userIDs = nil
}

// Exit ends a App.
// Note: If called outside of Start, Exit, or action callbacks, then you need to the instance to avoid race conditions
func (inst *Instance) Exit() {
	inst.engine.Lock()

	for i, v := range inst.engine.CurrentInstances {
		if v == inst {
			inst.engine.CurrentInstances = append(inst.engine.CurrentInstances[:i], inst.engine.CurrentInstances[i+1:]...)
		}
	}

	inst.engine.Unlock()

	inst.app.Exit(inst)
}

type App interface {

	// Initialize and Start your app here
	Start(instance *Instance) error
	// Perform cleanup here
	Exit(instance *Instance) error

	// Your app should be able to load from this serialized format after say a restart for example
	SerializeFullState() ([]byte, error)
	LoadFullState([]byte) error
}
