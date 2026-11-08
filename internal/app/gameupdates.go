//go:build windows

package app

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
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
		t.Dir, t.SteamID = a.exeGameDir(&g, exe, all, containers)
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// exeGameDir is the folder of a game with profiles that starts from exe: the
// deepest a scan found it in, else the one exe's path suggests.
func (a *App) exeGameDir(g *core.Game, exe string, all []core.FoundGame, containers []string) (dir string, steamID int) {
	for _, f := range all {
		if a.isFoundGame(g, f) && len(f.Dir) > len(dir) {
			dir, steamID = f.Dir, f.SteamID
		}
	}
	if dir == "" {
		dir = core.GameFolderOf(exe, containers)
	}
	return dir, steamID
}

func (a *App) updTarget(key string) (updTarget, error) {
	for _, t := range a.updTargets() {
		if t.Key == key {
			return t, nil
		}
	}
	if t, ok := a.gogUpdTarget(key); ok {
		return t, nil
	}
	return updTarget{}, errors.New("Lootnest can't update that game: its own launcher does, or it isn't in the Library any more.")
}

func (t updTarget) running() bool {
	if t.Process == "" {
		return false
	}
	procs, _ := win.ProcessNames()
	return t.runningIn(procs)
}

func (t updTarget) runningIn(procs map[string]bool) bool {
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
	Version string               `json:"version,omitempty"` // this copy's, as far as Lootnest knows
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
			rows[i].Version = core.CopyVersion(t.Dir, rows[i].Applied)
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
		if isInstaller(path) {
			info := a.installerInfo(t, path)
			out["plan"], out["dir"], out["free"] = info, t.Dir, info["free"]
			return
		}
		p, err := a.planUpdate(t, path)
		if err != nil {
			out["error"] = sentence(err)
			return
		}
		from, to := core.UpdateVersions(p.Name)
		out["check"] = core.CheckVersions(core.CopyVersion(t.Dir, core.GameUpdates(a.updatesDir(t))), from, to)
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
		if !a.modsOffBefore(t, out) {
			return
		}
		p, err := a.planUpdate(t, path) // again: it may have changed since
		if err != nil {
			out["error"] = sentence(err)
			return
		}
		defer clearStaging(t)
		if free := win.DiskFree(t.Dir); free >= 0 && free < p.Bytes+p.Kept+(64<<20) {
			out["error"] = fmt.Sprintf("There isn't enough room on %s: the update and the copy of the files it replaces need %s.", strings.ToUpper(filepath.VolumeName(t.Dir)), gb(p.Bytes+p.Kept))
			return
		}
		if err := a.keepSavesBefore(t, "update"); err != nil {
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
		a.itchApplied(key, path, u.ID)
	}()
	return nil
}

// stagingDir is where a .7z or .rar update is extracted: next to the game's
// folder, on the same drive, so there's room and moving files in is quick.
func stagingDir(t updTarget) string {
	h := sha1.Sum([]byte(strings.ToLower(t.Dir)))
	return filepath.Join(filepath.Dir(t.Dir), ".lootnest-update-"+hex.EncodeToString(h[:4]))
}

func clearStaging(t updTarget) { _ = os.RemoveAll(stagingDir(t)) }

var (
	stagedMu   sync.Mutex
	stagedFrom = map[string]string{} // staging folder -> the archive and its time and size it holds
)

// planUpdate plans an update, extracting a .7z or .rar first (once: the
// same archive isn't extracted again for Apply).
func (a *App) planUpdate(t updTarget, path string) (*core.UpdatePlan, error) {
	if !core.NeedsStaging(path) {
		p, err := core.PlanUpdate(t.Dir, path)
		if err == nil {
			p.Name = filepath.Base(path)
		}
		return p, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, errors.New("Lootnest can't find that update")
	}
	stamp := fmt.Sprintf("%s|%d|%d", strings.ToLower(path), st.Size(), st.ModTime().UnixNano())
	dir := stagingDir(t)
	stagedMu.Lock()
	ready := stagedFrom[dir] == stamp && dirExists(dir)
	stagedMu.Unlock()
	if !ready {
		if free := win.DiskFree(filepath.Dir(t.Dir)); free >= 0 && free < st.Size()*3 {
			return nil, fmt.Errorf("there isn't enough room on %s to open the update", strings.ToUpper(filepath.VolumeName(t.Dir)))
		}
		a.emit("updatePlan", map[string]any{"key": t.Key, "path": path, "opening": true})
		if _, err := core.StageUpdate(path, dir); err != nil {
			return nil, err
		}
		stagedMu.Lock()
		stagedFrom[dir] = stamp
		stagedMu.Unlock()
	}
	p, err := core.PlanUpdate(t.Dir, dir)
	if err != nil {
		return nil, err
	}
	p.Source, p.Name, p.Kind = path, filepath.Base(path), strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	return p, nil
}

// keepSavesBefore copies the game's saves to backups before an update or
// mods (what) change its folder.
func (a *App) keepSavesBefore(t updTarget, what string) error {
	if t.game != nil {
		a.mu.Lock()
		keep := a.cfg.Settings.BackupsToKeep
		a.mu.Unlock()
		_, err := a.eng.Backup(t.game, keep)
		return err
	}
	locs := a.savesOf(*t.found).Locs
	locs = append(locs, a.netLocs(*t.found)...)
	dir := filepath.Join(a.store.BackupsDir(), "saves-"+gameSavesKey(*t.found), time.Now().Format("20060102-150405")+"-before-"+what)
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
	for _, u := range core.GameUpdates(a.updatesDir(t)) { // newest first
		if u.State != "applied" {
			continue
		}
		if u.ID != id {
			return fmt.Errorf("Undo the newer update first (%s): updates come off one at a time, newest first.", u.Source)
		}
		break
	}
	go func() {
		out := map[string]any{"key": key, "undone": true}
		defer func() { a.emit("updateDone", out) }()
		updMu.Lock()
		defer updMu.Unlock()
		if !a.modsOffBefore(t, out) {
			return
		}
		u, err := core.UndoUpdate(t.Dir, filepath.Join(a.updatesDir(t), id))
		if err != nil {
			a.logf("ERROR undoing an update of %s: %v", t.Name, err)
			out["error"] = sentence(err)
			return
		}
		a.logf("Undid the update of %s from %s", t.Name, u.Source)
		out["update"] = u
		a.itchUndone(key, id)
		a.gogUndone(key, id)
	}()
	return nil
}

// modsOffBefore takes the game's mods out before an update changes its
// folder, and says whether to go on. The "updateDone" event then carries
// modsOff: they're not put back, as they may not fit the new version.
// Called with updMu held.
func (a *App) modsOffBefore(t updTarget, out map[string]any) bool {
	off, err := a.modsOffFor(t)
	if err != nil {
		out["error"] = err.Error()
		return false
	}
	if off {
		out["modsOff"] = true
	}
	return true
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

// ------------------------------------------------------------ installers --

// apiUpdateInstaller runs an update's installer (an .exe the user chose)
// for a game and records what it changed. With full, a copy of the game's
// folder is kept first so the update can be undone; only the files the
// installer changed stay in the record afterwards. Progress comes as
// "updateProgress" events and the end as "updateDone".
func (a *App) apiUpdateInstaller(key, path string, full bool) error {
	t, err := a.updTarget(key)
	if err != nil {
		return err
	}
	if !isInstaller(path) || !fileExists(path) {
		return errors.New("Choose the update's installer, an .exe.")
	}
	if isInside(path, t.Dir) {
		return errors.New("That's a program inside the game's own folder, not an update's installer.")
	}
	if t.running() {
		return fmt.Errorf("%s is running; close it first.", t.Name)
	}
	go func() {
		out := map[string]any{"key": key}
		defer func() { a.emit("updateDone", out) }()
		step := func(s string) { a.emit("updateProgress", map[string]any{"key": key, "step": s}) }
		updMu.Lock()
		defer updMu.Unlock()
		if !a.modsOffBefore(t, out) {
			return
		}
		keep := filepath.Join(a.updatesDir(t), time.Now().Format("20060102-150405"))
		fullCopy := ""
		if full {
			size := core.Size(t.Dir)
			if free := win.DiskFree(a.store.BackupsDir()); free >= 0 && free < size+(256<<20) {
				out["error"] = fmt.Sprintf("There isn't enough room for a copy of the game (%s). Run it without one, or make room first.", gb(size))
				return
			}
			step("copying")
			fullCopy = filepath.Join(keep, "full")
			if err := core.Mirror(t.Dir, fullCopy); err != nil {
				_ = os.RemoveAll(keep)
				out["error"] = "Nothing was changed: Lootnest couldn't copy the game first (" + err.Error() + ")."
				return
			}
		}
		if err := a.keepSavesBefore(t, "update"); err != nil {
			_ = os.RemoveAll(keep)
			out["error"] = "Nothing was changed: Lootnest couldn't copy the saves first (" + err.Error() + ")."
			return
		}
		before, _ := core.SnapshotDir(t.Dir)
		step("installer")
		a.logf("Running %s to update %s", path, t.Name)
		// GOG's installer runs without questions, so only its exit code
		// says it stopped part way.
		args := gogArgs(key, path, t.Dir)
		code, err := win.RunAndWait(a.hwnd, path, args, filepath.Dir(path))
		if err != nil {
			_ = os.RemoveAll(keep)
			out["error"] = sentence(err)
			return
		}
		stopped := ""
		if args != "" && code != 0 {
			stopped = fmt.Sprintf("GOG's installer stopped before it finished (code %d).", code)
			a.logf("ERROR updating %s: the installer ended with code %d", t.Name, code)
		}
		// Some installers hand the work to another process: wait until the
		// folder stops changing.
		step("checking")
		after, _ := core.SnapshotDir(t.Dir)
		for i := 0; i < 120; i++ {
			time.Sleep(5 * time.Second)
			next, _ := core.SnapshotDir(t.Dir)
			if sameSnapshot(after, next) {
				break
			}
			after = next
		}
		added, changed, removed := core.DiffSnapshots(before, after)
		if len(added)+len(changed)+len(removed) == 0 {
			_ = os.RemoveAll(keep)
			out["error"] = "The installer didn't change anything in " + t.Dir + ". It may have installed somewhere else."
			if stopped != "" {
				out["error"] = stopped + " Nothing in " + t.Dir + " changed."
			}
			return
		}
		u, err := core.RecordInstallerUpdate(t.Dir, fullCopy, keep, t.Name, filepath.Base(path), added, changed, removed, time.Now())
		if fullCopy != "" {
			_ = os.RemoveAll(fullCopy)
		}
		if err != nil {
			out["error"] = sentence(err)
			return
		}
		a.logf("%s's installer changed %d files, added %d and removed %d", t.Name, len(changed), len(added), len(removed))
		out["update"] = u
		if stopped != "" {
			out["error"] = stopped + " Undo puts back the files it changed."
			return
		}
		a.itchApplied(key, path, u.ID)
		a.gogApplied(key, path, u.ID)
	}()
	return nil
}

// isInstaller: an update's installer, an .exe (or the .msi a store sends).
func isInstaller(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".exe" || ext == ".msi"
}

func sameSnapshot(a, b map[string]core.FileStamp) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// installerInfo answers a plan for an installer: the game's size and the
// room for a copy of it.
func (a *App) installerInfo(t updTarget, path string) map[string]any {
	return map[string]any{"kind": "installer", "source": path, "name": filepath.Base(path), "gameSize": core.Size(t.Dir), "free": win.DiskFree(a.store.BackupsDir())}
}

// ------------------------------------------------------ is it working? --

// watchFirstStart follows a game's first start after an update: if it
// closes within 20 seconds, the update probably broke it, and the user is
// offered Undo. Once it runs longer, the update is fine.
func (a *App) watchFirstStart(key string) {
	t, err := a.updTarget(key)
	if err != nil || t.Process == "" {
		return
	}
	var u *core.AppliedUpdate
	for _, x := range core.GameUpdates(a.updatesDir(t)) {
		x := x
		if x.State == "applied" {
			if x.Verify {
				u = &x
			}
			break
		}
	}
	if u == nil {
		return
	}
	go func() {
		name := strings.ToLower(strings.TrimSuffix(t.Process, ".exe"))
		up := func() bool { procs, _ := win.ProcessNames(); return procs[name] }
		started := time.Time{}
		for i := 0; i < 90 && started.IsZero(); i++ {
			time.Sleep(time.Second)
			if up() {
				started = time.Now()
			}
		}
		if started.IsZero() {
			return // never seen: nothing to tell
		}
		for time.Since(started) < 20*time.Second {
			time.Sleep(2 * time.Second)
			if !up() {
				u.Broke = true
				_ = core.SaveUpdate(u)
				a.logf("%s closed %s after starting, its first start since the update from %s", t.Name, time.Since(started).Round(time.Second), u.Source)
				a.notify(t.Name+" closed right after starting", "It's the first start since the update from "+u.Source+". If it doesn't work, Undo puts the game back on the Updates page.")
				a.emit("updateBroke", map[string]any{"key": key, "id": u.ID})
				return
			}
		}
		u.Verify, u.Broke = false, false
		_ = core.SaveUpdate(u)
	}()
}
