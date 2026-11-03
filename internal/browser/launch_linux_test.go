//go:build linux

package browser

import (
	"reflect"
	"testing"
)

func TestLaunchCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantExe string
		wantArg []string
	}{
		{
			name:    "url field code at the end",
			command: "/usr/bin/microsoft-edge-stable %U",
			wantExe: "/usr/bin/microsoft-edge-stable",
			wantArg: []string{},
		},
		{
			name:    "lowercase url field code",
			command: "firefox %u",
			wantExe: "firefox",
			wantArg: []string{},
		},
		{
			name:    "quoted path with spaces",
			command: `"/opt/My Browser/browser" --new-window %u`,
			wantExe: "/opt/My Browser/browser",
			wantArg: []string{"--new-window"},
		},
		{
			name:    "no field code",
			command: "/usr/bin/browser --incognito",
			wantExe: "/usr/bin/browser",
			wantArg: []string{"--incognito"},
		},
		{
			name:    "escaped percent",
			command: "/usr/bin/browser --title=%%u %U",
			wantExe: "/usr/bin/browser",
			wantArg: []string{"--title=%u"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe, args := Browser{Command: tt.command}.launchCommand("https://example.com")
			if exe != tt.wantExe {
				t.Errorf("exe = %q, want %q", exe, tt.wantExe)
			}
			want := append(append([]string{}, tt.wantArg...), "https://example.com")
			if !reflect.DeepEqual(args, want) {
				t.Errorf("args = %q, want %q", args, want)
			}
		})
	}
}
