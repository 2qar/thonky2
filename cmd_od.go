package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bigheadgeorge/goverbuff"
	"github.com/bigheadgeorge/thonky2/battlefy"
	"github.com/bigheadgeorge/thonky2/gamebattles"
	"github.com/bwmarrin/discordgo"
)

const battlefyLogo = "http://s3.amazonaws.com/battlefy-assets/helix/images/logos/logo.png"

const (
	SiteUndefined int = iota - 1
	SiteBattlefy
	SiteGamebattles
)

func init() {
	examples := [][2]string{
		{"!od 1", "Get info on the other team in round 1."},
		{"!od cloud9", "Get info on cloud9, if they're in our tournament."},
	}
	AddCommand("od", "Get information about another team or a round of Open Division", examples, OD)
}

// Player has methods for getting information about a player.
type Player interface {
	Battletag() string
}

// ODTeam has methods for getting team info.
type ODTeam interface {
	Name() string
	Logo() string
	Link() string
}

// OD grabs information about another team
func OD(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No config for this guild.", nil
	} else if team.ODSite == SiteUndefined {
		return "No site configured; use !set_tournament.", nil
	} else if len(args) == 1 {
		return "No args.", nil
	}

	var siteLogoURL string
	switch team.ODSite {
	case SiteBattlefy:
		siteLogoURL = "http://s3.amazonaws.com/battlefy-assets/helix/images/logos/logo.png"
	case SiteGamebattles:
		siteLogoURL = "https://gamebattles.majorleaguegaming.com/gb-web/assets/favicon.ico"
	}

	var odt ODTeam
	var players []Player

	teamName := m.Content[4:]
	num, err := strconv.Atoi(teamName)
	if err != nil {
		switch team.ODSite {
		case SiteBattlefy:
			tournamentID := strings.Split(team.TournamentLink.String, "/")[5]
			var team battlefy.Team
			names, err := battlefy.FindTeam(tournamentID, teamName, &team)
			if err != nil {
				if strings.HasPrefix(err.Error(), "unable to find team") {
					return fmt.Sprintf("Unable to find team \"%s\"", teamName), nil
				}
				return err.Error(), err
			}

			if len(names) > 1 {
				return formatNames(names), nil
			}

			odt = team
			players = make([]Player, len(team.Players))
			for i := range team.Players {
				players[i] = team.Players[i]
			}
		case SiteGamebattles:
			urlSplit := strings.Split(team.TournamentLink.String, "/")
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
				if strings.Contains(team.TeamName, teamName) {
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
			odt = foundTeam
			players = make([]Player, len(foundTeam.Players))
			for i := range foundTeam.Players {
				players[i] = foundTeam.Players[i]
			}
		}
	} else {
		switch team.ODSite {
		case SiteBattlefy:
			// show other team in OD round n
			if !team.TeamID.Valid {
				return "No team ID for this team.", nil
			} else if !team.TournamentLink.Valid {
				return "No tournament link for this team.", nil
			}
			t, err := battlefy.GetOtherTeam(team.TournamentLink.String, team.TeamID.String, num)
			if err != nil {
				return fmt.Sprintf("No data for round %d. :(", num), err
			}

			odt = t
			players = make([]Player, len(t.Players))
			for i := range t.Players {
				players[i] = t.Players[i]
			}
		case SiteGamebattles:
			// TODO: this
			break
		}
	}

	embed := formatTeam(siteLogoURL, odt, convertPlayers(players))
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
	return "", nil
}

// convertPlayers takes a list of generic players and gets their overbuff stats
func convertPlayers(players []Player) []goverbuff.Player {
	var wg sync.WaitGroup
	wg.Add(len(players))
	playerChan := make(chan goverbuff.Player, len(players))

	client := goverbuff.DefaultClient()
	for _, player := range players {
		go func(player Player) {
			defer wg.Done()
			stats, _ := goverbuff.GetPlayer(client, player.Battletag())
			playerChan <- stats
		}(player)
	}

	wg.Wait()
	close(playerChan)

	var overbuffPlayers []goverbuff.Player
	for range players {
		overbuffPlayers = append(overbuffPlayers, <-playerChan)
	}
	return overbuffPlayers
}

func averageSR(players []goverbuff.Player) int {
	var avg int
	var n int
	for _, p := range players {
		if p.SR != -1 {
			avg += p.SR
			n++
		}
	}
	return avg / n
}

// formatNames formats a list of team names into a code block.
func formatNames(names []string) string {
	nameStr := "```"
	for _, name := range names {
		nameStr += name + "\n"
	}
	nameStr += "```"
	if len(nameStr) > 2000 {
		// message is above Discord's limit
		return "Too many results."
	}
	return nameStr
}

// formatTeam formats a team and it's players into a fancy embed
func formatTeam(logoURL string, odt ODTeam, players []goverbuff.Player) *discordgo.MessageEmbed {
	roleEmotes := map[string]string{
		"Defense": ":crossed_swords:",
		"Offense": ":crossed_swords:",
		"Tank":    ":shield:",
		"Support": ":ambulance:",
	}

	embed := &discordgo.MessageEmbed{
		Color: 0xe74c3c,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: odt.Logo(),
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name:    odt.Name(),
			URL:     odt.Link(),
			IconURL: logoURL,
		},
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].SR > players[j].SR
	})

	var playerString string
	for _, player := range players {
		emote := roleEmotes[player.Main()]
		if emote == "" {
			emote = ":ghost:"
		}

		var sr string
		if player.SR == -1 {
			sr = "???"
		} else {
			sr = fmt.Sprintf("%d", player.SR)
		}

		// TODO: split gamebattles and battlefy into their own commands to re-implement bolding active team members
		/*
			var name string
			if player.Active() {
				name = fmt.Sprintf("**%s**", player.BTag)
			} else {
				name = player.BTag
			}
		*/

		//playerString += fmt.Sprintf("%s %s: %s\n", emote, name, sr)
		playerString += fmt.Sprintf("%s %s: %s\n", emote, player.BTag, sr)
	}

	var title string
	if len(players) > 6 {
		playerString = fmt.Sprintf("**Average SR: %d**\n", averageSR(players)) + playerString
		title = fmt.Sprintf("Top 6 Average: %d", averageSR(players[:6]))
	} else {
		title = "Players"
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: title, Value: playerString})
	return embed
}
