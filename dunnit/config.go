package dun

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds user-configurable Dunzo preferences, loaded from a TOML
// file. Ported from the original dunnit zsh config-example.zsh.
type Config struct {
	// DunnitsDir is where ledger files (and someday summaries) are
	// stored. Overridable via DUNZO_DIR env var.
	DunnitsDir string `toml:"dunnits_dir"`

	// DayStart/DayEnd mark roughly when your working day runs, as
	// "HH:MM" 24-hour strings. Used to decide whether hourly popups
	// should fire at all.
	DayStart string `toml:"day_start"`
	DayEnd   string `toml:"day_end"`

	// HourlyMinute is the minute of every hour when the popup should
	// appear (e.g. 58 means it pops at :58 past each hour).
	HourlyMinute int `toml:"hourly_minute"`

	// LunchTime is "HH:MM" for a midday goals-reminder popup.
	LunchTime string `toml:"lunch_time"`
}

// defaultConfig mirrors the values from dunnit's config-example.zsh.
func defaultConfig() Config {
	return Config{
		DunnitsDir:   filepath.Join(configDir(), "mydunnits"),
		DayStart:     "08:00",
		DayEnd:       "17:30",
		HourlyMinute: 58,
		LunchTime:    "11:30",
	}
}

// configDir returns the directory holding dunzo's config file,
// defaulting to ~/.config/dunzo, overridable via DUNZO_CONFIG_DIR.
func configDir() string {
	if dir := os.Getenv("DUNZO_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dunzo")
}

func configPath() string {
	return filepath.Join(configDir(), "config.toml")
}

// LoadConfig reads config.toml, creating it with defaults on first run
// if it doesn't exist yet.
func LoadConfig() Config {
	cfg := defaultConfig()
	path := configPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir(), 0755); err != nil {
			log.Println("Error creating config dir:", err)
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
