package main

import discord "github.com/bwmarrin/discordgo"

// Commands is a package-level map of commands, where the key is the name of the command.
// Add commands using AddCommand.
var Commands = make(map[string]*Command)

// cmd is the template for any command.
// Any command should take a session, the message, and a slice of arguments.
type cmd func(*discord.Session, *discord.MessageCreate, []string)

// Command is a struct holding info about a command and the command itself
type Command struct {
	Name string
	Doc  string
	Call cmd
}

// AddAliases adds an alias for a command
func (c *Command) AddAliases(aliases []string) {
	for _, alias := range aliases {
		Commands[alias] = c
	}
}

// AddCommand adds a command to the list
func AddCommand(name string, doc string, c cmd) *Command {
	command := &Command{name, doc, c}
	Commands[name] = command
	return command
}
