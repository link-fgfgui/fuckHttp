// Package browser finds the browsers installed on the machine and opens URLs
// in them.
package browser

import (
	"errors"
	"os/exec"
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
