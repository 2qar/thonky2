package main

import (
	"encoding/json"
	"fmt"
	"math"
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
	ID    json.Number
	Teams [2]struct {
		Name string
		Logo string
	} `json:"competitors"`
	Scores [2]struct {
		Value int
	}
	Status   string
	Timezone string    `json:"timeZone"`
	Start    time.Time `json:"startDate"`
	End      time.Time `json:"endDate"`
}

// date returns a Time in the format month/day
func date(t *time.Time) string {
	return t.Format("01/02")
}

// addTime returns a string that'll never say something stupid like "1 minutes"
func addTime(t int, s, post string) string {
	if t == 1 {
		return "1 " + s + post
	} else if t > 0 {
		return fmt.Sprintf("%d %ss%s", t, s, post)
	}
	return ""
}

// owlEmbed returns a template for an OWL web embed
func owlEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Color: 0xFF8C08,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://styleguide.overwatchleague.com/6.8.3/assets/toolkit/images/logo-tracer.png",
		},
	}
}

// nextMatchEmbed creates an embed out of a pending OWL match
func nextMatchEmbed(m *match) *discordgo.MessageEmbed {
	embed := owlEmbed()

	localTZ, _ := time.LoadLocation("America/Los_Angeles")
	localStart := m.Start.In(localTZ)
	embed.Author = &discordgo.MessageEmbedAuthor{
		URL:  fmt.Sprintf("https://www.overwatchleague.com/en-us/match/%s", m.ID),
		Name: fmt.Sprintf("%s vs %s, %s at %s PST", m.Teams[0].Name, m.Teams[1].Name, date(&localStart), localStart.Format(time.Kitchen)),
	}

	untilStr := "Starting in "
	until := localStart.Sub(time.Now())

	minutes := int(until.Minutes())
	if minutes >= 60 {
		hours := int(math.Floor(float64(minutes / 60)))
		minutes %= 60
		if hours >= 24 {
			days := int(math.Floor(float64(hours / 24)))
			hours %= 24
			untilStr += fmt.Sprintf("%d days, ", days)
		}
		untilStr += addTime(hours, "hour", ", ")
	}
	untilStr += addTime(minutes, "minute", "")

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: untilStr,
	}
	return embed
}

// getMatches grabs the schedule from the OWL website
func getMatches() (*schedule, error) {
	r, err := http.Get("https://api.overwatchleague.com/schedule")
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var s schedule
	err = json.NewDecoder(r.Body).Decode(&s)
	return &s, err
}

// OWL posts information about Overwatch League games
func OWL(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	if len(args) == 1 || len(args) > 2 {
		return "Weird number of args", nil
	}

	switch strings.ToLower(args[1]) {
	case "today":
		sched, err := getMatches()
		if err != nil {
			return "Error grabbing OWL schedule.", err
		}
		var matches []match
		now := time.Now()
		localTZ, _ := time.LoadLocation("America/Los_Angeles")

		for _, stage := range sched.Data.Stages {
			for _, match := range stage.Matches {
				localStart := match.Start.In(localTZ)
				if localStart.Day() == now.Day() && localStart.Month() == now.Month() {
					matches = append(matches, match)
				} else if len(matches) > 0 {
					embed := owlEmbed()
					embed.Author = &discordgo.MessageEmbedAuthor{
						URL:  "http://overwatchleague.com/en-us/schedule",
						Name: fmt.Sprintf("Overwatch League Games on %s, %s", now.Weekday(), date(&now)),
					}
					embed.Footer = &discordgo.MessageEmbedFooter{
						Text: "Times shown in PST",
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
					return "", nil
				}
			}
		}

		return "No games today.", nil
	case "next":
		sched, err := getMatches()
		if err != nil {
			return "Error grabbing OWL schedule.", err
		}
		for _, stage := range sched.Data.Stages {
			for _, match := range stage.Matches {
				if match.Status == "PENDING" {
					s.ChannelMessageSendEmbed(m.ChannelID, nextMatchEmbed(&match))
					return "", nil
				}
			}
		}

		return "No games left. :(", nil
	case "now":
		r, err := http.Get("https://api.overwatchleague.com/live-match")
		if err != nil {
			return "Error grabbing live match.", err
		}
		defer r.Body.Close()

		live := struct {
			Data struct {
				LiveMatch match
			}
		}{}
		err = json.NewDecoder(r.Body).Decode(&live)
		if err != nil {
			return "Error parsing live match info.", err
		}
		match := &live.Data.LiveMatch

		var embed *discordgo.MessageEmbed
		if match.Status == "PENDING" || match.Status == "" {
			embed = nextMatchEmbed(match)
		} else {
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Color: 0x633FA3,
				Title: fmt.Sprintf("%s %s - %s %s", timeEmotes[match.Scores[0].Value], match.Teams[0].Name, match.Teams[1].Name, timeEmotes[match.Scores[1].Value]),
				URL:   "https://www.twitch.tv/overwatchleague",
				Video: &discordgo.MessageEmbedVideo{
					URL:    "https://player.twitch.tv/?channel=overwatchleague&autoplay=true",
					Width:  620,
					Height: 378,
				},
			})
		}

		s.ChannelMessageSendEmbed(m.ChannelID, embed)
	default:
		return fmt.Sprintf("Invalid option %q.", args[1]), nil
	}
	return "", nil
}
