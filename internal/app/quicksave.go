package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// ---------------------------------------------------------------------------
// Save now (core/quicksave.go): a save point made while a game runs first
// presses the game's own quick save key, the one the user chose for it, and
// waits until the game has written its saves. Only a key press, as the
// keyboard makes; never in a game with an anti-cheat, and only into the
// game's own window, in front.
// ---------------------------------------------------------------------------

var (
	saveInGameWait   = 12 * time.Second        // for the game to start writing
	saveInGameSettle = 1500 * time.Millisecond // and to be done
)

func (a *App) quickSaveKey(key string) string { return a.launchOpts(key).QuickSave }

// gameDir is a point target's game folder, for its anti-cheat.
func (a *App) gameDir(t pointTarget) string {
	if t.game == nil {
		return t.dir
	}
	if t.game.Launch.Type == "exe" {
		if exe, err := a.env.ResolveExe(t.game.Launch.Exe); err == nil {
			return filepath.Dir(exe)
		}
	}
	if t.appID > 0 {
		for _, f := range a.libraryGames() {
			if f.Dir != "" && (f.AppID == t.appID || f.SteamID == t.appID) {
				return f.Dir
			}
		}
	}
	return ""
}

// saveInGame presses the game's quick save key and waits for the game to
// save. It does nothing, and says so with false, when the game isn't
// running, has no key chosen, or has an anti-cheat.
func (a *App) saveInGame(t pointTarget, step func(string)) (bool, error) {
	name := a.quickSaveKey(t.key)
	vk, ok := core.QuickSaveVK(name)
	if !t.running || !ok || t.proc == "" {
		return false, nil
	}
	if ac := core.AntiCheat(a.gameDir(t)); ac != "" {
		a.logf("%s has %s, so Lootnest doesn't press %s in it; the save point has its last save", t.name, ac, name)
		return false, nil
	}
	before, err := core.StampSaves(t.locs)
	if err != nil {
		return false, err
	}
	step("saving")
	if !plat.ForegroundIs(t.proc) {
		gw := plat.GameWindow(t.proc)
		if gw == 0 {
			return false, fmt.Errorf("Lootnest couldn't find %s's window to press %s in. Nothing was made.", t.name, name)
		}
		a.mu.Lock()
		behind := a.conGame == conKey(t.key)
		a.mu.Unlock()
		if behind && a.console() {
			a.ui(a.tvResume) // the Home menu closes and the game comes back
		} else {
			a.ui(func() { plat.ForceForeground(gw) })
		}
		waitFor(3*time.Second, func() bool { return plat.ForegroundIs(t.proc) })
	}
	// The hotkey's Ctrl and Shift are let go first, or the game would get
	// Ctrl+Shift+F5.
	waitFor(3*time.Second, func() bool { return !plat.ModifiersDown() })
	if !plat.ForegroundIs(t.proc) {
		return false, errors.New(t.name + " isn't the window in front, so Lootnest didn't press its quick save key. Nothing was made.")
	}
	if plat.ModifiersDown() {
		return false, errors.New("Ctrl, Shift or Alt is still held, so Lootnest didn't press the quick save key. Nothing was made.")
	}
	plat.TapGameKey(vk)
	a.logf("Pressed %s in %s to save before a save point", name, t.name)
	if !core.WaitForGameSave(t.locs, before, saveInGameWait, saveInGameSettle, 250*time.Millisecond) {
		return false, fmt.Errorf("%s didn't save after %s. Some games can't save during a fight or a scene, or keep quick saves somewhere else. Nothing was made; try again in a moment.", t.name, name)
	}
	return true, nil
}

// apiSetQuickSave chooses a game's quick save key ("" for none).
func (a *App) apiSetQuickSave(key, name string) error {
	name, ok := core.CleanQuickSave(name)
	if !ok {
		return errors.New("Choose one of the keys F1 to F12.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	if cfg.LaunchOpts == nil {
		cfg.LaunchOpts = map[string]core.LaunchOptions{}
	}
	o := cfg.LaunchOpts[key]
	o.QuickSave = name
	if o.Empty() {
		delete(cfg.LaunchOpts, key)
	} else {
		cfg.LaunchOpts[key] = o
	}
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	a.apiSavePoints(key)
	return nil
}
