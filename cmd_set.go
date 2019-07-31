package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/bigheadgeorge/thonky2/db"
	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx/types"
	"github.com/bigheadgeorge/spreadsheet"
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
}

// Set is used for updating info on a Spreadsheet
func Set(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "No config for this guild.")
		return
	} else if !info.DocKey.Valid {
		s.ChannelMessageSend(m.ChannelID, "No doc key for this guild.")
		return
	}

	if len(args) >= 3 {
		day := info.Week.DayInt(args[1])
		if day != -1 {
			// update w/ day
			log.Printf("update day %q w/ index %d\n", args[1], day)
			sheet, err := info.Sheet.SheetByTitle("Weekly Schedule")
			if err != nil {
				log.Println(err)
				return
			}
			err = tryUpdate(sheet, info.Week.Cells[day], info.Week.StartTime, 2, args, info.Sheet.ValidActivities)
			if err != nil {
				log.Println(err)
				s.ChannelMessageSend(m.ChannelID, err.Error())
			}

			s.ChannelMessageSend(m.ChannelID, "Updated week schedule.")
			return
		}

		var player *Player
		playerName := strings.ToLower(args[1])
		for _, p := range info.Players {
			if playerName == strings.ToLower(p.Name) {
				player = p
			}
		}

		if player != nil {
			day = info.Week.DayInt(args[2])
			if day != -1 {
				// update w/ player
				log.Printf("update player %q\n", player.Name)
				sheet, err := info.Sheet.SheetByTitle(player.Name)
				if err != nil {
					log.Println(err)
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error grabbing %s's sheet.", player.Name))
					return
				}
				err = tryUpdate(sheet, player.Cells[day], info.Week.StartTime, 3, args, []string{"Yes", "Maybe", "No"})
				if err != nil {
					log.Println(err)
					s.ChannelMessageSend(m.ChannelID, err.Error())
				}

				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Updated %s's schedule.", player.Name))
				return
			}

			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid day %q", args[2]))
			return
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid day / player %q", args[1]))
		return
	}

	s.ChannelMessageSend(m.ChannelID, "weird amount of args")
}

// Reset loads the default week schedule for a sheet
func Reset(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, "Error grabbing info")
	}

	handler, err := db.NewHandler()
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, "Error connecting to database, something stupid happened")
	}
	defer handler.Close()

	var j types.JSONText
	err = handler.Get(&j, "SELECT default_week FROM sheet_info WHERE id = $1", info.DocKey)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			s.ChannelMessageSend(m.ChannelID, "No default week schedule for this sheet")
		} else {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, "Error loading default week schedule")
		}
		return
	}

	sheet, err := info.Sheet.SheetByTitle("Weekly Schedule")
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, "Error grabbing week schedule")
		return
	}

	var w Week
	err = j.Unmarshal(&w)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, "Error parsing default week schedule, something stupid happened")
		return
	}

	activities := w.Values()
	for i, c := range info.Week.Cells {
		update(sheet, c[:], activities[i][:])
	}
	err = sheet.Synchronize()
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, "Error synchronizing sheets")
		return
	}

	s.ChannelMessageSend(m.ChannelID, "Loaded default week schedule. :)")
}

func update(sheet *spreadsheet.Sheet, cells []*spreadsheet.Cell, newValues []string) {
	safeUpdate := func(cell *spreadsheet.Cell, i int) {
		if cell.Value != newValues[i] {
			sheet.Update(int(cell.Row), int(cell.Column), newValues[i])
			cell.Value = newValues[i]
		}
	}
	if len(newValues) > 1 {
		for i, cell := range cells {
			safeUpdate(cell, i)
		}
	} else {
		for _, cell := range cells {
			safeUpdate(cell, 0)
		}
	}
}

func tryUpdate(sheet *spreadsheet.Sheet, cells [6]*spreadsheet.Cell, startTime, valueStart int, args, validArgs []string) error {
	if match, _ := regexp.MatchString(`\d{1,2}-\d{1,2}`, args[valueStart]); match {
		rangeStart, rangeEnd, err := getTimeRange(args[valueStart], startTime)
		if err != nil {
			return err
		}
		var updateCells []*spreadsheet.Cell
		if rangeStart == rangeEnd {
			updateCells = []*spreadsheet.Cell{cells[rangeStart]}
		} else {
			updateCells = cells[rangeStart:rangeEnd]
		}

		parsed, err := parseArgs(args[valueStart+1:], validArgs)
		if err != nil {
			return err
		} else if len(updateCells) != len(parsed) && len(parsed) != 1 {
			return fmt.Errorf("Invalid amount of activities for this range: %d cells =/= %d responses", len(updateCells), len(parsed))
		}

		update(sheet, updateCells, parsed)
		return sheet.Synchronize()
	} else if i, err := strconv.Atoi(args[valueStart]); err == nil {
		if i < startTime {
			return fmt.Errorf("Invalid time: %d < %d", i, startTime)
		}
		parsed, err := parseArgs(args[valueStart+1:], validArgs)
		if err != nil {
			return err
		} else if len(parsed) != 1 {
			return fmt.Errorf("Too many arguments: %d != 1", len(parsed))
		}

		update(sheet, []*spreadsheet.Cell{cells[i-4]}, parsed)
		return sheet.Synchronize()
	} else {
		parsed, err := parseArgs(args[valueStart:], validArgs)
		if err != nil {
			return err
		} else if len(parsed) != 1 {
			return fmt.Errorf("Too many arguments: %d =/= 1", len(parsed))
		}

		var updateCells []*spreadsheet.Cell
		for _, cell := range cells {
			updateCells = append(updateCells, cell)
		}
		update(sheet, updateCells, parsed)
		return sheet.Synchronize()
	}
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
