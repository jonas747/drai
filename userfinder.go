package drai

import (
	"github.com/bwmarrin/discordgo"
	"time"
)

type UserFinder struct {
	Instance         *Instance
	Users            []*discordgo.User
	NumUsersToFind   int
	UsersFoundCB     func([]*discordgo.User)
	usersFoundCalled bool

	MessageID string

	addAction    *Action
	removeAction *Action
}

func (u *UserFinder) Start() error {
	u.Instance.AllowAllUsers = true

	err := u.UpdateMessage()
	if err != nil {
		return err
	}

	u.addAction = &Action{
		Emoji:     "➕",
		Callback:  u.onActionAdd,
		MessageID: u.MessageID,
	}

	u.removeAction = &Action{
		Emoji:     "➖",
		Callback:  u.onActionRemove,
		MessageID: u.MessageID,
	}

	return u.Instance.AddActions(u.addAction, u.removeAction)
}

func (u *UserFinder) UpdateMessage() error {
	content := "**Waiting for users to join.**\n```\n"
	for i := 0; i < u.NumUsersToFind; i++ {
		if i < len(u.Users) {
			content += u.Users[i].Username + "#" + u.Users[i].Discriminator + "\n"
		} else {
			content += "- Open Slot -\n"
		}
	}

	content += "```\nReact with ➕ to join, and ➖ to leave.\n"

	if len(u.Users) >= u.NumUsersToFind {
		content += "\nAll users found! Starting in 1 second..."
	}

	if u.MessageID == "" {
		m, err := u.Instance.Session.ChannelMessageSend(u.Instance.ChannelID, content)
		if err != nil {
			return err
		}
		u.MessageID = m.ID
	} else {
		_, err := u.Instance.Session.ChannelMessageEdit(u.Instance.ChannelID, u.MessageID, content)
		return err
	}

	return nil
}

func (u *UserFinder) HandleAction(userID string, action *Action) error {
	if action == u.addAction {
		return u.onActionAdd(userID, action)
	} else if action == u.removeAction {
		return u.onActionRemove(userID, action)
	}

	return nil
}

func (u *UserFinder) onActionAdd(userID string, action *Action) error {
	for _, v := range u.Users {
		if v.ID == userID {
			// Already added
			return nil
		}
	}

	member, err := u.Instance.Session.GuildMember(u.Instance.GuildID, userID)
	if err != nil {
		return err
	}

	u.Users = append(u.Users, member.User)
	if len(u.Users) >= u.NumUsersToFind {
		go u.DelayedCallDB()
	}

	return u.UpdateMessage()
}

func (u *UserFinder) onActionRemove(userID string, action *Action) error {

	for i, v := range u.Users {
		if v.ID == userID {
			u.Users = append(u.Users[:i], u.Users[i+1:]...)
		}
	}

	return u.UpdateMessage()
}

func (u *UserFinder) DelayedCallDB() {
	time.Sleep(time.Second)

	// Have to explicitly lock it here since were outside of any of the managed functions
	u.Instance.Lock()
	if len(u.Users) >= u.NumUsersToFind && !u.usersFoundCalled {
		u.UsersFoundCB(u.Users)
		u.usersFoundCalled = true

		u.Instance.RemoveActions(u.addAction, u.removeAction)
	}

	u.Instance.Unlock()
}
