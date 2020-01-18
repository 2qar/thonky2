package state

import (
	"database/sql"
	"net/http"

	"github.com/bigheadgeorge/spreadsheet"
	"github.com/bigheadgeorge/thonky2/pkg/db"
	"github.com/bigheadgeorge/thonky2/pkg/schedule"
	"github.com/bigheadgeorge/thonky2/pkg/team"
	"github.com/bwmarrin/discordgo"
)

// State holds all of the services needed for thonky to operate glued together.
type State struct {
	Session   *discordgo.Session
	DB        *db.Handler
	Client    *http.Client
	Service   *spreadsheet.Service
	Teams     map[string][]*team.Team
	Schedules map[string]*schedule.Schedule
}

// FindTeam finds a team in a channel in a guild.
func (s *State) FindTeam(guildID, channelID string) *team.Team {
	if len(s.Teams[guildID]) == 1 {
		return s.Teams[guildID][0]
	}

	for _, team := range s.Teams[guildID] {
		for _, channel := range team.Channels {
			if channel == channelID {
				return team
			}
		}
	}
	return s.GuildTeam(guildID)
}

// GuildTeam returns the guild's team.
func (s *State) GuildTeam(guildID string) *team.Team {
	if len(s.Teams[guildID]) == 1 {
		return s.Teams[guildID][0]
	}
	for _, team := range s.Teams[guildID] {
		if team.Guild() {
			return team
		}
	}
	return nil
}

// FindSchedule looks for a team, then looks for their schedule
func (s *State) FindSchedule(guildID, channelID string) *schedule.Schedule {
	team := s.FindTeam(guildID, channelID)
	if team == nil {
		s.Session.ChannelMessageSend(channelID, "No team in this channel or server.")
		return nil
	}
	spreadsheetID, err := s.DB.SpreadsheetID(team.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.Session.ChannelMessageSend(channelID, "No spreadsheet for this team.")
		} else {
			s.Session.ChannelMessageSend(channelID, "Error grabbing spreadsheet ID: "+err.Error())
		}
		return nil
	}
	return s.Schedules[spreadsheetID]
}
