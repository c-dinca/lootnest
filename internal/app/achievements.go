//go:build windows

package app

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
)

var (
	achMu     sync.Mutex
	achLogged = map[string]bool{}
)

// apiAchievements reads a game's achievements from this PC, off the UI
// thread, and sends them as an "achievements" event (set is null when
// there are none to show). Nothing goes online: names and pictures come
// from Steam's library cache or from the files next to the game.
func (a *App) apiAchievements(key string, appID int, dir, name string) {
	go func() {
		set := a.achievements(key, appID, dir, name, true)
		a.emit("achievements", map[string]any{"key": key, "set": set})
	}()
}

// achievements reads one game's. online lets it ask Steam for the names of
// a game's achievements (with the user's Steam Web API key); otherwise only
// what's saved on this PC is used.
func (a *App) achievements(key string, appID int, dir, name string, online bool) *core.AchievementSet {
	var g *core.Game
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		if gg := a.cfg.Game(id); gg != nil {
			c := *gg
			g = &c
		}
		a.mu.Unlock()
	}
	if strings.EqualFold(filepath.Ext(dir), ".exe") {
		dir = filepath.Dir(dir) // the game's exe was given
	}
	if g != nil && appID <= 0 {
		appID = core.GameAppID(g)
	}
	if appID <= 0 && g == nil {
		if f, ok := a.foundByKey(key); ok {
			appID = f.SteamID // the id another launcher (Hydra) knows
		}
	}
	if appID <= 0 {
		appID = core.FolderAppID(dir)
	}
	if appID <= 0 {
		appID = knownStoreAppID(name)
	}
	if appID <= 0 {
		a.logAchOnce("none|"+key, "Achievements for %s: no Steam app id next to the game, so none to read", name)
		return nil
	}
	var list []core.Achievement
	total := 0
	if a.env.SteamUser != nil {
		if u, err := a.env.SteamUser(); err == nil {
			list, total, _ = core.SteamCacheAchievements(u, appID)
		}
	}
	if len(list) == 0 {
		list = core.LocalSchema(dir)
	}
	unlocks, file := a.env.PCUnlocks(appID)
	// Steam's library cache keeps only some of a game's achievements, and a
	// game in a folder may have only their ids: Steam's schema fills in the
	// rest, when the user gave a Steam Web API key.
	if len(list) == 0 && unlocks == nil && total == 0 {
		// nothing on this PC: no reason to ask
	} else if len(list) < total || len(list) == 0 || unnamed(list) {
		if schema := a.steamSchema(appID, online); len(schema) > 0 {
			list = core.CompleteList(list, schema)
			if total < len(list) {
				total = len(list)
			}
		}
	}

	// Each player's own, when the game has profiles and its achievements
	// go with them: the loaded player's are on the PC, the others' in
	// their profiles.
	var own map[string]map[string]time.Time
	if g != nil && !g.SharedAchievements {
		owner := a.store.LoadState()[g.ID]
		a.mu.Lock()
		players := append([]core.Player{}, a.cfg.Players...)
		a.mu.Unlock()
		found := map[string]map[string]time.Time{}
		for _, p := range players {
			if p.ID == owner {
				if unlocks != nil {
					found[p.ID] = unlocks
				}
			} else if u, ok := a.eng.ProfileUnlocks(p.ID, g); ok {
				found[p.ID] = u
			}
		}
		if len(found) > 0 {
			own = found
			// Players with a profile and none yet start at zero.
			for _, p := range players {
				if _, ok := own[p.ID]; !ok && (a.eng.HasProfile(p.ID, g.ID) || p.ID == owner) {
					own[p.ID] = map[string]time.Time{}
				}
			}
		}
	}
	if len(list) == 0 && total == 0 && unlocks == nil && own == nil {
		a.logAchOnce("none|"+key, "Achievements for %s (app %d): none found on this PC", name, appID)
		return nil
	}
	set := &core.AchievementSet{AppID: appID, Source: "steam"}
	if unlocks != nil || own != nil {
		set.Source, set.File = "pc", file
	}
	set.List, set.Total, set.Unlocked = core.MergeAchievements(append([]core.Achievement{}, list...), total, unlocks)
	if own != nil {
		set.Players = map[string]*core.AchievementSet{}
		for pid, u := range own {
			ps := &core.AchievementSet{AppID: appID, Source: "pc"}
			ps.List, ps.Total, ps.Unlocked = core.MergeAchievements(append([]core.Achievement{}, list...), total, u)
			set.Players[pid] = ps
			if ps.Total > set.Total {
				set.Total = ps.Total
			}
		}
		for _, ps := range set.Players {
			ps.Total = set.Total // so the counts compare
		}
	}
	a.mu.Lock()
	rare := a.cfg.Settings.AchRarity
	a.mu.Unlock()
	if rare {
		if pct := a.globalPercent(appID, online); pct != nil {
			core.ApplyPercentages(set.List, pct)
			for _, ps := range set.Players {
				core.ApplyPercentages(ps.List, pct)
			}
		}
	}
	achMu.Lock()
	first := !achLogged[file+"|"+strconv.Itoa(appID)]
	achLogged[file+"|"+strconv.Itoa(appID)] = true
	achMu.Unlock()
	if first {
		from := "Steam's library cache"
		if file != "" {
			from = file
		}
		if own != nil {
			from += ", each player's own"
		}
		a.logf("Achievements for %s (app %d): %d of %d, from %s", name, appID, set.Unlocked, set.Total, from)
	}
	return set
}

// apiSetSharedAchievements chooses whether a game's achievements are each
// player's own (swapped with their profile) or one set for everyone.
func (a *App) apiSetSharedAchievements(gameID string, shared bool) error {
	if a.activeSessionFor(gameID) != nil {
		return errors.New("Wait until the game has closed, then change this.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil {
		return errors.New("That game no longer exists.")
	}
	g.SharedAchievements = shared
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	if shared {
		a.logf("%s: achievements are shared by everyone", g.Name)
	} else {
		a.logf("%s: each player has their own achievements", g.Name)
	}
	return nil
}

// achSummary is one game on the Achievements page: its counts and the
// latest unlocked, for everyone (or each profile, when they have their own).
type achSummary struct {
	Key      string                `json:"key"`
	Name     string                `json:"name"`
	Source   string                `json:"source"`
	Total    int                   `json:"total"`
	Unlocked int                   `json:"unlocked"`
	Recent   []core.Achievement    `json:"recent"`
	Players  map[string]achPerSlot `json:"players,omitempty"`
}

type achPerSlot struct {
	Unlocked int                `json:"unlocked"`
	Recent   []core.Achievement `json:"recent"`
}

func latestUnlocked(list []core.Achievement, n int) []core.Achievement {
	out := []core.Achievement{}
	for _, a := range list { // unlocked come first, newest first
		if !a.Unlocked || len(out) >= n {
			break
		}
		out = append(out, a)
	}
	return out
}

// apiAllAchievements reads every game's achievements for the Achievements
// page, off the UI thread, and sends them as an "allAchievements" event.
func (a *App) apiAllAchievements() {
	go func() {
		type item struct {
			key, name, dir string
			appID          int
		}
		var items []item
		a.mu.Lock()
		for _, g := range a.cfg.Games {
			items = append(items, item{"game:" + g.ID, g.Name, g.Launch.Exe, g.Launch.AppID})
		}
		a.mu.Unlock()
		for _, f := range a.visibleFound(a.libraryGames()) {
			items = append(items, item{f.Key, f.Name, f.Dir, f.AppID})
		}
		out := []achSummary{}
		for _, it := range items {
			set := a.achievements(it.key, it.appID, it.dir, it.name, false)
			if set == nil || set.Total == 0 {
				continue
			}
			s := achSummary{Key: it.key, Name: it.name, Source: set.Source, Total: set.Total, Unlocked: set.Unlocked, Recent: latestUnlocked(set.List, 8)}
			if len(set.Players) > 0 {
				s.Players = map[string]achPerSlot{}
				for pid, ps := range set.Players {
					s.Players[pid] = achPerSlot{Unlocked: ps.Unlocked, Recent: latestUnlocked(ps.List, 8)}
				}
			}
			out = append(out, s)
		}
		a.emit("allAchievements", map[string]any{"games": out})
	}()
}

func unnamed(list []core.Achievement) bool {
	for _, a := range list {
		if a.Name == "" {
			return true
		}
	}
	return false
}

var (
	schemaMu     sync.Mutex
	schemaFailed = map[int]bool{}
)

func schemaFile(appID int) string {
	return filepath.Join(DataDir(), "achievements", strconv.Itoa(appID)+".json")
}

// steamSchema is a game's full list of achievements from Steam's Web API,
// kept on this PC for a week. Only with the user's own key; without one, or
// offline, it's whatever was saved before.
func (a *App) steamSchema(appID int, online bool) []core.Achievement {
	file := schemaFile(appID)
	b, err := os.ReadFile(file)
	fresh := err == nil && time.Since(modTime(file)) < 7*24*time.Hour
	if !fresh && online {
		a.mu.Lock()
		key := revealSecret(a.cfg.Settings.SteamAPIKey)
		a.mu.Unlock()
		schemaMu.Lock()
		failed := schemaFailed[appID]
		schemaMu.Unlock()
		if strings.TrimSpace(key) != "" && !failed {
			u := "https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?l=english&appid=" + strconv.Itoa(appID) + "&key=" + url.QueryEscape(strings.TrimSpace(key))
			if nb, ferr := fetchURL(u); ferr == nil && len(nb) > 0 {
				if len(core.ParseSteamSchema(nb, appID)) > 0 {
					_ = os.MkdirAll(filepath.Dir(file), 0o755)
					_ = os.WriteFile(file, nb, 0o644)
				}
				b = nb
			} else {
				schemaMu.Lock()
				schemaFailed[appID] = true // once per run
				schemaMu.Unlock()
				if ferr != nil {
					a.logf("Achievement names for app %d: %v", appID, scrubKey(ferr.Error(), key))
				}
			}
		}
	}
	if len(b) == 0 {
		return nil
	}
	return core.ParseSteamSchema(b, appID)
}

// scrubKey keeps the user's key out of the log.
func scrubKey(s, key string) string {
	if k := strings.TrimSpace(key); k != "" {
		s = strings.ReplaceAll(s, k, "…")
	}
	return s
}

// knownStoreAppID is the Steam app id Lootnest already looked up by name
// for a game outside Steam, for its cover or its news (saved on this PC).
// It never goes online itself.
func knownStoreAppID(name string) int {
	if strings.TrimSpace(name) == "" {
		return 0
	}
	offline := func(string) ([]byte, error) { return nil, errors.New("offline") }
	for _, d := range []string{newsDir(), coversDir()} {
		if id, err := core.StoreAppID(offline, d, name); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

// logAchOnce writes a line to the log once per run, so "Show recent
// activity" says why a game shows no achievements.
func (a *App) logAchOnce(id, format string, args ...any) {
	achMu.Lock()
	first := !achLogged[id]
	achLogged[id] = true
	achMu.Unlock()
	if first {
		a.logf(format, args...)
	}
}
