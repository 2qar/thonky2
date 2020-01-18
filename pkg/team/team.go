package team

import "github.com/lib/pq"

// Team holds the config for a team in a guild.
type Team struct {
	ID       int            `db:"id"`
	GuildID  string         `db:"server_id"`
	Name     string         `db:"team_name"`
	Channels pq.StringArray `db:"channels"`
}

// Guild returns whether this team represents an entire Discord guild or not
func (t *Team) Guild() bool {
	return len(t.Name) == 0
}
