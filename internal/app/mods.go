//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Mods (core/mods.go): a set of mods per game, and per profile and run for
// games with profiles, put into its folder by copy, with every game file
// they replace kept and the way back one click. Play puts in the set of
// whoever plays, after their saves. The user brings each mod; Lootnest
// never downloads one. Changing the game's folder is refused while the game
// runs and in safe mode, and the saves are copied before mods go in.
// ---------------------------------------------------------------------------

const modsSafeNote = "Lootnest is in safe mode, which leaves the game's folder as it is. Restart it normally to change its mods."

var (
	modsMu  sync.Mutex
	modsNow string // the game's folder a mods run is changing
)

func (a *App) modsRoot(key string) string { return core.ModsDir(DataDir(), key) }

// modSet is the set owner (a profile id, "" for the game) plays with, and
// whose set it is.
func (a *App) modSet(root, owner string) (string, []core.ModEntry) {
	of := ""
	a.mu.Lock()
	if p := a.cfg.Player(owner); p != nil {
		of = p.Of
	}
	a.mu.Unlock()
	return core.ModSetFor(root, owner, of)
}

// modding marks dir as having its mods changed, so games starting from it
// wait; the func it returns clears that.
func modding(dir string) func() {
	modsMu.Lock()
	modsNow = dir
	modsMu.Unlock()
	return func() {
		modsMu.Lock()
		modsNow = ""
		modsMu.Unlock()
	}
}

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

// apiMods is a game's Mods tab: each set's mods in order ("" the game's,
// else a profile's own), whose set each profile plays with, what's in its
// folder now, and why it can't have mods, if it can't.
func (a *App) apiMods(key string) (map[string]any, error) {
	t, err := a.modTarget(key)
	if err != nil {
		return nil, err
	}
	root := a.modsRoot(key)
	kept := map[string]*core.Mod{}
	all := core.LoadModSets(root)
	sets := map[string][]ModRow{"": nil}
	for o := range all {
		sets[o] = nil
	}
	for o := range sets {
		_, set := core.ModSetIn(all, o, "")
		rows := []ModRow{}
		for _, e := range set {
			if kept[e.ID] == nil {
				kept[e.ID], _ = core.LoadMod(root, e.ID)
			}
			if m := kept[e.ID]; m != nil {
				rows = append(rows, modRow(m, e.On))
			}
		}
		sets[o] = rows
	}
	uses := map[string]string{}
	if t.game != nil {
		a.mu.Lock()
		players := slices.Clone(a.cfg.Players)
		a.mu.Unlock()
		for _, p := range players {
			uses[p.ID], _ = core.ModSetIn(all, p.ID, p.Of)
		}
	}
	out := map[string]any{"dir": t.Dir, "sets": sets, "uses": uses, "pending": core.ModJournalPending(root)}
	if a.safe {
		out["safe"] = modsSafeNote
	}
	if d := core.LoadDeployed(root); d != nil {
		out["deployed"] = map[string]any{"mods": d.Mods, "owner": d.Owner}
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

// apiModToggle turns a mod on or off in the set owner plays with; a mod
// just added joins the end of it. The game's folder changes on
// ps_modsApply, or when a profile plays.
func (a *App) apiModToggle(key, owner, id string, on bool) error {
	root := a.modsRoot(key)
	o, set := a.modSet(root, owner)
	i := slices.IndexFunc(set, func(e core.ModEntry) bool { return e.ID == id })
	if i < 0 {
		if _, err := core.LoadMod(root, id); err != nil || !on {
			return errors.New("That mod isn't kept any more.")
		}
		set = append(set, core.ModEntry{ID: id, On: true})
	} else {
		set[i].On = on
	}
	return core.SaveModSet(root, o, set)
}

// apiModSetOwn gives a profile a set of its own, a copy of the one it plays
// with now; with on false, every mod in it is off.
func (a *App) apiModSetOwn(key, owner string, on bool) error {
	a.mu.Lock()
	p := a.cfg.Player(owner)
	a.mu.Unlock()
	if p == nil {
		return errors.New("That profile no longer exists.")
	}
	root := a.modsRoot(key)
	_, set := a.modSet(root, owner)
	for i := range set {
		set[i].On = set[i].On && on
	}
	return core.SaveModSet(root, owner, set)
}

// apiModOrder moves a mod one place up (by -1) or down (by 1). A later mod
// wins where two have the same file.
func (a *App) apiModOrder(key, owner, id string, by int) error {
	root := a.modsRoot(key)
	o, set := a.modSet(root, owner)
	i := slices.IndexFunc(set, func(e core.ModEntry) bool { return e.ID == id })
	j := i + by
	if i < 0 || (by != -1 && by != 1) || j < 0 || j >= len(set) {
		return nil
	}
	set[i], set[j] = set[j], set[i]
	return core.SaveModSet(root, o, set)
}

// apiModRemove deletes Lootnest's copy of a mod. A mod with files in the
// game's folder comes out of it first (the other mods of the set that's in
// go back in), and the answer is a "modsDone" event; then it returns true.
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
		res, err := a.applyModSet(t, d.Owner, id, step)
		if err == nil {
			err = core.RemoveMod(root, id)
		}
		return res, err
	})
}

// apiModsApply puts the mods that are on in the set owner plays with into
// the game's folder, in order, after taking out what's there now. The saves
// are copied first. Play switches sets again from then on.
func (a *App) apiModsApply(key, owner string) error {
	return a.modsRun(key, "apply", func(t updTarget, step func(int, int)) (core.ModsResult, error) {
		res, err := a.applyModSet(t, owner, "", step)
		if err == nil {
			_ = os.Remove(modsOffMark(a.modsRoot(key)))
		}
		return res, err
	})
}

// modsOffMark is there while an update took a game's mods out: Play leaves
// it as it is until the user applies mods again.
func modsOffMark(root string) string { return filepath.Join(root, "off-after-update") }

// applyModSet puts the mods that are on in the set owner plays with, all
// but skip, into the game's folder, copying the saves first.
func (a *App) applyModSet(t updTarget, owner, skip string, step func(int, int)) (core.ModsResult, error) {
	root := a.modsRoot(t.Key)
	o, set := a.modSet(root, owner)
	var ids []string
	for _, e := range set {
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
	return core.ApplyMods(root, t.Dir, t.Exe, o, mods, time.Now(), step)
}

// apiModsRestore puts the game back as it was: every mod file out, every
// replaced game file back.
func (a *App) apiModsRestore(key string) error {
	return a.modsRun(key, "restore", func(t updTarget, step func(int, int)) (core.ModsResult, error) {
		return core.ApplyMods(a.modsRoot(key), t.Dir, t.Exe, "", nil, time.Now(), step)
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
		defer modding(t.Dir)()
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

// modsForPlay puts the mods a profile plays with into the game's folder
// once its saves are in, when they aren't the ones there: the same journaled
// run as Apply, with the session saying how far it is. The saves need no
// copy here: Activate has just kept them in the profile. A set with no mods
// on, or one that can't go in (the folder refuses mods, a kept mod is gone),
// puts the game back as it was. In safe mode, and after an update took the
// mods out, the game starts as it is.
func (a *App) modsForPlay(g *core.Game, playerID, who string) error {
	key := "game:" + g.ID
	root := a.modsRoot(key)
	if a.safe || !dirExists(root) || fileExists(modsOffMark(root)) {
		return nil
	}
	if core.ModJournalPending(root) {
		return errors.New("the last change to its mods didn't finish")
	}
	o, set := a.modSet(root, playerID)
	var on, in []string
	for _, e := range set {
		if e.On {
			on = append(on, e.ID)
		}
	}
	if d := core.LoadDeployed(root); d != nil {
		in = d.Mods
	}
	if slices.Equal(on, in) {
		return nil
	}
	t, err := a.modTarget(key)
	if err != nil {
		return err
	}
	mods, err := core.LoadMods(root, on)
	if err == nil {
		err = core.ModsRefused(t.Dir)
	}
	if err != nil {
		a.logf("%s's mods for %s can't go in, so it starts without them: %v", who, g.Name, err)
		if len(in) == 0 {
			return nil
		}
		on, mods = nil, nil
	}
	if !updMu.TryLock() {
		return errors.New("Lootnest is changing a game's folder. Start the game once that's done")
	}
	defer updMu.Unlock()
	defer modding(t.Dir)()
	defer a.emit("modsFinished", nil)
	var last time.Time
	progress := func(done, total int) {
		if done != total && time.Since(last) < 250*time.Millisecond {
			return
		}
		last = time.Now()
		a.mu.Lock()
		if s := a.sessions[g.ID]; s != nil {
			s.Phase, s.Mods = "mods", &SessionMods{Done: done, Total: total, Off: len(on) == 0}
		}
		a.mu.Unlock()
		a.emitSessions()
	}
	progress(0, 0)
	res, err := core.ApplyMods(root, t.Dir, t.Exe, o, mods, time.Now(), progress)
	if err != nil {
		a.logf("ERROR mods for %s before it started for %s: %v", g.Name, who, err)
		return err
	}
	a.mu.Lock()
	if s := a.sessions[g.ID]; s != nil {
		s.Phase, s.Mods = "loading", nil
	}
	a.mu.Unlock()
	a.emitSessions()
	a.logf("Mods for %s switched for %s (%d on); %d changed files kept in %s", g.Name, who, len(on), res.Changed, res.ChangedDir)
	return nil
}

// modsOffFor puts a game with mods in its folder back as it was before an
// update changes it (journaled, as Put the game back), and says whether it
// did. The caller holds updMu.
func (a *App) modsOffFor(t updTarget) (bool, error) {
	root := a.modsRoot(strings.TrimPrefix(t.Key, "found:"))
	if core.LoadDeployed(root) == nil && !core.ModJournalPending(root) {
		return false, nil
	}
	if a.safe {
		return false, errors.New("Mods are in this game's folder, and safe mode leaves them as they are. Restart Lootnest normally to update it.")
	}
	defer modding(t.Dir)()
	if _, err := core.ApplyMods(root, t.Dir, t.Exe, "", nil, time.Now(), nil); err != nil {
		return false, fmt.Errorf("Nothing was updated: Lootnest couldn't take the mods out first. %s", sentence(err))
	}
	if err := os.WriteFile(modsOffMark(root), nil, 0o644); err != nil {
		a.logf("ERROR marking %s's mods as off after an update: %v", t.Name, err)
	}
	a.logf("Took the mods out of %s before updating it", t.Name)
	a.emit("modsFinished", nil)
	return true, nil
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
