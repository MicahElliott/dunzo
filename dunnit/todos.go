package dun

import "strings"

// OpenItem is a not-yet-resolved TODO or GOAL line pulled from
// today's ledger (FR-07), shown together under Daybook's "Upcoming"
// section.
type OpenItem struct {
	Category string // "TODO" or "GOAL"
	Text     string
}

// resolvingCategories are the categories a TODO/GOAL item can be
// "resolved" into from the Upcoming list, and what button triggers
// each: DONE via the "Done" button (actually completed), SOMEDAY via
// the "Postpone" button (deliberately deferred rather than pretending
// it's done -- see FR-07 follow-up). Both leave the original line
// untouched (append-only ledger design) and are recognized by
// parseOpenItems as removing the item from the open/Upcoming list.
var resolvingCategories = []string{"DONE", "SOMEDAY"}

// convertedSuffix marks a resolving line (DONE or SOMEDAY) as having
// been generated from an open TODO/GOAL, so parseOpenItems can
// recognize it and exclude the original from the "open" list. Kept as
// an exact, greppable suffix rather than a separate marker file, to
// stay append-only/plain-text (per project's ledger design).
func convertedSuffix(category string) string {
	return " (via " + category + ")"
}

// parseLedgerLine splits a ledger line "[HH:MM:SS] CATEGORY text"
// into category and text. Returns ok=false if the line doesn't look
// like a well-formed ledger entry.
func parseLedgerLine(line string) (category, text string, ok bool) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// parseOpenItems scans ledger lines for TODO/GOAL entries that have
// not yet been resolved (converted to DONE or postponed to SOMEDAY,
// via recordConvertedDone/recordPostponed), in first-seen order.
func parseOpenItems(lines []string) []OpenItem {
	var open []OpenItem
	resolved := make(map[string]bool) // "CATEGORY\x00text" -> true

	for _, line := range lines {
		cat, text, ok := parseLedgerLine(line)
		if !ok {
			continue
		}
		isResolving := false
		for _, rc := range resolvingCategories {
			if cat == rc {
				isResolving = true
				break
			}
		}
		if isResolving {
			for _, srcCat := range []string{"TODO", "GOAL"} {
				if suffix := convertedSuffix(srcCat); strings.HasSuffix(text, suffix) {
					orig := strings.TrimSuffix(text, suffix)
					resolved[srcCat+"\x00"+orig] = true
				}
			}
			continue
		}
		if cat == "TODO" || cat == "GOAL" {
			open = append(open, OpenItem{Category: cat, Text: text})
		}
	}

	var result []OpenItem
	for _, item := range open {
		if !resolved[item.Category+"\x00"+item.Text] {
			result = append(result, item)
		}
	}
	return result
}

// getOpenItems returns today's open (unresolved) TODO/GOAL items.
func getOpenItems() []OpenItem {
	return parseOpenItems(readLedgerLines())
}

// recordConvertedDone logs a DONE entry referencing an original
// TODO/GOAL item's text, marking it as resolved (FR-07). The
// original line is left untouched (append-only ledger design).
func recordConvertedDone(item OpenItem) {
	recordActivity(item.Text+convertedSuffix(item.Category), "DONE")
}

// recordPostponed logs a SOMEDAY entry referencing an original
// TODO/GOAL item's text, marking it as resolved without pretending it
// was completed -- for deliberately deferring an item so the Upcoming
// list doesn't grow unbounded. The original line is left untouched.
func recordPostponed(item OpenItem) {
	recordActivity(item.Text+convertedSuffix(item.Category), "SOMEDAY")
}
