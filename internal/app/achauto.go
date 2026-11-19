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

// ---------------------------------------------------------------------------
// Achievements that find their own way to the page: read again by
// themselves after a game closes (a tool writes its file at the first
// unlock, Steam its cache when the game ends), and from the user's own
// Steam account for games they also own there (Settings, Online).
// ---------------------------------------------------------------------------

// accountAge is how long the account's answer for a game is kept before
// Steam is asked again (only when the game's page is open, or after it
// was played).
const accountAge = time.Hour

var (
	accountMu    sync.Mutex
	accountStale = map[int]bool{} // played since it was asked
	accountTried = map[int]time.Time{}
	accountSaid  = map[int]string{} // what Steam last said: ok, private, none, error
)

func noteAccount(appID int, said string) {
	accountMu.Lock()
	accountSaid[appID] = said
	accountMu.Unlock()
}

// accountAchievements is a game's achievements on the user's own Steam
// account, unlocked ones with their time, or nil: when it's off, there's no
// key or account, or Steam has none for this account. online lets it ask
// Steam; otherwise only the answer kept on this PC is used.
func (a *App) accountAchievements(appID int, online bool) []core.Achievement {
	a.mu.Lock()
	on, key := a.cfg.Settings.SteamAccountAch, strings.TrimSpace(revealSecret(a.cfg.Settings.SteamAPIKey))
	a.mu.Unlock()
	if !on || key == "" || appID <= 0 || a.env.SteamUser == nil {
		return nil
	}
	u, err := a.env.SteamUser()
	if err != nil {
		return nil
	}
	sid := core.SteamID64(u)
	if sid == "" {
		return nil
	}
	file := filepath.Join(DataDir(), "achievements", "account-"+sid+"-"+strconv.Itoa(appID)+".json")
	b, err := os.ReadFile(file)
	accountMu.Lock()
	stale := err != nil || time.Since(modTime(file)) > accountAge || accountStale[appID]
	recent := time.Since(accountTried[appID]) < time.Minute
	if stale && online && !recent {
		accountTried[appID] = time.Now()
		delete(accountStale, appID)
	}
	accountMu.Unlock()
	if stale && online && !recent {
		q := "https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v1/?l=english&appid=" + strconv.Itoa(appID) + "&steamid=" + sid + "&key=" + url.QueryEscape(key)
		nb, ferr := fetchURL(q)
		switch {
		case ferr != nil:
			// A private profile answers 403 Forbidden.
			if strings.Contains(ferr.Error(), "403") {
				noteAccount(appID, "private")
			} else {
				noteAccount(appID, "error")
			}
			a.logAchOnce("accountfail|"+strconv.Itoa(appID), "Achievements for app %d from the Steam account: %s (is the profile's Game details setting Public?)", appID, scrubKey(ferr.Error(), key))
		case len(nb) > 0:
			// Kept whatever Steam said, "none" included, so it isn't asked
			// again for an hour.
			_ = os.MkdirAll(filepath.Dir(file), 0o755)
			_ = os.WriteFile(file, nb, 0o644)
			b = nb
			if _, perr := core.ParsePlayerAchievements(nb); perr != nil && !errors.Is(perr, core.ErrSteamNoStats) {
				a.logAchOnce("accountfail|"+strconv.Itoa(appID), "Achievements for app %d from the Steam account: %v", appID, perr)
			}
		}
	}
	if len(b) == 0 {
		return nil
	}
	list, err := core.ParsePlayerAchievements(b)
	switch {
	case errors.Is(err, core.ErrSteamPrivate):
		noteAccount(appID, "private")
	case errors.Is(err, core.ErrSteamNoStats):
		noteAccount(appID, "none")
	case err != nil:
		noteAccount(appID, "error")
	default:
		noteAccount(appID, "ok")
	}
	if err != nil {
		return nil
	}
	// Pictures (and hidden flags) come from the game's list, which the same
	// key fetches.
	if schema := a.steamSchema(appID, online); len(schema) > 0 {
		list = core.CompleteList(list, schema)
	}
	return list
}

// achAfterPlay reads a game's achievements again once it has closed, and
// sends them, so new unlocks show without the page being opened again.
func (a *App) achAfterPlay(key string) {
	time.Sleep(8 * time.Second) // let the game, a tool or Steam finish writing
	var g *core.Game
	var appID int
	var dir, name string
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		if gg := a.cfg.Game(id); gg != nil {
			c := *gg
			g, appID, dir, name = &c, gg.Launch.AppID, gg.Launch.Exe, gg.Name
		}
		a.mu.Unlock()
		if g == nil {
			return
		}
	} else if f, ok := a.foundByKey(key); ok {
		appID, dir, name = f.AppID, f.Dir, f.Name
	} else {
		return
	}
	if id := a.achAppID(key, g, appID, dir, name); id > 0 {
		accountMu.Lock()
		accountStale[id] = true
		accountMu.Unlock()
	}
	out := map[string]any{"key": key, "set": a.achievements(key, appID, dir, name, true), "after": true}
	if out["set"] == (*core.AchievementSet)(nil) {
		out["why"] = a.achWhy(key, appID, dir, name)
	}
	a.emit("achievements", out)
}

// achWhy says, for a game with none to show, what showing its Steam
// account's achievements still needs: its Steam app id (0 when none is
// known, and then nothing else matters), whether Steam is signed in on this
// PC, the key, the setting, and what Steam last said ("" when not asked).
func (a *App) achWhy(key string, appID int, dir, name string) map[string]any {
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
	id := a.achAppID(key, g, appID, dir, name)
	if id <= 0 {
		return map[string]any{"appId": 0}
	}
	a.mu.Lock()
	on, hasKey := a.cfg.Settings.SteamAccountAch, strings.TrimSpace(a.cfg.Settings.SteamAPIKey) != ""
	a.mu.Unlock()
	signed := false
	if a.env.SteamUser != nil {
		if u, err := a.env.SteamUser(); err == nil {
			signed = core.SteamID64(u) != ""
		}
	}
	accountMu.Lock()
	said := accountSaid[id]
	accountMu.Unlock()
	return map[string]any{"appId": id, "steam": signed, "key": hasKey, "on": on, "said": said}
}
