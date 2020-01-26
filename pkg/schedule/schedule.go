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

// DriveScope is the scope that HTTP clients passed into New() should be authenticated with.
const DriveScope = "https://www.googleapis.com/auth/drive.metadata.readonly"

// Schedule wraps spreadsheet.Spreadsheet with more metadata like the last modified time etc.
// Schedules should be created with New() and populated with schedule.Update().
type Schedule struct {
	Week            Week
	ValidActivities []string
	Players         []Player
	LastModified    time.Time
	updating        bool
	client          *http.Client
	service         *spreadsheet.Service
	*spreadsheet.Spreadsheet
}

// New returns a new Schedule with its last modified time populated.
func New(service *spreadsheet.Service, client *http.Client, sheetID string) (*Schedule, error) {
	spreadsheet, err := service.FetchSpreadsheet(sheetID)
	if err != nil {
		return nil, fmt.Errorf("error getting spreadsheet: %s", err)
	}
	s := &Schedule{Spreadsheet: &spreadsheet, client: client, service: service}
	s.LastModified, err = lastModified(client, spreadsheet.ID)
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
		return fmt.Errorf("error getting players: %s", err)
	}

	s.ValidActivities, err = validActivities(s.client, s.ID)
	if err != nil {
		return fmt.Errorf("error getting valid activities: %s", err)
	}

	err = s.getWeek("Weekly Schedule")
	if err != nil {
		return fmt.Errorf("error getting week: %s", err)
	}

	s.LastModified, err = lastModified(s.client, s.ID)
	if err != nil {
		return fmt.Errorf("error getting last modified time: %s", err)
	}

	return s.service.ReloadSpreadsheet(s.Spreadsheet)
}

// Updated returns whether the sheet is updated or not
func (s *Schedule) Updated() (bool, error) {
	modified, err := lastModified(s.client, s.ID)
	if err != nil {
		return false, err
	}
	return modified.Before(s.LastModified) || modified.Equal(s.LastModified), nil
}

// SyncSheet pushes all of the changes on a sheet and updates the modified time.
func (s *Schedule) SyncSheet(sheet *spreadsheet.Sheet) (err error) {
	err = s.service.SyncSheet(sheet)
	if err != nil {
		return
	}
	s.LastModified = time.Now().UTC()
	return
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
				player.Fill(sheet, 2, 7, 2, 6)
				pCh <- player
			}(name, currentRole)
			continue
		}
		wg.Done()
	}

	wg.Wait()
	close(pCh)

	s.Players = nil
	for i := 0; i < playerCount; i++ {
		s.Players = append(s.Players, <-pCh)
	}
	return nil
}

// getWeek parses the week schedule.
func (s *Schedule) getWeek(sheetName string) error {
	sheet, err := s.SheetByTitle(sheetName)
	if err != nil {
		return err
	}

	s.Week.Date = strings.Split(sheet.Rows[2][1].Value, ", ")[1]

	var blocks int
	for {
		valid := false
		for _, activity := range s.ValidActivities {
			if sheet.Rows[2][blocks+2].Value == activity {
				valid = true
				break
			}
		}
		if !valid {
			break
		}
		blocks++
	}
	s.Week.Fill(sheet, 2, 7, 2, blocks)
	for i := 2; i < 9; i++ {
		s.Week.Days[i-2] = sheet.Rows[i][1].Value
	}

	blockRange := strings.Split(sheet.Rows[1][2].Value, "-")
	s.Week.StartTime, err = strconv.Atoi(blockRange[0])
	if err != nil {
		return err
	}

	blockEnd, err := strconv.Atoi(blockRange[1])
	if err != nil {
		return err
	}
	s.Week.BlockLength = blockEnd - s.Week.StartTime

	s.Week.Timezone = sheet.Rows[1][8].Value

	return err
}
