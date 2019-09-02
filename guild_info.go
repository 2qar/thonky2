package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bigheadgeorge/thonky2/db"
	"github.com/bigheadgeorge/thonky2/schedule"
)

var (
	guildInfo    = map[string]*GuildInfo{}
	schedulePool = map[string]*schedule.Schedule{}
)

// TeamInfo stores info about a team
type TeamInfo struct {
	*schedule.Schedule
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
	teamInfo := &TeamInfo{TeamConfig: config}
	if config.DocKey.Valid {
		log.Printf("grabbing schedule for guild [%s] with name \"%s\"\n", config.GuildID, config.TeamName)
		if schedulePool[config.DocKey.String] != nil {
			log.Printf("grabbed schedule for guild [%s] with name %q from pool", config.GuildID, config.TeamName)
			teamInfo.Schedule = schedulePool[config.DocKey.String]
			return teamInfo, nil
		}
		schedule, err := schedule.New(Service, Client, config.DocKey.String)
		if err != nil {
			return teamInfo, err
		}

		var t time.Time
		err = DB.Get(&t, "SELECT modified FROM cache WHERE id = $1", schedule.ID)
		var update bool
		if err != nil {
			if err == sql.ErrNoRows {
				update = true
			} else {
				return teamInfo, err
			}
		} else {
			if schedule.LastModified.After(t) {
				update = true
			} else {
				err = DB.Get(schedule, "SELECT * FROM cache WHERE id = $1", schedule.ID)
			}
		}

		if update {
			err = schedule.Update()
			if err != nil {
				return teamInfo, err
			}

			err = DB.CacheSchedule(schedule)
			if err != nil {
				return teamInfo, err
			}
		}

		teamInfo.Schedule = schedule
		schedulePool[config.DocKey.String] = schedule
	} else {
		return teamInfo, fmt.Errorf("err: no dockey for guild [%s] with name \"%s\"", config.GuildID, config.TeamName)
	}
	go func(t *TeamInfo) {
		for {
			time.Sleep(time.Duration(t.UpdateInterval) * time.Minute)
			log.Printf("bg updating [%s]\n", t.Schedule.ID)
			updated, err := t.Schedule.Updated()
			if err != nil {
				log.Println(err)
			} else if !updated {
				err = t.Update()
				if err != nil {
					log.Println(err)
				}
				err = DB.CacheSchedule(t.Schedule)
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
	config, err := DB.GetGuild(guildID)
	if err != nil {
		return
	}

	guildTeam, err := getTeamInfo(config)
	if err != nil {
		return nil, err
	}
	g = &GuildInfo{TeamInfo: guildTeam}
	teams, err := DB.GetTeams(guildID)
	if err != nil {
		return
	}
	for _, teamConfig := range teams {
		err = g.AddTeam(teamConfig)
		if err != nil {
			log.Println(err)
		}
	}
	return g, nil
}
