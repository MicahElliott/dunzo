package dun

// Category defines one selectable Daybook category: its short code
// (written verbatim into ledger lines) and the emoji-prefixed label
// shown in the picker UI. This is the single source of truth for the
// category list (FR-03) -- other code (category picker, legend UI in
// FR-06) should build off Categories rather than hardcoding its own
// copy.
type Category struct {
	Emoji string
	Code  string
	// Help is a one-line description of intended use, surfaced by
	// FR-06's in-app legend.
	Help string
	// Group buckets categories for the picker's quick-filter buttons:
	// "now" (day-to-day capture, including the DONE/FAIL/WASTED
	// endpoints that "plan" items resolve into), "plan" (future-
	// facing, open/tracked items), or "reflect" (freestanding
	// notable-event markers, not tied to resolving any specific
	// "plan" item -- see docs/category-taxonomy.md). Purely a UI
	// convenience -- doesn't affect what's written to the ledger.
	Group string
	// Sentiment is "positive", "negative", or "" (neutral), used to
	// color-code the Category Legend (dark green / dark red / default).
	Sentiment string
	// EODOnly marks a category as written only via a dedicated flow
	// (EOD's Finalize Day, SOM's wizard) rather than picked by hand
	// in Daybook's live category picker -- e.g. IMPACT/SUMMARY/
	// MEETING_HOURS are always recordActivity'd directly by eod.go/
	// som.go with a fixed code, never selected from the dropdown.
	// Still included in Categories (and thus the Help legend,
	// annual review scans, etc) for documentation purposes -- only
	// excluded from CategoryLabelsForGroup's picker options.
	EODOnly bool
}

// timeTrackableCategories are the codes for which the optional "mins"
// field (see BuildMainWindow) makes the most sense -- temporal/
// effort-bearing entries. Not enforced strictly; just used to hint in
// the UI.
var timeTrackableCategories = map[string]bool{
	"DONE": true, "ONGOING": true, "TODO": true, "FAIL": true,
	"WASTED": true, "MEETING": true, "WAITING": true,
}

// IsTimeTrackable reports whether mins tracking is conventionally
// meaningful for this category code.
func IsTimeTrackable(code string) bool {
	return timeTrackableCategories[code]
}

// Label returns the picker-facing string, e.g. "✔️ DONE".
func (c Category) Label() string {
	return c.Emoji + " " + c.Code
}

// GroupLabel returns a display name + description for a group, used
// as section headers in the Category Legend.
func GroupLabel(group string) string {
	switch group {
	case "now":
		return "Now — day-to-day capture, including how open (\"Plan\") items resolve (DONE/FAIL/WASTED)"
	case "plan":
		return "Plan — future-facing, open items tracked toward a resolution to DONE in \"Now\""
	case "reflect":
		return "Reflect — freestanding notable-event markers, not tied to resolving any specific Plan item; best captured during End of Day wrap-up"
	}
	return group
}

// Categories is the full ordered list of categories offered in
// Daybook's picker, most common/important first within each group
// (DONE, TODO, IDEA, QUESTION, TIL, MEETING lead "now"), negative-
// sentiment categories placed last within their group. `MTG` was
// dropped in favor of `MEETING` only (FR-05); `BLOCKER`/`BLOCKED` was
// replaced by `WAITING` (FR-03).
//
// "Endpoints" (2026-09-02 regroup): DONE/FAIL/WASTED are the terminal
// states a "Plan"-group item (TODO/IDEA/GOAL/FIXME/etc.) resolves
// into -- logged in the moment just like any other "Now" capture, not
// retrospective in the "looking back over a stretch of time" sense
// that IMPACT/MILESTONE/CAREER are. Moved FAIL/WASTED here (from
// "reflect") to sit next to DONE for that reason. IMPACT/MILESTONE/
// CAREER stay in "reflect" -- they're freestanding notable-event
// markers that don't resolve any specific open item, a genuinely
// different concept from an endpoint. See docs/category-taxonomy.md
// for the fuller design discussion behind this split, including the
// still-informal "(from TODO)"-style promotion-annotation convention
// (a plain-text marker written by hand when closing a Plan item into
// one of these endpoints -- not yet a structured/enforced mechanism).
var Categories = []Category{
	// now: day-to-day capture, plus the three endpoint categories
	// (DONE/FAIL/WASTED) that Plan-group items resolve into.
	{"✔️", "DONE", "Something you completed. The most common endpoint a \"Plan\" item (TODO/IDEA/GOAL/etc.) resolves into — see docs/category-taxonomy.md.", "now", "positive", false},
	{"⏩", "ONGOING", "Still working on something (e.g. what \"Ditto\" now logs) — not finished yet. Purely an internal/mechanical marker (Ditto's own bookkeeping), not part of the endpoint/promotion taxonomy.", "now", "", false},
	{"🌱", "TIL", "Today I Learned — something new you picked up.", "now", "positive", false},
	{"🙌", "KUDOS", "Recognition given to someone else, or received from someone else.", "now", "positive", false},
	{"🏆", "WIN", "A win worth celebrating.", "now", "positive", false},
	{"❌", "FAIL", "Something that didn't go as hoped — an endpoint a \"Plan\" item can resolve into, same as DONE, just the unsuccessful outcome.", "now", "negative", false},
	{"🗑️", "WASTED", "Time/effort that felt wasted — an endpoint a \"Plan\" item can resolve into, same as DONE, just the unsuccessful outcome.", "now", "negative", false},

	// plan: future-facing -- includes the "open item, needs follow-up
	// or resolution" categories (WAITING/QUESTION/FIXME/RISK moved
	// here from "now", alongside TODO/GOAL, since they share the same
	// pattern: logged now, tracked as open, resolved/reviewed later
	// via SOD/SOM/Daybook's Upcoming list -- not truly "day-to-day
	// capture" like DONE/TIL/etc). IDEA also moved here (from "now")
	// -- it's future-facing/not-yet-actioned just like SOMEDAY, and
	// som.go's step 2 already treats IDEA/SOMEDAY as a matched pair
	// for triage, so grouping them together in the picker too keeps
	// that pairing consistent. TODO leads the group (2026-09-02,
	// moved ahead of IDEA per explicit request) since it's the most
	// common/actionable item in this group.
	{"📌", "TODO", "A small, tight, near-term item — actively encouraged. Roughly Jira's \"Task\": scoped and ready to act on (vs. IDEA, which is the same thing before it's scoped).", "plan", "", false},
	{"💡", "IDEA", "A new idea worth capturing, not yet scoped/ready to act on — an earlier maturity stage of TODO (loosely: an un-scoped Jira \"Story\"), not a different type of item.", "plan", "", false},
	{"🎯", "GOAL", "A bigger overarching aim, reviewed on a longer cadence (not daily). Roughly Jira's \"Epic\": a rollup that TODOs/FIXMEs work toward, not itself a single actionable item.", "plan", "", false},
	{"❓", "QUESTION", "An open question to follow up on.", "plan", "", false},
	{"⏳", "WAITING", "Blocked on someone/something else; not actionable right now.", "plan", "", false},
	{"🔧", "FIXME", "Something broken that needs fixing — roughly Jira's \"Bug\": a genuinely different *type* of item than TODO (fixing vs. building), not just a maturity stage like IDEA is.", "plan", "negative", false},
	{"⚠️", "RISK", "A risk worth flagging/tracking.", "plan", "negative", false},
	{"📅", "MEETING", "Scratch agenda-builder notes for an upcoming meeting (tag-scoped).", "plan", "", false},
	{"🕰️", "SOMEDAY", "Something you might want to do eventually, not now (also where stalled TODOs/GOALs land).", "plan", "", false},
	{"🏎️", "OPTIMIZE", "Something working but worth improving/speeding up.", "plan", "", false},

	// reflect: freestanding notable-event markers -- not tied to
	// resolving any specific "Plan" item (unlike DONE/FAIL/WASTED,
	// which moved to "now" -- see the Categories doc comment above).
	// IMPACT/MILESTONE/CAREER stay pickable by hand (you might want
	// to log one the moment it happens, not just at EOD).
	// SUMMARY/PRODUCTIVITY/MEETING_HOURS are EODOnly: they're always
	// written by eod.go's Finalize Day flow with a fixed value/text,
	// never meaningfully hand-picked mid-day from the dropdown -- day-
	// level meta-notes, arguably a fourth concept of their own (see
	// docs/category-taxonomy.md) but left bundled into "reflect" for
	// now rather than splitting into a new group.
	{"💥", "IMPACT", "Notable impact of your work.", "reflect", "positive", false},
	{"🏁", "MILESTONE", "A significant milestone reached.", "reflect", "positive", false},
	{"💼", "CAREER", "A big, resume/CV-worthy accomplishment or realization — not a plan, a retrospective note that something huge happened.", "reflect", "positive", false},
	{"🔚", "SUMMARY", "A wrap-up/summary note (typically written via End of Day).", "reflect", "", true},
	{"📈", "PRODUCTIVITY", "A note on your own productivity/efficiency (typically written via End of Day).", "reflect", "", true},
	{"🕑", "MEETING_HOURS", "How many hours of meetings you were in today (typically written via End of Day).", "reflect", "", true},
}

// CategoryLabels returns the Label() strings for all Categories, in
// order, for use in widget.NewSelect.
func CategoryLabels() []string {
	labels := make([]string, len(Categories))
	for i, c := range Categories {
		labels[i] = c.Label()
	}
	return labels
}

// CategoryLabelsForGroup returns Label() strings for categories
// matching the given group ("now", "plan", "reflect"), or all
// categories if group is "" or "all". Used by the picker's quick-
// filter buttons. Excludes EODOnly categories (e.g. SUMMARY/
// PRODUCTIVITY/MEETING_HOURS) -- those are only ever written via
// eod.go's Finalize Day flow, not meant to be hand-picked here.
func CategoryLabelsForGroup(group string) []string {
	var labels []string
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		if group == "" || group == "all" || c.Group == group {
			labels = append(labels, c.Label())
		}
	}
	return labels
}

// CategoryLabelsForFaves returns Label() strings for the user's
// configured "Faves" bucket (Config.FavoriteCategories, a freely
// user-chosen set of category codes -- unlike Now/Plan/Reflect, which
// are Categories' own fixed Group field, Faves is entirely
// user-defined and can mix codes across groups, e.g. the suggested
// default DONE/TODO/IDEA/FIXME/MEETING). Preserves Categories' overall
// order (not the order codes were added to the config) and silently
// skips any code that isn't a real/current category (e.g. after a
// category is ever renamed/removed) or is EODOnly.
func CategoryLabelsForFaves(cfg Config) []string {
	want := make(map[string]bool, len(cfg.FavoriteCategories))
	for _, code := range cfg.FavoriteCategories {
		want[code] = true
	}
	var labels []string
	for _, c := range Categories {
		if c.EODOnly || !want[c.Code] {
			continue
		}
		labels = append(labels, c.Label())
	}
	return labels
}
