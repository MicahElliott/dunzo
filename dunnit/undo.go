package dun

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// removeLastLedgerLine rewrites today's ledger file with its last
// line removed. Always operates on the literal current contents of
// the file (re-read fresh, not a cached value), so it stays correct
// even if the file was edited externally since the app last wrote to
// it (FR-08). No-op if the ledger is empty/missing.
func removeLastLedgerLine() error {
	lines := readLedgerLines()
	if len(lines) == 0 {
		return nil
	}
	return writeLedgerLines(lines[:len(lines)-1])
}

// replaceLastLedgerLine rewrites today's ledger file with its last
// line replaced by newLine. Same freshness guarantee as
// removeLastLedgerLine. No-op if the ledger is empty/missing.
func replaceLastLedgerLine(newLine string) error {
	lines := readLedgerLines()
	if len(lines) == 0 {
		return nil
	}
	lines[len(lines)-1] = newLine
	return writeLedgerLines(lines)
}

// writeLedgerLines overwrites today's ledger file with the given
// lines (each gets a trailing newline).
func writeLedgerLines(lines []string) error {
	_, fname := getLedger()
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l + "\n"); err != nil {
			return err
		}
	}
	return nil
}

// showUndoEditLastEntry opens a small window showing the literal last
// line of today's ledger (freshly read from disk), with an editable
// text field and two actions: "Undo" (remove the line entirely) and
// "Save Edit" (replace the line with the edited text) (FR-08). Calls
// onChange after either action succeeds, so callers can refresh any
// dependent UI (e.g. Daybook's "last entry" label, Upcoming list).
func showUndoEditLastEntry(a fyne.App, onChange func()) {
	lines := readLedgerLines()
	if len(lines) == 0 {
		dialog.ShowInformation("Nothing to Undo/Edit",
			"Today's ledger is empty.", nil)
		return
	}
	last := lines[len(lines)-1]

	w := a.NewWindow("Dunzo: Undo/Edit Last Entry")

	entry := widget.NewMultiLineEntry()
	entry.SetText(last)

	undoBtn := widget.NewButton("Undo (Remove)", func() {
		if err := removeLastLedgerLine(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		onChange()
		w.Close()
	})
	saveBtn := widget.NewButton("Save Edit", func() {
		if err := replaceLastLedgerLine(entry.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		onChange()
		w.Close()
	})

	w.SetContent(container.NewBorder(
		widget.NewLabel("Last ledger line:"), nil, nil, nil,
		container.NewBorder(nil, container.NewHBox(undoBtn, saveBtn), nil, nil, entry),
	))
	w.Resize(fyne.NewSize(420, 200))
	w.Show()
}
