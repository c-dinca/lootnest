//go:build windows

package app

import (
	"errors"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Launch options (core/launchopts.go): the folder a game starts in, Run as
// administrator, and the apps closed and programs started around play. Only
// ordinary programs the user named are closed or started; nothing is ever
// injected into a game.
// ---------------------------------------------------------------------------

func (a *App) launchOpts(key string) core.LaunchOptions {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.LaunchOpts[key]
}

// apiSaveLaunchOptions saves a game's arguments, where they were kept
// before (Game.Launch.Args, LibraryPref.Args), and its launch options. The
// folder and administrator rights are only for games started by their exe.
func (a *App) apiSaveLaunchOptions(key, args string, o core.LaunchOptions) error {
	o, err := core.CleanLaunchOptions(o)
	if err != nil {
		return err
	}
	p, err := core.CleanLibraryPref(core.LibraryPref{Args: args})
	if err != nil {
		return err
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	exe := false
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		g := cfg.Game(id)
		if g == nil {
			return errors.New("That game no longer exists.")
		}
		g.Launch.Args, exe = p.Args, g.Launch.Type == "exe"
	} else {
		f, ok := a.foundByKey(key)
		if !ok {
			return errFoundGone
		}
		if cfg.LibraryPrefs == nil {
			cfg.LibraryPrefs = map[string]core.LibraryPref{}
		}
		lp := cfg.LibraryPrefs[key]
		lp.Args = p.Args
		if lp.Empty() {
			delete(cfg.LibraryPrefs, key)
		} else {
			cfg.LibraryPrefs[key] = lp
		}
		exe = f.AppID == 0 && f.URI == ""
	}
	if !exe {
		o.WorkDir, o.RunAsAdmin = "", false
	}
	if cfg.LaunchOpts == nil {
		cfg.LaunchOpts = map[string]core.LaunchOptions{}
	}
	if o.Empty() {
		delete(cfg.LaunchOpts, key)
	} else {
		cfg.LaunchOpts[key] = o
	}
	return a.saveConfig(cfg)
}

// startExe starts a game by its exe, in the folder and with the rights its
// launch options ask for.
func (a *App) startExe(key, exe, args string) error {
	o := a.launchOpts(key)
	dir := dirOf(exe)
	if o.WorkDir != "" {
		dir = o.WorkDir
	}
	if o.RunAsAdmin {
		a.logf("Starting %s as administrator, in %s", filepath.Base(exe), dir)
		return win.StartElevated(a.hwnd, exe, args, dir)
	}
	return win.StartDetached(exe, args, dir)
}

// apiRunningApps lists the programs with a window open, for the apps to
// close before play. Windows' own aren't listed.
func (a *App) apiRunningApps() []win.RunningApp {
	apps := win.RunningApps()
	slices.SortFunc(apps, func(x, y win.RunningApp) int {
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return apps
}

// apiSteamProperties opens a Steam game's properties in Steam, where its
// own launch options are set.
func (a *App) apiSteamProperties(appID int) error {
	if appID <= 0 {
		return errors.New("That isn't a Steam game.")
	}
	if a.env.SteamPath == "" {
		return errors.New("Steam wasn't found on this PC")
	}
	return win.ShellOpen("steam://gameproperties/" + strconv.Itoa(appID))
}

// playRun is what the steps before a game's start did, for the steps after.
type playRun struct {
	name   string
	closed []string // the paths of the apps closed, to open again
}

var playRuns sync.Map // Library key -> playRun

const closeWait = 10 * time.Second

// stepsBefore closes the apps a game's options name and starts its programs
// before it starts. Apps are asked to close as if their window was closed;
// one that's still running after closeWait is ended only if the user ticked
// Force for it. Not in safe mode, and once for each run of the game.
func (a *App) stepsBefore(key, name string) {
	o := a.launchOpts(key)
	if a.safe || !o.HasSteps() {
		return
	}
	if _, running := playRuns.Load(key); running {
		return
	}
	type closing struct {
		app   core.CloseApp
		proc  string
		paths []string
	}
	var waiting []closing
	for _, c := range o.CloseApps {
		proc := strings.ToLower(strings.TrimSuffix(c.Name, filepath.Ext(c.Name)))
		if paths := win.ProcessPaths(proc); len(paths) > 0 {
			win.CloseProcessWindows(proc)
			waiting = append(waiting, closing{c, proc, paths})
		}
	}
	if len(waiting) > 0 {
		names := []string{}
		for _, w := range waiting {
			names = append(names, strings.TrimSuffix(w.app.Name, filepath.Ext(w.app.Name)))
		}
		list := names[len(names)-1]
		if len(names) > 1 {
			list = strings.Join(names[:len(names)-1], ", ") + " and " + list
		}
		a.emit("toast", map[string]any{"kind": "ok", "text": "Closing " + list + " before " + name})
	}
	var closed []string
	until := time.Now().Add(closeWait)
	for len(waiting) > 0 && time.Now().Before(until) {
		time.Sleep(250 * time.Millisecond)
		procs, err := win.ProcessNames()
		if err != nil {
			break
		}
		left := waiting[:0]
		for _, w := range waiting {
			if procs[w.proc] {
				left = append(left, w)
				continue
			}
			closed = append(closed, w.paths...)
			a.logf("Closed %s before %s", w.app.Name, name)
		}
		waiting = left
	}
	for _, w := range waiting {
		if !w.app.Force {
			a.logf("%s didn't close before %s (it may stay in the tray), so it keeps running", w.app.Name, name)
			continue
		}
		if win.KillProcesses(w.proc) == 0 {
			a.logf("Couldn't end %s before %s (it may run as administrator), so it keeps running", w.app.Name, name)
			continue
		}
		closed = append(closed, w.paths...)
		a.logf("Ended %s before %s (Force is on for it)", w.app.Name, name)
	}
	for _, p := range o.StartBefore {
		a.startProgram(p, "before "+name)
	}
	playRuns.Store(key, playRun{name, closed})
}

// stepsAfter starts a game's programs for after play once it has closed,
// and opens the apps closed before it again if its options ask. When the
// game didn't start, or wasn't seen starting (played is false), only the
// apps open again. Only after stepsBefore ran for it.
func (a *App) stepsAfter(key string, played bool) {
	v, ok := playRuns.LoadAndDelete(key)
	if !ok {
		return
	}
	r, o := v.(playRun), a.launchOpts(key)
	if !played {
		a.logf("%s wasn't seen running, so the programs for after play don't start", r.name)
		o.StartAfter = nil
	}
	for _, p := range o.StartAfter {
		a.startProgram(p, "after "+r.name)
	}
	if o.ReopenClosed {
		for _, p := range r.closed {
			a.startProgram(core.Program{Path: p}, "again after "+r.name)
		}
	}
}

// moveLaunchOpts gives a Library game's launch options to the game with
// profiles it became. The folder and administrator rights stay only for a
// game started by its exe.
func (a *App) moveLaunchOpts(from string, g core.Game) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	o, ok := cfg.LaunchOpts[from]
	if !ok {
		return
	}
	delete(cfg.LaunchOpts, from)
	if g.Launch.Type != "exe" {
		o.WorkDir, o.RunAsAdmin = "", false
	}
	if !o.Empty() {
		cfg.LaunchOpts["game:"+g.ID] = o
	}
	if err := a.saveConfig(cfg); err != nil {
		a.logf("ERROR moving %s's launch options: %v", g.Name, err)
	}
}

func (a *App) startProgram(p core.Program, when string) {
	if err := win.StartDetached(p.Path, p.Args, dirOf(p.Path)); err != nil {
		a.logf("Couldn't start %s %s: %v", p.Path, when, err)
		return
	}
	a.logf("Started %s %s", filepath.Base(p.Path), when)
}
