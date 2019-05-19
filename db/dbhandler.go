package db

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
)

// NewHandler constructs a new Handler.
func NewHandler() (handler Handler, err error) {
	var b []byte
	b, err = ioutil.ReadFile("config.json")
	if err != nil {
		return
	}

	config := struct {
		User string
		Pw   string
		Host string
	}{}
	err = json.Unmarshal(b, &config)

	connStr := fmt.Sprintf("user=%s password=%s host=%s dbname=thonkydb", config.User, config.Pw, config.Host)
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

// GetTeamName returns the name of a team in a given channel
func (d *Handler) GetTeamName(channelID string) (string, error) {
	var teamName string
	err := d.Get(&teamName, "SELECT team_name FROM teams WHERE $1 = ANY(channels)", channelID)
	return teamName, err
}

// GetTeams gets the config for each team in a server
func (d *Handler) GetTeams(guildID string) ([]*TeamConfig, error) {
	teams := []*TeamConfig{}
	err := d.Select(&teams, "SELECT * FROM teams WHERE server_id=$1", guildID)
	return teams, err
}

// GetGuild gets the config for a guild
func (d *Handler) GetGuild(guildID string) (*TeamConfig, error) {
	guild := &TeamConfig{}
	err := d.Get(guild, "SELECT * FROM server_config WHERE server_id=$1", guildID)
	return guild, err
}
