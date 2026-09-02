package dun

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds user-configurable Dunzo preferences, loaded from a TOML
// file at DunzoDir()/config.toml. Ported from the original dunnit zsh
// config-example.zsh (minus dunnits_dir, which is now just DunzoDir()
// itself -- everything dunzo owns lives under one root directory).
type Config struct {
	// DayStart/DayEnd mark roughly when your working day runs, as
	// "HH:MM" 24-hour strings. Used to decide whether hourly popups
	// should fire at all.
	DayStart string `toml:"day_start"`
	DayEnd   string `toml:"day_end"`

	// HourlyMinute is the minute of every hour when the popup should
	// appear (e.g. 58 means it pops at :58 past each hour).
	// Deprecated: replaced by NudgeIntervalMinutes (FR-04). Kept only
	// so old config.toml files with this key don't fail to decode;
	// no longer read by the scheduler.
	HourlyMinute int `toml:"hourly_minute"`

	// NudgeIntervalMinutes is how often (in minutes) the capture
	// nudge fires during work hours, e.g. 30/45/60/90.
	NudgeIntervalMinutes int `toml:"nudge_interval_minutes"`

	// LunchTime is "HH:MM" for a midday goals-reminder popup.
	LunchTime string `toml:"lunch_time"`

	// RecurringMeetings is the FR-15 mini-calendar: a small,
	// purely user-entered list of weekly recurring meeting slots
	// (tag + day-of-week + time), used by FR-16's pre-meeting
	// nudge. No real calendar/.ics/EventKit integration.
	RecurringMeetings []RecurringMeeting `toml:"recurring_meeting"`

	// WeeklyDigestDay/Time (FR-19) configure when the proactive
	// weekly digest nudge fires, e.g. "Friday" "16:00". Empty
	// DigestDay disables the nudge (default: disabled, since it
	// shells out to gh copilot and Micah may not want it firing
	// unprompted until explicitly configured).
	WeeklyDigestDay  string `toml:"weekly_digest_day"`
	WeeklyDigestTime string `toml:"weekly_digest_time"`

	// AutoDraftDailySummary (FR-18) gates whether EOD's Finalize Day
	// automatically drafts/opens the daily summary doc via gh
	// copilot. Default false -- open design questions remain about
	// whether EOD is even the right trigger timing, and what should
	// differentiate the doc's content from Summarize's existing Day
	// output (see FR-18 questions in
	// docs/open-design-questions.md). Manual drafting via the "Daily
	// Summary Doc..." tray item is unaffected by this flag.
	AutoDraftDailySummary bool `toml:"auto_draft_daily_summary"`

	// SnoozeMinutes (FR-26) is the default duration used by the
	// "Snooze" action (both Daybook's button and the tray menu's
	// default top-level item). Defaults to 15 if unset/invalid. The
	// tray menu also offers a few fixed alternatives (15/30/60) via a
	// submenu regardless of this setting.
	SnoozeMinutes int `toml:"snooze_minutes"`

	// DoNotDisturb (FR-27) is a manual on/off flag the user toggles
	// from the tray menu -- while true, the periodic capture nudge
	// (sched.go) is suppressed entirely, same as an indefinite
	// Snooze. Kept simple deliberately: no OS-level DND/screen-share
	// detection (there's no stable public API for that on macOS, and
	// it was decided not to build a heuristic file-based guess
	// either) -- just a manual toggle the user flips themselves.
	// Persisted so it survives app restarts. Only gates the periodic
	// nudge, not SOD/EOD/meeting nudges, same scope as Snooze.
	DoNotDisturb bool `toml:"do_not_disturb"`

	// RecurringItems is the recurring-TODO/GOAL feature (see
	// RECURRING-ITEMS-DESIGN-SEED.md): a small, hand-maintained list
	// of items to be suggested (not auto-seeded) on a daily/weekly/
	// monthly cadence. Daily/weekly are surfaced in SOD; monthly in
	// SOM. Managed via showRecurringItemsDialog (recurring.go).
	RecurringItems []RecurringItem `toml:"recurring_item"`

	// SkipUSFederalHolidays, when true, treats the 11 US federal
	// holidays (see holidays.go's isUSFederalHoliday) the same as a
	// weekend day -- no hourly/lunch/SOD/EOD nudges. Default false
	// (opt-in), toggled via Settings.
	SkipUSFederalHolidays bool `toml:"skip_us_federal_holidays"`

	// KickoffEnabled/ReviewEnabled gate each of the 5 units'
	// Kickoff/Review surfaces independently (see
	// docs/kickoff-review-design.md) -- the primary lever for "5
	// units is a lot to ask": a unit's toggle off means it's never
	// shown, automatically or via the Kickoff.../Review... tray
	// submenus. Day/Week/Month default on (they cover today's
	// existing SOD/EOD/SOM); Quarter/Year default off since they're
	// new, no-prior-art surfaces a user should opt into.
	KickoffDayEnabled     bool `toml:"kickoff_day_enabled"`
	KickoffWeekEnabled    bool `toml:"kickoff_week_enabled"`
	KickoffMonthEnabled   bool `toml:"kickoff_month_enabled"`
	KickoffQuarterEnabled bool `toml:"kickoff_quarter_enabled"`
	KickoffYearEnabled    bool `toml:"kickoff_year_enabled"`

	ReviewDayEnabled     bool `toml:"review_day_enabled"`
	ReviewWeekEnabled    bool `toml:"review_week_enabled"`
	ReviewMonthEnabled   bool `toml:"review_month_enabled"`
	ReviewQuarterEnabled bool `toml:"review_quarter_enabled"`
	ReviewYearEnabled    bool `toml:"review_year_enabled"`

	// ThemeDay/Week/Month/Quarter/Year hold each unit's standing
	// default Review theme (one of the Theme* constants in
	// period.go) -- individual Review invocations may still override
	// this just for that one instance via a dropdown on the dialog,
	// without changing this stored default.
	ThemeDay     string `toml:"theme_day"`
	ThemeWeek    string `toml:"theme_week"`
	ThemeMonth   string `toml:"theme_month"`
	ThemeQuarter string `toml:"theme_quarter"`
	ThemeYear    string `toml:"theme_year"`

	// ExtendWorkWeekTo7Days, when true, shows the full Mon-Sun 7-day
	// span in Week Kickoff/Review labels instead of the default
	// Mon-Fri 5-day work week. Default false (5-day), toggled via
	// Settings. Display-only -- Week's actual data-gathering range
	// (periodNominalRange/periodDataRange for periodWeek) always
	// covers the full Mon-Sun week regardless of this setting, so a
	// weekend ledger entry is never silently excluded from a Week
	// Review just because the label says "Mon-Fri".
	ExtendWorkWeekTo7Days bool `toml:"extend_work_week_to_7_days"`
}

// defaultConfig mirrors the values from dunnit's config-example.zsh.
func defaultConfig() Config {
	return Config{
		DayStart:             "08:00",
		DayEnd:               "17:30",
		NudgeIntervalMinutes: 60,
		LunchTime:            "11:30",
		SnoozeMinutes:        15,

		// Day/Week/Month default on (they cover today's existing
		// SOD/EOD/SOM); Quarter/Year default off, opt-in, since
		// they're new surfaces with no prior art (see
		// docs/kickoff-review-design.md).
		KickoffDayEnabled:     true,
		KickoffWeekEnabled:    true,
		KickoffMonthEnabled:   true,
		KickoffQuarterEnabled: false,
		KickoffYearEnabled:    false,

		ReviewDayEnabled:     true,
		ReviewWeekEnabled:    true,
		ReviewMonthEnabled:   true,
		ReviewQuarterEnabled: false,
		ReviewYearEnabled:    false,

		ThemeDay:     ThemePersonalNotes,
		ThemeWeek:    ThemePersonalNotes,
		ThemeMonth:   ThemeStatusReport,
		ThemeQuarter: ThemeFormalReport,
		ThemeYear:    ThemeFormalReport,
	}
}

// DunzoDir is the single root directory for everything dunzo owns:
// ledger files (DunzoDir()/<year>/w<week>-<month>/ledger-*.txt) and
// config.toml. Overridable via the DUNZO_DIR env var; defaults to
// ~/.config/dunzo.
func DunzoDir() string {
	if dir := os.Getenv("DUNZO_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dunzo")
}

func configPath() string {
	return filepath.Join(DunzoDir(), "config.toml")
}

// LoadConfig reads config.toml, creating it with defaults on first run
// if it doesn't exist yet.
func LoadConfig() Config {
	cfg := defaultConfig()
	path := configPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(DunzoDir(), 0755); err != nil {
			log.Println("Error creating dunzo dir:", err)
			return cfg
		}
		if err := writeConfig(cfg); err != nil {
			log.Println("Error writing default config:", err)
		}
		return cfg
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		log.Println("Error reading config, using defaults:", err)
		return defaultConfig()
	}
	return cfg
}

func writeConfig(cfg Config) error {
	f, err := os.Create(configPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
