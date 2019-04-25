package main

import (
	"github.com/bwmarrin/discordgo"
)

func init() {
	AddCommand("help", "duh", "Examples:\n\n!help\n\tyup", help)
}

func help(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) > 2 {
		s.ChannelMessageSend(m.ChannelID, "what the heck are you doing")
	} else if len(args) == 2 {
		cmd := Commands[args[1]]
		if cmd.Name != "" {
			cmdHelp := "```\n" + "!" + cmd.Name + ": " + cmd.ShortDoc + "\n\n" + cmd.LongDoc + "\n```"
			s.ChannelMessageSend(m.ChannelID, cmdHelp)
		} else {
			s.ChannelMessageSend(m.ChannelID, "No command named \""+args[1]+"\"")
		}
	} else {
		cmdList := "```\n"
		for name, cmd := range Commands {
			cmdList += "!" + name + ":\t" + cmd.ShortDoc + "\n"
		}
		cmdList += "```"
		s.ChannelMessageSend(m.ChannelID, cmdList)
	}
}
