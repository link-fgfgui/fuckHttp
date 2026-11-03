// Package browser finds the browsers installed on the machine and opens URLs
// in them.
package browser

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Browser is one entry of the system's browser list.
type Browser struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Hidden  bool   `json:"hidden"`
}

// Executable is the program a Windows browser's command runs: the quoted path,
// or the first word of an unquoted one.
func (b Browser) Executable() string {
	command := strings.TrimSpace(b.Command)
	if command == "" {
		return ""
	}
	if strings.HasPrefix(command, "\"") {
		if end := strings.Index(command[1:], "\""); end >= 0 {
			return command[1 : end+1]
		}
		return command
	}
	if fields := strings.Fields(command); len(fields) > 0 {
		return fields[0]
	}
	return command
}

// Launch opens target in the browser, without waiting for it.
func (b Browser) Launch(target string) error {
	executable, args := b.launchCommand(target)
	if executable == "" {
		return errors.New("browser has no command")
	}
	cmd := exec.Command(executable, args...)
	cmd.SysProcAttr = detached()
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// selfExecutable is the running program, with symlinks resolved.
func selfExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

// isSelfExecutable reports whether path is the program the app runs as: the
// app must not offer to open a URL in itself.
func isSelfExecutable(path string) bool {
	if path == "" {
		return false
	}
	self := selfExecutable()
	if self == "" {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return samePath(path, self)
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	return runtime.GOOS == "windows" && strings.EqualFold(a, b)
}

// ApplyHidden marks the browsers whose name is in hidden.
func ApplyHidden(browsers []Browser, hidden []string) {
	for i := range browsers {
		for _, name := range hidden {
			if browsers[i].Name == name {
				browsers[i].Hidden = true
				break
			}
		}
	}
}

// HiddenNames lists the names of the browsers marked hidden.
func HiddenNames(browsers []Browser) []string {
	names := make([]string, 0, len(browsers))
	for _, b := range browsers {
		if b.Hidden {
			names = append(names, b.Name)
		}
	}
	return names
}
