//go:build windows

package browser

import "golang.org/x/sys/windows/registry"

// Installed lists the browsers registered under StartMenuInternet, machine
// wide first and then for the user, once each by display name.
func Installed() ([]Browser, error) {
	var browsers []Browser
	seen := make(map[string]bool)

	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		names, err := subKeyNames(root, `SOFTWARE\Clients\StartMenuInternet`)
		if err != nil {
			continue
		}
		for _, key := range names {
			b, ok := readBrowser(root, key)
			if !ok || seen[b.Name] {
				continue
			}
			seen[b.Name] = true
			browsers = append(browsers, b)
		}
	}
	return browsers, nil
}

func subKeyNames(root registry.Key, path string) ([]string, error) {
	key, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		return nil, err
	}
	defer key.Close()
	return key.ReadSubKeyNames(0)
}

func readBrowser(root registry.Key, subKey string) (Browser, bool) {
	path := `SOFTWARE\Clients\StartMenuInternet\` + subKey

	key, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		return Browser{}, false
	}
	name, _, err := key.GetStringValue("")
	key.Close()
	if err != nil || name == "" {
		return Browser{}, false
	}

	commandKey, err := registry.OpenKey(root, path+`\shell\open\command`, registry.READ)
	if err != nil {
		return Browser{}, false
	}
	command, _, err := commandKey.GetStringValue("")
	commandKey.Close()
	if err != nil || command == "" {
		return Browser{}, false
	}

	return Browser{Name: name, Command: command}, true
}
