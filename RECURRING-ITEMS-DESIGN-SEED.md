# Dunzo: Recurring TODOs/GOALs -- Design Seed (for next session)

Status: **not started, design-only**. Captured at end of the
2026-08-31/09-01 session so a new session can pick this up fresh
without re-deriving the thinking below.

## Problem

Daybook has one-off TODO/GOAL capture (typed in as needed) and
recurring *meetings* (FR-15's mini-calendar, `[[recurring_meeting]]`
in config.toml), but no concept of a recurring *task* -- something
the user wants reminded of on a repeating cadence (daily/weekly/
monthly) without having to re-type it from scratch each time, and
without it silently getting forgotten if they don't think to log it
that day.

## Design thinking captured this session

- **Expected scale is small.** Micah's own framing: "probably only a
  couple daily/weekly/monthly repeating" items -- this is NOT meant to
  become a general-purpose recurring-task engine. Design for a short,
  hand-maintained list (single digits per cadence), not hundreds.
- **The core value is the reminder, not the tracking.** These are
  "things that can be forgotten if not reminded" -- so the design
  should prioritize surfacing them at the right moment (SOD for daily,
  SOM for monthly, and presumably some weekly equivalent -- note:
  there isn't yet a "Start of Week" nudge/window at all; may need one,
  or piggyback on existing SOD/weekly-digest timing) over building rich
  recurrence-editing UI.
- **Relationship to existing recurring-meeting mini-calendar
  (FR-15/`minicalendar.go`)**: that system already has day-of-week +
  time-of-day config plumbing (`RecurringMeeting` struct,
  `nextOccurrence`/`lastOccurrence`/`dueForPreMeetingNudge` helpers in
  `minicalendar.go`) for *meetings* specifically. A recurring-
  TODO/GOAL system is conceptually adjacent (also config-driven,
  also "seed something into the ledger on a schedule") but almost
  certainly needs a *simpler* recurrence model (daily / weekly-on-day-
  X / monthly-on-day-Y, no time-of-day precision needed since these
  aren't meetings) -- worth deciding whether to reuse/extend
  `RecurringMeeting`-style config plumbing or build a distinct,
  simpler `RecurringItem` type. Leaning toward a distinct simpler type
  given the "just a couple, don't overbuild" framing.
- **SOM's relationship to this feature -- explicitly deferred this
  session**: the original ask was "seed SOM with monthly recurring
  items, so it becomes a month checklist," but Micah said "wait on
  just this item, actually" -- i.e. build the recurring-items feature
  itself first (as its own thing), then revisit whether/how SOM should
  auto-seed from it, rather than backing into the feature via SOM's
  UI. Don't assume SOM-seeding is definitely still wanted in its
  original form once the real feature exists -- ask again once it's
  built.
- **Likely touch points once designed:**
  - `config.go` -- new config section, e.g. `[[recurring_item]]`
    array-of-tables (category: TODO/GOAL, text, cadence: daily/
    weekly/monthly, day-of-week or day-of-month as applicable).
  - `sched.go` -- a new job (or extend SOD's existing daily job) that
    checks recurring items due "today" and seeds them into today's
    ledger (or surfaces them for one-tap confirm/add, TBD -- auto-
    seeding is simpler but risks duplicate-looking ledger noise if the
    user already logged the same thing manually that day; may want a
    dedup check similar to FR-01's "already logged recently" idea, or
    simply key off whether an identical open TODO/GOAL with that exact
    text already exists today).
  - `sod.go`/`som.go` -- likely surfaced/reviewable here once seeded
    (not necessarily a new window of their own).
  - Settings or a new small management window -- some way to
    add/edit/remove recurring items without hand-editing
    `config.toml` (though for "just a couple," hand-editing
    config.toml directly might be an acceptable v1, deferring a GUI
    editor -- worth asking Micah rather than assuming).
- **Open questions to resolve at start of next session:**
  1. Auto-seed into the ledger automatically each due day, vs. surface
     as a suggestion the user explicitly confirms/adds (affects
     append-only ledger cleanliness and dedup complexity)?
  2. Does this need its own GUI (add/edit/remove), or is hand-editing
     `config.toml` acceptable for v1 given the expected small scale?
  3. Is a "Start of Week" concept needed for the weekly cadence, or
     does weekly-recurring just piggyback on the existing daily SOD
     nudge (checking "is today the configured day-of-week for this
     item")?
  4. Revisit the original "seed SOM as a month checklist" idea once
     the base feature exists -- still wanted, and if so in what shape?

## Explicitly NOT decided yet

No code has been written for this feature. No `RecurringItem` type,
config key, or scheduling logic exists yet as of this doc. Next
session should start with these open questions, get quick answers,
then design + implement.
