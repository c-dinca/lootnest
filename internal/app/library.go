//go:build windows

package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Library: every game on the PC, with or without profiles.
//
// Covers come from Steam's own cache on this PC; nothing is downloaded.
// ---------------------------------------------------------------------------

var (
	coverMu     sync.Mutex
	coverCache  = map[int]string{}  // Steam app id -> data: URL from Steam's own cache, or ""
	coverFlight = map[string]bool{} // downloads in progress, by key
)

func dataURL(p string) string {
	b, err := os.ReadFile(p)
	if err != nil || len(b) >= 4<<20 {
		return ""
	}
	mime := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(p), ".png") {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

func coversDir() string { return filepath.Join(DataDir(), "covers") }

var coverHTTP = &http.Client{Timeout: 15 * time.Second}

func fetchURL(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	resp, err := coverHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // no such image: treated as "no cover"
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", req.URL.Host, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// apiCover returns a game's cover as a data: URL, or "" for none yet.
// A Steam game's cover comes from Steam's cache on this PC. Only when the
// user turned on "Download covers", a missing cover is downloaded in the
// background (never on the UI thread) and arrives as a "cover" event for key.
func (a *App) apiCover(appID int, name, key string) string {
	if appID > 0 {
		coverMu.Lock()
		c, ok := coverCache[appID]
		coverMu.Unlock()
		if !ok {
			if p := core.SteamCover(a.env.SteamPath, appID); p != "" {
				c = dataURL(p)
			}
			coverMu.Lock()
			coverCache[appID] = c
			coverMu.Unlock()
		}
		if c != "" {
			return c
		}
	}
	a.mu.Lock()
	on := a.cfg.Settings.DownloadCovers
	a.mu.Unlock()
	if !on {
		return ""
	}
	dir := coversDir()
	if p := core.SavedCover(dir, name, appID); p != "" {
		return dataURL(p)
	}
	coverMu.Lock()
	busy := coverFlight[key]
	coverFlight[key] = true
	coverMu.Unlock()
	if !busy {
		go func() {
			defer func() { coverMu.Lock(); delete(coverFlight, key); coverMu.Unlock() }()
			p, err := core.DownloadedCover(fetchURL, dir, name, appID)
			if err != nil {
				a.logf("Couldn't download the cover for %s: %v", name, err)
				return
			}
			if p != "" {
				a.logf("Downloaded the cover for %s", name)
				a.emit("cover", map[string]any{"key": key, "url": dataURL(p)})
			}
		}()
	}
	return ""
}

// apiLaunchFound starts a game that isn't set up for profiles: Steam games
// through Steam, others by their exe.
func (a *App) apiLaunchFound(key string) error {
	f, found := a.foundByKey(key)
	if !found {
		return errFoundGone
	}
	name, exe, appID, args := f.Name, f.Exe, f.AppID, f.Args
	if appID > 0 {
		if a.env.SteamPath == "" {
			return errors.New("Steam wasn't found on this PC")
		}
		a.logf("Starting %s through Steam", name)
		return win.StartDetached(a.env.SteamPath+`\steam.exe`, strings.TrimSpace(fmt.Sprintf("-applaunch %d %s", appID, args)), a.env.SteamPath)
	}
	path, err := a.env.ResolveExe(exe)
	if err != nil {
		return err
	}
	a.logf("Starting %s", name)
	return win.StartDetached(path, args, dirOf(path))
}
