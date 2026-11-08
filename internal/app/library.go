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

func dataURL(p string) string { return dataURLMax(p, 4<<20) }

func dataURLMax(p string, limit int64) string {
	if st, err := os.Stat(p); err != nil || st.Size() >= limit {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil || int64(len(b)) >= limit {
		return ""
	}
	mime := "image/jpeg"
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png":
		mime = "image/png"
	case ".webp":
		mime = "image/webp"
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
	if _, busy := startingFound.LoadOrStore(key, true); busy {
		return nil // already starting, once its saves are kept
	}
	_ = a.store.RecordPlay(key, "", time.Now())
	if updated := a.updatedSince(key, a.gameBuild(f.AppID, f.Exe)); updated || !a.launchOpts(key).Empty() {
		// Updated since it last started: its saves are kept as a point
		// first, away from the window's thread, then it starts. Apps to
		// close before it, and Windows' administrator prompt, wait there too.
		go func() {
			defer startingFound.Delete(key)
			if updated {
				a.pointBeforeUpdate(key)
			}
			if err := a.launchFound(key, f); err != nil {
				a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
			}
		}()
		return nil
	}
	startingFound.Delete(key)
	return a.launchFound(key, f)
}

// startingFound holds the Library games waiting for a save point before
// they start, so a second Play doesn't start one twice.
var startingFound sync.Map

func (a *App) launchFound(key string, f core.FoundGame) error {
	if f.ProcessName != "" { // followed until it closes, for the steps after
		a.stepsBefore(key, f.Name)
	} else if a.launchOpts(key).HasSteps() {
		a.logf("Lootnest can't tell when %s closes, so nothing runs before or after it", f.Name)
	}
	err := a.startFound(key, f)
	if err != nil {
		a.stepsAfter(key, false)
	}
	return err
}

func (a *App) startFound(key string, f core.FoundGame) error {
	name, exe, appID, args := f.Name, f.Exe, f.AppID, f.Args
	if appID > 0 {
		if a.env.SteamPath == "" {
			return errors.New("Steam wasn't found on this PC")
		}
		a.logf("Starting %s through Steam", name)
		if err := win.StartDetached(a.env.SteamPath+`\steam.exe`, strings.TrimSpace(fmt.Sprintf("-applaunch %d %s", appID, args)), a.env.SteamPath); err != nil {
			return err
		}
		a.followFound(key, f)
		return nil
	}
	if f.URI != "" && core.SafeLaunchURI(f.URI) {
		a.logf("Starting %s through its launcher", name)
		if err := win.ShellOpen(f.URI); err != nil {
			return err
		}
		a.followFound(key, f)
		return nil
	}
	path, err := a.env.ResolveExe(exe)
	if err != nil {
		return err
	}
	a.logf("Starting %s", name)
	if err := a.startExe(key, path, args); err != nil {
		return err
	}
	a.watchFirstStart("found:" + key)
	a.followFound(key, f)
	return nil
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
	key := revealSecret(a.cfg.Settings.GridDBKey)
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
