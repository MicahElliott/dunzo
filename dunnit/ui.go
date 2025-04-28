package dun

import (
	"time"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	// "fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func recordActivity(text, category string) {
	log.Println("Content was:", text)
	os.Stderr.WriteString(text)

	home, _ := os.UserHomeDir()
	yr, wk := time.Now().ISOWeek()
	// mo := time.Now().Month()
	t := time.Now().UTC()
	tn := time.Now()
	yr8 := tn.Format("20060102")
	moname := t.Format("Jan")
	fname0 := "ledger-"+yr8+".txt"
	fpath := filepath.Join(home, ".dunnit", "mydunnits",
		strconv.Itoa(yr), "w"+strconv.Itoa(wk)+"-"+moname)
	fname := filepath.Join(fpath, fname0)
	// os.MkdirAll(fpath, os.ModePerm)
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		log.Println("Making new dir:", fpath)
		os.MkdirAll(fpath, os.ModePerm)
	}
	fmt.Println(fpath)
	fmt.Println(fname)

	// f, err := os.OpenFile("/tmp/foo", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	log.Println(err)
	hm := time.Now().Format("[15:04]")
	// outstr := hm + " DONE " + text + "\n"
	outstr := hm + " " + category + " " + text + "\n"
	fmt.Println(outstr)
	f.WriteString(outstr)
	f.Close()
}

func MakeUI() *fyne.App {
	a := app.New()
	return &a
}

func StartUI(a fyne.App) {
	// a := app.New()

	//// New windows
	w := a.NewWindow("Dunnit: Hello World")
	a.Settings().SetTheme(theme.LightTheme())

	w2 := a.NewWindow("Dunnit: Larger")
	w2.SetContent(widget.NewLabel("More content"))
	w2.SetContent(widget.NewButton("Open new", func() {
		w3 := a.NewWindow("Dunnit: Third")
		w3.SetContent(widget.NewLabel("Third"))
		w3.Show()
	}))
	w2.Resize(fyne.NewSize(500, 300))
	w2.Show()

	w4 := a.NewWindow("Dunzo")
	// label1 := widget.NewLabel("Label 1")
	// value1 := widget.NewLabel("Value")
	// label2 := widget.NewLabel("Label 2")
	// value2 := widget.NewLabel("Something")

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter text...")
	// widget.Entry{Se}
	// defaultCat := "DONE"
	selectedCat := "FOOBAR"
	// widget.NewSelectEntry
	category := widget.NewSelect([]string{
		// "✔️ DONE", "🎯 GOAL", "📅 MTG", "🚫 BLKR", "💥 IMPACT", "📈 PRDTY", "🔚 SMRY" },
		"✔️ DONE", "🎯 GOAL", "📅 MEETING", "🚫 BLOCKED", "💥 IMPACT", "📈 PRODUCTIVITY", "🔚 SUMMARY" },
		func(cat string) { fmt.Println("saw a category:", cat)
			selectedCat = cat
		})
	category.SetSelected("DONE") // default to DONE
	content := container.NewVBox(
		widget.NewLabel("What would you like to record?"),
		category,
		input,
		widget.NewButton("Save", func() {
			recordActivity(input.Text, selectedCat) // TODO trim emoji off front, and shorten to 4-char code
			// clear form
			input.SetPlaceHolder("Enter text...") // FIXME not working
			// input.OnSubmitted: func() {input.SetPlaceHolder("Cleared on-submitted")}
		}),
		// label1, value1, label2, value2
	)

	// grid := container.New(layout.NewFormLayout(),
	// 	label1, value1, label2, value2, content)

	// w4.SetContent(grid)
	w4.SetContent(content)
	w4.Show()
	// w4.Sh


	// Notification
	a.SendNotification( fyne.NewNotification("Something happened", "Maybe this is serious.") )

	// Menu
	if desk, ok := a.(desktop.App); ok {
		h := fyne.NewMenuItem("Hello", func() {})
		h.Icon = theme.HomeIcon()
		m := fyne.NewMenu("Dunnit",
			fyne.NewMenuItem("Show", func() { w.Show() }),
			h,
		)
		// desk.SetSystemTrayIcon("icon.png")
		desk.SetSystemTrayMenu(m)
	}

	fmt.Println("hi there")
	// w.SetContent(widget.NewLabel("Hello World!"))
	// clock := widget.NewLabel(quote.Go())
	clock := widget.NewLabel("my cluck")
	// formatted := time.Now()
	w.SetContent(clock)

	w.SetCloseIntercept(func() { w.Hide() })

	// Dynamic clock counter
	// go func() { for range time.Tick(time.Second) { updateTime(clock) } }()
	w.Show()

	a.Run()

	a.SendNotification( fyne.NewNotification("Something else", "After Run") )

	tidyUp()
}

func tidyUp() { fmt.Println("cleaning shit up") }

func updateTime(clock *widget.Label) {
	formatted := time.Now().Format("Dunnit: 03:04:05")
	clock.SetText(formatted)
}
