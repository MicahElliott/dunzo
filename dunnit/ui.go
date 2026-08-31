package dun

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"fyne.io/fyne/v2/driver/desktop"
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

// ledgerDirFor returns the year/week/month directory (same scheme as
// getLedger) for the given ISO year/week and month abbreviation --
// factored out so other date-scoped files (e.g. FR-18's daily
// summary docs) can share the exact same path layout as ledgers.
func ledgerDirFor(yr, wk int, moname string) string {
	return filepath.Join(DunzoDir(), strconv.Itoa(yr), "w"+strconv.Itoa(wk)+"-"+moname)
}

// lastActivityAt tracks the wall-clock time of the most recent
// recordActivity() call, so the scheduler (sched.go) can suppress a
// periodic nudge if the user already logged something recently (see
// FR-01). Zero value means "nothing recorded since process start".
var lastActivityAt time.Time

// LastActivityAt returns the time of the most recent recorded entry
// (zero time if none yet this run).
func LastActivityAt() time.Time {
	return lastActivityAt
}

// snoozedUntil tracks a "not now, remind me later" request (FR-26) --
// while non-zero and in the future, the periodic capture nudge
// (sched.go) skips firing. Doesn't affect other nudges (SOD/EOD/
// meeting/etc), only the recurring "what are you working on?" one.
var snoozedUntil time.Time

// Snooze suppresses the next periodic capture nudge(s) until now+d
// (FR-26).
func Snooze(d time.Duration) {
	snoozedUntil = time.Now().Add(d)
}

// SnoozedUntil returns the current snooze expiry (zero if not
// snoozed / already expired).
func SnoozedUntil() time.Time {
	if time.Now().After(snoozedUntil) {
		return time.Time{}
	}
	return snoozedUntil
}

// IsDoNotDisturb reports whether the user has manually toggled Do Not
// Disturb on (FR-27) -- while true, the periodic capture nudge is
// suppressed entirely, same scope as Snooze.
func IsDoNotDisturb() bool {
	return LoadConfig().DoNotDisturb
}

// SetDoNotDisturb persists the Do Not Disturb flag (FR-27) to
// config.toml so it survives app restarts.
func SetDoNotDisturb(on bool) {
	cfg := LoadConfig()
	cfg.DoNotDisturb = on
	if err := writeConfig(cfg); err != nil {
		log.Println("Error saving Do Not Disturb setting:", err)
	}
}

// defaultSnoozeDuration returns the configured default snooze
// duration (cfg.SnoozeMinutes), falling back to 15 minutes if unset/
// invalid.
func defaultSnoozeDuration() time.Duration {
	minutes := LoadConfig().SnoozeMinutes
	if minutes <= 0 {
		minutes = 15
	}
	return time.Duration(minutes) * time.Minute
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
	lastActivityAt = time.Now()
	if len(extractTags(text)) > 0 {
		InvalidateTagCache()
	}
}

// readLedgerLines returns all lines from today's ledger file (empty if
// none exist yet).
func readLedgerLines() []string {
	_, fname := getLedger()
	return readLedgerLinesFrom(fname)
}

// readLedgerLinesFrom returns all lines from the given ledger file
// path (empty if it doesn't exist). Factored out of readLedgerLines
// so callers needing a specific day's file (e.g. FR-17's standup
// export, which wants the last workday's ledger rather than today's)
// can reuse the same scan logic.
func readLedgerLinesFrom(fname string) []string {
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

// showHelp opens a static window listing every category's emoji/code
// and one-line intended-use description (FR-06), grouped by Now/Plan/
// Reflect with section headers. Text comes directly from Categories
// (categories.go), so it can't drift out of sync with the actual
// picker options. Positive-sentiment categories render bold dark
// green; negative-sentiment ones render dark red. Named "Help" (not
// "Category Legend") in the UI -- broader framing for a window that
// may grow beyond just categories later.
//
// Note: this coloring only applies to the static Help window -- Fyne's
// widget.Select doesn't support per-option rich text/color in its
// dropdown, so the live category picker itself stays plain text.
func showHelp(a fyne.App) {
	darkGreen := color.NRGBA{R: 0, G: 100, B: 0, A: 255}
	darkRed := color.NRGBA{R: 139, G: 0, B: 0, A: 255}

	w := a.NewWindow("Dunzo: Help")
	rows := container.NewVBox()
	lastGroup := ""
	for _, c := range Categories {
		if c.Group != lastGroup {
			header := canvas.NewText(GroupLabel(c.Group), theme.Color(theme.ColorNameForeground))
			header.TextSize = 12
			header.TextStyle = fyne.TextStyle{Bold: true}
			rows.Add(header)
			lastGroup = c.Group
		}
		txt := canvas.NewText(c.Label()+" -- "+c.Help, theme.Color(theme.ColorNameForeground))
		txt.TextSize = 10
		txt.TextStyle = fyne.TextStyle{Monospace: true}
		switch c.Sentiment {
		case "positive":
			txt.Color = darkGreen
			txt.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		case "negative":
			txt.Color = darkRed
		}
		rows.Add(txt)
	}
	scroll := container.NewVScroll(rows)
	scroll.SetMinSize(fyne.NewSize(560, 340))
	w.SetContent(scroll)
	w.Show()
}

func MakeUI() *fyne.App {
	a := app.New()
	return &a
}

// closeShortcutEntry is a widget.Entry that additionally recognizes
// the given close shortcut (Cmd+W/Ctrl+W) even while it has focus.
// Fyne's Entry widget has its own ShortcutHandler and normally
// swallows all TypedShortcut calls when focused, so a shortcut added
// only via Canvas().AddShortcut never reaches the window-level
// handler in that case (FR-02). We check for our specific shortcut
// first and invoke onClose directly; anything else falls through to
// the embedded Entry's normal shortcut handling (cut/copy/paste etc).
type closeShortcutEntry struct {
	widget.Entry
	closeKey fyne.KeyName
	closeMod fyne.KeyModifier
	onClose  func()
}

func newCloseShortcutEntry(closeKey fyne.KeyName, closeMod fyne.KeyModifier, onClose func()) *closeShortcutEntry {
	e := &closeShortcutEntry{closeKey: closeKey, closeMod: closeMod, onClose: onClose}
	e.ExtendBaseWidget(e)
	return e
}

func (e *closeShortcutEntry) TypedShortcut(shortcut fyne.Shortcut) {
	if cs, ok := shortcut.(*desktop.CustomShortcut); ok &&
		cs.KeyName == e.closeKey && cs.Modifier == e.closeMod {
		e.onClose()
		return
	}
	e.Entry.TypedShortcut(shortcut)
}

// BuildMainWindow constructs the main Dunzo entry window and tray menu,
// but does not show it or start the Fyne event loop -- call a.Run()
// yourself after this (see dunnit.go). Returns the window so callers
// (e.g. the scheduler) can Show()/RequestFocus() it later.
func BuildMainWindow(a fyne.App) fyne.Window {
	a.Settings().SetTheme(theme.LightTheme())

	w4 := a.NewWindow("Dunzo: Daybook")
	// label1 := widget.NewLabel("Label 1")
	// value1 := widget.NewLabel("Value")
	// label2 := widget.NewLabel("Label 2")
	// value2 := widget.NewLabel("Something")

	// TODO show day's GOALs

	input := newCloseShortcutEntry(fyne.KeyW, fyne.KeyModifierShortcutDefault, func() { w4.Hide() })
	input.SetPlaceHolder("Enter text...")
	// input.Resize(fyne.NewSize(100.0, 50.0))

	// Tag autocomplete (FR-10): as the user types a "#tag" fragment,
	// show a popup menu of matching previously-used tags (scanned
	// from ledger history, cached -- see tags.go). Selecting an entry
	// replaces the in-progress fragment with the full tag.
	var tagPopup *widget.PopUpMenu
	dismissTagPopup := func() {
		if tagPopup != nil {
			tagPopup.Hide()
			tagPopup = nil
		}
	}
	input.OnChanged = func(text string) {
		dismissTagPopup()
		start, fragment, ok := currentTagFragment(text, input.CursorColumn)
		if !ok || len(fragment) < 2 { // need at least "#" + 1 char
			return
		}
		matches := matchingTags(KnownTags(), fragment[1:])
		if len(matches) == 0 {
			return
		}
		items := make([]*fyne.MenuItem, len(matches))
		for i, tag := range matches {
			tag := tag // capture
			items[i] = fyne.NewMenuItem(tag, func() {
				runes := []rune(text)
				newText := string(runes[:start]) + tag + string(runes[input.CursorColumn:])
				input.SetText(newText)
				input.CursorColumn = start + len([]rune(tag))
				input.Refresh()
				dismissTagPopup()
			})
		}
		canvas := fyne.CurrentApp().Driver().CanvasForObject(input)
		if canvas == nil {
			return
		}
		tagPopup = widget.NewPopUpMenu(fyne.NewMenu("", items...), canvas)
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(input)
		tagPopup.ShowAtPosition(pos.Add(fyne.NewPos(0, input.Size().Height)))
		// PopUpMenu.Show() unconditionally steals keyboard focus via
		// canvas.Focus(p). A synchronous re-focus call right after
		// ShowAtPosition isn't enough -- Show()'s own focus-steal
		// seems to still win. Defer the refocus via fyne.Do so it
		// runs after the current UI update cycle, once the popup's
		// own Show() has finished.
		fyne.Do(func() {
			canvas.Focus(input)
		})
	}

	// minsInput is an optional free-text "minutes spent" field (very
	// informal time tracking). When non-empty and numeric, its value
	// is appended to the recorded text as " @Nm" (e.g. "@20m").
	// Wrapped in a fixed-size container (minsWrapper) so it renders
	// at a comfortable width regardless of its own placeholder-driven
	// MinSize -- stretchRowLayout below treats it as a fixed-width
	// object (like groupFilter/category), giving all remaining space
	// to `input` instead.
	minsInput := widget.NewEntry()
	minsInput.SetPlaceHolder("mins")
	minsWrapper := container.NewGridWrap(fyne.NewSize(64, minsInput.MinSize().Height), minsInput)

	// withMins appends " @Nm" to text if minsInput has a valid
	// positive integer in it; otherwise returns text unchanged.
	withMins := func(text string) string {
		raw := strings.TrimSpace(minsInput.Text)
		if raw == "" {
			return text
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return text
		}
		return text + " @" + raw + "m"
	}

	selectedCat := "DONE"
	// widget.NewSelectEntry
	//
	// Note: Fyne's widget.Select renders its selected-text label via
	// an internal RichText segment that it resets (alignment, color)
	// on every refresh, with no exposed hook to set TextStyle.
	// Monospace -- so unlike the Upcoming list and Category Legend
	// (both of which use canvas.Text/widget.Label directly, which we
	// do control), this live picker can't be made monospace without
	// forking/reimplementing Select's renderer. Not worth that cost
	// for a cosmetic detail; left in the default font.
	category := widget.NewSelect(CategoryLabelsForGroup("now"),
		func(cat string) { fmt.Println("saw a category:", cat)
			res := strings.Split(cat, " ")
			// selectedCat = cat
			selectedCat = res[1]
		})
	category.SetSelected(Categories[0].Label()) // default to DONE

	// groupFilter narrows the category picker to a subset (FR-06
	// follow-up): "now" (default, day-to-day capture), "plan"
	// (future-facing), "reflect" (retrospective/EOD-ish), or "all".
	// Purely a UI convenience over the same Categories list.
	groupFilter := widget.NewSelect([]string{"Now", "Plan", "Reflect", "All"},
		func(g string) {
			category.Options = CategoryLabelsForGroup(strings.ToLower(g))
			category.SetSelected(category.Options[0])
			category.Refresh()
		})
	groupFilter.SetSelected("Now")

	// Dunzo aims to be a mouseless/mouse-optional UI -- keyboard-only
	// operation should always be possible. IMPORTANT: Fyne's focus
	// traversal (Tab/Shift+Tab) follows each container's *Objects
	// slice order*, not visual/layout position. container.NewBorder's
	// constructor appends objects in the fixed order (center content
	// first, then top, bottom, left, right) regardless of which
	// visual position they occupy -- so a NewBorder-based row can
	// easily end up with Tab order that doesn't match what's on
	// screen. stretchRowLayout (stretchrow.go) is used instead: it
	// preserves the exact slice order passed to container.New (so Tab
	// order matches visual left-to-right order) while still letting
	// `input` stretch to fill available width like NewBorder's center
	// content would have.
	doneWrapper := container.New(newStretchRowLayout(input), groupFilter, category, input, minsWrapper)

	fmt.Println(input.MinSize())

	// openItemsBox displays currently-open item lines together under
	// one "Upcoming" heading (FR-07, extended for WAITING/QUESTION/
	// FIXME/RISK), split into per-category sub-sections, each with
	// "Done" (convert to DONE) and "Postpone" (defer to SOMEDAY,
	// keeping the list from growing unbounded) actions.
	// refreshOpenItems rebuilds it from the current ledger contents;
	// called after any save/convert/postpone action so the list stays
	// in sync. Wrapped in a widget.Accordion (itemsAccordion, below)
	// so it can be collapsed out of the way once reviewed.
	openItemsBox := container.NewVBox()
	var refreshOpenItems func()
	var refreshCompleted func() // forward decl -- used inside refreshOpenItems's Done button, defined below
	refreshOpenItems = func() {
		openItemsBox.RemoveAll()
		items := getOpenItems()
		if len(items) == 0 {
			openItemsBox.Add(widget.NewLabel("Nothing open right now."))
			openItemsBox.Refresh()
			return
		}
		addRow := func(item OpenItem) {
			row := container.NewBorder(nil, nil, nil,
				container.NewHBox(
					newHoverButton("❌", "Discard", func() {
						recordDiscarded(item)
						refreshOpenItems()
					}),
					newHoverButton("🐌", "Postpone", func() {
						recordPostponed(item)
						refreshOpenItems()
					}),
					newHoverButton("✔️", "Done", func() {
						recordConvertedDone(item)
						refreshOpenItems()
						refreshCompleted()
					}),
				),
				widget.NewLabel("- "+item.Text))
			openItemsBox.Add(row)
		}
		cats, grouped := groupOpenItemsByCategory(items)
		for _, cat := range cats {
			openItemsBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
			for _, item := range grouped[cat] {
				addRow(item)
			}
		}
		openItemsBox.Refresh()
	}
	refreshOpenItems()

	// completedBox displays today's DONE entries -- since Daybook no
	// longer shows a running list of everything logged today, this
	// restores visibility into today's finished items. Also
	// collapsible, placed right below Upcoming in the same accordion.
	completedBox := container.NewVBox()
	refreshCompleted = func() {
		completedBox.RemoveAll()
		done := getCompletedItems()
		if len(done) == 0 {
			completedBox.Add(widget.NewLabel("Nothing completed yet today."))
			completedBox.Refresh()
			return
		}
		for _, text := range done {
			completedBox.Add(widget.NewLabel("- " + text))
		}
		completedBox.Refresh()
	}
	refreshCompleted()

	// Upcoming/Completed are both collapsible (via widget.Accordion)
	// so either can be tucked out of the way once reviewed, freeing
	// room for the other. Completed listed first (with Upcoming below
	// it) per Micah's preferred ordering.
	upcomingItem := widget.NewAccordionItem("Upcoming", openItemsBox)
	completedItem := widget.NewAccordionItem("Completed", completedBox)
	completedItem.Open = true
	itemsAccordion := widget.NewAccordion(completedItem, upcomingItem)

	saveEntry := func() {
		if strings.TrimSpace(input.Text) == "" {
			return
		}
		recordActivity(withMins(input.Text), selectedCat) // TODO trim emoji off front, and shorten to 4-char code
		input.SetText("")
		minsInput.SetText("")
		refreshOpenItems()
		refreshCompleted()
	}
	input.OnSubmitted = func(string) { saveEntry() }
	minsInput.OnSubmitted = func(string) { saveEntry() }

	buttons := container.NewHBox(
		widget.NewButton("Save", saveEntry),
		widget.NewButton("Ditto", func() {
			// Ditto now logs an ONGOING entry for the last recorded
			// text, rather than just copying it into the input box
			// under whatever category happens to be selected --
			// repeating "still working on X" isn't the same as a
			// fresh DONE each time.
			if txt := lastEntryText(); txt != "" {
				recordActivity(withMins(txt), "ONGOING")
				minsInput.SetText("")
				refreshOpenItems()
			}
		}),
		widget.NewButton("Snooze", func() {
			Snooze(defaultSnoozeDuration())
			w4.Hide()
		}),
		widget.NewButton("Help...", func() { showHelp(a) }),
	)

	// showAllTagsBtn opens a standalone window listing every known
	// tag (KnownTags(), full ledger-history scan) -- "Frecent tags:"
	// only shows the top few by commonAndRecentTags's blended
	// frequency+recency score, this is the escape hatch to see
	// everything. (Editing/deleting tags across history is a possible
	// future extension, not implemented here -- see tags.go.)
	commonTagsRow := container.NewBorder(nil, nil, nil,
		widget.NewButton("Show all", func() { showAllTagsWindow(a) }),
		widget.NewLabel("Frecent tags: "+strings.Join(commonAndRecentTags(8), " ")))

	content := container.NewVBox(
		doneWrapper,
		// category, input,
		commonTagsRow,
		buttons,
		widget.NewSeparator(),
		itemsAccordion,
	)
	log.Println(content)

	// grid := container.New(layout.NewFormLayout(),
	// 	label1, value1, label2, value2, content)

	w4.SetContent(content)
	w4.Resize(fyne.NewSize(560, 400))
	w4.SetCloseIntercept(func() { w4.Hide() })
	w4.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyW,
		Modifier: fyne.KeyModifierShortcutDefault, // Cmd+W on macOS, Ctrl+W elsewhere
	}, func(fyne.Shortcut) { w4.Hide() })

	// Menu
	if desk, ok := a.(desktop.App); ok {
		meetingsMenu := fyne.NewMenu("Meetings",
			fyne.NewMenuItem("Meeting Prep...", func() { showMeetingPrepDialog(a) }),
			fyne.NewMenuItem("Post-Meeting Capture...", func() { showPostMeetingCapture(a, "") }),
			fyne.NewMenuItem("Recurring Meetings...", func() {
				showMiniCalendarDialog(a, w4)
			}),
		)
		meetingsItem := fyne.NewMenuItem("Meetings", nil)
		meetingsItem.ChildMenu = meetingsMenu

		reportsMenu := fyne.NewMenu("Reports",
			fyne.NewMenuItem("Summarize...", func() { showSummarizeDialog(a) }),
			fyne.NewMenuItem("Standup Summary...", func() { showStandupExport(a) }),
			fyne.NewMenuItem("Status Report...", func() { showStatusReportDialog(a) }),
			fyne.NewMenuItem("Annual Review...", func() { showAnnualReviewDialog(a) }),
			fyne.NewMenuItem("Trend View...", func() { showTrendView(a) }),
		)
		reportsItem := fyne.NewMenuItem("Reports", nil)
		reportsItem.ChildMenu = reportsMenu

		ledgerMenu := fyne.NewMenu("Ledger",
			fyne.NewMenuItem("Show Today's Ledger...", func() {
				w3 := a.NewWindow("Dunzo: Today")
				w3.SetContent(widget.NewLabel(strings.Join(readLedgerLines(), "\n")))
				w3.Resize(fyne.NewSize(500, 400))
				w3.Show()
			}),
			fyne.NewMenuItem("Edit Today's Ledger...", func() {
				_, fname := getLedger()
				openInEditor(fname)
			}),
			fyne.NewMenuItem("Undo/Edit Last Entry...", func() {
				showUndoEditLastEntry(a, func() {
					refreshOpenItems()
					refreshCompleted()
				})
			}),
			fyne.NewMenuItem("Search...", func() { showSearchDialog(a) }),
			fyne.NewMenuItem("Daily Summary Doc...", func() {
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
			}),
		)
		ledgerItem := fyne.NewMenuItem("Ledger", nil)
		ledgerItem.ChildMenu = ledgerMenu

		snoozeMenu := fyne.NewMenu("Snooze",
			fyne.NewMenuItem("15 min", func() { Snooze(15 * time.Minute) }),
			fyne.NewMenuItem("30 min", func() { Snooze(30 * time.Minute) }),
			fyne.NewMenuItem("1 hour", func() { Snooze(60 * time.Minute) }),
		)
		snoozeItem := fyne.NewMenuItem("Snooze", func() { Snooze(defaultSnoozeDuration()) })
		snoozeItem.ChildMenu = snoozeMenu

		var dndItem *fyne.MenuItem
		var m *fyne.Menu
		dndItem = fyne.NewMenuItem("Do Not Disturb", func() {
			on := !dndItem.Checked
			SetDoNotDisturb(on)
			dndItem.Checked = on
			desk.SetSystemTrayMenu(m)
		})
		dndItem.Checked = IsDoNotDisturb()

		// Since Daybook is normally hidden and only pops up briefly
		// (per Micah), the tray menu -- not Daybook -- is the primary
		// surface for anything that isn't a direct reaction to
		// Daybook already being on screen. Frequent/time-sensitive
		// items (Show, SOD/EOD/SOM, Snooze) stay top-level and
		// un-buried; everything else groups into a submenu by domain
		// (Meetings/Reports/Ledger) rather than by FR number or
		// chronology.
		m = fyne.NewMenu("Dunzo",
			fyne.NewMenuItem("Show", func() {
				refreshOpenItems()
				refreshCompleted()
				w4.Show()
				w4.RequestFocus()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Start of Day...", func() { showSODWindow(a) }),
			fyne.NewMenuItem("End of Day...", func() { showEODWindow(a) }),
			fyne.NewMenuItem("Start of Month...", func() { showSOMWindow(a) }),
			snoozeItem,
			dndItem,
			fyne.NewMenuItemSeparator(),
			meetingsItem,
			reportsItem,
			ledgerItem,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Help...", func() { showHelp(a) }),
			fyne.NewMenuItem("Settings...", func() { showSettings(a) }),
		)
		desk.SetSystemTrayMenu(m)
	}

	w4.Show()

	return w4
}

func updateTime(clock *widget.Label) {
	formatted := time.Now().Format("Dunnit: 03:04:05")
	clock.SetText(formatted)
}
