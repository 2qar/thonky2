package main

import (
	"github.com/bwmarrin/discordgo"
)

func init() {
	AddCommand("help", "duh", [][2]string{{"!help", "yeah"}}, help)
}

func help(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) > 2 {
		s.ChannelMessageSend(m.ChannelID, "what the heck are you doing")
	} else if len(args) == 2 {
		cmd := Commands[args[1]]
		if cmd.Name != "" {
			longDoc := "Examples:\n\n"
			for _, example := range cmd.Examples {
				longDoc += example[0] + "\n\t" + example[1] + "\n"
			}
			cmdHelp := "```\n" + "!" + cmd.Name + ": " + cmd.ShortDoc + "\n\n" + longDoc + "```"
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
