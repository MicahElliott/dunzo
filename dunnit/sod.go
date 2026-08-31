package dun

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// showSODWindow shows a Start-of-Day readback (FR-13): today's still-
// open TODOs and current GOALs (reusing FR-07's getOpenItems), with a
// quick-entry field to log a fresh TODO or GOAL before the day gets
// going. Mirrors showEODWindow's one-window-with-everything approach
// rather than a chain of separate popups.
func showSODWindow(a fyne.App) {
	w := a.NewWindow("Dunzo: Start of Day")

	var todos, goals []OpenItem
	for _, item := range getOpenItems() {
		if item.Category == "GOAL" {
			goals = append(goals, item)
		} else {
			todos = append(todos, item)
		}
	}

	listBox := container.NewVBox()
	if len(todos) == 0 && len(goals) == 0 {
		listBox.Add(widget.NewLabel("Nothing open right now -- clean slate!"))
	}
	if len(todos) > 0 {
		listBox.Add(widget.NewLabelWithStyle("Open TODOs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		for _, item := range todos {
			listBox.Add(widget.NewLabel("- " + item.Text))
		}
	}
	if len(goals) > 0 {
		listBox.Add(widget.NewLabelWithStyle("Current GOALs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		for _, item := range goals {
			listBox.Add(widget.NewLabel("- " + item.Text))
		}
	}

	newItemCat := widget.NewSelect([]string{"TODO", "GOAL"}, nil)
	newItemCat.SetSelected("TODO")
	newItemText := widget.NewEntry()
	newItemText.SetPlaceHolder("Add a new TODO or GOAL for today...")

	addItem := func() {
		text := strings.TrimSpace(newItemText.Text)
		if text == "" {
			return
		}
		recordActivity(text, newItemCat.Selected)
		newItemText.SetText("")
		w.Close()
		showSODWindow(a) // simplest way to refresh the list after adding
	}
	newItemText.OnSubmitted = func(string) { addItem() }
	addBtn := widget.NewButton("Add", addItem)

	content := container.NewVBox(
		widget.NewLabel("Good morning! Here's where things stand:"),
		streakLabel(),
		container.NewVScroll(listBox),
		container.NewBorder(nil, nil, newItemCat, addBtn, newItemText),
		widget.NewButton("Done", func() { w.Close() }),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(420, 400))
	w.Show()
}
