package dun

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
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

// replaceLedgerLineCategoryAt rewrites the line at idx to use
// newCategory instead of its current category, preserving its
// original timestamp and text. No-op if idx is out of range or the
// line isn't well-formed. Used by recordExtended to retroactively
// mark a prior DONE entry as ONGOING when Ditto is used on it.
func replaceLedgerLineCategoryAt(idx int, newCategory string) error {
	lines := readLedgerLines()
	if idx < 0 || idx >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[idx], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[idx] = parts[0] + " " + newCategory + " " + parts[2]
	return writeLedgerLines(lines)
}

// replaceLedgerLineTextAt rewrites the line at idx to use newText
// instead of its current text, preserving its original timestamp and
// category, and trimming newText's leading/trailing whitespace. No-op
// if idx is out of range or the line isn't well-formed. Used by
// showEditItemDialog's Save action.
func replaceLedgerLineTextAt(idx int, newText string) error {
	lines := readLedgerLines()
	if idx < 0 || idx >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[idx], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[idx] = parts[0] + " " + parts[1] + " " + strings.TrimSpace(newText)
	return writeLedgerLines(lines)
}

// showEditItemDialog opens a small modal (dialog.NewCustomConfirm --
// not a separate window) pre-filled with item.Text, letting the user
// edit and save it back over the original ledger line (preserving
// timestamp/category), or cancel. Calls onSave after a successful
// save so callers can refresh dependent UI. Used by Daybook's inline
// ✏️ Edit action, currently on Activity items -- generic enough to
// reuse for Planned/Reflections too if wanted later.
func showEditItemDialog(parent fyne.Window, item OpenItem, onSave func()) {
	entry := widget.NewEntry()
	entry.SetText(item.Text)
	d := dialog.NewCustomConfirm("Edit Entry", "Save", "Cancel", entry, func(save bool) {
		if !save {
			return
		}
		if err := replaceLedgerLineTextAt(item.LineIndex, entry.Text); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		onSave()
	}, parent)
	// Enter submits (same as clicking Save) -- Confirm() runs the
	// callback above with save=true, same as the Save button.
	entry.OnSubmitted = func(string) { d.Confirm() }
	// Esc dismisses (same as clicking Cancel) -- Fyne's CustomConfirm
	// doesn't wire this by default. Registered on the parent window's
	// canvas (standard Fyne shortcut pattern) rather than the entry
	// itself, since Escape isn't a key Entry's own TypedKey handling
	// reacts to. Fyne doesn't provide a per-dialog "closed" removal
	// hook for canvas shortcuts, but that's harmless here: the
	// shortcut just calls Dismiss() on an already-hidden dialog if
	// triggered again later, a no-op.
	parent.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEscape},
		func(fyne.Shortcut) { d.Dismiss() })
	// Fyne's default dialog width is quite narrow for a full ledger
	// line of text -- widen it so longer entries aren't cramped/
	// wrapped awkwardly while editing.
	d.Resize(fyne.NewSize(520, 160))
	d.Show()
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
