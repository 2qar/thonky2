package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/bigheadgeorge/thonky2/schedule"
	"github.com/lib/pq"
)

var (
	teams        = map[string][]*Team{}
	schedulePool = map[string]*schedule.Schedule{}
)

// Team holds the config for a team in a guild.
type Team struct {
	GuildID          string         `db:"server_id"`
	Name             string         `db:"team_name"`
	Channels         pq.StringArray `db:"channels"`
	AnnounceChannel  sql.NullString `db:"announce_channel"`
	DocKey           sql.NullString `db:"doc_key"`
	RemindActivities pq.StringArray `db:"remind_activities"`
	RemindIntervals  pq.Int64Array  `db:"remind_intervals"`
	RoleMention      sql.NullString `db:"role_mention"`
	TeamID           sql.NullString `db:"team_id"`
	UpdateInterval   int            `db:"update_interval"`
	StageID          sql.NullString `db:"stage_id"`
	TournamentLink   sql.NullString `db:"tournament_link"`
	ODSite           int            `db:"od_site"`
}

// Guild returns whether this team represents an entire Discord guild or not
func (t *Team) Guild() bool {
	return len(t.Name) == 0
}

// Schedule returns the schedule for this team, or nil if they haven't configured one
func (t *Team) Schedule() *schedule.Schedule {
	if t.DocKey.Valid {
		return schedulePool[t.DocKey.String]
	}
	return nil
}

// SheetLink returns the link to the team's sheet
func (t *Team) SheetLink() string {
	return "https://docs.google.com/spreadsheets/d/" + t.DocKey.String
}

func initTeam(team *Team) error {
	if !team.DocKey.Valid {
		return fmt.Errorf("err: no dockey for guild [%s] with name \"%s\"", team.GuildID, team.Name)
	}
	log.Printf("grabbing schedule for guild [%s] with name \"%s\"\n", team.GuildID, team.Name)
	if schedulePool[team.DocKey.String] != nil {
		log.Printf("grabbed schedule for guild [%s] with name %q from pool", team.GuildID, team.Name)
		return nil
	}
	schedule, err := schedule.New(Service, Client, team.DocKey.String)
	if err != nil {
		return err
	}

	var t time.Time
	err = DB.Get(&t, "SELECT modified FROM cache WHERE id = $1", schedule.ID)
	var update bool
	if err != nil {
		if err == sql.ErrNoRows {
			update = true
		} else {
			return err
		}
	} else {
		if schedule.LastModified.After(t) {
			update = true
		} else {
			log.Println("grab from cache")
			err = DB.CachedSchedule(schedule)
			if err != nil {
				return err
			}
		}
	}

	if update {
		err = schedule.Update()
		if err != nil {
			return err
		}

		err = DB.CacheSchedule(schedule)
		if err != nil {
			return err
		}
	}

	schedulePool[team.DocKey.String] = schedule

	go func(t *Team) {
		for {
			time.Sleep(time.Duration(t.UpdateInterval) * time.Minute)
			updated, err := t.Schedule().Updated()
			if err != nil {
				log.Println(err)
			} else if !updated {
				log.Printf("bg updating [%s]\n", t.DocKey.String)
				err = t.Schedule().Update()
				if err != nil {
					log.Println(err)
				}
				err = DB.CacheSchedule(t.Schedule())
				if err != nil {
					log.Println(err)
				}
			}
		}
	}(team)
	return nil
}

// FindTeam returns a team, be it the guild's team or a different team in the guild
func FindTeam(guildID, channelID string) *Team {
	if len(teams[guildID]) == 1 {
		return teams[guildID][0]
	}

	for _, team := range teams[guildID] {
		for _, channel := range team.Channels {
			if channel == channelID {
				return team
			}
		}
	}
	log.Println("actually find a guild")
	return GuildTeam(guildID)
}

// GuildTeam returns the guild's team
func GuildTeam(guildID string) *Team {
	if len(teams[guildID]) == 1 {
		return teams[guildID][0]
	}
	for _, team := range teams[guildID] {
		if team.Guild() {
			return team
		}
	}
	return nil
}
