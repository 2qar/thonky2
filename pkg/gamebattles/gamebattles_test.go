package gamebattles

import "testing"

const (
	teamID             = "33834248"
	tournamentID       = "135008"
	tournamentName     = "Breakable-Barriers-NA-1"
	tournamentGame     = "overwatch"
	tournamentPlatform = "pc"
	participants       = 194
)

func TestGetTeam(t *testing.T) {
	_, err := GetTeam(teamID)
	if err != nil {
		t.Fatalf("error getting team w/ id %q: %s", teamID, err)
	} /*else if (team == Team{}) {
		t.Fatalf("empty team")
	}*/
}

func TestGetTeams(t *testing.T) {
	teams, err := GetTeams(tournamentID)
	if err != nil {
		t.Fatalf("error getting teams for tournament %q: %s", tournamentID, err)
	} else if len(teams) != participants {
		t.Fatalf("wrong amount of teams: %d != %d", len(teams), participants)
	}
}

func TestGetTournamentID(t *testing.T) {
	id, err := GetTournamentID(tournamentName, tournamentGame, tournamentPlatform)
	if err != nil {
		t.Fatalf("error getting team ID: %s", err)
	} else if id != tournamentID {
		t.Fatalf("mismatched IDs: %s != %s", id, teamID)
	}
}
