package schedule

import (
	"strings"
	"time"

	"github.com/bigheadgeorge/thonky2/schedule/utils"
)

// Week stores the schedule for the week.
type Week struct {
	Date      string
	Days      [7]string
	StartTime int
	Container
}

// ActivitiesOn returns the activities for a given day.
func (w *Week) ActivitiesOn(day int) []string {
	return w.Values()[day]
}

// DayInt returns the day of the week using the name of the day.
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

// Weekday returns the day of the week depending in the day order on the sheet.
func (w *Week) Weekday(day int) int {
	if strings.HasPrefix(w.Days[0], "Sunday") {
		return day
	}
	return utils.Weekday(day)
}

// Today returns today based on whether Sunday is first or not.
func (w *Week) Today() int {
	return w.Weekday(int(time.Now().Weekday()))
}
