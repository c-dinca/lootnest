//go:build windows

// Lootnest: separate game settings and saves for everyone sharing one PC.
//
//	Lootnest.exe                 open the app
//	Lootnest.exe --tray          start in the tray (used by "Start with Windows")
//	Lootnest.exe --launch <id>   ask who's playing, then start that game
//	Lootnest.exe --safe-mode     start with background work off (or hold Shift while it opens)
//	Lootnest.exe --tv            TV mode, for a streaming host: runs until TV mode ends
//	Lootnest.exe --app <id>      a site as an app, full screen (YouTube's TV interface)
//	Lootnest.exe --nexus <d/m/f> a mod's file on nexusmods.com, in a window of Lootnest's own
//	Lootnest.exe --uninstall     remove Lootnest
package main

import (
	"os"
	"runtime"
	"strconv"
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
	var tray, uninstall, quiet, safe, restarted, tv bool
	var launch, site, nexus string
	var on uintptr
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
		case "--tv", "/tv":
			tv = true
		case "--restarted": // the old Lootnest is still closing
			restarted = true
		case "--app":
			if i+1 < len(args) {
				site = args[i+1]
				i++
			}
		case "--nexus":
			if i+1 < len(args) {
				nexus = args[i+1]
				i++
			}
		case "--on": // the window whose monitor --app fills
			if i+1 < len(args) {
				n, _ := strconv.ParseUint(args[i+1], 10, 64)
				on = uintptr(n)
				i++
			}
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

	// A site shown as an app is a program of its own (app/webapp.go).
	if site != "" {
		os.Exit(app.RunWebApp(site, on))
	}

	// Nexus Mods in a window of Lootnest's own, for downloading a mod (app/nexus.go).
	if nexus != "" {
		os.Exit(app.RunNexusWindow(nexus))
	}

	// A streaming host's app: this process only holds the stream, while
	// the Lootnest that swaps files runs on its own (app/tv.go).
	if tv {
		os.Exit(app.TVHolder())
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
