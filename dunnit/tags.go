package dun

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// tagPattern matches a "#tag" token: '#' followed by one or more
// word-ish characters (letters, digits, underscore, hyphen, colon --
// covers things like "#pts:3").
var tagPattern = regexp.MustCompile(`#[\w:-]+`)

// extractTags returns all distinct #tag tokens found in text, in
// first-seen order.
func extractTags(text string) []string {
	seen := make(map[string]bool)
	var tags []string
	for _, m := range tagPattern.FindAllString(text, -1) {
		if !seen[m] {
			seen[m] = true
			tags = append(tags, m)
		}
	}
	return tags
}

// tagCache holds a scanned + deduplicated list of every #tag seen
// across all ledger files under DunzoDir(), refreshed at most every
// tagCacheTTL rather than rescanning on every keystroke (FR-10).
type tagCache struct {
	mu        sync.Mutex
	tags      []string
	scannedAt time.Time
}

const tagCacheTTL = 5 * time.Minute

var globalTagCache tagCache

// KnownTags returns the cached list of all distinct tags seen across
// ledger history, rescanning if the cache is empty or stale.
func KnownTags() []string {
	globalTagCache.mu.Lock()
	defer globalTagCache.mu.Unlock()
	if globalTagCache.tags == nil || time.Since(globalTagCache.scannedAt) > tagCacheTTL {
		globalTagCache.tags = scanAllTags()
		globalTagCache.scannedAt = time.Now()
	}
	return globalTagCache.tags
}

// InvalidateTagCache forces the next KnownTags() call to rescan,
// useful right after recording a new entry that might contain a tag
// not seen before.
func InvalidateTagCache() {
	globalTagCache.mu.Lock()
	defer globalTagCache.mu.Unlock()
	globalTagCache.tags = nil
}

// scanAllTags walks every ledger-*.txt file under DunzoDir() and
// collects every distinct #tag, sorted alphabetically.
func scanAllTags() []string {
	seen := make(map[string]bool)
	for _, path := range allLedgerFiles() {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			for _, tag := range extractTags(scanner.Text()) {
				seen[tag] = true
			}
		}
		f.Close()
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// tagStat tracks how often and how recently a tag has been used,
// for commonAndRecentTags's blended common/recent ranking.
type tagStat struct {
	count    int
	lastSeen time.Time
}

// commonAndRecentTags returns up to limit tags from ledger history,
// ranked by a blended score of frequency and recency -- leaning
// somewhat more toward recent tags than pure frequency would, per
// Micah's preference, rather than pure "most common of all time"
// (which tends to entrench old tags and never surface newer ones).
// Score = count + a recency bonus that decays with days since last
// use (so a tag used once yesterday can outrank one used many times
// months ago, but a heavily-used tag still generally wins over one
// used just once recently).
func commonAndRecentTags(limit int) []string {
	stats := map[string]*tagStat{}
	now := time.Now()
	for _, path := range allLedgerFiles() {
		date := ledgerFileDate(path)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			for _, tag := range extractTags(scanner.Text()) {
				st := stats[tag]
				if st == nil {
					st = &tagStat{}
					stats[tag] = st
				}
				st.count++
				if date != nil && date.After(st.lastSeen) {
					st.lastSeen = *date
				}
			}
		}
		f.Close()
	}

	type scored struct {
		tag   string
		score float64
	}
	var all []scored
	for tag, st := range stats {
		daysSince := 9999.0
		if !st.lastSeen.IsZero() {
			daysSince = now.Sub(st.lastSeen).Hours() / 24
		}
		recencyBonus := 14.0 / (daysSince + 1)
		all = append(all, scored{tag: tag, score: float64(st.count) + recencyBonus})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].score != all[j].score {
			return all[i].score > all[j].score
		}
		return all[i].tag < all[j].tag // stable tiebreak
	})
	if len(all) > limit {
		all = all[:limit]
	}
	out := make([]string, len(all))
	for i, s := range all {
		out[i] = s.tag
	}
	return out
}

// matchingTags returns tags from candidates that contain fragment as
// a case-insensitive substring, with prefix matches (fragment matches
// right after the tag's "#") sorted first, then other substring
// matches -- each group alphabetical. fragment should already have
// its leading "#" stripped. Returns nil if fragment is empty (no
// suggestions until the user has typed something after "#").
func matchingTags(candidates []string, fragment string) []string {
	if fragment == "" {
		return nil
	}
	fragment = strings.ToLower(fragment)
	var prefixMatches, otherMatches []string
	for _, tag := range candidates {
		lower := strings.ToLower(tag)
		body := strings.TrimPrefix(lower, "#")
		switch {
		case strings.HasPrefix(body, fragment):
			prefixMatches = append(prefixMatches, tag)
		case strings.Contains(lower, fragment):
			otherMatches = append(otherMatches, tag)
		}
	}
	if len(prefixMatches) == 0 && len(otherMatches) == 0 {
		return nil
	}
	return append(prefixMatches, otherMatches...)
}

// currentTagFragment inspects text up to cursor (a rune index) and,
// if the cursor is positioned within or immediately after an
// in-progress "#tag" token (i.e. the nearest "#" before the cursor has
// no whitespace between it and the cursor), returns that token's start
// offset and the fragment typed so far (including the "#"). ok is
// false if the cursor isn't in a tag-typing position.
func currentTagFragment(text string, cursor int) (start int, fragment string, ok bool) {
	runes := []rune(text)
	if cursor < 0 || cursor > len(runes) {
		return 0, "", false
	}
	i := cursor - 1
	for i >= 0 && runes[i] != '#' && !isTagBreak(runes[i]) {
		i--
	}
	if i < 0 || runes[i] != '#' {
		return 0, "", false
	}
	return i, string(runes[i:cursor]), true
}

func isTagBreak(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}
