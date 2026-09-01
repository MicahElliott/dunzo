package dun

import (
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// periodReportPath returns DunzoDir()/<kind>-<date formatted with
// format>.md, e.g. periodReportPath("dsu", now, "20060102") or
// periodReportPath("som", from, "200601"). Unifies what were
// previously two slightly-different ad hoc conventions (dsuSavePath
// in standup.go, SOM's inline digestSavePath in som.go).
// dailysummary.go's per-ledger-directory summary-<date>.md convention
// is intentionally left alone -- it lives alongside its ledger file
// rather than at DunzoDir()'s root, a deliberately different scheme.
func periodReportPath(kind string, date time.Time, format string) string {
	return filepath.Join(DunzoDir(), kind+"-"+date.Format(format)+".md")
}

// showGeneratedReport displays a markdown report in a small
// standalone window with Copy (clipboard) and Save (writes to
// savePath) actions, plus Close -- the shared shape behind what were
// previously separate near-duplicate implementations
// (showGeneratedStandupSummary in standup.go, SOM's inline digest
// Copy/Save in som.go). title is the window title; savePath is where
// Save writes text.
func showGeneratedReport(a fyne.App, title, savePath, text string) {
	body := widget.NewRichTextFromMarkdown(text)
	body.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(body)
	scroll.SetMinSize(fyne.NewSize(0, 260))

	w := a.NewWindow(title)

	copyBtn := widget.NewButtonWithIcon("Copy", theme.Icon(theme.IconNameContentCopy), func() {
		a.Clipboard().SetContent(text)
	})
	saveBtn := widget.NewButtonWithIcon("Save", theme.Icon(theme.IconNameDocumentSave), func() {
		if err := os.WriteFile(savePath, []byte(text), 0644); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Saved", "Saved to "+savePath, w)
	})

	w.SetContent(container.NewBorder(nil,
		container.NewHBox(copyBtn, saveBtn, widget.NewButton("Close", func() { w.Close() })),
		nil, nil,
		scroll,
	))
	w.Resize(fyne.NewSize(520, 420))
	w.Show()
}
