// Command fuckHttp is a browser selector for URLs that apps intercept.
//
// It is started with the intercepted URL, extracts the real one, and lets the
// user choose which browser opens it.
package main

import (
	"log"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/link-fgfgui/fuckHttp/internal/browser"
	"github.com/link-fgfgui/fuckHttp/internal/config"
	"github.com/link-fgfgui/fuckHttp/internal/extract"
	"github.com/link-fgfgui/fuckHttp/internal/sysint"
	"github.com/link-fgfgui/fuckHttp/internal/view"
)

const defaultURL = "https://www.google.com"

func main() {
	args := os.Args[1:]

	// Registering happens before any window exists, so that the elevated
	// process the app starts is short lived.
	if has(args, "--register") {
		if err := sysint.Register(); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if has(args, "--unregister") {
		if err := sysint.Unregister(); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	raw := target(args)
	result := extract.Resolve(raw)

	browsers, err := browser.Installed()
	if err != nil {
		log.Printf("listing browsers: %v", err)
	}
	browser.ApplyHidden(browsers, config.Load().HiddenBrowsers)

	state := view.New(raw, result, browsers)
	mygo.App.SetName("fuckHttp")
	mygo.App.WhenReady(state.Open)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// has reports whether args holds flag.
func has(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

// target is the URL to open: the first argument that is not a flag.
func target(args []string) string {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			return arg
		}
	}
	return defaultURL
}
