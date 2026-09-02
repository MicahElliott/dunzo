package dun

import (
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// showWeekReviewWindow shows the Week Review (docs/kickoff-review-
// design.md): mostly-automatic, generated via generateThemedReview
// (period.go) using cfg's ThemeWeek default (with a per-instance
// theme override dropdown), saved via periodReportPath's "review-
// week" convention (review.go's reviewReportPath) so a later Month
// Review can roll it up. Also offers TODO/QUESTION carry-forward
// checkboxes, same never-lose-data guarantee as EOD -- carry-forward
// applies regardless of whether the user engages with the generated
// digest at all.
func showWeekReviewWindow(a fyne.App) {
	now := time.Now()
	w := a.NewWindow("Dunzo: Week Review (" + periodLabel(periodWeek, now) + ")")

	digestBody := widget.NewRichTextFromMarkdown("*Generating, please wait...*")
	digestBody.Wrapping = fyne.TextWrapWord
	digestScroll := container.NewVScroll(digestBody)
	digestScroll.SetMinSize(fyne.NewSize(0, 260))

	digestText := ""
	savePath := reviewReportPath(periodWeek, now)
	cfg := LoadConfig()
	themeSelect := widget.NewSelect(
		[]string{ThemePersonalNotes, ThemeStatusReport, ThemeFormalReport, ThemeBragPreso},
		nil)
	themeSelect.SetSelected(themeFor(cfg, periodWeek))

	var generateDigest func()
	generateDigest = func() {
		digestBody.ParseMarkdown("*Generating, please wait...*")
		go func() {
			overrideCfg := cfg
			switch themeSelect.Selected {
			case ThemePersonalNotes, ThemeStatusReport, ThemeFormalReport, ThemeBragPreso:
				overrideCfg.ThemeWeek = themeSelect.Selected
			}
			summary, err := generateThemedReview(overrideCfg, periodWeek, now)
			if err != nil {
				log.Println("Error generating Week Review:", err)
				summary = "Error running gh copilot:\n" + err.Error()
			}
			digestText = summary
			fyne.Do(func() {
				digestBody.ParseMarkdown(summary)
			})
		}()
	}
	themeSelect.OnChanged = func(string) { generateDigest() }
	go generateDigest()

	copyBtn := widget.NewButtonWithIcon("Copy", theme.Icon(theme.IconNameContentCopy), func() {
		a.Clipboard().SetContent(digestText)
	})
	saveBtn := widget.NewButtonWithIcon("Save", theme.Icon(theme.IconNameDocumentSave), func() {
		if err := os.WriteFile(savePath, []byte(digestText), 0644); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Saved", "Saved to "+savePath, w)
	})

	// Carry-forward: same never-lose-data guarantee as EOD (defaults
	// checked, applies regardless of Save/Coolio choice below).
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

	applyCarryForward := func() {
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
	}

	// Save: persists the digest to disk. Coolio: declines saving the
	// digest (still just generated as FYI, nothing lost), but either
	// way carry-forward is applied -- never bundled into "declining
	// entry" (see EOD's Save/Coolio precedent in
	// docs/kickoff-review-design.md).
	saveAndCloseBtn := widget.NewButton("Save & Close", func() {
		if err := os.WriteFile(savePath, []byte(digestText), 0644); err != nil {
			dialog.ShowError(err, w)
			return
		}
		applyCarryForward()
		w.Close()
	})
	coolioBtn := widget.NewButton("Coolio (skip saving)", func() {
		applyCarryForward()
		w.Close()
	})

	content := container.NewVBox(
		widget.NewLabel("Week Review: "+periodLabel(periodWeek, now)),
		container.NewBorder(nil, nil, widget.NewLabel("Theme:"), nil, themeSelect),
		digestScroll,
		container.NewHBox(copyBtn, saveBtn),
		carryForwardBox,
		container.NewHBox(saveAndCloseBtn, coolioBtn),
	)

	w.SetContent(container.NewVScroll(content))
	w.Resize(fyne.NewSize(560, 600))
	w.Show()
}
