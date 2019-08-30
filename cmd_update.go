package main

import (
	"github.com/bwmarrin/discordgo"
	"log"
)

func init() {
	AddCommand("update", "Update the sheet", [][2]string{{"!update", "Update the sheet. :)"}}, Update)
}

// Update updates the sheet locally
func Update(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "No config for this guild.")
		return
	} else if !info.DocKey.Valid {
		s.ChannelMessageSend(m.ChannelID, "No doc key for this guild.")
		return
	}

	updated, err := info.Schedule.Updated()
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, "Error checking if the sheet is updated. :(")
		return
	} else if updated {
		s.ChannelMessageSend(m.ChannelID, "Nothing to update.")
		return
	}
	msg, _ := s.ChannelMessageSend(m.ChannelID, "Updating...")

	err = info.Update()
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
}
