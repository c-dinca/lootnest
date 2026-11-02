//go:build windows

package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Library: every game on the PC, with or without profiles.
//
// Covers come from Steam's own cache on this PC; nothing is downloaded.
// ---------------------------------------------------------------------------

var (
	coverMu    sync.Mutex
	coverCache = map[int]string{}
)

// apiCover returns a Steam game's cached portrait cover as a data: URL, or ""
// when Steam has none for it.
func (a *App) apiCover(appID int) string {
	coverMu.Lock()
	if c, ok := coverCache[appID]; ok {
		coverMu.Unlock()
		return c
	}
	coverMu.Unlock()
	url := ""
	if p := core.SteamCover(a.env.SteamPath, appID); p != "" {
		if b, err := os.ReadFile(p); err == nil && len(b) < 4<<20 {
			mime := "image/jpeg"
			if strings.HasSuffix(strings.ToLower(p), ".png") {
				mime = "image/png"
			}
			url = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
		}
	}
	coverMu.Lock()
	coverCache[appID] = url
	coverMu.Unlock()
	return url
}

// apiLaunchFound starts a game that isn't set up for profiles: Steam games
// through Steam, others by their exe.
func (a *App) apiLaunchFound(key string) error {
	a.mu.Lock()
	var name, exe string
	appID := 0
	found := false
	for _, f := range a.found {
		if f.Key == key {
			name, exe, appID, found = f.Name, f.Exe, f.AppID, true
			break
		}
	}
	a.mu.Unlock()
	if !found {
		return errors.New("That game isn't on the list anymore. Scan again from the Library.")
	}
	if appID > 0 {
		if a.env.SteamPath == "" {
			return errors.New("Steam wasn't found on this PC")
		}
		a.logf("Starting %s through Steam", name)
		return win.StartDetached(a.env.SteamPath+`\steam.exe`, fmt.Sprintf("-applaunch %d", appID), a.env.SteamPath)
	}
	path, err := a.env.ResolveExe(exe)
	if err != nil {
		return err
	}
	a.logf("Starting %s", name)
	return win.StartDetached(path, "", dirOf(path))
}
