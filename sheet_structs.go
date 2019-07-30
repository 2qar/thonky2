package main

import (
	"strings"
	"time"

	"gopkg.in/Iwark/spreadsheet.v2"
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

// Week stores the schedule for the week
type Week struct {
	Date      string
	Days      *[7]string
	StartTime int
	Notes     *[7][6]string
	Container
}

// ActivitiesOn returns the activities for a given day
func (w *Week) ActivitiesOn(day int) [6]string {
	return w.Values()[day]
}

// DayInt returns the day of the week using the name of the day
func (w *Week) DayInt(dayName string) int {
	day := -1
	if len(dayName) >= 6 {
		dayName = strings.ToLower(dayName)
		for i := 0; i < 7; i++ {
			currName := strings.ToLower(time.Weekday(i).String())
			if dayName == currName || dayName[:3] == currName[:3] {
				day = w.Weekday(i)
				break
			}
		}
	}
	return day
}

// Weekday returns the day of the week depending in the day order on the sheet
func (w *Week) Weekday(day int) int {
	if strings.HasPrefix(w.Days[0], "Sunday") {
		return day
	}
	return Weekday(day)
}
