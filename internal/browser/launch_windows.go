//go:build windows

package browser

import "syscall"

// detached starts the browser without a console window of its own.
func detached() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}

// launchCommand is the program and arguments that open target.
func (b Browser) launchCommand(target string) (string, []string) {
	return b.Executable(), []string{target}
}
