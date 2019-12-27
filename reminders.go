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
	for _, guild := range teams {
		for _, team := range guild {
			if team.DocKey.Valid && team.AnnounceChannel.Valid {
				today := time.Now()

				activities := team.Schedule().Week.ActivitiesOn(team.Schedule().Week.Weekday(int(today.Weekday())))
				var done bool
				for i, activity := range activities {
					if done {
						break
					}
					for _, reminder := range team.RemindActivities {
						if activity == reminder {
							done = true
							if i != today.Hour()-15 {
								break
							}

							announcement := fmt.Sprintf("%s in %d minutes", activity, 60-r.Time)
							if team.RoleMention.Valid {
								announcement = team.RoleMention.String + " " + announcement
							}

							r.Session.ChannelMessageSend(team.AnnounceChannel.String, announcement)

							announceLog := fmt.Sprintf("send announcement for %q in [%s]", activity, team.GuildID)
							if team.Name.Valid {
								announceLog += fmt.Sprintf("for %q", team.Name.String)
							}
							log.Println(announceLog)
							break
						}
					}
				}
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
