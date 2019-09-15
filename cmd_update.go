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
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		return "No config for this guild.", nil
	} else if !info.DocKey.Valid {
		return "No doc key for this guild.", nil
	}

	updated, err := info.Sheet.Updated()
	if err != nil {
		return "Error checking if the sheet is updated. :(", err
	} else if updated {
		return "Nothing to update.", nil
	} else if info.Updating {
		return "Already updating.", nil
	}
	msg, _ := s.ChannelMessageSend(m.ChannelID, "Updating...")

	err = info.Update(true)
	if err != nil {
		log.Println(err)
		s.ChannelMessageEdit(m.ChannelID, msg.ID, "Error updating. :(")
	} else {
		s.ChannelMessageEdit(m.ChannelID, msg.ID, "Finished updating. :)")
	}
	return "", nil
}
