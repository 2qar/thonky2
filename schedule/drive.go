package schedule

import (
	"net/http"
	"strings"
	"time"

	"github.com/bigheadgeorge/thonky2/schedule/utils"
)

// file represents a file in Google Drive.
type file struct {
	Sheets []struct {
		ConditionalFormats []struct {
			BooleanRule struct {
				Condition struct {
					Values []struct {
						UserEnteredValue string
					}
				}
			}
		}
		Properties struct {
			Title string
		}
	}
}

// lastModified returns the last modified time of a file on Google Drive.
func lastModified(c *http.Client, sheetID string) (t time.Time, err error) {
	f := struct {
		ModifiedTime string
	}{}
	url := "https://www.googleapis.com/drive/v3/files/" + sheetID + "?fields=modifiedTime"
	err = utils.Gets(c, &f, url)
	if err != nil {
		return
	}
	f.ModifiedTime = f.ModifiedTime[:strings.LastIndex(f.ModifiedTime, ".")]
	t, err = time.Parse("2006-01-02T15:04:05", f.ModifiedTime)
	if err != nil {
		return
	}
	return
}

// validActivities returns a list of valid activities based on a spreadsheet's conditional format rules.
func validActivities(c *http.Client, sheetID string) (activities []string, err error) {
	var f file
	err = utils.Gets(c, &f, "https://sheets.googleapis.com/v4/spreadsheets/"+sheetID)
	if err != nil {
		return
	}
	for _, sheet := range f.Sheets {
		if sheet.Properties.Title == "Weekly Schedule" {
			for _, rule := range sheet.ConditionalFormats {
				for _, value := range rule.BooleanRule.Condition.Values {
					activities = append(activities, value.UserEnteredValue)
				}
			}
		}
	}
	return
}
