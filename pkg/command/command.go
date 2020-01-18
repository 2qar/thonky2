package command

import (
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

// Commands is a package-level map of commands, where the key is the name of the command.
// Add commands using AddCommand.
var Commands = make(map[string]*Command)

// cmd is the template for any command.
// Any command should take a session, the message, and a slice of arguments.
// The string returned will be sent in Discord.
type cmd func(*state.State, *discordgo.MessageCreate, []string) (string, error)

// Command is a struct holding info about a command and the command itself
type Command struct {
	Name     string
	ShortDoc string
	Examples [][2]string
	Aliases  []string
	Call     cmd
}

// AddAliases adds an alias for a command
func (c *Command) AddAliases(aliases ...string) {
	for _, alias := range aliases {
		c.Aliases = append(c.Aliases, alias)
	}
}

// Match checks if the given strings matches the command's name or any of its aliases.
func (c *Command) Match(s string) bool {
	for _, alias := range c.Aliases {
		if s == alias {
			return true
		}
	}
	return false
}

// AddCommand adds a command to the list
func AddCommand(name, shortDoc string, examples [][2]string, c cmd) *Command {
	command := &Command{Name: name, ShortDoc: shortDoc, Examples: examples, Call: c, Aliases: []string{name}}
	Commands[name] = command
	return command
}
