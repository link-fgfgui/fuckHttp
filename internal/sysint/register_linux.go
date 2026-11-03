//go:build linux

// Package sysint registers the app as a browser with the desktop.
package sysint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	appName   = "fuckHttp"
	desktopID = "fuckhttp.desktop"
)

// mimeKeys are the types the app takes over, the schemes a browser opens.
var mimeKeys = []string{
	"x-scheme-handler/http",
	"x-scheme-handler/https",
	"text/html",
}

// IsElevated reports whether the process runs as root. The app never needs to:
// everything it writes lives in the user's own directories.
func IsElevated() bool {
	return os.Geteuid() == 0
}

// Register installs a desktop entry and makes it the default browser, as the
// Desktop Entry, MIME Applications and Base Directory specifications say.
func Register() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	applications := filepath.Join(dataHome(), "applications")
	if err := os.MkdirAll(applications, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(applications, desktopID), []byte(desktopEntry(exe)), 0o644); err != nil {
		return err
	}
	if err := setMimeDefaults(desktopID); err != nil {
		return err
	}

	// Both are best effort: the files above are what the specification reads.
	run("update-desktop-database", applications)
	run("xdg-settings", "set", "default-web-browser", desktopID)
	return nil
}

// Unregister removes the desktop entry and the associations with it.
func Unregister() error {
	applications := filepath.Join(dataHome(), "applications")
	if err := os.Remove(filepath.Join(applications, desktopID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := clearMimeDefaults(desktopID); err != nil {
		return err
	}
	run("update-desktop-database", applications)
	return nil
}

// IsRegistered reports whether the desktop entry is installed.
func IsRegistered() bool {
	_, err := os.Stat(filepath.Join(dataHome(), "applications", desktopID))
	return err == nil
}

// EnsureRegistered installs the desktop entry; no elevation is needed.
func EnsureRegistered() error {
	return Register()
}

// EnsureUnregistered removes the desktop entry; no elevation is needed.
func EnsureUnregistered() error {
	return Unregister()
}

// desktopEntry is the contents of the app's .desktop file.
func desktopEntry(exe string) string {
	var b strings.Builder
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Version=1.0\n")
	b.WriteString("Name=" + appName + "\n")
	b.WriteString("GenericName=Browser Selector\n")
	b.WriteString("Comment=A browser selector for intercepted URLs\n")
	b.WriteString("Exec=" + quoteExec(exe) + " %u\n")
	if icon := filepath.Join(filepath.Dir(exe), "resources", "icon.png"); fileExists(icon) {
		b.WriteString("Icon=" + icon + "\n")
	}
	b.WriteString("Terminal=false\n")
	b.WriteString("Categories=Network;WebBrowser;\n")
	b.WriteString("MimeType=" + strings.Join(mimeKeys, ";") + ";\n")
	b.WriteString("StartupNotify=false\n")
	return b.String()
}

// quoteExec quotes a program path for the Exec key.
func quoteExec(path string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"`", "\\`",
		"$", "\\$",
	)
	return "\"" + replacer.Replace(path) + "\""
}

// setMimeDefaults makes id the default application of the keys.
func setMimeDefaults(id string) error {
	lines, err := readLines(mimeAppsPath())
	if err != nil {
		return err
	}
	for _, key := range mimeKeys {
		lines = setInGroup(lines, "Default Applications", key, id+";")
	}
	return writeLines(mimeAppsPath(), lines)
}

// clearMimeDefaults removes id from the defaults, leaving the other applications.
func clearMimeDefaults(id string) error {
	lines, err := readLines(mimeAppsPath())
	if err != nil {
		return err
	}
	for _, key := range mimeKeys {
		lines = removeFromGroup(lines, "Default Applications", key, id)
	}
	return writeLines(mimeAppsPath(), lines)
}

// setInGroup sets key=value in group, adding the group when it is missing.
func setInGroup(lines []string, group, key, value string) []string {
	start, end, found := groupRange(lines, group)
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		return append(lines, "["+group+"]", key+"="+value)
	}
	for i := start; i < end; i++ {
		if strings.HasPrefix(lines[i], key+"=") {
			lines[i] = key + "=" + value
			return lines
		}
	}
	lines = append(lines, "")
	copy(lines[end+1:], lines[end:])
	lines[end] = key + "=" + value
	return lines
}

// removeFromGroup removes id from a semicolon separated value, and the key when
// nothing is left.
func removeFromGroup(lines []string, group, key, id string) []string {
	start, end, found := groupRange(lines, group)
	if !found {
		return lines
	}
	for i := start; i < end; i++ {
		if !strings.HasPrefix(lines[i], key+"=") {
			continue
		}
		kept := make([]string, 0, 4)
		for _, value := range strings.Split(strings.TrimPrefix(lines[i], key+"="), ";") {
			value = strings.TrimSpace(value)
			if value != "" && value != id {
				kept = append(kept, value)
			}
		}
		if len(kept) == 0 {
			return append(lines[:i], lines[i+1:]...)
		}
		lines[i] = key + "=" + strings.Join(kept, ";") + ";"
		return lines
	}
	return lines
}

// groupRange returns where a group's keys start and end.
func groupRange(lines []string, group string) (start, end int, found bool) {
	for i, line := range lines {
		if strings.TrimSpace(line) != "["+group+"]" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "[") {
				return i + 1, j, true
			}
		}
		return i + 1, len(lines), true
	}
	return 0, 0, false
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n"), nil
}

func writeLines(path string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func dataHome() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func configHome() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func mimeAppsPath() string {
	return filepath.Join(configHome(), "mimeapps.list")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// run runs a helper command and ignores what it says.
func run(name string, args ...string) {
	_ = exec.Command(name, args...).Run()
}
