package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/bigheadgeorge/thonky2/gamebattles"
	"github.com/bwmarrin/discordgo"
)

const gamebattlesLogo = "https://gamebattles.majorleaguegaming.com/gb-web/assets/favicon.ico"

func init() {
	examples := [][2]string{
		{"!gamebattles Feeders", "Search gamebattles for a team with \"Feeders\" in the name (case-sensitive)."},
		{"!gb Feeders", "Same as above, just a shortcut :)"},
	}
	AddCommand("gamebattles", "Get info about other teams in a Gamebattles tournament.", examples, Gamebattles).AddAliases("gb")
}

// Gamebattles gets team information off of gamebattles.
func Gamebattles(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	var embed discordgo.MessageEmbed
	msg, err := OD(m, searchGamebattles, matchBattlefy, &embed)
	if len(msg) > 0 || err != nil {
		return msg, err
	}

	embed.Color = 0x22242C
	embed.Author.IconURL = gamebattlesLogo
	s.ChannelMessageSendEmbed(m.ChannelID, &embed)
	return "", nil
}

func searchGamebattles(team_id int, name string, odi *ODInfo) (string, error) {
	var tournamentLink string
	err := DB.QueryRow("SELECT tournament_link FROM gamebattles WHERE team = $1", team_id).Scan(&tournamentLink)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No config for Gamebattles; use !set_tournament.", nil
		}
		return fmt.Sprintf("Error grabbing Gamebattles config: %s", err), err
	}

	urlSplit := strings.Split(tournamentLink, "/")
	id, err := gamebattles.GetTournamentID(urlSplit[6], urlSplit[4], urlSplit[3])
	if err != nil {
		return fmt.Sprintf("Error getting tournament ID: %s", id), err
	}
	teams, err := gamebattles.GetTeams(id)
	if err != nil {
		return fmt.Sprintf("Error getting participant list: %s", id), err
	}
	var foundTeams []gamebattles.Team
	for _, team := range teams {
		if strings.Contains(team.TeamName, name) {
			foundTeams = append(foundTeams, team)
		}
	}

	if len(foundTeams) > 1 {
		var names []string
		for _, team := range foundTeams {
			names = append(names, team.TeamName)
		}
		return formatNames(names), nil
	}

	// pepega Xd
	foundTeam, err := gamebattles.GetTeam(strconv.FormatUint(uint64(foundTeams[0].ID), 10))
	if err != nil {
		return fmt.Sprintf("Error grabbing players for team %s: %s", foundTeams[0].Name(), err), err
	}
	odi.Team = foundTeam
	odi.Players = make([]Player, len(foundTeam.Players))
	for i := range foundTeam.Players {
		odi.Players[i] = foundTeam.Players[i]
	}

	return "", nil
}

func matchGamebattles(team, round int, odi *ODInfo) (string, error) {
	// TODO: figure out how the rounds api stuff works on gamebattles
	//       https://gamebattles.majorleaguegaming.com/pc/overwatch/tournament/Breakable-Barriers-NA-1/bracket
	return "i haven't actually done this part yet", nil
}
