package utils

// Weekday returns an ISO-8601 compliant weekday, where Monday is the beginning of the week
func Weekday(day int) int {
	if day == 0 {
		return 6
	}
	return day - 1
}
