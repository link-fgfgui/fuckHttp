//go:build linux

package sysint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points the XDG directories at a temporary home, and keeps the helper
// commands from touching the real desktop.
func isolate(t *testing.T) (dataHome, configHome string) {
	t.Helper()
	root := t.TempDir()
	dataHome = filepath.Join(root, "data")
	configHome = filepath.Join(root, "config")
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("PATH", filepath.Join(root, "empty-bin"))
	return dataHome, configHome
}

func TestRegisterAndUnregister(t *testing.T) {
	dataHome, configHome := isolate(t)

	if IsRegistered() {
		t.Fatal("registered before Register")
	}
	if err := Register(); err != nil {
		t.Fatal(err)
	}
	if !IsRegistered() {
		t.Fatal("not registered after Register")
	}

	entry, err := os.ReadFile(filepath.Join(dataHome, "applications", desktopID))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=fuckHttp",
		"Categories=Network;WebBrowser;",
		"x-scheme-handler/http",
	} {
		if !strings.Contains(string(entry), want) {
			t.Errorf("desktop entry lacks %q:\n%s", want, entry)
		}
	}

	mime, err := os.ReadFile(filepath.Join(configHome, "mimeapps.list"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range mimeKeys {
		if !strings.Contains(string(mime), key+"="+desktopID+";") {
			t.Errorf("mimeapps.list lacks %s=%s:\n%s", key, desktopID, mime)
		}
	}

	if err := Unregister(); err != nil {
		t.Fatal(err)
	}
	if IsRegistered() {
		t.Fatal("still registered after Unregister")
	}
	mime, err = os.ReadFile(filepath.Join(configHome, "mimeapps.list"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mime), desktopID) {
		t.Errorf("mimeapps.list still names the app:\n%s", mime)
	}
}

func TestRegisterKeepsOtherEntries(t *testing.T) {
	_, configHome := isolate(t)

	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "[Default Applications]\nx-scheme-handler/http=firefox.desktop;\n"
	if err := os.WriteFile(filepath.Join(configHome, "mimeapps.list"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Register(); err != nil {
		t.Fatal(err)
	}
	mime, err := os.ReadFile(filepath.Join(configHome, "mimeapps.list"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mime), "text/html="+desktopID+";") {
		t.Errorf("text/html default missing:\n%s", mime)
	}
	if strings.Count(string(mime), "[Default Applications]") != 1 {
		t.Errorf("the group was duplicated:\n%s", mime)
	}
}
