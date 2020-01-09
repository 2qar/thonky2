package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/bigheadgeorge/spreadsheet"
	"github.com/bigheadgeorge/thonky2/schedule"
	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx/types"
)

func init() {
	examples := [][2]string{
		{"!set <player name> <day name> <time range> <availability>", "Update player availability."},
		{"!set <day name> <time range> <activity / activities>", "Update schedule."},
		{"To give multiple responses / activities, use commas:", "!set tydra monday 4-6 no, yes"},
		{"Give one response over a range to set it all to that one response:", "!set monday 4-10 free"},
	}
	AddCommand("set", "Update information on the configured spreadsheet.", examples, Set)

	examples = [][2]string{
		{"!reset", "Load a given default week schedule (use !save to do that)"},
	}
	AddCommand("reset", "Reset the week schedule on a sheet to default", examples, Reset)

	examples = [][2]string{
		{"!schedule monday 4-6 Inked", "Block out scrims 4-6 for Inked"},
	}
	AddCommand("schedule", "Add notes on the week schedule", examples, Schedule)
}

// Set is used for updating info on a Spreadsheet
func Set(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No config for this guild.", nil
	} else if _, err := DB.SpreadsheetID(team.ID); err != nil {
		return "No doc key for this guild.", nil
	}
	sched := team.Schedule()

	if len(args) >= 3 {
		day := sched.Week.DayInt(args[1])
		if day != -1 {
			// update w/ day
			log.Printf("update day %q w/ index %d\n", args[1], day)
			sheet, err := sched.SheetByTitle("Weekly Schedule")
			if err != nil {
				return err.Error(), err
			}
			err = tryUpdate(sheet, sched.Week.Container[day], sched.Week.StartTime, 2, args, sched.ValidActivities, updateCell)
			if err != nil {
				return err.Error(), err
			}
			err = sched.SyncSheet(sheet)
			if err != nil {
				return err.Error(), err
			}

			return "Updated week schedule.", nil
		}

		var player *schedule.Player
		playerName := strings.ToLower(args[1])
		for _, p := range sched.Players {
			if playerName == strings.ToLower(p.Name) {
				player = &p
			}
		}

		if player != nil {
			day = sched.Week.DayInt(args[2])
			if day != -1 {
				// update w/ player
				log.Printf("update player %q\n", player.Name)
				sheet, err := sched.SheetByTitle(player.Name)
				if err != nil {
					return fmt.Sprintf("Error grabbing %s's sheet.", player.Name), err
				}
				err = tryUpdate(sheet, player.Container[day], sched.Week.StartTime, 3, args, []string{"Yes", "Maybe", "No"}, updateCell)
				if err != nil {
					return err.Error(), err
				}
				err = sched.SyncSheet(sheet)
				if err != nil {
					return err.Error(), err
				}

				return fmt.Sprintf("Updated %s's schedule.", player.Name), nil
			}

			return fmt.Sprintf("Invalid day %q", args[2]), nil
		}

		return fmt.Sprintf("Invalid day / player %q", args[1]), nil
	}

	return "weird amount of args", nil
}

// Reset loads the default week schedule for a sheet
func Reset(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No team in this guild / channel", nil
	}
	sched := team.Schedule()

	spreadsheetID, err := DB.SpreadsheetID(team.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No spreadsheet for this team.", nil
		}
		return fmt.Sprintf("Error getting spreadsheet id: %s", err.Error()), err
	}

	var j types.JSONText
	err = DB.Get(&j, "SELECT default_week FROM sheet_info WHERE id = $1")
	if err != nil {
		if err == sql.ErrNoRows {
			return "No default week schedule for this sheet", nil
		} else {
			return "Error loading default week schedule", err
		}
	}

	sheet, err := sched.SheetByTitle("Weekly Schedule")
	if err != nil {
		return "Error grabbing week schedule", err
	}

	var w schedule.Week
	err = j.Unmarshal(&w)
	if err != nil {
		return "Error parsing default week schedule, something stupid happened", err
	}

	activities := w.Values()
	for i, c := range sched.Week.Container {
		update(sheet, c[:], activities[i][:], updateCell)
	}
	err = sched.SyncSheet(sheet)
	if err != nil {
		return "Error synchronizing sheets", err
	}
	err = DB.ExecJSON(fmt.Sprintf("UPDATE cache SET week = $1 WHERE id = '%s'", spreadsheetID), sched.Week)
	if err != nil {
		return "Error caching new default week", err
	}

	return "Loaded default week schedule. :)", nil
}

// Schedule updates notes on the week schedule, used for scheduling Scrims and stuff like that
func Schedule(s *discordgo.Session, m *discordgo.MessageCreate, args []string) (string, error) {
	team := FindTeam(m.GuildID, m.ChannelID)
	if team == nil {
		return "No team for this guild / channel", nil
	} else if _, err := DB.SpreadsheetID(team.ID); err != nil {
		return "No spreadsheet for this team.", nil
	}
	sched := team.Schedule()

	if len(args) >= 3 {
		day := sched.Week.DayInt(args[1])
		if day != -1 {
			sheet, err := sched.SheetByTitle("Weekly Schedule")
			if err != nil {
				return fmt.Sprintf("Error grabbing week schedule: %s", err), err
			}
			err = tryUpdate(sheet, sched.Week.Container[day], sched.Week.StartTime, 2, args, []string{}, updateNote)
			if err != nil {
				return "Error updating notes: " + err.Error(), err
			}
			err = sched.SyncSheet(sheet)
			if err != nil {
				return fmt.Sprintf("Error updating notes: %s", err), err
			}

			return "Updated scrim schedule.", nil
		} else {
			return fmt.Sprintf("Invalid day %q.", args[1]), nil
		}
	} else {
		return "weird amount of args", nil
	}
}

func update(sheet *spreadsheet.Sheet, cells []*spreadsheet.Cell, newValues []string, updater func(*spreadsheet.Sheet, *spreadsheet.Cell, string)) {
	if len(newValues) > 1 {
		for i, cell := range cells {
			updater(sheet, cell, newValues[i])
		}
	} else {
		for _, cell := range cells {
			updater(sheet, cell, newValues[0])
		}
	}
}

func updateCell(sheet *spreadsheet.Sheet, cell *spreadsheet.Cell, val string) {
	if cell.Value != val {
		sheet.Update(int(cell.Row), int(cell.Column), val)
		cell.Value = val
	}
}

func updateNote(sheet *spreadsheet.Sheet, cell *spreadsheet.Cell, val string) {
	if cell.Note != val {
		lowerVal := strings.ToLower(val)
		if lowerVal == "empty" || lowerVal == "none" || lowerVal == "blank" {
			val = ""
		}
		sheet.UpdateNote(int(cell.Row), int(cell.Column), val)
		cell.Note = val
	}
}

func tryUpdate(sheet *spreadsheet.Sheet, cells [6]*spreadsheet.Cell, startTime, valueStart int, args, validArgs []string, updater func(*spreadsheet.Sheet, *spreadsheet.Cell, string)) error {
	var updateCells []*spreadsheet.Cell
	var parsed []string
	var err error
	if match, _ := regexp.MatchString(`\d{1,2}-\d{1,2}`, args[valueStart]); match {
		rangeStart, rangeEnd, err := getTimeRange(args[valueStart], startTime)
		if err != nil {
			return err
		}
		if rangeStart == rangeEnd {
			updateCells = []*spreadsheet.Cell{cells[rangeStart]}
		} else {
			updateCells = cells[rangeStart:rangeEnd]
		}

		parsed, err = parseArgs(args[valueStart+1:], validArgs)
		if err != nil {
			return err
		} else if len(updateCells) != len(parsed) && len(parsed) != 1 {
			return fmt.Errorf("Invalid amount of activities for this range: %d cells =/= %d responses", len(updateCells), len(parsed))
		}
	} else if i, err := strconv.Atoi(args[valueStart]); err == nil {
		if i < startTime {
			return fmt.Errorf("Invalid time: %d < %d", i, startTime)
		}
		parsed, err = parseArgs(args[valueStart+1:], validArgs)
		if err != nil {
			return err
		} else if len(parsed) != 1 {
			return fmt.Errorf("Too many arguments: %d != 1", len(parsed))
		}

		updateCells[0] = cells[i-4]
	} else {
		parsed, err = parseArgs(args[valueStart:], validArgs)
		if err != nil {
			return err
		} else if len(parsed) != 1 {
			return fmt.Errorf("Too many arguments: %d =/= 1", len(parsed))
		}

		for _, cell := range cells {
			updateCells = append(updateCells, cell)
		}
	}
	update(sheet, updateCells, parsed, updater)
	return err
}

func getTimeRange(timeStr string, startTime int) (int, int, error) {
	timeStrings := strings.Split(timeStr, "-")
	var timeRange [2]int
	for i, timeStr := range timeStrings {
		time, err := strconv.Atoi(timeStr)
		if err != nil {
			return -1, -1, err
		}
		timeRange[i] = time
	}
	if timeRange[0] < startTime {
		return -1, -1, fmt.Errorf("Invalid start time")
	} else if timeRange[0] > timeRange[1] {
		return -1, -1, fmt.Errorf("Invalid time range: first time > second time")
	}
	rangeStart := timeRange[0] - startTime
	rangeEnd := rangeStart + (timeRange[1] - timeRange[0])
	return rangeStart, rangeEnd, nil
}

// parseArgs takes a list of unformatted arguments and tries to match them with a given list of valid arguments.
func parseArgs(args []string, validArgs []string) ([]string, error) {
	var argString string
	if len(args) > 1 {
		argString = strings.Join(args, " ")
	} else {
		argString = args[0]
	}
	csv := strings.Split(argString, ", ")

	if len(validArgs) == 0 {
		return csv, nil
	}

	var parsed []string
	for _, activity := range csv {
		found := false
		for _, valid := range validArgs {
			if strings.ToLower(activity) == strings.ToLower(valid) {
				found = true
				parsed = append(parsed, valid)
				break
			}
		}
		if !found {
			return []string{}, fmt.Errorf("Invalid activity: %q", activity)
		}
	}

	return parsed, nil
}
