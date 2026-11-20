package app

import (
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Every save the game makes: while a game runs, its save locations are
// looked at on each watch tick (about every 6 s). When they changed since
// the last point and have stayed the same for a tick, the game has saved,
// and that save is kept as a point ("Saved by the game"). The newest
// gamePointsKeep of those are kept per game; points the user made, and
// Lootnest's own before an update or going back, are never pruned.
// Settings.GamePointsOff turns it off.
// ---------------------------------------------------------------------------

const (
	gamePointsKeep  = 30
	gamePointsStill = 5 * time.Second   // unchanged this long: the game is done writing
	gamePointsMax   = 512 * 1024 * 1024 // saves larger than this aren't copied on every save
)

// gameSaves is what the tick knows of one running game.
type gameSaves struct {
	t      pointTarget
	off    string          // why it isn't followed ("" when it is)
	seen   core.SaveStamps // the saves at the last point, or when the game started
	cand   core.SaveStamps // a change waiting to stand still
	candAt time.Time
	busy   bool
}

var (
	gameSavesMu sync.Mutex
	gameSavesOf = map[string]*gameSaves{}
)

// savesSeen notes the saves as they are when a point was made, whoever
// made it, so the tick doesn't keep the same save twice.
func savesSeen(key string, locs []core.SaveLoc) {
	st, err := core.StampSaves(locs)
	if err != nil {
		return
	}
	gameSavesMu.Lock()
	if g := gameSavesOf[key]; g != nil {
		g.seen, g.cand = st, nil
	}
	gameSavesMu.Unlock()
}

// runningForSaves are the running games: those achTick follows, and Steam
// games in the Library without profiles, by their process.
func (a *App) runningForSaves(procs map[string]bool, running []achRun) []achRun {
	out := append([]achRun{}, running...)
	have := map[string]bool{}
	for _, r := range running {
		have[r.key] = true
	}
	for _, f := range a.libraryGames() {
		proc := strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))
		if f.AppID == 0 || proc == "" || !procs[proc] || have[f.Key] {
			continue
		}
		out = append(out, achRun{key: f.Key, name: f.Name, dir: f.Dir, appID: f.AppID, proc: proc})
	}
	return out
}

// gamePointsTick runs from achTick on the watch loop.
func (a *App) gamePointsTick(procs map[string]bool, running []achRun) {
	a.mu.Lock()
	off := a.cfg.Settings.GamePointsOff || a.safe
	a.mu.Unlock()
	gameSavesMu.Lock()
	defer gameSavesMu.Unlock()
	if off {
		clear(gameSavesOf)
		return
	}
	here := map[string]bool{}
	for _, r := range a.runningForSaves(procs, running) {
		here[r.key] = true
		g := gameSavesOf[r.key]
		if g == nil {
			g = &gameSaves{}
			gameSavesOf[r.key] = g
			t, err := a.pointTarget(r.key)
			switch {
			case err != nil:
				g.off = err.Error()
			default:
				g.t = t
				g.seen, err = core.StampSaves(t.locs)
				if err != nil {
					g.off = err.Error()
				} else if savesSize(g.seen) > gamePointsMax {
					g.off = "its saves are over 512 MB"
				}
			}
			if g.off != "" {
				a.logf("Lootnest doesn't keep each save of %s: %s", r.name, strings.TrimSuffix(g.off, "."))
			}
			continue
		}
		if g.off != "" || g.busy {
			continue
		}
		now, err := core.StampSaves(g.t.locs)
		if err != nil || core.SameSaves(now, g.seen) {
			g.cand = nil
			continue
		}
		if g.cand == nil || !core.SameSaves(now, g.cand) {
			g.cand, g.candAt = now, time.Now()
			continue
		}
		if time.Since(g.candAt) < gamePointsStill {
			continue
		}
		g.busy, g.cand = true, nil
		go a.keepGameSave(r.key, g.t)
	}
	for k := range gameSavesOf {
		if !here[k] {
			delete(gameSavesOf, k)
		}
	}
}

func savesSize(st core.SaveStamps) int64 {
	var n int64
	for _, files := range st {
		for _, f := range files {
			n += f.Size
		}
	}
	return n
}

// keepGameSave makes a point of the save the game just made.
func (a *App) keepGameSave(key string, t pointTarget) {
	defer func() {
		gameSavesMu.Lock()
		if g := gameSavesOf[key]; g != nil {
			g.busy = false
		}
		gameSavesMu.Unlock()
	}()
	t.running = true // for its picture; the copy reads back what it took
	p, err := a.makePoint(t, core.SavePoint{Name: "Saved by the game, " + time.Now().Format("15:04"), Game: true})
	if err != nil {
		if !strings.Contains(err.Error(), "no saves") {
			a.logf("Couldn't keep %s's new save as a point: %v", t.name, err)
		}
		return
	}
	if n, err := core.PruneGamePoints(a.store.SavePointsDir(key), gamePointsKeep); err != nil {
		a.logf("Couldn't let go of %s's oldest game saves: %v", t.name, err)
	} else if n > 0 {
		a.logf("Let go of %d of %s's oldest game saves (the newest %d are kept)", n, t.name, gamePointsKeep)
	}
	a.emit("savePointMade", map[string]any{"key": key, "point": p, "game": true})
	a.apiSavePoints(key)
}
