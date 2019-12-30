package battlefy

import (
	"testing"
)

const (
	tournamentID = "5d6fdb02c747ff732da36eb4"
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
