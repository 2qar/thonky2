package commands

import (
	"log"

	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

func init() {
	examples := [][2]string{
		{"!update", "Grab the spreadsheet if any new changes have been made."},
		{"!update force", "Grab the spreadsheet, even if there aren't any new changes."},
	}
	command.AddCommand("update", "Update the sheet", examples, Update)
}

// Update updates the sheet locally
func Update(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	sched := s.FindSchedule(m.GuildID, m.ChannelID)
	if sched == nil {
		return "", nil
	}

	if len(args) == 1 {
		updated, err := sched.Updated()
		if err != nil {
			return "Error checking if the sheet is updated. :(", err
		} else if updated {
			return "Nothing to update.", nil
		}
	} else if len(args) == 2 && args[1] != "force" {
		return "Unknown argument \"" + args[1] + "\"", nil
	}

	msg, _ := s.Session.ChannelMessageSend(m.ChannelID, "Updating...")

	err := sched.Update()
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
