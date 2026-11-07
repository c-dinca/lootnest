//go:build windows

package app

import (
	"errors"
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
func (a *App) apiAchievements(key string, appID int, dir string) {
	go func() {
		set := a.achievements(key, appID, dir)
		a.emit("achievements", map[string]any{"key": key, "set": set})
	}()
}

func (a *App) achievements(key string, appID int, dir string) *core.AchievementSet {
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
	if appID <= 0 {
		appID = core.FolderAppID(dir)
	}
	if appID <= 0 {
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
		a.logf("Achievements for app %d: %d of %d, from %s", appID, set.Unlocked, set.Total, from)
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
