package dun

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// priorMonthRange returns the [from, to] calendar-month bounds for
// the month immediately before now's month.
func priorMonthRange(now time.Time) (time.Time, time.Time) {
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	firstOfPriorMonth := firstOfThisMonth.AddDate(0, -1, 0)
	lastOfPriorMonth := firstOfThisMonth.AddDate(0, 0, -1)
	lastOfPriorMonth = time.Date(lastOfPriorMonth.Year(), lastOfPriorMonth.Month(), lastOfPriorMonth.Day(), 23, 59, 59, 0, now.Location())
	return firstOfPriorMonth, lastOfPriorMonth
}

// ideaSomedayCategories are the categories reviewed in SOM step 2.
var ideaSomedayCategories = map[string]bool{"IDEA": true, "SOMEDAY": true}

// ideaSomedayItem is one IDEA/SOMEDAY line found in the prior month,
// pending a promote/drop decision in the SOM wizard.
type ideaSomedayItem struct {
	Category string
	Text     string
}

// gatherIdeaSomedayItems scans ledger files dated within [from, to]
// for IDEA/SOMEDAY lines, in first-seen order. No resolved/dropped
// tracking beyond this run -- SOM step 2 is a one-time triage per
// invocation (append-only design: promoting/dropping never touches
// the original lines).
func gatherIdeaSomedayItems(from, to time.Time) []ideaSomedayItem {
	var out []ideaSomedayItem
	for _, path := range allLedgerFiles() {
		date := ledgerFileDate(path)
		if date == nil || date.Before(from) || date.After(to) {
			continue
		}
		for _, line := range readLedgerLinesFrom(path) {
			cat, text, ok := parseLedgerLine(line)
			if !ok || !ideaSomedayCategories[cat] {
				continue
			}
			out = append(out, ideaSomedayItem{Category: cat, Text: text})
		}
	}
	return out
}

// showSOMWindow runs the Start-of-Month wizard (FR-14), unifying the
// PRD's "EOM review" and "SOM wizard" into one 4-step sequence shown
// as a single scrollable window (same one-window-with-everything
// approach as SOD/EOD) rather than separate chained popups:
//  1. Prior month's digest (reuses the Summarize/gh copilot pipeline,
//     backgrounded, filled in once ready).
//  2. Review/triage prior month's open IDEA/SOMEDAY lines: promote
//     each to TODO or GOAL, or drop (append-only -- dropping just
//     means no further action, the original line is untouched).
//  3. Explicit IMPACT/MILESTONE prompts for the past month.
//  4. Log/review GOAL lines for the new month.
func showSOMWindow(a fyne.App) {
	now := time.Now()
	from, to := priorMonthRange(now)

	w := a.NewWindow("Dunzo: Start of Month")

	// Step 1: digest
	digestLabel := widget.NewLabel("Generating prior month's digest, please wait...")
	digestLabel.Wrapping = fyne.TextWrapWord
	go func() {
		ledgerText := gatherLedgerTextForRange(from, to, nil)
		var summary string
		if strings.TrimSpace(ledgerText) == "" {
			summary = "(no ledger entries found for last month)"
		} else {
			s, err := summarizeWithCopilot(ledgerText)
			if err != nil {
				summary = "Error running gh copilot:\n" + err.Error()
			} else {
				summary = s
			}
		}
		fyne.Do(func() {
			digestLabel.SetText(summary)
		})
	}()

	// Step 2: IDEA/SOMEDAY triage
	items := gatherIdeaSomedayItems(from, to)
	triageBox := container.NewVBox()
	if len(items) == 0 {
		triageBox.Add(widget.NewLabel("No open IDEA/SOMEDAY items from last month."))
	}
	handled := make([]bool, len(items))
	for i, item := range items {
		i, item := i, item // capture
		row := widget.NewLabel(item.Category + ": " + item.Text)
		promoteTodoBtn := widget.NewButton("-> TODO", func() {
			recordActivity(item.Text, "TODO")
			handled[i] = true
			row.SetText("[promoted to TODO] " + item.Text)
		})
		promoteGoalBtn := widget.NewButton("-> GOAL", func() {
			recordActivity(item.Text, "GOAL")
			handled[i] = true
			row.SetText("[promoted to GOAL] " + item.Text)
		})
		dropBtn := widget.NewButton("Drop", func() {
			handled[i] = true
			row.SetText("[dropped] " + item.Text)
		})
		triageBox.Add(container.NewBorder(nil, nil, nil,
			container.NewHBox(promoteTodoBtn, promoteGoalBtn, dropBtn), row))
	}

	// Step 3: IMPACT/MILESTONE prompts
	impactEntry := widget.NewMultiLineEntry()
	impactEntry.SetPlaceHolder("Any IMPACT items this month? One per line...")
	impactEntry.SetMinRowsVisible(2)
	milestoneEntry := widget.NewMultiLineEntry()
	milestoneEntry.SetPlaceHolder("Any MILESTONE items this month? One per line...")
	milestoneEntry.SetMinRowsVisible(2)

	// Step 4: new month's GOALs
	var currentGoals []OpenItem
	for _, item := range getOpenItems() {
		if item.Category == "GOAL" {
			currentGoals = append(currentGoals, item)
		}
	}
	currentGoalsBox := container.NewVBox()
	if len(currentGoals) == 0 {
		currentGoalsBox.Add(widget.NewLabel("(no current GOALs logged yet)"))
	}
	for _, item := range currentGoals {
		currentGoalsBox.Add(widget.NewLabel("- " + item.Text))
	}
	newGoalsEntry := widget.NewMultiLineEntry()
	newGoalsEntry.SetPlaceHolder("New/updated GOALs for this month? One per line...")
	newGoalsEntry.SetMinRowsVisible(2)

	// Step 5: monthly recurring items, surfaced as a month checklist
	// (see RECURRING-ITEMS-DESIGN-SEED.md) -- each due monthly item is
	// a suggestion the user explicitly taps "Add" for, not auto-seeded.
	recurringBox := container.NewVBox()
	dueMonthly := dueRecurringItems(LoadConfig(), now, "monthly")
	if box := recurringItemsSuggestionBox(dueMonthly, nil); box != nil {
		recurringBox.Add(box)
	} else {
		recurringBox.Add(widget.NewLabel("(no monthly recurring items due)"))
	}

	finishBtn := widget.NewButton("Commence "+now.Month().String(), func() {
		for _, line := range strings.Split(impactEntry.Text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				recordActivity(line, "IMPACT")
			}
		}
		for _, line := range strings.Split(milestoneEntry.Text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				recordActivity(line, "MILESTONE")
			}
		}
		for _, line := range strings.Split(newGoalsEntry.Text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				recordActivity(line, "GOAL")
			}
		}
		w.Close()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("1. Last Month's Digest", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		digestLabel,
		widget.NewLabelWithStyle("2. Review IDEA/SOMEDAY Items", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		triageBox,
		widget.NewLabelWithStyle("3. IMPACT / MILESTONE This Month", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		impactEntry,
		milestoneEntry,
		widget.NewLabelWithStyle("4. This Month's GOALs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		currentGoalsBox,
		newGoalsEntry,
		widget.NewLabelWithStyle("5. Monthly Recurring Items", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		recurringBox,
		finishBtn,
	)

	w.SetContent(container.NewVScroll(content))
	w.Resize(fyne.NewSize(520, 600))
	w.Show()
}
