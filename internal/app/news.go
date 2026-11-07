//go:build windows

package app

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"lootnest/internal/core"
)

// ---------------------------------------------------------------------------
// Game updates and news, only with Settings > "Show game updates and news"
// on. The feed is read in the background; the page gets a "news" event.
// ---------------------------------------------------------------------------

var (
	newsMu     sync.Mutex
	newsFlight = map[string]bool{}
)

func newsDir() string { return filepath.Join(DataDir(), "news") }

// apiGameNews reads a game's news for its page: by Steam app id, or for a
// game outside Steam by its name. exe, when set, is the game's exe: its
// date tells whether an update came out after this copy. The answer is a
// "news" event for key.
func (a *App) apiGameNews(key string, appID int, name, exe string) {
	a.mu.Lock()
	on := a.cfg.Settings.GameNews
	a.mu.Unlock()
	if !on {
		return
	}
	newsMu.Lock()
	if newsFlight[key] {
		newsMu.Unlock()
		return
	}
	newsFlight[key] = true
	newsMu.Unlock()
	go func() {
		defer func() { newsMu.Lock(); delete(newsFlight, key); newsMu.Unlock() }()
		out := map[string]any{"key": key}
		defer func() { a.emit("news", out) }()
		id := appID
		if id <= 0 {
			var err error
			if id, err = core.StoreAppID(fetchURL, newsDir(), name); err != nil {
				out["error"] = "Couldn't reach Steam's news for this game. Lootnest tries again next time."
				return
			}
		}
		if id <= 0 {
			out["none"] = true // not on the Steam store
			return
		}
		n, err := core.FetchNews(fetchURL, newsDir(), id)
		if err != nil && len(n.Items) == 0 {
			a.logf("Couldn't read the news for %s: %v", name, err)
			out["error"] = "Couldn't reach Steam's news for this game. Lootnest tries again next time."
			return
		}
		out["appId"], out["items"] = id, n.Items
		if appID <= 0 && exe != "" {
			if p, err := a.env.ResolveExe(exe); err == nil {
				if u := core.NewerThanCopy(n.Items, newestFile(filepath.Dir(p))); u != nil {
					out["newer"] = u
				}
			}
		}
	}()
}

// newestFile is when the files at the top of a game's folder last changed,
// which is roughly when this copy was installed or last updated.
func newestFile(dir string) time.Time {
	var t time.Time
	ents, err := os.ReadDir(dir)
	if err != nil {
		return t
	}
	for _, e := range ents {
		if info, err := e.Info(); err == nil && !e.IsDir() && info.ModTime().After(t) {
			t = info.ModTime()
		}
	}
	return t
}
