package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestHydraGames(t *testing.T) {
	root := t.TempDir()
	games := filepath.Join(root, "Games")
	exe := filepath.Join(games, "Night Tale", "bin", "NightTale.exe")
	write(t, exe, "MZ")
	write(t, filepath.Join(games, "Gone", "Gone.exe"), "MZ")
	src := filepath.Join(root, "hydra-db")
	db, err := leveldb.OpenFile(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	put := func(k string, v map[string]any) {
		b, _ := json.Marshal(v)
		if err := db.Put([]byte(k), b, nil); err != nil {
			t.Fatal(err)
		}
	}
	put("!games!steam:777", map[string]any{"title": "Night Tale", "objectId": "777", "shop": "steam", "executablePath": exe, "isDeleted": false})
	put("!games!steam:778", map[string]any{"title": "Gone", "objectId": "778", "shop": "steam", "executablePath": filepath.Join(games, "Gone", "Gone.exe"), "isDeleted": true})
	put("!games!steam:779", map[string]any{"title": "Not installed", "objectId": "779", "shop": "steam", "executablePath": nil, "isDeleted": false})
	put("!games!steam:780", map[string]any{"title": "Moved", "objectId": "780", "shop": "steam", "executablePath": filepath.Join(games, "Moved", "x.exe")})
	put("!downloads!steam:777", map[string]any{"title": "a download, not a game"})
	// Hydra is running: its database stays open (and locked) while Lootnest reads.
	defer db.Close()

	got, err := HydraGames(src, filepath.Join(root, "tmp"), []string{games})
	if err != nil || len(got) != 1 {
		t.Fatalf("HydraGames = %+v", got)
	}
	g := got[0]
	if g.Name != "Night Tale" || g.Source != "hydra" || g.SteamID != 777 || g.AppID != 0 ||
		g.Dir != filepath.Join(games, "Night Tale") || g.Exe != exe || g.ProcessName != "NightTale" {
		t.Fatalf("game = %+v", g)
	}
	if isDir(filepath.Join(root, "tmp")) {
		t.Fatal("the copy should be removed")
	}
	if got, err := HydraGames(filepath.Join(root, "none"), filepath.Join(root, "tmp"), nil); got != nil || err != nil {
		t.Fatal("no Hydra, no games")
	}
	_ = os.MkdirAll(filepath.Join(root, "empty-db"), 0o755)
	if _, err := HydraGames(filepath.Join(root, "empty-db"), filepath.Join(root, "tmp"), nil); err == nil {
		t.Fatal("a hydra-db Lootnest can't open should say why")
	}
}

func TestHydraDownloadsPath(t *testing.T) {
	root := t.TempDir()
	if p, err := HydraDownloadsPath(filepath.Join(root, "none"), filepath.Join(root, "tmp")); p != "" || err != nil {
		t.Fatalf("no Hydra: %q %v", p, err)
	}
	src := filepath.Join(root, "hydra-db")
	db, err := leveldb.OpenFile(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := HydraDownloadsPath(src, filepath.Join(root, "tmp")); p != "" {
		t.Fatalf("never set: %q", p)
	}
	dl := filepath.Join(root, "HydraDownloads")
	if err := os.MkdirAll(dl, 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"downloadsPath": dl, "language": "en"})
	if err := db.Put([]byte("userPreferences"), b, nil); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if p, err := HydraDownloadsPath(src, filepath.Join(root, "tmp")); p != dl || err != nil {
		t.Fatalf("set: %q %v", p, err)
	}
}
