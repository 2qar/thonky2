package commands

import (
	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

func init() {
	command.AddCommand("help", "duh", [][2]string{{"!help", "yeah"}}, help)
}

func help(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	if len(args) > 2 {
		s.Session.ChannelMessageSend(m.ChannelID, "what the heck are you doing")
	} else if len(args) == 2 {
		cmd := command.Commands[args[1]]
		if cmd.Name != "" {
			longDoc := "Examples:\n\n"
			for _, example := range cmd.Examples {
				longDoc += example[0] + "\n\t" + example[1] + "\n"
			}
			cmdHelp := "```\n" + "!" + cmd.Name + ": " + cmd.ShortDoc + "\n\n" + longDoc + "```"
			s.Session.ChannelMessageSend(m.ChannelID, cmdHelp)
		} else {
			s.Session.ChannelMessageSend(m.ChannelID, "No command named \""+args[1]+"\"")
		}
	} else {
		cmdList := "```\n"
		for name, cmd := range command.Commands {
			cmdList += "!" + name + ":\t" + cmd.ShortDoc + "\n"
		}
		cmdList += "```"
		s.Session.ChannelMessageSend(m.ChannelID, cmdList)
	}
	return "", nil
}
