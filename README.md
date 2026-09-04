# Dunzo

Dunzo is a KISS daily activity tracker: an hourly popup asks "what are
you working on?", and your answer gets appended to a timestamped
ledger file. Over a day/week/month this builds a factual record of
what you did — handy for standups, status reports, and reviews.

This is a Go/Fyne rewrite of the original
[dunnit](https://github.com/MicahElliott/dunnit) zsh proof-of-concept,
which relied on macOS-only tools (`terminal-notifier`, `alerter`) that
have since bit-rotted. Dunzo aims to be cross-platform (macOS + Linux)
and much smaller in scope.

## Guiding Principles

- tiny (20MB desktop app)
- text-based
- mouseless, keyboard driven
- similar to org-mode, but GUI and more intuitive and no Emacs required
- similar in scope to Todo-tracker but totally different approach
- ledger-based, text-only storage
- data can be stored/synced either via 1-table sqlite db or via text files/git
- optional AI-powered reporting
- assistance/automation for all writing of meeting prep and minutes, status reports,
  quarterly/annual reviews

## Status

Early and rough. The core loop (record an entry, browse today's
entries, edit the raw ledger, see goals) works. The scheduled hourly
popup and day-start/day-end prompts are not yet wired up — see
`dunnit/sched.go` for the scaffolding.

## Building & Running

Requires Go 1.23+.

```sh
make build   # -> ./dunzo
make run     # build + run directly (shows in terminal, generic icon)
make vet
```

On macOS, for a proper `.app` bundle with a real icon (so Cmd-Tab and
the Dock show "Dunzo" instead of a terminal window), install the Fyne
packaging tool once:

```sh
go install fyne.io/tools/cmd/fyne@latest
```

Then:

```sh
make package     # -> Dunzo.app
open Dunzo.app
```

## Data Storage

Dunzo keeps everything (ledger files and `config.toml`) under a
single root directory, `~/.config/dunzo` by default, overridable
with the `DUNZO_DIR` env var (e.g. point it at a private git repo you
sync across machines).

Ledger files, one per day:

```
$DUNZO_DIR/<year>/w<week>-<month>/ledger-<YYYYMMDD>.txt
```

Each line looks like:

```
[14:36] DONE Added string splitting for categories #dunnit
```

`$DUNZO_DIR` is expected to be (or contain) a git repo (e.g. a private
`mydunnits` repo) so your history syncs across machines, mirroring the
original dunnit setup.

## Configuration

`$DUNZO_DIR/config.toml` is created automatically on first run with
these defaults (ported from dunnit's `config-example.zsh`):

```toml
day_start     = "08:00"
day_end       = "17:30"
hourly_minute = 58
lunch_time    = "11:30"
```

- `day_start` / `day_end`: your typical working hours, used to decide
  whether hourly popups should fire at all.
- `hourly_minute`: minute-of-the-hour the popup should appear.
- `lunch_time`: when a midday goals-reminder should show.

## Editing the Ledger

The "Edit Dunnits" button opens today's ledger file in `$EDITOR` if
set (flags are supported, e.g. `EDITOR="emacsclient -c"`), otherwise
falls back to the OS default opener (`open` on macOS, `xdg-open` on
Linux).

## History / Prior Art

See `../dunnit/README.md` and `../dunnit/dunnit.zsh` for the original
zsh implementation this project is modeled on — that version has a
lot more built out (weekly objectives, end-of-day summaries, impact
reports, TODOs/blockers, pandoc-generated HTML reports) that Dunzo
doesn't yet have. Treat it as a reference for behavior/conventions
worth porting, not as current working code.
