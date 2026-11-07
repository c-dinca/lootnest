//go:build windows

package app

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
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
		grid := a.grid()
		go func() {
			defer func() { coverMu.Lock(); delete(coverFlight, key); coverMu.Unlock() }()
			p, err := core.DownloadedArt(fetchURL, grid, dir, name, appID, core.ArtCover)
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
	_ = a.store.RecordPlay(key, "", time.Now())
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

var (
	iconMu    sync.Mutex
	iconCache = map[string]string{} // exe path -> data: URL of its icon, or ""
)

// apiGameIcon returns the icon of a game's exe as a PNG data: URL, or ""
// when it has none. It's what the Library shows for a game without a cover;
// nothing goes online. Icons are kept in <data>\covers once extracted.
func (a *App) apiGameIcon(exe string) string {
	path, err := a.env.ResolveExe(exe)
	if err != nil {
		return ""
	}
	key := strings.ToLower(path)
	iconMu.Lock()
	c, ok := iconCache[key]
	iconMu.Unlock()
	if ok {
		return c
	}
	sum := sha1.Sum([]byte(key))
	file := filepath.Join(coversDir(), "icon-"+hex.EncodeToString(sum[:8])+".png")
	if st, err := os.Stat(file); err != nil || st.ModTime().Before(modTime(path)) {
		c = ""
		if img, err := win.ExeIcon(path, 256); err == nil {
			var buf bytes.Buffer
			if png.Encode(&buf, img) == nil && os.MkdirAll(coversDir(), 0o755) == nil {
				_ = os.WriteFile(file, buf.Bytes(), 0o644)
			}
		}
	}
	if _, err := os.Stat(file); err == nil {
		c = dataURL(file)
	}
	iconMu.Lock()
	iconCache[key] = c
	iconMu.Unlock()
	return c
}

func modTime(p string) time.Time {
	if st, err := os.Stat(p); err == nil {
		return st.ModTime()
	}
	return time.Time{}
}

var (
	artMu    sync.Mutex
	artCache = map[string]string{} // "<kind>:<appid>" -> data: URL from Steam's cache, or ""
)

// fetchAuth is fetchURL with an "Authorization: Bearer" header (SteamGridDB).
func fetchAuth(u, bearer string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := coverHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	case http.StatusNotFound:
		return nil, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, errors.New("SteamGridDB didn't accept the API key in Settings")
	}
	return nil, fmt.Errorf("%s answered %s", req.URL.Host, resp.Status)
}

// grid is SteamGridDB with the user's key, or nil without one.
func (a *App) grid() *core.GridDB {
	a.mu.Lock()
	key := a.cfg.Settings.GridDBKey
	a.mu.Unlock()
	if strings.TrimSpace(key) == "" {
		return nil
	}
	return &core.GridDB{Key: key, Fetch: fetchAuth}
}

// apiArt returns a game's wide banner ("hero") or transparent logo ("logo")
// as a data: URL, or "". A Steam game's comes from Steam's cache on this PC.
// Only with "Download covers and artwork" on, missing art is downloaded in
// the background and arrives as an "art" event for "<kind>:<key>".
func (a *App) apiArt(appID int, name, key, kind string) string {
	if kind != core.ArtHero && kind != core.ArtLogo {
		return ""
	}
	if appID > 0 {
		k := fmt.Sprintf("%s:%d", kind, appID)
		artMu.Lock()
		c, ok := artCache[k]
		artMu.Unlock()
		if !ok {
			p := core.SteamHero(a.env.SteamPath, appID)
			if kind == core.ArtLogo {
				p = core.SteamLogo(a.env.SteamPath, appID)
			}
			if p != "" {
				c = dataURL(p)
			}
			artMu.Lock()
			artCache[k] = c
			artMu.Unlock()
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
	if p := core.SavedArt(dir, name, appID, kind); p != "" {
		return dataURL(p)
	}
	flight := kind + ":" + key
	coverMu.Lock()
	busy := coverFlight[flight]
	coverFlight[flight] = true
	coverMu.Unlock()
	if !busy {
		grid := a.grid()
		go func() {
			defer func() { coverMu.Lock(); delete(coverFlight, flight); coverMu.Unlock() }()
			p, err := core.DownloadedArt(fetchURL, grid, dir, name, appID, kind)
			if err != nil {
				a.logf("Couldn't download the %s for %s: %v", kind, name, err)
				return
			}
			if p != "" {
				a.emit("art", map[string]any{"key": flight, "url": dataURL(p)})
			}
		}()
	}
	return ""
}

var (
	activityMu   sync.Mutex
	activityAt   time.Time // localconfig.vdf's time when last read
	activityPath string
	activityData map[int]core.SteamPlay
)

// steamActivity is core.SteamActivity, read again only when Steam changed
// its settings file.
func (a *App) steamActivity() map[int]core.SteamPlay {
	dir, err := a.env.SteamUser()
	if err != nil {
		return nil
	}
	p := filepath.Join(dir, "config", "localconfig.vdf")
	t := modTime(p)
	activityMu.Lock()
	defer activityMu.Unlock()
	if p != activityPath || !t.Equal(activityAt) || activityData == nil {
		activityData, activityAt, activityPath = core.SteamActivity(dir), t, p
	}
	return activityData
}
