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

	// guildInfo holds the config for each guild the bot is in
	guildInfo = make(map[string]*GuildInfo)
)

// GetInfo returns TeamInfo or GuildInfo, depending on what it finds with the given channelID and guildID
func GetInfo(guildID, channelID string) (*TeamInfo, error) {
	info := guildInfo[guildID]
	if info != (&GuildInfo{}) {
		for _, team := range info.Teams {
			for _, id := range team.Channels {
				if string(id) == channelID {
					return team, nil
				}
			}
			return info.TeamInfo, nil
		}
	}
	return &TeamInfo{}, fmt.Errorf("no info for guild [%s]", guildID)
}

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
	d.AddHandler(ready)

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

func ready(s *discordgo.Session, r *discordgo.Ready) {
	for _, guild := range r.Guilds {
		info, err := GetGuildInfo(guild.ID)
		if err != nil {
			fmt.Println(err)
			continue
		}
		guildInfo[guild.ID] = info
		fmt.Println("added config for", guild.ID)
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
