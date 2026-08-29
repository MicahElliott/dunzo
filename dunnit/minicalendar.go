package dun

import (
	"errors"
	"regexp"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// hmPattern validates a strict "HH:MM" 24-hour time string, since
// parseHM silently returns zeros on any parse failure (which would
// otherwise be indistinguishable from a legitimately-entered
// "00:00").
var hmPattern = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)

// RecurringMeeting is one user-entered recurring weekly meeting slot
// (FR-15) -- purely user-entered, no calendar/.ics/EventKit
// integration. Tag should include the leading "#" (normalized on
// save). DOW is 0=Sunday..6=Saturday (matches time.Weekday) so it
// sorts/compares naturally against time.Now().Weekday(). TimeHM is
// "HH:MM" 24-hour.
type RecurringMeeting struct {
	Tag  string `toml:"tag"`
	DOW  int    `toml:"dow"`
	Time string `toml:"time"`
}

// dowNames indexes by time.Weekday (0=Sunday..6=Saturday).
var dowNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// showMiniCalendarDialog lets the user add/edit/delete recurring
// meeting entries (FR-15), persisted in config.toml's
// recurring_meeting array-of-tables (see Config.RecurringMeetings).
func showMiniCalendarDialog(a fyne.App, parent fyne.Window) {
	cfg := LoadConfig()
	meetings := append([]RecurringMeeting(nil), cfg.RecurringMeetings...)

	list := widget.NewList(
		func() int { return len(meetings) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			m := meetings[i]
			o.(*widget.Label).SetText(m.Tag + " -- " + dowNames[m.DOW] + " " + m.Time)
		},
	)

	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("#tag (e.g. #dsu, #boss)")

	dowSelect := widget.NewSelect(dowNames, nil)
	dowSelect.SetSelected(dowNames[time.Monday])

	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("HH:MM")

	saveAll := func() {
		newCfg := cfg
		newCfg.RecurringMeetings = meetings
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, parent)
		}
	}

	addBtn := widget.NewButton("Add", func() {
		tag := normalizeTag(tagEntry.Text)
		if tag == "" {
			dialog.ShowError(errors.New("tag is required"), parent)
			return
		}
		if !hmPattern.MatchString(timeEntry.Text) {
			dialog.ShowError(errors.New("time must be HH:MM (24-hour)"), parent)
			return
		}
		dow := 0
		for i, n := range dowNames {
			if n == dowSelect.Selected {
				dow = i
			}
		}
		meetings = append(meetings, RecurringMeeting{Tag: tag, DOW: dow, Time: timeEntry.Text})
		saveAll()
		list.Refresh()
		tagEntry.SetText("")
		timeEntry.SetText("")
	})

	var selected widget.ListItemID = -1
	list.OnSelected = func(id widget.ListItemID) { selected = id }
	deleteBtn := widget.NewButton("Delete Selected", func() {
		if selected < 0 || selected >= len(meetings) {
			return
		}
		meetings = append(meetings[:selected], meetings[selected+1:]...)
		saveAll()
		list.Refresh()
		selected = -1
	})

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Recurring Meetings"),
			tagEntry,
			container.NewHBox(dowSelect, timeEntry, addBtn),
		),
		deleteBtn,
		nil, nil,
		list,
	)

	w := a.NewWindow("Recurring Meetings")
	w.SetContent(content)
	w.Resize(fyne.NewSize(420, 400))
	w.Show()
}

// nextOccurrence returns the next time (today or later) that m is
// scheduled, given now. Used by the FR-16 pre-meeting nudge check.
func nextOccurrence(m RecurringMeeting, now time.Time) time.Time {
	hh, mm := parseHM(m.Time)
	daysAhead := (m.DOW - int(now.Weekday()) + 7) % 7
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location()).AddDate(0, 0, daysAhead)
	if candidate.Before(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

// dueForPreMeetingNudge reports whether m's next occurrence starts
// within the next window duration from now (FR-16 -- "~15 min
// before"). Since the scheduler check itself runs periodically (every
// 15 min, per FR-16 v1), window should be set a bit wider than the
// check interval to avoid missing a meeting between checks.
func dueForPreMeetingNudge(m RecurringMeeting, now time.Time, window time.Duration) bool {
	next := nextOccurrence(m, now)
	delta := next.Sub(now)
	return delta > 0 && delta <= window
}
