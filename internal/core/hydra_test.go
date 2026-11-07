package core

import (
	"encoding/json"
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

	got := HydraGames(src, filepath.Join(root, "tmp"), []string{games})
	if len(got) != 1 {
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
	if HydraGames(filepath.Join(root, "none"), filepath.Join(root, "tmp"), nil) != nil {
		t.Fatal("no Hydra, no games")
	}
}
