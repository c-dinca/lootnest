package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Downloaded covers (opt-in: Settings > "Download covers for games outside
// Steam"). Off by default; with it off, nothing here is ever called.
//
// A game is looked up on the Steam store by name (the only thing sent), and
// its portrait cover is saved under <data>\covers so it's fetched once.
// ---------------------------------------------------------------------------

// Fetch gets a URL's body. The app passes a real HTTP client; tests a fake.
type Fetch func(url string) ([]byte, error)

const (
	storeSearchURL = "https://store.steampowered.com/api/storesearch/?l=english&cc=us&term="
	coverURL       = "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/%d/library_600x900.jpg"
	// A game the store doesn't know isn't looked up again for this long.
	coverMissRetry = 7 * 24 * time.Hour
)

// CoverKey is the file name a game's downloaded cover is saved under.
func CoverKey(name string) string { return norm(name) }

// FindStoreApp looks a game name up on the Steam store and returns the app
// id of the best match: an exact match of the name (ignoring case, spaces and
// punctuation), else a result whose name contains it or is contained in it
// (both at least 5 letters, so "Dota" doesn't pick "Dota 2").
func FindStoreApp(fetch Fetch, name string) (int, error) {
	key := norm(name)
	if key == "" {
		return 0, errors.New("no name to look up")
	}
	b, err := fetch(storeSearchURL + url.QueryEscape(name))
	if err != nil {
		return 0, err
	}
	var res struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return 0, fmt.Errorf("unexpected store reply: %w", err)
	}
	for _, it := range res.Items {
		if norm(it.Name) == key {
			return it.ID, nil
		}
	}
	for _, it := range res.Items {
		n := norm(it.Name)
		if len(n) >= 5 && len(key) >= 5 && (strings.Contains(n, key) || strings.Contains(key, n)) {
			return it.ID, nil
		}
	}
	return 0, nil
}

// findStoreAppLoose looks the name up, then, if the store doesn't know it,
// the name without its last word or two ("Game Name Deluxe Edition",
// "Game Name Tag"), never shorter than two words.
func findStoreAppLoose(fetch Fetch, name string) (int, error) {
	words := strings.Fields(name)
	for n, tries := len(words), 0; n >= 1 && tries < 3; n, tries = n-1, tries+1 {
		if tries > 0 && n < 2 {
			break
		}
		id, err := FindStoreApp(fetch, strings.Join(words[:n], " "))
		if err != nil || id > 0 {
			return id, err
		}
	}
	return 0, nil
}

func coverFile(dir, name string, appID int) (key, path string) {
	key = CoverKey(name)
	if appID > 0 {
		key = fmt.Sprintf("app%d", appID)
	}
	if key == "" {
		return "", ""
	}
	return key, filepath.Join(dir, key+".jpg")
}

// SavedCover returns a cover downloaded earlier, without any network access.
func SavedCover(dir, name string, appID int) string {
	if _, p := coverFile(dir, name, appID); p != "" && exists(p) {
		return p
	}
	return ""
}

// DownloadedCover returns the path of a game's saved cover, downloading it
// first when needed: by Steam app id when known, else by looking the name up
// on the store. "" means there is no cover (remembered for a while, so the
// store isn't asked again on every start).
func DownloadedCover(fetch Fetch, dir, name string, appID int) (string, error) {
	key, path := coverFile(dir, name, appID)
	if key == "" {
		return "", nil
	}
	miss := filepath.Join(dir, key+".none")
	if exists(path) {
		return path, nil
	}
	if st, err := os.Stat(miss); err == nil && time.Since(st.ModTime()) < coverMissRetry {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if appID <= 0 {
		id, err := findStoreAppLoose(fetch, name)
		if err != nil {
			return "", err // offline or the store failed: try again next time
		}
		appID = id
	}
	if appID <= 0 {
		_ = os.WriteFile(miss, nil, 0o644)
		return "", nil
	}
	img, err := fetch(fmt.Sprintf(coverURL, appID))
	if err != nil {
		return "", err
	}
	if len(img) < 1024 || !(img[0] == 0xFF && img[1] == 0xD8) { // not a JPEG: the store has no portrait cover
		_ = os.WriteFile(miss, nil, 0o644)
		return "", nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, img, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}
