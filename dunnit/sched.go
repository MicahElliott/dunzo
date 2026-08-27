package dun

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"github.com/go-co-op/gocron/v2"
)

// parseHM parses "HH:MM" into hour, minute ints. Returns zeros on
// parse failure (caller should treat that as "not configured").
func parseHM(s string) (hour, minute int) {
	fmt.Sscanf(s, "%d:%d", &hour, &minute)
	return
}

// withinWorkHours reports whether now falls between the configured
// day_start and day_end (inclusive), Mon-Fri only.
func withinWorkHours(cfg Config, now time.Time) bool {
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	startH, startM := parseHM(cfg.DayStart)
	endH, endM := parseHM(cfg.DayEnd)
	start := time.Date(now.Year(), now.Month(), now.Day(), startH, startM, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), endH, endM, 0, 0, now.Location())
	return !now.Before(start) && !now.After(end)
}

// Schedule sets up the recurring popups (hourly activity prompt, and a
// lunchtime goals reminder), reading times from config.toml. It shows
// (raises) the given main window rather than just sending a passive
// notification, since the whole point is to prompt for input.
func Schedule(a fyne.App, w fyne.Window) gocron.Scheduler {
	cfg := LoadConfig()

	s, err := gocron.NewScheduler()
	if err != nil {
		fmt.Println("Error creating scheduler:", err)
		return s
	}

	_, err = s.NewJob(
		gocron.CronJob(fmt.Sprintf("%d * * * *", cfg.HourlyMinute), false),
		gocron.NewTask(func() {
			if !withinWorkHours(cfg, time.Now()) {
				return
			}
			a.SendNotification(fyne.NewNotification(
				"Dunzo", "What are you working on?"))
			fyne.Do(func() {
				w.Show()
				w.RequestFocus()
			})
		}),
	)
	if err != nil {
		fmt.Println("Error scheduling hourly job:", err)
	}

	if lh, lm := parseHM(cfg.LunchTime); lh != 0 || lm != 0 {
		_, err = s.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(lh), uint(lm), 0))),
			gocron.NewTask(func() {
				if !withinWorkHours(cfg, time.Now()) {
					return
				}
				a.SendNotification(fyne.NewNotification(
					"Dunzo Lunchtime", "How are your goals coming along?"))
				fyne.Do(func() {
					w.Show()
					w.RequestFocus()
				})
			}),
		)
		if err != nil {
			fmt.Println("Error scheduling lunchtime job:", err)
		}
	}

	if eh, em := parseHM(cfg.DayEnd); eh != 0 || em != 0 {
		_, err = s.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(eh), uint(em), 0))),
			gocron.NewTask(func() {
				now := time.Now()
				if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
					return
				}
				a.SendNotification(fyne.NewNotification(
					"Dunzo", "End of day! Let's wrap up."))
				fyne.Do(func() {
					showEODWindow(a)
				})
			}),
		)
		if err != nil {
			fmt.Println("Error scheduling end-of-day job:", err)
		}
	}

	s.Start()
	return s
}
