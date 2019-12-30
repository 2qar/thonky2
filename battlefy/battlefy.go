package battlefy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const cloudfront = "https://dtmwra1jsgyb0.cloudfront.net/"

// FindTeam returns a list of team names found, or an error if none are found.
// If only one team is found, the Team will be written to t.
func FindTeam(tournamentID, name string, t *Team) ([]string, error) {
	url := cloudfront + "tournaments/" + tournamentID + "/" + "teams?name=" + name
	resp, err := http.Get(url)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	var teams []Team
	err = json.NewDecoder(resp.Body).Decode(&teams)
	if err != nil {
		return []string{}, err
	}
	if len(teams) == 0 {
		return []string{}, fmt.Errorf("unable to find team \"%s\"", name)
	} else if len(teams) == 1 {
		teams[0].link = "https://battlefy.com/teams/" + teams[0].PersistentTeam.ID
		markActivePlayers(&teams[0])
		*t = teams[0]
		return []string{teams[0].Name()}, nil
	} else {
		names := []string{}
		for _, team := range teams {
			names = append(names, team.PersistentTeam.Name)
		}
		return names, nil
	}
}

// GetOtherTeam get information on the enemy team in a round of Open Division
func GetOtherTeam(tournamentLink, teamID string, round int) (Team, error) {
	cutIndex := strings.LastIndex(tournamentLink, "/") + 1
	stageID := tournamentLink[cutIndex:]

	m, err := getMatch(stageID, teamID, round)
	if err != nil {
		return Team{}, err
	}

	team := m.Team()
	team.Team.link = tournamentLink + "/match/" + m.ID
	markActivePlayers(&team.Team)
	return team.Team, nil
}

type matchTeam struct {
	Team Team `json:"team"`
}

type match struct {
	ID     string    `json:"_id"`
	Top    matchTeam `json:"top"`
	Bottom matchTeam `json:"bottom"`
	isTop  bool
}

func (m *match) Team() matchTeam {
	if m.isTop {
		return m.Top
	}
	return m.Bottom
}

// Team stores a bunch of team metadata.
type Team struct {
	Players        []Player `json:"players"`
	PersistentTeam struct {
		ID                  string   `json:"_id"`
		Name                string   `json:"name"`
		Logo                string   `json:"logoUrl"`
		PersistentPlayerIDs []string `json:"persistentPlayerIDs"`
		PersistentCaptainID string   `json:"persistentCaptainID"`
	} `json:"persistentTeam"`
	link string
}

func (t Team) Name() string {
	return t.PersistentTeam.Name
}

func (t Team) Link() string {
	return t.link
}

func (t Team) Logo() string {
	return t.PersistentTeam.Logo
}

func markActivePlayers(t *Team) {
	ids := append(t.PersistentTeam.PersistentPlayerIDs, t.PersistentTeam.PersistentCaptainID)
	for p := range t.Players {
		for i, id := range ids {
			if id == t.Players[p].PID {
				ids = append(ids[:i], ids[i+1:]...)
				t.Players[p].active = true
				break
			}
		}
	}
}

// Player stores the Battlefy information about a player.
type Player struct {
	ID   string `json:"_id"`
	PID  string `json:"persistentPlayerID"`
	IGN  string `json:"inGameName"`
	User struct {
		Name  string `json:"username"`
		Accts struct {
			Bnet struct {
				Btag string `json:"battletag"`
			} `'json:"battlenet"`
		} `json:"accounts"`
	} `json:"user"`
	active bool
}

// Battletag returns a player's battletag, or an empty string if they don't have one.
func (p Player) Battletag() string {
	if p.IGN != "" {
		return p.IGN
	} else if p.User.Accts.Bnet.Btag != "" {
		return p.User.Accts.Bnet.Btag
	}
	return ""
}

// Active returns whether a player is active on their team.
func (p Player) Active() bool {
	return p.active
}

// Find a match in the given round where a team with the given id is playing
func getMatch(stageID, teamID string, round int) (match, error) {
	matchesLink := fmt.Sprintf(cloudfront+"stages/%s/rounds/%d/matches", stageID, round)

	resp, err := http.Get(matchesLink)
	if err != nil {
		return match{}, err
	}
	defer resp.Body.Close()

	var matches []match
	err = json.NewDecoder(resp.Body).Decode(&matches)
	if err != nil {
		return match{}, err
	}

	var (
		foundMatch bool
		m          match
		pos        string
	)
	for _, m = range matches {
		if m.Top.Team.PersistentTeam.ID == teamID {
			pos = "bottom"
			m.isTop = false
			foundMatch = true
			break
		} else if m.Bottom.Team.PersistentTeam.ID == teamID {
			pos = "top"
			m.isTop = true
			foundMatch = true
			break
		}
	}
	if !foundMatch {
		return match{}, errors.New("match not found")
	}

	matchLink := fmt.Sprintf(cloudfront+"matches/%s?extend[%s.team][players][users]&extend[%s.team][persistentTeam]", m.ID, pos, pos)
	resp, err = http.Get(matchLink)
	if err != nil {
		return match{}, nil
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&matches)
	if err != nil {
		return match{}, err
	}

	return matches[0], nil
}
