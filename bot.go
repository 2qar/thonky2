package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bigheadgeorge/spreadsheet"
	"github.com/bigheadgeorge/thonky2/db"
	"github.com/bigheadgeorge/thonky2/schedule"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2/google"
)

var (
	botUserID string

	// DB is used for accessing the PSQL db
	DB *db.Handler

	// Client is an authenticated http client for accessing Google APIs
	Client *http.Client

	// Service is the service used to grab spreadsheets
	Service *spreadsheet.Service
)

func main() {
	if _, err := os.Open("config.json"); os.IsNotExist(err) {
		panic(fmt.Errorf("no config file; rename config.json.example to config.json and fill the fields"))
	}
	b, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	config := struct {
		Token        string
		GoogleAPIKey string `json:"google_api_key"`
	}{}
	err = json.Unmarshal(b, &config)
	if err != nil {
		panic(err)
	}
	if config.Token == "" {
		panic(fmt.Errorf("no token in config.json"))
	} else if config.GoogleAPIKey == "" {
		panic("no google api key in config.json")
	}

	h, err := db.NewHandler()
	if err != nil {
		panic(err)
	}
	DB = &h
	defer DB.Close()

	b, err = ioutil.ReadFile("service_account.json")
	if err != nil {
		panic(err)
	}
	c, err := google.JWTConfigFromJSON(b, spreadsheet.Scope, schedule.DriveScope)
	if err != nil {
		panic(err)
	}
	Client = c.Client(context.Background())
	Service = spreadsheet.NewServiceWithClient(Client)

	logFile := StartLog()

	d, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		panic(err)
	}

	d.AddHandler(messageCreate)
	d.AddHandler(ready)

	err = d.Open()
	if err != nil {
		panic(err)
	}

	err = StartReminders(d)
	if err != nil {
		log.Println(err)
	}

	log.Println("running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	CompressLog(logFile)
	d.Close()
}

func ready(s *discordgo.Session, r *discordgo.Ready) {
	log.Println("ready")
	botUserID = r.User.ID

	for _, guild := range r.Guilds {
		info, err := NewGuildInfo(guild.ID)
		if err != nil {
			log.Println(err)
			continue
		}
		guildInfo[guild.ID] = info
		log.Println("added config for", guild.ID)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		args := strings.Split(m.Content, " ")
		for name, command := range Commands {
			if string(args[0][1:]) == name {
				msg, err := command.Call(s, m, args)
				if err != nil {
					log.Println(err)
				}
				if msg != "" {
					s.ChannelMessageSend(m.ChannelID, msg)
				}
			}
		}
	}
}
