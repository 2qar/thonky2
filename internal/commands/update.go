package commands

import (
	"log"

	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

func init() {
	command.AddCommand("update", "Update the sheet", [][2]string{{"!update", "Update the sheet. :)"}}, Update)
}

// Update updates the sheet locally
func Update(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	sched := s.FindSchedule(m.GuildID, m.ChannelID)
	if sched == nil {
		return "", nil
	}

	updated, err := sched.Updated()
	if err != nil {
		return "Error checking if the sheet is updated. :(", err
	} else if updated {
		return "Nothing to update.", nil
	}
	msg, _ := s.Session.ChannelMessageSend(m.ChannelID, "Updating...")

	err = sched.Update()
	if err != nil {
		if err.Error() == "already updating" {
			s.Session.ChannelMessageEdit(m.ChannelID, msg.ID, "Already updating.")
		} else {
			log.Println(err)
			s.Session.ChannelMessageEdit(m.ChannelID, msg.ID, "Error updating. :(")
		}
	} else {
		s.Session.ChannelMessageEdit(m.ChannelID, msg.ID, "Finished updating. :)")
	}
	return "", nil
}
