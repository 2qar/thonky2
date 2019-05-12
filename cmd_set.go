package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	examples := [][2]string{
		{"!set <player name> <day name> <time range> <availability>", "Update player availability."},
		{"!set <day name> <time range> <activity / activities>", "Update schedule."},
		{"To give multiple responses / activities, use commas:", "!set tydra monday 4-6 no, yes"},
		{"Give one response over a range to set it all to that one response:", "!set monday 4-10 free"},
	}
	AddCommand("set", "Update information on the configured spreadsheet.", examples, Set)
}

// Set is used for updating info on a Spreadsheet
func Set(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "No config for this guild.")
		return
	} else if !info.DocKey.Valid {
		s.ChannelMessageSend(m.ChannelID, "No doc key for this guild.")
		return
	}

	if len(args) >= 3 {
		day := dayInt(args[1])
		if day != -1 {
			// update w/ day
			log.Printf("update day %q w/ index %d\n", args[1], day)
			args, err := parseArgs(args[2:], info.Sheet.ValidActivities)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(args)
			return
		}

		var player *Player
		playerName := strings.ToLower(args[1])
		for _, p := range info.Players {
			if playerName == strings.ToLower(p.Name) {
				player = p
			}
		}

		if player != nil {
			day = dayInt(args[2])
			if day != -1 {
				// update w/ player
				log.Printf("update player %q\n", player.Name)
				args, err := parseArgs(args[3:], []string{"Yes", "Maybe", "No"})
				if err != nil {
					log.Println(err)
					return
				}
				log.Println(args)
				return
			}

			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid day %q", args[2]))
			return
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid day / player %q", args[1]))
		return
	}

	s.ChannelMessageSend(m.ChannelID, "weird amount of args")
}

// dayInt gets a weekday int from a day name.
func dayInt(dayName string) int {
	day := -1
	dayName = strings.ToLower(dayName)
	for i := 0; i < 7; i++ {
		currName := strings.ToLower(time.Weekday(i).String())
		if dayName == currName || dayName[:3] == currName[:3] {
			day = Weekday(i)
			break
		}
	}
	return day
}

// parseArgs takes a list of unformatted arguments and tries to match them with a given list of valid arguments.
func parseArgs(args []string, validArgs []string) ([]string, error) {
	var argString string
	if len(args) > 1 {
		argString = strings.Join(args, " ")
	} else {
		argString = args[0]
	}
	csv := strings.Split(argString, ", ")

	var parsed []string
	for _, activity := range csv {
		found := false
		for _, valid := range validArgs {
			if strings.ToLower(activity) == strings.ToLower(valid) {
				found = true
				parsed = append(parsed, valid)
				break
			}
		}
		if !found {
			return []string{}, fmt.Errorf("Invalid activity: %q", activity)
		}
	}

	return parsed, nil
}
