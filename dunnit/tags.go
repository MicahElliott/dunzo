package dun

import (
	"bufio"
	"math"
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

// tagRecencyHalfLife controls how fast a tag's contribution to
// commonAndRecentTags's score decays with age. Each occurrence's
// weight halves every tagRecencyHalfLife days, so tags stop showing
// up in "common/recent" once they've been unused for a while, no
// matter how many times they were used long ago.
const tagRecencyHalfLife = 14.0

// commonAndRecentTags returns up to limit tags from ledger history,
// ranked by a blended score of frequency and recency -- leaning
// somewhat more toward recent tags than pure frequency would, per
// Micah's preference, rather than pure "most common of all time"
// (which tends to entrench old tags and never surface newer ones).
// Each occurrence of a tag contributes a weight that exponentially
// decays with the age of that occurrence (half-life
// tagRecencyHalfLife days) -- so a tag's score is dominated by its
// recent usage, and heavy historical-but-stale usage fades away
// rather than permanently outranking genuinely recent tags. (Bug fix
// 2026-08-31: the previous version added an unbounded lifetime count
// to a recency bonus capped at 14, so any tag used enough times ever
// would permanently outrank newer tags regardless of staleness.)
func commonAndRecentTags(limit int) []string {
	scores := map[string]float64{}
	now := time.Now()
	for _, path := range allLedgerFiles() {
		date := ledgerFileDate(path)
		if date == nil {
			continue
		}
		daysSince := now.Sub(*date).Hours() / 24
		if daysSince < 0 {
			daysSince = 0
		}
		weight := math.Pow(0.5, daysSince/tagRecencyHalfLife)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			for _, tag := range extractTags(scanner.Text()) {
				scores[tag] += weight
			}
		}
		f.Close()
	}

	type scored struct {
		tag   string
		score float64
	}
	all := make([]scored, 0, len(scores))
	for tag, score := range scores {
		all = append(all, scored{tag: tag, score: score})
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
