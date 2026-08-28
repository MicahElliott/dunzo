package dun

// Category defines one selectable Daybook category: its short code
// (written verbatim into ledger lines) and the emoji-prefixed label
// shown in the picker UI. This is the single source of truth for the
// category list (FR-03) -- other code (category picker, future
// legend/tooltip UI in FR-06) should build off Categories rather than
// hardcoding its own copy.
type Category struct {
	Emoji string
	Code  string
	// Help is a one-line description of intended use, surfaced by
	// FR-06's in-app legend.
	Help string
}

// Label returns the picker-facing string, e.g. "✔️ DONE".
func (c Category) Label() string {
	return c.Emoji + " " + c.Code
}

// Categories is the full ordered list of categories offered in
// Daybook's picker. `MTG` was dropped in favor of `MEETING` only
// (FR-05); `BLOCKER`/`BLOCKED` was replaced by `WAITING` (FR-03).
var Categories = []Category{
	{"✔️", "DONE", "Something you completed."},
	{"🎯", "GOAL", "A bigger overarching aim, reviewed on a longer cadence (not daily)."},
	{"📅", "MEETING", "Scratch agenda-builder notes for an upcoming meeting (tag-scoped)."},
	{"📌", "TODO", "A small, tight, near-term item -- actively encouraged."},
	{"🕰️", "SOMEDAY", "Something you might want to do eventually, not now."},
	{"⏳", "WAITING", "Blocked on someone/something else; not actionable right now."},
	{"🔧", "FIXME", "Something broken that needs fixing."},
	{"⚡", "OPTIMIZE", "Something working but worth improving/speeding up."},
	{"❓", "QUESTION", "An open question to follow up on."},
	{"🙌", "KUDOS", "Recognition/praise for someone else's work."},
	{"⚠️", "RISK", "A risk worth flagging/tracking."},
	{"⛔❌", "FAIL", "Something that didn't go as hoped."},
	{"💥", "IMPACT", "Notable impact of your work."},
	{"📈", "PRODUCTIVITY", "A note on your own productivity/efficiency."},
	{"🔚", "SUMMARY", "A wrap-up/summary note."},
	{"🧠", "TIL", "Today I Learned -- something new you picked up."},
	{"💡", "IDEA", "A new idea worth capturing."},
	{"🏆", "WIN", "A win worth celebrating."},
	{"💼", "CAREER (rare)", "A career-relevant note (used rarely)."},
	{"🗑️", "WASTED", "Time/effort that felt wasted."},
	{"🏁", "MILESTONE", "A significant milestone reached."},
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
