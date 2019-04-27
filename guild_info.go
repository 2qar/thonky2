package main

import (
	"github.com/bigheadgeorge/thonky2/db"
	spreadsheet "gopkg.in/Iwark/spreadsheet.v2"
	"log"
)

// BaseInfo acts as the crappy interface for TeamInfo and GuildInfo, but it works :)
type BaseInfo interface {
	SheetLink() string
}

// TeamInfo stores info about a team
type TeamInfo struct {
	Sheet   *spreadsheet.Spreadsheet
	Players []*Player
	Week    *Week
	*db.TeamConfig
}

// SheetLink returns the link to the team's sheet
func (t *TeamInfo) SheetLink() string {
	return "https://docs.google.com/spreadsheets/d/" + t.DocKey.String
}

// GuildInfo stores the default TeamInfo for the guild, plus info on every team in the guild
type GuildInfo struct {
	Teams []*TeamInfo
	*TeamInfo
}

// GetGuildInfo returns info about a guild, including info about each of the teams in that guild.
func GetGuildInfo(guildID string) (g *GuildInfo, err error) {
	handler, err := db.NewHandler()
	if err != nil {
		return
	}
	defer handler.Close()

	config, err := handler.GetGuild(guildID)
	if err != nil {
		return
	}

	getTeamInfo := func(config *db.TeamConfig) *TeamInfo {
		var sheet spreadsheet.Spreadsheet
		var err error
		teamInfo := &TeamInfo{TeamConfig: config}
		if config.DocKey.Valid {
			sheet, err = Service.FetchSpreadsheet(config.DocKey.String)
			if err != nil {
				log.Println(err)
				return teamInfo
			}
			teamInfo.Sheet = &sheet
		}
		teamInfo.Players, err = GetPlayers(&sheet)
		if err != nil {
			return teamInfo
		}
		log.Println("grabbed players")
		teamInfo.Week, err = GetWeek(&sheet)
		if err != nil {
			return teamInfo
		}
		log.Println("grabbed week")
		return teamInfo
	}
	g = &GuildInfo{TeamInfo: getTeamInfo(config)}
	teams, err := handler.GetTeams(guildID)
	if err != nil {
		return
	}
	for _, teamConfig := range teams {
		g.Teams = append(g.Teams, getTeamInfo(teamConfig))
	}
	return
}
