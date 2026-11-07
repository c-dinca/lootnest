//go:build windows

package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Achievements as they unlock (core/achlive.go). The watch loop calls
// achTick every few seconds: for each game running with a Steam-compatible
// tool's achievement file, what's new since the last look is shown in a
// toast over the game, in the colour of the profile playing it.
// ---------------------------------------------------------------------------

type achLive struct {
	appID  int
	known  map[string]time.Time
	list   []core.Achievement
	name   string
	player string
	off    bool // no app id, or the game ships with an anti-cheat
}

// achRun is a game running now.
type achRun struct {
	key, name, dir, player string
	appID                  int
	game                   *core.Game
}

// Only the watch loop uses achLives.
var achLives = map[string]*achLive{}

func (a *App) apiSetSeat(id string) {
	a.mu.Lock()
	a.seat = id
	a.mu.Unlock()
}

// achRunning lists the games running now: the ones with profiles being
// played, and Library games outside Steam whose process is up.
func (a *App) achRunning(procs map[string]bool) []achRun {
	var out []achRun
	a.mu.Lock()
	seat := a.seat
	for id, s := range a.sessions {
		if s.Phase != "playing" {
			continue
		}
		if g := a.cfg.Game(id); g != nil {
			c := *g
			out = append(out, achRun{key: "game:" + id, name: g.Name, player: s.PlayerID, game: &c})
		}
	}
	a.mu.Unlock()
	for i := range out {
		if exe, err := a.env.ResolveExe(out[i].game.Launch.Exe); err == nil {
			out[i].dir = filepath.Dir(exe)
		}
		out[i].appID = core.GameAppID(out[i].game)
	}
	for _, f := range a.libraryGames() {
		if f.AppID > 0 || f.ProcessName == "" || !procs[strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))] {
			continue
		}
		out = append(out, achRun{key: f.Key, name: f.Name, dir: f.Dir, player: seat, appID: f.SteamID})
	}
	return out
}

func (a *App) achTick(procs map[string]bool) {
	running := a.achRunning(procs)
	a.hotkeyTick(running)
	a.mu.Lock()
	off := a.cfg.Settings.AchToastsOff
	a.mu.Unlock()
	if off {
		clear(achLives)
		return
	}
	here := map[string]bool{}
	for _, r := range running {
		here[r.key] = true
		l := achLives[r.key]
		if l == nil {
			achLives[r.key] = a.achStart(r)
			continue
		}
		if l.off {
			continue
		}
		now, _ := a.env.PCUnlocks(l.appID)
		if now == nil {
			continue
		}
		fresh := core.NewUnlocks(l.known, now)
		l.known = now
		if len(fresh) == 0 {
			continue
		}
		a.achShow(r, l, fresh)
	}
	for k := range achLives {
		if !here[k] {
			delete(achLives, k)
		}
	}
}

// achStart notes what a game has unlocked when it starts, so only what's
// unlocked from then on is shown.
func (a *App) achStart(r achRun) *achLive {
	l := &achLive{name: r.name, player: r.player, known: map[string]time.Time{}}
	appID := r.appID
	if appID <= 0 {
		appID = core.FolderAppID(r.dir)
	}
	if appID <= 0 {
		appID = knownStoreAppID(r.name)
	}
	if appID <= 0 {
		l.off = true
		return l
	}
	if ac := core.AntiCheat(r.dir); ac != "" {
		a.logAchOnce("ac|"+r.key, "No achievement toasts for %s: it ships with %s", r.name, ac)
		l.off = true
		return l
	}
	l.appID = appID
	if u, _ := a.env.PCUnlocks(appID); u != nil {
		l.known = u
	}
	if set := a.achievements(r.key, appID, r.dir, r.name, false); set != nil {
		l.list = set.List
	}
	a.mu.Lock()
	rare := a.cfg.Settings.AchRarity
	a.mu.Unlock()
	if rare {
		go a.globalPercent(appID, true) // ready by the first unlock
	}
	return l
}

func (a *App) achShow(r achRun, l *achLive, fresh []string) {
	byID := map[string]core.Achievement{}
	find := func() {
		for _, x := range l.list {
			byID[x.ID] = x
			byID[strings.ToLower(x.ID)] = x
		}
	}
	find()
	for _, id := range fresh {
		if _, ok := byID[id]; !ok {
			// Names may only now be on the PC: read them again once.
			if set := a.achievements(r.key, l.appID, r.dir, r.name, false); set != nil {
				l.list = set.List
				find()
			}
			break
		}
	}
	a.mu.Lock()
	rare := a.cfg.Settings.AchRarity
	color, who, many := "", "", len(a.cfg.Players) > 1
	if p := a.cfg.Player(l.player); p != nil {
		color, who = p.Color, p.Name
	}
	a.mu.Unlock()
	var pct map[string]float64
	if rare {
		pct = a.globalPercent(l.appID, false)
	}
	kicker := "Achievement unlocked"
	if who != "" && many {
		kicker += " for " + who
	}
	if len(fresh) > 4 {
		// A whole file came in at once (restored saves, a tool set up):
		// one toast, not a minute of them.
		win.ShowToast(win.Toast{Kicker: kicker, Title: strconv.Itoa(len(fresh)) + " achievements unlocked", Line: l.name, Accent: color})
	} else {
		for _, id := range fresh {
			x, ok := byID[id]
			if !ok {
				x, ok = byID[strings.ToLower(id)]
			}
			name, line := id, l.name
			if ok {
				if x.Name != "" {
					name = x.Name
				}
				p := x.Percent
				if p <= 0 {
					if v, has := pct[x.ID]; has {
						p = v
					}
				}
				if t := core.RarityText(p); t != "" {
					line = t
				} else if x.Description != "" {
					line = x.Description
				}
			}
			win.ShowToast(win.Toast{Kicker: kicker, Title: name, Line: line, Accent: color})
		}
	}
	a.logf("%s: %d achievement(s) unlocked", l.name, len(fresh))
	a.emit("achUnlocked", map[string]any{"key": r.key, "count": len(fresh)})
}

// apiTestToast shows what a toast over a game looks like.
func (a *App) apiTestToast() {
	a.mu.Lock()
	color := ""
	if p := a.cfg.Player(a.seat); p != nil {
		color = p.Color
	}
	a.mu.Unlock()
	win.ShowToast(win.Toast{Kicker: "Achievement unlocked", Title: "This is how an unlock looks", Line: "Rare, 4.2% of players have it", Accent: color})
}

// ------------------------------------------------------------- rarity ---

var (
	pctMu     sync.Mutex
	pctFailed = map[int]bool{}
)

func percentFile(appID int) string {
	return filepath.Join(DataDir(), "achievements", strconv.Itoa(appID)+".global.json")
}

// globalPercent is the share of Steam's players who have each of a game's
// achievements, kept on this PC for three days. Only with "How rare
// achievements are" on; online lets it ask Steam (no key: the public
// numbers).
func (a *App) globalPercent(appID int, online bool) map[string]float64 {
	file := percentFile(appID)
	b, err := os.ReadFile(file)
	fresh := err == nil && time.Since(modTime(file)) < 3*24*time.Hour
	pctMu.Lock()
	failed := pctFailed[appID]
	pctMu.Unlock()
	if !fresh && online && !failed {
		u := "https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v0002/?format=json&gameid=" + strconv.Itoa(appID)
		if nb, ferr := fetchURL(u); ferr == nil && core.ParseGlobalPercentages(nb) != nil {
			_ = os.MkdirAll(filepath.Dir(file), 0o755)
			_ = os.WriteFile(file, nb, 0o644)
			b = nb
		} else {
			pctMu.Lock()
			pctFailed[appID] = true // once per run
			pctMu.Unlock()
		}
	}
	if len(b) == 0 {
		return nil
	}
	return core.ParseGlobalPercentages(b)
}
