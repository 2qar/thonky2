package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/bigheadgeorge/thonky2/schedule"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// NewHandler constructs a new Handler.
func NewHandler() (handler Handler, err error) {
	var b []byte
	b, err = ioutil.ReadFile("config.json")
	if err != nil {
		return
	}

	config := struct {
		User     string
		Pw       string
		Host     string
		Database string
	}{}
	err = json.Unmarshal(b, &config)

	connStr := fmt.Sprintf("user=%s password=%s host=%s dbname=%s", config.User, config.Pw, config.Host, config.Database)
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return
	}
	handler.DB = db
	return
}

// Handler makes grabbing and updating config easier
type Handler struct {
	*sqlx.DB
}

// AddTeam adds a team to the database
func (d *Handler) AddTeam(guildID, name, channel string) error {
	var t Team
	err := d.Get(&t, "SELECT * FROM teams WHERE server_id=$1", "0")
	if err != nil {
		return nil
	}
	t.GuildID = guildID
	t.Name = name
	t.Channels = pq.StringArray([]string{channel})
	_, err = d.Query(`INSERT INTO teams
	(server_id, team_name, channels, remind_activities, remind_intervals, update_interval)
	VALUES ($1, $2, $3, $4, $5, $6)`, t.GuildID, t.Name, t.Channels, t.RemindActivities, t.RemindIntervals, t.UpdateInterval)
	return err
}

// GetName returns the name of a team in a given channel
func (d *Handler) GetName(channelID string) (string, error) {
	var teamName string
	err := d.Get(&teamName, "SELECT team_name FROM teams WHERE $1 = ANY(channels)", channelID)
	return teamName, err
}

// GetTeams gets the config for each team in a server
func (d *Handler) GetTeams(guildID string) ([]*Team, error) {
	teams := []*Team{}
	err := d.Select(&teams, "SELECT * FROM teams WHERE server_id=$1", guildID)
	return teams, err
}

// ExecJSON runs a query with a JSON representation of v.
func (d *Handler) ExecJSON(query string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = d.Exec(query, b)
	return err
}

// CacheSchedule adds a new schedule to the cache, or updates an existing cache for the schedule.
func (d *Handler) CacheSchedule(s *schedule.Schedule) (err error) {
	r, err := d.Query("SELECT id FROM cache WHERE id = $1", s.ID)
	if err != nil {
		return
	}
	var query string
	update := r.Next()
	if update {
		query = "UPDATE cache SET modified = $1, players = $2, week = $3, activities = $4 WHERE id = $5"
	} else {
		query = "INSERT INTO cache(id, modified, players, week, activities) VALUES($1, $2, $3, $4, $5)"
	}

	var b [2][]byte
	b[0], err = json.Marshal(s.Players)
	if err != nil {
		return
	}
	b[1], err = json.Marshal(s.Week)
	if err != nil {
		return
	}
	activities := pq.StringArray(s.ValidActivities)

	if update {
		_, err = d.Exec(query, s.LastModified, b[0], b[1], activities, s.ID)
	} else {
		_, err = d.Exec(query, s.ID, s.LastModified, b[0], b[1], activities)
	}
	return
}

// CachedSchedule returns a cached schedule
func (d *Handler) CachedSchedule(s *schedule.Schedule) (err error) {
	var data [3][]byte
	r := d.QueryRow("SELECT players, week, activities FROM cache WHERE id = $1", s.ID)
	err = r.Scan(&data[0], &data[1], &data[2])
	if err != nil {
		return
	}
	for i, v := range []interface{}{&s.Players, &s.Week} {
		err = json.Unmarshal(data[i], v)
		if err != nil {
			return
		}
	}
	activities := string(data[2])
	// TODO: replace this with something a little less hacky
	for _, p := range []string{"{", "}", "\""} {
		activities = strings.ReplaceAll(activities, p, "")
	}
	s.ValidActivities = strings.Split(activities, ",")
	return
}
