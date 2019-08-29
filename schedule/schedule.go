package schedule

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bigheadgeorge/spreadsheet"
)

// Schedule wraps spreadsheet.Spreadsheet with more metadata like the last modified time etc.
// Schedules should be created with New(), or populated with schedule.Update().
type Schedule struct {
	Week            Week
	ValidActivities []string
	Players         []Player
	LastModified    time.Time
	updating        bool
	updatedModified time.Time
	client          *http.Client
	service         *spreadsheet.Service
	*spreadsheet.Spreadsheet
}

// New returns a new Schedule with all of its fields populated.
func New(service *spreadsheet.Service, client *http.Client, sheetID string) (*Schedule, error) {
	spreadsheet, err := service.FetchSpreadsheet(sheetID)
	if err != nil {
		return nil, err
	}
	s := &Schedule{Spreadsheet: &spreadsheet, client: client, service: service}
	err = s.Update()
	return s, err
}

// Update repopulates the fields of a Schedule with updated values.
func (s *Schedule) Update() error {
	if s.updating {
		return fmt.Errorf("already updating schedule")
	}

	s.updating = true
	defer func() {
		s.updating = false
	}()

	err := s.getPlayers()
	if err != nil {
		return err
	}
	err = s.getWeek()
	if err != nil {
		return err
	}
	if s.LastModified.Before(s.updatedModified) {
		s.LastModified = s.updatedModified
	} else {
		s.LastModified, err = lastModified(s.client, s.ID)
		if err != nil {
			return err
		}
	}
	s.ValidActivities, err = validActivities(s.client, s.ID)
	if err != nil {
		return err
	}

	return s.service.ReloadSpreadsheet(s.Spreadsheet)
}

// Updated returns whether the sheet is updated or not
func (s *Schedule) Updated() (bool, error) {
	var err error
	s.updatedModified, err = lastModified(s.client, s.ID)
	if err != nil {
		return false, err
	}
	return s.updatedModified.Before(s.LastModified) || s.updatedModified.Equal(s.LastModified), nil
}

// getPlayers returns all of the players on a sheet.
func (s *Schedule) getPlayers() error {
	sheet, err := s.SheetByTitle("Team Availability")
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(12)
	pCh := make(chan Player, 12)
	var currentRole string
	var playerCount int
	for i := 3; i < 15; i++ {
		role := sheet.Rows[i][1].Value
		if role != "" && currentRole != role {
			currentRole = role
		}

		name := sheet.Rows[i][2].Value
		if name != "" {
			playerCount++
			go func(name, role string) {
				defer wg.Done()
				// TODO: handle errors, probably
				sheet, _ := s.SheetByTitle(name)
				player := Player{
					Name: name,
					Role: role,
				}
				player.Fill(sheet, 2, 2)
				pCh <- player
			}(name, currentRole)
			continue
		}
		wg.Done()
	}

	wg.Wait()
	close(pCh)

	var players []*Player
	for i := 0; i < playerCount; i++ {
		p := <-pCh
		players = append(players, &p)
	}
	return nil
}

// getWeek parses the week schedule.
func (s *Schedule) getWeek() error {
	sheet, err := s.SheetByTitle("Weekly Schedule")
	if err != nil {
		return err
	}

	date := strings.Split(sheet.Rows[2][1].Value, ", ")[1]

	week := &Week{Date: date}
	week.Fill(sheet, 2, 2)
	for i := 2; i < 9; i++ {
		week.Days[i-2] = sheet.Rows[i][1].Value
	}

	startStr := strings.Split(sheet.Rows[1][2].Value, "-")[0]
	startTime, err := strconv.Atoi(startStr)
	if err != nil {
		return err
	}
	week.StartTime = startTime

	return err
}
