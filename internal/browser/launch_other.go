//go:build !windows && !linux

package browser

import "syscall"

func detached() *syscall.SysProcAttr {
	return nil
}

// launchCommand is the program and arguments that open target.
func (b Browser) launchCommand(target string) (string, []string) {
	return b.Executable(), []string{target}
}
