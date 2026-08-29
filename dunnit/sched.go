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

	// intervalDuration is how often this job fires, from
	// cfg.NudgeIntervalMinutes (FR-04; falls back to 60 if unset/
	// invalid, e.g. an old config.toml predating this key). FR-01: if
	// the user already logged an entry more recently than this
	// interval, skip the nudge -- they're clearly already engaged, no
	// need to interrupt.
	intervalMinutes := cfg.NudgeIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 60
	}
	intervalDuration := time.Duration(intervalMinutes) * time.Minute

	_, err = s.NewJob(
		gocron.DurationJob(intervalDuration),
		gocron.NewTask(func() {
			now := time.Now()
			if !withinWorkHours(cfg, now) {
				return
			}
			if last := LastActivityAt(); !last.IsZero() && now.Sub(last) < intervalDuration {
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
		fmt.Println("Error scheduling interval job:", err)
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

	// FR-13: Start-of-Day nudge, fires once per workday near
	// cfg.DayStart, showing today's open TODOs/GOALs (readback) and a
	// chance to add more before the day gets going.
	if sh, sm := parseHM(cfg.DayStart); sh != 0 || sm != 0 {
		_, err = s.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(sh), uint(sm), 0))),
			gocron.NewTask(func() {
				now := time.Now()
				if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
					return
				}
				a.SendNotification(fyne.NewNotification(
					"Dunzo", "Good morning! Here's where things stand."))
				fyne.Do(func() {
					showSODWindow(a)
				})
			}),
		)
		if err != nil {
			fmt.Println("Error scheduling start-of-day job:", err)
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

	// FR-16: pre-meeting nudge, checking every 15 min (per Micah's
	// call -- simplest fixed interval, not tied to NudgeIntervalMinutes)
	// whether any FR-15 recurring meeting's next occurrence starts
	// within the next ~15 min. firedFor dedupes so the same occurrence
	// doesn't nudge repeatedly across multiple 15-min checks while
	// still inside the window. TODO(FR-17): once a standup export
	// exists, special-case a configurable "standup" tag (e.g. #dsu) to
	// show that instead of the generic Meeting Prep dialog.
	firedFor := map[string]time.Time{} // "tag" -> occurrence time already nudged for
	_, err = s.NewJob(
		gocron.DurationJob(15*time.Minute),
		gocron.NewTask(func() {
			now := time.Now()
			cfg := LoadConfig()
			for _, m := range cfg.RecurringMeetings {
				if !dueForPreMeetingNudge(m, now, 15*time.Minute) {
					continue
				}
				occ := nextOccurrence(m, now)
				if fired, ok := firedFor[m.Tag]; ok && fired.Equal(occ) {
					continue
				}
				firedFor[m.Tag] = occ
				a.SendNotification(fyne.NewNotification(
					"Dunzo", "Upcoming meeting "+m.Tag+" at "+m.Time))
				fyne.Do(func() {
					w.Show()
					showMeetingPrepDialog(a, w)
				})
			}
		}),
	)
	if err != nil {
		fmt.Println("Error scheduling pre-meeting nudge job:", err)
	}

	s.Start()
	return s
}
