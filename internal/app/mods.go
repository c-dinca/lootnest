//go:build windows

package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Mods (core/mods.go): one set of mods per game, put into its folder by
// copy, with every game file they replace kept and the way back one click.
// The user brings each mod; Lootnest never downloads one. Changing the
// game's folder is refused while the game runs and in safe mode, and the
// saves are copied before mods go in.
// ---------------------------------------------------------------------------

const modsSafeNote = "Lootnest is in safe mode, which leaves the game's folder as it is. Restart it normally to change its mods."

var (
	modsMu  sync.Mutex
	modsNow string // the game's folder a mods run is changing
)

func (a *App) modsRoot(key string) string { return core.ModsDir(DataDir(), key) }

// modTarget is the Library game a key names, with its folder.
func (a *App) modTarget(key string) (updTarget, error) {
	t := updTarget{Key: key}
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		g := a.cfg.Clone().Game(id)
		a.mu.Unlock()
		if g != nil {
			t.Name, t.Process, t.game = g.Name, g.ProcessName, g
			if g.Launch.Type == "steam" {
				t.Dir = a.steamInstallDir(g.Launch.AppID)
			} else if exe, err := a.env.ResolveExe(g.Launch.Exe); err == nil {
				_, containers := gameFolders(a.env)
				t.Exe = exe
				t.Dir, _ = a.exeGameDir(g, exe, a.libraryGames(), containers)
			}
		}
	} else if f, ok := a.foundByKey(key); ok {
		t.Name, t.Dir, t.Process, t.found = f.Name, f.Dir, f.ProcessName, &f
		t.Exe, _ = a.env.ResolveExe(f.Exe)
	}
	if t.Name == "" {
		return t, errors.New("That game isn't in the Library any more.")
	}
	if t.Dir == "" {
		return t, fmt.Errorf("Lootnest doesn't know where %s's folder is, so it can't put mods into it.", t.Name)
	}
	return t, nil
}

// ModRow is one mod on the game's Mods tab.
type ModRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
	Into   string `json:"into,omitempty"`
	AtRoot bool   `json:"atRoot,omitempty"`
	On     bool   `json:"on"`
}

func modRow(m *core.Mod, on bool) ModRow {
	r := ModRow{ID: m.ID, Name: m.Name, Files: len(m.Files), Into: m.Into, AtRoot: m.AtRoot, On: on}
	for _, f := range m.Files {
		r.Bytes += f.Size
	}
	return r
}

// apiMods is a game's Mods tab: its mods in order, what's in its folder
// now, and why it can't have mods, if it can't.
func (a *App) apiMods(key string) (map[string]any, error) {
	t, err := a.modTarget(key)
	if err != nil {
		return nil, err
	}
	root := a.modsRoot(key)
	rows := []ModRow{}
	for _, e := range core.LoadModSet(root) {
		if m, err := core.LoadMod(root, e.ID); err == nil {
			rows = append(rows, modRow(m, e.On))
		}
	}
	out := map[string]any{"dir": t.Dir, "mods": rows, "pending": core.ModJournalPending(root)}
	if a.safe {
		out["safe"] = modsSafeNote
	}
	if d := core.LoadDeployed(root); d != nil {
		out["deployed"] = map[string]any{"mods": d.Mods}
	}
	if err := core.ModsRefused(t.Dir); err != nil {
		out["refused"] = sentence(err)
	}
	return out, nil
}

// apiAddMod keeps a copy of the mod at path for the game and answers with a
// "modPlan" event: what putting it in would do, or why it can't go in. The
// game's folder doesn't change; the mod joins the list on ps_modToggle.
func (a *App) apiAddMod(key, path string) error {
	t, err := a.modTarget(key)
	if err != nil {
		return err
	}
	if err := core.ModsRefused(t.Dir); err != nil {
		return errors.New(sentence(err))
	}
	go func() {
		out := map[string]any{"key": key, "path": path}
		defer func() { a.emit("modPlan", out) }()
		updMu.Lock()
		defer updMu.Unlock()
		root := a.modsRoot(key)
		m, err := core.AddMod(root, t.Dir, path, time.Now())
		if errors.Is(err, core.ErrTooBig) {
			out["error"] = "That mod holds more than 2 GB once it's opened, more than Lootnest keeps for a mod."
			return
		}
		if err != nil {
			out["error"] = sentence(err)
			return
		}
		out["mod"] = modRow(m, false)
		p, err := core.PlanMods(t.Dir, t.Exe, []*core.Mod{m}, core.LoadDeployed(root))
		if err != nil {
			_ = core.RemoveMod(root, m.ID)
			out["refused"] = sentence(err)
			return
		}
		out["plan"] = p
	}()
	return nil
}

// apiModToggle turns a mod on or off in the game's set; a mod just added
// joins the end of the list. The game's folder changes on ps_modsApply.
func (a *App) apiModToggle(key, id string, on bool) error {
	root := a.modsRoot(key)
	set := core.LoadModSet(root)
	i := slices.IndexFunc(set, func(e core.ModEntry) bool { return e.ID == id })
	if i < 0 {
		if _, err := core.LoadMod(root, id); err != nil || !on {
			return errors.New("That mod isn't kept any more.")
		}
		set = append(set, core.ModEntry{ID: id, On: true})
	} else {
		set[i].On = on
	}
	return core.SaveModSet(root, set)
}

// apiModOrder moves a mod one place up (by -1) or down (by 1). A later mod
// wins where two have the same file.
func (a *App) apiModOrder(key, id string, by int) error {
	root := a.modsRoot(key)
	set := core.LoadModSet(root)
	i := slices.IndexFunc(set, func(e core.ModEntry) bool { return e.ID == id })
	j := i + by
	if i < 0 || (by != -1 && by != 1) || j < 0 || j >= len(set) {
		return nil
	}
	set[i], set[j] = set[j], set[i]
	return core.SaveModSet(root, set)
}

// apiModRemove deletes Lootnest's copy of a mod. A mod with files in the
// game's folder comes out of it first (the other mods that are on go back
// in), and the answer is a "modsDone" event; then it returns true.
func (a *App) apiModRemove(key, id string) (bool, error) {
	root := a.modsRoot(key)
	d := core.LoadDeployed(root)
	if d == nil || !slices.Contains(d.Mods, id) {
		if core.ModJournalPending(root) || !updMu.TryLock() {
			return false, errors.New("Lootnest is changing a game's folder. Try again once it's done.")
		}
		defer updMu.Unlock()
		return false, core.RemoveMod(root, id)
	}
	return true, a.modsRun(key, "remove", func(t updTarget, step func(int, int)) (core.ModsResult, error) {
		res, err := a.applyModSet(t, id, step)
		if err == nil {
			err = core.RemoveMod(root, id)
		}
		return res, err
	})
}

// apiModsApply puts the game's mods that are on into its folder, in order,
// after taking out what's there now. The saves are copied first.
func (a *App) apiModsApply(key string) error {
	return a.modsRun(key, "apply", func(t updTarget, step func(int, int)) (core.ModsResult, error) {
		return a.applyModSet(t, "", step)
	})
}

// applyModSet puts the game's mods that are on, all but skip, into its
// folder, copying the saves first.
func (a *App) applyModSet(t updTarget, skip string, step func(int, int)) (core.ModsResult, error) {
	root := a.modsRoot(t.Key)
	var ids []string
	for _, e := range core.LoadModSet(root) {
		if e.On && e.ID != skip {
			ids = append(ids, e.ID)
		}
	}
	mods, err := core.LoadMods(root, ids)
	if err != nil {
		return core.ModsResult{}, err
	}
	if len(mods) > 0 {
		if err := a.keepSavesBefore(t, "mods"); err != nil {
			return core.ModsResult{}, fmt.Errorf("nothing was changed: Lootnest couldn't copy the saves first (%v)", err)
		}
	}
	return core.ApplyMods(root, t.Dir, t.Exe, mods, time.Now(), step)
}

// apiModsRestore puts the game back as it was: every mod file out, every
// replaced game file back.
func (a *App) apiModsRestore(key string) error {
	return a.modsRun(key, "restore", func(t updTarget, step func(int, int)) (core.ModsResult, error) {
		return core.ApplyMods(a.modsRoot(key), t.Dir, t.Exe, nil, time.Now(), step)
	})
}

// modsRun changes a game's folder in the background, one change at a time,
// with "modsProgress" events and a "modsDone" event at the end.
func (a *App) modsRun(key, what string, run func(t updTarget, step func(done, total int)) (core.ModsResult, error)) error {
	if a.safe {
		return errors.New(modsSafeNote)
	}
	t, err := a.modTarget(key)
	if err != nil {
		return err
	}
	if a.inUse(t) {
		return fmt.Errorf("%s is running; close it first.", t.Name)
	}
	go func() {
		out := map[string]any{"key": key, "what": what}
		defer func() { a.emit("modsDone", out) }()
		updMu.Lock()
		defer updMu.Unlock()
		modsMu.Lock()
		modsNow = t.Dir
		modsMu.Unlock()
		defer func() {
			modsMu.Lock()
			modsNow = ""
			modsMu.Unlock()
		}()
		var last time.Time
		res, err := run(t, func(done, total int) {
			if done == total || time.Since(last) > 250*time.Millisecond {
				last = time.Now()
				a.emit("modsProgress", map[string]any{"key": key, "done": done, "total": total})
			}
		})
		out["result"] = res
		if err != nil {
			a.logf("ERROR mods (%s) for %s: %v", what, t.Name, err)
			out["error"] = sentence(err)
			return
		}
		a.logf("Mods (%s) for %s done; %d changed files kept in %s", what, t.Name, res.Changed, res.ChangedDir)
	}()
	return nil
}

// finishMods finishes the mods runs cut short when Lootnest closed or the
// PC turned off, before the games are played; not in safe mode, and not for
// a game that's running, which waits for Apply or the next start.
func (a *App) finishMods() {
	if a.safe {
		return
	}
	updMu.Lock()
	defer updMu.Unlock()
	procs, _ := win.ProcessNames()
	finished := false
	for _, root := range core.ModJournals(DataDir()) {
		game := core.ModJournalGame(root)
		if exesRunning(procs, game) {
			a.logf("The last mods run in %s waits: a program from %s is running", root, game)
			continue
		}
		if _, res, err := core.FinishModJournal(root); err != nil {
			a.logf("ERROR finishing the last mods run in %s: %v", root, err)
		} else {
			finished = true
			a.logf("Finished the last mods run in %s, cut short when Lootnest closed; %d changed files kept", root, res.Changed)
		}
	}
	if finished {
		a.emit("modsFinished", nil)
	}
}

// exesRunning says whether a program from the game's folder dir runs.
func exesRunning(procs map[string]bool, dir string) bool {
	for _, e := range core.GameExes(dir, "") {
		if procs[strings.TrimSuffix(strings.ToLower(filepath.Base(e)), ".exe")] {
			return true
		}
	}
	return false
}

// errModding is why a game that starts from p can't start now: Lootnest is
// changing its mods, or the last change to them didn't finish. Else nil.
func (a *App) errModding(p string) error {
	modsMu.Lock()
	dir := modsNow
	modsMu.Unlock()
	if dir != "" && isInside(p, dir) {
		return errors.New("Lootnest is changing this game's mods. Start it once that's done.")
	}
	for _, root := range core.ModJournals(DataDir()) {
		if g := core.ModJournalGame(root); g != "" && isInside(p, g) {
			return errors.New("The last change to this game's mods didn't finish. Apply them, or put the game back as it was, on its Mods tab, then start it.")
		}
	}
	return nil
}
