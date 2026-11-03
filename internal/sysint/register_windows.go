//go:build windows

// Package sysint registers the app as a browser with Windows.
package sysint

import (
	"fmt"
	"os"

	"github.com/arduino/go-windows-runas"
	"golang.org/x/sys/windows/registry"
)

const (
	appName            = "fuckHttp"
	protocolKey        = `Software\Classes\fuckHttp`
	capabilitiesPath   = `Software\fuckHttp\Capabilities`
	registeredAppsPath = `Software\RegisteredApplications`
)

// IsElevated reports whether the process runs as administrator.
func IsElevated() bool {
	elevated, err := runas.IsAdminProcess()
	return err == nil && elevated
}

// Register writes the registry entries that make Windows offer the app as a
// browser. It needs administrator rights for the machine wide keys.
func Register() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	icon := fmt.Sprintf("%q,0", exe)
	command := fmt.Sprintf("%q %q", exe, "%1")

	if err := setString(registry.CURRENT_USER, protocolKey, "", "URL:fuckHttp Protocol"); err != nil {
		return err
	}
	if err := setString(registry.CURRENT_USER, protocolKey, "URL Protocol", ""); err != nil {
		return err
	}
	if err := setString(registry.CURRENT_USER, protocolKey+`\DefaultIcon`, "", icon); err != nil {
		return err
	}
	if err := setString(registry.CURRENT_USER, protocolKey+`\shell\open\command`, "", command); err != nil {
		return err
	}

	if err := setString(registry.LOCAL_MACHINE, capabilitiesPath, "ApplicationName", appName); err != nil {
		return err
	}
	if err := setString(registry.LOCAL_MACHINE, capabilitiesPath, "ApplicationIcon", icon); err != nil {
		return err
	}
	if err := setString(registry.LOCAL_MACHINE, capabilitiesPath, "ApplicationDescription", "A custom browser selector."); err != nil {
		return err
	}
	if err := setString(registry.LOCAL_MACHINE, capabilitiesPath+`\URLAssociations`, "http", appName); err != nil {
		return err
	}
	if err := setString(registry.LOCAL_MACHINE, capabilitiesPath+`\URLAssociations`, "https", appName); err != nil {
		return err
	}

	return setString(registry.LOCAL_MACHINE, registeredAppsPath, appName, capabilitiesPath)
}

// Unregister removes everything Register wrote.
func Unregister() error {
	if err := deleteTree(registry.CURRENT_USER, protocolKey); err != nil {
		return err
	}
	if err := deleteTree(registry.LOCAL_MACHINE, `Software\fuckHttp`); err != nil {
		return err
	}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registeredAppsPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.DeleteValue(appName); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// IsRegistered reports whether the entries exist.
func IsRegistered() bool {
	if _, err := registry.OpenKey(registry.CURRENT_USER, protocolKey, registry.READ); err != nil {
		return false
	}
	if _, err := registry.OpenKey(registry.LOCAL_MACHINE, capabilitiesPath, registry.READ); err != nil {
		return false
	}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registeredAppsPath, registry.READ)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue(appName)
	return err == nil && value != ""
}

// EnsureRegistered registers the app, asking for administrator rights when it
// does not have them.
func EnsureRegistered() error {
	if IsElevated() {
		return Register()
	}
	return runElevated("--register")
}

// EnsureUnregistered unregisters the app, asking for administrator rights when
// it does not have them.
func EnsureUnregistered() error {
	if IsElevated() {
		return Unregister()
	}
	return runElevated("--unregister")
}

func setString(root registry.Key, path, name, value string) error {
	key, _, err := registry.CreateKey(root, path, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(name, value)
}

// runElevated starts the app again with arg, as administrator, and waits for it.
func runElevated(arg string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	code, err := runas.RunElevated(exe, "", []string{arg}, true)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("the elevated process exited with code %d", code)
	}
	return nil
}

// deleteTree removes a key and everything under it.
func deleteTree(root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	names, err := key.ReadSubKeyNames(0)
	key.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := deleteTree(root, path+`\`+name); err != nil {
			return err
		}
	}
	if err := registry.DeleteKey(root, path); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}
