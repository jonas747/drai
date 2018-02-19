package tictactoe

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/drai"
)

func init() {
	drai.RegisterApp("github.com/jonas747/drai/tictactoe", &Game{})
}

func (g *Game) SerializeState() ([]byte, error) {
	return json.Marshal(g)
}

func (g *Game) LoadState(instance *drai.Instance, data []byte) error {
	err := json.Unmarshal(data, g)
	if err != nil {
		return err
	}

	g.Instance = instance

	if g.UserFinder != nil {
		g.UserFinder.UsersFoundCB = g.onUsersFound
		g.UserFinder.Instance = instance
	}

	return nil
}

type Game struct {
	Instance *drai.Instance `json:"-"`

	UserFinder *drai.UserFinder
	Player1    *discordgo.User
	Player2    *discordgo.User
	UsersFound bool

	// 0 1 2
	// 3 4 5
	// 6 7 8
	Board []string

	CurrentTurn int

	MessageID string
}

var Emojis = []string{
	"1⃣", "2⃣", "3⃣",
	"4⃣", "5⃣", "6⃣",
	"7⃣", "8⃣", "9⃣",
}

func NewGame(author *discordgo.User) *Game {
	return &Game{
		Player1: author,
	}
}

// Called by engine when starting
func (g *Game) Start(instance *drai.Instance) error {
	g.Instance = instance

	g.UserFinder = &drai.UserFinder{
		Instance:       instance,
		Users:          []*discordgo.User{g.Player1},
		NumUsersToFind: 2,
		UsersFoundCB:   g.onUsersFound,
	}

	g.Board = make([]string, 9)
	for i, _ := range g.Board {
		g.Board[i] = " "
		// g.Board[i] = strconv.Itoa(i + 1)
	}

	// m := instance.Session().ChannelMessageSend(instance.Channel(), "Setting up...")
	return g.UserFinder.Start()
}

// Perform cleanup here
func (g *Game) Exit(instance *drai.Instance) error { return nil }

func (g *Game) UpdateMessage() {
	currentTurn := g.TurnPlayer(g.CurrentTurn)

	// Draw the board itself
	content := `-------------
| %s | %s | %s |
-------------
| %s | %s | %s |
------------
| %s | %s | %s |
------------`
	content = fmt.Sprintf("```\n"+content+"\n```", g.Board[0], g.Board[1], g.Board[2], g.Board[3], g.Board[4], g.Board[5], g.Board[6], g.Board[7], g.Board[8])

	content += fmt.Sprintf("\nCurrent Turn: %d", g.CurrentTurn)
	winner := g.CheckForWinner()
	if winner != nil {
		content += fmt.Sprintf("\n**%s** WON! WOOOHOO!", winner.Username)
	} else {
		content += fmt.Sprintf("\nCurrent Player: %s#%s", currentTurn.Username, currentTurn.Discriminator)
	}

	g.Instance.Session.ChannelMessageEdit(g.Instance.ChannelID, g.MessageID, content)
}

func (g *Game) TurnPlayer(turn int) *discordgo.User {
	var nextTurn *discordgo.User
	if turn%2 == 0 {
		nextTurn = g.Player1
	} else {
		nextTurn = g.Player2
	}
	return nextTurn
}

func (g *Game) CheckForWinner() *discordgo.User {
	if g.isWinner("O") {
		return g.Player1
	} else if g.isWinner("X") {
		return g.Player2
	}

	return nil
}

func (g *Game) isWinner(s string) bool {
	// First horizontal line
	if g.Board[0] == s && g.Board[1] == s && g.Board[2] == s {
		return true
	}

	// Second horizontal line
	if g.Board[3] == s && g.Board[4] == s && g.Board[5] == s {
		return true
	}

	// Third horizontal line
	if g.Board[6] == s && g.Board[7] == s && g.Board[8] == s {
		return true
	}

	// First vertical line
	if g.Board[0] == s && g.Board[3] == s && g.Board[6] == s {
		return true
	}

	// Second vertical line
	if g.Board[1] == s && g.Board[4] == s && g.Board[7] == s {
		return true
	}

	// Third vertical line
	if g.Board[2] == s && g.Board[5] == s && g.Board[8] == s {
		return true
	}

	// Diagonal
	if g.Board[0] == s && g.Board[4] == s && g.Board[8] == s {
		return true
	}
	// Diagonal
	if g.Board[2] == s && g.Board[4] == s && g.Board[6] == s {
		return true
	}

	return false
}

func (g *Game) onUsersFound(users []*discordgo.User) {
	g.Player1 = users[0]
	g.Player2 = users[1]
	g.UsersFound = true

	// Reuse this message
	g.MessageID = g.UserFinder.MessageID

	g.Instance.ClearActions()

	for i := 0; i < 9; i++ {
		a := &drai.Action{
			Emoji:     Emojis[i],
			MessageID: g.MessageID,
		}
		a.Set("index", i)
		g.Instance.AddActions(a)
	}

	g.UpdateMessage()

	return
}

func (g *Game) HandleAction(userID string, action *drai.Action) error {
	if handled, err := g.UserFinder.HandleAction(userID, action); handled {
		return err
	}

	cPlayer := g.TurnPlayer(g.CurrentTurn)
	if cPlayer.ID != userID {
		return nil
	}

	symbol := "O"
	if cPlayer == g.Player2 {
		symbol = "X"
	}

	index, _ := action.Int("index")
	if g.Board[index] == "O" || g.Board[index] == "X" {
		return nil // Already taken
	}

	g.Board[index] = symbol
	g.CurrentTurn++
	g.UpdateMessage()

	if g.CheckForWinner() != nil {
		g.Instance.Exit()
	}

	return nil
}
