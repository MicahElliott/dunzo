package dun

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

// RecurringMeeting is one user-entered recurring meeting slot
// (FR-15) -- purely user-entered, no calendar/.ics/EventKit
// integration. Tag should include the leading "#" (normalized on
// save). DOW is 0=Sunday..6=Saturday (matches time.Weekday) so it
// sorts/compares naturally against time.Now().Weekday(). Time is
// "HH:MM" 24-hour. IntervalWeeks is "every N weeks" (1 = every week,
// the common case; 2 = biweekly, etc; treated as 1 if <= 0, e.g. for
// entries saved before this field existed). AnchorDate ("YYYY-MM-DD")
// is the first occurrence's date, used to compute which weeks count
// for IntervalWeeks > 1 -- without it there'd be no way to know which
// week is "week 1" of the cadence. Set once at creation and left
// alone afterward.
type RecurringMeeting struct {
	Tag           string `toml:"tag"`
	DOW           int    `toml:"dow"`
	Time          string `toml:"time"`
	IntervalWeeks int    `toml:"interval_weeks"`
	AnchorDate    string `toml:"anchor_date"`
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
			interval := m.IntervalWeeks
			if interval <= 0 {
				interval = 1
			}
			every := "every week"
			if interval > 1 {
				every = fmt.Sprintf("every %d weeks", interval)
			}
			o.(*widget.Label).SetText(m.Tag + " -- " + dowNames[m.DOW] + " " + m.Time + " (" + every + ")")
		},
	)

	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("#tag (e.g. #dsu, #boss)")

	// Tag autocomplete, same approach as the main entry field (FR-10,
	// see ui.go) -- suggests previously-used tags from ledger
	// history as the user types "#...".
	var tagPopup *widget.PopUpMenu
	dismissTagPopup := func() {
		if tagPopup != nil {
			tagPopup.Hide()
			tagPopup = nil
		}
	}
	tagEntry.OnChanged = func(text string) {
		dismissTagPopup()
		start, fragment, ok := currentTagFragment(text, tagEntry.CursorColumn)
		if !ok || len(fragment) < 2 {
			return
		}
		matches := matchingTags(KnownTags(), fragment[1:])
		if len(matches) == 0 {
			return
		}
		items := make([]*fyne.MenuItem, len(matches))
		for i, tag := range matches {
			tag := tag
			items[i] = fyne.NewMenuItem(tag, func() {
				runes := []rune(text)
				newText := string(runes[:start]) + tag + string(runes[tagEntry.CursorColumn:])
				tagEntry.SetText(newText)
				tagEntry.CursorColumn = start + len([]rune(tag))
				tagEntry.Refresh()
				dismissTagPopup()
			})
		}
		canvas := fyne.CurrentApp().Driver().CanvasForObject(tagEntry)
		if canvas == nil {
			return
		}
		tagPopup = widget.NewPopUpMenu(fyne.NewMenu("", items...), canvas)
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(tagEntry)
		tagPopup.ShowAtPosition(pos.Add(fyne.NewPos(0, tagEntry.Size().Height)))
		fyne.Do(func() {
			canvas.Focus(tagEntry)
		})
	}

	dowSelect := widget.NewSelect(dowNames, nil)
	dowSelect.SetSelected(dowNames[time.Monday])

	// timeEntry/intervalEntry are wrapped in fixed-size GridWrap
	// containers (same fix as minsInput in ui.go) -- otherwise Fyne's
	// default layout can render a plain widget.Entry at an oddly
	// narrow width alongside other fixed-width siblings in an HBox.
	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("HH:MM")
	timeWrapper := container.NewGridWrap(fyne.NewSize(80, timeEntry.MinSize().Height), timeEntry)

	intervalEntry := widget.NewEntry()
	intervalEntry.SetPlaceHolder("1")
	intervalEntry.SetText("1")
	intervalWrapper := container.NewGridWrap(fyne.NewSize(50, intervalEntry.MinSize().Height), intervalEntry)

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
		interval, err := strconv.Atoi(strings.TrimSpace(intervalEntry.Text))
		if err != nil || interval <= 0 {
			dialog.ShowError(errors.New("every N weeks must be a positive number"), parent)
			return
		}
		dow := 0
		for i, n := range dowNames {
			if n == dowSelect.Selected {
				dow = i
			}
		}
		meetings = append(meetings, RecurringMeeting{
			Tag:           tag,
			DOW:           dow,
			Time:          timeEntry.Text,
			IntervalWeeks: interval,
			AnchorDate:    time.Now().Format("2006-01-02"),
		})
		saveAll()
		list.Refresh()
		tagEntry.SetText("")
		timeEntry.SetText("")
		intervalEntry.SetText("1")
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
			container.NewHBox(dowSelect, timeWrapper, widget.NewLabel("every"), intervalWrapper, widget.NewLabel("week(s)"), addBtn),
		),
		deleteBtn,
		nil, nil,
		list,
	)

	w := a.NewWindow("Recurring Meetings")
	w.SetContent(content)
	w.Resize(fyne.NewSize(480, 400))
	w.Show()
}

// nextOccurrence returns the next time (today or later) that m is
// scheduled, given now. For IntervalWeeks > 1, only weeks that are an
// exact multiple of IntervalWeeks away from AnchorDate's week count;
// candidates in between are skipped. Falls back to every-week
// behavior if AnchorDate is missing/unparseable (e.g. legacy entries).
func nextOccurrence(m RecurringMeeting, now time.Time) time.Time {
	hh, mm := parseHM(m.Time)
	interval := m.IntervalWeeks
	if interval <= 0 {
		interval = 1
	}

	daysAhead := (m.DOW - int(now.Weekday()) + 7) % 7
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location()).AddDate(0, 0, daysAhead)
	if candidate.Before(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}

	if interval == 1 {
		return candidate
	}

	anchor, err := time.ParseInLocation("2006-01-02", m.AnchorDate, now.Location())
	if err != nil {
		return candidate
	}
	for {
		weeksSinceAnchor := int(candidate.Sub(anchor).Hours() / (24 * 7))
		if weeksSinceAnchor >= 0 && weeksSinceAnchor%interval == 0 {
			return candidate
		}
		candidate = candidate.AddDate(0, 0, 7)
	}
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

// lastOccurrence returns the most recent past occurrence of m at or
// before now (the mirror of nextOccurrence). Used by FR-36's post-
// meeting nudge, which looks backward instead of forward.
func lastOccurrence(m RecurringMeeting, now time.Time) time.Time {
	next := nextOccurrence(m, now)
	if next.Equal(now) {
		return next
	}
	// nextOccurrence never returns a time <= now, so the actual most
	// recent past occurrence is exactly one cadence period before
	// whatever it returns when starting the search from just after
	// that prior occurrence. Simplest robust approach: step backward
	// a week at a time (respecting IntervalWeeks) until we find an
	// occurrence at or before now.
	interval := m.IntervalWeeks
	if interval <= 0 {
		interval = 1
	}
	candidate := next.AddDate(0, 0, -7*interval)
	for candidate.After(now) {
		candidate = candidate.AddDate(0, 0, -7*interval)
	}
	return candidate
}

// dueForPostMeetingNudge reports whether m's most recent occurrence
// ended between minAfter and maxAfter ago from now (FR-36 -- surfaced
// shortly after a recurring meeting's scheduled time passes). Assumes
// a nominal meeting length isn't tracked, so "ended" is approximated
// as "started" -- good enough for a soft suggestion nudge.
func dueForPostMeetingNudge(m RecurringMeeting, now time.Time, minAfter, maxAfter time.Duration) bool {
	last := lastOccurrence(m, now)
	elapsed := now.Sub(last)
	return elapsed >= minAfter && elapsed <= maxAfter
}
