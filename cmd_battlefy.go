package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bigheadgeorge/thonky2/battlefy"
	"github.com/bwmarrin/discordgo"
)

const battlefyLogo = "http://s3.amazonaws.com/battlefy-assets/helix/images/logos/logo.png"

func init() {
	examples := [][2]string{
		{"!battlefy Feeders", "Search the current tournament for teams with \"Feeders\" in their name."},
		{"!bf Feeders", "Same as above, but it's a shortcut. :)"},
	}
	AddCommand("battlefy", "Get team info from the current Battlefy tournament.", examples, Battlefy).AddAliases("bf")
}

// Battlefy gets team information from Battlefy.
func Battlefy(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	var embed discordgo.MessageEmbed
	msg, err := getTeamStats(m, searchBattlefy, matchBattlefy, &embed)
	if len(msg) > 0 || err != nil {
		return msg, err
	}

	embed.Color = 0xe74c3c
	embed.Author.IconURL = battlefyLogo

	s.ChannelMessageSendEmbed(m.ChannelID, &embed)
	return "", nil
}

// searchBattlefy populates the given []Player and ODTeam with Battlefy search results.
func searchBattlefy(team_id int, name string, odi *ODInfo) (string, error) {
	var tournamentLink string
	err := DB.QueryRow("SELECT tournament_link FROM battlefy WHERE team = $1", team_id).Scan(&tournamentLink)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No Battlefy config for this guild; use !set_tournament.", nil
		}
		return fmt.Sprintf("Error getting Battlefy config: %s", err), err
	}
	tournamentID := strings.Split(tournamentLink, "/")[5]
	var team battlefy.Team
	names, err := battlefy.FindTeam(tournamentID, name, &team)
	if err != nil {
		return fmt.Sprintf("Error searching Battlefy: %s", err), err
	}

	if len(names) > 1 {
		return formatNames(names), nil
	}

	odi.Team = team
	odi.Players = make([]Player, len(team.Players))
	for i := range team.Players {
		odi.Players[i] = team.Players[i]
	}
	return "", nil
}

// matchBattlefy gets stats on the opposing team in the given round of the tournament.
func matchBattlefy(team_id int, round int, odi *ODInfo) (string, error) {
	var tournamentLink string
	var teamID string
	err := DB.QueryRow("SELECT tournament_link, team_id FROM battlefy WHERE team = $1", team_id).Scan(&tournamentLink, &teamID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No Battlefy config; use !set_tournament.", nil
		}
		return fmt.Sprintf("Error getting Battlefy config: %s", err), err
	}
	t, err := battlefy.GetOtherTeam(tournamentLink, teamID, round)
	if err != nil {
		return fmt.Sprintf("Error grabbing team info from Battlefy: %s", err), err
	}

	odi.Team = t
	odi.Players = make([]Player, len(t.Players))
	for i := range t.Players {
		odi.Players[i] = t.Players[i]
	}
	return "", nil
}
