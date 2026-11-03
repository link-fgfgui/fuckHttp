//go:build linux

package browser

import (
	"strings"
	"syscall"
)

func detached() *syscall.SysProcAttr {
	return nil
}

// launchCommand expands the Exec line of a desktop entry, as the Desktop Entry
// specification says, and appends the URL.
func (b Browser) launchCommand(target string) (string, []string) {
	fields := expandFieldCodes(splitExec(b.Command))
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], append(fields[1:], target)
}

// executableOf is the program an Exec line runs.
func executableOf(command string) string {
	fields := expandFieldCodes(splitExec(command))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// splitExec splits a command line into arguments: spaces separate them, double
// quotes group them, and a backslash escapes the next character.
func splitExec(command string) []string {
	var (
		args    []string
		current strings.Builder
		quoted  bool
		started bool
	)
	flush := func() {
		if started {
			args = append(args, current.String())
			current.Reset()
			started = false
		}
	}
	for i := 0; i < len(command); i++ {
		switch c := command[i]; {
		case c == '\\' && i+1 < len(command):
			i++
			current.WriteByte(command[i])
			started = true
		case c == '"':
			quoted = !quoted
			started = true
		case (c == ' ' || c == '\t') && !quoted:
			flush()
		default:
			current.WriteByte(c)
			started = true
		}
	}
	flush()
	return args
}

// fieldCodes are the codes the specification defines, the deprecated ones
// included: they are removed from the command line.
var fieldCodes = []string{
	"%f", "%F", "%u", "%U", "%d", "%D", "%n", "%N",
	"%i", "%c", "%k", "%v", "%m",
}

// expandFieldCodes removes the field codes of the command line, keeping "%%"
// as a literal percent sign.
func expandFieldCodes(args []string) []string {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.ReplaceAll(arg, "%%", "\x00")
		for _, code := range fieldCodes {
			arg = strings.ReplaceAll(arg, code, "")
		}
		arg = strings.ReplaceAll(arg, "\x00", "%")
		if arg != "" {
			expanded = append(expanded, arg)
		}
	}
	return expanded
}
