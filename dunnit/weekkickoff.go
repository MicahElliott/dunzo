package dun

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// showWeekKickoffWindow shows the Week Kickoff (docs/kickoff-review-
// design.md): a forward-looking, weekly-scoped counterpart to
// showSODWindow -- open TODOs/GOALs readback, a quick-entry field,
// and weekly-cadence recurring-item suggestions. Deliberately does
// not duplicate Day's own daily-recurring-item surfacing (SOD already
// owns that); this only surfaces weekly-cadence items, so stacking
// Week+Day Kickoffs on a week-start day doesn't double-suggest the
// same items (see design doc's "Kickoff: overlap/stacking" section).
func showWeekKickoffWindow(a fyne.App) {
	now := time.Now()
	w := a.NewWindow("Dunzo: Week Kickoff (" + periodLabel(periodWeek, now) + ")")

	listBox := container.NewVBox()
	refreshList := func() {
		listBox.RemoveAll()
		cats, grouped := groupOpenItemsByCategory(getOpenItems())
		if len(cats) == 0 {
			listBox.Add(widget.NewLabel("Nothing open right now -- clean slate!"))
		}
		for _, cat := range cats {
			listBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, item := range grouped[cat] {
				listBox.Add(widget.NewLabel("- " + item.Text))
			}
		}
		listBox.Refresh()
	}
	refreshList()
	listScroll := container.NewVScroll(listBox)
	listScroll.SetMinSize(fyne.NewSize(0, 220))

	// Weekly-cadence recurring items due this week -- same
	// suggestion-box pattern as SOD (daily/weekly) and SOM (monthly),
	// filtered to just "weekly" here so Day's own Kickoff (which
	// includes "daily" and "weekly" together) isn't duplicating this
	// list; a user with both Day and Week Kickoff enabled will see
	// weekly items twice across the two dialogs when they coincide
	// on a week-start day, which is accepted per the design doc
	// (module stacking, not exact dedup).
	recurringBox := container.NewVBox()
	refreshRecurring := func() {
		recurringBox.RemoveAll()
		due := dueRecurringItems(LoadConfig(), now, "weekly")
		if box := recurringItemsSuggestionBox(due, refreshList); box != nil {
			recurringBox.Add(widget.NewLabelWithStyle("Recurring Items Due This Week", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			recurringBox.Add(box)
		}
		recurringBox.Refresh()
	}
	refreshRecurring()

	newItemCat := widget.NewSelect(openTrackedCategories, nil)
	newItemCat.SetSelected("GOAL")
	newItemText := widget.NewEntry()
	newItemText.SetPlaceHolder("Add a goal or open item for this week...")
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
		widget.NewLabel("Kicking off "+periodLabel(periodWeek, now)+" -- here's where things stand:"),
		listScroll,
		recurringBox,
		entryRow,
		widget.NewButton("Done", func() { w.Close() }),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(560, 480))
	w.Show()
}
