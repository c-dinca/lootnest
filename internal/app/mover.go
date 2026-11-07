//go:build windows

package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Game Mover: moves a Steam game to another Steam library on the PC. The
// copying happens in the background; the UI gets "moveInfo", "moveProgress"
// and "moveDone" events.
// ---------------------------------------------------------------------------

type MoveLib struct {
	Path    string `json:"path"`
	Drive   string `json:"drive"`
	Free    int64  `json:"free"` // -1 when Windows can't say
	Current bool   `json:"current"`
}

type MoveInfo struct {
	AppID        int       `json:"appId"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	Libraries    []MoveLib `json:"libraries"`
	SteamRunning bool      `json:"steamRunning"`
	Error        string    `json:"error,omitempty"`
}

// Room kept free on the new drive beyond the game's size.
const moveMargin = 512 << 20

var (
	moveMu     sync.Mutex
	movingApp  int // app id being moved, 0 when none
	movingGame string
)

// movingName is the game Game Mover is moving, or "".
func movingName() string {
	moveMu.Lock()
	defer moveMu.Unlock()
	return movingGame
}

// apiMoveInfo works out the game's size and the libraries it can move to,
// and answers with a "moveInfo" event (sizing a big game takes a moment).
func (a *App) apiMoveInfo(appID int) {
	go func() {
		info := MoveInfo{AppID: appID, Libraries: []MoveLib{}}
		defer func() { a.emit("moveInfo", map[string]any{"info": info}) }()
		in, err := core.FindSteamInstall(a.env.SteamPath, appID)
		if err != nil {
			info.Error = err.Error() + "."
			return
		}
		info.Name = in.Name
		size, err := in.Size()
		if err != nil {
			info.Error = "Lootnest can't move this game: " + err.Error() + "."
			return
		}
		info.Size = size
		for _, lib := range core.SteamLibraries(a.env.SteamPath) {
			if !dirExists(filepath.Join(lib, "steamapps")) {
				continue // a drive that isn't connected
			}
			info.Libraries = append(info.Libraries, MoveLib{
				Path: lib, Drive: strings.ToUpper(filepath.VolumeName(lib)), Free: win.DiskFree(lib),
				Current: strings.EqualFold(filepath.Clean(lib), filepath.Clean(in.Library)),
			})
		}
		info.SteamRunning = steamRunning()
	}()
}

// moveBlocker says why the game can't be moved right now, or "".
func (a *App) moveBlocker(appID int) string {
	a.mu.Lock()
	var names []string
	for _, g := range a.cfg.Games {
		if g.Launch.Type == "steam" && g.Launch.AppID == appID {
			if a.sessions[g.ID] != nil {
				a.mu.Unlock()
				return "Close " + g.Name + " first, then move it."
			}
			names = append(names, procName(&g))
		}
	}
	a.mu.Unlock()
	for _, f := range a.libraryGames() {
		if f.AppID == appID && f.ProcessName != "" {
			names = append(names, strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe")))
		}
	}
	if procs, err := win.ProcessNames(); err == nil {
		for _, n := range names {
			if n != "" && procs[n] {
				return "Close the game first, then move it."
			}
		}
	}
	return ""
}

// apiMoveGame starts moving a Steam game into another library. Progress and
// the outcome arrive as events.
func (a *App) apiMoveGame(appID int, lib string) error {
	moveMu.Lock()
	if movingApp != 0 {
		moveMu.Unlock()
		return errors.New("Lootnest is already moving a game. Wait for it to finish, then move this one.")
	}
	in, err := core.FindSteamInstall(a.env.SteamPath, appID)
	if err != nil {
		moveMu.Unlock()
		return fmt.Errorf("%s.", err.Error())
	}
	if msg := a.moveBlocker(appID); msg != "" {
		moveMu.Unlock()
		return errors.New(msg)
	}
	known := false
	for _, l := range core.SteamLibraries(a.env.SteamPath) {
		if strings.EqualFold(filepath.Clean(l), filepath.Clean(lib)) {
			known = true
		}
	}
	if !known {
		moveMu.Unlock()
		return errors.New("That isn't one of Steam's libraries on this PC.")
	}
	movingApp, movingGame = appID, in.Name
	moveMu.Unlock()

	go func() {
		var res core.MoveResult
		err := errors.New("the move stopped unexpectedly. Steam's Storage settings show where the game is; details are in lootnest.log")
		func() {
			defer func() {
				if r := recover(); r != nil {
					a.logf("ERROR move crashed: %v", r)
				}
			}()
			res, err = a.moveGame(in, lib)
		}()
		moveMu.Lock()
		movingApp, movingGame = 0, ""
		moveMu.Unlock()
		out := map[string]any{"appId": appID, "name": in.Name}
		if err != nil {
			a.logf("Move of %s to %s failed: %v", in.Name, lib, err)
			out["error"] = err.Error()
		} else {
			a.logf("Moved %s from %s to %s", in.Name, res.From, res.To)
			out["to"] = res.To
			out["lib"] = lib
			if res.Leftover != "" {
				a.logf("Some old files of %s couldn't be removed: %s", in.Name, res.Leftover)
				out["leftover"] = res.Leftover
			}
			a.notify(in.Name+" moved", in.Name+" is now in "+lib+".")
		}
		a.emit("moveDone", out)
		a.startScan()
	}()
	return nil
}

func (a *App) moveGame(in core.SteamInstall, lib string) (core.MoveResult, error) {
	size, err := in.Size()
	if err != nil {
		return core.MoveResult{}, fmt.Errorf("Lootnest can't move this game: %w", err)
	}
	if free := win.DiskFree(lib); free >= 0 && free < size+moveMargin {
		return core.MoveResult{}, fmt.Errorf("there isn't enough room on %s: %s needs %s and %s is free", strings.ToUpper(filepath.VolumeName(lib)), in.Name, gb(size), gb(free))
	}

	wasRunning := steamRunning()
	if wasRunning {
		a.logf("Closing Steam to move %s", in.Name)
		a.emit("moveProgress", map[string]any{"appId": in.AppID, "name": in.Name, "step": "steam"})
		if err := shutdownSteam(a.env.SteamPath); err != nil {
			return core.MoveResult{}, err
		}
	}
	if wasRunning {
		defer func() { _ = win.StartDetached(filepath.Join(a.env.SteamPath, "steam.exe"), "", a.env.SteamPath) }()
	}

	a.logf("Moving %s (%s) from %s to %s", in.Name, gb(size), in.Dir(), lib)
	last := time.Time{}
	progress := func(n int64) {
		if time.Since(last) < 250*time.Millisecond && n < size {
			return
		}
		last = time.Now()
		a.emit("moveProgress", map[string]any{"appId": in.AppID, "name": in.Name, "step": "copy", "done": min(n, size), "total": size})
	}
	check := func() error {
		if steamRunning() {
			return errors.New("Steam was started while the game was being copied")
		}
		if msg := a.moveBlocker(in.AppID); msg != "" {
			return errors.New("the game was started while it was being copied")
		}
		return nil
	}
	res, err := core.MoveSteamGame(in, lib, progress, check)
	if err != nil {
		return res, err
	}

	// Locations Lootnest swaps that were inside the game's folder.
	a.mu.Lock()
	changed := false
	for i := range a.cfg.Games {
		g := &a.cfg.Games[i]
		if g.Launch.Type == "steam" && g.Launch.AppID == in.AppID && a.env.RetargetPaths(g, res.From, res.To) {
			changed = true
		}
	}
	if changed {
		if err := a.store.SaveConfig(a.cfg); err != nil {
			a.logf("ERROR saving the moved game's locations: %v", err)
		}
	}
	a.mu.Unlock()
	if changed {
		a.emitSessions() // the UI reloads the config with it
	}
	return res, nil
}

func gb(n int64) string {
	if n < 1<<30 {
		return fmt.Sprintf("%d MB", (n+(1<<20)-1)>>20)
	}
	return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
}
