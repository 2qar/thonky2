package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron"
)

// scheduler keeps the Cron scheduler used for reminders from being eaten by the garbage collecter.
// idk if this is actually necessary but whatever :)
var scheduler *cron.Cron

// reminderCheck will check if there's an activity coming up that needs pinging
type reminderCheck struct {
	Time    int
	Session *discordgo.Session
}

// Run checks if there's an activity coming up, and sends an announcement if it's the first one of the day
func (r reminderCheck) Run() {
	for _, info := range guildInfo {
		if info.DocKey.Valid && info.AnnounceChannel.Valid {
			today := time.Now()

			activities := info.Week.ActivitiesOn(info.Week.Weekday(int(today.Weekday())))
			var done bool
			for i, activity := range activities {
				if done {
					break
				}
				for _, reminder := range info.RemindActivities {
					if activity == reminder {
						done = true
						if i != today.Hour()-15 {
							break
						}

						announcement := fmt.Sprintf("%s in %d minutes", activity, 60-r.Time)
						if info.RoleMention.Valid {
							announcement = info.RoleMention.String + " " + announcement
						}

						r.Session.ChannelMessageSend(info.AnnounceChannel.String, announcement)

						announceLog := fmt.Sprintf("send announcement for %q in [%s]", activity, info.GuildID)
						if info.TeamName != "" {
							announceLog += fmt.Sprintf("for %q", info.TeamName)
						}
						log.Println(announceLog)
						break
					}
				}
			}
		}
	}
}

func addReminder(time int, s *discordgo.Session) error {
	return scheduler.AddJob(fmt.Sprintf("0 %d 15-20 * * *", time), reminderCheck{Time: time, Session: s})
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
