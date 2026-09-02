package dun

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// showPeriodReviewWindow shows a generic Review dialog (docs/kickoff-
// review-design.md) for period: mostly-automatic, generated on-demand
// (the user picks a theme first, then taps Generate) via
// generateThemedReview (period.go), saved via reviewReportPath's
// convention so a larger unit's Review can roll it up. Also offers
// TODO/QUESTION carry-forward checkboxes, applied unconditionally on
// Done regardless of whether a report was ever generated or saved --
// same never-lose-data guarantee as EOD. Used for Week, Quarter, and
// Year, which have no bespoke dialog of their own (unlike Day/Month's
// existing EOD/SOM).
func showPeriodReviewWindow(a fyne.App, period summaryPeriod) {
	now := time.Now()
	cfg := LoadConfig()
	label := periodLabel(cfg, period, now)
	w := a.NewWindow("Dunzo: " + string(period) + " Review (" + label + ")")

	savePath := reviewReportPath(period, now)

	themeSelect := widget.NewSelect(themeOptions(), nil)
	themeSelect.SetSelected(themeDisplayNames[themeFor(cfg, period)])

	statusLabel := widget.NewLabel("Pick a theme, then tap Generate.")

	generateBtn := widget.NewButton("Generate", nil)
	generateBtn.OnTapped = func() {
		selectedTheme := themeFromDisplayName(themeSelect.Selected)
		if selectedTheme == "" {
			return
		}
		generateBtn.Disable()
		statusLabel.SetText("Generating, please wait\u2026")
		go func() {
			overrideCfg := cfg
			setTheme(&overrideCfg, period, selectedTheme)
			summary, err := generateThemedReview(overrideCfg, period, now)
			fyne.Do(func() {
				generateBtn.Enable()
				if err != nil {
					log.Println("Error generating "+string(period)+" Review:", err)
					statusLabel.SetText("Error generating report \u2014 see logs.")
					dialog.ShowError(err, w)
					return
				}
				statusLabel.SetText("Generated.")
				showEditableReportWindow(a,
					"Dunzo: "+string(period)+" Review Report ("+label+")",
					savePath, summary)
			})
		}()
	}

	// Carry-forward: same never-lose-data guarantee as EOD (defaults
	// checked). Applied on Done regardless of whether a report was
	// ever generated -- generating/saving a report and carrying
	// forward open items are independent actions.
	todoBox, openTodos, todoChecks := eodOpenItemsSection("TODO")
	questionBox, openQuestions, questionChecks := eodOpenItemsSection("QUESTION")
	carryForwardBox := container.NewVBox()
	if len(openTodos) > 0 {
		carryForwardBox.Add(widget.NewLabelWithStyle("Carry Forward Open TODOs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		carryForwardBox.Add(todoBox)
	}
	if len(openQuestions) > 0 {
		carryForwardBox.Add(widget.NewLabelWithStyle("Carry Forward Open QUESTIONs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		carryForwardBox.Add(questionBox)
	}

	doneBtn := widget.NewButton("Done", func() {
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
		w.Close()
	})

	content := container.NewVBox(
		widget.NewLabel(string(period)+" Review: "+label),
		container.NewBorder(nil, nil, widget.NewLabel("Theme:"), generateBtn, themeSelect),
		statusLabel,
		carryForwardBox,
		doneBtn,
	)

	w.SetContent(container.NewVScroll(content))
	w.Resize(fyne.NewSize(520, 420))
	w.Show()
}
