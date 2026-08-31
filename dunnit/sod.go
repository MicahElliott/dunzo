package dun

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// showSODWindow shows a Start-of-Day readback (FR-13): today's still-
// open items (TODO/GOAL/WAITING/QUESTION/FIXME/RISK, per
// openTrackedCategories) grouped by category, with a quick-entry
// field to log a fresh one before the day gets going. Mirrors
// showEODWindow's one-window-with-everything approach rather than a
// chain of separate popups.
func showSODWindow(a fyne.App) {
	w := a.NewWindow("Dunzo: Start of Day")

	listBox := container.NewVBox()

	// listAsText renders the currently-shown open items as plain
	// text, for the Copy button -- the list itself isn't selectable
	// (it's built of widget.Labels, not a text widget), so this is
	// the simplest way to let the user get this content into a doc/
	// clipboard elsewhere.
	var listAsText func() string

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

		listAsText = func() string {
			var b strings.Builder
			for i, cat := range cats {
				if i > 0 {
					b.WriteString("\n")
				}
				b.WriteString(categoryPlural(cat) + "\n")
				for _, item := range grouped[cat] {
					b.WriteString("- " + item.Text + "\n")
				}
			}
			return b.String()
		}
	}
	refreshList()

	// Scroll area for the list is given a tall fixed min-height
	// (enough for ~10 lines) so a short list doesn't render as a tiny
	// sliver, and a long list scrolls internally rather than growing
	// the whole window unboundedly.
	listScroll := container.NewVScroll(listBox)
	listScroll.SetMinSize(fyne.NewSize(0, 260))

	newItemCat := widget.NewSelect(openTrackedCategories, nil)
	newItemCat.SetSelected("TODO")
	newItemText := widget.NewEntry()
	newItemText.SetPlaceHolder("Add a new open item for today...")

	// Tag autocomplete (FR-10), same pattern as the main Daybook entry
	// (ui.go) and Recurring Meetings' tag field (minicalendar.go).
	var tagPopup *widget.PopUpMenu
	dismissTagPopup := func() {
		if tagPopup != nil {
			tagPopup.Hide()
			tagPopup = nil
		}
	}
	newItemText.OnChanged = func(text string) {
		dismissTagPopup()
		start, fragment, ok := currentTagFragment(text, newItemText.CursorColumn)
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
				newText := string(runes[:start]) + tag + string(runes[newItemText.CursorColumn:])
				newItemText.SetText(newText)
				newItemText.CursorColumn = start + len([]rune(tag))
				newItemText.Refresh()
				dismissTagPopup()
			})
		}
		canvas := fyne.CurrentApp().Driver().CanvasForObject(newItemText)
		if canvas == nil {
			return
		}
		tagPopup = widget.NewPopUpMenu(fyne.NewMenu("", items...), canvas)
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(newItemText)
		tagPopup.ShowAtPosition(pos.Add(fyne.NewPos(0, newItemText.Size().Height)))
		fyne.Do(func() {
			canvas.Focus(newItemText)
		})
	}

	addItem := func() {
		text := strings.TrimSpace(newItemText.Text)
		if text == "" {
			return
		}
		recordActivity(text, newItemCat.Selected)
		newItemText.SetText("")
		refreshList() // rebuild the list in place, rather than closing/reopening the window
	}
	newItemText.OnSubmitted = func(string) { addItem() }
	addBtn := widget.NewButton("Add", addItem)

	copyBtn := widget.NewButton("Copy", func() {
		if listAsText != nil {
			a.Clipboard().SetContent(listAsText())
		}
	})

	entryRow := container.New(newStretchRowLayout(newItemText), newItemCat, newItemText, addBtn)

	// Note: new items logged here go straight to the ledger, but
	// Daybook's own "Upcoming" section only refreshes when Daybook
	// itself is shown/interacted with -- so if Daybook is already
	// open in the background, it won't reflect these until it's
	// closed and reopened (or otherwise refreshed).
	syncNote := widget.NewLabel("Note: new items here will show up in Daybook's Upcoming section next time it's opened.")
	syncNote.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		widget.NewLabel("Good morning! Here's where things stand:"),
		streakLabel(),
		listScroll,
		copyBtn,
		entryRow,
		syncNote,
		widget.NewButton("Done", func() { w.Close() }),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(560, 480))
	w.Show()
}
