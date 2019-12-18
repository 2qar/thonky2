package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bigheadgeorge/goverbuff"
	"github.com/bigheadgeorge/thonky2/battlefy"
	"github.com/bwmarrin/discordgo"
)

const battlefyLogo = "http://s3.amazonaws.com/battlefy-assets/helix/images/logos/logo.png"

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
	// Active() bool
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
	} else if len(args) == 1 {
		return "No args.", nil
	}

	var odt ODTeam
	var players []Player

	teamName := m.Content[4:]
	num, err := strconv.Atoi(teamName)
	if err != nil {
		// search for a team
		if !team.TournamentLink.Valid {
			return "No tournament link for this team.", nil
		}

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
			nameList := "```"
			for _, n := range names {
				nameList += n + "\n"
			}
			nameList += "```"
			if len(nameList) < 2000 {
				return nameList, nil
			}
			return "Too many results.", nil
		}

		odt = team
		players = make([]Player, len(team.Players))
		for i := range team.Players {
			players[i] = team.Players[i]
		}
	} else {
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
	}

	embed := formatTeam(odt, convertPlayers(players))
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

// formatTeam formats a team and it's players into a fancy embed
func formatTeam(odt ODTeam, players []goverbuff.Player) *discordgo.MessageEmbed {
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
			Name: odt.Name(),
			URL:  odt.Link(),
			// TODO: take the logo as an argument (gamebattles logo, battlefy logo)
			IconURL: battlefyLogo,
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
