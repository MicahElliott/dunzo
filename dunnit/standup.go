package dun

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// standupCategories are the categories pulled into the deterministic
// standup export (FR-17) -- what you actually got done, plus
// call-out wins. Deliberately narrower than Summarize's full-ledger
// LLM pass; this is meant to be instant and no-network.
var standupCategories = map[string]bool{"DONE": true, "WIN": true}

// lastWorkdayLedgerFile returns the ledger file path for the most
// recent workday before today (skips back over Sat/Sun so a Monday
// standup still pulls Friday's entries), or "" if none exists.
func lastWorkdayLedgerFile(now time.Time) string {
	d := now.AddDate(0, 0, -1)
	for i := 0; i < 7; i++ { // bounded loop, no infinite spin if something's off
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			break
		}
		d = d.AddDate(0, 0, -1)
	}
	target := "ledger-" + d.Format("20060102") + ".txt"
	for _, path := range allLedgerFiles() {
		if strings.HasSuffix(path, target) {
			return path
		}
	}
	return ""
}

// gatherStandupLines reads the last workday's ledger and returns its
// DONE/WIN lines' text (category prefix stripped), in original order.
func gatherStandupLines(now time.Time) []string {
	path := lastWorkdayLedgerFile(now)
	if path == "" {
		return nil
	}
	lines := readLedgerLinesFrom(path)
	var out []string
	for _, line := range lines {
		cat, text, ok := parseLedgerLine(line)
		if !ok || !standupCategories[cat] {
			continue
		}
		out = append(out, text)
	}
	return out
}

// formatStandup renders lines as a simple bullet list, or a
// placeholder message if there's nothing to report.
func formatStandup(lines []string) string {
	if len(lines) == 0 {
		return "(no DONE/WIN entries found for the last workday)"
	}
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString("- ")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}

// showStandupExport builds the deterministic standup summary (FR-17
// -- no LLM/network call, pure local ledger parsing), copies it to
// the clipboard, and shows it in a window.
func showStandupExport(a fyne.App) {
	lines := gatherStandupLines(time.Now())
	text := formatStandup(lines)
	a.Clipboard().SetContent(text)

	body := widget.NewMultiLineEntry()
	body.SetText(text)
	body.Wrapping = fyne.TextWrapWord

	w := a.NewWindow("Dunzo: Standup Summary")
	w.SetContent(container.NewVScroll(body))
	w.Resize(fyne.NewSize(480, 400))
	w.Show()
}
