package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bigheadgeorge/goverbuff"
	"github.com/bwmarrin/discordgo"
)

// searchOD searches the participants in a tournament for the given name.
type searchOD func(int, string, *ODInfo) (string, error)

// matchOD gets stats for the opposing team in a given round in a tournament.
type matchOD func(int, int, *ODInfo) (string, error)

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

// ODInfo glues a ODTeam and []Player together
type ODInfo struct {
	Team    ODTeam
	Players []Player
}

// OD grabs information about another team.
func OD(m *discordgo.MessageCreate, search searchOD, match matchOD, embed *discordgo.MessageEmbed) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No config for this guild.", nil
	} else if strings.Count(m.Content, " ") == 0 { // hacky argument check
		return "No args.", nil
	}

	var msg string
	var odi ODInfo

	teamName := m.Content[4:]
	num, err := strconv.Atoi(teamName)
	if err != nil {
		msg, err = search(team.ID, teamName, &odi)
	} else {
		msg, err = match(team.ID, num, &odi)
	}
	if len(msg) > 0 || err != nil {
		return msg, err
	}

	embed = formatTeam(odi.Team, convertPlayers(odi.Players))
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
func formatTeam(odt ODTeam, players []goverbuff.Player) *discordgo.MessageEmbed {
	roleEmotes := map[string]string{
		"Defense": ":crossed_swords:",
		"Offense": ":crossed_swords:",
		"Tank":    ":shield:",
		"Support": ":ambulance:",
	}

	embed := &discordgo.MessageEmbed{
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: odt.Logo(),
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name: odt.Name(),
			URL:  odt.Link(),
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
