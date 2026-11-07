//go:build windows

package app

import (
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Steam Cloud guard. Steam Cloud stays on. When a session starts, Lootnest
// notes the content of the swapped files Steam syncs; once the game runs it
// checks whether Steam replaced them with the cloud's copy. If it did, the
// session's files are kept as a backup instead of being saved over the
// player's profile, and the player's own files go back in place.
// ---------------------------------------------------------------------------

type cloudWatch struct {
	loaded   map[string]string // swapped file -> SHA-1 right after loading
	replaced bool
}

var (
	cloudMu      sync.Mutex
	cloudWatches = map[string]*cloudWatch{} // by game id
)

func (a *App) cloudMatches(g *core.Game) []core.CloudMatch {
	if g == nil || g.Launch.Type != "steam" {
		return nil
	}
	userDir, err := a.env.SteamUser()
	if err != nil {
		return nil
	}
	files := core.CloudFiles(userDir, g.Launch.AppID)
	if len(files) == 0 {
		return nil
	}
	roots, err := a.eng.LivePaths(g)
	if err != nil {
		return nil
	}
	return core.CloudMatches(files, userDir, g.Launch.AppID, roots)
}

// armCloudWatch runs right after a player's files were loaded for a launch.
func (a *App) armCloudWatch(g *core.Game) {
	ms := a.cloudMatches(g)
	cloudMu.Lock()
	defer cloudMu.Unlock()
	if len(ms) == 0 {
		delete(cloudWatches, g.ID)
		return
	}
	cloudWatches[g.ID] = &cloudWatch{loaded: core.HashMatches(ms)}
}

// checkCloudWatch runs once the game is running: Steam syncs before the
// game starts, so by now any cloud copy is in place. It checks again a
// little later, in case Steam was still writing.
func (a *App) checkCloudWatch(cfg *core.Config, g *core.Game, playerID string) {
	for _, wait := range []time.Duration{2 * time.Second, 15 * time.Second} {
		time.Sleep(wait)
		cloudMu.Lock()
		w := cloudWatches[g.ID]
		cloudMu.Unlock()
		if w == nil || w.replaced {
			return
		}
		// Read the cloud list again: a sync from another PC updates it.
		replaced := core.CloudReplaced(a.cloudMatches(g), w.loaded)
		if len(replaced) == 0 {
			continue
		}
		cloudMu.Lock()
		w.replaced = true
		cloudMu.Unlock()
		who := cfg.PlayerName(playerID)
		a.logf("Steam Cloud replaced %d of %s's files for %s with its own copy: %v", len(replaced), who, g.Name, replaced)
		a.noteConflict(g.ID, playerID, replaced)
		a.notify("Steam Cloud replaced "+who+"'s files",
			g.Name+" is using Steam Cloud's copy. When it closes, Lootnest keeps "+who+"'s profile as it was and saves this session's files as a backup.")
		a.emit("toast", map[string]any{"kind": "error", "text": "Steam Cloud replaced " + who + "'s files in " + g.Name + " with its own copy. " + who + "'s profile stays as it was; this session is kept as a backup."})
		return
	}
}

// takeCloudReplaced reports (and forgets) whether this session's files came
// from Steam Cloud instead of the player's profile.
func takeCloudReplaced(gameID string) bool {
	cloudMu.Lock()
	defer cloudMu.Unlock()
	w := cloudWatches[gameID]
	delete(cloudWatches, gameID)
	return w != nil && w.replaced
}

type CloudStatus struct {
	State string `json:"state"` // "none", "elsewhere" or "overlap"
	Count int    `json:"count"`
}

// apiCloudStatus says whether Steam Cloud syncs any of a game's swapped files.
func (a *App) apiCloudStatus(gameID string) CloudStatus {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil || g.Launch.Type != "steam" {
		return CloudStatus{State: "none"}
	}
	userDir, err := a.env.SteamUser()
	if err != nil || len(core.CloudFiles(userDir, g.Launch.AppID)) == 0 {
		return CloudStatus{State: "none"}
	}
	if n := len(a.cloudMatches(g)); n > 0 {
		return CloudStatus{State: "overlap", Count: n}
	}
	return CloudStatus{State: "elsewhere"}
}
