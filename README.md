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

Ledger files live under a `dunnits` directory, one file per day:

```
<dunnits-dir>/<year>/w<week>-<month>/ledger-<YYYYMMDD>.txt
```

Each line looks like:

```
[14:36] DONE Added string splitting for categories #dunnit
```

The directory is expected to be a git repo (e.g. a private
`mydunnits` repo) so your history syncs across machines, mirroring the
original dunnit setup.

## Configuration

Settings live in a TOML file, `~/.config/dunzo/config.toml` by
default (override the directory with `DUNZO_CONFIG_DIR`). It's
created automatically on first run with these defaults (ported
from dunnit's `config-example.zsh`):

```toml
dunnits_dir   = "~/.config/dunzo/mydunnits"
day_start     = "08:00"
day_end       = "17:30"
hourly_minute = 58
lunch_time    = "11:30"
```

- `dunnits_dir`: where ledger files are stored. Can also be
  overridden per-invocation with the `DUNZO_DIR` env var (which takes
  priority over the config file).
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
