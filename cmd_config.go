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
		{"!set_tournament https://battlefy.com/overwatch-open-division-north-america/2019-overwatch-open-division-practice-season-north-america/5d6fdb02c747ff732da36eb4/stage/5d7b716bb7758c268b771f83/bracket/1", "Update the current tournament to a Battlefy tournament."},
		{"!set_tournament https://gamebattles.majorleaguegaming.com/pc/overwatch/tournament/Breakable-Barriers-EMEA-2", "Update the current tournament to a Gamebattles tournament."},
		{"!set tournament https://gamebattles.majorleaguegaming.com/pc/overwatch/tournament/Breakable-Barriers-EMEA-2 https://gamebattles.majorleaguegaming.com/pc/overwatch/team/33834248", "Update the current tournament and team."},
	}
	AddCommand("set_tournament", "Update the current tournament and team.", examples, SetTournament).AddAliases("set_tourney")
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

	if team.Name == "" {
		return "No team in this channel.", nil
	}

	givenChannels := args[1:]
	for _, arg := range givenChannels {
		if !isChannel(arg) {
			return fmt.Sprintf("Invalid channel %q.", arg), nil
		} else if name, err := DB.GetName(channelID(arg)); err == nil {
			if name == team.Name {
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

// SetTournament updates the tournament a team is participating in and, optionally, their team on the tournament site.
func SetTournament(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "Error grabbing team", nil
	}

	if len(args) > 3 {
		return "Too many arguments.", nil
	} else if len(args) == 1 {
		return "No arguments; missing tournament link and team link.", nil
	}

	tournamentRegexes := []string{
		`https://battlefy.com/[\w\d-]{1,}/[\w\d-]{1,}/[\d\w]{24}/stage/[\d\w]{24}`,
		`https://gamebattles.majorleaguegaming.com/pc/.+/tournament/[\w\d-]+`,
	}
	var url, re string
	var site int
	for site, re = range tournamentRegexes {
		url = regexp.MustCompile(re).FindString(args[1])
		if len(url) > 0 {
			break
		}
	}
	var siteTable string
	switch site {
	case 0:
		siteTable = "battlefy"
	case 1:
		siteTable = "gamebattles"
	}
	if len(url) == 0 {
		return "Invalid / unsupported tournament URL.", nil
	} else if len(args) == 2 {
		err := DB.QueryRow(fmt.Sprintf("SELECT team FROM %s WHERE team = $1", siteTable), team.ID).Scan(&team.ID)
		if err == sql.ErrNoRows {
			return "No config yet; give me both your tournament link AND your team link.", err
		}
	}

	var teamURL string
	if len(args) == 3 {
		teamRegexes := []string{
			`https://battlefy.com/teams/.+`,
			`https://gamebattles.majorleaguegaming.com/pc/.+/team/\d+`,
		}
		teamURL = regexp.MustCompile(teamRegexes[site]).FindString(args[2])
		if len(teamURL) == 0 {
			return "Incompatible team link; your tournament and team links are from two different websites.", nil
		}
	}

	var err error
	teamID := teamURL[strings.LastIndex(teamURL, "/")+1:]
	switch site {
	case 0:
		//_, err = DB.Exec("UPDATE battlefy SET stage_id = $1, tournament_link = $2, team_id = $3 WHERE team = $4", url[strings.LastIndex(url, "/")+1:], url, teamID, team.ID)
		_, err = DB.Exec("INSERT INTO battlefy (team, stage_id, tournament_link, team_id) VALUES ($1, $2, $3, $4) ON CONFLICT (team) DO UPDATE SET stage_id = EXCLUDED.stage_id, tournament_link = EXCLUDED.tournament_link, team_id = EXCLUDED.team_id", team.ID, url[strings.LastIndex(url, "/")+1:], url, teamID)
	case 1:
		log.Println("updating gamebattles")
		//_, err = DB.Exec("UPDATE gamebattles SET tournament_link = $1, team_id = $2 WHERE team = $3", url, teamID, team.ID)
		_, err = DB.Exec("INSERT INTO gamebattles (team, tournament_link, team_id) VALUES ($1, $2, $3) ON CONFLICT (team) DO UPDATE SET tournament_link = EXCLUDED.tournament_link, team_id = EXCLUDED.team_id", team.ID, url, teamID)
	}
	if err != nil {
		return "Error updating tournament: " + err.Error(), err
	}
	return "Updated tournament. :)", nil
}
