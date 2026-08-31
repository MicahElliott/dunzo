package dun

import (
	"log"
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

// carryForwardItem appends a fresh line of the given category (plain
// re-listing, no state/history beyond what's already in today's
// ledger) to tomorrow's ledger for an item the user chose not to
// convert/postpone today (FR-09, extended beyond just TODO to also
// cover QUESTION carry-forward).
func carryForwardItem(category, text string) {
	appendTomorrowLine("[05:00] " + category + " " + text)
}

// eodOpenItemsSection builds one carry-forward-checkbox section (used
// for both TODO and QUESTION) of showEODWindow: a checkbox per still-
// open item of the given category, checked by default, so the user
// can carry it forward into tomorrow's ledger -- or uncheck to leave
// it behind on purpose (not forced).
func eodOpenItemsSection(category string) (box *fyne.Container, items []OpenItem, checks []*widget.Check) {
	for _, item := range getOpenItems() {
		if item.Category == category {
			items = append(items, item)
		}
	}
	box = container.NewVBox()
	checks = make([]*widget.Check, len(items))
	for i, item := range items {
		c := widget.NewCheck(item.Text, nil)
		c.SetChecked(true)
		checks[i] = c
		box.Add(c)
	}
	return box, items, checks
}

// showEODWindow recreates (in spirit) dunnit.zsh's dunnit-eod sequence:
// a short daily wrap-up showing everything logged today, an AI-drafted
// summary (editable before saving), a productivity score, a meeting-
// hours count, a sentiment rating, goals for tomorrow, and (FR-09,
// extended) a chance to carry forward any TODOs/QUESTIONs not
// resolved today. Rather than a chain of separate popups (as the
// original zsh alerter-based flow did), this is one window with all
// the questions -- simpler to implement and to answer.
func showEODWindow(a fyne.App) {
	w := a.NewWindow("Dunzo: End of Day")

	// Today's items, shown first -- read-only, so the user has the
	// full day in view before answering anything below.
	todayBody := widget.NewMultiLineEntry()
	todayBody.SetText(strings.Join(readLedgerLines(), "\n"))
	todayBody.Wrapping = fyne.TextWrapWord
	todayBody.SetMinRowsVisible(10)
	todayBody.Disable() // read-only display

	// AI-drafted summary: fed today's ledger text via the same
	// summarizeWithCopilot pipeline used elsewhere (Summarize/SOM),
	// rather than asking the user to hand-write one. Runs in the
	// background since it shells out to gh copilot; the field starts
	// with a placeholder and is editable once (or before) the draft
	// arrives, so the user can always tweak/replace it before Finalize
	// Day.
	summary := widget.NewMultiLineEntry()
	summary.SetPlaceHolder("Generating an AI summary of today, please wait (feel free to edit once it arrives, or type your own now)...")
	summary.SetMinRowsVisible(5)
	go func() {
		ledgerText := gatherLedgerTextForDate(time.Now())
		if strings.TrimSpace(ledgerText) == "" {
			return
		}
		draft, err := summarizeWithCopilot(ledgerText)
		fyne.Do(func() {
			if err != nil {
				log.Println("Error drafting EOD summary:", err)
				return
			}
			if strings.TrimSpace(summary.Text) == "" {
				summary.SetText(draft)
			}
		})
	}()

	productivity := widget.NewSelect([]string{"1", "2", "3", "4", "5"}, nil)
	productivity.SetSelected("3")

	meetingHours := widget.NewEntry()
	meetingHours.SetPlaceHolder("e.g. 2.5")

	sentiment := widget.NewSelect([]string{"Negative", "Neutral", "Positive"}, nil)
	sentiment.SetSelected("Neutral")

	goals := widget.NewMultiLineEntry()
	goals.SetPlaceHolder("Any goals for tomorrow? One per line...")
	goals.SetMinRowsVisible(3)

	// FR-09 (extended): open TODOs and QUESTIONs each get their own
	// carry-forward checkbox section.
	todoBox, openTodos, todoChecks := eodOpenItemsSection("TODO")
	questionBox, openQuestions, questionChecks := eodOpenItemsSection("QUESTION")

	items := []*widget.FormItem{
		widget.NewFormItem("Today's Items", container.NewVScroll(todayBody)),
		widget.NewFormItem("Summary", summary),
		widget.NewFormItem("Productivity (1-5)", productivity),
		widget.NewFormItem("Meeting Hours", meetingHours),
		widget.NewFormItem("Sentiment", sentiment),
		widget.NewFormItem("Tomorrow's Goals", goals),
	}
	if len(openTodos) > 0 {
		items = append(items, widget.NewFormItem("Carry Forward Open TODOs", todoBox))
	}
	if len(openQuestions) > 0 {
		items = append(items, widget.NewFormItem("Carry Forward Open QUESTIONs", questionBox))
	}
	form := widget.NewForm(items...)
	form.SubmitText = "Finalize Day"
	form.OnSubmit = func() {
		if strings.TrimSpace(summary.Text) != "" {
			recordActivity(summary.Text, "SUMMARY")
		}
		recordActivity(productivity.Selected, "PRODUCTIVITY")
		if hrs := strings.TrimSpace(meetingHours.Text); hrs != "" {
			recordActivity(hrs, "MEETING_HOURS")
		}
		recordActivity(sentiment.Selected, "SENTIMENT")

		if strings.TrimSpace(goals.Text) != "" {
			recordTomorrowGoals(strings.Split(goals.Text, "\n"))
		}
		for i, item := range openTodos {
			if todoChecks[i].Checked {
				carryForwardItem("TODO", item.Text)
			}
		}
		for i, item := range openQuestions {
			if questionChecks[i].Checked {
				carryForwardItem("QUESTION", item.Text)
			}
		}
		// FR-18: draft (if not already present) today's hand-editable
		// summary doc, now that the day's SUMMARY/PRODUCTIVITY/
		// SENTIMENT lines above have just been recorded. Gated behind
		// AutoDraftDailySummary (default off) -- open design
		// questions remain about EOD-vs-other trigger timing and how
		// this doc's content should differ from Summarize's existing
		// Day output; see docs/open-design-questions.md. Manual
		// drafting via the "Daily Summary Doc..." tray item always
		// works regardless of this setting. Runs in the background
		// since it shells out to gh copilot; opens in $EDITOR when
		// ready rather than blocking Finalize Day.
		if LoadConfig().AutoDraftDailySummary {
			go func() {
				path, _, err := ensureDailySummaryDoc(time.Now())
				if err != nil {
					log.Println("Error drafting daily summary doc:", err)
					return
				}
				if path != "" {
					openInEditor(path)
				}
			}()
		}
		w.Close()
	}

	w.SetContent(container.NewVScroll(form))
	w.Resize(fyne.NewSize(560, 820))
	w.Show()
}
