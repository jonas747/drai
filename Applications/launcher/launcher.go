package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dcmd"
	"github.com/jonas747/drai"
	"github.com/jonas747/drai/Applications/tictactoe"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

var (
	engine  *drai.Engine
	session *discordgo.Session

	cmdSys *dcmd.System
)

func debugSrv() {
	err := http.ListenAndServe(":5000", nil)
	if err != nil {
		logrus.WithError(err).Error("Failed starting debug http server")
	}
}

func main() {
	go debugSrv()

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

	err = engine.RestoreApps(session)
	if err != nil {
		logrus.WithError(err).Fatal("Failed restoring apps")
	}

	err = session.Open()
	if err != nil {
		logrus.WithError(err).Fatal("Failed Connecting to discord")
	}

	logrus.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	logrus.Info("Shutting down, saving all running apps...")
	err = engine.StopAndSaveStates()
	if err != nil {
		logrus.WithError(err).Error("Failed stopping one or more apps.")
	} else {
		logrus.Info("All apps saved")
	}
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
