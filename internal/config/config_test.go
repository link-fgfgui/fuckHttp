package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolate points the config directories at a temporary home.
func isolate(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(root, "local"))
	t.Setenv("APPDATA", filepath.Join(root, "roaming"))
	return root
}

func TestSaveAndLoad(t *testing.T) {
	isolate(t)

	want := Config{HiddenBrowsers: []string{"Firefox", "Safari"}}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(Path()); err != nil {
		t.Fatalf("config not written to %s: %v", Path(), err)
	}

	got := Load()
	if len(got.HiddenBrowsers) != 2 || got.HiddenBrowsers[0] != "Firefox" {
		t.Errorf("Load() = %v, want %v", got.HiddenBrowsers, want.HiddenBrowsers)
	}
}

func TestLoadMissingFile(t *testing.T) {
	isolate(t)
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("TMP", t.TempDir())
	t.Setenv("TEMP", t.TempDir())

	if got := Load(); len(got.HiddenBrowsers) != 0 {
		t.Errorf("Load() = %v, want empty", got.HiddenBrowsers)
	}
}

func TestLoadMovesLegacyFile(t *testing.T) {
	isolate(t)
	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)

	legacy := filepath.Join(temp, "browser_selector_config.json")
	if err := os.WriteFile(legacy, []byte(`{"hidden_browsers":["Edge"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Load()
	if len(got.HiddenBrowsers) != 1 || got.HiddenBrowsers[0] != "Edge" {
		t.Fatalf("Load() = %v, want [Edge]", got.HiddenBrowsers)
	}
	if _, err := os.Stat(Path()); err != nil {
		t.Errorf("the config was not moved to %s: %v", Path(), err)
	}
}
