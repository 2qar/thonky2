package commands

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/db"
	"github.com/bigheadgeorge/thonky2/pkg/gamebattles"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

const gamebattlesLogo = "https://gamebattles.majorleaguegaming.com/gb-web/assets/favicon.ico"

func init() {
	examples := [][2]string{
		{"!gamebattles Feeders", "Search gamebattles for a team with \"Feeders\" in the name (case-sensitive)."},
		{"!gb Feeders", "Same as above, just a shortcut :)"},
	}
	command.AddCommand("gamebattles", "Get info about other teams in a Gamebattles tournament.", examples, Gamebattles).AddAliases("gb")
}

// Gamebattles gets team information off of gamebattles.
func Gamebattles(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	var teamStats TeamStats
	msg, err := getTeamStats(s, m, searchGamebattles, matchBattlefy, &teamStats)
	if len(msg) > 0 || err != nil {
		return msg, err
	}

	embed := formatTeamStats(teamStats.Team, convertPlayers(teamStats.Players))
	embed.Color = 0x22242C
	embed.Author.IconURL = gamebattlesLogo
	s.Session.ChannelMessageSendEmbed(m.ChannelID, &embed)
	return "", nil
}

// gamebattlesPlayers converts a slice of gamebattles players to a slice of generic players.
func gamebattlesPlayers(players []gamebattles.Player) []Player {
	genericPlayers := make([]Player, len(players))
	for i := range players {
		genericPlayers[i] = players[i]
	}
	return genericPlayers
}

func searchGamebattles(db *db.Handler, team_id int, name string, teamStats *TeamStats) (string, error) {
	var tournamentLink string
	err := db.QueryRow("SELECT tournament_link FROM gamebattles WHERE team = $1", team_id).Scan(&tournamentLink)
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
	} else if len(foundTeams) == 0 {
		return fmt.Sprintf("No teams in the tournament have %q in their name.", name), nil
	}

	// pepega Xd
	foundTeam, err := gamebattles.GetTeam(strconv.FormatUint(uint64(foundTeams[0].ID), 10))
	if err != nil {
		return fmt.Sprintf("Error grabbing players for team %s: %s", foundTeams[0].Name(), err), err
	}
	teamStats.Team = foundTeam
	teamStats.Players = gamebattlesPlayers(foundTeam.Players)

	return "", nil
}

func matchGamebattles(db *db.Handler, team, round int, teamStats *TeamStats) (string, error) {
	// TODO: figure out how the rounds api stuff works on gamebattles
	//       https://gamebattles.majorleaguegaming.com/pc/overwatch/tournament/Breakable-Barriers-NA-1/bracket
	return "i haven't actually done this part yet", nil
}
