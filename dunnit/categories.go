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
	// "now" (day-to-day capture, shown by default), "plan"
	// (future-facing), or "reflect" (retrospective, best suited for
	// End of Day wrap-up). Purely a UI convenience -- doesn't affect
	// what's written to the ledger.
	Group string
	// Sentiment is "positive", "negative", or "" (neutral), used to
	// color-code the Category Legend (dark green / dark red / default).
	Sentiment string
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
		return "Now -- day-to-day capture"
	case "plan":
		return "Plan -- future-facing"
	case "reflect":
		return "Reflect -- retrospective, best captured during End of Day wrap-up"
	}
	return group
}

// Categories is the full ordered list of categories offered in
// Daybook's picker, most common/important first within each group
// (DONE, TODO, IDEA, QUESTION, TIL, MEETING lead "now"), negative-
// sentiment categories placed last within their group. `MTG` was
// dropped in favor of `MEETING` only (FR-05); `BLOCKER`/`BLOCKED` was
// replaced by `WAITING` (FR-03).
var Categories = []Category{
	// now: day-to-day capture
	{"✔️", "DONE", "Something you completed.", "now", "positive"},
	{"⏩", "ONGOING", "Still working on something (e.g. what \"Ditto\" now logs) -- not finished yet.", "now", ""},
	{"💡", "IDEA", "A new idea worth capturing.", "now", ""},
	{"❓", "QUESTION", "An open question to follow up on.", "now", ""},
	{"🧠", "TIL", "Today I Learned -- something new you picked up.", "now", "positive"},
	{"📅", "MEETING", "Scratch agenda-builder notes for an upcoming meeting (tag-scoped).", "now", ""},
	{"⏳", "WAITING", "Blocked on someone/something else; not actionable right now.", "now", ""},
	{"🙌", "KUDOS", "Recognition given to someone else, or received from someone else.", "now", "positive"},
	{"🏆", "WIN", "A win worth celebrating.", "now", "positive"},
	{"🔧", "FIXME", "Something broken that needs fixing.", "now", "negative"},
	{"⚠️", "RISK", "A risk worth flagging/tracking.", "now", "negative"},

	// plan: future-facing
	{"📌", "TODO", "A small, tight, near-term item -- actively encouraged.", "plan", ""},
	{"🎯", "GOAL", "A bigger overarching aim, reviewed on a longer cadence (not daily).", "plan", ""},
	{"🕰️", "SOMEDAY", "Something you might want to do eventually, not now (also where stalled TODOs/GOALs land).", "plan", ""},
	{"🏎️", "OPTIMIZE", "Something working but worth improving/speeding up.", "plan", ""},

	// reflect: retrospective, best suited for End of Day wrap-up
	{"💥", "IMPACT", "Notable impact of your work.", "reflect", "positive"},
	{"🏁", "MILESTONE", "A significant milestone reached.", "reflect", "positive"},
	{"💼", "CAREER", "A big, resume/CV-worthy accomplishment or realization -- not a plan, a retrospective note that something huge happened.", "reflect", "positive"},
	{"🔚", "SUMMARY", "A wrap-up/summary note (typically written via End of Day).", "reflect", ""},
	{"📈", "PRODUCTIVITY", "A note on your own productivity/efficiency (typically written via End of Day).", "reflect", ""},
	{"❌", "FAIL", "Something that didn't go as hoped.", "reflect", "negative"},
	{"🗑️", "WASTED", "Time/effort that felt wasted.", "reflect", "negative"},
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
// filter buttons.
func CategoryLabelsForGroup(group string) []string {
	if group == "" || group == "all" {
		return CategoryLabels()
	}
	var labels []string
	for _, c := range Categories {
		if c.Group == group {
			labels = append(labels, c.Label())
		}
	}
	return labels
}
