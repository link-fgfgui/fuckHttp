package browser

import "testing"

func TestExecutable(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "quoted path",
			command: `"C:\Program Files\Google\Chrome\Application\chrome.exe"`,
			want:    `C:\Program Files\Google\Chrome\Application\chrome.exe`,
		},
		{
			name:    "quoted path with arguments",
			command: `"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe" --single-argument %1`,
			want:    `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		},
		{
			name:    "unquoted path",
			command: `C:\Windows\notepad.exe`,
			want:    `C:\Windows\notepad.exe`,
		},
		{
			name:    "unquoted path with arguments",
			command: `C:\tools\browser.exe --new-window %1`,
			want:    `C:\tools\browser.exe`,
		},
		{
			name:    "unterminated quote",
			command: `"C:\broken\browser.exe`,
			want:    `"C:\broken\browser.exe`,
		},
		{
			name:    "empty",
			command: "   ",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Browser{Command: tt.command}.Executable()
			if got != tt.want {
				t.Errorf("Executable() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyHidden(t *testing.T) {
	browsers := []Browser{{Name: "Chrome"}, {Name: "Firefox"}, {Name: "Edge"}}
	ApplyHidden(browsers, []string{"Firefox", "Safari"})

	if browsers[1].Hidden != true {
		t.Error("Firefox should be hidden")
	}
	if browsers[0].Hidden || browsers[2].Hidden {
		t.Error("only Firefox should be hidden")
	}

	hidden := HiddenNames(browsers)
	if len(hidden) != 1 || hidden[0] != "Firefox" {
		t.Errorf("HiddenNames() = %v, want [Firefox]", hidden)
	}
}
