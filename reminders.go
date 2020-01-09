package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
	"github.com/robfig/cron"
)

// ReminderConfig holds a team's reminder configuration.
type ReminderConfig struct {
	Team            int            `db:"team"`
	Activities      pq.StringArray `db:"activities"`
	AnnounceChannel string         `db:"announce_channel"`
	RoleMention     sql.NullString `db:"role_mention"`
	Intervals       pq.Int64Array  `db:"intervals"`
}

// reminderCheck will check if there's an activity coming up that needs pinging
type reminderCheck struct {
	Time    int
	Session *discordgo.Session
}

var scheduler *cron.Cron

// Run checks if there's an activity coming up, and sends an announcement if it's the first one of the day
func (r reminderCheck) Run() {
	for _, guild := range teams {
		for _, team := range guild {
			checkReminders(DB, team, &r)
		}
	}
}

func checkReminders(db *Handler, team *Team, r *reminderCheck) {
	reminderConfig, err := db.ReminderConfig(team.ID)
	if err != nil {
		return
	}
	today := time.Now()

	activities := team.Schedule().Week.ActivitiesOn(team.Schedule().Week.Weekday(int(today.Weekday())))
	var done bool
	for i, activity := range activities {
		if done {
			break
		} else if i != today.Hour()-15 {
			continue
		}
		for _, reminder := range reminderConfig.Activities {
			if activity == reminder {
				done = true

				announcement := fmt.Sprintf("%s in %d minutes", activity, 60-r.Time)
				if reminderConfig.RoleMention.Valid {
					announcement = fmt.Sprintf("%s %s", reminderConfig.RoleMention.Value, announcement)
				}

				r.Session.ChannelMessageSend(reminderConfig.AnnounceChannel, announcement)

				announceLog := fmt.Sprintf("send announcement for %q in [%s]", activity, team.GuildID)
				if !team.Guild() {
					announceLog += fmt.Sprintf("for %q", team.Name)
				}
				log.Println(announceLog)
				break
			}
		}
	}
}

func addReminder(time int, s *discordgo.Session) error {
	return scheduler.AddJob(fmt.Sprintf("0 %d 13-23 * * *", time), reminderCheck{Time: time, Session: s})
}

// StartReminders starts checking for reminders 45 and 15 minutes before each hour
func StartReminders(s *discordgo.Session) error {
	scheduler = cron.New()
	err := addReminder(15, s)
	if err != nil {
		return err
	}
	err = addReminder(45, s)
	if err != nil {
		return err
	}
	scheduler.Start()
	return nil
}
