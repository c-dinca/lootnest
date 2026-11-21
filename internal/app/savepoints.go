package app

import (
	"cmp"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
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
	proc              string // its process, for a picture while it runs
	game              *core.Game
	locs              []core.SaveLoc
	running           bool
	appID             int
	dir               string // a Library game's folder
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
		t.name, t.game, t.locs, t.appID, t.proc = c.Name, &c, locs, core.GameAppID(&c), procName(&c)
		t.player = a.store.LoadState()[c.ID]
		t.running = active || a.isRunning(&c)
		if procs, err := plat.ProcessNames(); err == nil {
			t.proc = cmp.Or(liveName(a.gameNames(&c), procs), t.proc)
		}
		return t, nil
	}
	f, ok := a.foundByKey(key)
	if !ok {
		return t, errFoundGone
	}
	locs := a.savesOf(f).Locs
	if f.Dir != "" {
		// And the folders engines keep saves in inside the game's folder,
		// as the safety net does.
		for _, l := range a.netLocs(f) {
			dup := false
			for _, k := range locs {
				dup = dup || strings.EqualFold(k.Resolved, l.Resolved) || isInside(l.Resolved, k.Resolved) || isInside(k.Resolved, l.Resolved)
			}
			if !dup {
				locs = append(locs, l)
			}
		}
	}
	for _, l := range locs {
		t.locs = append(t.locs, core.SaveLoc{ID: core.LocID(l.Path), Live: l.Resolved, Path: l.Path})
	}
	if len(t.locs) == 0 {
		return t, errors.New("Lootnest doesn't know where this game keeps its saves yet. Choose them under Cloud save on its page.")
	}
	t.name, t.appID, t.proc, t.dir = f.Name, f.SteamID, strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe")), f.Dir
	a.mu.Lock()
	t.player = a.seat
	a.mu.Unlock()
	names := runNames(a.launchOpts(key), t.proc)
	if len(names) > 0 {
		if procs, err := plat.ProcessNames(); err == nil {
			if live := liveName(names, procs); live != "" {
				t.proc, t.running = live, true
			}
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
		root := a.store.SavePointsDir(key)
		pts := []pointOut{}
		for _, p := range core.SavePoints(root) {
			pts = append(pts, pointOut{p, dataURL(filepath.Join(root, p.ID, core.PointThumb))})
		}
		out["points"], out["player"], out["running"], out["profiles"] = pts, t.player, t.running, t.game != nil
		out["quickSave"], out["suggest"] = a.quickSaveKey(key), core.QuickSaveSuggestion(t.appID, t.name)
		if ac := core.AntiCheat(a.gameDir(t)); ac != "" {
			out["antiCheat"] = ac
		}
	}()
}

// pointOut is a point as the window shows it, with its picture's thumbnail.
type pointOut struct {
	core.SavePoint
	Thumb string `json:"thumb,omitempty"`
}

// playingFor is how long the game has run in this sitting, for player,
// which the playtime kept adds only when it closes.
func (a *App) playingFor(key, player string) time.Duration {
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		if s := a.sessions[id]; s != nil && s.Phase == "playing" && !s.Since.IsZero() && (player == "" || s.PlayerID == player) {
			return time.Since(s.Since)
		}
		return 0
	}
	foundRunsMu.Lock()
	defer foundRunsMu.Unlock()
	if r := foundRuns[key]; r != nil && !r.since.IsZero() {
		return time.Since(r.since)
	}
	return 0
}

// makePoint makes a point now, with the playtime and achievements by then,
// and a picture of the game when it's running.
func (a *App) makePoint(t pointTarget, p core.SavePoint) (core.SavePoint, error) {
	var shot image.Image
	if t.running && t.proc != "" {
		img, err := captureGame(t.proc)
		if err != nil {
			a.logf("No picture for the save point of %s: %v", t.name, err)
		} else {
			shot = img
		}
	}
	pointMu.Lock()
	defer pointMu.Unlock()
	p.Player = t.player
	if pt := a.store.LoadPlaytime()[t.key]; pt != nil {
		p.Playtime = pt[t.player]
		if t.player == "" {
			for _, s := range pt {
				p.Playtime += s
			}
		}
	}
	p.Playtime += int64(a.playingFor(t.key, t.player) / time.Second)
	if t.appID > 0 {
		if u, _ := a.env.PCUnlocks(t.appID); u != nil {
			p.Unlocked = len(u)
		}
	}
	tries := 1
	if t.running {
		tries = 4 // the game may be writing: read back, and again if it was
	}
	root := a.store.SavePointsDir(t.key)
	p, err := core.MakeSavePoint(root, t.locs, p, time.Now(), tries, 1500*time.Millisecond)
	if err == nil {
		savesSeen(t.key, t.locs)
	}
	if err == nil && shot != nil {
		if err := core.SavePointPictures(root, p.ID, shot); err != nil {
			a.logf("Couldn't keep the picture of the save point %q: %v", p.Name, err)
		}
	}
	return p, err
}

func (a *App) apiMakeSavePoint(key, name string) error {
	t, err := a.pointTarget(key)
	if err != nil {
		return err
	}
	go func() {
		out := map[string]any{"key": key}
		step := func(s string) { a.emit("pointStep", map[string]any{"key": key, "step": s}) }
		saved, err := a.saveInGame(t, step)
		if err != nil {
			out["error"] = sentence(err)
			a.emit("savePointMade", out)
			a.apiSavePoints(key)
			return
		}
		step("copying")
		p, err := a.makePoint(t, core.SavePoint{Name: name, InGame: saved})
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

// apiRestoreSavePoint takes a game back to a point: the game is in that
// moment again when it next loads. A running game is asked to close first,
// as closing its window would, and Lootnest waits until it and its session
// are done. The saves as they are then are kept as a point, the point's
// saves go in as the newest (core.RestoreSavePoint), and with play the page
// starts the game again on them.
func (a *App) apiRestoreSavePoint(key, id string, play bool) error {
	t, err := a.pointTarget(key)
	if err != nil {
		return err
	}
	root := a.store.SavePointsDir(key)
	p, err := core.LoadSavePoint(root, id)
	if err != nil {
		return err
	}
	if err := a.pointOwner(t, p); err != nil {
		return err
	}
	if t.running && t.proc == "" {
		return errors.New(t.name + " is running. Close it first, so it doesn't write over the saves you go back to.")
	}
	go func() {
		out := map[string]any{"key": key, "restored": id}
		defer func() { a.emit("savePointMade", out); a.apiSavePoints(key) }()
		step := func(s string) { a.emit("pointStep", map[string]any{"key": key, "step": s}) }
		if t.running {
			step("closing")
			if err := a.closeForPoint(t); err != nil {
				out["error"] = err.Error()
				return
			}
			// What the game wrote as it closed, and whose files are loaded
			// now that its session is done.
			if t, err = a.pointTarget(key); err != nil {
				out["error"] = "Nothing changed: " + err.Error()
				return
			}
			if t.running {
				out["error"] = "Nothing changed: " + t.name + " started again before Lootnest could go back."
				return
			}
			if err := a.pointOwner(t, p); err != nil {
				out["error"] = "Nothing changed: " + err.Error()
				return
			}
		}
		step("copying")
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
		// The saves as they are now become a point first, so going back
		// is never a loss.
		if _, err := a.makePoint(t, core.SavePoint{Name: "Before going back to " + p.Name, Auto: true}); err != nil && !strings.Contains(err.Error(), "no saves") {
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
		if play {
			out["play"], out["player"] = true, t.player
		}
		a.logf("%s went back to the save point %q", t.name, p.Name)
	}()
	return nil
}

// pointOwner refuses a point of a game with profiles whose saves are
// someone else's than the ones loaded now.
func (a *App) pointOwner(t pointTarget, p core.SavePoint) error {
	if t.game == nil || p.Player == "" || p.Player == t.player {
		return nil
	}
	a.mu.Lock()
	who := a.cfg.PlayerName(p.Player)
	a.mu.Unlock()
	return fmt.Errorf("This point has %s's saves, and someone else's are loaded now. Load %s's files first, from Manage.", who, who)
}

// How long a game has to close when going back to a point, and its session
// to finish after (its files go back to the profile).
var (
	pointCloseWait   = 30 * time.Second
	pointSessionWait = 60 * time.Second
	pointCloudWait   = 3 * time.Second // Steam Cloud sends what the game wrote as it closed
)

// closeForPoint asks a running game to quit, as closing its window would,
// and waits until it's gone. It never ends the game by force: a game that
// asks something first (Quit? Save?) is answered by the player.
func (a *App) closeForPoint(t pointTarget) error {
	if plat.CloseProcessWindows(t.proc) == 0 {
		return errors.New("Nothing changed: Lootnest couldn't close " + t.name + ". Quit it from its own menu, then go back again.")
	}
	a.logf("Asked %s to close, to go back to a save point", t.name)
	gone := func() bool {
		procs, err := plat.ProcessNames()
		return err == nil && !procs[t.proc]
	}
	if !waitFor(pointCloseWait, gone) {
		return errors.New("Nothing changed: " + t.name + " didn't close. It may be asking something on its screen; answer it, or quit from its menu, then go back again.")
	}
	if t.game != nil {
		id := t.game.ID
		done := func() bool {
			a.mu.Lock()
			_, on := a.sessions[id]
			a.mu.Unlock()
			return !on
		}
		if !waitFor(pointSessionWait, done) {
			return errors.New("Nothing changed: " + t.name + " closed, but its saves are still going back to the profile. Try again in a moment.")
		}
	}
	if t.appID > 0 {
		time.Sleep(pointCloudWait)
	}
	return nil
}

// waitFor checks ok every half second until it's true or limit passes.
func waitFor(limit time.Duration, ok func() bool) bool {
	end := time.Now().Add(limit)
	for {
		if ok() {
			return true
		}
		if time.Now().After(end) {
			return false
		}
		time.Sleep(500 * time.Millisecond)
	}
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
	name = trimRunes(name, 24)
	if name == "" {
		head := person.Name + " from "
		if room := 24 - len([]rune(head)); room >= 4 {
			name = head + trimWords(p.Name, room)
		} else {
			name = trimRunes(person.Name+" "+p.Name, 24)
		}
		name = freeName(name, cfg.Players)
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
		_ = os.RemoveAll(a.store.ProfileDir(run.ID, t.game.ID))
		return "", err
	}
	a.logf("New run %s of %s in %s, from the save point %q", run.Name, person.Name, t.name, p.Name)
	return run.ID, nil
}

// gameBuild is a game's build (core/gamebuild.go): Steam's with an app id,
// else its exe's.
func (a *App) gameBuild(appID int, exe string) string {
	if appID == 0 {
		exe, _ = a.env.ResolveExe(exe)
	}
	return core.GameBuild(a.env.SteamPath, appID, exe)
}

func (a *App) profileBuild(g *core.Game) string {
	if g.Launch.Type == "steam" {
		return a.gameBuild(g.Launch.AppID, "")
	}
	return a.gameBuild(0, g.Launch.Exe)
}

// updatedSince notes the build of a game about to start and says whether
// it updated since it last started from Lootnest, with a save point before
// updates on.
func (a *App) updatedSince(key, build string) bool {
	changed, err := a.store.NoteBuild(key, build)
	if err != nil {
		a.logf("Couldn't note which version of %s is installed: %v", key, err)
	}
	a.mu.Lock()
	off := a.cfg.Settings.UpdatePointOff
	a.mu.Unlock()
	return changed && !off
}

// pointBeforeUpdate keeps the saves as they are before an updated game
// first starts, as an Auto point. It's never in the game's way: a game with
// no saves to keep is skipped, and a failure is only logged.
func (a *App) pointBeforeUpdate(key string) {
	t, err := a.pointTarget(key)
	if err != nil {
		return
	}
	p, err := a.makePoint(t, core.SavePoint{Name: "Before the update", Auto: true, Update: true})
	if err != nil {
		if !strings.Contains(err.Error(), "no saves") {
			a.logf("Couldn't make a save point before %s's update, so it starts without one: %v", t.name, err)
		}
		return
	}
	a.logf("%s updated since it last started: save point %q (%d KB)", t.name, p.Name, (p.Size+1023)/1024)
	a.apiSavePoints(key)
}

// --------------------------------------------------------------- hotkey --

const hotkeyPoint = 0x4C4E // "LN"

var hotkeysOn = map[int]bool{} // only the watch loop changes it

// hotkeyTick holds Ctrl+Shift+F5 and Ctrl+Shift+F12 only while a game runs,
// so they take nothing from other programs the rest of the time.
func (a *App) hotkeyTick(running []achRun) {
	a.mu.Lock()
	s := a.cfg.Settings
	a.mu.Unlock()
	a.holdHotkey(hotkeyPoint, plat.VK_F5, len(running) > 0 && !s.PointHotkeyOff, "Ctrl+Shift+F5 is taken by another program; save points still work from the game's page")
	a.holdHotkey(hotkeyShot, plat.VK_F12, len(running) > 0 && !s.ScreenshotsOff, "Ctrl+Shift+F12 is taken by another program, so it doesn't take screenshots")
	a.fpsTick(running)
}

func (a *App) holdHotkey(id int, vk uint32, want bool, taken string) {
	if want == hotkeysOn[id] {
		return
	}
	hotkeysOn[id] = want
	a.ui(func() {
		if want {
			if !plat.RegisterHotKey(a.hwnd, id, plat.ModControl|plat.ModShift|plat.ModNoRepeat, vk) {
				a.logf("%s", taken)
			}
		} else {
			plat.UnregisterHotKey(a.hwnd, id)
		}
	})
}

// runningNow is the first game running now, with its profile's colour.
func (a *App) runningNow() (achRun, string, bool) {
	procs, err := plat.ProcessNames()
	if err != nil {
		return achRun{}, "", false
	}
	running := a.achRunning(procs)
	if len(running) == 0 {
		return achRun{}, "", false
	}
	r, color := running[0], ""
	a.mu.Lock()
	if p := a.cfg.Player(r.player); p != nil {
		color = p.Color
	}
	a.mu.Unlock()
	return r, color, true
}

// quickPoint is the hotkey: a point for the game running now.
func (a *App) quickPoint() {
	r, color, ok := a.runningNow()
	if !ok {
		return
	}
	go func() {
		t, err := a.pointTarget(r.key)
		if err == nil {
			var p core.SavePoint
			step := func(s string) { a.emit("pointStep", map[string]any{"key": r.key, "step": s}) }
			var saved bool
			if saved, err = a.saveInGame(t, step); err != nil {
				plat.ShowToast(plat.Toast{Kicker: "No save point", Title: r.name, Line: sentence(err), Accent: color})
				a.emit("savePointMade", map[string]any{"key": r.key, "error": sentence(err), "hotkey": true})
				return
			}
			kicker := "Save point made"
			if saved {
				kicker = "Saved and kept"
			}
			if p, err = a.makePoint(t, core.SavePoint{Name: "Quick save point, " + time.Now().Format("15:04"), InGame: saved}); err == nil {
				plat.ShowToast(plat.Toast{Kicker: kicker, Title: p.Name, Line: r.name + ". Go back to it from the game's page.", Accent: color})
				a.logf("Save point %q for %s (hotkey)", p.Name, r.name)
				native := core.AntiCheat(r.dir) == "" && a.nativeSound(pointWAV)
				a.emit("savePointMade", map[string]any{"key": r.key, "point": p, "hotkey": true, "native": native})
				a.apiSavePoints(r.key)
				return
			}
		}
		plat.ShowToast(plat.Toast{Kicker: "No save point", Title: r.name, Line: sentence(err), Accent: color})
	}()
}
