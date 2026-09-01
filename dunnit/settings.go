package dun

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func SetThings() {
	fmt.Println("Setting things")
}

// showSettings pops up a window to view/edit config.toml values
// (day_start, day_end, hourly_minute, lunch_time).
func showSettings(a fyne.App) {
	cfg := LoadConfig()
	w := a.NewWindow("Dunzo Settings")

	dayStart := widget.NewEntry()
	dayStart.SetText(cfg.DayStart)

	dayEnd := widget.NewEntry()
	dayEnd.SetText(cfg.DayEnd)

	nudgeInterval := widget.NewEntry()
	nudgeInterval.SetText(strconv.Itoa(cfg.NudgeIntervalMinutes))

	lunchTime := widget.NewEntry()
	lunchTime.SetText(cfg.LunchTime)

	digestDay := widget.NewSelect([]string{"", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}, nil)
	digestDay.SetSelected(cfg.WeeklyDigestDay)

	digestTime := widget.NewEntry()
	digestTime.SetText(cfg.WeeklyDigestTime)
	digestTime.SetPlaceHolder("HH:MM")

	autoDraft := widget.NewCheck("", nil)
	autoDraft.SetChecked(cfg.AutoDraftDailySummary)

	snoozeMinutes := widget.NewEntry()
	snoozeMinutes.SetText(strconv.Itoa(cfg.SnoozeMinutes))

	skipHolidays := widget.NewCheck("", nil)
	skipHolidays.SetChecked(cfg.SkipUSFederalHolidays)

	form := widget.NewForm(
		widget.NewFormItem("Day Start (HH:MM)", dayStart),
		widget.NewFormItem("Day End (HH:MM)", dayEnd),
		widget.NewFormItem("Nudge Interval (minutes)", nudgeInterval),
		widget.NewFormItem("Lunch Time (HH:MM)", lunchTime),
		widget.NewFormItem("Weekly Digest Day", digestDay),
		widget.NewFormItem("Weekly Digest Time (HH:MM)", digestTime),
		widget.NewFormItem("Auto-draft Daily Summary at EOD", autoDraft),
		widget.NewFormItem("Default Snooze (minutes)", snoozeMinutes),
		widget.NewFormItem("Skip US Federal Holidays", skipHolidays),
	)
	form.OnSubmit = func() {
		minutes, err := strconv.Atoi(nudgeInterval.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Nudge Interval must be a number: %w", err), w)
			return
		}
		snooze, err := strconv.Atoi(snoozeMinutes.Text)
		if err != nil || snooze <= 0 {
			dialog.ShowError(fmt.Errorf("Default Snooze must be a positive number"), w)
			return
		}
		// Start from the loaded config rather than a blank Config{}
		// so fields not represented in this form (e.g.
		// RecurringMeetings, FR-15) aren't silently wiped out on save.
		newCfg := cfg
		newCfg.DayStart = dayStart.Text
		newCfg.DayEnd = dayEnd.Text
		newCfg.NudgeIntervalMinutes = minutes
		newCfg.LunchTime = lunchTime.Text
		newCfg.WeeklyDigestDay = digestDay.Selected
		newCfg.WeeklyDigestTime = digestTime.Text
		newCfg.AutoDraftDailySummary = autoDraft.Checked
		newCfg.SnoozeMinutes = snooze
		newCfg.SkipUSFederalHolidays = skipHolidays.Checked
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, w)
			return
		}
		w.Close()
	}
	form.SubmitText = "Save"

	// Recurring Meetings (FR-15) is its own dialog/config section, but
	// Settings is a natural place to also reach it from -- both are
	// "configure how Dunzo behaves" surfaces.
	recurringMeetingsBtn := widget.NewButton("Recurring Meetings...", func() {
		showMiniCalendarDialog(a, w)
	})

	recurringItemsBtn := widget.NewButton("Recurring Items...", func() {
		showRecurringItemsDialog(a, w)
	})

	w.SetContent(container.NewVBox(form, recurringMeetingsBtn, recurringItemsBtn))
	w.Resize(fyne.NewSize(320, 380))
	w.Show()
}
