package main

import (
	"log"
	"regexp"

	"github.com/bigheadgeorge/thonky2/db"
	"github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

func init() {
	examples := [][2]string{
		{"!add_team Test #general", "Add a team with the name \"Test\" in #general chat"},
	}
	AddCommand("add_team", "Add a team to the server.", examples, AddTeam)
}

// AddTeam adds a team to a guild
func AddTeam(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) != 3 {
		if len(args) == 2 {
			s.ChannelMessageSend(m.ChannelID, "Bad amount of args; no channel given!")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Bad amount of args.")
		}
		return
	}
	matched, err := regexp.MatchString(`<#\d{18}>`, args[2])
	if err != nil {
		log.Println(err)
		return
	} else if !matched {
		s.ChannelMessageSend(m.ChannelID, "Invalid channel.")
		return
	}
	channelID := args[2][2 : len(args[2])-1]
	me, err := s.User("@me")
	perms, err := s.State.UserChannelPermissions(me.ID, channelID)
	if err != nil {
		log.Println(err)
		return
	} else if perms&discordgo.PermissionSendMessages == 0 {
		s.ChannelMessageSend(m.ChannelID, "I don't have permission to send messages in that channel. :(")
		return
	}

	handler, err := db.NewHandler()
	if err != nil {
		log.Println(err)
		return
	}
	defer handler.Close()

	r, err := handler.Query("SELECT team_name FROM teams WHERE $1 = ANY(channels)", channelID)
	if err != nil {
		log.Println(err)
		return
	} else if r.Next() {
		defer r.Close()
		var teamName string
		err = r.Scan(&teamName)
		if err != nil {
			log.Println(err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Channel already occupied by "+teamName)
		return
	}
	r.Close()

	template, err := handler.GetGuild(m.GuildID)
	if err != nil {
		log.Println(err)
		return
	}
	r, err = handler.Query("INSERT INTO teams (server_id, team_name, channels, remind_activities, remind_intervals, update_interval) VALUES ($1, $2, $3, $4, $5, $6)", m.GuildID, args[1], pq.StringArray([]string{channelID}), template.RemindActivities, template.RemindIntervals, template.UpdateInterval)
	if err != nil {
		log.Println(err)
		return
	}
	r.Close()

	s.ChannelMessageSend(m.ChannelID, "Added team.")
	log.Printf("added team %q to guild [%s]\n", args[1], m.GuildID)
}
