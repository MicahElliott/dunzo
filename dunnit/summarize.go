package dun

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// summaryPeriod identifies how far back to gather ledger entries for a
// summary.
type summaryPeriod string

const (
	periodDay     summaryPeriod = "Day"
	periodMonth   summaryPeriod = "Month"
	periodQuarter summaryPeriod = "Quarter"
)

// ledgerFilesFor returns the paths of ledger files under DunzoDir()
// whose date falls within the given period, relative to now.
func ledgerFilesFor(period summaryPeriod, now time.Time) []string {
	var cutoff time.Time
	switch period {
	case periodDay:
		cutoff = now.AddDate(0, 0, -1)
	case periodMonth:
		cutoff = now.AddDate(0, -1, 0)
	case periodQuarter:
		cutoff = now.AddDate(0, -3, 0)
	default:
		cutoff = now.AddDate(0, 0, -1)
	}

	var files []string
	for _, path := range allLedgerFiles() {
		datePart := ledgerFileDate(path)
		if datePart == nil {
			continue
		}
		if !datePart.Before(cutoff) && !datePart.After(now) {
			files = append(files, path)
		}
	}
	return files
}

// allLedgerFiles returns the paths of every ledger-*.txt file under
// DunzoDir(), regardless of date, in filesystem walk order.
func allLedgerFiles() []string {
	var files []string
	root := DunzoDir()
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasPrefix(name, "ledger-") || !strings.HasSuffix(name, ".txt") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

// ledgerFileDate parses the YYYYMMDD date out of a ledger file's
// name (e.g. ".../ledger-20260828.txt"), returning nil if it doesn't
// match the expected naming pattern.
func ledgerFileDate(path string) *time.Time {
	name := filepath.Base(path)
	datePart := strings.TrimSuffix(strings.TrimPrefix(name, "ledger-"), ".txt")
	t, err := time.ParseInLocation("20060102", datePart, time.Local)
	if err != nil {
		return nil
	}
	return &t
}

// gatherLedgerText concatenates the content of all ledger files for the
// given period into one string (file path headers included, for
// context).
func gatherLedgerText(period summaryPeriod) string {
	files := ledgerFilesFor(period, time.Now())
	var sb strings.Builder
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sb.WriteString("# " + filepath.Base(path) + "\n")
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			sb.WriteString(scanner.Text())
			sb.WriteString("\n")
		}
		f.Close()
	}
	return sb.String()
}

// summarizeWithCopilot shells out to `gh copilot` with a one-shot prompt
// asking it to summarize the given ledger text into a brief impact
// report. Requires the `gh` CLI with the Copilot extension available.
func summarizeWithCopilot(ledgerText string) (string, error) {
	prompt := "Summarize this ledger of daily activity entries into a brief " +
		"impact report suitable for a standup or status update. Be concise " +
		"and group related work together.\n\n" + ledgerText

	cmd := exec.Command("gh", "copilot", "-p", prompt, "--silent", "--allow-all-tools")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh copilot failed: %w\n%s", err, out)
	}
	return string(out), nil
}

// showSummarizeDialog lets the user pick a period, then runs the
// summary and displays the result in a new window.
func showSummarizeDialog(a fyne.App, parent fyne.Window) {
	options := []string{string(periodDay), string(periodMonth), string(periodQuarter)}
	periodSelect := widget.NewSelect(options, nil)
	periodSelect.SetSelected(string(periodDay))

	d := dialog.NewCustomConfirm("Summarize", "Generate", "Cancel",
		container.NewVBox(
			widget.NewLabel("Summarize accomplishments for:"),
			periodSelect,
		),
		func(ok bool) {
			if !ok {
				return
			}
			period := summaryPeriod(periodSelect.Selected)
			runSummarize(a, period)
		}, parent)
	d.Show()
}

func runSummarize(a fyne.App, period summaryPeriod) {
	ledgerText := gatherLedgerText(period)
	if strings.TrimSpace(ledgerText) == "" {
		w := a.NewWindow("Dunzo: Summary")
		w.SetContent(widget.NewLabel("No ledger entries found for that period."))
		w.Show()
		return
	}

	progress := a.NewWindow("Dunzo: Summarizing...")
	progress.SetContent(widget.NewLabel(
		"Asking gh copilot to summarize, please wait...\n" +
			"The generated report will be copied to your clipboard automatically."))
	progress.Show()

	go func() {
		summary, err := summarizeWithCopilot(ledgerText)
		fyne.Do(func() {
			progress.Close()
			w := a.NewWindow(fmt.Sprintf("Dunzo: %s Summary", period))
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
