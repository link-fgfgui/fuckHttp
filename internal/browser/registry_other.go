//go:build !windows && !linux

package browser

// Installed returns nothing where the app knows no browser list.
func Installed() ([]Browser, error) {
	return nil, nil
}
