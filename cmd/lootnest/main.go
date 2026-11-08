//go:build windows

// Lootnest: separate game settings and saves for everyone sharing one PC.
//
//	Lootnest.exe                 open the app
//	Lootnest.exe --tray          start in the tray (used by "Start with Windows")
//	Lootnest.exe --launch <id>   ask who's playing, then start that game
//	Lootnest.exe --safe-mode     start with background work off (or hold Shift while it opens)
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
	var tray, uninstall, quiet, safe, restarted bool
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
		case "--safe-mode", "/safe-mode":
			safe = true
		case "--restarted": // the old Lootnest is still closing
			restarted = true
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

	got := win.AcquireSingleInstance(app.MutexName)
	for i := 0; !got && restarted && i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		got = win.AcquireSingleInstance(app.MutexName)
	}
	if !got {
		// Already running: hand the request to that instance.
		for i := 0; i < 20; i++ {
			if h := win.FindWindow(app.WindowClass); h != 0 {
				cmd := "show"
				if safe {
					cmd = "safe"
				} else if launch != "" {
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

	if err := app.Run(app.Options{MigrateErr: migrateErr, StartHidden: tray, LaunchGame: launch, Safe: safe || (!tray && launch == "" && !restarted && win.ShiftDown())}); err != nil {
		os.Exit(1)
	}
}
