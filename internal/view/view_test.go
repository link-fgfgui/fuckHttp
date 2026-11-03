package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/link-fgfgui/fuckHttp/internal/browser"
	"github.com/link-fgfgui/fuckHttp/internal/extract"
)

func testState() *State {
	return New(
		"https://example.com",
		extract.Result{URL: "https://example.com", Source: extract.SourceNone},
		[]browser.Browser{
			{Name: "Chrome", Command: `"C:\chrome.exe"`},
			{Name: "Firefox", Command: `"C:\firefox.exe"`},
			{Name: "Safari", Command: `"C:\safari.exe"`, Hidden: true},
		},
	)
}

func TestViewShowsURLAndVisibleBrowsers(t *testing.T) {
	state := testState()
	tester := ui.NewTester(state.View, windowWidth, 400)

	if !tester.HasText("https://example.com") {
		t.Errorf("the URL is missing, texts: %q", tester.Texts())
	}
	if !tester.HasText("Chrome") || !tester.HasText("Firefox") {
		t.Errorf("browser buttons are missing, texts: %q", tester.Texts())
	}
	if tester.HasText("Safari") {
		t.Error("a hidden browser is shown")
	}
}

func TestSettingsShowsHiddenBrowsers(t *testing.T) {
	state := testState()
	tester := ui.NewTester(state.View, windowWidth, settingsHeight)

	if err := tester.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if !state.Settings {
		t.Fatal("settings did not open")
	}
	if !tester.HasText("Safari") {
		t.Errorf("the hidden browser is missing from the settings, texts: %q", tester.Texts())
	}
}

func TestHeightGrowsWithBrowsers(t *testing.T) {
	state := testState()
	three := state.Height()

	state.Browsers = state.Browsers[:1]
	if one := state.Height(); one >= three {
		t.Errorf("height with one browser = %d, want less than %d", one, three)
	}

	state.Settings = true
	if got := state.Height(); got != settingsHeight {
		t.Errorf("settings height = %d, want %d", got, settingsHeight)
	}
}
