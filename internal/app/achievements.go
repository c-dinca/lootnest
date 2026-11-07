//go:build windows

package app

import (
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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
		set := a.achievements(appID, dir)
		a.emit("achievements", map[string]any{"key": key, "set": set})
	}()
}

func (a *App) achievements(appID int, dir string) *core.AchievementSet {
	if strings.EqualFold(filepath.Ext(dir), ".exe") {
		dir = filepath.Dir(dir) // the game's exe was given
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
	if len(list) == 0 && total == 0 && unlocks == nil {
		return nil
	}
	set := &core.AchievementSet{AppID: appID, Source: "steam"}
	if unlocks != nil {
		set.Source, set.File = "pc", file
	}
	set.List, set.Total, set.Unlocked = core.MergeAchievements(list, total, unlocks)
	achMu.Lock()
	first := !achLogged[file+"|"+strconv.Itoa(appID)]
	achLogged[file+"|"+strconv.Itoa(appID)] = true
	achMu.Unlock()
	if first {
		from := "Steam's library cache"
		if file != "" {
			from = file
		}
		a.logf("Achievements for app %d: %d of %d, from %s", appID, set.Unlocked, set.Total, from)
	}
	return set
}
