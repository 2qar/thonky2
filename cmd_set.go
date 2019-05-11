package main

import (
	"strings"

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
