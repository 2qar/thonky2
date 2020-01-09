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
	var teamStats TeamStats
	msg, err := getTeamStats(m, searchBattlefy, matchBattlefy, &teamStats)
	if len(msg) > 0 || err != nil {
		return msg, err
	}

	// bold active players
	players := convertPlayers(teamStats.Players)
	for _, player := range teamStats.Players {
		if player.Active() {
			for o := range players {
				if player.Battletag() == players[o].BTag {
					players[o].BTag = "**" + players[o].BTag + "**"
				}
			}
		}
	}

	embed := formatTeamStats(teamStats.Team, players)
	embed.Color = 0xe74c3c
	embed.Author.IconURL = battlefyLogo
	s.ChannelMessageSendEmbed(m.ChannelID, &embed)
	return "", nil
}

// battlefyPlayers converts a slice of battlefy players to a slice of generic players.
func battlefyPlayers(players []battlefy.Player) []Player {
	genericPlayers := make([]Player, len(players))
	for i := range players {
		genericPlayers[i] = players[i]
	}
	return genericPlayers
}

// searchBattlefy populates the given []Player and ODTeam with Battlefy search results.
func searchBattlefy(team_id int, name string, teamStats *TeamStats) (string, error) {
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

	teamStats.Team = team
	teamStats.Players = battlefyPlayers(team.Players)
	return "", nil
}

// matchBattlefy gets stats on the opposing team in the given round of the tournament.
func matchBattlefy(team_id int, round int, teamStats *TeamStats) (string, error) {
	var tournamentLink string
	var teamID string
	err := DB.QueryRow("SELECT tournament_link, team_id FROM battlefy WHERE team = $1", team_id).Scan(&tournamentLink, &teamID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No Battlefy config; use !set_tournament.", nil
		}
		return fmt.Sprintf("Error getting Battlefy config: %s", err), err
	}
	t, err := battlefy.FindMatch(tournamentLink, teamID, round)
	if err != nil {
		return fmt.Sprintf("Error grabbing team info from Battlefy: %s", err), err
	}

	teamStats.Team = t
	teamStats.Players = battlefyPlayers(t.Players)
	return "", nil
}
