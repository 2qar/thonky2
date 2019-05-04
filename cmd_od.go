package main

import (
	"fmt"
	"github.com/bigheadgeorge/odscraper"
	"github.com/bwmarrin/discordgo"
	"log"
	"strconv"
	"strings"
)

const (
	tournamentID   = "5c7ccfe88d004d0345bbd0cd"
	teamID         = "5bfe1b9418ddd9114f14efb0"
	tournamentLink = "https://battlefy.com/overwatch-open-division-north-america/2019-overwatch-open-division-season-2-north-america/5c7ccfe88d004d0345bbd0cd/stage/5c929d720bc67d0345180aa6"
	battlefyLogo   = "http://s3.amazonaws.com/battlefy-assets/helix/images/logos/logo.png"
)

func init() {
	examples := [][2]string{
		{"!od 1", "Get info on the other team in round 1."},
		{"!od cloud9", "Get info on cloud9, if they're in our tournament."},
	}
	AddCommand("od", "Get information about another team or a round of Open Division", examples, OD)
}

// OD grabs information about another team
func OD(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(m.Content) < 5 {
		s.ChannelMessageSend(m.ChannelID, "No team!")
		return
	}
	name := m.Content[4:]
	num, err := strconv.Atoi(name)
	if err != nil {
		var info odscraper.TeamInfo
		names, err := odscraper.FindTeam(tournamentID, name, &info)
		if err != nil {
			if strings.HasPrefix(err.Error(), "unable to find team") {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Unable to find team \"%s\"", name))
			} else {
				log.Println(err)
				return
			}
		}
		if len(names) > 1 {
			nameList := "```"
			for _, n := range names {
				nameList += n + "\n"
			}
			nameList += "```"
			if len(nameList) < 2000 {
				s.ChannelMessageSend(m.ChannelID, nameList)
			} else {
				s.ChannelMessageSend(m.ChannelID, "Too many results.")
			}
		} else {
			embed := formatTeamInfo(&info)
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		}
	} else {
		t, err := odscraper.GetOtherTeam(tournamentLink, teamID, num)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("No data for round %d. :(", num))
			return
		}
		embed := formatTeamInfo(&t)
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
	}
}

func sortPlayers(a []odscraper.PlayerInfo, n int) {
	if n > 0 {
		sortPlayers(a, n-1)
		x := a[n]
		j := n - 1
		for j >= 0 && a[j].Stats.SR > x.Stats.SR {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = x
	}
}

// formatTeamInfo formats team info into a readable Discord embed
func formatTeamInfo(t *odscraper.TeamInfo) *discordgo.MessageEmbed {
	roleEmotes := map[string]string{
		"Defense": ":crossed_swords:",
		"Offense": ":crossed_swords:",
		"Tank":    ":shield:",
		"Support": ":ambulance:",
	}

	embed := &discordgo.MessageEmbed{
		Color: 0xe74c3c,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: t.Logo,
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name:    t.Name,
			URL:     t.Link,
			IconURL: battlefyLogo,
		},
	}

	sortPlayers(t.Players, len(t.Players)-1)
	for i, j := 0, len(t.Players)-1; i < j; i, j = i+1, j-1 {
		t.Players[i], t.Players[j] = t.Players[j], t.Players[i]
	}
	var players string
	for _, player := range t.Players {
		emote := roleEmotes[player.Stats.GetMain()]
		if emote == "" {
			emote = ":ghost:"
		}

		var sr string
		if player.Stats.SR == -1 {
			sr = "???"
		} else {
			sr = fmt.Sprintf("%d", player.Stats.SR)
		}

		var name string
		if player.Active {
			name = fmt.Sprintf("**%s**", player.Name)
		} else {
			name = player.Name
		}

		players += fmt.Sprintf("%s %s: %s\n", emote, name, sr)
	}
	getAverage := func(players []odscraper.PlayerInfo) int {
		var avg int
		var n int
		for _, p := range players {
			if p.Stats.SR != -1 {
				avg += p.Stats.SR
				n++
			}
		}
		return avg / n
	}

	var title string
	if len(t.Players) > 6 {
		players = fmt.Sprintf("**Average SR: %d**\n", getAverage(t.Players)) + players
		title = fmt.Sprintf("Top 6 Average: %d", getAverage(t.Players[:6]))
	} else {
		title = "Players"
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: title, Value: players})
	return embed
}
