package schedule

// Player stores availability for each day and info about a player.
type Player struct {
	Name string
	Role string
	container
}

// Availability returns the availability of a player for a week.
func (p *Player) Availability() [7][6]string {
	return p.Values()
}

// AvailabilityOn returns the availability of a player on a day.
func (p *Player) AvailabilityOn(day int) [6]string {
	return p.Availability()[day]
}

// AvailabilityAt returns the availability of a player at a given time.
func (p *Player) AvailabilityAt(day, time, start int) string {
	return p.AvailabilityOn(day)[time-start]
}
