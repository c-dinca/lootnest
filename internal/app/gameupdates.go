//go:build windows

package app

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// The update centre (core/gameupdate.go). Games no store keeps up to date
// (in a folder, added by hand, from a shortcut or Hydra) list whether a
// newer version is out, from Steam's news, and take an update the user
// brings: a folder or a .zip. Lootnest copies the saves and every file it
// replaces first, and the update can be undone. It never finds or
// downloads an update itself.
// ---------------------------------------------------------------------------

var updMu sync.Mutex // one update applied or undone at a time

// updTarget is a game an update can go into.
type updTarget struct {
	Key     string `json:"key"` // "found:<library key>" or "game:<id>"
	Name    string `json:"name"`
	Source  string `json:"source"`
	Dir     string `json:"dir"`
	Exe     string `json:"-"`
	Process string `json:"-"`
	SteamID int    `json:"-"`
	found   *core.FoundGame
	game    *core.Game
}

// updatableSource: games their own launcher doesn't update.
func updatableSource(s string) bool {
	switch s {
	case "folder", "added", "shortcut", "hydra", "":
		return true
	}
	return false
}

// updTargets lists the games an update can go into.
func (a *App) updTargets() []updTarget {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	all := a.libraryGames()
	_, containers := gameFolders(a.env)
	var out []updTarget
	for _, f := range a.visibleFound(all) {
		f := f
		if !updatableSource(f.Source) || f.Dir == "" || f.AppID > 0 {
			continue
		}
		exe, _ := a.env.ResolveExe(f.Exe)
		out = append(out, updTarget{Key: "found:" + f.Key, Name: f.Name, Source: f.Source, Dir: f.Dir, Exe: exe, Process: f.ProcessName, SteamID: f.SteamID, found: &f})
	}
	src := a.gameSources(cfg)
	for i := range cfg.Games {
		g := cfg.Games[i]
		if g.Launch.Type != "exe" || !updatableSource(src[g.ID]) {
			continue
		}
		exe, err := a.env.ResolveExe(g.Launch.Exe)
		if err != nil {
			continue
		}
		t := updTarget{Key: "game:" + g.ID, Name: g.Name, Source: src[g.ID], Exe: exe, Process: g.ProcessName, game: &g}
		for _, f := range all {
			if a.isFoundGame(&g, f) && len(f.Dir) > len(t.Dir) {
				t.Dir, t.SteamID = f.Dir, f.SteamID
			}
		}
		if t.Dir == "" {
			t.Dir = core.GameFolderOf(exe, containers)
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func (a *App) updTarget(key string) (updTarget, error) {
	for _, t := range a.updTargets() {
		if t.Key == key {
			return t, nil
		}
	}
	return updTarget{}, errors.New("Lootnest can't update that game: its own launcher does, or it isn't in the Library any more.")
}

func (t updTarget) running() bool {
	if t.Process == "" {
		return false
	}
	procs, _ := win.ProcessNames()
	return procs[strings.ToLower(strings.TrimSuffix(t.Process, ".exe"))]
}

// updatesDir is where one game's applied updates are kept.
func (a *App) updatesDir(t updTarget) string {
	h := sha1.Sum([]byte(strings.ToLower(t.Key)))
	return filepath.Join(a.store.BackupsDir(), "updates", core.CoverKey(t.Name)+"-"+hex.EncodeToString(h[:4]))
}

// UpdateRow is one game on the Updates page.
type UpdateRow struct {
	updTarget
	Newer   *core.NewsItem       `json:"newer,omitempty"`
	Changed time.Time            `json:"changed"` // when its files last changed, roughly
	Applied []core.AppliedUpdate `json:"applied"`
	NoNews  bool                 `json:"noNews,omitempty"` // not on the Steam store, so Lootnest can't tell
}

// apiUpdates answers with an "updates" event: every game an update can go
// into, and whether a newer version is out (with "Show game updates and
// news" on).
func (a *App) apiUpdates() {
	go func() {
		a.mu.Lock()
		newsOn := a.cfg.Settings.GameNews
		a.mu.Unlock()
		ts := a.updTargets()
		rows := make([]UpdateRow, len(ts))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		for i, t := range ts {
			rows[i] = UpdateRow{updTarget: t, Applied: core.GameUpdates(a.updatesDir(t))}
			if rows[i].Applied == nil {
				rows[i].Applied = []core.AppliedUpdate{}
			}
			if t.Exe != "" {
				rows[i].Changed = newestFile(filepath.Dir(t.Exe))
			}
			for _, u := range rows[i].Applied {
				if u.State == "applied" && u.At.After(rows[i].Changed) {
					rows[i].Changed = u.At
				}
			}
			if !newsOn {
				continue
			}
			wg.Add(1)
			go func(r *UpdateRow) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				id := r.SteamID
				if id <= 0 {
					id = core.FolderAppID(r.Dir)
				}
				if id <= 0 {
					id, _ = core.StoreAppID(fetchURL, newsDir(), r.Name)
				}
				if id <= 0 {
					r.NoNews = true
					return
				}
				if n, err := core.FetchNews(fetchURL, newsDir(), id); err == nil || len(n.Items) > 0 {
					r.Newer = core.NewerThanCopy(n.Items, r.Changed)
				}
			}(&rows[i])
		}
		wg.Wait()
		a.emit("updates", map[string]any{"games": rows, "newsOn": newsOn})
	}()
}

// apiUpdatePlan reads an update the user chose and says what applying it
// would do, as an "updatePlan" event.
func (a *App) apiUpdatePlan(key, path string) {
	go func() {
		out := map[string]any{"key": key, "path": path}
		defer func() { a.emit("updatePlan", out) }()
		t, err := a.updTarget(key)
		if err != nil {
			out["error"] = err.Error()
			return
		}
		p, err := core.PlanUpdate(t.Dir, path)
		if err != nil {
			out["error"] = sentence(err)
			return
		}
		p.Files = nil // the counts and folders are enough for the page
		out["plan"], out["dir"], out["free"] = p, t.Dir, win.DiskFree(t.Dir)
	}()
}

// apiUpdateApply applies the update at path to the game. The saves and the
// files it replaces are copied first. The answer is an "updateDone" event.
func (a *App) apiUpdateApply(key, path string) error {
	t, err := a.updTarget(key)
	if err != nil {
		return err
	}
	if t.running() {
		return fmt.Errorf("%s is running; close it first.", t.Name)
	}
	go func() {
		out := map[string]any{"key": key}
		defer func() { a.emit("updateDone", out) }()
		updMu.Lock()
		defer updMu.Unlock()
		p, err := core.PlanUpdate(t.Dir, path) // again: it may have changed since
		if err != nil {
			out["error"] = sentence(err)
			return
		}
		if free := win.DiskFree(t.Dir); free >= 0 && free < p.Bytes+p.Kept+(64<<20) {
			out["error"] = fmt.Sprintf("There isn't enough room on %s: the update and the copy of the files it replaces need %s.", strings.ToUpper(filepath.VolumeName(t.Dir)), gb(p.Bytes+p.Kept))
			return
		}
		if err := a.keepSavesBeforeUpdate(t); err != nil {
			out["error"] = "Nothing was changed: Lootnest couldn't copy the saves first (" + err.Error() + ")."
			return
		}
		keep := filepath.Join(a.updatesDir(t), time.Now().Format("20060102-150405"))
		u, err := core.ApplyUpdate(t.Dir, p, keep, t.Name, time.Now())
		if err != nil {
			a.logf("ERROR updating %s from %s: %v", t.Name, path, err)
			out["error"] = sentence(err)
			return
		}
		a.logf("Updated %s from %s: %d files added, %d replaced (the old ones are in %s)", t.Name, path, len(u.Added), len(u.Replaced), keep)
		out["update"] = u
	}()
	return nil
}

// keepSavesBeforeUpdate copies the game's saves to backups first.
func (a *App) keepSavesBeforeUpdate(t updTarget) error {
	if t.game != nil {
		a.mu.Lock()
		keep := a.cfg.Settings.BackupsToKeep
		a.mu.Unlock()
		_, err := a.eng.Backup(t.game, keep)
		return err
	}
	locs := a.savesOf(*t.found).Locs
	locs = append(locs, a.netLocs(*t.found)...)
	dir := filepath.Join(a.store.BackupsDir(), "saves-"+gameSavesKey(*t.found), time.Now().Format("20060102-150405")+"-before-update")
	seen := map[string]bool{}
	for i, l := range locs {
		if seen[strings.ToLower(l.Resolved)] {
			continue
		}
		seen[strings.ToLower(l.Resolved)] = true
		if err := core.Mirror(l.Resolved, filepath.Join(dir, fmt.Sprint(i))); err != nil {
			return err
		}
	}
	return nil
}

// apiUpdateUndo takes an applied update back out, as an "updateDone" event.
func (a *App) apiUpdateUndo(key, id string) error {
	t, err := a.updTarget(key)
	if err != nil {
		return err
	}
	if t.running() {
		return fmt.Errorf("%s is running; close it first.", t.Name)
	}
	if id == "" || id != filepath.Base(id) {
		return errors.New("unknown update")
	}
	go func() {
		out := map[string]any{"key": key, "undone": true}
		defer func() { a.emit("updateDone", out) }()
		updMu.Lock()
		defer updMu.Unlock()
		u, err := core.UndoUpdate(t.Dir, filepath.Join(a.updatesDir(t), id))
		if err != nil {
			a.logf("ERROR undoing an update of %s: %v", t.Name, err)
			out["error"] = sentence(err)
			return
		}
		a.logf("Undid the update of %s from %s", t.Name, u.Source)
		out["update"] = u
	}()
	return nil
}

// sentence makes an error read as a sentence on the page.
func sentence(err error) string {
	s := err.Error()
	if s == "" {
		return s
	}
	s = strings.ToUpper(s[:1]) + s[1:]
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}
