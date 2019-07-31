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
	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	botUserID string

	// Service is the service used to grab spreadsheets
	Service *spreadsheet.Service

	// FilesService is the service used to grab spreadsheet metadata
	FilesService *drive.FilesService

	// SpreadsheetsService is the service for grabbing Spreadsheet info not exposed by gopkg.in/Iwark/spreadsheet.v2
	SpreadsheetsService *sheets.SpreadsheetsService
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

	b, err = ioutil.ReadFile("service_account.json")
	if err != nil {
		panic(err)
	}
	c, err := google.JWTConfigFromJSON(b, spreadsheet.Scope)
	if err != nil {
		panic(err)
	}
	Service = spreadsheet.NewServiceWithClient(c.Client(oauth2.NoContext))

	ctx := context.Background()
	opt := option.WithAPIKey(config.GoogleAPIKey)
	service, err := drive.NewService(ctx, opt)
	if err != nil {
		panic(err)
	}
	FilesService = drive.NewFilesService(service)

	sheetService, err := sheets.NewService(ctx, opt)
	if err != nil {
		panic(err)
	}
	SpreadsheetsService = sheets.NewSpreadsheetsService(sheetService)

	logFile := StartLog()

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
				command.Call(s, m, args)
			}
		}
	}
}
