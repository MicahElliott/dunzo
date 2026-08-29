package dun

import (
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// annualReviewCategories are the categories gathered for the Annual
// Review (FR-22): IMPACT/MILESTONE per the FR spec, plus WIN
// ("optionally", per the FR) since it's cheap to include and framed
// around accomplishments too.
var annualReviewCategories = map[string]bool{
	"IMPACT": true, "MILESTONE": true, "WIN": true,
}

const annualReviewPrompt = "Summarize the following IMPACT/MILESTONE/WIN " +
	"ledger entries from across a full calendar year into a narrative " +
	"performance-review-style summary, framed around accomplishments and " +
	"impact rather than day-to-day activity. Group related items together " +
	"and highlight the most significant ones."

// showAnnualReviewDialog lets the user pick a year (default: current)
// and generates a narrative summary via the shared Summarize/gh
// copilot plumbing, scoped to that year's IMPACT/MILESTONE/WIN lines.
func showAnnualReviewDialog(a fyne.App, parent fyne.Window) {
	currentYear := time.Now().Year()
	yearEntry := widget.NewEntry()
	yearEntry.SetText(strconv.Itoa(currentYear))

	d := dialog.NewCustomConfirm("Annual Review", "Generate", "Cancel",
		container.NewVBox(
			widget.NewLabel("Gather IMPACT/MILESTONE/WIN entries for year:"),
			yearEntry,
		),
		func(ok bool) {
			if !ok {
				return
			}
			year, err := strconv.Atoi(yearEntry.Text)
			if err != nil {
				year = currentYear
			}
			runAnnualReview(a, year)
		}, parent)
	d.Show()
}

func runAnnualReview(a fyne.App, year int) {
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(year, time.December, 31, 23, 59, 59, 0, time.Local)
	ledgerText := gatherLedgerTextForRange(from, to, annualReviewCategories)
	if ledgerText == "" {
		w := a.NewWindow("Dunzo: Annual Review")
		w.SetContent(widget.NewLabel("No IMPACT/MILESTONE/WIN entries found for that year."))
		w.Show()
		return
	}

	progress := a.NewWindow("Dunzo: Generating Annual Review...")
	progress.SetContent(widget.NewLabel(
		"Asking gh copilot to summarize, please wait...\n" +
			"The generated report will be copied to your clipboard automatically."))
	progress.Show()

	go func() {
		summary, err := summarizeWithCopilotPrompt(annualReviewPrompt, ledgerText)
		fyne.Do(func() {
			progress.Close()
			w := a.NewWindow("Dunzo: Annual Review " + strconv.Itoa(year))
			if err != nil {
				w.SetContent(widget.NewLabel("Error running gh copilot:\n" + err.Error()))
			} else {
				a.Clipboard().SetContent(summary)
				body := widget.NewMultiLineEntry()
				body.SetText(summary)
				body.Wrapping = fyne.TextWrapWord
				w.SetContent(container.NewVScroll(body))
			}
			w.Resize(fyne.NewSize(600, 500))
			w.Show()
		})
	}()
}
