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
	key = artKey(name, appID, ArtCover)
	if key == "" {
		return "", ""
	}
	return key, filepath.Join(dir, key+".jpg")
}

// Kinds of game art: the portrait cover, the wide banner shown on a game's
// page and the Library's "last played" hero, and the transparent logo.
const (
	ArtCover = "cover"
	ArtHero  = "hero"
	ArtLogo  = "logo"
)

var steamArtURL = map[string]string{
	ArtCover: coverURL,
	ArtHero:  "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/%d/library_hero.jpg",
	ArtLogo:  "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/%d/logo.png",
}

func artKey(name string, appID int, kind string) string {
	key := CoverKey(name)
	if appID > 0 {
		key = fmt.Sprintf("app%d", appID)
	}
	if key == "" {
		return ""
	}
	if kind != ArtCover {
		key += "-" + kind
	}
	return key
}

// savedArt finds art downloaded earlier, as .jpg or .png.
func savedArt(dir, key string) string {
	for _, ext := range []string{".jpg", ".png"} {
		if p := filepath.Join(dir, key+ext); exists(p) {
			return p
		}
	}
	return ""
}

// SavedCover returns a cover downloaded earlier, without any network access.
func SavedCover(dir, name string, appID int) string { return SavedArt(dir, name, appID, ArtCover) }

// SavedArt returns art downloaded earlier, without any network access.
func SavedArt(dir, name string, appID int, kind string) string {
	if key := artKey(name, appID, kind); key != "" {
		return savedArt(dir, key)
	}
	return ""
}

// imageExt says whether b is a usable image for kind and how to save it:
// covers and heroes are JPEG or PNG, logos PNG (they need transparency).
func imageExt(b []byte, kind string) string {
	if len(b) < 1024 {
		return ""
	}
	isPNG := len(b) > 8 && string(b[1:4]) == "PNG"
	isJPEG := b[0] == 0xFF && b[1] == 0xD8
	switch {
	case isPNG:
		return ".png"
	case isJPEG && kind != ArtLogo:
		return ".jpg"
	}
	return ""
}

// DownloadedCover is DownloadedArt for a cover, from the Steam store only.
func DownloadedCover(fetch Fetch, dir, name string, appID int) (string, error) {
	return DownloadedArt(fetch, nil, dir, name, appID, ArtCover)
}

// DownloadedArt returns the path of a game's saved art, downloading it first
// when needed: from the Steam store by app id (for a game outside Steam, the
// id the store gives for its name), else from SteamGridDB when the user gave
// a key. "" means there is none (remembered for a while, so nobody is asked
// again on every start; adding a SteamGridDB key asks again).
func DownloadedArt(fetch Fetch, grid *GridDB, dir, name string, appID int, kind string) (string, error) {
	key := artKey(name, appID, kind)
	if key == "" || steamArtURL[kind] == "" {
		return "", nil
	}
	if p := savedArt(dir, key); p != "" {
		return p, nil
	}
	miss := filepath.Join(dir, key+".none")
	if st, err := os.Stat(miss); err == nil && time.Since(st.ModTime()) < coverMissRetry {
		if b, _ := os.ReadFile(miss); string(b) == "grid" || !grid.on() {
			return "", nil
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	id := appID
	if id <= 0 {
		var err error
		if id, err = StoreAppID(fetch, dir, name); err != nil {
			return "", err // offline or the store failed: try again next time
		}
	}
	var img []byte
	ext := ""
	if id > 0 {
		b, err := fetch(fmt.Sprintf(steamArtURL[kind], id))
		if err != nil {
			return "", err
		}
		if ext = imageExt(b, kind); ext != "" {
			img = b
		}
	}
	if img == nil && grid.on() {
		u, err := grid.artURL(kind, name, id)
		if err != nil {
			return "", err
		}
		if u != "" {
			b, err := fetch(u)
			if err != nil {
				return "", err
			}
			if ext = imageExt(b, kind); ext != "" {
				img = b
			}
		}
	}
	if img == nil {
		mark := ""
		if grid.on() {
			mark = "grid"
		}
		_ = os.WriteFile(miss, []byte(mark), 0o644)
		return "", nil
	}
	path := filepath.Join(dir, key+ext)
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

// ---------------------------------------------------------------------------
// SteamGridDB: covers, heroes and logos for nearly every game, Steam or not.
// Only used with the user's own API key (Settings), which goes to
// steamgriddb.com with each request; the images come from its CDN.
// ---------------------------------------------------------------------------

const gridAPI = "https://www.steamgriddb.com/api/v2"

// FetchAuth gets a URL with an "Authorization: Bearer" header.
type FetchAuth func(url, bearer string) ([]byte, error)

type GridDB struct {
	Key   string
	Fetch FetchAuth
}

func (g *GridDB) on() bool { return g != nil && strings.TrimSpace(g.Key) != "" && g.Fetch != nil }

func (g *GridDB) get(path string, v any) error {
	b, err := g.Fetch(gridAPI+path, strings.TrimSpace(g.Key))
	if err != nil {
		return err
	}
	if b == nil {
		return nil // 404: nothing there
	}
	return json.Unmarshal(b, v)
}

// gameID finds a game on SteamGridDB by name, matched like FindStoreApp.
func (g *GridDB) gameID(name string) (int, error) {
	key := norm(name)
	if key == "" {
		return 0, nil
	}
	var res struct {
		Data []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := g.get("/search/autocomplete/"+url.PathEscape(name), &res); err != nil {
		return 0, err
	}
	for _, d := range res.Data {
		if norm(d.Name) == key {
			return d.ID, nil
		}
	}
	for _, d := range res.Data {
		n := norm(d.Name)
		if len(n) >= 5 && len(key) >= 5 && (strings.Contains(n, key) || strings.Contains(key, n)) {
			return d.ID, nil
		}
	}
	return 0, nil
}

// artURL is the address of a game's best static image of a kind on
// SteamGridDB: by Steam app id when known, else by name.
func (g *GridDB) artURL(kind, name string, steamID int) (string, error) {
	typ := map[string]string{ArtCover: "grids", ArtHero: "heroes", ArtLogo: "logos"}[kind]
	query := "?types=static"
	if kind == ArtCover {
		query += "&dimensions=600x900,342x482,660x930"
	}
	var path string
	if steamID > 0 {
		path = fmt.Sprintf("/%s/steam/%d%s", typ, steamID, query)
	} else {
		id, err := g.gameID(name)
		if err != nil || id == 0 {
			return "", err
		}
		path = fmt.Sprintf("/%s/game/%d%s", typ, id, query)
	}
	var res struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := g.get(path, &res); err != nil {
		return "", err
	}
	for _, d := range res.Data {
		if p, err := url.Parse(d.URL); err == nil && p.Scheme == "https" && strings.HasSuffix(strings.ToLower(p.Hostname()), "steamgriddb.com") {
			return d.URL, nil
		}
	}
	return "", nil
}
