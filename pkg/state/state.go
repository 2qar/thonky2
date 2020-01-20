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
	Schedules map[string]*schedule.Schedule
}

// FindTeam finds a team in a channel in a guild.
func (s *State) FindTeam(guildID, channelID string) team.Team {
	var t team.Team
	err := s.DB.Get(&t, "SELECT * FROM teams WHERE server_id = $1 AND $2 = ANY(channels)", guildID, channelID)
	if err != nil && err == sql.ErrNoRows {
		return s.GuildTeam(guildID)
	}
	return t
}

// GuildTeam returns the guild's team.
func (s *State) GuildTeam(guildID string) team.Team {
	var t team.Team
	err := s.DB.Get(&t, "SELECT * FROM teams WHERE server_id = $1 AND 0 = LENGTH(team_name)", guildID)
	if err != nil {
		return team.Team{}
	}
	return t
}

// FindSchedule looks for a team, then looks for their schedule
func (s *State) FindSchedule(guildID, channelID string) *schedule.Schedule {
	team := s.FindTeam(guildID, channelID)
	if team.ID == 0 {
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
