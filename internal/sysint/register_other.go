//go:build !windows && !linux

package sysint

import "errors"

var errUnsupported = errors.New("registering a browser is not supported on this platform")

// IsElevated reports whether the process runs as administrator.
func IsElevated() bool { return false }

// Register writes the entries that make the system offer the app as a browser.
func Register() error { return errUnsupported }

// Unregister removes everything Register wrote.
func Unregister() error { return errUnsupported }

// IsRegistered reports whether the entries exist.
func IsRegistered() bool { return false }

// EnsureRegistered registers the app, asking for administrator rights when it
// does not have them.
func EnsureRegistered() error { return errUnsupported }

// EnsureUnregistered unregisters the app, asking for administrator rights when
// it does not have them.
func EnsureUnregistered() error { return errUnsupported }
