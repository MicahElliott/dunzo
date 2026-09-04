# TODO Carry-Forward Design

Status: informal design notes, not a spec -- same footing as
`docs/category-taxonomy.md`/`docs/navigator-design.md`. Written
2026-09-02 after discovering `getOpenItems()` (Daybook's Planned
section, and SOD) only ever reads *today's* ledger file, so any
TODO/GOAL/WAITING/QUESTION/FIXME/RISK logged on a prior day and never
resolved silently vanishes from Planned the next day -- an oversight,
not an intentional design choice. Read this before changing
open-item scanning, Postpone's behavior, or SOMEDAY's role.

## The problem

`getOpenItems()`/`parseOpenItems()` (`todos.go`) only scan
`readLedgerLines()` -- today's file. There is no cross-day open-item
tracking at all today. This session's new shared ledger index
(`AllLedgerEntries()`, `ledgerindex.go`) makes an all-history scan
technically easy, but a naive "look back N days" approach was
considered and rejected -- see below.

## Rejected: rolling lookback window

Scanning the last N days (e.g. 7) for unresolved items and showing
them all in today's Planned section was considered, but rejected:

- Planned's contents would depend on "what day is it today" relative
  to when each item was logged, silently changing shape as time
  passes with no record of *why* an item is still showing up.
- Every action that currently assumes "today's open items live in
  today's file" (Done/Postpone/Nah, inline Edit, `replaceLedgerLineTextAt`/
  `replaceLedgerLineCategoryAt`) would need to become file-aware
  (which day did this item actually come from), a broader refactor
  for comparatively little benefit.
- Doesn't match the stated goal: "I wanted today's ledger to only
  ever be what user cares about today." A lookback window keeps that
  *false* by construction -- Planned always potentially shows things
  that aren't literally in today's file.

## Chosen approach: copy-forward at first touch of the day

Every morning (see "When it runs" below), unresolved
`openTrackedCategories` items (TODO/GOAL/WAITING/QUESTION/FIXME/RISK)
still open as of yesterday are copied into today's ledger file as
fresh lines, annotated with their original log date:

```
[05:00:00] TODO Finish the report (since 2026-08-28)
```

`(since YYYY-MM-DD)` is a plain, greppable, human-readable annotation
-- parallel to the existing `(via CATEGORY)` convention
(`convertedSuffix`) already used for DONE/SOMEDAY resolution lines,
not a new tag/marker syntax. No `#stale`/`!stale` tag is introduced.

After copy-forward, `getOpenItems()`/`parseOpenItems()` stay exactly
as simple as they are today -- read today's file, done. No
cross-file scanning, no file-aware Done/Postpone/Edit bookkeeping.
This is the main simplicity win over the lookback approach, and it
keeps "today's ledger = today's concerns" true *by construction*
rather than by filtering.

### Staleness display, not staleness syntax

Since each carried-forward line carries its original `(since DATE)`,
Planned can compute `today - since` at render time and show a visual
staleness indicator (e.g. dimmed text, a small "4d" badge) for
anything past some threshold (a handful of days, exact cutoff TBD/
tunable) -- purely a display computation, nothing written to the
ledger beyond the one annotation already there. Nothing for the user
to remember to type, nothing that itself goes stale.

### What "Postpone" means now

Previously ambiguous ("defer to SOMEDAY... but what did that ever
mean, given items already silently vanished after today anyway?").
With automatic daily copy-forward as the default behavior for
anything unresolved, Postpone gets a clear, distinct job: **it's the
explicit opt-out from continued daily carry-forward.** An item stays
in Planned every day by default (that's just "not yet resolved",
requiring no action); Postpone is how a user says "stop showing me
this every morning, park it in the SOMEDAY backlog on purpose,
revisit only if/when I go looking." Nah/Discard remains the sibling
"stop showing me this, and I don't want it back" action. Neither
Postpone nor Discard's mechanics need to change -- they already write
a resolving line (`resolvingCategories`) that today's
`parseOpenItems` recognizes as resolved; the only change is *when*
copy-forward would otherwise have kept re-surfacing the original item
daily, this is what now actually stops that.

No separate "renew a TODO" action is needed -- staying in Planned
*is* the default, unresolved state; there's nothing to renew.

### SOMEDAY becomes ledger-history-only, needs its own browse view

Since Postponed (SOMEDAY) items are deliberately *not* carried
forward, they stop appearing anywhere in the day-to-day UI once
postponed -- only visible via ledger history. This needs a small
dedicated "Browse SOMEDAY items" view (a filtered list, similar in
spirit to Reports Library's browse-by-kind, or just a Navigator
category filter on SOMEDAY) so a postponed item can actually be
found and re-promoted back to an active TODO later (a plain "convert
this SOMEDAY line to a fresh TODO" action, mirroring
`recordConvertedDone`/`recordPostponed`'s shape). Not designed in
detail yet -- flagged as the concrete next follow-up once
copy-forward itself lands. Month Review's existing IDEA/SOMEDAY
triage step (`monthreview.go`) is a related, existing precedent for
"periodically resurface SOMEDAY-ish items" and may end up
superseding or merging with this need -- worth revisiting once both
exist.

## When it runs -- no wizard gate, so it can't be silently skipped

Day Kickoff (`showSODWindow`) is currently just an optional tray menu
item, not something every session passes through -- so tying
copy-forward's correctness to "did the user run Kickoff" would mean
a user who starts logging DONEs first thing (without ever opening
SOD) gets no carry-forward at all, silently. To avoid this:

- Copy-forward runs **automatically and idempotently**, guarded by a
  persisted "last carry-forward date" marker (same pattern as
  `RecurringMeeting`'s `lastOccurrence`) -- not by checking whether
  today's ledger file is empty, since that check breaks in exactly
  this scenario (file is no longer "empty" after the user's first
  manual DONE, even though carry-forward never ran).
- The check fires at the **first natural touchpoint of the day**,
  whichever happens first: Daybook being shown (`refreshOpenItems`/
  `BuildMainWindow`), SOD being opened, or the first
  `recordActivity` call. Whichever comes first performs the
  copy-forward (if not already done today, per the marker) and
  updates the marker. Order doesn't matter for correctness -- a user
  who logs 3 DONEs before ever opening SOD still gets carry-forward
  the moment any of those touchpoints fires, since it's driven by a
  fresh scan of *prior* unresolved items, not by "what's already in
  today's file."
- No modal/wizard-completion gate is needed for correctness. SOD
  keeps its existing role as a nice readback/quick-entry surface, but
  stops being load-bearing for whether carry-forward actually
  happened.
- A small passive, one-time surfaced note ("Carried forward N
  item(s) from earlier") the first time it fires each day is a nice
  touch worth adding, but not required for correctness -- a label,
  not a dialog, so it never blocks/nags.

## Open questions (not resolved, revisit later)

1. ~~Exact staleness display threshold/styling~~ -- implemented as
   `staleDaysThreshold = 4` days, shown as a plain " \u26a0 4d" suffix
   (`staleBadge`, `carryforward.go`) wherever Planned/SOD/Kickoff
   render an open item's text. Threshold/styling are easy to tune
   further if 4 days feels wrong in practice.
2. ~~SOMEDAY browse/re-promote view~~ -- **implemented**
   (`somedaybrowser.go`, "SOMEDAY Items..." under the Ledger tray
   menu, and a "Browse SOMEDAY Items..." button at the bottom of
   Planned's expanded "Show all" view). Lists every still-unhandled
   SOMEDAY item across all ledger history with `-> TODO`/`-> GOAL`/
   `Discard` actions. Reuses the exact same `(via SOMEDAY)` resolution-
   suffix convention as `todos.go`'s `convertedSuffix`/
   `parseOpenItems` (no new marker syntax) -- promoting logs a fresh
   TODO/GOAL entry *and* a `DISCARDED ... (via SOMEDAY)` line so the
   browser stops listing it; Discard alone just logs the latter.
   Cold-start guard for the initial backlog-dump scenario was
   considered but skipped (no plausible way a brand-new user without
   ledger history hits it).
3. Should the "last carry-forward date" marker live in `Config`
   (config.toml) or a separate small state file? **Implemented in
   `Config.LastCarryForwardDate`** (config.go) for now, per
   `RecurringMeeting.lastOccurrence`'s precedent -- worth a second
   look if config.toml ever needs to become more clearly
   "user-editable preferences only."
4. Interaction with Month Kickoff/Review's own IDEA/SOMEDAY triage
   step (`monthreview.go`) -- possible overlap/merge opportunity with
   the SOMEDAY browse view above. Not reconciled yet.

## Implementation status (2026-09-02)

- [x] `Config.LastCarryForwardDate` marker field (`config.go`)
- [x] `carryforward.go`: `priorOpenItems` (all-history scan via
      `AllLedgerEntries`, excluding today, honoring
      `resolvingCategories`/`convertedSuffix` same as
      `todos.go`'s `parseOpenItems`), `runCarryForwardIfNeeded`
      (idempotent per calendar day), `carryForwardSinceSuffix`/
      `parseCarryForwardSince`/`stripCarryForwardSince`, `staleBadge`
- [x] Wired at three touchpoints: `recordActivity` (`ui.go`),
      `BuildMainWindow` (`ui.go`), `showSODWindow` (`sod.go`) -- no
      wizard/dialog-completion gate
- [x] Planned (`ui.go`), SOD (`sod.go`), and generic period Kickoff
      (`periodkickoff.go`) all display `stripCarryForwardSince(...)`
      + `staleBadge(...)` instead of the raw annotated text
- [x] EOD's (`eod.go`) and generic period Review's
      (`periodreview.go`) former "Carry Forward Open TODOs/QUESTIONs"
      checkbox sections repurposed into **Postpone-opt-out** sections
      (default unchecked; checking now calls `recordPostponed`
      instead of the old `carryForwardItem`, which has been removed
      entirely) -- see `eodOpenItemsSection`'s updated doc comment
- [x] Tests: `carryforward_test.go` (copy/skip-resolved/idempotent/
      since-date-preserved/stale-badge/strip-suffix cases),
      `somedaybrowser_test.go` (list/promote/discard cases)
- [ ] Passive "Carried forward N item(s) from earlier" one-time
      surfaced note -- not built, optional per "When it runs" above
- [x] SOMEDAY browse/re-promote view -- `somedaybrowser.go`, see open
      question 2 above for details
