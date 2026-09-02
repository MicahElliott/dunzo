package dun

import (
	"bytes"
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yuin/goldmark"
)

// markdownToHTML renders md as a minimal standalone HTML document via
// goldmark (already an indirect Fyne dependency, promoted here to a
// direct import -- no new dependency actually added). Used by
// showEditableReportWindow's "Copy as HTML" action. Returns md
// unchanged (wrapped in <pre>) if conversion fails, rather than
// erroring out the whole Copy action.
func markdownToHTML(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		log.Println("Error converting markdown to HTML:", err)
		return "<pre>" + md + "</pre>"
	}
	return "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"></head><body>\n" +
		buf.String() + "\n</body></html>"
}

// showEditableReportWindow displays a generated report in an editable
// Markdown text window (not the read-only showGeneratedReport from
// report.go -- Reviews are meant to be tweakable before saving, per
// docs/kickoff-review-design.md's Review model) with a live Markdown
// preview below, and Save/Copy as HTML/Close actions. Save writes the
// *current edited text* (not the original draft) to savePath.
func showEditableReportWindow(a fyne.App, title, savePath, initialText string) {
	w := a.NewWindow(title)

	editor := widget.NewMultiLineEntry()
	editor.SetText(initialText)
	editor.Wrapping = fyne.TextWrapWord

	preview := widget.NewRichTextFromMarkdown(initialText)
	preview.Wrapping = fyne.TextWrapWord
	editor.OnChanged = func(text string) {
		preview.ParseMarkdown(text)
	}

	editorScroll := container.NewVScroll(editor)
	editorScroll.SetMinSize(fyne.NewSize(0, 260))
	previewScroll := container.NewVScroll(preview)
	previewScroll.SetMinSize(fyne.NewSize(0, 260))

	saveBtn := widget.NewButtonWithIcon("Save", theme.Icon(theme.IconNameDocumentSave), func() {
		if err := os.WriteFile(savePath, []byte(editor.Text), 0644); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Saved", "Saved to "+savePath, w)
	})
	copyHTMLBtn := widget.NewButtonWithIcon("Copy as HTML", theme.Icon(theme.IconNameContentCopy), func() {
		a.Clipboard().SetContent(markdownToHTML(editor.Text))
	})
	closeBtn := widget.NewButton("Close", func() { w.Close() })

	content := container.NewBorder(
		widget.NewLabelWithStyle("Edit (Markdown):", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		container.NewVBox(
			widget.NewLabelWithStyle("Preview:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			previewScroll,
			container.NewHBox(saveBtn, copyHTMLBtn, closeBtn),
		),
		nil, nil,
		editorScroll,
	)
	w.SetContent(content)
	w.Resize(fyne.NewSize(640, 720))
	w.Show()
}

// showWeekReviewWindow shows the Week Review (docs/kickoff-review-
// design.md): mostly-automatic, generated on-demand (not eagerly on
// open -- the user picks a theme first, then taps Generate) via
// generateThemedReview (period.go), saved via reviewReportPath's
// "review-week" convention so a later Month Review can roll it up.
// Also offers TODO/QUESTION carry-forward checkboxes, applied
// unconditionally on Close/Done regardless of whether a report was
// ever generated or saved -- same never-lose-data guarantee as EOD.
func showWeekReviewWindow(a fyne.App) {
	now := time.Now()
	cfg := LoadConfig()
	w := a.NewWindow("Dunzo: Week Review (" + periodLabel(cfg, periodWeek, now) + ")")

	savePath := reviewReportPath(periodWeek, now)

	themeSelect := widget.NewSelect(themeOptions(), nil)
	themeSelect.SetSelected(themeDisplayNames[themeFor(cfg, periodWeek)])

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
			overrideCfg.ThemeWeek = selectedTheme
			summary, err := generateThemedReview(overrideCfg, periodWeek, now)
			fyne.Do(func() {
				generateBtn.Enable()
				if err != nil {
					log.Println("Error generating Week Review:", err)
					statusLabel.SetText("Error generating report \u2014 see logs.")
					dialog.ShowError(err, w)
					return
				}
				statusLabel.SetText("Generated.")
				showEditableReportWindow(a,
					"Dunzo: Week Review Report ("+periodLabel(cfg, periodWeek, now)+")",
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
		widget.NewLabel("Week Review: "+periodLabel(cfg, periodWeek, now)),
		container.NewBorder(nil, nil, widget.NewLabel("Theme:"), generateBtn, themeSelect),
		statusLabel,
		carryForwardBox,
		doneBtn,
	)

	w.SetContent(container.NewVScroll(content))
	w.Resize(fyne.NewSize(520, 420))
	w.Show()
}
