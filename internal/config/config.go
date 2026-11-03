// Package config stores which browsers the user hid.
//
// The file lives in the user's configuration directory, so that it survives a
// reboot: %APPDATA%\fuckHttp on Windows, ~/.config/fuckHttp on Linux and
// ~/Library/Application Support/fuckHttp on macOS, as os.UserConfigDir defines
// them. A file left in the temp directory by an older build is read once and
// moved.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the app's saved state.
type Config struct {
	HiddenBrowsers []string `json:"hidden_browsers"`
}

// Dir is the directory the config lives in.
func Dir() string {
	return filepath.Join(baseDir(), "fuckHttp")
}

// Path is the config file.
func Path() string {
	return filepath.Join(Dir(), "config.json")
}

// legacyPath is where the Rust version and the first Go builds kept the file.
func legacyPath() string {
	return filepath.Join(os.TempDir(), "browser_selector_config.json")
}

// baseDir is the platform's directory for per-user configuration.
func baseDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return dir
	}
	return "."
}

// Load reads the config, returning an empty one when it is missing or broken.
func Load() Config {
	if cfg, ok := read(Path()); ok {
		return cfg
	}
	// The temp directory is cleared on reboot, so move what is there.
	if cfg, ok := read(legacyPath()); ok {
		_ = Save(cfg)
		return cfg
	}
	return Config{}
}

// Save writes the config.
func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o644)
}

// read loads a config file, reporting whether it existed and parsed.
func read(path string) (Config, bool) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, false
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, false
	}
	return cfg, true
}
