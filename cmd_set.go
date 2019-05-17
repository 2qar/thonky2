package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	spreadsheet "gopkg.in/Iwark/spreadsheet.v2"
)

func init() {
	examples := [][2]string{
		{"!set <player name> <day name> <time range> <availability>", "Update player availability."},
		{"!set <day name> <time range> <activity / activities>", "Update schedule."},
		{"To give multiple responses / activities, use commas:", "!set tydra monday 4-6 no, yes"},
		{"Give one response over a range to set it all to that one response:", "!set monday 4-10 free"},
	}
	AddCommand("set", "Update information on the configured spreadsheet.", examples, Set)
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
		day := dayInt(args[1])
		if day != -1 {
			// update w/ day
			log.Printf("update day %q w/ index %d\n", args[1], day)
			sheet, err := info.Sheet.SheetByTitle("Weekly Schedule")
			if err != nil {
				log.Println(err)
				return
			}
			err = tryUpdate(sheet, info.Week.Cells[day], 2, args, info.Sheet.ValidActivities)
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
			day = dayInt(args[2])
			if day != -1 {
				// update w/ player
				log.Printf("update player %q\n", player.Name)
				sheet, err := info.Sheet.SheetByTitle(player.Name)
				if err != nil {
					log.Println(err)
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error grabbing %s's sheet.", player.Name))
					return
				}
				err = tryUpdate(sheet, player.Cells[day], 3, args, []string{"Yes", "Maybe", "No"})
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

func update(sheet *spreadsheet.Sheet, cells []*spreadsheet.Cell, newValues []string) error {
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
	err := sheet.Synchronize()
	return err
}

func tryUpdate(sheet *spreadsheet.Sheet, cells [6]*spreadsheet.Cell, valueStart int, args, validArgs []string) error {
	if match, _ := regexp.MatchString(`\d{1,2}-\d{1,2}`, args[valueStart]); match {
		rangeStart, rangeEnd, err := getTimeRange(args[valueStart])
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

		return update(sheet, updateCells, parsed)
	} else if i, err := strconv.Atoi(args[valueStart]); err == nil {
		if i < 4 {
			return fmt.Errorf("Invalid time: %d < 4", i)
		}
		parsed, err := parseArgs(args[valueStart+1:], validArgs)
		if err != nil {
			return err
		} else if len(parsed) != 1 {
			return fmt.Errorf("Too many arguments: %d != 1", len(parsed))
		}

		return update(sheet, []*spreadsheet.Cell{cells[i-4]}, parsed)
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
		return update(sheet, updateCells, parsed)
	}
}

func getTimeRange(timeStr string) (int, int, error) {
	timeStrings := strings.Split(timeStr, "-")
	var timeRange [2]int
	for i, timeStr := range timeStrings {
		time, err := strconv.Atoi(timeStr)
		if err != nil {
			return -1, -1, err
		}
		timeRange[i] = time
	}
	if timeRange[0] < 4 {
		return -1, -1, fmt.Errorf("Invalid start time")
	} else if timeRange[0] > timeRange[1] {
		return -1, -1, fmt.Errorf("Invalid time range: first time > second time")
	}
	rangeStart := timeRange[0] - 4
	rangeEnd := rangeStart + (timeRange[1] - timeRange[0])
	return rangeStart, rangeEnd, nil
}

// dayInt gets a weekday int from a day name.
func dayInt(dayName string) int {
	day := -1
	if len(dayName) >= 6 {
		dayName = strings.ToLower(dayName)
		for i := 0; i < 7; i++ {
			currName := strings.ToLower(time.Weekday(i).String())
			if dayName == currName || dayName[:3] == currName[:3] {
				day = Weekday(i)
				break
			}
		}
	}
	return day
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
