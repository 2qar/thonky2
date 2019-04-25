package db

import (
	"database/sql"
	"github.com/lib/pq"
)

// TeamConfig holds the config for a team in a guild.
type TeamConfig struct {
	GuildID          string         `db:"server_id"`
	TeamName         string         `db:"team_name"`
	Channels         pq.Int64Array  `db:"channels"`
	AnnounceChannel  sql.NullString `db:"announce_channel"`
	DocKey           sql.NullString `db:"doc_key"`
	RemindActivities pq.StringArray `db:"remind_activities"`
	RemindIntervals  pq.Int64Array  `db:"remind_intervals"`
	RoleMention      sql.NullString `db:"role_mention"`
	TeamID           sql.NullString `db:"team_id"`
	UpdateInterval   int            `db:"update_interval"`
	StageID          sql.NullString `db:"stage_id"`
	TournamentLink   sql.NullString `db:"tournament_link"`
}
