package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	spreadsheet "gopkg.in/Iwark/spreadsheet.v2"
)

var (
	// Token is the token for the bot
	Token string

	// SheetID is the ID of the sheet to grab info from
	SheetID = "19LIrH878DY9Ltaux3KlfIenmMFfPTA16NWnnQQMHG0Y"

	// Service is the service used to grab spreadsheets
	Service *spreadsheet.Service
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot token")
	flag.Parse()

	if Token == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	d, err := discordgo.New("Bot " + Token)
	if err != nil {
		panic(err)
	}

	d.AddHandler(messageCreate)

	err = d.Open()
	if err != nil {
		panic(err)
	}

	Service, err = spreadsheet.NewService()
	if err != nil {
		panic(err)
	}

	fmt.Println("running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	d.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		args := strings.Split(m.Content, " ")
		for name, command := range Commands {
			if string(args[0][1:]) == name {
				command.Call(s, m, args)
			}
		}
	}
}
