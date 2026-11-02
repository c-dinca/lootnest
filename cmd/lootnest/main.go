//go:build windows

// Lootnest: separate game settings and saves for everyone sharing one PC.
//
//	Lootnest.exe                 open the app
//	Lootnest.exe --tray          start in the tray (used by "Start with Windows")
//	Lootnest.exe --launch <id>   ask who's playing, then start that game
//	Lootnest.exe --uninstall     remove Lootnest
package main

import (
	"os"
	"runtime"
	"strings"
	"time"

	"lootnest/internal/app"
	"lootnest/internal/win"
)

func init() {
	// The window and WebView2 live on the main OS thread.
	runtime.LockOSThread()
}

func main() {
	var tray, uninstall, quiet bool
	var launch string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch strings.ToLower(args[i]) {
		case "--tray", "/tray":
			tray = true
		case "--uninstall", "/uninstall":
			uninstall = true
		case "--quiet", "/quiet", "/s":
			quiet = true
		case "--launch", "/launch":
			if i+1 < len(args) {
				launch = args[i+1]
				i++
			}
		}
	}

	// Started by Steam through the game's launch options.
	if appID, command, ok := app.ParseSteamArgs(args); ok {
		os.Exit(app.SteamWrapper(appID, command))
	}

	if uninstall {
		app.Uninstall(quiet)
		return
	}

	if !win.AcquireSingleInstance(app.MutexName) {
		// Already running: hand the request to that instance.
		for i := 0; i < 20; i++ {
			if h := win.FindWindow(app.WindowClass); h != 0 {
				cmd := "show"
				if launch != "" {
					cmd = "launch:" + launch
				} else if tray {
					return
				}
				win.SendCommand(h, cmd)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		return
	}

	// Lootnest used to be PlaySwap: move its data folder over (once).
	migrateErr := app.MigrateLegacyData()

	if err := app.Run(app.Options{MigrateErr: migrateErr, StartHidden: tray, LaunchGame: launch}); err != nil {
		os.Exit(1)
	}
}
