package dun

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"bufio"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
)

// Get today's ledger file path and name.
func getLedger() (string, string) {
	yr, wk := time.Now().ISOWeek()
	// mo := time.Now().Month()
	t := time.Now().UTC()
	tn := time.Now()
	yr8 := tn.Format("20060102")
	moname := t.Format("Jan")
	fname0 := "ledger-" + yr8 + ".txt"
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

// mainInputEntry holds the Daybook window's main text-entry widget,
// captured once by BuildMainWindow, so FocusMainInput (called
// whenever Daybook is raised, e.g. by sched.go's nudges and the tray
// menu's "Show") can request keyboard focus land there directly,
// rather than wherever focus happened to be left (or nowhere).
var mainInputEntry *closeShortcutEntry

// trayApp/trayWindow cache BuildMainWindow's fyne.App/main-window
// references so RebuildTrayMenu (called after Settings saves a
// changed Kickoff/Review toggle) can rebuild+reapply the tray menu
// without needing BuildMainWindow itself to be re-run. Set once by
// BuildMainWindow; nil until then (RebuildTrayMenu no-ops if so).
var trayApp fyne.App
var trayWindow fyne.Window

// trayRefreshAll refreshes Daybook's open/completed/reflections/last-
// done sections -- set once by BuildMainWindow (which owns the actual
// refreshX closures, scoped to its own widget state), called by
// buildTrayMenu's "Show" item. nil-checked before use since it isn't
// set until BuildMainWindow has run.
var trayRefreshAll func()

// FocusMainInput requests keyboard focus on Daybook's main entry box,
// if it's been built yet. Safe to call even before BuildMainWindow
// has run (no-op).
func FocusMainInput() {
	if mainInputEntry == nil {
		return
	}
	fyne.Do(func() {
		if c := fyne.CurrentApp().Driver().CanvasForObject(mainInputEntry); c != nil {
			c.Focus(mainInputEntry)
		}
	})
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
	text = strings.TrimSpace(text)
	log.Println("Content was:", text)
	fpath, fname := getLedger()
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		log.Println("Making new dir:", fpath)
		os.MkdirAll(fpath, os.ModePerm)
	}
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
	InvalidateLedgerIndex()
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
// may grow beyond just categories later. EODOnly categories (SUMMARY/
// PRODUCTIVITY/MEETING_HOURS) are excluded entirely -- they're always
// machine-written bookkeeping from eod.go's Finalize Day flow, never
// hand-picked, so they'd only clutter a legend meant to help someone
// choose a category from the live picker.
//
// Note: this coloring only applies to the static Help window -- Fyne's
// widget.Select doesn't support per-option rich text/color in its
// dropdown, so the live category picker itself stays plain text.
func showHelp(a fyne.App) {
	darkGreen := color.NRGBA{R: 0, G: 100, B: 0, A: 255}
	darkRed := color.NRGBA{R: 139, G: 0, B: 0, A: 255}

	// labelColWidth is the fixed column width (in characters, since
	// labels render Monospace) each category's Label() is padded out
	// to before appending its Help text -- without this, Help text
	// starts at a different column per row (labels vary quite a bit
	// in length, e.g. "✔️ DONE" vs "🏁 MILESTONE"), making the whole
	// legend look jagged. Computed as the longest non-EODOnly Label()
	// (rune count) + 1, so it stays correct as categories are added/
	// renamed rather than needing a hand-picked constant kept in sync.
	labelColWidth := 0
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		if n := len([]rune(c.Label())); n > labelColWidth {
			labelColWidth = n
		}
	}
	labelColWidth++

	w := a.NewWindow("Dunzo: Help")
	rows := container.NewVBox()
	lastGroup := ""
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		if c.Group != lastGroup {
			header := canvas.NewText(GroupLabel(c.Group), theme.Color(theme.ColorNameForeground))
			header.TextSize = 12
			header.TextStyle = fyne.TextStyle{Bold: true}
			rows.Add(header)
			lastGroup = c.Group
		}
		label := c.Label()
		labelRunes := []rune(label)
		if pad := labelColWidth - len(labelRunes); pad > 0 {
			label += strings.Repeat(" ", pad)
		}
		txt := canvas.NewText(label+"\u2014 "+c.Help, theme.Color(theme.ColorNameForeground))
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
	// Theme is set in BuildMainWindow (which also needs LightTheme's
	// color scheme, see eod.go's comment) rather than here -- setting
	// it in both places is harmless (SetTheme just replaces), but
	// BuildMainWindow is the actual single point of truth so it's
	// only done there to avoid the two calls silently drifting out of
	// sync the way they did before (see compactTheme's doc comment).
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
	a.Settings().SetTheme(newCompactTheme())
	cfg := LoadConfig()

	w4 := a.NewWindow("Dunzo: Daybook")
	// label1 := widget.NewLabel("Label 1")
	// value1 := widget.NewLabel("Value")
	// label2 := widget.NewLabel("Label 2")
	// value2 := widget.NewLabel("Something")

	// TODO show day's GOALs

	input := newCloseShortcutEntry(fyne.KeyW, fyne.KeyModifierShortcutDefault, func() { w4.Hide() })
	input.SetPlaceHolder("Enter text\u2026")
	mainInputEntry = input
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
	//
	// defaultGroupFilter/defaultGroupOptions: if cfg.FavoriteCategories
	// is non-empty, "Faves" is the picker's default-active filter
	// (replacing "whatever group was last used" -- Micah wants his
	// chosen bucket active every time Daybook pops up, not sticky
	// last-used state) and is added to the groupFilter dropdown's
	// options; otherwise behaves exactly as before (defaults to
	// "Now", no Faves option shown at all).
	faves := CategoryLabelsForFaves(cfg)
	defaultGroupFilter := "Now"
	defaultGroupOptions := []string{"Now", "Plan", "Reflect", "All"}
	defaultCategoryOptions := CategoryLabelsForGroup("now")
	if len(faves) > 0 {
		defaultGroupFilter = "Faves"
		defaultGroupOptions = []string{"Faves", "Now", "Plan", "Reflect", "All"}
		defaultCategoryOptions = faves
	}

	category := widget.NewSelect(defaultCategoryOptions,
		func(cat string) {
			fmt.Println("saw a category:", cat)
			res := strings.Split(cat, " ")
			// selectedCat = cat
			selectedCat = res[1]
		})
	category.SetSelected(defaultCategoryOptions[0])

	// groupFilter narrows the category picker to a subset (FR-06
	// follow-up): "Faves" (user-configured, see
	// Config.FavoriteCategories -- shown/default only if configured),
	// "now" (day-to-day capture), "plan" (future-facing), "reflect"
	// (retrospective/EOD-ish), or "all". Purely a UI convenience over
	// the same Categories list.
	groupFilter := widget.NewSelect(defaultGroupOptions,
		func(g string) {
			if g == "Faves" {
				category.Options = faves
			} else {
				category.Options = CategoryLabelsForGroup(strings.ToLower(g))
			}
			category.SetSelected(category.Options[0])
			category.Refresh()
		})
	groupFilter.SetSelected(defaultGroupFilter)

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
	// doneWrapper is the main entry row (groupFilter/category/input/
	// minsWrapper) -- Daybook's single most important row, where
	// virtually every interaction starts. Window-edge spacing is now
	// handled once, uniformly, by the outer contentPad wrap below
	// (see its comment) -- this local wrap only adds a bit of extra
	// *bottom* padding to visually separate this row from
	// commonTagsRow underneath it, since it's the primary piece of
	// Daybook and deserves to stand apart from the rest. Value is an
	// arbitrary "looks reasonable" pick, not theme-driven -- adjust
	// here if it looks off.
	doneWrapper := container.New(layout.NewCustomPaddedLayout(0, 6, 0, 0),
		container.New(newStretchRowLayout(input), groupFilter, category, input, minsWrapper))

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
	var refreshCompleted func()          // forward decl -- used inside refreshOpenItems's Done button, defined below
	var refreshLastDone func()           // forward decl -- used inside refreshOpenItems's Done button and saveEntry, defined further below (needs lastDoneLabel)
	var itemsAccordion *widget.Accordion // forward decl -- used inside refreshOpenItems's Done/Postpone/Discard buttons, defined below
	// showAllPlanned toggles whether Planned's non-TODO categories
	// (GOAL/WAITING/QUESTION/FIXME/RISK) are shown -- default false
	// (TODO-only) since the full set together was feeling
	// overwhelming; a "Show all" / "Show TODOs only" toggle button
	// reveals/hides the rest without losing them.
	showAllPlanned := false
	refreshOpenItems = func() {
		openItemsBox.RemoveAll()
		items := getOpenItems()
		if len(items) == 0 {
			openItemsBox.Add(widget.NewLabel("Nothing open right now."))
			openItemsBox.Refresh()
			return
		}
		addRow := func(item OpenItem) {
			// Icon-only widget.NewButtonWithIcon (empty label), not
			// newHoverIconButton -- this Planned section specifically
			// hit the tooltip-popup click-swallowing bug
			// hoverbutton.go documents (a click needs two separate
			// taps to register once the hover tooltip has appeared),
			// which the existing tapped-forwarding patch there
			// doesn't fully close. Sidestepping the tooltip-popup
			// mechanism entirely removes the whole bug class rather
			// than chasing Fyne's overlay hit-testing further. No
			// text label and no hover tooltip either (Micah doesn't
			// want either) -- old labels noted in comments below for
			// reference.
			row := container.NewBorder(nil, nil, nil,
				container.NewHBox(
					widget.NewButtonWithIcon("", theme.Icon(theme.IconNameContentClear), func() { // "Discard"
						recordDiscarded(item)
						fyne.Do(func() {
							refreshOpenItems()
							itemsAccordion.Refresh()
						})
					}),
					widget.NewButtonWithIcon("", theme.Icon(theme.IconNameHistory), func() { // "Postpone"
						recordPostponed(item)
						fyne.Do(func() {
							refreshOpenItems()
							itemsAccordion.Refresh()
						})
					}),
					widget.NewButtonWithIcon("", theme.Icon(theme.IconNameConfirm), func() { // "Done"
						recordConvertedDone(item)
						fyne.Do(func() {
							refreshOpenItems()
							refreshCompleted()
							refreshLastDone()
							itemsAccordion.Refresh()
						})
					}),
					widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDocumentCreate), func() { // "Edit"
						showEditItemDialog(w4, item, func() {
							fyne.Do(func() {
								refreshOpenItems()
								itemsAccordion.Refresh()
							})
						})
					}),
				),
				widget.NewLabel("\u2022 "+item.Text))
			openItemsBox.Add(row)
		}
		cats, grouped := groupOpenItemsByCategory(items)
		otherCount := 0
		for _, cat := range cats {
			if cat != "TODO" {
				otherCount += len(grouped[cat])
				continue
			}
			openItemsBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
			for _, item := range grouped[cat] {
				addRow(item)
			}
		}
		if otherCount > 0 {
			toggleLabel := fmt.Sprintf("Show all (%d more)", otherCount)
			if showAllPlanned {
				toggleLabel = "Show TODOs only"
			}
			openItemsBox.Add(widget.NewButton(toggleLabel, func() {
				showAllPlanned = !showAllPlanned
				refreshOpenItems()
				itemsAccordion.Refresh()
			}))
			if showAllPlanned {
				for _, cat := range cats {
					if cat == "TODO" {
						continue
					}
					openItemsBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
					for _, item := range grouped[cat] {
						addRow(item)
					}
				}
			}
		}
		openItemsBox.Refresh()
	}
	refreshOpenItems()

	// completedBox displays today's "now"-group entries (DONE/ONGOING/
	// TIL/KUDOS/WIN -- not just DONE) grouped by category with
	// per-category sub-headings, mirroring how Planned already splits
	// TODO/GOAL/etc into their own sections (see groupOpenItemsByCategory).
	// Also collapsible, placed right below Planned in the same accordion.
	// Section is labeled "Activity" (not "Completed") since it covers
	// all "now" cats, not just finished/DONE ones.
	completedBox := container.NewVBox()
	refreshCompleted = func() {
		completedBox.RemoveAll()
		items := getCategoryGroupItems("now")
		if len(items) == 0 {
			completedBox.Add(widget.NewLabel("Nothing logged yet today."))
			completedBox.Refresh()
			return
		}
		cats, grouped := groupCategoryItemsByGroup("now", items)
		for _, cat := range cats {
			completedBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
			for _, item := range grouped[cat] {
				item := item // capture
				row := container.NewBorder(nil, nil, nil,
					widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDocumentCreate), func() { // "Edit"
						showEditItemDialog(w4, item, func() {
							fyne.Do(func() {
								refreshCompleted()
								refreshLastDone()
								itemsAccordion.Refresh()
							})
						})
					}),
					widget.NewLabel("\u2022 "+item.Text))
				completedBox.Add(row)
			}
		}
		completedBox.Refresh()
	}
	refreshCompleted()

	// reflectionsBox displays today's "reflect"-group entries
	// (IMPACT/MILESTONE/CAREER/FAIL/WASTED -- excludes the EODOnly
	// SUMMARY/PRODUCTIVITY/MEETING_HOURS codes only in the sense that
	// those are rare to see mid-day, though if present they'd still
	// show here; EODOnly only gates the *picker*, not this readback),
	// grouped by category with sub-headings, same pattern as
	// Completed/Planned.
	reflectionsBox := container.NewVBox()
	var refreshReflections func()
	refreshReflections = func() {
		reflectionsBox.RemoveAll()
		items := getCategoryGroupItems("reflect")
		if len(items) == 0 {
			reflectionsBox.Add(widget.NewLabel("Nothing reflected on yet today."))
			reflectionsBox.Refresh()
			return
		}
		cats, grouped := groupCategoryItemsByGroup("reflect", items)
		for _, cat := range cats {
			reflectionsBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
			for _, item := range grouped[cat] {
				item := item // capture
				row := container.NewBorder(nil, nil, nil,
					widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDocumentCreate), func() { // "Edit"
						showEditItemDialog(w4, item, func() {
							fyne.Do(func() {
								refreshReflections()
								itemsAccordion.Refresh()
							})
						})
					}),
					widget.NewLabel("\u2022 "+item.Text))
				reflectionsBox.Add(row)
			}
		}
		reflectionsBox.Refresh()
	}
	refreshReflections()

	// Planned/Activity/Reflections are all collapsible (via
	// widget.Accordion) so any can be tucked out of the way once
	// reviewed. All three start collapsed -- Daybook pops up briefly
	// (per Micah) and the last-DONE-item label + buttons row above
	// already surface the most immediately relevant info, so nothing
	// needs to auto-expand.
	upcomingItem := widget.NewAccordionItem("Planned", openItemsBox)
	completedItem := widget.NewAccordionItem("Activity", completedBox)
	reflectionsItem := widget.NewAccordionItem("Reflections", reflectionsBox)
	itemsAccordion = widget.NewAccordion(completedItem, upcomingItem, reflectionsItem)

	saveEntry := func() {
		if strings.TrimSpace(input.Text) == "" {
			return
		}
		recordActivity(withMins(input.Text), selectedCat) // TODO trim emoji off front, and shorten to 4-char code
		input.SetText("")
		minsInput.SetText("")
		refreshOpenItems()
		refreshCompleted()
		refreshReflections()
		refreshLastDone()
	}
	input.OnSubmitted = func(string) { saveEntry() }
	minsInput.OnSubmitted = func(string) { saveEntry() }

	buttons := container.NewHBox(
		widget.NewButton("Save", saveEntry),
		widget.NewButton("Snooze", func() {
			Snooze(defaultSnoozeDuration())
			w4.Hide()
		}),
		widget.NewButton("Help...", func() { showHelp(a) }),
	)

	// lastDoneLabel shows the most recently logged DONE entry's text
	// just below the buttons row -- a quick "what did I just finish?"
	// glance without opening the (now-collapsed-by-default) Activity
	// section. dittoBtn sits immediately to its left since Ditto acts
	// on that same "last entry" (see lastEntryText, which is the more
	// general "last logged line of any category" used by Ditto itself
	// -- lastDoneLabel specifically shows the last *DONE* one).
	lastDoneLabel := widget.NewLabel("")
	refreshLastDone = func() {
		done := getCompletedItems()
		if len(done) == 0 {
			lastDoneLabel.SetText("(nothing completed yet today)")
			return
		}
		lastDoneLabel.SetText("Last done: " + done[len(done)-1])
	}
	refreshLastDone()

	dittoBtn := widget.NewButton("Ditto", func() {
		// Ditto extends the last DONE item: logs a fresh DONE entry
		// with the same text (so it looks freshly completed again),
		// and rewrites the *original* DONE line's category to ONGOING
		// (it was actually still being worked on, not really "done"
		// at that point -- semantically this is "extend", though
		// that word is never shown to the user, it's just quiet
		// historical editing of an append-only-in-spirit ledger).
		// Deliberately keyed off the last *DONE* entry specifically
		// (lastDoneItem), not lastEntryText's "last logged line of any
		// category" -- Ditto should never accidentally repeat/extend
		// a RISK or other non-DONE entry just because it happened to
		// be logged more recently.
		if item, ok := lastDoneItem(); ok {
			replaceLedgerLineCategoryAt(item.LineIndex, "ONGOING")
			recordActivity(withMins(item.Text), "DONE")
			minsInput.SetText("")
			refreshOpenItems()
			refreshCompleted()
			refreshLastDone()
		}
	})
	lastDoneRow := container.NewHBox(dittoBtn, lastDoneLabel)

	// showAllTagsBtn opens a standalone window listing every known
	// tag (KnownTags(), full ledger-history scan) -- "Frecent tags:"
	// only shows the top few by commonAndRecentTags's blended
	// frequency+recency score, this is the escape hatch to see
	// everything. (Editing/deleting tags across history is a possible
	// future extension, not implemented here -- see tags.go.)
	//
	// Each tag is rendered as a clickable blue tagLink (not plain
	// text) -- clicking one inserts it at the cursor position in the
	// main entry box and refocuses it, so tags can be added without
	// typing "#" and waiting for autocomplete.
	insertTagAtCursor := func(tag string) {
		runes := []rune(input.Text)
		col := input.CursorColumn
		if col < 0 || col > len(runes) {
			col = len(runes)
		}
		insert := tag
		if col > 0 && !isTagBreak(runes[col-1]) {
			insert = " " + insert
		}
		insert += " "
		newText := string(runes[:col]) + insert + string(runes[col:])
		input.SetText(newText)
		input.CursorColumn = col + len([]rune(insert))
		input.Refresh()
		FocusMainInput()
	}
	frecentTags, frecentCounts := commonAndRecentTagsWithCounts(8)
	frecentTagsRow := container.NewHBox(widget.NewLabel("Frecent tags:"))
	for i, tag := range frecentTags {
		tag := tag // capture
		frecentTagsRow.Add(newTagLink(formatTagWithCount(tag, &tagStat{count: frecentCounts[i]}), func() {
			insertTagAtCursor(tag)
		}))
	}
	commonTagsRow := container.NewBorder(nil, nil, nil,
		widget.NewButton("Show all", func() { showAllTagsWindow(a) }),
		frecentTagsRow)

	content := container.NewVBox(
		doneWrapper,
		// category, input,
		commonTagsRow,
		buttons,
		lastDoneRow,
		widget.NewSeparator(),
		itemsAccordion,
	)
	log.Println(content)

	// grid := container.New(layout.NewFormLayout(),
	// 	label1, value1, label2, value2, content)

	// contentPad wraps the entire window content in fixed edge
	// padding, independent of compactTheme's Size overrides (theme.go)
	// -- those only affect spacing *between* sibling widgets, not the
	// gap between the outermost content and the window frame, which
	// is why the earlier attempt at window-edge spacing (doneWrapper's
	// own internal CustomPaddedLayout) only visibly helped the very
	// top row (it happens to sit flush against the window's top/left/
	// right edges as the first VBox child) and did nothing for the
	// left/right/bottom edges of every other section below it, or the
	// bottom edge overall. This single outer wrap covers all edges,
	// for every section, in one place.
	contentPad := container.New(layout.NewCustomPaddedLayout(10, 10, 10, 10), content)

	w4.SetContent(contentPad)
	w4.Resize(fyne.NewSize(560, 400))
	w4.SetCloseIntercept(func() { w4.Hide() })
	w4.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyW,
		Modifier: fyne.KeyModifierShortcutDefault, // Cmd+W on macOS, Ctrl+W elsewhere
	}, func(fyne.Shortcut) { w4.Hide() })

	// Menu
	if desk, ok := a.(desktop.App); ok {
		trayApp = a
		trayWindow = w4
		trayRefreshAll = func() {
			refreshOpenItems()
			refreshCompleted()
			refreshReflections()
			refreshLastDone()
		}
		desk.SetSystemTrayMenu(buildTrayMenu(a, w4))
	}

	w4.Show()
	FocusMainInput()

	return w4
}

// buildTrayMenu constructs the full tray/system-menu structure for
// BuildMainWindow's Fyne app+main-window (a, w4) -- factored out so
// RebuildTrayMenu can call it again after Settings changes a Kickoff/
// Review toggle, without needing BuildMainWindow itself to re-run.
func buildTrayMenu(a fyne.App, w4 fyne.Window) *fyne.Menu {
	desk, ok := a.(desktop.App)
	if !ok {
		return nil
	}

	meetingsMenu := fyne.NewMenu("Meetings",
		fyne.NewMenuItem("Meeting Prep...", func() { showMeetingPrepDialog(a) }),
		fyne.NewMenuItem("Post-Meeting Capture...", func() { showPostMeetingCapture(a, "") }),
		fyne.NewMenuItem("Standup Summary...", func() { showStandupExport(a) }),
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
		fyne.NewMenuItem("Reports Library...", func() { showReportsLibraryWindow(a) }),
	)
	reportsItem := fyne.NewMenuItem("Reports", nil)
	reportsItem.ChildMenu = reportsMenu

	// Kickoff.../Review... submenus (docs/kickoff-review-design.md),
	// replacing the old flat Start of Day/End of Day/Start of Month
	// items. Day keeps its existing bespoke dialogs (SOD/EOD); Month
	// has its own dedicated Kickoff/Review windows
	// (monthkickoff.go/monthreview.go); Week/Quarter/Year route
	// through the generic showPeriodKickoffWindow/
	// showPeriodReviewWindow (periodkickoff.go/periodreview.go),
	// which have no bespoke dialog of their own since their shape is
	// identical. Each included item is gated by its own
	// kickoffEnabled/reviewEnabled Config toggle, re-read fresh each
	// time this function runs -- Quarter/Year default off (see
	// Config's doc comment), so they won't appear until explicitly
	// enabled via Settings.
	cfg := LoadConfig()
	var kickoffItems, reviewItems []*fyne.MenuItem
	if kickoffEnabled(cfg, periodDay) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Day...", func() { showSODWindow(a) }))
	}
	if kickoffEnabled(cfg, periodWeek) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Week...", func() { showPeriodKickoffWindow(a, periodWeek, time.Now()) }))
	}
	if kickoffEnabled(cfg, periodMonth) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Month...", func() { showMonthKickoffWindow(a, time.Now()) }))
	}
	if kickoffEnabled(cfg, periodQuarter) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Quarter...", func() { showPeriodKickoffWindow(a, periodQuarter, time.Now()) }))
	}
	if kickoffEnabled(cfg, periodYear) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Year...", func() { showPeriodKickoffWindow(a, periodYear, time.Now()) }))
	}
	if reviewEnabled(cfg, periodDay) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Day...", func() { showEODWindow(a) }))
	}
	// Week/Month/Quarter/Year Review all route through
	// showPeriodPicker first (docs/kickoff-review-design.md's "which
	// period" fix) rather than hardcoding "the previous period" --
	// lets the user pick this period (so far), last period, or a
	// short back-list instead of always landing on the wrong month.
	if reviewEnabled(cfg, periodWeek) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Week...", func() {
			showPeriodPicker(a, cfg, periodWeek, func(anchor time.Time) {
				showPeriodReviewWindow(a, periodWeek, anchor)
			})
		}))
	}
	// Month's Review and Kickoff are now split (showMonthReviewWindow/
	// showMonthKickoffWindow), matching the generic Week/Quarter/Year
	// pattern -- see docs/kickoff-review-design.md's "Scope note".
	if reviewEnabled(cfg, periodMonth) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Month...", func() {
			showPeriodPicker(a, cfg, periodMonth, func(anchor time.Time) {
				showMonthReviewWindow(a, anchor)
			})
		}))
	}
	if reviewEnabled(cfg, periodQuarter) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Quarter...", func() {
			showPeriodPicker(a, cfg, periodQuarter, func(anchor time.Time) {
				showPeriodReviewWindow(a, periodQuarter, anchor)
			})
		}))
	}
	if reviewEnabled(cfg, periodYear) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Year...", func() {
			showPeriodPicker(a, cfg, periodYear, func(anchor time.Time) {
				showPeriodReviewWindow(a, periodYear, anchor)
			})
		}))
	}
	kickoffItem := fyne.NewMenuItem("Kickoff", nil)
	kickoffItem.ChildMenu = fyne.NewMenu("Kickoff", kickoffItems...)
	reviewItem := fyne.NewMenuItem("Review", nil)
	reviewItem.ChildMenu = fyne.NewMenu("Review", reviewItems...)

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
				if trayRefreshAll != nil {
					trayRefreshAll()
				}
			})
		}),
		fyne.NewMenuItem("Search...", func() { showSearchDialog(a) }),
		fyne.NewMenuItem("Navigator...", func() { showNavigatorWindow(a) }),
		fyne.NewMenuItem("Recurring Items...", func() {
			showRecurringItemsDialog(a, w4)
		}),
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

	// Since Daybook is normally hidden and only pops up briefly (per
	// Micah), the tray menu -- not Daybook -- is the primary surface
	// for anything that isn't a direct reaction to Daybook already
	// being on screen. Frequent/time-sensitive items (Show, Kickoff/
	// Review, Snooze) stay top-level and un-buried; everything else
	// groups into a submenu by domain (Meetings/Reports/Ledger)
	// rather than by FR number or chronology.
	m = fyne.NewMenu("Dunzo",
		fyne.NewMenuItem("Show", func() {
			if trayRefreshAll != nil {
				trayRefreshAll()
			}
			w4.Show()
			w4.RequestFocus()
			FocusMainInput()
		}),
		fyne.NewMenuItemSeparator(),
		kickoffItem,
		reviewItem,
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
	return m
}

// RebuildTrayMenu re-reads Config and reapplies the tray menu, so a
// Kickoff/Review toggle changed in Settings (or any other config.toml
// change affecting menu contents) takes effect immediately rather
// than requiring an app restart. No-op if BuildMainWindow hasn't run
// yet (trayApp/trayWindow unset) or the platform has no desktop tray.
func RebuildTrayMenu() {
	if trayApp == nil || trayWindow == nil {
		return
	}
	desk, ok := trayApp.(desktop.App)
	if !ok {
		return
	}
	desk.SetSystemTrayMenu(buildTrayMenu(trayApp, trayWindow))
}

func updateTime(clock *widget.Label) {
	formatted := time.Now().Format("Dunnit: 03:04:05")
	clock.SetText(formatted)
}
