package owl

import (
	"encoding/json"
	"net/http"
	"time"
)

// Match represents an Overwatch League match.
type Match struct {
	ID    json.Number
	Teams [2]struct {
		Name string
		Logo string
	} `json:"competitors"`
	Scores [2]struct {
		Value int
	}
	Status   string
	Timezone string    `json:"timeZone"`
	Start    time.Time `json:"startDate"`
	End      time.Time `json:"endDate"`
}

type liveMatch struct {
	Data struct {
		LiveMatch Match
	} `json:"match"`
}

// Schedule shows all of the stages for OWL
type Schedule struct {
	Data struct {
		Stages []struct {
			Matches []Match
		}
	}
}

// Matches gets the schedule for the current season of OWL.
func Matches() (Schedule, error) {
	resp, err := http.Get("https://api.overwatchleague.com/schedule")
	if err != nil {
		return Schedule{}, err
	}
	defer resp.Body.Close()

	var s Schedule
	err = json.NewDecoder(resp.Body).Decode(&s)
	return s, err
}

// Live returns the Overwatch League match being streamed right now.
func Live() (Match, error) {
	resp, err := http.Get("https://api.overwatchleague.com/live-match")
	if err != nil {
		return Match{}, err
	}
	defer resp.Body.Close()

	var live liveMatch
	err = json.NewDecoder(resp.Body).Decode(&live)
	return live.Data.LiveMatch, err
}
