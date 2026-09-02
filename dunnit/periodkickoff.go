package dun

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// periodKickoffTitle/periodRecurringCadence/periodKickoffDefaultCat
// are the small per-unit differences showPeriodKickoffWindow needs --
// everything else about a Kickoff dialog's shape is identical across
// units with no special-cased dialog of their own (i.e. all but Day/
// Month, which keep their existing bespoke SOD/SOM dialogs).
func periodRecurringCadence(period summaryPeriod) string {
	switch period {
	case periodWeek:
		return "weekly"
	case periodMonth:
		return "monthly"
	default:
		return "" // Quarter/Year have no recurring-item cadence (recurring.go's cadenceOptions is daily/weekly/monthly only)
	}
}

// showPeriodKickoffWindow shows a generic Kickoff dialog (docs/
// kickoff-review-design.md) for period -- open TODOs/GOALs readback,
// a quick-entry field, and (for Week/Month, which have a matching
// recurring-item cadence) recurring-item suggestions. Used for Week,
// Quarter, and Year, which have no bespoke dialog of their own (unlike
// Day/Month's existing SOD/SOM). Deliberately does not duplicate a
// smaller unit's own recurring-item surfacing when Kickoffs stack on
// a shared boundary day (see design doc's "Kickoff: overlap/stacking"
// section) -- each unit only ever surfaces its own cadence's items.
func showPeriodKickoffWindow(a fyne.App, period summaryPeriod) {
	now := time.Now()
	cfg := LoadConfig()
	label := periodLabel(cfg, period, now)
	w := a.NewWindow("Dunzo: " + string(period) + " Kickoff (" + label + ")")

	listBox := container.NewVBox()
	refreshList := func() {
		listBox.RemoveAll()
		cats, grouped := groupOpenItemsByCategory(getOpenItems())
		if len(cats) == 0 {
			listBox.Add(widget.NewLabel("Nothing open right now \u2014 clean slate!"))
		}
		for _, cat := range cats {
			listBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, item := range grouped[cat] {
				listBox.Add(widget.NewLabel("\u2022 " + item.Text))
			}
		}
		listBox.Refresh()
	}
	refreshList()
	listScroll := container.NewVScroll(listBox)
	listScroll.SetMinSize(fyne.NewSize(0, 220))

	recurringBox := container.NewVBox()
	cadence := periodRecurringCadence(period)
	refreshRecurring := func() {
		recurringBox.RemoveAll()
		if cadence == "" {
			return
		}
		due := dueRecurringItems(cfg, now, cadence)
		if box := recurringItemsSuggestionBox(due, refreshList); box != nil {
			recurringBox.Add(widget.NewLabelWithStyle("Recurring Items Due This "+string(period), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			recurringBox.Add(box)
		}
		recurringBox.Refresh()
	}
	refreshRecurring()

	newItemCat := widget.NewSelect(openTrackedCategories, nil)
	newItemCat.SetSelected("GOAL")
	newItemText := widget.NewEntry()
	newItemText.SetPlaceHolder("Add a goal or open item for this " + strings.ToLower(string(period)) + "\u2026")
	addItem := func() {
		text := strings.TrimSpace(newItemText.Text)
		if text == "" {
			return
		}
		recordActivity(text, newItemCat.Selected)
		newItemText.SetText("")
		refreshList()
		refreshRecurring()
	}
	newItemText.OnSubmitted = func(string) { addItem() }
	addBtn := widget.NewButton("Add", addItem)
	entryRow := container.New(newStretchRowLayout(newItemText), newItemCat, newItemText, addBtn)

	content := container.NewVBox(
		widget.NewLabel("Kicking off "+label+" \u2014 here\u2019s where things stand:"),
		listScroll,
		recurringBox,
		entryRow,
		widget.NewButton("Done", func() { w.Close() }),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(560, 480))
	w.Show()
}
