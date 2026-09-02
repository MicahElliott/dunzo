package dun

import (
	"fmt"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// standupCategories are the categories pulled into the deterministic
// standup export (FR-17) -- what you actually got done, plus
// call-out wins. Deliberately narrower than Summarize's full-ledger
// LLM pass; this is meant to be instant and no-network.
var standupCategories = map[string]bool{"DONE": true, "WIN": true}

// standupTag is the tag whose recurring meeting entry (if configured)
// drives the "since" boundary for gatherStandupLines -- see
// standupWindowStart.
const standupTag = "#dsu"

// standupWindowStart returns the timestamp standup items should be
// gathered from (exclusive lower bound), given now: if a "#dsu"
// RecurringMeeting is configured, that's its lastOccurrence (the
// meeting time itself) -- so the standup naturally covers "everything
// since the last time this meeting happened," spanning across
// midnight into today, not just "yesterday's ledger". Falls back to
// standupSourceDates' fixed weekday-aware heuristic (start of
// yesterday, or Friday on a Monday) if no #dsu meeting is configured,
// preserving the original behavior for anyone who hasn't set one up.
func standupWindowStart(cfg Config, now time.Time) time.Time {
	for _, m := range cfg.RecurringMeetings {
		if strings.EqualFold(m.Tag, standupTag) {
			return lastOccurrence(m, now)
		}
	}
	dates := standupSourceDates(now)
	return time.Date(dates[0].Year(), dates[0].Month(), dates[0].Day(), 0, 0, 0, 0, now.Location())
}

// standupSourceDates returns the dates whose ledgers should feed the
// standup export, given now. Normally just "yesterday". On Monday,
// also include Saturday and Sunday (in addition to Friday) in case
// weekend work was logged, since those days otherwise have no
// standup of their own to surface them. Returned oldest-first. Used
// as a fallback by standupWindowStart when no #dsu recurring meeting
// is configured, and to enumerate which ledger files to scan (a
// timestamp alone doesn't tell you which files to open).
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

// parseLedgerLineTime parses a ledger line's "[HH:MM:SS]" timestamp
// prefix combined with the given date into a full time.Time, for
// comparing against standupWindowStart's boundary. Returns ok=false
// if the line doesn't start with a well-formed "[HH:MM:SS]" stamp.
func parseLedgerLineTime(line string, date time.Time) (t time.Time, ok bool) {
	if len(line) < 10 || line[0] != '[' {
		return time.Time{}, false
	}
	end := strings.IndexByte(line, ']')
	if end < 0 {
		return time.Time{}, false
	}
	hms := line[1:end]
	parsed, err := time.ParseInLocation("15:04:05", hms, date.Location())
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, date.Location()), true
}

// gatherStandupLines reads ledger files covering standupSourceDates
// (plus today's, so a window that started yesterday but runs through
// "now" still picks up this morning's entries) and returns the
// DONE/WIN lines' text (category prefix stripped) that fall at or
// after since, oldest-first, deduplicated (exact-text repeats
// collapsed to one bullet, first occurrence kept).
func gatherStandupLines(cfg Config, now time.Time) []string {
	since := standupWindowStart(cfg, now)
	dates := append(append([]time.Time{}, standupSourceDates(now)...), now)

	seen := make(map[string]bool)
	var out []string
	for _, date := range dates {
		path := ledgerFileForDate(date)
		if path == "" {
			continue
		}
		for _, line := range readLedgerLinesFrom(path) {
			cat, text, ok := parseLedgerLine(line)
			if !ok || !standupCategories[cat] {
				continue
			}
			ts, ok := parseLedgerLineTime(line, date)
			if !ok || ts.Before(since) {
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
		return "(no DONE/WIN entries found for the covered period)"
	}
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString("\u2022 ")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}

// summarizeStandupWithCopilot runs the given (already-filtered,
// hidden items excluded) lines through the same summarizeWithCopilot
// pipeline used elsewhere, framed for a standup specifically.
func summarizeStandupWithCopilot(lines []string) (string, error) {
	return summarizeWithCopilotPrompt(
		"Summarize these standup items into a brief, well-organized "+
			"daily standup update (what was done, call out anything "+
			"notable). Be concise.", strings.Join(lines, "\n"))
}

// showGeneratedStandupSummary displays an AI-generated standup
// summary via the shared showGeneratedReport window (Copy/Save/
// Close), saving to periodReportPath("dsu", today, "20060102").
func showGeneratedStandupSummary(a fyne.App, parent fyne.Window, summary string) {
	showGeneratedReport(a, "Dunzo: Generated Standup Summary",
		periodReportPath("dsu", time.Now(), "20060102"), summary)
}

// showStandupExport builds the deterministic standup summary (FR-17
// -- no LLM/network call, pure local ledger parsing) covering
// everything since the last #dsu meeting (or the weekday-aware
// yesterday fallback), copies it to the clipboard, and shows it in a
// window with a per-item Hide (visibility-off icon) action -- Hide
// only removes an item from this temporary view, it never writes
// anything to the ledger -- and a bottom "Generate Summary" button
// that runs the still-visible (non-hidden) items through
// summarizeStandupWithCopilot and shows the result via
// showGeneratedStandupSummary.
//
// Reordering/drag-to-sort was considered but isn't implemented: Fyne
// has no built-in draggable-list-reorder widget, and building one
// from scratch (custom drag gesture handling + list reflow) was
// judged not worth the effort for this feature -- items stay in
// chronological order.
func showStandupExport(a fyne.App) {
	now := time.Now()
	cfg := LoadConfig()
	lines := gatherStandupLines(cfg, now)
	text := formatStandup(lines)
	a.Clipboard().SetContent(text)

	w := a.NewWindow("Dunzo: Standup Summary")

	itemsBox := container.NewVBox()
	hidden := make([]bool, len(lines))
	var rebuildItemsBox func()
	rebuildItemsBox = func() {
		itemsBox.RemoveAll()
		anyVisible := false
		for i, line := range lines {
			if hidden[i] {
				continue
			}
			anyVisible = true
			i := i // capture
			row := container.NewBorder(nil, nil, nil,
				newHoverIconButton(theme.Icon(theme.IconNameVisibilityOff), "Hide", func() {
					hidden[i] = true
					rebuildItemsBox()
				}),
				widget.NewLabel("\u2022 "+line))
			itemsBox.Add(row)
		}
		if !anyVisible {
			itemsBox.Add(widget.NewLabel("(nothing to show \u2014 either no DONE/WIN entries in the covered period, or everything\u2019s hidden)"))
		}
		itemsBox.Refresh()
	}
	rebuildItemsBox()

	generateBtn := widget.NewButton("Generate Summary", func() {
		var visible []string
		for i, line := range lines {
			if !hidden[i] {
				visible = append(visible, line)
			}
		}
		if len(visible) == 0 {
			dialog.ShowInformation("Nothing to Summarize", "All items are hidden (or there were none to begin with).", w)
			return
		}
		progress := dialog.NewCustomWithoutButtons("Generating Summary", widget.NewLabel("Running gh copilot, please wait\u2026"), w)
		progress.Show()
		go func() {
			summary, err := summarizeStandupWithCopilot(visible)
			fyne.Do(func() {
				progress.Hide()
				if err != nil {
					log.Println("Error generating standup summary:", err)
					dialog.ShowError(err, w)
					return
				}
				showGeneratedStandupSummary(a, w, summary)
			})
		}()
	})

	content := container.NewBorder(
		widget.NewLabel(fmt.Sprintf("Standup items since %s:", standupWindowStart(cfg, now).Format("Mon 15:04"))),
		generateBtn,
		nil, nil,
		container.NewVScroll(itemsBox),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(480, 440))
	w.Show()
}
