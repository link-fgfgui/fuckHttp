// Package config stores which browsers the user hid.
//
// The file keeps the original name, location and field so that an existing
// installation carries over.
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

// Path is where the config lives.
func Path() string {
	return filepath.Join(os.TempDir(), "browser_selector_config.json")
}

// Load reads the config, returning an empty one when it is missing or broken.
func Load() Config {
	var cfg Config
	data, err := os.ReadFile(Path())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

// Save writes the config.
func Save(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o644)
}
