// Lootnest on Linux (docs/ROADMAP.md, section 9):
//
//	lootnest                 open the app
//	lootnest --tray          start hidden
//	lootnest --launch <id>   ask who's playing, then start that game
//	lootnest --safe-mode     start with background work off
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"lootnest/internal/app"
)

func init() {
	// GTK and WebKit live on the main OS thread.
	runtime.LockOSThread()
}

func main() {
	var tray, safe bool
	var launch string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch strings.ToLower(args[i]) {
		case "--tray":
			tray = true
		case "--safe-mode":
			safe = true
		case "--launch":
			if i+1 < len(args) {
				launch = args[i+1]
				i++
			}
		}
	}

	if !app.AcquireInstance() {
		// Already running: hand the request to that Lootnest.
		cmd := "show"
		switch {
		case safe:
			cmd = "safe"
		case launch != "":
			cmd = "launch:" + launch
		case tray:
			return
		}
		if !app.SendToRunning(cmd) {
			fmt.Fprintln(os.Stderr, "Lootnest is already running but didn't answer.")
			os.Exit(1)
		}
		return
	}

	if err := app.Run(app.Options{StartHidden: tray, LaunchGame: launch, Safe: safe}); err != nil {
		fmt.Fprintln(os.Stderr, "Lootnest:", err)
		os.Exit(1)
	}
}
