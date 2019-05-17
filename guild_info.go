package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bigheadgeorge/thonky2/db"
)

var guildInfo = map[string]*GuildInfo{}

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

// AddTeam adds a team to a guild's list of teams
func (g *GuildInfo) AddTeam(config *db.TeamConfig) error {
	team, err := getTeamInfo(config)
	if err != nil {
		return err
	}
	g.Teams = append(g.Teams, team)
	return nil
}

func getTeamInfo(config *db.TeamConfig) (*TeamInfo, error) {
	var updated bool
	teamInfo := &TeamInfo{TeamConfig: config}
	if config.DocKey.Valid {
		log.Printf("grabbing sheet for guild [%s] with name \"%s\"\n", config.GuildID, config.TeamName)
		var sheet Sheet
		var err error
		updated, err = GetSheet(config.DocKey.String, &sheet)
		if err != nil {
			return teamInfo, fmt.Errorf("error grabbing sheet for [%s]: %s", config.GuildID, err)
		}
		teamInfo.Sheet = &sheet
	} else {
		return teamInfo, fmt.Errorf("err: no dockey for guild [%s] with name \"%s\"", config.GuildID, config.TeamName)
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
	return teamInfo, nil
}

// GetInfo returns TeamInfo or GuildInfo, depending on what it finds with the given channelID and guildID
func GetInfo(guildID, channelID string) (*TeamInfo, error) {
	info := guildInfo[guildID]
	if info != nil {
		for _, team := range info.Teams {
			for _, id := range team.Channels {
				if strconv.FormatInt(id, 10) == channelID {
					log.Printf("grabbed info for team %q in [%s]\n", team.TeamName, team.GuildID)
					return team, nil
				}
			}
		}
		log.Printf("grabbed info for guild [%s]\n", info.GuildID)
		return info.TeamInfo, nil
	}
	return &TeamInfo{}, fmt.Errorf("no info for guild [%s]", guildID)
}

// NewGuildInfo returns info about a guild, including info about each of the teams in that guild.
func NewGuildInfo(guildID string) (g *GuildInfo, err error) {
	handler, err := db.NewHandler()
	if err != nil {
		return
	}
	defer handler.Close()

	config, err := handler.GetGuild(guildID)
	if err != nil {
		return
	}

	guildTeam, err := getTeamInfo(config)
	if err != nil {
		return nil, err
	}
	g = &GuildInfo{TeamInfo: guildTeam}
	teams, err := handler.GetTeams(guildID)
	if err != nil {
		return
	}
	for _, teamConfig := range teams {
		err = g.AddTeam(teamConfig)
		if err != nil {
			log.Println(err)
		}
	}
	return
}
