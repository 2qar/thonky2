package main

import (
	"github.com/bwmarrin/discordgo"
	"log"
)

func init() {
	AddCommand("update", "Update the sheet", [][2]string{{"!update", "Update the sheet. :)"}}, Update)
}

// Update updates the sheet locally
func Update(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No config for this guild.", nil
	} else if !team.DocKey.Valid {
		return "No doc key for this guild.", nil
	}
	sched := team.Schedule()

	updated, err := sched.Updated()
	if err != nil {
		return "Error checking if the sheet is updated. :(", err
	} else if updated {
		return "Nothing to update.", nil
	}
	msg, _ := s.ChannelMessageSend(m.ChannelID, "Updating...")

	err = sched.Update()
	if err != nil {
		if err.Error() == "already updating" {
			s.ChannelMessageEdit(m.ChannelID, msg.ID, "Already updating.")
		} else {
			log.Println(err)
			s.ChannelMessageEdit(m.ChannelID, msg.ID, "Error updating. :(")
		}
	} else {
		s.ChannelMessageEdit(m.ChannelID, msg.ID, "Finished updating. :)")
	}
	return "", nil
}
