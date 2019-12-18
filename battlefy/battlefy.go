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
// If only one team is found, the TeamInfo will be written to t.
func FindTeam(tournamentID, name string, t *TeamInfo) ([]string, error) {
	url := cloudfront + "tournaments/" + tournamentID + "/" + "teams?name=" + name
	resp, err := http.Get(url)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	var teams []teamData
	err = json.NewDecoder(resp.Body).Decode(&teams)
	if err != nil {
		return []string{}, err
	}
	if len(teams) == 0 {
		return []string{}, fmt.Errorf("unable to find team \"%s\"", name)
	} else if len(teams) == 1 {
		info, err := getTeamInfo(teams[0])
		if err != nil {
			return []string{}, err
		}
		*t = info
		return []string{info.Name}, nil
	} else {
		names := []string{}
		for _, team := range teams {
			names = append(names, team.Name)
		}
		return names, nil
	}
}

// GetOtherTeam get information on the enemy team in a round of Open Division
func GetOtherTeam(tournamentLink, teamID string, round int) (e TeamInfo, err error) {
	cutIndex := strings.LastIndex(tournamentLink, "/") + 1
	stageID := tournamentLink[cutIndex:]

	m, err := getMatch(stageID, teamID, round)
	if err != nil {
		return
	}

	e, err = getTeamInfo(m.Team().Info)
	if err != nil {
		return
	}

	e.Link = tournamentLink + "/match/" + m.ID
	return
}

func getTeamInfo(t teamData) (TeamInfo, error) {
	resp, err := http.Get(cloudfront + "persistent-teams/" + t.PID)
	if err != nil {
		return TeamInfo{}, err
	}
	defer resp.Body.Close()

	var pts [1]persistentTeam
	err = json.NewDecoder(resp.Body).Decode(&pts)
	if err != nil {
		return TeamInfo{}, err
	}
	pt := pts[0]

	ids := t.ActiveIDS
	for _, p := range t.Players {
		for i, id := range ids {
			if id == p.ID {
				ids = append(ids[:i], ids[i+1:]...)
				p.active = true
			}
		}
	}

	return TeamInfo{
		Link:    "https://www.battlefy.com/teams/" + t.PID,
		Name:    pt.Name,
		Logo:    pt.Logo,
		Players: append([]Player{pt.Captain}, t.Players[:]...)}, nil
}

type match struct {
	ID     string `json:"_id"`
	Top    team   `json:"top"`
	Bottom team   `json:"bottom"`
	isTop  bool
}

func (m *match) Team() team {
	if m.isTop {
		return m.Top
	}
	return m.Bottom
}

type team struct {
	ID             string         `json:"teamID"`
	Info           teamData       `json:"team"`
	PersistentTeam persistentTeam `json:"persistentTeam"`
}

type teamData struct {
	Name      string   `json:"name"`
	ActiveIDS []string `json:"playerIDs"`
	PID       string   `json:"persistentTeamID"`
	Players   []Player `json:"players"`
}

type persistentTeam struct {
	Name    string   `json:"name"`
	Logo    string   `json:"logoUrl"`
	Captain Player   `json:"persistentCaptain"`
	Players []Player `json:"persistentPlayers"`
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

// TeamInfo stores info about a team scraped from Battlefy, including stats about their players.
type TeamInfo struct {
	Link    string
	Name    string
	Logo    string
	Players []Player
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
		if m.Top.Info.PID == teamID {
			pos = "bottom"
			m.isTop = false
			foundMatch = true
			break
		} else if m.Bottom.Info.PID == teamID {
			pos = "top"
			m.isTop = true
			foundMatch = true
			break
		}
	}
	if !foundMatch {
		return match{}, errors.New("match not found")
	}

	matchLink := fmt.Sprintf(cloudfront+"matches/%s?extend[%s.team][players][users]", m.ID, pos)
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
