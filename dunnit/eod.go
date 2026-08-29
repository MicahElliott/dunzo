package dun

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// tomorrowLedgerPath returns the ledger directory and filename for
// tomorrow's date (same year/week/month path scheme as getLedger, but
// for time.Now().AddDate(0, 0, 1) instead of today).
func tomorrowLedgerPath() (string, string) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	yr, wk := tomorrow.ISOWeek()
	moname := tomorrow.Format("Jan")
	fname0 := "ledger-" + tomorrow.Format("20060102") + ".txt"
	fpath := filepath.Join(DunzoDir(), strconv.Itoa(yr), "w"+strconv.Itoa(wk)+"-"+moname)
	fname := filepath.Join(fpath, fname0)
	return fpath, fname
}

// appendTomorrowLine appends a single pre-formatted ledger line (sans
// trailing newline) to tomorrow's ledger file, creating the directory
// and file as needed.
func appendTomorrowLine(line string) error {
	fpath, fname := tomorrowLedgerPath()
	if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
		return err
	}
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

// recordTomorrowGoals appends each non-blank line as a GOAL entry to
// tomorrow's ledger file (so it's ready to go first thing).
func recordTomorrowGoals(lines []string) {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		appendTomorrowLine("[05:00] GOAL " + line)
	}
}

// carryForwardTodo appends a fresh TODO line (plain re-listing, no
// state/history beyond what's already in today's ledger) to
// tomorrow's ledger for an item the user chose not to convert/
// postpone today (FR-09).
func carryForwardTodo(text string) {
	appendTomorrowLine("[05:00] TODO " + text)
}

// showEODWindow recreates (in spirit) dunnit.zsh's dunnit-eod sequence:
// a short daily wrap-up capturing a summary, a productivity score, a
// sentiment rating, goals for tomorrow, and (FR-09) a chance to carry
// forward any TODOs not converted/postponed today. Rather than a
// chain of separate popups (as the original zsh alerter-based flow
// did), this is one window with all the questions -- simpler to
// implement and to answer.
func showEODWindow(a fyne.App) {
	w := a.NewWindow("Dunzo: End of Day")

	summary := widget.NewMultiLineEntry()
	summary.SetPlaceHolder("Summarize your day in a couple sentences...")
	summary.SetMinRowsVisible(3)

	productivity := widget.NewSelect([]string{"1", "2", "3", "4", "5"}, nil)
	productivity.SetSelected("3")

	sentiment := widget.NewSelect([]string{"Negative", "Neutral", "Positive"}, nil)
	sentiment.SetSelected("Neutral")

	goals := widget.NewMultiLineEntry()
	goals.SetPlaceHolder("Any goals for tomorrow? One per line...")
	goals.SetMinRowsVisible(3)

	// FR-09: any TODOs still open (not converted to DONE or postponed
	// to SOMEDAY) get a checkbox, checked by default, so the user can
	// carry them forward into tomorrow's ledger -- or uncheck to
	// leave one behind on purpose (not forced).
	var openTodos []OpenItem
	for _, item := range getOpenItems() {
		if item.Category == "TODO" {
			openTodos = append(openTodos, item)
		}
	}
	carryForwardBox := container.NewVBox()
	checks := make([]*widget.Check, len(openTodos))
	for i, item := range openTodos {
		c := widget.NewCheck(item.Text, nil)
		c.SetChecked(true)
		checks[i] = c
		carryForwardBox.Add(c)
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Summary", summary),
		widget.NewFormItem("Productivity (1-5)", productivity),
		widget.NewFormItem("Sentiment", sentiment),
		widget.NewFormItem("Tomorrow's Goals", goals),
	}
	if len(openTodos) > 0 {
		items = append(items, widget.NewFormItem("Carry Forward Open TODOs", carryForwardBox))
	}
	form := widget.NewForm(items...)
	form.SubmitText = "Finalize Day"
	form.OnSubmit = func() {
		if strings.TrimSpace(summary.Text) != "" {
			recordActivity(summary.Text, "SUMMARY")
		}
		recordActivity(productivity.Selected, "PRODUCTIVITY")
		recordActivity(sentiment.Selected, "SENTIMENT")

		if strings.TrimSpace(goals.Text) != "" {
			recordTomorrowGoals(strings.Split(goals.Text, "\n"))
		}
		for i, item := range openTodos {
			if checks[i].Checked {
				carryForwardTodo(item.Text)
			}
		}
		w.Close()
	}

	w.SetContent(container.NewVScroll(form))
	w.Resize(fyne.NewSize(420, 400))
	w.Show()
}
