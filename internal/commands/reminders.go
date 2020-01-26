package commands

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/reminders"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

func init() {
	examples := [][2]string{{"!reminders", "Show the reminder config for this team."}}
	command.AddCommand("reminders", "Get reminder config for this team.", examples, Reminders)

	examples = [][2]string{{"!reminders_set activities Scrim", "Set \"Scrim\" as the only valid reminder activity for this team."}}
	command.AddCommand("reminders_set", "Update reminder config.", examples, RemindersSet)

	examples = [][2]string{{"!reminders_add activities Scrim", "Add \"Scrim\" to the activities list."}}
	command.AddCommand("reminders_add", "Add an item to a field in the reminder config.", examples, RemindersAdd)
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
		_, err = s.DB.Exec("UPDATE reminders SET activities = $1 WHERE team = $2", pq.Array(activities), t.ID)
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
		return fmt.Sprintf("Error updating %s: %s", args[1], err), err
	}

	return "Updated " + args[1] + ".", nil
}

// RemindersAdd adds an item to either activities or intervals in a team's config
func RemindersAdd(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	t := s.FindTeam(m.GuildID, m.ChannelID)
	if t.ID == 0 {
		return "No team in this channel / guild.", nil
	}

	if len(args) < 3 {
		return "usage: !reminders_add <activities|intervals> <values...>", nil
	}

	var config reminders.Config
	err := s.DB.Get(&config, "SELECT activities, intervals FROM reminders WHERE team = $1", t.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No reminder config set up.", nil
		}
		return "Error grabbing reminder config: " + err.Error(), err
	}
	switch args[1] {
	case "activities":
		sched := s.FindSchedule(m.GuildID, m.ChannelID)
		if sched == nil {
			return "", nil
		}

		activities, parseErr := parseArgs(args[2:], sched.ValidActivities)
		if parseErr != nil {
			return err.Error(), nil
		}
		var foundUnique bool
		for _, activity := range activities {
			dupe := false
			for _, addedActivity := range config.Activities {
				if activity == addedActivity {
					dupe = true
					break
				}
			}
			if !dupe {
				config.Activities = append(config.Activities, activity)
				foundUnique = true
			}
		}
		if !foundUnique {
			return "No new activities to add.", nil
		}
		_, err = s.DB.Exec("UPDATE reminders SET activities = $1 WHERE team = $2", config.Activities, t.ID)
	case "intervals":
		nums := make([]int64, 0, len(args[2:]))
		for i, num := range args[2:] {
			if num[len(num)-1] == ',' {
				num = num[:len(num)-1]
			}
			nums[i], err = strconv.ParseInt(num, 10, 8)
			if err != nil {
				return fmt.Sprintf("Invalid number %q", num), nil
			}
		}

		var foundUnique bool
		for _, num := range nums {
			dupe := false
			for _, interval := range config.Intervals {
				if num == interval {
					dupe = true
					break
				}
			}
			if !dupe {
				config.Intervals = append(config.Intervals, num)
				foundUnique = true
			}
		}
		if !foundUnique {
			return "No new intervals to add.", nil
		}
		_, err = s.DB.Exec("UPDATE reminders SET intervals = $1 WHERE team = $2", config.Intervals, t.ID)
	default:
		return fmt.Sprintf("Invalid field %q, valid options: activities, intervals", args[1]), nil
	}

	if err != nil {
		return fmt.Sprintf("Error updating %q: %s", args[1], err), err
	}

	return "Updated " + args[1] + ".", nil
}
