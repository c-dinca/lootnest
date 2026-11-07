//go:build windows

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Save Safety Net. Lootnest keeps copies of the saves of every Library game
// outside Steam on this PC whenever they change and the game is closed, in
// the same versions format as cloud saves (safety-net\Lootnest\saves\<game>),
// with no setting and nothing sent anywhere: the saves it knows of (chosen,
// from the list, or folders named like the game) and the ones inside the
// game's own folder (Ren'Py's game\saves, RPG Maker's www\save), which a
// reinstall or an update that replaces the folder would take with it. When
// a game's in-folder saves are gone, the user is told, more copies are kept
// from then on, and a copy can be put back from the game's page.
// ---------------------------------------------------------------------------

const (
	netKeep    = 5       // copies kept per game
	netMaxSize = 1 << 30 // saves bigger than this aren't copied
	// A save location outside the game's folder bigger than this is left
	// out: a folder named like the game can hold caches or mods too.
	vaultLocMax = 256 << 20
)

var netMu sync.Mutex // one safety net pass (or restore) at a time

func netStatePath() string { return filepath.Join(DataDir(), "safetynet.json") }

// netStore is where the copies are: a folder of Lootnest's own, read and
// written like a cloud folder, never encrypted (profiles aren't either).
func netStore() *core.Cloud {
	return &core.Cloud{Root: filepath.Join(DataDir(), "safety-net"), Computer: computerName()}
}

type netEntry struct {
	Quick string    `json:"quick,omitempty"`
	Hash  string    `json:"hash,omitempty"`
	At    time.Time `json:"at,omitempty"`
	// Gone: the saves were in the folder and aren't any more; the user was
	// told once. Keep holds more copies from then on, so the ones from
	// before stay while the game makes new saves.
	Gone bool `json:"gone,omitempty"`
	Keep int  `json:"keep,omitempty"`
	// Folder: the last copy had saves from inside the game's folder. Before
	// 1.52 every copy did; Hash without Folder is read as one.
	Folder bool `json:"folder,omitempty"`
	Vault  bool `json:"vault,omitempty"` // written from 1.52 on
}

func loadNet() map[string]netEntry {
	st := map[string]netEntry{}
	if b, err := os.ReadFile(netStatePath()); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

func saveNet(st map[string]netEntry) {
	b, _ := json.Marshal(st)
	if os.WriteFile(netStatePath()+".tmp", b, 0o644) == nil {
		_ = os.Rename(netStatePath()+".tmp", netStatePath())
	}
}

// netLocs are a game's save locations inside its own folder: the known ones
// (chosen, or from the list) and the folders engines use there.
func (a *App) netLocs(f core.FoundGame) []core.SaveLocation {
	if f.Dir == "" {
		return nil
	}
	locs := core.InFolder(a.savesOf(f).Locs, f.Dir)
	for _, l := range a.env.FolderSaves(f.Dir) {
		dup := false
		for _, k := range locs {
			dup = dup || strings.EqualFold(k.Resolved, l.Resolved) || isInside(l.Resolved, k.Resolved) || isInside(k.Resolved, l.Resolved)
		}
		if !dup {
			locs = append(locs, l)
		}
	}
	return locs
}

// vaultLocs are every save location of a game the safety net copies: the
// in-folder ones and the others Lootnest knows of, without duplicates or
// folders inside one another.
func (a *App) vaultLocs(f core.FoundGame) []core.SaveLocation {
	locs := a.netLocs(f)
	for _, l := range a.savesOf(f).Locs {
		if l.Size > vaultLocMax {
			continue // more than saves: caches, mods, a whole game
		}
		dup := false
		for _, k := range locs {
			dup = dup || strings.EqualFold(k.Resolved, l.Resolved) || isInside(l.Resolved, k.Resolved)
		}
		if dup {
			continue
		}
		// A known folder that holds in-folder ones replaces them.
		kept := locs[:0]
		for _, k := range locs {
			if !isInside(k.Resolved, l.Resolved) {
				kept = append(kept, k)
			}
		}
		locs = append(kept, l)
	}
	return locs
}

func withFiles(locs []core.SaveLocation) []core.SaveLocation {
	var out []core.SaveLocation
	for _, l := range locs {
		if core.HasFiles(l.Resolved) {
			out = append(out, l)
		}
	}
	return out
}

// safetyNet makes a copy of each Library game's saves that changed, and
// tells the user when a game's saves are gone from its folder.
func (a *App) safetyNet() {
	netMu.Lock()
	defer netMu.Unlock()
	procs, _ := win.ProcessNames()
	st := loadNet()
	c := netStore()
	changed := false
	for _, f := range a.visibleFound(a.libraryGames()) {
		if !gameSavesOn(f) || (f.Dir != "" && !dirExists(f.Dir)) {
			continue
		}
		if f.ProcessName != "" && procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))] {
			continue // the game may be writing them
		}
		key := gameSavesKey(f)
		e := st[key]
		inFolder := len(withFiles(a.netLocs(f))) > 0
		locs := withFiles(a.vaultLocs(f))
		hadFolder := e.Hash != "" && (e.Folder || !e.Vault)
		if hadFolder && !inFolder && !e.Gone {
			e.Gone = true
			e.Keep = len(c.GameVersions(key)) + netKeep
			st[key], changed = e, true
			a.logf("Safety net: %s's saves are gone from %s; copies are in %s", f.Name, f.Dir, c.Root)
			a.notify(f.Name+"'s saves are gone from its folder",
				"If it was reinstalled or updated, Lootnest has a copy from "+e.At.Local().Format("2 Jan, 15:04")+". Put it back from the game's page.")
		}
		if e.Gone && inFolder {
			e.Gone = false
			st[key], changed = e, true
		}
		if len(locs) == 0 {
			continue
		}
		var size int64
		for _, l := range locs {
			size += l.Size
		}
		if size > netMaxSize {
			continue
		}
		quick := quickPrint(locs)
		if quick == e.Quick {
			continue
		}
		hash, err := core.SavesHash(locs)
		if err != nil {
			continue
		}
		e.Quick = quick
		if hash != e.Hash {
			if vs := c.GameVersions(key); len(vs) == 0 || vs[0].Hash != hash {
				keep := netKeep
				if e.Keep > keep {
					keep = e.Keep
				}
				v, err := c.SaveGameVersion(key, f.Name, locs, hash, keep)
				if err != nil {
					a.logf("ERROR safety net for %s: %v", f.Name, err)
					continue
				}
				e.At = v.Created
				a.logf("Safety net: kept a copy of %s's saves (%d bytes)", f.Name, v.Size)
			}
			e.Hash = hash
			if !e.Gone {
				e.Folder = inFolder
			}
			e.Vault = true
		}
		st[key], changed = e, true
	}
	if changed {
		saveNet(st)
	}
}

// apiGameSaveRestoreLocal puts a copy from the safety net back in the
// game's folder. The saves there now are kept in backups first.
func (a *App) apiGameSaveRestoreLocal(key, id string) error {
	f, ok := a.foundByKey(key)
	if !ok {
		return errFoundGone
	}
	if procs, _ := win.ProcessNames(); f.ProcessName != "" && procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))] {
		return fmt.Errorf("%s is running; close it first.", f.Name)
	}
	go func() {
		out := map[string]any{"key": key, "restored": true}
		defer func() { a.emit("gameSaves", out); a.apiGameSaves(key) }()
		netMu.Lock()
		defer netMu.Unlock()
		if err := a.restoreVersion(netStore(), f, id, a.vaultLocs(f)); err != nil {
			out["error"] = err.Error()
			return
		}
		st := loadNet()
		delete(st, gameSavesKey(f)) // read them again next time
		saveNet(st)
	}()
	return nil
}

// restoreVersion puts version id of a game's saves from c in place over
// locs, keeping the saves there now in backups\saves-<game> first.
func (a *App) restoreVersion(c *core.Cloud, f core.FoundGame, id string, locs []core.SaveLocation) error {
	keep := filepath.Join(a.store.BackupsDir(), "saves-"+gameSavesKey(f), time.Now().Format("20060102-150405")+"-before-restore")
	for i, l := range locs {
		if err := core.Mirror(l.Resolved, filepath.Join(keep, fmt.Sprint(i))); err != nil {
			return errors.New("Nothing was restored: Lootnest couldn't keep the saves here first (" + err.Error() + ").")
		}
	}
	tmp := filepath.Join(DataDir(), "saves-tmp-"+time.Now().Format("150405.000"))
	if _, err := c.RestoreGameVersion(gameSavesKey(f), id, locs, a.env.Resolve, tmp); err != nil {
		return err
	}
	a.logf("Restored %s's saves from version %s (the saves before are in %s)", f.Name, id, keep)
	return nil
}
