//go:build linux

package browser

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Installed lists the browsers of the desktop entries in the XDG data
// directories, in the order the Base Directory specification searches them.
func Installed() ([]Browser, error) {
	var browsers []Browser
	seenID := make(map[string]bool)
	seenName := make(map[string]bool)

	for _, dir := range applicationDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			id := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(id, ".desktop") || seenID[id] {
				continue
			}
			// A desktop file in a more important directory shadows the ones
			// of the same name below it.
			seenID[id] = true

			fields, ok := readDesktopEntry(filepath.Join(dir, id))
			if !ok {
				continue
			}
			b, ok := browserFromEntry(fields)
			if !ok || seenName[b.Name] || isSelfExecutable(executableOf(b.Command)) {
				continue
			}
			seenName[b.Name] = true
			browsers = append(browsers, b)
		}
	}

	sort.Slice(browsers, func(i, j int) bool {
		return strings.ToLower(browsers[i].Name) < strings.ToLower(browsers[j].Name)
	})
	return browsers, nil
}

// applicationDirs are the applications directories to search, most important
// first: $XDG_DATA_HOME, then $XDG_DATA_DIRS.
func applicationDirs() []string {
	home, _ := os.UserHomeDir()
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}

	dirs := []string{dataHome}
	for _, dir := range strings.Split(dataDirs, ":") {
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}

	applications := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		applications = append(applications, filepath.Join(dir, "applications"))
	}
	return applications
}

// readDesktopEntry reads the [Desktop Entry] group of a desktop file.
func readDesktopEntry(path string) (map[string]string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	fields := make(map[string]string)
	group := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			group = line[1 : len(line)-1]
			continue
		}
		if group != "Desktop Entry" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		fields[strings.TrimSpace(key)] = unescapeDesktopValue(value)
	}
	return fields, len(fields) > 0
}

// browserFromEntry reports whether a desktop entry is a browser, and builds it.
func browserFromEntry(fields map[string]string) (Browser, bool) {
	if fields["Type"] != "Application" {
		return Browser{}, false
	}
	if fields["Hidden"] == "true" || fields["NoDisplay"] == "true" || fields["Terminal"] == "true" {
		return Browser{}, false
	}
	name := fields["Name"]
	command := fields["Exec"]
	if name == "" || command == "" {
		return Browser{}, false
	}
	if !isBrowser(fields) {
		return Browser{}, false
	}
	return Browser{Name: name, Command: command}, true
}

// isBrowser reports whether the entry says it opens web pages: the WebBrowser
// category, or the http scheme in its MIME types.
func isBrowser(fields map[string]string) bool {
	for _, category := range strings.Split(fields["Categories"], ";") {
		if strings.TrimSpace(category) == "WebBrowser" {
			return true
		}
	}
	for _, mime := range strings.Split(fields["MimeType"], ";") {
		switch strings.TrimSpace(mime) {
		case "x-scheme-handler/http", "x-scheme-handler/https":
			return true
		}
	}
	return false
}

// unescapeDesktopValue undoes the escapes of a string value.
func unescapeDesktopValue(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 's':
			b.WriteByte(' ')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
