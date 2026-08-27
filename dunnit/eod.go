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

// showEODWindow recreates (in spirit) dunnit.zsh's dunnit-eod sequence:
// a short daily wrap-up capturing a summary, a productivity score, a
// sentiment rating, and goals for tomorrow. Rather than a chain of
// separate popups (as the original zsh alerter-based flow did), this
// is one window with all the questions -- simpler to implement and
// to answer.
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

	form := widget.NewForm(
		widget.NewFormItem("Summary", summary),
		widget.NewFormItem("Productivity (1-5)", productivity),
		widget.NewFormItem("Sentiment", sentiment),
		widget.NewFormItem("Tomorrow's Goals", goals),
	)
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
		w.Close()
	}

	w.SetContent(container.NewVScroll(form))
	w.Resize(fyne.NewSize(420, 400))
	w.Show()
}

// recordTomorrowGoals appends each non-blank line as a GOAL entry to
// tomorrow's ledger file (so it's ready to go first thing).
func recordTomorrowGoals(lines []string) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	yr, wk := tomorrow.ISOWeek()
	moname := tomorrow.Format("Jan")
	fname0 := "ledger-" + tomorrow.Format("20060102") + ".txt"
	fpath := filepath.Join(DunzoDir(), strconv.Itoa(yr), "w"+strconv.Itoa(wk)+"-"+moname)
	fname := filepath.Join(fpath, fname0)

	if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
		return
	}
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f.WriteString("[05:00] GOAL " + line + "\n")
	}
}
