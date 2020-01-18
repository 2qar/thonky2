package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bigheadgeorge/goverbuff"
	"github.com/bigheadgeorge/thonky2/pkg/db"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

// searchOD searches the participants in a tournament for the given name.
type searchOD func(*db.Handler, int, string, *TeamStats) (string, error)

// matchOD gets stats for the opposing team in a given round in a tournament.
type matchOD func(*db.Handler, int, int, *TeamStats) (string, error)

// Player has methods for getting information about a player.
type Player interface {
	Battletag() string
	Active() bool
}

// ODTeam has methods for getting team info.
type ODTeam interface {
	Name() string
	Logo() string
	Link() string
}

// TeamStats glues a ODTeam and []Player together
type TeamStats struct {
	Team    ODTeam
	Players []Player
}

// getTeamStats gets the SR of every player on a team found with the given search and match methods.
func getTeamStats(s *state.State, m *discordgo.MessageCreate, search searchOD, match matchOD, teamStats *TeamStats) (string, error) {
	team := s.FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No config for this guild.", nil
	} else if strings.Count(m.Content, " ") == 0 { // hacky argument check
		return "No args.", nil
	}

	var msg string

	teamName := m.Content[strings.Index(m.Content, " ")+1:]
	num, err := strconv.Atoi(teamName)
	if err != nil {
		msg, err = search(s.DB, team.ID, teamName, teamStats)
	} else {
		msg, err = match(s.DB, team.ID, num, teamStats)
	}
	return msg, err
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

// formatTeamStats formats a team and it's players into a fancy embed
func formatTeamStats(odt ODTeam, players []goverbuff.Player) discordgo.MessageEmbed {
	roleEmotes := map[string]string{
		"Defense": ":crossed_swords:",
		"Offense": ":crossed_swords:",
		"Tank":    ":shield:",
		"Support": ":ambulance:",
	}

	embed := discordgo.MessageEmbed{
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
