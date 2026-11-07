//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// A new copy of the same game (core/newcopy.go). After each scan the ledger
// of folder games is updated; the Updates page and the new copy's own page
// ask "Is this a new copy of X?", and Carry over moves its saves, profiles
// and everything recorded for it. The old folder can then be deleted, with
// its saves copied first.
// ---------------------------------------------------------------------------

// noteCopies updates the ledger with what a scan found.
func (a *App) noteCopies() {
	var folder []core.FoundGame
	for _, f := range a.libraryGames() {
		if updatableSource(f.Source) && f.AppID == 0 && f.Dir != "" {
			folder = append(folder, f)
		}
	}
	if err := a.store.UpdateCopies(func(l *core.CopyLedger) { l.NoteSeen(folder, time.Now(), dirExists) }); err != nil {
		a.logf("ERROR noting the games in folders: %v", err)
		return
	}
	if n := len(a.store.LoadCopies().Matches()); n > 0 {
		a.emit("newCopies", map[string]any{"count": n})
	}
}

// CopyOffer is one "Is this a new copy?" on the page.
type CopyOffer struct {
	core.CopyMatch
	Saves    []string `json:"saves"`    // the old copy's in-folder saves that come along
	Profiles bool     `json:"profiles"` // the old copy has profiles: they move with it
}

func (a *App) apiNewCopies() {
	go func() {
		var out []CopyOffer
		for _, m := range a.store.LoadCopies().Matches() {
			o := CopyOffer{CopyMatch: m, Saves: []string{}}
			if _, g := a.copyGame(m.Old); g != nil {
				o.Profiles = true
			}
			for rel := range a.oldCopySaves(m.Old) {
				o.Saves = append(o.Saves, rel)
			}
			out = append(out, o)
		}
		if out == nil {
			out = []CopyOffer{}
		}
		a.emit("newCopies", map[string]any{"offers": out})
	}()
}

// copyGame finds the old copy in the Library, and the game with profiles
// started from it, if any.
func (a *App) copyGame(old core.SeenGame) (*core.FoundGame, *core.Game) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	f := core.FoundGame{Key: old.Key, Name: old.Name, Dir: old.Dir, SteamID: old.SteamID}
	for _, lf := range a.libraryGames() {
		if lf.Key == old.Key {
			f = lf
		}
	}
	for i := range cfg.Games {
		g := cfg.Games[i]
		if g.Launch.Type != "exe" {
			continue
		}
		if a.isFoundGame(&g, f) {
			return &f, &g
		}
		if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil && f.Dir != "" && isInside(exe, f.Dir) {
			return &f, &g
		}
	}
	return &f, nil
}

// oldCopySaves are the saves the old copy keeps inside its folder, by path
// inside it: from the folder, or from the safety net's newest copy when
// the folder is gone (as a source to restore from, keyed "net:<id>").
func (a *App) oldCopySaves(old core.SeenGame) map[string]string {
	out := map[string]string{}
	f, _ := a.copyGame(old)
	if dirExists(old.Dir) {
		locs := core.InFolder(a.savesOf(*f).Locs, old.Dir)
		locs = append(locs, a.env.FolderSaves(old.Dir)...)
		for _, l := range locs {
			if rel, err := filepath.Rel(old.Dir, l.Resolved); err == nil && !strings.HasPrefix(rel, "..") && core.HasFiles(l.Resolved) {
				out[filepath.ToSlash(rel)] = l.Resolved
			}
		}
		return out
	}
	vs := netStore().GameVersions(gameSavesKey(*f))
	if len(vs) == 0 {
		return out
	}
	for _, p := range vs[0].Paths {
		if r, err := a.env.Resolve(p); err == nil {
			if rel, err := filepath.Rel(old.Dir, r); err == nil && !strings.HasPrefix(rel, "..") {
				out[filepath.ToSlash(rel)] = "net:" + vs[0].ID
			}
		}
	}
	return out
}

// apiCarryOver moves everything from the old copy to the new one, as a
// "carried" event.
func (a *App) apiCarryOver(oldKey, newKey string) error {
	l := a.store.LoadCopies()
	old, ok1 := l.Games[oldKey]
	nw, ok2 := l.Games[newKey]
	if !ok1 || !ok2 {
		return errors.New("Lootnest doesn't know one of those copies any more. Scan again from the Library.")
	}
	go func() {
		out := map[string]any{"old": oldKey, "new": newKey}
		defer func() { a.emit("carried", out) }()
		updMu.Lock()
		defer updMu.Unlock()
		of, g := a.copyGame(old)
		if f, ok := a.foundByKey(newKey); ok && f.ProcessName != "" {
			if procs, _ := win.ProcessNames(); procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))] {
				out["error"] = f.Name + " is running; close it first."
				return
			}
		}
		keep := filepath.Join(a.store.BackupsDir(), "new-copies", core.CoverKey(nw.Name)+"-"+time.Now().Format("20060102-150405"))
		var carried []core.CarriedSave
		if g != nil {
			// Profiles: the game now starts from the new copy, its locations
			// inside the old folder move to the new one, and the files of
			// whoever owns it go in.
			nf, ok := a.foundByKey(newKey)
			if !ok {
				out["error"] = errFoundGone.Error()
				return
			}
			a.mu.Lock()
			cfg := a.cfg
			gg := cfg.Game(g.ID)
			if gg == nil {
				a.mu.Unlock()
				out["error"] = "That game no longer exists."
				return
			}
			if exe, err := a.env.ResolveExe(gg.Launch.Exe); err == nil {
				if rel, err := filepath.Rel(old.Dir, exe); err == nil && !strings.HasPrefix(rel, "..") && fileExists(filepath.Join(nw.Dir, rel)) {
					gg.Launch.Exe = a.env.Tokenize(filepath.Join(nw.Dir, rel))
				} else if nf.Exe != "" {
					gg.Launch.Exe = nf.Exe
				}
			}
			a.env.RetargetPaths(gg, old.Dir, nw.Dir)
			err := a.store.SaveConfig(cfg)
			snapshot := cfg.Clone()
			keepN := cfg.Settings.BackupsToKeep
			a.mu.Unlock()
			if err != nil {
				out["error"] = err.Error()
				return
			}
			who, err := a.eng.Reseat(snapshot, g.ID, keepN)
			if err != nil {
				out["error"] = "The game now starts from the new copy, but its profile's files couldn't go in: " + err.Error()
				return
			}
			out["profile"] = who
		} else {
			from := a.oldCopySaves(old)
			direct := map[string]string{}
			fromNet := ""
			for rel, src := range from {
				if strings.HasPrefix(src, "net:") {
					fromNet = strings.TrimPrefix(src, "net:")
				} else {
					direct[rel] = src
				}
			}
			var err error
			if carried, err = core.CarrySaves(nw.Dir, direct, filepath.Join(keep, "new copy")); err != nil {
				out["error"] = "Some saves couldn't come along: " + err.Error()
				return
			}
			if fromNet != "" {
				// The old folder is gone: the safety net's newest copy goes in,
				// at the same places inside the new folder.
				for rel := range from {
					dst := filepath.Join(nw.Dir, filepath.FromSlash(rel))
					if core.HasFiles(dst) {
						if err := core.Mirror(dst, filepath.Join(keep, "new copy", filepath.FromSlash(rel))); err != nil {
							out["error"] = err.Error()
							return
						}
					}
				}
				resolve := func(p string) (string, error) {
					r, err := a.env.Resolve(p)
					if err != nil {
						return "", err
					}
					if rel, err := filepath.Rel(old.Dir, r); err == nil && !strings.HasPrefix(rel, "..") {
						return filepath.Join(nw.Dir, rel), nil
					}
					return r, nil
				}
				tmp := filepath.Join(DataDir(), "saves-tmp-"+time.Now().Format("150405.000"))
				if _, err := netStore().RestoreGameVersion(gameSavesKey(*of), fromNet, nil, resolve, tmp); err != nil {
					out["error"] = "The saves Lootnest kept couldn't go into the new copy: " + err.Error()
					return
				}
				for rel := range from {
					carried = append(carried, core.CarriedSave{Rel: rel})
				}
			}
		}
		if err := a.store.RekeyGame(oldKey, newKey); err != nil && err.Error() != "nothing to move" {
			a.logf("ERROR moving %s's history to its new copy: %v", old.Name, err)
		}
		a.mu.Lock()
		if core.RekeyConfig(a.cfg, oldKey, newKey) {
			_ = a.store.SaveConfig(a.cfg)
		}
		a.mu.Unlock()
		_ = a.store.UpdateCopies(func(l *core.CopyLedger) { l.Carried[oldKey] = newKey })
		a.logf("Carried %s over to its new copy in %s (%d save locations)", old.Name, nw.Dir, len(carried))
		out["saves"], out["oldThere"] = carried, dirExists(old.Dir)
		a.emitSessions()
	}()
	return nil
}

func (a *App) apiDismissCopy(oldKey, newKey string) error {
	return a.store.UpdateCopies(func(l *core.CopyLedger) { l.Dismissed[oldKey+"|"+newKey] = true })
}

// apiDeleteOldCopy deletes the old copy's folder once it was carried over:
// its saves are copied to backups first.
func (a *App) apiDeleteOldCopy(oldKey string) error {
	l := a.store.LoadCopies()
	old, ok := l.Games[oldKey]
	if !ok || l.Carried[oldKey] == "" {
		return errors.New("Carry the game over to its new copy first.")
	}
	if !dirExists(old.Dir) {
		return nil
	}
	if err := a.env.CheckSafe(old.Dir); err != nil {
		return fmt.Errorf("Lootnest won't delete %s: %v", old.Dir, err)
	}
	if nw := l.Games[l.Carried[oldKey]]; nw.Dir != "" && (isInside(nw.Dir, old.Dir) || isInside(old.Dir, nw.Dir)) {
		return errors.New("The new copy is inside the old folder, so it can't be deleted.")
	}
	go func() {
		out := map[string]any{"old": oldKey, "deleted": true}
		defer func() { a.emit("carried", out) }()
		keep := filepath.Join(a.store.BackupsDir(), "old-copies", core.CoverKey(old.Name)+"-"+time.Now().Format("20060102-150405"))
		for rel, src := range a.oldCopySaves(old) {
			if strings.HasPrefix(src, "net:") {
				continue
			}
			if err := core.Mirror(src, filepath.Join(keep, filepath.FromSlash(rel))); err != nil {
				out["error"] = "Nothing was deleted: Lootnest couldn't copy its saves first (" + err.Error() + ")."
				return
			}
		}
		if err := os.RemoveAll(old.Dir); err != nil {
			out["error"] = "Some of the old folder couldn't be deleted (" + err.Error() + "). Its saves are in Lootnest's backups."
			return
		}
		a.logf("Deleted the old copy of %s in %s; its saves are in %s", old.Name, old.Dir, keep)
		a.startScan()
	}()
	return nil
}

// apiKeepSavesNow copies a game's saves to backups, before the user
// reinstalls it.
func (a *App) apiKeepSavesNow(key string) error {
	t, err := a.updTarget(key)
	if err != nil {
		return err
	}
	go func() {
		out := map[string]any{"key": key, "kept": true}
		if err := a.keepSavesBeforeUpdate(t); err != nil {
			out["error"] = err.Error()
		}
		a.emit("updateDone", out)
	}()
	return nil
}
