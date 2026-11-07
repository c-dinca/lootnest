package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeStore(t *testing.T, search string, images map[string]bool) (Fetch, *[]string) {
	var calls []string
	jpeg := append([]byte{0xFF, 0xD8}, make([]byte, 2048)...)
	return func(u string) ([]byte, error) {
		calls = append(calls, u)
		switch {
		case strings.HasPrefix(u, storeSearchURL):
			return []byte(search), nil
		case images[u]:
			return jpeg, nil
		case strings.Contains(u, "steamstatic"):
			return []byte("<html>not found</html>"), nil
		}
		return nil, errors.New("offline")
	}, &calls
}

func TestFindStoreApp(t *testing.T) {
	fetch, _ := fakeStore(t, `{"items":[{"id":3240220,"name":"Grand Theft Auto V Enhanced"},{"id":271590,"name":"Grand Theft Auto V Legacy"}]}`, nil)
	if id, err := FindStoreApp(fetch, "Grand Theft Auto V"); err != nil || id != 3240220 {
		t.Fatalf("contained match: %d %v", id, err)
	}
	fetch, _ = fakeStore(t, `{"items":[{"id":1,"name":"It Takes Two Soundtrack"},{"id":1426210,"name":"It Takes Two"}]}`, nil)
	if id, _ := FindStoreApp(fetch, "ItTakesTwo"); id != 1426210 {
		t.Fatalf("exact match should win, got %d", id)
	}
	fetch, _ = fakeStore(t, `{"items":[{"id":570,"name":"Dota 2"}]}`, nil)
	if id, _ := FindStoreApp(fetch, "Dota"); id != 0 {
		t.Fatalf("short names must match exactly, got %d", id)
	}
}

func TestDownloadedCover(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "covers")
	img := "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/1426210/library_600x900.jpg"
	fetch, calls := fakeStore(t, `{"items":[{"id":1426210,"name":"It Takes Two"}]}`, map[string]bool{img: true})
	p, err := DownloadedCover(fetch, dir, "It Takes Two", 0)
	if err != nil || p != filepath.Join(dir, "ittakestwo.jpg") {
		t.Fatalf("first download: %q %v", p, err)
	}
	n := len(*calls)
	if p2, _ := DownloadedCover(fetch, dir, "It Takes Two", 0); p2 != p || len(*calls) != n {
		t.Fatal("a saved cover must not be fetched again")
	}

	// Unknown to the store: remembered, not asked again.
	fetch, calls = fakeStore(t, `{"items":[]}`, nil)
	if p, _ := DownloadedCover(fetch, dir, "Memoriez", 0); p != "" {
		t.Fatalf("unknown game got %q", p)
	}
	n = len(*calls)
	if _, err := DownloadedCover(fetch, dir, "Memoriez", 0); err != nil || len(*calls) != n {
		t.Fatal("a miss must be remembered")
	}

	// Offline: an error, and nothing remembered, so it's tried again later.
	offline := func(string) ([]byte, error) { return nil, errors.New("offline") }
	if _, err := DownloadedCover(offline, dir, "Kingdom Two Crowns", 0); err == nil {
		t.Fatal("offline should report an error")
	}
	if _, err := os.Stat(filepath.Join(dir, "kingdomtwocrowns.none")); err == nil {
		t.Fatal("an offline failure must not be remembered as a miss")
	}

	// By Steam app id (a Steam game whose cover isn't in Steam's cache).
	img = "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/730/library_600x900.jpg"
	fetch, _ = fakeStore(t, `{}`, map[string]bool{img: true})
	if p, err := DownloadedCover(fetch, dir, "Counter-Strike 2", 730); err != nil || p != filepath.Join(dir, "app730.jpg") {
		t.Fatalf("by app id: %q %v", p, err)
	}
}

func TestFindStoreAppLoose(t *testing.T) {
	var terms []string
	fetch := func(u string) ([]byte, error) {
		term := strings.TrimPrefix(u, storeSearchURL)
		terms = append(terms, term)
		if term == "Phantom+Blade+Zero" {
			return []byte(`{"items":[{"id":2,"name":"Phantom Blade Zero"}]}`), nil
		}
		return []byte(`{"items":[]}`), nil
	}
	if id, err := findStoreAppLoose(fetch, "Phantom Blade Zero Extra Words"); err != nil || id != 2 {
		t.Fatalf("the name without its last words should be found: %d %v (%v)", id, err, terms)
	}
	terms = nil
	if id, _ := findStoreAppLoose(fetch, "Unknown Game"); id != 0 || len(terms) != 1 {
		t.Fatalf("a two-word name is not shortened to one word: %d %v", id, terms)
	}
}

func pngBytes() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 2048)...)
}

func TestDownloadedArtHeroAndLogo(t *testing.T) {
	dir := t.TempDir()
	hero := "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/1426210/library_hero.jpg"
	fetch, _ := fakeStore(t, `{"items":[{"id":1426210,"name":"It Takes Two"}]}`, map[string]bool{hero: true})
	p, err := DownloadedArt(fetch, nil, dir, "It Takes Two", 0, ArtHero)
	if err != nil || p != filepath.Join(dir, "ittakestwo-hero.jpg") {
		t.Fatalf("hero for a game outside Steam, by its store id: %q %v", p, err)
	}
	logo := "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/730/logo.png"
	png := pngBytes()
	f2 := func(u string) ([]byte, error) {
		if u == logo {
			return png, nil
		}
		return fetch(u)
	}
	if p, err := DownloadedArt(f2, nil, dir, "Counter-Strike 2", 730, ArtLogo); err != nil || p != filepath.Join(dir, "app730-logo.png") {
		t.Fatalf("logo by app id, saved as png: %q %v", p, err)
	}
	if SavedArt(dir, "Counter-Strike 2", 730, ArtLogo) == "" {
		t.Fatal("saved art is found again offline")
	}
}

func TestDownloadedArtFromSteamGridDB(t *testing.T) {
	dir := t.TempDir()
	store, _ := fakeStore(t, `{"items":[]}`, nil) // the store doesn't know the game
	cdn := "https://cdn2.steamgriddb.com/grid/abc.png"
	png := pngBytes()
	fetch := func(u string) ([]byte, error) {
		if u == cdn {
			return png, nil
		}
		return store(u)
	}
	var asked []string
	grid := &GridDB{Key: "k", Fetch: func(u, bearer string) ([]byte, error) {
		if bearer != "k" {
			t.Fatalf("the key goes with every request: %q", bearer)
		}
		asked = append(asked, u)
		switch {
		case strings.HasPrefix(u, gridAPI+"/search/autocomplete/"):
			return []byte(`{"success":true,"data":[{"id":42,"name":"Kingdom Two Crowns"}]}`), nil
		case strings.HasPrefix(u, gridAPI+"/grids/game/42"):
			return []byte(`{"success":true,"data":[{"url":"` + cdn + `"},{"url":"https://evil.example/x.png"}]}`), nil
		}
		return nil, nil
	}}

	// Without a key: a miss, remembered.
	if p, _ := DownloadedArt(fetch, nil, dir, "Kingdom Two Crowns", 0, ArtCover); p != "" {
		t.Fatalf("no key, no cover: %q", p)
	}
	// With a key, the miss is asked again, now from SteamGridDB.
	p, err := DownloadedArt(fetch, grid, dir, "Kingdom Two Crowns", 0, ArtCover)
	if err != nil || p != filepath.Join(dir, "kingdomtwocrowns.png") {
		t.Fatalf("cover from SteamGridDB: %q %v (%v)", p, err, asked)
	}
}
