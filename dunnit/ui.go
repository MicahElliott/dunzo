package dun

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	// "fyne.io/fyne/v2/layout"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
	"bufio"
)

// Get today's ledger file path and name.
func getLedger() (string, string) {
	yr, wk := time.Now().ISOWeek()
	// mo := time.Now().Month()
	t := time.Now().UTC()
	tn := time.Now()
	yr8 := tn.Format("20060102")
	moname := t.Format("Jan")
	fname0 := "ledger-"+yr8+".txt"
	fpath := filepath.Join(DunzoDir(),
		strconv.Itoa(yr), "w"+strconv.Itoa(wk)+"-"+moname)
	fname := filepath.Join(fpath, fname0)
	return fpath, fname
}

func recordActivity(text, category string) {
	log.Println("Content was:", text)
	fpath, fname := getLedger()
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		log.Println("Making new dir:", fpath)
		os.MkdirAll(fpath, os.ModePerm) }
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Error opening ledger:", err)
		return
	}
	stamp := time.Now().Format("[15:04:05]")
	outstr := stamp + " " + category + " " + text + "\n"
	f.WriteString(outstr)
	f.Close()
}

// readLedgerLines returns all lines from today's ledger file (empty if
// none exist yet).
func readLedgerLines() []string {
	_, fname := getLedger()
	f, err := os.Open(fname)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// lastEntryText returns the free-text portion (after timestamp+category)
// of the most recent ledger line, or "" if there isn't one.
func lastEntryText() string {
	lines := readLedgerLines()
	if len(lines) == 0 {
		return ""
	}
	last := lines[len(lines)-1]
	parts := strings.SplitN(last, " ", 3)
	if len(parts) < 3 {
		return last
	}
	return parts[2]
}

// openInEditor opens the given file with $EDITOR, falling back to the
// OS default opener (macOS `open`, Linux `xdg-open`). $EDITOR may
// include flags (e.g. "emacsclient -c"), so it's split on whitespace.
func openInEditor(path string) {
	if editor := os.Getenv("EDITOR"); editor != "" {
		fields := strings.Fields(editor)
		args := append(fields[1:], path)
		cmd := exec.Command(fields[0], args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Println("Error launching $EDITOR:", err)
		}
		return
	}
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	if err := exec.Command(opener, path).Start(); err != nil {
		log.Println("Error opening file:", err)
	}
}

func MakeUI() *fyne.App {
	a := app.New()
	return &a
}

func StartUI(a fyne.App) {
	a.Settings().SetTheme(theme.LightTheme())

	w4 := a.NewWindow("Dunzo")
	// label1 := widget.NewLabel("Label 1")
	// value1 := widget.NewLabel("Value")
	// label2 := widget.NewLabel("Label 2")
	// value2 := widget.NewLabel("Something")

	green := color.NRGBA{R: 0, G: 180, B: 0, A: 255}
	colorText := canvas.NewText("I colored this", green)
	lastDunnitLabel := widget.NewLabel("Last Dunnit: " + getLastDunnit())
	commonTopics := getCommonTopics()

	// TODO show day's GOALs

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter text...")
	// input.Resize(fyne.NewSize(100.0, 50.0))

	selectedCat := "DONE"
	// widget.NewSelectEntry
	category := widget.NewSelect([]string{
		// "✔️ DONE", "🎯 GOAL", "📅 MTG", "🚫 BLKR", "💥 IMPACT", "📈 PRDTY", "🔚 SMRY" },
		"✔️ DONE", "🎯 GOAL", "📅 MEETING", "🚫 BLOCKED", "⛔❌ FAIL",
		"💥 IMPACT", "📈 PRODUCTIVITY", "🔚 SUMMARY",
		"🧠 TIL", "💡 IDEA", "🏆 WIN", "💼 CAREER (rare)", "🗑️ WASTED", "🏁 MILESTONE" },
		// or "rock" 🪨 for "milestone"
		func(cat string) { fmt.Println("saw a category:", cat)
			res := strings.Split(cat, " ")
			// selectedCat = cat
			selectedCat = res[1]
		})
	category.SetSelected("✔️ DONE") // default to DONE

	doneWrapper := container.NewBorder(nil, nil, category, nil, input)

	fmt.Println(input.MinSize())

	saveEntry := func() {
		if strings.TrimSpace(input.Text) == "" {
			return
		}
		recordActivity(input.Text, selectedCat) // TODO trim emoji off front, and shorten to 4-char code
		input.SetText("")
		lastDunnitLabel.SetText("Last Dunnit: " + getLastDunnit())
	}
	input.OnSubmitted = func(string) { saveEntry() }

	buttons := container.NewHBox(
		widget.NewButton("Save", saveEntry),
		widget.NewButton("Ditto", func() {
			if txt := lastEntryText(); txt != "" {
				input.SetText(txt)
			}
		}),
		widget.NewButton("Show Dunnits", func() {
			w3 := a.NewWindow("Dunzo: Today")
			w3.SetContent(widget.NewLabel(strings.Join(readLedgerLines(), "\n")))
			w3.Resize(fyne.NewSize(500, 400))
			w3.Show()
		}),
		widget.NewButton("Edit Dunnits", func() {
			_, fname := getLedger()
			openInEditor(fname)
		}),
		widget.NewButton("Show Goals", func() {
			goals := getGoals()
			if len(goals) == 0 {
				dialog.ShowInformation("No Goals Yet",
					"You haven't set any goals today.\nRecord one with the GOAL category to get started!",
					w4)
				return
			}
			w3 := a.NewWindow("Dunzo: Goals")
			w3.SetContent(widget.NewLabel(strings.Join(goals, "\n")))
			w3.Show()

		} ),
	)

	content := container.NewVBox(
		// widget.NewLabel(colorText),
		widget.NewLabel("Common topics you use:" + commonTopics),
		colorText,
		lastDunnitLabel,
		widget.NewLabel("What would you like to record?"),
		doneWrapper,
		// category, input,
		buttons,
	)
	log.Println(content)

	// grid := container.New(layout.NewFormLayout(),
	// 	label1, value1, label2, value2, content)

	w4.SetContent(content)
	w4.SetCloseIntercept(func() { w4.Hide() })
	w4.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName: fyne.KeyEscape,
	}, func(fyne.Shortcut) { w4.Hide() })

	// Menu
	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("Dunzo",
			fyne.NewMenuItem("Show", func() { w4.Show() }),
			fyne.NewMenuItem("Settings...", func() { showSettings(a) }),
		)
		desk.SetSystemTrayMenu(m)
	}

	w4.Show()

	a.Run()

	tidyUp()
}

func getGoals() []string {
	_, lgr := getLedger()
	fmt.Println("[FAKE] show today's dunnits")
	fmt.Println(lgr)
	goals := []string{}
	f, _ := os.Open(lgr)
	defer f.Close()
	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)
	for scanner.Scan() { // https://golang.org/pkg/bufio/#Scanner.Scan
		if ln := scanner.Text(); strings.Contains(ln, " GOAL ") { goals = append(goals, ln) }
	}
	fmt.Println(goals)
	return goals
}

// TODO Look over last month's most-used tags. May need to support a list of non-work topics
func getCommonTopics() string {
	return "#personal #ticketno #dunnit #interview #lob #emacs #pts:3"
}

func getLastDunnit() string {
	if txt := lastEntryText(); txt != "" {
		return txt
	}
	return "(nothing recorded yet today)"
}

func tidyUp() { fmt.Println("cleaning shit up") }

func updateTime(clock *widget.Label) {
	formatted := time.Now().Format("Dunnit: 03:04:05")
	clock.SetText(formatted)
}
