package gamebattles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
)

const (
	apiv1    = "https://gb-api.majorleaguegaming.com/api/v1/"
	apiv1Web = "https://gb-api.majorleaguegaming.com/api/web/v1/"
)

// Player info API
// http://profile.majorleaguegaming.com/Tydra_/
// http://profile.majorleaguegaming.com/api/profile-page-data/Tydra_

// Player stores the information about a player.
type Player struct {
	ID       uint
	UserID   uint
	Username string
	Gamertag string
	active   bool `json:"active"`
}

func (p Player) Battletag() string {
	if m, _ := regexp.MatchString(`.+#\d{1,}`, p.Gamertag); m {
		// they put periods at the end of battletags on GameBattles for some reason :)
		return p.Gamertag[:len(p.Gamertag)-1]
	}
	return ""
}

func (p Player) Active() bool {
	return p.active
}

// Team info API
// https://gamebattles.majorleaguegaming.com/pc/overwatch/team/33834248

// Team stores team information and info on the team's players.
type Team struct {
	Players   []Player `json:"-"`
	TeamName  string   `json:"name"`
	AvatarURL string   `json:"avatarUrl"`
	URL       string   `json:"url"`
}

func (t Team) Name() string {
	return t.TeamName
}

func (t Team) Link() string {
	return t.URL
}

func (t Team) Logo() string {
	return t.AvatarURL
}

func getEndpoint(link string) ([]byte, error) {
	resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// GetTeam gets a team.
func GetTeam(id string) (Team, error) {
	b, err := getEndpoint(apiv1Web + "team-members-extended/team/" + id)
	if err != nil {
		return Team{}, err
	}

	playersJSON := struct {
		Errors []struct {
			Code string
		}
		Body []struct {
			TeamMember Player
		}
	}{}
	err = json.Unmarshal(b, &playersJSON)
	if err != nil {
		return Team{}, err
	} else if len(playersJSON.Errors) > 0 {
		return Team{}, fmt.Errorf("error getting team %q: %s", id, playersJSON.Errors[0].Code)
	}

	players := []Player{}
	for _, p := range playersJSON.Body {
		players = append(players, p.TeamMember)
	}

	b, err = getEndpoint(apiv1Web + "team-screen/" + id)
	if err != nil {
		return Team{}, err
	}

	teamJSON := struct {
		Errors []struct {
			Code string
		}
		Body struct {
			// cool name
			TeamWithEligibilityAndPremiumStatus struct {
				Team Team `json:"team"`
			}
		}
	}{}
	err = json.Unmarshal(b, &teamJSON)
	if err != nil {
		return Team{}, err
	} else if len(teamJSON.Errors) > 0 {
		return Team{}, fmt.Errorf("error getting team %q: %s", id, teamJSON.Errors[0].Code)
	}

	teamJSON.Body.TeamWithEligibilityAndPremiumStatus.Team.Players = players
	return teamJSON.Body.TeamWithEligibilityAndPremiumStatus.Team, nil
}

// GetTeams returns the teams participating in the tournament with the given ID.
func GetTeams(tournamentID string) ([]Team, error) {
	b, err := getEndpoint(apiv1 + "tournament-team/with-eligibility-and-premium-status/" + tournamentID + "?pageSize=1000&pageNumber=1")
	if err != nil {
		return []Team{}, err
	}

	teamsJSON := struct {
		Errors []struct {
			Code string
		}
		Body struct {
			Records []Team
		}
	}{}

	err = json.Unmarshal(b, &teamsJSON)
	if err != nil {
		return []Team{}, err
	} else if len(teamsJSON.Errors) > 0 {
		return []Team{}, fmt.Errorf("error getting teams in tournament %q: %s", tournamentID, teamsJSON.Errors[0].Code)
	}

	return teamsJSON.Body.Records, nil
}

// GetTournamentID gets the ID of the tournament with the given name.
// `name` should be the name at the end of the url: the name for gamebattles.com/../../tournament/Breakable-Barriers would be "Breakable-Barriers"
func GetTournamentID(name, game, platform string) (string, error) {

	b, err := getEndpoint(apiv1 + "tournaments/simple-url/" + name + "/arena-simple-url/" + game + "/platform-simple-url/" + platform)
	if err != nil {
		return "", err
	}

	tournamentJSON := struct {
		Errors []struct {
			Code string
		}
		Body struct {
			ID uint64
		}
	}{}

	err = json.Unmarshal(b, &tournamentJSON)
	if err != nil {
		return "", err
	} else if len(tournamentJSON.Errors) > 0 {
		return "", fmt.Errorf("error getting tournament %q: %s", name, tournamentJSON.Errors[0].Code)
	}

	return strconv.FormatUint(tournamentJSON.Body.ID, 10), nil
}
