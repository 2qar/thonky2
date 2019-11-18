package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

func init() {
	examples := [][2]string{
		{"!add_team Test #general", "Add a team with the name \"Test\" in #general chat"},
	}
	AddCommand("add_team", "Add a team to the server.", examples, AddTeam)

	examples = [][2]string{
		{"!add_channel #general-2", "Add #general-2 to the team in this channel."},
		{"!add_channels #general-2 #general-3", "Add #general-2 and #general-3 to the team in this channel."},
	}
	AddCommand("add_channel", "Add one or more channels to a team.", examples, AddChannels).AddAliases("add_channels")

	examples = [][2]string{
		{"!save", "Save the current week schedule as default"},
	}
	AddCommand("save", "Save the week schedule", examples, Save)

	examples = [][2]string{
		{"!set_tournament https://battlefy.com/overwatch-open-division-north-america/2019-overwatch-open-division-practice-season-north-america/5d6fdb02c747ff732da36eb4/stage/5d7b716bb7758c268b771f83/bracket/1", "Update the current tournament"},
	}
	AddCommand("set_tournament", "Update the current tournament", examples, SetTournament).AddAliases("set_tourney")
}

func isChannel(s string) bool {
	match, _ := regexp.MatchString(`<#\d{18}>`, s)
	return match
}

// channelID gets the channelID from a channel mention, ex. <#477928874450354176> returns 477928874450354176
func channelID(s string) string {
	return s[2 : len(s)-1]
}

// sendPermission checks whether the bot has permission to send messages in a channel
func sendPermission(s *discordgo.Session, channelID string) (bool, error) {
	perms, err := s.State.UserChannelPermissions(botUserID, channelID)
	if err != nil {
		return false, err
	}
	return perms&discordgo.PermissionSendMessages != 0, nil
}

// AddTeam adds a team to a guild
func AddTeam(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	info := guildInfo[m.GuildID]
	if info == nil {
		return "No info for this guild.", fmt.Errorf("no info for [%s]\n", m.GuildID)
	}

	if len(args) != 3 {
		if len(args) == 2 {
			return "Bad amount of args; no channel given!", nil
		} else {
			return "Bad amount of args.", nil
		}
	}
	if !isChannel(args[2]) {
		return "Invalid channel", nil
	}
	chanID := channelID(args[2])
	canSend, err := sendPermission(s, chanID)
	if err != nil {
		return err.Error(), err
	} else if !canSend {
		return "I don't have permission to send messages in that channel. :(", nil
	}

	if name, err := DB.GetTeamName(chanID); err == nil {
		return fmt.Sprintf("Channel already occupied by %q", name), nil
	}

	config, err := DB.GetGuild("0")
	if err != nil {
		return err.Error(), err
	}
	config.GuildID = m.GuildID
	config.TeamName = args[1]
	channelInt, err := strconv.ParseInt(chanID, 10, 64)
	if err != nil {
		return "Error adding team, something stupid happened", err
	}
	config.Channels = pq.Int64Array([]int64{channelInt})
	r, err := DB.Query("INSERT INTO teams (server_id, team_name, channels, remind_activities, remind_intervals, update_interval) VALUES ($1, $2, $3, $4, $5, $6)", config.GuildID, config.TeamName, config.Channels, config.RemindActivities, config.RemindIntervals, config.UpdateInterval)
	if err != nil {
		return err.Error(), err
	}
	defer r.Close()

	err = info.AddTeam(config)
	if err != nil {
		return "Error adding team: " + err.Error(), err
	}
	log.Printf("added team %q to guild [%s]\n", args[1], m.GuildID)
	return "Added team.", nil
}

// AddChannels adds channels to the team in the channel the command is called from
func AddChannels(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	if len(args) < 2 {
		return "No channels given!", nil
	}

	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		return "No config for this guild.", nil
	}

	if info.TeamName == "" {
		return "No team in this channel.", nil
	}

	givenChannels := args[1:]
	for _, arg := range givenChannels {
		if !isChannel(arg) {
			return fmt.Sprintf("Invalid channel %q.", arg), nil
		} else if name, err := DB.GetTeamName(channelID(arg)); err == nil {
			if name == info.TeamName {
				return arg + " already added.", nil
			} else {
				return fmt.Sprintf("%s already occupied by %q.", arg, name), nil
			}
		}
		canSend, err := sendPermission(s, arg[2:len(arg)-1])
		if err != nil {
			log.Println(err)
		} else if !canSend {
			return "I don't have permission to send messages in that channel. :(", nil
		}
	}

	for _, id := range info.Channels {
		for i, givenID := range givenChannels {
			if strconv.FormatInt(id, 10) == givenID[2:len(givenID)-1] {
				givenChannels = append(givenChannels[:i], givenChannels[i+1:]...)
			}
		}
	}

	for i, channel := range givenChannels {
		givenChannels[i] = channelID(channel)
	}

	for _, id := range givenChannels {
		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return "Error updating channels.", err
		}
		info.Channels = append(info.Channels, i)
	}

	r, err := DB.Query("UPDATE teams SET channels = $1 WHERE server_id = $2 AND team_name = $3", info.Channels, m.GuildID, info.TeamName)
	defer r.Close()
	if err != nil {
		return "Error updating channels.", err
	}

	return "Added Channels.", nil
}

// Save saves a sheet's current week schedule for resetting to
func Save(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		return "Error grabbing info: " + err.Error(), err
	}

	var b []byte
	b, err = json.Marshal(info.Week)
	if err != nil {
		return "Error encoding schedule, something stupid happened", err
	}

	r, err := DB.Query("SELECT id FROM sheet_info WHERE id = $1", info.DocKey)
	if err != nil {
		return "Error querying database, something stupid happened", nil
	}
	defer r.Close()

	if r.Next() {
		_, err := DB.Query("UPDATE sheet_info SET default_week = $1 WHERE id = $2", b, info.DocKey)
		if err != nil {
			return "Error updating default", err
		}
	} else {
		_, err := DB.Query("INSERT INTO sheet_info (id, default_week) VALUES ($1, $2)", info.DocKey, b)
		if err != nil {
			return "Error setting default", err
		}
	}
	return "Updated default week schedule. :)", nil
}

// SetTournament sets the battlefy tournament for the current team
func SetTournament(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error grabbing info")
		return
	}

	if len(args) > 2 {
		s.ChannelMessageSend(m.ChannelID, "Too many arguments")
		return
	} else if len(args) == 1 {
		s.ChannelMessageSend(m.ChannelID, "No URL given")
		return
	}

	url := regexp.MustCompile(`https://battlefy.com/[\w\d-]{1,}/[\w\d-]{1,}/[\d\w]{24}/stage/[\d\w]{24}`).FindString(args[1])
	if url == "" {
		s.ChannelMessageSend(m.ChannelID, "Invalid tournament URL")
		return
	}

	// TODO: merge "server_config" and "teams" table so i don't have to do this shit
	if info.TeamName == "" {
		_, err = DB.Exec("UPDATE server_config SET tournament_link = $1 WHERE server_id = $2", url, info.GuildID)
	} else {
		_, err = DB.Exec("UPDATE teams SET stage_id = $1 WHERE server_id = $2 AND team_name = $3", url[strings.LastIndex(url, "/"):], info.GuildID, info.TeamName)
	}
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error updating tournament url: "+err.Error())
		return
	}
	s.ChannelMessageSend(m.ChannelID, "Updated tournament URL. :)")
}
