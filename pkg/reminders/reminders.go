package reminders

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bigheadgeorge/thonky2/pkg/team"
	"github.com/lib/pq"
	"github.com/robfig/cron"
)

var scheduler *cron.Cron

// Config holds a team's reminder configuration.
type Config struct {
	Team            int            `db:"team"`
	Activities      pq.StringArray `db:"activities"`
	AnnounceChannel string         `db:"announce_channel"`
	RoleMention     sql.NullString `db:"role_mention"`
	Intervals       pq.Int64Array  `db:"intervals"`
}

// Reminder will check if there's an activity coming up that needs pinging
type Reminder struct {
	State  *state.State
	Team   *team.Team
	Config *Config

	time int
}

// Run checks if there's an activity coming up, and sends an announcement if it's the first one of the day
func (r Reminder) Run() {
	today := time.Now()

	var activities pq.StringArray
	err := r.State.DB.QueryRow("SELECT activities FROM schedules WHERE team = $1", r.Team.ID).Scan(&activities)
	if err != nil {
		log.Printf("error grabbing reminder activities for team %d: %s\n", r.Team.ID, err)
		return
	}
	spreadsheetID, err := r.State.DB.SpreadsheetID(r.Team.ID)
	if err != nil {
		log.Printf("error grabbing spreadsheet id for team %d: %s\n", r.Team.ID, err)
		return
	}
	week := r.State.Schedules[spreadsheetID].Week

	var done bool
	for i, activity := range week.ActivitiesOn(week.Weekday(int(today.Weekday()))) {
		if done {
			break
		} else if i != today.Hour()-15 {
			continue
		}
		for _, reminder := range activities {
			if activity == reminder {
				done = true

				announcement := fmt.Sprintf("%s in %d minutes", activity, 60-r.time)
				if r.Config.RoleMention.Valid {
					announcement = fmt.Sprintf("%s %s", r.Config.RoleMention.Value, announcement)
				}

				r.State.Session.ChannelMessageSend(r.Config.AnnounceChannel, announcement)

				announceLog := fmt.Sprintf("send announcement for %q in [%s]", activity, r.Team.GuildID)
				if !r.Team.Guild() {
					announceLog += fmt.Sprintf("for %q", r.Team.Name)
				}
				log.Println(announceLog)
				break
			}
		}
	}
}

// AddReminder adds a reminder to the scheduler.
func AddReminder(r Reminder) error {
	for _, time := range r.Config.Intervals {
		r.time = int(time)
		err := scheduler.AddJob(fmt.Sprintf("0 %d 13-23 * * *", time), r)
		if err != nil {
			log.Println("error adding reminders for team %d, interval %d: %s", r.Team.ID, time, err)
			return err
		}
	}
	return nil
}

// Init initializes the reminder scheduler.
func Init() {
	scheduler = cron.New()
}

// Start starts the reminder scheduler.
func Start() {
	scheduler.Start()
}
