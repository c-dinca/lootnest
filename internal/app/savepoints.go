//go:build windows

package app

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Save points (core/savepoints.go): the game page's timeline, and
// Ctrl+Shift+F5 while a game runs. Going back to a point keeps the saves as
// they are now as a point first; a point of a game with profiles can start
// a new run.
// ---------------------------------------------------------------------------

var pointMu sync.Mutex // one point made or restored at a time

// pointTarget is where a game keeps its saves, whose they are, and whether
// it's running.
type pointTarget struct {
	key, name, player string
	game              *core.Game
	locs              []core.SaveLoc
	running           bool
	appID             int
}

func (a *App) pointTarget(key string) (pointTarget, error) {
	t := pointTarget{key: key}
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		g := a.cfg.Game(id)
		var c core.Game
		if g != nil {
			c = *g
		}
		_, active := a.sessions[id]
		a.mu.Unlock()
		if g == nil {
			return t, errors.New("That game no longer exists.")
		}
		locs, err := a.eng.SaveLocs(&c)
		if err != nil {
			return t, err
		}
		if len(locs) == 0 {
			return t, errors.New("This game has no save locations yet. Edit the game to choose them.")
		}
		t.name, t.game, t.locs, t.appID = c.Name, &c, locs, core.GameAppID(&c)
		t.player = a.store.LoadState()[c.ID]
		t.running = active || isRunning(&c)
		return t, nil
	}
	f, ok := a.foundByKey(key)
	if !ok {
		return t, errFoundGone
	}
	for _, l := range a.savesOf(f).Locs {
		t.locs = append(t.locs, core.SaveLoc{ID: core.LocID(l.Path), Live: l.Resolved, Path: l.Path})
	}
	if len(t.locs) == 0 {
		return t, errors.New("Lootnest doesn't know where this game keeps its saves yet. Choose them under Cloud save on its page.")
	}
	t.name, t.appID = f.Name, f.SteamID
	a.mu.Lock()
	t.player = a.seat
	a.mu.Unlock()
	if f.ProcessName != "" {
		if procs, err := win.ProcessNames(); err == nil {
			t.running = procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))]
		}
	}
	return t, nil
}

func (a *App) apiSavePoints(key string) {
	go func() {
		out := map[string]any{"key": key}
		defer func() { a.emit("savePoints", out) }()
		t, err := a.pointTarget(key)
		if err != nil {
			out["error"], out["points"] = err.Error(), []core.SavePoint{}
			return
		}
		pts := core.SavePoints(a.store.SavePointsDir(key))
		if pts == nil {
			pts = []core.SavePoint{}
		}
		out["points"], out["player"], out["running"], out["profiles"] = pts, t.player, t.running, t.game != nil
	}()
}

// makePoint makes a point now, with the playtime and achievements by then.
func (a *App) makePoint(t pointTarget, name string, auto bool) (core.SavePoint, error) {
	pointMu.Lock()
	defer pointMu.Unlock()
	p := core.SavePoint{Name: name, Player: t.player, Auto: auto}
	if pt := a.store.LoadPlaytime()[t.key]; pt != nil {
		p.Playtime = pt[t.player]
		if t.player == "" {
			for _, s := range pt {
				p.Playtime += s
			}
		}
	}
	if t.appID > 0 {
		if u, _ := a.env.PCUnlocks(t.appID); u != nil {
			p.Unlocked = len(u)
		}
	}
	tries := 1
	if t.running {
		tries = 4 // the game may be writing: read back, and again if it was
	}
	return core.MakeSavePoint(a.store.SavePointsDir(t.key), t.locs, p, time.Now(), tries, 1500*time.Millisecond)
}

func (a *App) apiMakeSavePoint(key, name string) error {
	t, err := a.pointTarget(key)
	if err != nil {
		return err
	}
	go func() {
		out := map[string]any{"key": key}
		p, err := a.makePoint(t, name, false)
		if err != nil {
			out["error"] = sentence(err)
		} else {
			out["point"] = p
			a.logf("Save point %q for %s (%d KB)", p.Name, t.name, (p.Size+1023)/1024)
		}
		a.emit("savePointMade", out)
		a.apiSavePoints(key)
	}()
	return nil
}

func (a *App) apiRestoreSavePoint(key, id string) error {
	t, err := a.pointTarget(key)
	if err != nil {
		return err
	}
	if t.running {
		return errors.New(t.name + " is running. Close it first, so it doesn't write over the saves you go back to.")
	}
	root := a.store.SavePointsDir(key)
	p, err := core.LoadSavePoint(root, id)
	if err != nil {
		return err
	}
	if t.game != nil && p.Player != "" && p.Player != t.player {
		a.mu.Lock()
		who := a.cfg.PlayerName(p.Player)
		a.mu.Unlock()
		return fmt.Errorf("This point has %s's saves, and someone else's are loaded now. Load %s's files first, from Manage.", who, who)
	}
	// Locations gone from the PC since, found again by their path.
	have := map[string]bool{}
	for _, l := range t.locs {
		have[l.ID] = true
	}
	for lid, path := range p.Paths {
		if !have[lid] {
			if r, err := a.env.Resolve(path); err == nil {
				t.locs = append(t.locs, core.SaveLoc{ID: lid, Live: r, Path: path})
			}
		}
	}
	go func() {
		out := map[string]any{"key": key, "restored": id}
		defer func() { a.emit("savePointMade", out); a.apiSavePoints(key) }()
		// The saves as they are now become a point first, so going back
		// is never a loss.
		if _, err := a.makePoint(t, "Before going back to "+p.Name, true); err != nil && !strings.Contains(err.Error(), "no saves") {
			out["error"] = "Nothing changed: Lootnest couldn't keep the saves as they are now (" + err.Error() + ")."
			return
		}
		pointMu.Lock()
		err := core.RestoreSavePoint(root, id, t.locs)
		pointMu.Unlock()
		if err != nil {
			out["error"] = "Going back didn't finish: " + err.Error() + ". The saves from before are the newest point."
			return
		}
		out["point"] = p
		a.logf("%s went back to the save point %q", t.name, p.Name)
	}()
	return nil
}

func (a *App) apiRenameSavePoint(key, id, name string) error {
	return core.RenameSavePoint(a.store.SavePointsDir(key), id, name)
}

func (a *App) apiDeleteSavePoint(key, id string) error {
	if err := core.DeleteSavePoint(a.store.SavePointsDir(key), id); err != nil {
		return err
	}
	a.apiSavePoints(key)
	return nil
}

// apiBranchSavePoint starts a new run (a profile of the point's person)
// from a point, for a game with profiles.
func (a *App) apiBranchSavePoint(key, id, name string) (string, error) {
	t, err := a.pointTarget(key)
	if err != nil {
		return "", err
	}
	if t.game == nil {
		return "", errors.New("Turn on profiles for this game first: a run is a profile of its own.")
	}
	root := a.store.SavePointsDir(key)
	p, err := core.LoadSavePoint(root, id)
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	person := cfg.Player(p.Player)
	if person == nil {
		return "", errors.New("This point's profile no longer exists.")
	}
	if person.Of != "" {
		if up := cfg.Player(person.Of); up != nil {
			person = up
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = person.Name + " from " + p.Name
	}
	if r := []rune(name); len(r) > 32 {
		name = string(r[:32])
	}
	run := core.Player{ID: core.NewID("p"), Name: name, Color: person.Color, Of: person.ID}
	players := append(append([]core.Player{}, cfg.Players...), run)
	if err := core.ValidatePlayers(players); err != nil {
		return "", err
	}
	if err := a.eng.ProfileFromSavePoint(root, id, t.game, run.ID); err != nil {
		return "", err
	}
	if err := a.apiSavePlayers(players); err != nil {
		return "", err
	}
	a.logf("New run %s of %s in %s, from the save point %q", run.Name, person.Name, t.name, p.Name)
	return run.ID, nil
}

// --------------------------------------------------------------- hotkey --

const hotkeyPoint = 0x4C4E // "LN"

var hotkeyOn bool // only the watch loop changes it

// hotkeyTick holds Ctrl+Shift+F5 only while a game runs, so it takes
// nothing from other programs the rest of the time.
func (a *App) hotkeyTick(running []achRun) {
	a.mu.Lock()
	off := a.cfg.Settings.PointHotkeyOff
	a.mu.Unlock()
	want := len(running) > 0 && !off
	if want == hotkeyOn {
		return
	}
	hotkeyOn = want
	a.ui(func() {
		if want {
			if !win.RegisterHotKey(a.hwnd, hotkeyPoint, win.ModControl|win.ModShift|win.ModNoRepeat, win.VK_F5) {
				a.logf("Ctrl+Shift+F5 is taken by another program; save points still work from the game's page")
			}
		} else {
			win.UnregisterHotKey(a.hwnd, hotkeyPoint)
		}
	})
}

// quickPoint is the hotkey: a point for the game running now.
func (a *App) quickPoint() {
	procs, err := win.ProcessNames()
	if err != nil {
		return
	}
	running := a.achRunning(procs)
	if len(running) == 0 {
		return
	}
	r := running[0]
	go func() {
		color := ""
		a.mu.Lock()
		if p := a.cfg.Player(r.player); p != nil {
			color = p.Color
		}
		a.mu.Unlock()
		t, err := a.pointTarget(r.key)
		if err == nil {
			var p core.SavePoint
			if p, err = a.makePoint(t, "Quick save point, "+time.Now().Format("15:04"), false); err == nil {
				win.ShowToast(win.Toast{Kicker: "Save point made", Title: p.Name, Line: r.name + ". Go back to it from the game's page.", Accent: color})
				a.logf("Save point %q for %s (hotkey)", p.Name, r.name)
				a.apiSavePoints(r.key)
				return
			}
		}
		win.ShowToast(win.Toast{Kicker: "No save point", Title: r.name, Line: sentence(err), Accent: color})
	}()
}
