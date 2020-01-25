package commands

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/reminders"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

func init() {
	examples := [][2]string{{"!reminders", "Show the reminder config for this team."}}
	command.AddCommand("reminders", "Get reminder config for this team.", examples, Reminders)

	examples[0] = [2]string{"!reminders_set activities Scrim", "Set \"Scrim\" as the only valid reminder activity for this team."}
	command.AddCommand("reminders_set", "Update reminder config.", examples, RemindersSet)
}

// Reminders shows the reminder config for this team.
func Reminders(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	t := s.FindTeam(m.GuildID, m.ChannelID)
	if t.ID == 0 {
		return "No team in this channel / guild.", nil
	}

	var config reminders.Config
	err := s.DB.Get(&config, "SELECT * FROM reminders WHERE team = $1", t.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No reminder config for this team.", nil
		}
		return "Error grabbing reminder config: " + err.Error(), err
	}

	var embed discordgo.MessageEmbed
	if t.Guild() {
		guild, _ := s.Session.Guild(m.GuildID)
		embed.Author = &discordgo.MessageEmbedAuthor{Name: guild.Name, IconURL: guild.IconURL()}
	} else {
		embed.Title = t.Name
	}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Activities", Value: strings.Join(config.Activities, ","), Inline: true},
		{Name: "Channel", Value: config.AnnounceChannel, Inline: true},
		{Name: "Role", Value: config.RoleMention.String, Inline: true},
	}
	intervals := discordgo.MessageEmbedField{Name: "Intervals", Inline: true}
	for i := 0; i < len(config.Intervals)-1; i++ {
		intervals.Value += fmt.Sprintf("%d,", config.Intervals[i])
	}
	intervals.Value += fmt.Sprintf("%d", config.Intervals[len(config.Intervals)-1])

	s.Session.ChannelMessageSendEmbed(m.ChannelID, &embed)
	return "", nil
}

// RemindersSet updates a field in the reminder config for a team.
func RemindersSet(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	t := s.FindTeam(m.GuildID, m.ChannelID)
	if t.ID == 0 {
		return "No team in this channel / guild.", nil
	}

	if len(args) < 3 {
		return "usage: !reminders_set <field> <value...>", nil
	}

	var err error
	switch args[1] {
	case "activities":
		sched := s.FindSchedule(m.GuildID, m.ChannelID)
		if sched == nil {
			return "No schedule for this team; can't validate activities.", nil
		}

		activities, parseErr := parseArgs(args[2:], sched.ValidActivities)
		if parseErr != nil {
			return err.Error(), nil
		}
		_, err = s.DB.Exec("UPDATE reminders SET activities = $1 WHERE team = $2", activities)
	case "channel":
		if !isChannel(args[2]) {
			return "Invalid channel mention.", nil
		}
		_, err = s.DB.Exec("UPDATE reminders SET announce_channel = $1 WHERE team = $2", args[2], t.ID)
	case "role":
		m, rerr := regexp.MatchString(`\d+`, args[2])
		if rerr != nil {
			log.Println(rerr)
		}
		if !m {
			return "Invalid role ID.", nil
		}
		_, err = s.DB.Exec("UPDATE reminders SET role_mention = $1 WHERE team = $2", "<@&"+args[2]+">", t.ID)
	default:
		return fmt.Sprintf("Invalid field %q, valid options: activities, channel, role", args[2]), nil
	}

	if err != nil {
		return fmt.Sprintf("Error updating %s: %s", args[2], err), err
	}

	return "Updated " + args[1] + ".", nil
}
