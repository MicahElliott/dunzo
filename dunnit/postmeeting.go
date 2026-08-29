package dun

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// postMeetingCategories are the quick-entry fields offered in the
// post-meeting capture session (FR-36) -- early shape per Micah:
// TIL/TODO/GOAL/RISK, expected to grow/change with real usage.
var postMeetingCategories = []string{"TIL", "TODO", "GOAL", "RISK"}

// showPostMeetingCapture starts a quick multi-category capture
// session (FR-36), tag-scoped to the same meeting tag used for prep
// (e.g. "#boss") so entries logically group with that meeting's
// MEETING-tagged prep notes. Each filled field writes its own
// correctly-categorized line, tagged with the given tag; skipped
// fields write nothing. tag may be "" (e.g. invoked from the tray
// rather than a specific recurring meeting) -- the user can still
// type one in.
func showPostMeetingCapture(a fyne.App, parent fyne.Window, tag string) {
	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("#tag (e.g. #boss) -- entries are grouped under this tag")
	tagEntry.SetText(tag)

	fields := make(map[string]*widget.Entry, len(postMeetingCategories))
	formItems := make([]*widget.FormItem, 0, len(postMeetingCategories)+1)
	formItems = append(formItems, widget.NewFormItem("Tag", tagEntry))
	for _, cat := range postMeetingCategories {
		e := widget.NewEntry()
		e.SetPlaceHolder("(skip if nothing to add)")
		fields[cat] = e
		formItems = append(formItems, widget.NewFormItem(cat, e))
	}

	content := container.NewVBox(
		widget.NewLabel("Post-Meeting Capture -- quick multi-category dump, skip any field:"),
		widget.NewForm(formItems...),
	)

	d := dialog.NewCustomConfirm("Post-Meeting Capture", "Save", "Cancel", content,
		func(ok bool) {
			if !ok {
				return
			}
			meetingTag := normalizeTag(tagEntry.Text)
			for _, cat := range postMeetingCategories {
				text := strings.TrimSpace(fields[cat].Text)
				if text == "" {
					continue
				}
				if meetingTag != "" {
					text = meetingTag + " " + text
				}
				recordActivity(text, cat)
			}
		}, parent)
	d.Resize(fyne.NewSize(480, 360))
	d.Show()
}
