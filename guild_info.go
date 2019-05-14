package main

import (
	"github.com/bigheadgeorge/thonky2/db"
	"log"
	"time"
)


// TeamInfo stores info about a team
type TeamInfo struct {
	Sheet    *Sheet
	Players  []*Player
	Week     *Week
	Updating bool
	*db.TeamConfig
}

// SheetLink returns the link to the team's sheet
func (t *TeamInfo) SheetLink() string {
	return "https://docs.google.com/spreadsheets/d/" + t.DocKey.String
}

// cacheSheetInfo grabs the Players and Week from it's sheet
func (t *TeamInfo) cacheSheetInfo(save bool) (err error) {
	t.Players, err = t.Sheet.GetPlayers()
	if err != nil {
		return
	}
	log.Println("grabbed players")
	t.Week, err = t.Sheet.GetWeek()
	if err != nil {
		return
	}
	log.Println("grabbed week")
	err = t.Sheet.UpdateModified()
	if err != nil {
		return
	}
	if save {
		log.Printf("caching info for [%s]\n", t.Sheet.ID)
		err = t.Sheet.Save()
	}
	return err
}

// Update reloads the sheet, and parses the new Players and Week
func (t *TeamInfo) Update(save bool) error {
	t.Updating = true
	err := Service.ReloadSpreadsheet(t.Sheet.Spreadsheet)
	if err != nil {
		return err
	}
	err = t.cacheSheetInfo(save)
	t.Updating = false
	return err
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
		var updated bool
		teamInfo := &TeamInfo{TeamConfig: config}
		if config.DocKey.Valid {
			log.Printf("grabbing sheet for guild [%s] with name \"%s\"\n", config.GuildID, config.TeamName)
			var sheet Sheet
			var err error
			updated, err = GetSheet(config.DocKey.String, &sheet)
			if err != nil {
				log.Println("error grabbing sheet for", config.GuildID, err)
				return teamInfo
			}
			teamInfo.Sheet = &sheet
		} else {
			log.Printf("err: no dockey for guild [%s] with name \"%s\"\n", config.GuildID, config.TeamName)
			return teamInfo
		}
		err := teamInfo.cacheSheetInfo(!updated)
		if err != nil {
			log.Println(err)
		}
		go func(t *TeamInfo) {
			for {
				time.Sleep(time.Duration(t.UpdateInterval) * time.Minute)
				log.Printf("bg updating [%s]\n", t.Sheet.ID)
				updated, err := t.Sheet.Updated()
				if err != nil {
					log.Println(err)
				} else {
					err = t.Update(!updated)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}(teamInfo)
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
