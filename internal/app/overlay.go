package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// ---------------------------------------------------------------------------
// The FPS counter inside story games (core/overlay.go, native/overlay): when
// the counter is on and the game in front has no anti-cheat and isn't
// played online, Lootnest loads its DLL into it once, as Steam's overlay
// goes into games. The DLL counts the game's frames itself and draws the
// number into them, over exclusive fullscreen too; Lootnest's own window
// then stays hidden for that game. Direct3D 12 games are only counted (the
// window shows the number), and a game the DLL can't serve falls back to
// the frames Windows reports (app/framereader.go).
// ---------------------------------------------------------------------------

type overlayGame struct {
	share     *plat.OverlayShare
	name      string
	drawn     int32
	drawnAt   time.Time
	beat      int64
	beatAt    time.Time
	saidAPI   bool
	saidError int32
}

var overlays struct {
	mu      sync.Mutex
	games   map[uint32]*overlayGame // by pid: the DLL is in
	asked   map[uint32]bool         // pids looked at (in or not)
	dllPath string
}

// overlayWanted: the in-game counter isn't turned off in Settings, and
// not in safe mode.
func (a *App) overlayWanted() bool {
	a.mu.Lock()
	off := a.cfg.Settings.FPSInGameOff
	a.mu.Unlock()
	return !off && !a.safe && len(overlayDLL) > 0
}

// overlayFor is the DLL's state in pid, or nil.
func overlayFor(pid uint32) *overlayGame {
	overlays.mu.Lock()
	defer overlays.mu.Unlock()
	return overlays.games[pid]
}

// overlayConsider looks once at the game in front (pid) and, if it may,
// puts the DLL in, on its own goroutine: looking through a game's folder
// for an anti-cheat takes a moment.
func (a *App) overlayConsider(pid uint32) {
	if !a.overlayWanted() {
		return
	}
	overlays.mu.Lock()
	if overlays.asked == nil {
		overlays.asked, overlays.games = map[uint32]bool{}, map[uint32]*overlayGame{}
	}
	if overlays.asked[pid] {
		overlays.mu.Unlock()
		return
	}
	overlays.asked[pid] = true
	overlays.mu.Unlock()
	go a.overlayIn(pid)
}

// overlayIn decides and, when it may, loads the DLL into pid.
func (a *App) overlayIn(pid uint32) {
	r, ok := a.runningOfPID(pid)
	if !ok {
		return // not a game Lootnest knows (a browser, a launcher)
	}
	dirs := []string{r.dir}
	if p := plat.ProcessPath(pid); p != "" {
		dirs = append(dirs, filepath.Dir(p))
	}
	facts := core.InGameFacts{Dirs: dirs, AppID: r.appID, Name: r.name}
	if r.appID > 0 {
		facts.SteamOnline = core.SteamOnline(a.env.SteamPath, map[int]bool{r.appID: true})[r.appID]
	}
	if procs, err := plat.ProcessNames(); err == nil {
		facts.Running = procs
	}
	if ok, why := core.InGameAllowed(facts); !ok {
		a.logf("FPS counter: not inside %s, since %s; the counter stays outside the game", r.name, why)
		return
	}
	if !plat.Is64BitProcess(pid) {
		a.logf("FPS counter: not inside %s, a 32-bit program; the counter stays outside the game", r.name)
		return
	}
	dll, err := overlayDLLPath()
	if err != nil {
		a.logf("FPS counter: couldn't write its in-game part: %v", err)
		return
	}
	share, err := plat.NewOverlayShare(pid, core.OverlayMagic, core.OverlayVersion, core.OverlaySize)
	if err != nil {
		a.logf("FPS counter: couldn't make the block for %s: %v", r.name, err)
		return
	}
	a.mu.Lock()
	share.SetInt32(core.OverCorner, core.FPSCornerIndex(a.cfg.Settings.FPSCorner))
	a.mu.Unlock()
	share.SetInt32(core.OverOn, 1)
	if err := plat.InjectDLL(pid, dll); err != nil {
		share.Close()
		if errors.Is(err, plat.ErrNot64Bit) {
			a.logf("FPS counter: not inside %s, a 32-bit program", r.name)
		} else {
			a.logf("FPS counter: couldn't go inside %s: %v; the counter stays outside the game", r.name, err)
		}
		return
	}
	a.logf("FPS counter: inside %s (no anti-cheat), drawn in the game", r.name)
	overlays.mu.Lock()
	overlays.games[pid] = &overlayGame{share: share, name: r.name}
	overlays.mu.Unlock()
}

// runningOfPID is the running game whose program is pid's.
func (a *App) runningOfPID(pid uint32) (achRun, bool) {
	name := plat.ProcessName(pid)
	if name == "" {
		return achRun{}, false
	}
	procs, err := plat.ProcessNames()
	if err != nil {
		return achRun{}, false
	}
	for _, r := range a.achRunning(procs) {
		if r.proc == name {
			return r, true
		}
	}
	return achRun{}, false
}

// overlayDLLPath writes the embedded DLL to the data folder once per
// version (a game keeps the file it loaded open) and lets older ones go.
func overlayDLLPath() (string, error) {
	overlays.mu.Lock()
	defer overlays.mu.Unlock()
	if overlays.dllPath != "" {
		return overlays.dllPath, nil
	}
	sum := sha256.Sum256(overlayDLL)
	dir := filepath.Join(DataDir(), "overlay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := core.OverlayDLLName(hex.EncodeToString(sum[:]))
	p := filepath.Join(dir, name)
	if b, err := os.ReadFile(p); err != nil || sha256.Sum256(b) != sum {
		tmp := p + ".part"
		if err := os.WriteFile(tmp, overlayDLL, 0o644); err != nil {
			return "", err
		}
		if err := os.Rename(tmp, p); err != nil {
			os.Remove(tmp)
			return "", err
		}
	}
	core.PruneOverlayDLLs(dir, name)
	overlays.dllPath = p
	return p, nil
}

// overlayTick runs with the counter's loop: it tells every game with the
// DLL whether to draw and where, lets go of games that ended, and says
// what the game in front shows. inGame: the number is drawn in the game
// (Lootnest's window stays hidden); fps: the DLL's count, when it counts
// but can't draw.
func (a *App) overlayTick(front uint32, on bool) (inGame bool, fps int, counted bool) {
	a.mu.Lock()
	corner := core.FPSCornerIndex(a.cfg.Settings.FPSCorner)
	a.mu.Unlock()
	overlays.mu.Lock()
	defer overlays.mu.Unlock()
	now := time.Now()
	for pid, g := range overlays.games {
		if !plat.ProcessAlive(pid) {
			g.share.Close()
			delete(overlays.games, pid)
			delete(overlays.asked, pid)
			continue
		}
		g.share.SetInt32(core.OverCorner, corner)
		g.share.SetInt32(core.OverOn, map[bool]int32{true: 1, false: 0}[on])
		if d := g.share.Int32(core.OverDrawn); d != g.drawn {
			g.drawn, g.drawnAt = d, now
		}
		if b := g.share.Int64(core.OverBeat); b != g.beat {
			g.beat, g.beatAt = b, now
		}
		if api := g.share.Int32(core.OverAPI); api != 0 && !g.saidAPI {
			g.saidAPI = true
			a.logf("FPS counter: %s draws with %s", g.name, core.OverlayAPIs[api].Name)
		}
		if e := g.share.Int32(core.OverError); e != 0 && e != g.saidError {
			g.saidError = e
			a.logf("FPS counter: inside %s, %s", g.name, core.OverlayErrors[e])
		}
	}
	g := overlays.games[front]
	if g == nil {
		return false, 0, false
	}
	if on && now.Sub(g.drawnAt) < 1500*time.Millisecond {
		return true, 0, true
	}
	if now.Sub(g.beatAt) < 3*time.Second {
		return false, int(g.share.Int32(core.OverFPS)), true
	}
	return false, 0, false
}

// overlaysOff tells every game with the DLL to stop drawing.
func overlaysOff() {
	overlays.mu.Lock()
	defer overlays.mu.Unlock()
	for _, g := range overlays.games {
		g.share.SetInt32(core.OverOn, 0)
	}
}
