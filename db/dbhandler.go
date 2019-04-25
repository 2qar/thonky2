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

// GetTeam gets config for a team in a server
func (d *Handler) GetTeam(guildID, name string) (*TeamConfig, error) {
	team := &TeamConfig{}
	err := d.Get(team, "SELECT * FROM teams WHERE server_id=$1 AND team_name=$2", guildID, name)
	return team, err
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
