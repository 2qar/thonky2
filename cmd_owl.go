package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	examples := [][2]string{
		{"!owl today", "Get a list of games happening today"},
	}
	AddCommand("owl", "Get info on Overwatch League games", examples, OWL)
}

type schedule struct {
	Data struct {
		Stages []struct {
			Matches []match
		}
	}
}

type match struct {
	Teams [2]struct {
		Name string
		Logo string
	} `json:"competitors"`
	Status   string
	Timezone string    `json:"timeZone"`
	Start    time.Time `json:"startDate"`
	End      time.Time `json:"endDate"`
}

// OWL posts information about Overwatch League games
func OWL(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 1 || len(args) > 2 {
		s.ChannelMessageSend(m.ChannelID, "Weird number of args")
		return
	}

	r, err := http.Get("https://api.overwatchleague.com/schedule")
	if err != nil {
		log.Println("err:", err)
		s.ChannelMessageSend(m.ChannelID, "Error grabbing OWL schedule.")
		return
	}
	defer r.Body.Close()

	var sched schedule
	err = json.NewDecoder(r.Body).Decode(&sched)
	if err != nil {
		log.Println("err:", err)
		s.ChannelMessageSend(m.ChannelID, "Error grabbing OWL schedule.")
		return
	}

	switch strings.ToLower(args[1]) {
	case "today":
		var matches []match
		now := time.Now()
		localTZ, _ := time.LoadLocation("America/Los_Angeles")

		for _, stage := range sched.Data.Stages {
			for _, match := range stage.Matches {
				localStart := match.Start.In(localTZ)
				if localStart.Day() == now.Day() && localStart.Month() == now.Month() {
					matches = append(matches, match)
				} else if len(matches) > 0 {
					embed := &discordgo.MessageEmbed{
						Color: 0xFF8C08,
						Author: &discordgo.MessageEmbedAuthor{
							URL:  "http://overwatchleague.com/en-us/schedule",
							Name: fmt.Sprintf("Overwatch League Games on %s, %s", now.Weekday(), now.Format("01/02")),
						},
						Footer: &discordgo.MessageEmbedFooter{
							Text: "Times shown in PST",
						},
						Thumbnail: &discordgo.MessageEmbedThumbnail{
							URL: "https://styleguide.overwatchleague.com/6.8.3/assets/toolkit/images/logo-tracer.png",
						},
					}

					var foundCurrent bool
					for _, match = range matches {
						localStart, localEnd := match.Start.In(localTZ), match.End.In(localTZ)
						times := fmt.Sprintf("%s - %s", localStart.Format(time.Kitchen), localEnd.Format(time.Kitchen))
						if now.Before(localEnd) && !foundCurrent {
							times = "**" + times + "**"
							foundCurrent = true
						}

						embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
							Name:  times,
							Value: fmt.Sprintf("%s vs %s", match.Teams[0].Name, match.Teams[1].Name),
						})
					}

					s.ChannelMessageSendEmbed(m.ChannelID, embed)
					return
				}
			}
		}
	}
}
