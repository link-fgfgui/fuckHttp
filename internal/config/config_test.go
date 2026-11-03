package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TEMP", dir)

	want := Config{HiddenBrowsers: []string{"Firefox", "Safari"}}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "browser_selector_config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config not written to %s: %v", path, err)
	}

	got := Load()
	if len(got.HiddenBrowsers) != 2 || got.HiddenBrowsers[0] != "Firefox" {
		t.Errorf("Load() = %v, want %v", got.HiddenBrowsers, want.HiddenBrowsers)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("TEMP", t.TempDir())

	if got := Load(); len(got.HiddenBrowsers) != 0 {
		t.Errorf("Load() = %v, want empty", got.HiddenBrowsers)
	}
}
