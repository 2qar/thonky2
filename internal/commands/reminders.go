package commands

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/reminders"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

func init() {
	command.AddCommand("reminders", "Get reminder config for this team.", [][2]string{{"!reminders", "Show the reminder config for this team."}}, Reminders)
}

// Reminders shows the reminder config for this team
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
