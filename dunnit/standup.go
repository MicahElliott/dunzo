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

// standupSourceDates returns the dates whose ledgers should feed the
// standup export, given now. Normally just "yesterday". On Monday,
// also include Saturday and Sunday (in addition to Friday) in case
// weekend work was logged, since those days otherwise have no
// standup of their own to surface them. Returned oldest-first.
func standupSourceDates(now time.Time) []time.Time {
	yesterday := now.AddDate(0, 0, -1)
	if now.Weekday() != time.Monday {
		return []time.Time{yesterday}
	}
	friday := now.AddDate(0, 0, -3)
	saturday := now.AddDate(0, 0, -2)
	sunday := yesterday
	return []time.Time{friday, saturday, sunday}
}

// ledgerFileForDate returns the ledger file path for date if it
// exists among allLedgerFiles(), or "" if none.
func ledgerFileForDate(date time.Time) string {
	target := "ledger-" + date.Format("20060102") + ".txt"
	for _, path := range allLedgerFiles() {
		if strings.HasSuffix(path, target) {
			return path
		}
	}
	return ""
}

// gatherStandupLines reads the standup source dates' ledgers (see
// standupSourceDates -- normally just the last workday, but Fri+Sat+
// Sun on a Monday) and returns their DONE/WIN lines' text (category
// prefix stripped), oldest-first, deduplicated (exact-text repeats
// collapsed to one bullet, first occurrence kept).
func gatherStandupLines(now time.Time) []string {
	seen := make(map[string]bool)
	var out []string
	for _, date := range standupSourceDates(now) {
		path := ledgerFileForDate(date)
		if path == "" {
			continue
		}
		for _, line := range readLedgerLinesFrom(path) {
			cat, text, ok := parseLedgerLine(line)
			if !ok || !standupCategories[cat] {
				continue
			}
			if seen[text] {
				continue
			}
			seen[text] = true
			out = append(out, text)
		}
	}
	return out
}

// formatStandup renders lines as a simple bullet list, or a
// placeholder message if there's nothing to report.
func formatStandup(lines []string) string {
	if len(lines) == 0 {
		return "(no DONE/WIN entries found for the covered day(s))"
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
