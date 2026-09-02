package dun

import (
	"fmt"
	"time"
)

// periodConfig holds the small set of things that differ per unit
// for the Kickoff/Review system (see
// docs/kickoff-review-design.md): which unit rolls up into it
// (SubPeriod, empty if none), and its default Review theme.
// Toggle/theme *settings* live in Config (config.go); this struct is
// static, code-level metadata about each unit's shape, not
// user-editable state.
type periodConfig struct {
	Period       summaryPeriod
	SubPeriod    summaryPeriod // "" if none (Day, Week have no sub-tier)
	DefaultTheme string
}

// periodConfigs is the static table of the 5 units' shape, keyed by
// summaryPeriod. Quarters are traditional calendar quarters
// (Jan-Mar/Apr-Jun/Jul-Sep/Oct-Dec); years are Jan 1 - Dec 31 -- no
// fiscal-year config, at least for now.
var periodConfigs = map[summaryPeriod]periodConfig{
	periodDay:     {Period: periodDay, DefaultTheme: ThemePersonalNotes},
	periodWeek:    {Period: periodWeek, DefaultTheme: ThemePersonalNotes},
	periodMonth:   {Period: periodMonth, SubPeriod: periodWeek, DefaultTheme: ThemeStatusReport},
	periodQuarter: {Period: periodQuarter, SubPeriod: periodMonth, DefaultTheme: ThemeFormalReport},
	periodYear:    {Period: periodYear, SubPeriod: periodQuarter, DefaultTheme: ThemeFormalReport},
}

// Theme names for Review generation (docs/kickoff-review-design.md).
// Kept as plain string constants (rather than a typed enum) since
// they're stored directly as Config TOML values and compared against
// user-editable strings.
const (
	ThemePersonalNotes = "personal_notes"
	ThemeStatusReport  = "status_report"
	ThemeFormalReport  = "formal_report"
	ThemeBragPreso     = "brag_preso"
)

// quarterOf returns 1-4 for the traditional calendar quarter
// containing t (Jan-Mar=1, Apr-Jun=2, Jul-Sep=3, Oct-Dec=4).
func quarterOf(t time.Time) int {
	return (int(t.Month())-1)/3 + 1
}

// periodLabel returns the human/report-facing name for a period,
// naming the *nominal* calendar unit containing anchor -- always
// independent of whatever partial data-fetch window a Review actually
// managed to pull (see periodDataRange), so titles never read
// something awkward like "covers Apr 1 - Jun 25".
//
// Examples: periodLabel(periodDay, t) -> "Mon Sep 1, 2026";
// periodLabel(periodWeek, t) -> "Week of Sep 1, 2026" (week start is
// Monday, matching time.Time.ISOWeek's convention used elsewhere in
// this codebase); periodLabel(periodMonth, t) -> "Sep 2026";
// periodLabel(periodQuarter, t) -> "Q3 2026 (Jul-Sep)";
// periodLabel(periodYear, t) -> "2026".
func periodLabel(period summaryPeriod, anchor time.Time) string {
	switch period {
	case periodDay:
		return anchor.Format("Mon Jan 2, 2006")
	case periodWeek:
		start := weekStart(anchor)
		return "Week of " + start.Format("Jan 2, 2006")
	case periodMonth:
		return anchor.Format("Jan 2006")
	case periodQuarter:
		q := quarterOf(anchor)
		monthNames := [4]string{"Jan-Mar", "Apr-Jun", "Jul-Sep", "Oct-Dec"}
		return fmt.Sprintf("Q%d %d (%s)", q, anchor.Year(), monthNames[q-1])
	case periodYear:
		return fmt.Sprintf("%d", anchor.Year())
	default:
		return string(period)
	}
}

// weekStart returns the Monday (00:00) of the ISO week containing t,
// matching the week-boundary convention time.Time.ISOWeek already
// uses elsewhere in this codebase (e.g. getLedger's directory
// scheme).
func weekStart(t time.Time) time.Time {
	// time.Weekday: Sunday=0 .. Saturday=6. Convert to ISO
	// (Monday=0..Sunday=6) so Monday is treated as day 0 of the week.
	isoWeekday := (int(t.Weekday()) + 6) % 7
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -isoWeekday)
}

// periodNominalRange returns the exact calendar boundaries [from, to]
// of the unit containing anchor -- e.g. for periodQuarter, the first
// instant of the quarter through the last instant of its last day.
// Used both for periodLabel-adjacent bookkeeping and as the basis
// periodDataRange pads outward from.
func periodNominalRange(period summaryPeriod, anchor time.Time) (from, to time.Time) {
	loc := anchor.Location()
	switch period {
	case periodDay:
		from = time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, loc)
		to = from.AddDate(0, 0, 1).Add(-time.Second)
	case periodWeek:
		from = weekStart(anchor)
		to = from.AddDate(0, 0, 7).Add(-time.Second)
	case periodMonth:
		from = time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, loc)
		to = from.AddDate(0, 1, 0).Add(-time.Second)
	case periodQuarter:
		q := quarterOf(anchor)
		startMonth := time.Month((q-1)*3 + 1)
		from = time.Date(anchor.Year(), startMonth, 1, 0, 0, 0, 0, loc)
		to = from.AddDate(0, 3, 0).Add(-time.Second)
	case periodYear:
		from = time.Date(anchor.Year(), time.January, 1, 0, 0, 0, 0, loc)
		to = from.AddDate(1, 0, 0).Add(-time.Second)
	default:
		from, to = anchor, anchor
	}
	return from, to
}

// periodDataRangePadDays is how many extra days beyond the nominal
// boundary periodDataRange pads its returned window by, in either
// direction. Deliberately loose (see docs/kickoff-review-design.md's
// "Configurable lead-time/window" section) -- duplicate coverage of
// the same days across adjacent reports is an acceptable tradeoff for
// avoiding a coverage gap at a boundary; no need for exact interval
// math anywhere that consumes this.
const periodDataRangePadDays = 1

// periodDataRange returns the [from, to] window to actually pull
// source material from for period's Review of the unit containing
// anchor -- the nominal calendar boundaries, padded loosely by
// periodDataRangePadDays on each side. Titles/labels should always
// use periodLabel instead of this range, since this range is
// intentionally imprecise.
func periodDataRange(period summaryPeriod, anchor time.Time) (from, to time.Time) {
	from, to = periodNominalRange(period, anchor)
	pad := time.Duration(periodDataRangePadDays) * 24 * time.Hour
	return from.Add(-pad), to.Add(pad)
}

// kickoffEnabled/reviewEnabled/themeFor read a unit's Kickoff/Review
// toggle or theme setting out of cfg, keyed by period -- small
// switch-based accessors so callers work in terms of summaryPeriod
// rather than reaching into Config's individual per-unit fields
// directly.
func kickoffEnabled(cfg Config, period summaryPeriod) bool {
	switch period {
	case periodDay:
		return cfg.KickoffDayEnabled
	case periodWeek:
		return cfg.KickoffWeekEnabled
	case periodMonth:
		return cfg.KickoffMonthEnabled
	case periodQuarter:
		return cfg.KickoffQuarterEnabled
	case periodYear:
		return cfg.KickoffYearEnabled
	default:
		return false
	}
}

func reviewEnabled(cfg Config, period summaryPeriod) bool {
	switch period {
	case periodDay:
		return cfg.ReviewDayEnabled
	case periodWeek:
		return cfg.ReviewWeekEnabled
	case periodMonth:
		return cfg.ReviewMonthEnabled
	case periodQuarter:
		return cfg.ReviewQuarterEnabled
	case periodYear:
		return cfg.ReviewYearEnabled
	default:
		return false
	}
}

// themeFor returns cfg's configured default theme for period, falling
// back to periodConfigs' DefaultTheme if unset (e.g. an older
// config.toml written before theme fields existed, decoded to "").
func themeFor(cfg Config, period summaryPeriod) string {
	var theme string
	switch period {
	case periodDay:
		theme = cfg.ThemeDay
	case periodWeek:
		theme = cfg.ThemeWeek
	case periodMonth:
		theme = cfg.ThemeMonth
	case periodQuarter:
		theme = cfg.ThemeQuarter
	case periodYear:
		theme = cfg.ThemeYear
	}
	if theme != "" {
		return theme
	}
	return periodConfigs[period].DefaultTheme
}

// reviewLengthWords is the rough target word-count ceiling to instruct
// the copilot prompt with for each unit's Review, scaled so a Day
// digest stays tight while a Year digest is allowed more room to
// cover a full year's worth of rolled-up material. Deliberately a
// ceiling stated in the prompt itself (not just achieved structurally
// via the rollup in review.go) -- see docs/kickoff-review-design.md's
// "Keeping prompts/outputs small" section: structural rollup and
// explicit prompt instruction are both needed, since rollup alone
// doesn't stop an LLM from padding its own output with prose.
var reviewLengthWords = map[summaryPeriod]int{
	periodDay:     150,
	periodWeek:    250,
	periodMonth:   400,
	periodQuarter: 600,
	periodYear:    800,
}

// reviewLengthConstraint returns a short instruction fragment, scaled
// to period, to append to any Review-generating copilot prompt --
// e.g. summarizeWithCopilotPrompt(someInstructions+reviewLengthConstraint(periodMonth), ...).
// Kept as a standalone appendable fragment (not baked into
// summarizeWithCopilotPrompt itself) so non-Review callers (e.g.
// Standup's summarizeStandupWithCopilot, which already has its own
// "Be concise" framing) aren't forced to adopt it.
func reviewLengthConstraint(period summaryPeriod) string {
	words, ok := reviewLengthWords[period]
	if !ok {
		words = 300
	}
	return fmt.Sprintf(" Keep the total output under roughly %d words; be terse, "+
		"favor bullet points over prose.", words)
}
