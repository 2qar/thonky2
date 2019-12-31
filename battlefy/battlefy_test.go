package battlefy

import (
	"testing"
)

const (
	tournamentID   = "5d6fdb02c747ff732da36eb4"
	tournamentLink = "https://battlefy.com/overwatch-open-division-north-america/2019-overwatch-open-division-practice-season-north-america/5d6fdb02c747ff732da36eb4/stage/5d7b716bb7758c268b771f83"
	teamID         = "5bfe1b9418ddd9114f14efb0"
)

func TestFindTeam(t *testing.T) {
	const teamName = "Feeders"
	var team Team
	names, err := FindTeam(tournamentID, teamName, &team)
	if err != nil {
		t.Fatalf("error finding teams: %s", err)
	} else if len(names) != 1 {
		t.Fatalf("wrong amount of names for name %q: %d != 1", teamName, len(names))
	}

	for _, player := range team.Players {
		t.Logf("%+v\n", player)
	}
}

func TestFindMatch(t *testing.T) {
	team, err := FindMatch(tournamentLink, teamID, 1)
	if err != nil {
		t.Fatalf("error finding match: %s\n", err.Error())
	} else if len(team.PersistentTeam.Name) == 0 {
		t.Logf("%+v\n", team)
		t.Fatalf("empty team!\n")
	} else if team.PersistentTeam.Name != "Event Horizon" {
		t.Fatalf("name mismatch: %s != Event Horizon", team.PersistentTeam.Name)
	}
}

func TestMatchActivePlayers(t *testing.T) {
	team := Team{
		Players: []Player{
			{PID: "5c9fbba74469a003339b8a36", IGN: "emagoot#1615"},
			{PID: "5d9431bbb2a42c6ceb618448", IGN: "mOmO!#1326"},
		},
	}
	team.PersistentTeam.PersistentPlayerIDs = []string{"5c9fbba74469a003339b8a36"}
	markActivePlayers(&team)

	if !team.Players[0].Active() {
		t.Fatalf("active player %q with id %q not marked active\n", team.Players[0].IGN, team.Players[0].PID)
	}
}
