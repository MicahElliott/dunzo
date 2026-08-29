package dun

import (
	"os"
	"path/filepath"
	"time"
)

// dailySummaryPath returns the path for date's markdown summary doc
// (FR-18), living alongside that date's ledger file in the same
// year/week/month directory (same naming scheme as getLedger, just
// "summary-" instead of "ledger-" and ".md" instead of ".txt").
func dailySummaryPath(date time.Time) (dir, path string) {
	yr, wk := date.ISOWeek()
	moname := date.Format("Jan")
	dir = ledgerDirFor(yr, wk, moname)
	path = filepath.Join(dir, "summary-"+date.Format("20060102")+".md")
	return dir, path
}

// draftDailySummary generates initial markdown content for date's
// summary doc via the existing gh copilot pipeline (summarize.go),
// scoped to just that single day's ledger. Returns "" (with the
// error) if there's nothing to summarize or the copilot call fails.
func draftDailySummary(date time.Time) (string, error) {
	ledgerText := gatherLedgerTextForDate(date)
	if ledgerText == "" {
		return "", nil
	}
	return summarizeWithCopilot(ledgerText)
}

// ensureDailySummaryDoc creates today's (or the given date's) summary
// doc with LLM-drafted content, only if it doesn't already exist --
// never overwrites existing hand-edited content (FR-18's core
// guarantee). Returns the file path and whether it was newly created.
func ensureDailySummaryDoc(date time.Time) (path string, created bool, err error) {
	dir, path := dailySummaryPath(date)
	if _, statErr := os.Stat(path); statErr == nil {
		return path, false, nil // already exists, leave it alone
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return path, false, err
	}
	content, err := draftDailySummary(date)
	if err != nil {
		return path, false, err
	}
	if content == "" {
		content = "# " + date.Format("2006-01-02") + "\n\n(no ledger entries to summarize yet)\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return path, false, err
	}
	return path, true, nil
}
