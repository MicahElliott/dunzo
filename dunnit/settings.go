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

	extendWorkWeek := widget.NewCheck("", nil)
	extendWorkWeek.SetChecked(cfg.ExtendWorkWeekTo7Days)

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
		widget.NewFormItem("Extend Work Week to 7 Days", extendWorkWeek),
	)

	// Kickoff/Review section (docs/kickoff-review-design.md): one
	// enable-checkbox pair + one theme dropdown per unit. Built as a
	// second widget.NewForm (rather than folding into the form above)
	// since it's a visually distinct block of settings; both forms
	// share the same OnSubmit-driven save via periodToggles below.
	type unitRow struct {
		period      summaryPeriod
		kickoff     *widget.Check
		review      *widget.Check
		themeSelect *widget.Select
	}
	var unitRows []unitRow
	for _, p := range []summaryPeriod{periodDay, periodWeek, periodMonth, periodQuarter, periodYear} {
		kickoffCheck := widget.NewCheck("", nil)
		kickoffCheck.SetChecked(kickoffEnabled(cfg, p))
		reviewCheck := widget.NewCheck("", nil)
		reviewCheck.SetChecked(reviewEnabled(cfg, p))
		themeSelect := widget.NewSelect(themeOptions(), nil)
		themeSelect.SetSelected(themeDisplayNames[themeFor(cfg, p)])
		unitRows = append(unitRows, unitRow{period: p, kickoff: kickoffCheck, review: reviewCheck, themeSelect: themeSelect})
	}
	periodItems := []*widget.FormItem{}
	for _, row := range unitRows {
		periodItems = append(periodItems,
			widget.NewFormItem(string(row.period)+" Kickoff", row.kickoff),
			widget.NewFormItem(string(row.period)+" Review", row.review),
			widget.NewFormItem(string(row.period)+" Theme", row.themeSelect),
		)
	}
	periodForm := widget.NewForm(periodItems...)

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
		newCfg.ExtendWorkWeekTo7Days = extendWorkWeek.Checked
		for _, row := range unitRows {
			setKickoffEnabled(&newCfg, row.period, row.kickoff.Checked)
			setReviewEnabled(&newCfg, row.period, row.review.Checked)
			if theme := themeFromDisplayName(row.themeSelect.Selected); theme != "" {
				setTheme(&newCfg, row.period, theme)
			}
		}
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, w)
			return
		}
		RebuildTrayMenu()
		w.Close()
	}
	form.SubmitText = "Save"

	// Recurring Meetings (FR-15) is its own dialog/config section, but
	// Settings is a natural place to also reach it from -- both are
	// "configure how Dunzo behaves" surfaces.
	recurringMeetingsBtn := widget.NewButton("Recurring Meetings\u2026", func() {
		showMiniCalendarDialog(a, w)
	})

	recurringItemsBtn := widget.NewButton("Recurring Items\u2026", func() {
		showRecurringItemsDialog(a, w)
	})

	w.SetContent(container.NewVScroll(container.NewVBox(form,
		widget.NewLabelWithStyle("Kickoff / Review", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		periodForm,
		recurringMeetingsBtn, recurringItemsBtn)))
	w.Resize(fyne.NewSize(420, 620))
	w.Show()
}
