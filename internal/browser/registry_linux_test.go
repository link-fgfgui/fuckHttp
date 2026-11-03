//go:build linux

package browser

import (
	"os"
	"path/filepath"
	"testing"
)

// writeEntry installs a desktop entry in dir.
func writeEntry(t *testing.T, dir, id, name, exec string) {
	t.Helper()
	body := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=" + name + "\n" +
		"Exec=" + exec + "\n" +
		"Categories=Network;WebBrowser;\n"
	if err := os.WriteFile(filepath.Join(dir, id), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledSkipsSelf(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dataHome, "system"))

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	applications := filepath.Join(dataHome, "applications")
	if err := os.MkdirAll(applications, 0o755); err != nil {
		t.Fatal(err)
	}
	writeEntry(t, applications, "fuckhttp.desktop", "fuckHttp", "\""+self+"\" %u")
	writeEntry(t, applications, "other.desktop", "Other Browser", "/usr/bin/other-browser %u")

	browsers, err := Installed()
	if err != nil {
		t.Fatal(err)
	}
	if len(browsers) != 1 {
		t.Fatalf("got %d browsers, want 1: %+v", len(browsers), browsers)
	}
	if browsers[0].Name != "Other Browser" {
		t.Errorf("got %q, want Other Browser", browsers[0].Name)
	}
}

func TestInstalledReadsDesktopEntries(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dataHome, "system"))

	applications := filepath.Join(dataHome, "applications")
	if err := os.MkdirAll(applications, 0o755); err != nil {
		t.Fatal(err)
	}
	writeEntry(t, applications, "browser.desktop", "Some Browser", "/usr/bin/some-browser --new-window %u")
	// Hidden and NoDisplay entries are not offered.
	hidden := "[Desktop Entry]\nType=Application\nName=Hidden Browser\nExec=/usr/bin/hidden %u\n" +
		"Categories=Network;WebBrowser;\nNoDisplay=true\n"
	if err := os.WriteFile(filepath.Join(applications, "hidden.desktop"), []byte(hidden), 0o644); err != nil {
		t.Fatal(err)
	}
	// Entries that are not browsers are left out.
	notBrowser := "[Desktop Entry]\nType=Application\nName=Editor\nExec=/usr/bin/editor %f\n" +
		"Categories=Utility;TextEditor;\n"
	if err := os.WriteFile(filepath.Join(applications, "editor.desktop"), []byte(notBrowser), 0o644); err != nil {
		t.Fatal(err)
	}

	browsers, err := Installed()
	if err != nil {
		t.Fatal(err)
	}
	if len(browsers) != 1 || browsers[0].Name != "Some Browser" {
		t.Fatalf("got %+v, want only Some Browser", browsers)
	}
	if browsers[0].Command != "/usr/bin/some-browser --new-window %u" {
		t.Errorf("command = %q", browsers[0].Command)
	}
}
