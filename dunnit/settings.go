package dun

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
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

	form := widget.NewForm(
		widget.NewFormItem("Day Start (HH:MM)", dayStart),
		widget.NewFormItem("Day End (HH:MM)", dayEnd),
		widget.NewFormItem("Nudge Interval (minutes)", nudgeInterval),
		widget.NewFormItem("Lunch Time (HH:MM)", lunchTime),
	)
	form.OnSubmit = func() {
		minutes, err := strconv.Atoi(nudgeInterval.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Nudge Interval must be a number: %w", err), w)
			return
		}
		newCfg := Config{
			DayStart:             dayStart.Text,
			DayEnd:               dayEnd.Text,
			NudgeIntervalMinutes: minutes,
			LunchTime:            lunchTime.Text,
		}
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, w)
			return
		}
		w.Close()
	}
	form.SubmitText = "Save"

	w.SetContent(form)
	w.Resize(fyne.NewSize(320, 180))
	w.Show()
}
