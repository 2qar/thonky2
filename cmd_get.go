package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	longDoc := "Examples:\n\n!get week\n\tShow the schedule for this week."
	AddCommand("get", "Get information from the configured spreadsheet.", longDoc, Get)
}

// Get formats information from a given spreadsheet into a Discord embed.
func Get(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	info, err := GetInfo(m.GuildID, m.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "No config for this guild.")
		return
	} else if !info.DocKey.Valid {
		s.ChannelMessageSend(m.ChannelID, "No doc key for this guild.")
		return
	}

	if len(args) == 2 {
		if args[1] == "week" {
			log.Println("getting week")
			if info.Week == nil {
				s.ChannelMessageSend(m.ChannelID, "No week schedule, something broke")
				return
			}
			embed := formatWeek(s, info.Week, info.SheetLink())
			for _, field := range embed.Fields {
				log.Println(*field)
			}
			_, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, err.Error())
			}
			log.Println("sent week :)")
		}
	}
}

// baseEmbed returns a template embed with the decorative stuff set up all ez
func baseEmbed(title, sheetLink string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Color: 0x2ecc71,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn.discordapp.com/attachments/437847669839495170/476837854966710282/thonk.png",
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name:    title,
			URL:     sheetLink,
			IconURL: "https://www.clicktime.com/images/web-based/timesheet/integration/googlesheets.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Times shown in PST",
		},
	}
}

// addTimeField adds a field to the given embed with time emotes
func addTimeField(e *discordgo.MessageEmbed, title string, startTime int) {
	timeEmotes := []string{":zero:", ":one:", ":two:", ":three:", ":four:", ":five:", ":six:", ":seven:", ":eight:", ":nine:", ":keycap_ten:", "<:eleven:548022654540578827>", "<:twelve:548220054353870849>"}
	var timeString string
	for i := 0; i < 5; i++ {
		timeString += timeEmotes[i+startTime] + ", "
	}
	timeString += timeEmotes[5+startTime]
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: title, Value: timeString})
}

// formatWeek formats week information into a Discord embed
func formatWeek(s *discordgo.Session, w *Week, sheetLink string) *discordgo.MessageEmbed {
	embed := baseEmbed("Week of "+w.Date, sheetLink)
	addTimeField(embed, "Times", 4)
	emojiGuild, err := s.Guild("437847669839495168")
	if err != nil {
		return embed
	}

	activityEmoji := func(activity string) string {
		if activity == "" || activity == "TBD" {
			return ":grey_question:"
		}
		emojiName := strings.Replace(strings.ToLower(activity), " ", "_", -1)
		for _, emoji := range emojiGuild.Emojis {
			if emoji.Name == emojiName {
				return emoji.MessageFormat()
			}
		}
		return fmt.Sprintf(":regional_indicator_%s:", strings.ToLower(string(activity[0])))
	}
	formatDay := func(day [6]string) string {
		var activityEmojis []string
		for i := 0; i < 6; i++ {
			activityEmojis = append(activityEmojis, activityEmoji(day[i]))
		}
		return strings.Join(activityEmojis, ", ")
	}

	today := int(time.Now().Weekday())
	days := w.Values()
	for i := 0; i < 7; i++ {
		activityEmojis := formatDay(days[i])
		var dayName string
		if i == today-1 {
			dayName = "**" + w.Days[i] + "**"
		} else {
			dayName = w.Days[i]
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: dayName, Value: activityEmojis, Inline: false})
	}

	return embed
}
