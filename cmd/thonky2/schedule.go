package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/bigheadgeorge/thonky2/pkg/db"
	"github.com/bigheadgeorge/thonky2/pkg/schedule"
	botstate "github.com/bigheadgeorge/thonky2/pkg/state"
)

func fetchSchedule(s *botstate.State, spreadsheetID string, updateInterval int) (*schedule.Schedule, error) {
	schedule, err := schedule.New(s.Service, s.Client, spreadsheetID)
	if err != nil {
		return nil, err
	}

	var modified time.Time
	err = s.DB.Get(&modified, "SELECT modified FROM cache WHERE id = $1", schedule.ID)
	var update bool
	if err != nil {
		if err == sql.ErrNoRows {
			update = true
		} else {
			return nil, err
		}
	} else {
		if schedule.LastModified.After(modified) {
			update = true
		} else {
			log.Println("grab from cache")
			err = s.DB.CachedSchedule(schedule)
			if err != nil {
				return nil, err
			}
		}
	}

	if update {
		err = schedule.Update()
		if err != nil {
			return nil, err
		}

		err = s.DB.CacheSchedule(schedule)
		if err != nil {
			return nil, err
		}
	}

	go monitorSchedule(s.DB, schedule, updateInterval)
	return schedule, nil
}

func monitorSchedule(db *db.Handler, schedule *schedule.Schedule, updateInterval int) {
	for {
		time.Sleep(time.Duration(updateInterval) * time.Minute)
		updated, err := schedule.Updated()
		if err != nil {
			log.Println(err)
		} else if !updated {
			log.Printf("bg updating [%s]\n", schedule.ID)
			err = schedule.Update()
			if err != nil {
				log.Println(err)
			}
			err = db.CacheSchedule(schedule)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
