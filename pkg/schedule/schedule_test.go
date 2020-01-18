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
	container := *v
	if len(container) != len(data) {
		t.Fatalf("length mismatch: %d != %d", len(container), len(data))
	} else if len(container[0]) != len(data[0]) {
		t.Fatalf("row length mismatch: %d != %d", len(container[0]), len(data[0]))
	}
	for i, row := range v.Values() {
		for j, value := range row {
			if value != data[i][j] {
				t.Fatalf("%q != %q at %d,%d", value, data[i][j], i, j)
			}
		}
	}
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

func verifyWeek(week *Week, testWeek *Week, t *testing.T) {
	if week.StartTime != testWeek.StartTime {
		t.Errorf("wrong start time: %d != %d", week.StartTime, testWeek.StartTime)
	}

	if week.BlockLength != testWeek.BlockLength {
		t.Errorf("wrong block length: %d != %d", week.BlockLength, testWeek.StartTime)
	}
}

func verifyDays(days [7]string, testDays [7]string, t *testing.T) {
	for i, day := range days {
		if day != testDays[i] {
			t.Fatalf("verifyDays: %s != %s at %d", day, testDays[i], i)
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
	verifyWeek(&schedule.Week, &Week{StartTime: 4, BlockLength: 1}, t)
	verifyDays(schedule.Week.Days, [7]string{
		"Monday, 10/08",
		"Tuesday, 10/09",
		"Wednesday, 10/10",
		"Thursday, 10/11",
		"Friday, 10/12",
		"Saturday, 10/13",
		"Sunday, 10/14",
	}, t)
}

func TestScheduleFlexible(t *testing.T) {
	week := [][]string{
		{"Free", "Free", "Free"},
		{"Free", "Free", "Free"},
		{"Free", "Free", "Free"},
		{"Scrim", "Scrim", "Free"},
		{"Scrim", "Scrim", "Free"},
		{"Scrim", "Scrim", "Free"},
		{"Scrim", "Scrim", "Free"},
	}

	err := schedule.getWeek("SheetRaw")
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	verifyContainer(&schedule.Week.Container, week, t)
	if len(schedule.Week.Container[0]) != 3 {
		t.Errorf("wrong amount of activities parsed: %d != 3", len(schedule.Week.Container[0]))
	}
	verifyWeek(&schedule.Week, &Week{StartTime: 3, BlockLength: 2}, t)
	verifyDays(schedule.Week.Days, [7]string{
		"Monday, 12/16",
		"Tuesday, 12/17",
		"Wednesday, 12/18",
		"Thursday, 12/19",
		"Friday, 12/20",
		"Saturday, 12/21",
		"Sunday, 12/22",
	}, t)
}
