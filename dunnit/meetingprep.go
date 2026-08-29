package dun

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// normalizeTag ensures s starts with "#" (adding it if the user typed
// the tag name without it), after trimming surrounding whitespace.
func normalizeTag(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "#") {
		return s
	}
	return "#" + s
}

// showMeetingPrepDialog prompts for a tag (e.g. "#jeff") and a
// free-text note, then logs a MEETING entry under that tag to
// today's ledger (FR-11). This is the capture half of the
// prep-now/agenda-later workflow -- FR-12 later pulls these back out
// grouped by tag as a ready-made agenda.
func showMeetingPrepDialog(a fyne.App, parent fyne.Window) {
	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("#tag (e.g. #jeff, #boss)")

	noteEntry := widget.NewMultiLineEntry()
	noteEntry.SetPlaceHolder("Agenda note for this meeting...")
	noteEntry.SetMinRowsVisible(3)

	content := container.NewVBox(
		widget.NewLabel("Meeting Prep"),
		tagEntry,
		noteEntry,
	)

	d := dialog.NewCustomConfirm("Meeting Prep", "Save", "Cancel", content,
		func(ok bool) {
			if !ok {
				return
			}
			tag := strings.TrimSpace(tagEntry.Text)
			note := strings.TrimSpace(noteEntry.Text)
			if tag == "" || note == "" {
				return
			}
			tag = normalizeTag(tag)
			recordActivity(tag+" "+note, "MEETING")
		}, parent)
	d.Resize(fyne.NewSize(420, 260))
	d.Show()
}
