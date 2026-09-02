package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// reviewReportKind returns the periodReportPath "kind" prefix used
// for a unit's saved Review reports, e.g. "review-week",
// "review-month". Distinct from other existing kinds ("dsu", "som")
// already in use by standup.go/som.go.
func reviewReportKind(period summaryPeriod) string {
	return "review-" + strings.ToLower(string(period))
}

// reviewReportPath returns the save path for period's Review report
// covering the nominal unit containing anchor, themed for theme --
// e.g. "review-month-202609-status_report.md". Including theme in
// the filename (added alongside the period-picker/mid-period work,
// see docs/kickoff-review-design.md) lets multiple differently-themed
// reports coexist for the same period instead of one overwriting the
// other, and lets listReviewReportsForPeriod tell them apart.
func reviewReportPath(period summaryPeriod, anchor time.Time, theme string) string {
	token := reviewReportDateToken(period, anchor)
	if theme != "" {
		token += "-" + theme
	}
	return periodReportPathRaw(reviewReportKind(period), token)
}

// listReviewReportsForPeriod returns the saved Review report paths
// (with their theme, parsed back out of the filename) whose nominal
// range exactly matches the unit containing anchor -- used by the
// period-picker/Review window to show "a report already exists for
// this period" plus which theme(s), before the user taps Generate
// again. Unlike listReviewReportsOverlapping (used for the rollup,
// which intentionally wants loose overlap across sub-tier
// boundaries), this is an exact match against periodNominalRange
// since it's answering "does *this specific* period already have a
// saved report", not "what covers part of this range".
func listReviewReportsForPeriod(period summaryPeriod, anchor time.Time) (paths []string, themes []string) {
	pattern := periodReportPathRaw(reviewReportKind(period), "*")
	matches, _ := filepath.Glob(pattern)
	prefix := reviewReportKind(period) + "-"
	wantFrom, wantTo := periodNominalRange(period, anchor)
	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), ".md")
		rest := strings.TrimPrefix(name, prefix)
		// rest is "<dateToken>" or "<dateToken>-<theme>" -- theme
		// names (ThemePersonalNotes etc.) never contain "-" so a
		// single split on the first "-" after the date token would
		// be ambiguous for periodQuarter's "2026Q3" token (no dash)
		// but safe in general; instead, try stripping each known
		// theme suffix explicitly.
		token, foundTheme := rest, ""
		for _, th := range themeDisplayOrder {
			if suffix := "-" + th; strings.HasSuffix(rest, suffix) {
				token = strings.TrimSuffix(rest, suffix)
				foundTheme = th
				break
			}
		}
		anchorFromToken, ok := reviewReportAnchorFromToken(period, token)
		if !ok {
			continue
		}
		rFrom, rTo := periodNominalRange(period, anchorFromToken)
		if rFrom.Equal(wantFrom) && rTo.Equal(wantTo) {
			paths = append(paths, path)
			themes = append(themes, foundTheme)
		}
	}
	return paths, themes
}

// reviewReportDateToken encodes anchor's nominal unit as the filename
// date component for period's Review report, and reviewReportAnchorFromToken
// reverses it. Day/Week/Month/Year use plain time-parseable formats;
// Quarter doesn't have a time.Parse token so it's encoded/decoded
// manually as e.g. "2026Q3".
func reviewReportDateToken(period summaryPeriod, anchor time.Time) string {
	switch period {
	case periodDay:
		return anchor.Format("20060102")
	case periodWeek:
		return weekStart(anchor).Format("20060102")
	case periodMonth:
		return anchor.Format("200601")
	case periodQuarter:
		return fmt.Sprintf("%dQ%d", anchor.Year(), quarterOf(anchor))
	case periodYear:
		return anchor.Format("2006")
	default:
		return anchor.Format("20060102")
	}
}

// reviewReportAnchorFromToken parses a filename date token (as
// produced by reviewReportDateToken) back into a representative
// anchor time.Time for that period -- enough to feed back into
// periodNominalRange to recover the report's covered range. Returns
// ok=false if token doesn't parse as expected for period.
func reviewReportAnchorFromToken(period summaryPeriod, token string) (t time.Time, ok bool) {
	switch period {
	case periodDay:
		t, err := time.ParseInLocation("20060102", token, time.Local)
		return t, err == nil
	case periodWeek:
		t, err := time.ParseInLocation("20060102", token, time.Local)
		return t, err == nil
	case periodMonth:
		t, err := time.ParseInLocation("200601", token, time.Local)
		return t, err == nil
	case periodQuarter:
		parts := strings.SplitN(token, "Q", 2)
		if len(parts) != 2 {
			return time.Time{}, false
		}
		year, err1 := strconv.Atoi(parts[0])
		q, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || q < 1 || q > 4 {
			return time.Time{}, false
		}
		startMonth := time.Month((q-1)*3 + 1)
		return time.Date(year, startMonth, 1, 0, 0, 0, 0, time.Local), true
	case periodYear:
		t, err := time.ParseInLocation("2006", token, time.Local)
		return t, err == nil
	default:
		return time.Time{}, false
	}
}

// periodReportPathRaw joins DunzoDir()/<kind>-<token>.md directly,
// bypassing periodReportPath's time.Time-based Format call since
// Quarter's token isn't produced via time.Time.Format.
func periodReportPathRaw(kind, token string) string {
	return filepath.Join(DunzoDir(), kind+"-"+token+".md")
}

// listReviewReportsOverlapping returns the saved Review report file
// paths of the given subPeriod kind whose nominal range overlaps
// [from, to] at all (loose -- any overlap counts, see
// docs/kickoff-review-design.md's looseness note), along with each
// found file's own nominal [from, to] range (for gap-detection in
// gatherReviewSourceMaterial).
func listReviewReportsOverlapping(subPeriod summaryPeriod, from, to time.Time) []struct {
	Path     string
	From, To time.Time
} {
	pattern := periodReportPathRaw(reviewReportKind(subPeriod), "*")
	matches, _ := filepath.Glob(pattern)
	var out []struct {
		Path     string
		From, To time.Time
	}
	prefix := reviewReportKind(subPeriod) + "-"
	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), ".md")
		token := strings.TrimPrefix(name, prefix)
		anchor, ok := reviewReportAnchorFromToken(subPeriod, token)
		if !ok {
			continue
		}
		rFrom, rTo := periodNominalRange(subPeriod, anchor)
		if rTo.Before(from) || rFrom.After(to) {
			continue // no overlap at all
		}
		out = append(out, struct {
			Path     string
			From, To time.Time
		}{Path: path, From: rFrom, To: rTo})
	}
	return out
}

// reviewSourceMaterial is what gets fed to the copilot prompt for a
// Review: already-generated sub-tier reports (their saved, possibly
// hand-edited markdown bodies) found to overlap the requested range,
// plus raw ledger text for whatever days aren't covered by one of
// those reports.
type reviewSourceMaterial struct {
	SubReports []string // markdown bodies of covered sub-period reports, in filename order
	RawLedger  string   // ledger text for whatever's left uncovered
}

// gatherReviewSourceMaterial builds the rollup source material for
// period's Review of [from, to] (see docs/kickoff-review-design.md's
// "Hierarchical rollup" section):
//
//  1. Day and Week have no sub-tier (periodConfigs[period].SubPeriod
//     == "") -- always just raw ledger for the whole range.
//  2. Otherwise, find saved sub-tier reports overlapping [from, to]
//     (loosely -- any overlap counts) via
//     listReviewReportsOverlapping, and read their markdown bodies.
//  3. Track the union of days covered by those found sub-reports'
//     own nominal ranges (clamped to [from, to]).
//  4. For any day in [from, to] not covered by a found sub-report,
//     fall back to that day's raw ledger text.
//  5. Return both lists separately so the calling prompt can frame
//     them differently ("Prior summaries:" vs "Additional raw
//     entries not yet summarized:").
//
// No exact interval-math precision is attempted -- occasional overlap
// between a sub-report's coverage and the raw-ledger fallback is
// acceptable (see the loose-padding rationale in period.go); a
// coverage gap is the failure mode this guards against, not
// duplication.
func gatherReviewSourceMaterial(period summaryPeriod, from, to time.Time) reviewSourceMaterial {
	subPeriod := periodConfigs[period].SubPeriod
	if subPeriod == "" {
		return reviewSourceMaterial{RawLedger: gatherLedgerTextForRange(from, to, nil)}
	}

	found := listReviewReportsOverlapping(subPeriod, from, to)

	covered := make(map[string]bool) // "20060102" -> true, for each day covered by a found sub-report
	var subReports []string
	for _, f := range found {
		body, err := os.ReadFile(f.Path)
		if err != nil || strings.TrimSpace(string(body)) == "" {
			continue
		}
		subReports = append(subReports, string(body))
		day := f.From
		for !day.After(f.To) && !day.After(to) {
			if !day.Before(from) {
				covered[day.Format("20060102")] = true
			}
			day = day.AddDate(0, 0, 1)
		}
	}

	var gaps []string
	day := from
	for !day.After(to) {
		if !covered[day.Format("20060102")] {
			gaps = append(gaps, day.Format("20060102"))
		}
		day = day.AddDate(0, 0, 1)
	}

	var rawLedger string
	if len(gaps) > 0 {
		// Simplest correct approach: gather each uncovered day
		// individually and concatenate -- gaps are rarely more than
		// a handful of scattered days in practice (partial coverage
		// case), so per-day granularity is fine here rather than
		// trying to collapse them back into contiguous ranges.
		var parts []string
		for _, token := range gaps {
			d, err := time.ParseInLocation("20060102", token, time.Local)
			if err != nil {
				continue
			}
			text := gatherLedgerTextForDate(d)
			if strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		rawLedger = strings.Join(parts, "\n\n")
	}

	return reviewSourceMaterial{SubReports: subReports, RawLedger: rawLedger}
}
