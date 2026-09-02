package dun

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// searchResult is one matching line found by searchLedgers, with the
// source file's base name for context.
type searchResult struct {
	file string
	line string
}

// searchLedgers scans every ledger file (allLedgerFiles, from
// summarize.go) for lines containing query as a case-insensitive
// substring, returning matches in file-walk order (roughly
// chronological, since ledger dirs/files sort that way). Empty query
// matches nothing (avoids an accidental full dump).
func searchLedgers(query string) []searchResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	lowerQuery := strings.ToLower(query)
	var out []searchResult
	for _, path := range allLedgerFiles() {
		for _, line := range readLedgerLinesFrom(path) {
			if strings.Contains(strings.ToLower(line), lowerQuery) {
				out = append(out, searchResult{file: filepath.Base(path), line: line})
			}
		}
	}
	return out
}

// showSearchDialog lets the user search across all ledger history by
// keyword/tag/category (FR-21), showing matches with file context.
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showSearchDialog(a fyne.App) {
	w := a.NewWindow("Dunzo: Search Ledger History")

	queryEntry := widget.NewEntry()
	queryEntry.SetPlaceHolder("Search term (tag, category, or keyword)\u2026")

	results := widget.NewMultiLineEntry()
	results.Wrapping = fyne.TextWrapOff
	results.SetMinRowsVisible(15)

	runSearch := func() {
		matches := searchLedgers(queryEntry.Text)
		if len(matches) == 0 {
			results.SetText("(no matches)")
			return
		}
		var sb strings.Builder
		for _, m := range matches {
			sb.WriteString(m.file + ": " + m.line + "\n")
		}
		results.SetText(sb.String())
	}
	queryEntry.OnSubmitted = func(string) { runSearch() }
	searchBtn := widget.NewButton("Search", runSearch)

	content := container.NewVBox(
		container.NewBorder(nil, nil, nil, searchBtn, queryEntry),
		results,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(560, 480))
	w.Show()
}
