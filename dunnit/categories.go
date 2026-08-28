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
	// (future-facing), or "reflect" (retrospective/EOD-ish). Purely a
	// UI convenience -- doesn't affect what's written to the ledger.
	Group string
}

// Label returns the picker-facing string, e.g. "✔️ DONE".
func (c Category) Label() string {
	return c.Emoji + " " + c.Code
}

// Categories is the full ordered list of categories offered in
// Daybook's picker, most common/important first (DONE, TODO, IDEA,
// QUESTION, TIL, MEETING), then the rest roughly by expected
// frequency of use. `MTG` was dropped in favor of `MEETING` only
// (FR-05); `BLOCKER`/`BLOCKED` was replaced by `WAITING` (FR-03).
var Categories = []Category{
	{"✔️", "DONE", "Something you completed.", "now"},
	{"📌", "TODO", "A small, tight, near-term item -- actively encouraged.", "now"},
	{"💡", "IDEA", "A new idea worth capturing.", "now"},
	{"❓", "QUESTION", "An open question to follow up on.", "now"},
	{"🧠", "TIL", "Today I Learned -- something new you picked up.", "now"},
	{"📅", "MEETING", "Scratch agenda-builder notes for an upcoming meeting (tag-scoped).", "now"},
	{"⏳", "WAITING", "Blocked on someone/something else; not actionable right now.", "now"},
	{"🔧", "FIXME", "Something broken that needs fixing.", "now"},
	{"⚠️", "RISK", "A risk worth flagging/tracking.", "now"},
	{"🙌", "KUDOS", "Recognition/praise for someone else's work.", "now"},
	{"🏆", "WIN", "A win worth celebrating.", "now"},
	{"🎯", "GOAL", "A bigger overarching aim, reviewed on a longer cadence (not daily).", "plan"},
	{"🕰️", "SOMEDAY", "Something you might want to do eventually, not now.", "plan"},
	{"🏎️", "OPTIMIZE", "Something working but worth improving/speeding up.", "plan"},
	{"💼", "CAREER", "A career-relevant note -- e.g. resume/CV-worthy accomplishment.", "plan"},
	{"❌", "FAIL", "Something that didn't go as hoped.", "reflect"},
	{"💥", "IMPACT", "Notable impact of your work.", "reflect"},
	{"🏁", "MILESTONE", "A significant milestone reached.", "reflect"},
	{"🗑️", "WASTED", "Time/effort that felt wasted.", "reflect"},
	{"🔚", "SUMMARY", "A wrap-up/summary note (typically written via End of Day).", "reflect"},
	{"📈", "PRODUCTIVITY", "A note on your own productivity/efficiency (typically written via End of Day).", "reflect"},
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
