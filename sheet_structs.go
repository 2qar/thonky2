package main

import "gopkg.in/Iwark/spreadsheet.v2"

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
	Date string
	Days *[7]string
	Container
}

// ActivitiesOn returns the activities for a given day
func (w *Week) ActivitiesOn(day int) [6]string {
	return w.Values()[day]
}
