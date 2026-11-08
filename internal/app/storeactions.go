//go:build windows

package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Install, uninstall and verify through each game's own store
// (core/storeactions.go), the games you own but haven't installed
// (core/owned.go), and Remove from this PC for folder games
// (core/removegame.go).
// ---------------------------------------------------------------------------

var (
	ownedMu   sync.Mutex
	ownedLast []core.OwnedGame // the last list sent: the only games Install takes
)

func storeIDOf(source string, appID int, storeID string) string {
	if source == "steam" {
		return strconv.Itoa(appID)
	}
	return storeID
}

// apiStoreAction hands a game to its store's launcher, for install,
// uninstall, verify or open, then scans again once the launcher has had
// time to start on it.
func (a *App) apiStoreAction(key, action string) error {
	var source, id, name string
	if f, ok := a.foundByKey(key); ok {
		source, id, name = f.Source, storeIDOf(f.Source, f.AppID, f.StoreID), f.Name
	} else {
		ownedMu.Lock()
		for _, o := range ownedLast {
			if o.Key == key {
				source, id, name = o.Source, storeIDOf(o.Source, o.AppID, o.StoreID), o.Name
			}
		}
		ownedMu.Unlock()
	}
	if source == "" {
		return errFoundGone
	}
	u := core.StoreAction(source, id, action)
	if u == "" || !core.SafeHandoff(u) {
		return errors.New("Its store has no way to do that from Lootnest.")
	}
	if err := win.ShellOpen(u); err != nil {
		return fmt.Errorf("Windows couldn't open the launcher for %s: %w", name, err)
	}
	a.logf("Asked the %s launcher to %s %s (%s)", source, action, name, u)
	if action == "install" || action == "uninstall" {
		time.AfterFunc(90*time.Second, a.startScan)
	}
	return nil
}

// apiOwnedGames reads the games the stores' files on this PC say you have,
// with the itch.io and GOG libraries when they're connected, and sends the ones not
// installed as an "owned" event.
func (a *App) apiOwnedGames() {
	go func() {
		// A game installed from any store, or as a folder game, isn't listed.
		steamIDs, epicIDs, gogIDs, names, keys := map[int]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, f := range a.libraryGames() {
			steamIDs[f.AppID], steamIDs[f.SteamID], names[core.CoverKey(f.Name)], keys[f.Key] = true, true, true, true
			switch f.Source {
			case "epic":
				epicIDs[f.StoreID] = true
			case "gog":
				gogIDs[f.StoreID] = true
			}
		}
		var all []core.OwnedGame
		if ud, err := a.env.SteamUser(); err == nil {
			all = core.SteamOwned(a.env.SteamPath, ud, steamIDs)
		}
		if pd := win.ProgramData(); pd != "" {
			all = append(all, core.EpicOwned(filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Catalog", "catcache.bin"), epicIDs)...)
		}
		itch, itchErr, itchReading := a.itchOwnedGames(keys)
		gog, gogErr, gogReading := a.gogOwnedGames(keys, gogIDs)
		all = append(append(all, itch...), gog...)
		out := []core.OwnedGame{}
		for _, o := range all {
			if !names[core.CoverKey(o.Name)] {
				out = append(out, o)
			}
		}
		ownedMu.Lock()
		ownedLast = out
		ownedMu.Unlock()
		a.emit("owned", map[string]any{"games": out, "itchError": itchErr, "itchReading": itchReading, "gogError": gogErr, "gogReading": gogReading})
	}()
}

// apiRemoveGame removes a folder game from this PC. Without goAhead it
// only plans, answered as a "removePlan" event: what's copied, and whether
// it can go at all. With goAhead, the saves are copied to backups first and
// the folder goes to the Recycle Bin; the answer is a "gameRemoved" event.
func (a *App) apiRemoveGame(key string, goAhead bool) error {
	f, ok := a.foundByKey(key)
	if !ok {
		return errFoundGone
	}
	if !updatableSource(f.Source) || f.AppID > 0 || f.URI != "" {
		return errors.New("Its store or launcher removes this game: use Uninstall in Manage.")
	}
	go func() {
		running := false
		if f.ProcessName != "" {
			procs, _ := win.ProcessNames()
			running = procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))]
		}
		p, err := a.env.PlanRemove(f.Name, f.Dir, withFiles(a.vaultLocs(f)), a.store.BackupsDir(), running, time.Now())
		if !goAhead {
			out := map[string]any{"key": key, "plan": p}
			if err != nil {
				out = map[string]any{"key": key, "error": err.Error()}
			}
			a.emit("removePlan", out)
			return
		}
		out := map[string]any{"key": key, "name": f.Name}
		defer func() { a.emit("gameRemoved", out) }()
		if err != nil {
			out["error"] = err.Error()
			return
		}
		if err := p.CopySaves(); err != nil {
			out["error"] = "Nothing was removed: Lootnest couldn't copy its saves first (" + err.Error() + ")."
			return
		}
		if err := win.Recycle(a.hwnd, p.Dir); err != nil {
			out["error"] = "The folder wasn't removed (" + err.Error() + "). Its saves are copied in Lootnest's backups."
			if errors.Is(err, win.ErrRecycleCancelled) {
				out["error"] = "The folder stays where it is. Its saves are copied in Lootnest's backups too."
			}
			return
		}
		if err := a.store.UpdateCopies(func(l *core.CopyLedger) { l.NoteRemoved(f, p.KeptSaves(), time.Now()) }); err != nil {
			a.logf("ERROR noting that %s was removed: %v", f.Name, err)
		}
		if f.Added { // added by hand: it leaves the Library with its folder
			a.mu.Lock()
			cfg := a.cfg.Clone()
			a.mu.Unlock()
			cfg.AddedGames = withoutAdded(cfg.AddedGames, key)
			if err := a.saveConfig(cfg); err != nil {
				a.logf("ERROR taking %s out of the Library: %v", f.Name, err)
			}
		}
		a.logf("Removed %s from this PC: %s went to the Recycle Bin, its saves are in %s", f.Name, p.Dir, p.Keep)
		out["keep"] = p.Keep
		a.startScan()
	}()
	return nil
}
