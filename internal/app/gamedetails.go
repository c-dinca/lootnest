//go:build windows

package app

import (
	"errors"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Game details (core/gamedetails.go), only with Settings.GameDetails on and
// only for the page that's open. The page gets a "details" event: the copy
// kept on this PC at once, then Steam's newer one when that copy is missing
// or a month old. The user's edits always win.
// ---------------------------------------------------------------------------

var (
	detailsMu     sync.Mutex
	detailsFlight = map[string]bool{}
)

// detailsAppID is the Steam app id of a Library game, from what's on this
// PC only: its launch, the id another launcher knows, the folder, or a name
// already looked up for its cover or news.
func (a *App) detailsAppID(key string) int {
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		gg := a.cfg.Game(id)
		var g core.Game
		if gg != nil {
			g = *gg
		}
		a.mu.Unlock()
		if gg == nil {
			return 0
		}
		if n := core.GameAppID(&g); n > 0 {
			return n
		}
		return knownStoreAppID(g.Name)
	}
	f, ok := a.foundByKey(key)
	if !ok {
		return 0
	}
	if n := max(f.AppID, f.SteamID); n > 0 {
		return n
	}
	if n := core.FolderAppID(f.Dir); n > 0 {
		return n
	}
	return knownStoreAppID(f.Name)
}

func (a *App) detailsOn() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.Settings.GameDetails
}

// emitDetails sends what the page shows: Steam's details with the user's
// edits over them, or nothing when there's neither.
func (a *App) emitDetails(key string, fetched *core.GameDetails, extra map[string]any) {
	out := map[string]any{"key": key}
	for k, v := range extra {
		out[k] = v
	}
	if fetched != nil && fetched.Unknown {
		fetched = nil
	}
	under, from := a.pluginDetails(key)
	fetched = core.LayerDetails(under, from, fetched)
	e := core.LoadDetailsEdit(DataDir(), key)
	if fetched != nil || !e.Empty() {
		out["details"], out["edited"] = core.MergeDetails(fetched, e), !e.Empty()
	}
	a.emit("details", out)
}

// apiGameDetails answers with "details" events for the game at key.
func (a *App) apiGameDetails(key string) {
	if !a.detailsOn() {
		if len(a.plugins().meta) > 0 {
			go a.emitDetails(key, nil, nil)
		}
		return
	}
	detailsMu.Lock()
	if detailsFlight[key] {
		detailsMu.Unlock()
		return
	}
	detailsFlight[key] = true
	detailsMu.Unlock()
	go func() {
		defer func() { detailsMu.Lock(); delete(detailsFlight, key); detailsMu.Unlock() }()
		dir, id := DataDir(), a.detailsAppID(key)
		var kept *core.GameDetails
		stale := true
		if id > 0 {
			kept, stale = core.LoadDetails(dir, id)
		}
		if !stale || id <= 0 {
			a.emitDetails(key, kept, nil)
			return
		}
		if kept != nil {
			a.emitDetails(key, kept, map[string]any{"fetching": true})
		}
		d, err := core.FetchGameDetails(fetchURL, id, "english")
		if err != nil {
			a.logf("Game details for app %d: %v", id, err)
			extra := map[string]any{}
			if kept == nil {
				extra["error"] = "Couldn't reach the Steam store. Lootnest tries again next time you open this page."
			}
			a.emitDetails(key, kept, extra)
			return
		}
		if d == nil && kept != nil && !kept.Unknown {
			// Gone from the store, or not sold in this region: what was read
			// before still holds.
			d, d.Fetched = kept, time.Now().UTC()
		} else if d == nil {
			d = &core.GameDetails{Unknown: true, Fetched: time.Now().UTC()}
		}
		if err := core.SaveDetails(dir, id, d); err != nil {
			a.logf("Couldn't keep the game details for app %d: %v", id, err)
		}
		a.emitDetails(key, d, nil)
	}()
}

// apiSaveDetailsEdit keeps the fields the user changed (those that differ
// from Steam's); an empty edit puts Steam's back.
func (a *App) apiSaveDetailsEdit(key string, e core.DetailsEdit) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("unknown game")
	}
	dir := DataDir()
	var kept *core.GameDetails
	if id := a.detailsAppID(key); id > 0 {
		if kept, _ = core.LoadDetails(dir, id); kept != nil && kept.Unknown {
			kept = nil
		}
	}
	under, from := a.pluginDetails(key)
	if err := core.SaveDetailsEdit(dir, key, core.CleanDetailsEdit(e, core.LayerDetails(under, from, kept))); err != nil {
		return err
	}
	a.emitDetails(key, kept, nil)
	return nil
}
