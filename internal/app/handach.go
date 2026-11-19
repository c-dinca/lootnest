package app

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Achievements marked by hand (core/handach.go), for a game that keeps none
// on this PC. Turning it on for a game asks Steam's public achievement list
// once (no key; only the game's app id is sent) and keeps it on this PC;
// each profile then marks what it has done. Lootnest never writes into the
// game or its saves for this.
// ---------------------------------------------------------------------------

var (
	handMu     sync.Mutex
	handFailed = map[int]bool{}
)

// handListAge is how long a game's list from Steam is kept before it's
// asked again (only when its page is open).
const handListAge = 30 * 24 * time.Hour

func handListFile(appID int) string {
	return filepath.Join(DataDir(), "achievements", "public-"+strconv.Itoa(appID)+".json")
}

// handList is a game's achievements as Steam lists them for everyone. With
// online, a list older than handListAge (or none) is asked for; otherwise
// it's what's kept on this PC.
func (a *App) handList(appID int, online bool) ([]core.Achievement, error) {
	file := handListFile(appID)
	b, err := os.ReadFile(file)
	fresh := err == nil && time.Since(modTime(file)) < handListAge
	if !fresh && online {
		handMu.Lock()
		failed := handFailed[appID]
		handMu.Unlock()
		if !failed || len(b) == 0 {
			u := "https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?language=english&appid=" + strconv.Itoa(appID)
			nb, ferr := fetchURL(u)
			if ferr == nil && len(core.ParseGameAchievements(nb, appID)) > 0 {
				_ = os.MkdirAll(filepath.Dir(file), 0o755)
				_ = os.WriteFile(file, nb, 0o644)
				b = nb
			} else {
				handMu.Lock()
				handFailed[appID] = true // once per run, unless there's nothing kept
				handMu.Unlock()
				if ferr != nil {
					a.logf("Achievement list for app %d from Steam: %v", appID, ferr)
				} else {
					a.logf("Achievement list for app %d from Steam: none listed", appID)
				}
				if len(b) == 0 {
					if ferr != nil {
						return nil, errors.New("Lootnest couldn't reach Steam for this game's achievements. Check the connection, then try again.")
					}
					return nil, errors.New("Steam lists no achievements for this game.")
				}
			}
		}
	}
	list := core.ParseGameAchievements(b, appID)
	if len(list) == 0 {
		return nil, errors.New("Steam lists no achievements for this game.")
	}
	return list, nil
}

// handSet is a game's achievements as each profile marked them.
func (a *App) handSet(appID int, hand *core.HandGame, name string, online bool) *core.AchievementSet {
	list, err := a.handList(appID, online)
	if err != nil {
		a.logAchOnce("hand|"+strconv.Itoa(appID), "Achievements for %s (app %d), marked by hand: %v", name, appID, err)
		return nil
	}
	a.mu.Lock()
	var ids []string
	for _, p := range a.cfg.Players {
		ids = append(ids, p.ID)
	}
	a.mu.Unlock()
	set := core.HandSet(list, appID, ids, hand)
	if set != nil {
		a.logAchOnce("hand|"+strconv.Itoa(appID), "Achievements for %s (app %d): %d, marked by hand", name, appID, set.Total)
	}
	return set
}

// achArgs is what achievements needs for a Library key.
func (a *App) achArgs(key string) (appID int, dir, name string, err error) {
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		g := a.cfg.Game(id)
		if g == nil {
			return 0, "", "", errors.New("That game no longer exists.")
		}
		return g.Launch.AppID, g.Launch.Exe, g.Name, nil
	}
	f, ok := a.foundByKey(key)
	if !ok {
		return 0, "", "", errFoundGone
	}
	return f.AppID, f.Dir, f.Name, nil
}

// sendAchievements reads a game's achievements again and sends them.
func (a *App) sendAchievements(key string, online bool) {
	appID, dir, name, err := a.achArgs(key)
	if err != nil {
		return
	}
	a.emit("achievements", map[string]any{"key": key, "set": a.achievements(key, appID, dir, name, online)})
}

// apiHandAch turns marking by hand on (asking Steam for the game's list
// first) or off (the marks are kept) for a game. The answer comes as a
// "handAch" event, then its achievements as an "achievements" event.
func (a *App) apiHandAch(key string, on bool) error {
	appID, dir, name, err := a.achArgs(key)
	if err != nil {
		return err
	}
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
		dir = filepath.Dir(dir)
	}
	appID = a.achAppID(key, g, appID, dir, name)
	if on && appID <= 0 {
		return errors.New("Lootnest doesn't know which Steam game this is, so it can't get its list of achievements.")
	}
	go func() {
		out := map[string]any{"key": key, "on": on}
		defer func() { a.emit("handAch", out) }()
		if on {
			handMu.Lock()
			delete(handFailed, appID) // the user asked: try again now
			handMu.Unlock()
			if _, err := a.handList(appID, true); err != nil {
				out["error"] = err.Error()
				return
			}
		}
		if err := a.store.UpdateHandAch(func(h *core.HandAchievements) error { h.SetHandOn(key, appID, on); return nil }); err != nil {
			out["error"] = err.Error()
			return
		}
		if on {
			a.logf("%s: achievements marked by hand (app %d)", name, appID)
		} else {
			a.logf("%s: achievements no longer marked by hand; the marks are kept", name)
		}
		a.sendAchievements(key, false)
	}()
	return nil
}

// apiMarkAch marks (on) or unmarks one achievement of a game for a profile.
func (a *App) apiMarkAch(key, player, id string, on bool) error {
	a.mu.Lock()
	known := false
	for _, p := range a.cfg.Players {
		known = known || p.ID == player
	}
	a.mu.Unlock()
	if !known {
		return errors.New("That profile no longer exists.")
	}
	if err := a.store.UpdateHandAch(func(h *core.HandAchievements) error { return h.Mark(key, player, id, on, time.Now()) }); err != nil {
		return err
	}
	go a.sendAchievements(key, false)
	return nil
}
