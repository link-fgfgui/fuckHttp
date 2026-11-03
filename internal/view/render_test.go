package view

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestRenderSnapshots writes what the window shows to the temp directory, for
// looking at.
func TestRenderSnapshots(t *testing.T) {
	if os.Getenv("RENDER_SNAPSHOTS") == "" {
		t.Skip("set RENDER_SNAPSHOTS=1 to write snapshots")
	}

	for _, dark := range []bool{false, true} {
		state := testState()
		tester := ui.NewTester(state.View, windowWidth, state.Height())
		tester.SetDark(dark)
		writePNG(t, tester, snapshotPath(themeName(dark)+".png"))

		if err := tester.Click("设置"); err != nil {
			t.Fatal(err)
		}
		tester.SetSize(windowWidth, settingsHeight)
		writePNG(t, tester, snapshotPath(themeName(dark)+"-settings.png"))
	}

	// No browsers found: the window still shows the whole message.
	empty := testState()
	empty.Browsers = nil
	tester := ui.NewTester(empty.View, windowWidth, empty.Height())
	writePNG(t, tester, snapshotPath("empty.png"))
}

func snapshotPath(name string) string {
	return filepath.Join(os.TempDir(), "view-"+name)
}

func themeName(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

func writePNG(t *testing.T, tester *ui.Tester, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tester.Image()); err != nil {
		t.Fatal(err)
	}
}
