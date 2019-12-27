package schedule

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/bigheadgeorge/spreadsheet"
	"golang.org/x/oauth2/google"
)

const sheetID = "1FFMJ3L9ZynKHXGwJ-XJpPK3C8CmzizpFlxsVJaAdW7o"

var (
	service  *spreadsheet.Service
	client   *http.Client
	schedule *Schedule
)

func TestMain(m *testing.M) {
	b, err := ioutil.ReadFile("../service_account.json")
	if err != nil {
		panic(err)
	}
	c, err := google.JWTConfigFromJSON(b, spreadsheet.Scope, DriveScope)
	if err != nil {
		panic(err)
	}
	client = c.Client(context.Background())
	service = spreadsheet.NewServiceWithClient(client)

	schedule, err = New(service, client, sheetID)
	if err != nil {
		panic(err)
	}
	err = schedule.Update()
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func verifyContainer(v *Container, data [][]string, t *testing.T) {
	for i, row := range v.Values() {
		for j, value := range row {
			if value != data[i][j] {
				t.Fatalf("%q != %q at %d,%d", value, data[i][j], i, j)
			}
		}
	}
}

func TestScheduleWeek(t *testing.T) {
	week := [][]string{
		{"Free", "Scrim", "Scrim", "Scrim", "Scrim", "Free"},
		{"Free", "Scrim", "Scrim", "Free", "Free", "Free"},
		{"Free", "Scrim", "Scrim", "Free", "Free", "Free"},
		{"Free", "Free", "Free", "Scrim", "Scrim", "Player VOD"},
		{"Free", "Scrim", "Scrim", "Scrim", "Scrim", "Free"},
		{"Free", "Free", "Free", "Free", "Free", "Free"},
		{"Free", "Free", "Free", "Free", "Free", "Free"},
	}
	verifyContainer(&schedule.Week.Container, week, t)
}

func TestSchedulePlayers(t *testing.T) {
	availability := [][]string{
		{"Maybe", "Yes", "Yes", "Yes", "Yes", "No"},
		{"Maybe", "Yes", "Yes", "No", "Yes", "Maybe"},
		{"No", "Maybe", "Yes", "Yes", "No", "Maybe"},
		{"No", "Maybe", "Yes", "Yes", "Yes", "Maybe"},
		{"No", "Maybe", "Yes", "Yes", "Yes", "Maybe"},
		{"Maybe", "Yes", "Yes", "Yes", "Yes", "Yes"},
		{"Maybe", "Yes", "Yes", "Yes", "Yes", "Yes"},
	}
	var p Player
	for _, p = range schedule.Players {
		if p.Name == "Taub" {
			break
		}
	}
	verifyContainer(&p.Container, availability, t)
}
