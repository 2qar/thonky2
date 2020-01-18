package commands

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bigheadgeorge/spreadsheet"
	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/schedule"
	"github.com/bigheadgeorge/thonky2/pkg/state"
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
	command.AddCommand("set", "Update information on the configured spreadsheet.", examples, Set)

	examples = [][2]string{
		{"!reset", "Load a given default week schedule (use !save to do that)"},
	}
	command.AddCommand("reset", "Reset the week schedule on a sheet to default", examples, Reset)

	examples = [][2]string{
		{"!set monday 4-6 scrim", "Set the 4-6 block on Monday to Scrim"},
	}
	command.AddCommand("set", "Update cells on the spreadsheet.", examples, Set)

	examples = [][2]string{
		{"!set_note monday 4-6 Inked", "Block out scrims 4-6 for Inked"},
	}
	command.AddCommand("set_note", "Add notes on the week schedule", examples, SetNote)
}

type updater func(*spreadsheet.Sheet, *spreadsheet.Cell, string)

// updateSheet updates cells, notes, whatever on the spreadsheet by parsing a whatever spaghetti people shove in as arguments
func updateSheet(s *state.State, m *discordgo.MessageCreate, args []string, validWeekArgs, validPlayerArgs []string, updater updater) (string, error) {
	sched := s.FindSchedule(m.GuildID, m.ChannelID)
	if sched == nil {
		return "", nil
	}

	if len(args) >= 3 {
		day := sched.Week.DayInt(args[1])
		if day != -1 {
			// update w/ day
			return updateRange("Weekly Schedule", sched, sched.Week.Container[day], 1, args[1:], validWeekArgs, &sched.Week, updater)
		}

		var player *schedule.Player
		playerName := strings.ToLower(args[1])
		for _, p := range sched.Players {
			if playerName == strings.ToLower(p.Name) {
				player = &p
				break
			}
		}

		if player != nil {
			day = sched.Week.DayInt(args[2])
			if day != -1 {
				// update w/ player
				return updateRange(player.Name, sched, player.Container[day], 2, args[2:], validPlayerArgs, &sched.Week, updater)
			}

			return fmt.Sprintf("Invalid day %q", args[2]), nil
		}

		return fmt.Sprintf("Invalid day / player %q", args[1]), nil
	}

	return "weird amount of args", nil
}

func updateRange(title string, sched *schedule.Schedule, cells []*spreadsheet.Cell, argStartIndex int, args, validArgs []string, week *schedule.Week, updater updater) (string, error) {
	sheet, err := sched.SheetByTitle(title)
	if err != nil {
		return err.Error(), err
	}
	updateCells, err := cellsToUpdate(sheet, cells[:], argStartIndex, week.StartTime, week.BlockLength, args)
	if err != nil {
		return fmt.Sprintf("Error parsing input: %s", err.Error()), err
	}

	parsed, err := parseArgs(args, validArgs)
	if err != nil {
		return fmt.Sprintf("Error parsing input: %s", err.Error()), err
	} else if len(cells) != len(parsed) {
		return fmt.Sprintf("Input mismatch; cell count != parsed count (%d cells != %d parsed arguments)", len(cells), len(parsed)), nil
	}
	update(sheet, updateCells, parsed, updater)

	err = sched.SyncSheet(sheet)
	if err != nil {
		return err.Error(), err
	}

	return "Updated schedule.", nil
}

// Reset loads the default week schedule for a sheet
func Reset(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	sched := s.FindSchedule(m.GuildID, m.ChannelID)
	if sched == nil {
		return "", nil
	}

	var j types.JSONText
	err := s.DB.Get(&j, "SELECT default_week FROM sheet_info WHERE id = $1")
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
	err = s.DB.ExecJSON(fmt.Sprintf("UPDATE cache SET week = $1 WHERE id = '%s'", sched.ID), sched.Week)
	if err != nil {
		return "Error caching new default week", err
	}

	return "Loaded default week schedule. :)", nil
}

// Set updates a cell on a sheet.
func Set(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	if sched := s.FindSchedule(m.GuildID, m.ChannelID); sched != nil {
		return updateSheet(s, m, args, sched.ValidActivities, []string{"Yes", "Maybe", "No"}, updateCell)
	}
	return "", nil
}

// SetNote updates a note on a sheet.
func SetNote(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	return updateSheet(s, m, args, []string{}, []string{}, updateNote)
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

func cellsToUpdate(sheet *spreadsheet.Sheet, cells []*spreadsheet.Cell, argStartIndex, startTime, blockLength int, args []string) ([]*spreadsheet.Cell, error) {
	if match, _ := regexp.MatchString(`\d{1,2}-\d{1,2}`, args[0]); match {
		rangeStart, rangeEnd, err := parseTimeRange(args[0], startTime, blockLength)
		if err != nil {
			return cells, err
		}
		if rangeStart == rangeEnd {
			cells = cells[rangeStart : rangeStart+1]
		} else {
			cells = cells[rangeStart:rangeEnd]
		}

		args = args[1:]
		argStartIndex++
	} else if i, err := strconv.Atoi(args[0]); err == nil {
		if i < startTime {
			return cells, fmt.Errorf("Invalid time: %d < %d", i, startTime)
		}
		cells = cells[i-startTime : i-startTime+1]
	}
	return cells[argStartIndex:], nil
}

func parseTimeRange(timeStr string, startTime, blockLength int) (int, int, error) {
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
	}
	if timeRange[0] > timeRange[1] {
		return -1, -1, fmt.Errorf("Invalid time range: first time > second time")
	}
	if timeRange[0]-startTime != 0 && (timeRange[0]-startTime)/blockLength != 0 {
		return -1, -1, fmt.Errorf("range does not conform to block length")
	}
	rangeStart := (timeRange[0] - startTime) / blockLength
	rangeEnd := rangeStart + ((timeRange[1] - timeRange[0]) / blockLength)
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
