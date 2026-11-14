//go:build windows

package app

import (
	"errors"
	"strconv"
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
	// Will it run: this PC against the requirements. The game is already
	// here, so free space isn't part of it.
	if fetched != nil && (len(fetched.Minimum) > 0 || len(fetched.Recommended) > 0) {
		pc := a.pcSpecs()
		pc.Free, pc.FreeOn = 0, ""
		v := core.WillItRun(pc, fetched.Minimum, fetched.Recommended)
		out["run"] = map[string]any{"verdict": v, "plan": core.PlanRun(v), "pc": pc,
			"minimum": fetched.Minimum, "recommended": fetched.Recommended}
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

// Reviews read this run, by app id and kind, kept for six hours.
var reviewsKept = struct {
	sync.Mutex
	m map[string]reviewsEntry
}{m: map[string]reviewsEntry{}}

type reviewsEntry struct {
	at    time.Time
	items []core.Review
}

// apiGameReviews answers with a "reviews" event: the most helpful Steam
// reviews of the past year for the game at key (kind "all", "positive" or
// "negative"). Only with Game details on; nothing is kept on disk.
func (a *App) apiGameReviews(key, kind string) {
	if !a.detailsOn() {
		return
	}
	if kind != "positive" && kind != "negative" {
		kind = "all"
	}
	go func() {
		out := map[string]any{"key": key, "kind": kind}
		id := a.detailsAppID(key)
		if id <= 0 {
			out["none"] = true
			a.emit("reviews", out)
			return
		}
		ck := strconv.Itoa(id) + "/" + kind
		reviewsKept.Lock()
		e, ok := reviewsKept.m[ck]
		reviewsKept.Unlock()
		if !ok || time.Since(e.at) > 6*time.Hour {
			items, err := core.FetchReviews(fetchURL, id, "english", kind)
			if err != nil {
				a.logf("Reviews for app %d: %v", id, err)
				out["error"] = "Couldn't reach the Steam store."
				a.emit("reviews", out)
				return
			}
			e = reviewsEntry{time.Now(), items}
			reviewsKept.Lock()
			reviewsKept.m[ck] = e
			reviewsKept.Unlock()
		}
		out["items"], out["appId"] = e.items, id
		a.emit("reviews", out)
	}()
}
