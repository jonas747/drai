package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dcmd"
	"github.com/jonas747/drai"
	"github.com/jonas747/drai/Applications/tictactoe"
	"os"
)

var (
	engine  *drai.Engine
	session *discordgo.Session

	cmdSys *dcmd.System
)

func main() {
	token := os.Getenv("DG_TOKEN")
	if token == "" {
		logrus.Fatal("No token provided using the DG_TOKEN env variable")
	}

	var err error
	session, err = discordgo.New(token)
	if err != nil {
		logrus.WithError(err).Fatal("Failed creating discordgo session")
	}

	engine = drai.NewEngine()
	session.AddHandler(engine.HandleMessageReactionAdd)

	cmdSys := dcmd.NewStandardSystem("!g")
	cmdSys.Root.AddCommand(dcmd.NewStdHelpCommand(), dcmd.NewTrigger("help"))
	cmdSys.Root.AddCommand(cmdTicTacToe, dcmd.NewTrigger("tictactoe", "ttc"))

	session.AddHandler(cmdSys.HandleMessageCreate)
	err = session.Open()
	if err != nil {
		logrus.WithError(err).Fatal("Failed Connecting to discord")
	}
	logrus.Info("Running...")
	select {}
}

var cmdTicTacToe = &dcmd.SimpleCmd{
	ShortDesc: "Play tic tac toe",
	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		game := tictactoe.NewGame(data.Msg.Author)
		_, err := engine.StartApp(data.Session, game, data.Guild.ID, data.Channel.ID)
		if err != nil {
			logrus.WithError(err).Error("Failed starting tic tac toe :(")
			return "Failed starting :(", err
		}
		return "", nil
	},
}
