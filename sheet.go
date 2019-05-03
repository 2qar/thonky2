package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	spreadsheet "gopkg.in/Iwark/spreadsheet.v2"
)

// Container stores cells in a format that fits each day of the week.
type Container struct {
	Cells *[7][6]*spreadsheet.Cell
}

// Values returns the string values of each cell
func (c *Container) Values() [7][6]string {
	var values [7][6]string
	for i, row := range c.Cells {
		for j, cell := range row {
			values[i][j] = cell.Value
		}
	}
	return values
}

// Player stores availability for each day and info about the player
type Player struct {
	Name string
	Role string
	Container
}

// Week stores the schedule for the week
type Week struct {
	Date string
	Days *[7]string
	Container
}

// ActivitiesOn returns the activities for a given day
func (w *Week) ActivitiesOn(day int) [6]string {
	return w.Values()[day]
}

// Availability returns the availability of a player for a week
func (p *Player) Availability() [7][6]string {
	return p.Values()
}

// AvailabilityOn returns the availability of a player on a day
func (p *Player) AvailabilityOn(day int) [6]string {
	return p.Availability()[day]
}

// AvailabilityAt returns the availability of a player at a given time
func (p *Player) AvailabilityAt(day, time, start int) string {
	return p.AvailabilityOn(day)[time-start]
}

func (s *Sheet) getPlayer(name string) (Player, error) {
	sheet, err := s.SheetByTitle(name)
	if err != nil {
		return Player{}, err
	}
	var availability [7][6]*spreadsheet.Cell
	for i := 2; i < 9; i++ {
		for j := 2; j < 8; j++ {
			availability[i-2][j-2] = &sheet.Rows[i][j]
		}
	}

	p := Player{Name: name}
	p.Cells = &availability
	return p, nil
}

// GetSheet returns a Sheet based on sheetID
func GetSheet(sheetID string, s *Sheet) (updated bool, err error) {
	sheet, err := Service.FetchSpreadsheet(sheetID)
	if err != nil {
		return
	}
	*s = Sheet{Spreadsheet: &sheet}
	if _, ferr := os.Open(cacheFilename("modified", sheetID)); ferr == nil {
		log.Println("loading sheet")
		var b []byte
		b, err = loadSheetAttr("modified", sheetID)
		if err != nil {
			return
		}
		var t time.Time
		err = json.Unmarshal(b, &t)
		s.LastModified = &t
		updated, err = s.Updated()
		if err != nil {
			log.Println("error grabbing s.Updated():", err)
			return
		} else if !updated {
			log.Printf("cache for [%s] outdated\n", sheetID)
			return
		}

		b, err = loadSheetAttr("players", sheetID)
		if err != nil {
			return
		}
		var players []*Player
		err = json.Unmarshal(b, &players)
		if err != nil {
			return
		}
		s.PlayerCache = players

		b, err = loadSheetAttr("week", sheetID)
		if err != nil {
			return
		}
		var week *Week
		err = json.Unmarshal(b, &week)
		if err != nil {
			return
		}
		s.WeekCache = week

		log.Println("loaded sheet")
		return
	}
	err = s.UpdateModified()
	log.Println("got sheet")
	return
}

// Sheet wraps spreadsheet.Spreadsheet with more metadata like the last modified time etc.
type Sheet struct {
	LastModified *time.Time
	PlayerCache  []*Player
	WeekCache    *Week
	*spreadsheet.Spreadsheet
}

// sheetLastModified returns the sheet's last modified time according to Google Drive
func sheetLastModified(sheetID string) (*time.Time, error) {
	call := FilesService.Get(sheetID)
	call = call.Fields("modifiedTime")
	f, err := call.Do()
	if err != nil {
		return nil, err
	}
	timeString := f.ModifiedTime[:strings.LastIndex(f.ModifiedTime, ".")]
	t, err := time.Parse("2006-01-02T15:04:05", timeString)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Updated returns whether the sheet is updated or not
func (s *Sheet) Updated() (bool, error) {
	lastModified, err := sheetLastModified(s.ID)
	if err != nil {
		return false, err
	}
	return lastModified.Before(*s.LastModified) || lastModified.Equal(*s.LastModified), nil
}

// UpdateModified syncs the sheet's modified time with Google Drive's time
func (s *Sheet) UpdateModified() error {
	lastModified, err := sheetLastModified(s.ID)
	if err != nil {
		return err
	}
	s.LastModified = lastModified
	return nil
}

// GetPlayers returns all of the players on a sheet.
func (s *Sheet) GetPlayers() ([]*Player, error) {
	updated, err := s.Updated()
	if err != nil {
		return []*Player{}, nil
	} else if updated && s.PlayerCache != nil {
		return s.PlayerCache, nil
	}

	sheet, err := s.SheetByTitle("Team Availability")
	if err != nil {
		return []*Player{}, err
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
				player, _ := s.getPlayer(name)
				player.Role = role
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
	s.PlayerCache = players
	return players, nil
}

// GetWeek returns the week schedule on a sheet.
func (s *Sheet) GetWeek() (*Week, error) {
	updated, err := s.Updated()
	if err != nil {
		return nil, nil
	} else if updated && s.WeekCache != nil {
		return s.WeekCache, nil
	}

	sheet, err := s.SheetByTitle("Weekly Schedule")
	if err != nil {
		return &Week{}, err
	}

	date := strings.Split(sheet.Rows[2][1].Value, ", ")[1]

	week := &Week{Date: date}
	var cells [7][6]*spreadsheet.Cell
	var days [7]string
	for i := 2; i < 9; i++ {
		days[i-2] = sheet.Rows[i][1].Value
		for j := 2; j < 8; j++ {
			cells[i-2][j-2] = &sheet.Rows[i][j]
		}
	}
	week.Days = &days
	week.Cells = &cells

	s.WeekCache = week
	return week, err
}

// Save writes the sheet info to disk.
func (s *Sheet) Save() (err error) {
	if _, err = os.Open("cache"); os.IsNotExist(err) {
		err = os.Mkdir("cache", 0700)
		if err != nil {
			return
		}
	} else if _, err = os.Open("cache/" + s.ID); os.IsNotExist(err) {
		err = os.Mkdir("cache/"+s.ID, 0700)
		if err != nil {
			return
		}
	}

	err = saveSheetAttr(s.LastModified, "modified", s.ID)
	if err != nil {
		return
	}

	err = saveSheetAttr(s.PlayerCache, "players", s.ID)
	if err != nil {
		return
	}
	err = saveSheetAttr(s.WeekCache, "week", s.ID)
	if err != nil {
		return
	}

	return
}

// cacheFilename returns a filename based on attr and sheetID
func cacheFilename(attr, sheetID string) string {
	return "cache/" + sheetID + "/" + attr + ".json"
}

func saveSheetAttr(c interface{}, attr, sheetID string) error {
	log.Printf("saving %s for [%s]\n", attr, sheetID)
	filename := cacheFilename(attr, sheetID)
	m, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, m, 0644)
	if err != nil {
		return err
	}
	return nil
}

func loadSheetAttr(attr, sheetID string) (b []byte, err error) {
	log.Printf("loading %s for [%s]\n", attr, sheetID)
	b, err = ioutil.ReadFile(cacheFilename(attr, sheetID))
	return
}
