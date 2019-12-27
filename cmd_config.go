package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
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
	perms, err := s.State.UserChannelPermissions(s.State.User.ID, channelID)
	if err != nil {
		return false, err
	}
	return perms&discordgo.PermissionSendMessages != 0, nil
}

// AddTeam adds a team to a guild
func AddTeam(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := GuildTeam(m.GuildID)
	if team == nil {
		return "No team for this guild.", fmt.Errorf("no team for [%s]\n", m.GuildID)
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

	if name, err := DB.GetName(chanID); err == nil {
		return fmt.Sprintf("Channel already occupied by %q", name), nil
	}

	err = DB.AddTeam(m.GuildID, args[1], chanID)
	if err != nil {
		return err.Error(), err
	}

	log.Printf("added team %q to guild [%s]\n", args[1], m.GuildID)
	return "Added team.", nil
}

// AddChannels adds channels to the team in the channel the command is called from
func AddChannels(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	if len(args) < 2 {
		return "No channels given!", nil
	}

	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No config for this guild.", nil
	}

	if team.Name.String == "" {
		return "No team in this channel.", nil
	}

	givenChannels := args[1:]
	for _, arg := range givenChannels {
		if !isChannel(arg) {
			return fmt.Sprintf("Invalid channel %q.", arg), nil
		} else if name, err := DB.GetName(channelID(arg)); err == nil {
			if name == team.Name.String {
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

	for _, id := range team.Channels {
		for i, givenID := range givenChannels {
			if id == givenID[2:len(givenID)-1] {
				givenChannels = append(givenChannels[:i], givenChannels[i+1:]...)
			}
		}
	}

	for i, channel := range givenChannels {
		givenChannels[i] = channelID(channel)
	}

	for _, id := range givenChannels {
		team.Channels = append(team.Channels, id)
	}

	r, err := DB.Query("UPDATE teams SET channels = $1 WHERE server_id = $2 AND team_name = $3", team.Channels, m.GuildID, team.Name)
	defer r.Close()
	if err != nil {
		return "Error updating channels.", err
	}

	return "Added Channels.", nil
}

// Save saves a sheet's current week schedule for resetting to
func Save(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No teams in this server / channel", nil
	}
	var spreadsheetID string
	err := DB.QueryRow("SELECT spreadsheet_id FROM schedules WHERE team = $1", team.ID).Scan(&spreadsheetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No spreadsheet configured.", nil
		}
		return fmt.Sprintf("Error grabbing spreadsheet id: %s", err.Error()), err
	}

	b, err := json.Marshal(team.Schedule().Week)
	if err != nil {
		return "Error encoding schedule, something stupid happened", err
	}

	r, err := DB.Query("SELECT id FROM sheet_info WHERE id = $1", spreadsheetID)
	if err != nil {
		return "Error querying database, something stupid happened", nil
	}
	defer r.Close()

	if r.Next() {
		_, err := DB.Query("UPDATE sheet_info SET default_week = $1 WHERE id = $2", b, spreadsheetID)
		if err != nil {
			return "Error updating default", err
		}
	} else {
		_, err := DB.Query("INSERT INTO sheet_info (id, default_week) VALUES ($1, $2)", spreadsheetID, b)
		if err != nil {
			return "Error setting default", err
		}
	}
	return "Updated default week schedule. :)", nil
}

// SetTournament sets the battlefy tournament for the current team
func SetTournament(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "Error grabbing team", nil
	}

	if len(args) > 2 {
		return "Too many arguments", nil
	} else if len(args) == 1 {
		return "No URL given.", nil
	}

	url := regexp.MustCompile(`https://battlefy.com/[\w\d-]{1,}/[\w\d-]{1,}/[\d\w]{24}/stage/[\d\w]{24}`).FindString(args[1])
	if url == "" {
		return "Invalid tournament URL", nil
	}

	_, err := DB.Exec("UPDATE teams SET stage_id = $1 WHERE server_id = $2 AND team_name = $3", url[strings.LastIndex(url, "/"):], team.GuildID, team.Name)
	if err != nil {
		return "Error updating tournament url: " + err.Error(), err
	}
	_, err = DB.Exec("UPDATE teams SET tournament_link = $1 WHERE server_id = $2 AND team_name = $3", url, team.GuildID, team.Name)
	if err != nil {
		return "Error updating tournament url: " + err.Error(), err
	}
	return "Updated tournament URL. :)", nil
}
