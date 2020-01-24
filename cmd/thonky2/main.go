package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bigheadgeorge/spreadsheet"
	_ "github.com/bigheadgeorge/thonky2/internal/commands"
	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/db"
	"github.com/bigheadgeorge/thonky2/pkg/reminders"
	"github.com/bigheadgeorge/thonky2/pkg/schedule"
	botstate "github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bigheadgeorge/thonky2/pkg/team"
	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2/google"
)

var state botstate.State

type config struct {
	Token        string
	GoogleAPIKey string `json:"google_api_key"`
	User         string
	Pw           string
	Host         string
	Database     string
}

func main() {
	state.Schedules = make(map[string]*schedule.Schedule)

	if _, err := os.Open("config.json"); os.IsNotExist(err) {
		panic(fmt.Errorf("no config file; rename config.json.example to config.json and fill the fields"))
	}
	b, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	var config config
	err = json.Unmarshal(b, &config)
	if err != nil {
		panic(err)
	}
	if config.Token == "" {
		panic(fmt.Errorf("no token in config.json"))
	} else if config.GoogleAPIKey == "" {
		panic("no google api key in config.json")
	}

	d, err := sqlx.Open("postgres", fmt.Sprintf("user=%s password=%s host=%s dbname=%s", config.User, config.Pw, config.Host, config.Database))
	if err != nil {
		panic(err)
	}
	state.DB = &db.Handler{d}

	b, err = ioutil.ReadFile("service_account.json")
	if err != nil {
		panic(err)
	}
	c, err := google.JWTConfigFromJSON(b, spreadsheet.Scope, schedule.DriveScope)
	if err != nil {
		panic(err)
	}
	state.Client = c.Client(context.Background())
	state.Service = spreadsheet.NewServiceWithClient(state.Client)

	state.Session, err = discordgo.New("Bot " + config.Token)
	if err != nil {
		panic(err)
	}

	state.Session.AddHandler(messageCreate)
	state.Session.AddHandler(ready)

	err = state.Session.Open()
	if err != nil {
		panic(err)
	}
	defer state.Session.Close()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime + log.Lshortfile)

	reminders.Init()
	reminders.Start()

	log.Println("running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func ready(s *discordgo.Session, r *discordgo.Ready) {
	log.Println("ready")

	for _, guild := range r.Guilds {
		var teams []*team.Team
		err := state.DB.Select(&teams, "SELECT * FROM teams WHERE server_id = $1", guild.ID)
		if err != nil {
			log.Printf("error grabbing teams in guild [%s]: %s\n", guild.ID, err)
			continue
		}

		var config reminders.Config
		var spreadsheetID string
		var updateInterval int
		for _, team := range teams {
			err = state.DB.Get(&config, "SELECT * FROM reminders WHERE team = $1", team.ID)
			if err != nil {
				log.Printf("error grabbing reminders for team %d: %s\n", team.ID, err)
			} else {
				reminders.AddReminder(reminders.Reminder{State: &state, Team: team, Config: &config})
			}

			err = state.DB.QueryRow("SELECT spreadsheet_id, update_interval FROM schedules WHERE team = $1", team.ID).Scan(&spreadsheetID, &updateInterval)
			if err != nil {
				log.Printf("error grabbing spreadsheet info for team %d: %s\n", team.ID, err)
			} else if state.Schedules[spreadsheetID] == nil {
				state.Schedules[spreadsheetID], err = fetchSchedule(&state, spreadsheetID, updateInterval)
				if err != nil {
					log.Printf("error grabbing spreadsheet for team %d: %s\n", team.ID, err)
				} else {
					log.Printf("grabbed spreadsheet [%s]\n", spreadsheetID)
				}
			}

			log.Printf("added config for team %d\n", team.ID)
		}
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		args := strings.Split(m.Content, " ")
		for _, c := range command.Commands {
			if c.Match(args[0][1:]) {
				msg, err := c.Call(&state, m, args)
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
