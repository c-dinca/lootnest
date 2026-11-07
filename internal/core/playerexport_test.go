package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlayerExportImport(t *testing.T) {
	// PC 1: Ana and Dan, one Steam game; Ana's files are in place.
	data, live, cfg := backupFixture(t)
	src := &Store{Dir: data}
	_ = src.SetShelf("p1", "game:g1", ShelfEntry{Status: "finished", Rating: 5}, time.Now())
	_ = src.AddPlaytime("game:g1", "p1", 2*time.Hour)
	_ = src.AddPlaytime("game:g1", "p2", time.Hour)
	cfg.Players[0].Favorites = []string{"game:g1", "steam:620"}
	_ = src.SaveConfig(cfg)

	for _, enc := range []bool{false, true} {
		out := t.TempDir()
		opt := BackupOptions{}
		pw := ""
		if enc {
			salt := NewBackupSalt()
			key, _ := BackupKey("profile pw", salt, BackupKDFIters)
			opt, pw = BackupOptions{Key: key, Salt: salt, Iters: BackupKDFIters}, "profile pw"
		}
		file := filepath.Join(out, PlayerFileName("Ana", time.Now(), enc))
		liveSrc := []LiveSource{{LiveEntry{GameID: "g1", EntryID: "saves", Owner: "p1"}, live}}
		if _, err := src.WritePlayerFile(file, cfg, "p1", liveSrc, BackupManifest{Computer: "PC1", Created: time.Now()}, opt); err != nil {
			t.Fatal(err)
		}
		ob, err := OpenBackup(file, filepath.Join(out, "open"), pw, nil)
		if err != nil {
			t.Fatal(err)
		}
		if ob.Manifest.Kind != "player" || len(ob.Manifest.Players) != 1 {
			t.Fatalf("%+v", ob.Manifest)
		}
		if _, err := os.Stat(filepath.Join(ob.Dir, "profiles", "p2")); err == nil {
			t.Fatal("Dan's saves aren't in Ana's profile file")
		}
		// A profile file is never restored as a backup of everything.
		if _, err := (&Store{Dir: t.TempDir()}).Restore(ob, DefaultConfig()); err == nil {
			t.Fatal("restoring a player file is refused")
		}

		// PC 2: someone else, and the same Steam game added on its own
		// (other game and path ids), with that player's files loaded.
		dst := &Store{Dir: t.TempDir()}
		here := DefaultConfig()
		here.Players = []Player{{ID: "x1", Name: "Ana", Color: "#999999"}}
		here.Games = []Game{{ID: "local", Name: "Game", ProcessName: "game", Launch: Launch{Type: "steam", AppID: 10},
			Paths: []PathEntry{{ID: "d1", Path: live}}}}
		_ = dst.SaveConfig(here)
		got, res, err := dst.ImportPlayer(ob, here)
		if err != nil {
			t.Fatal(err)
		}
		if res.Replaced || res.Added != 0 || res.Games != 1 || res.Skipped != 0 {
			t.Fatalf("%+v", res)
		}
		if len(got.Players) != 2 || got.Player("p1") == nil || got.Player("p1").Name != "Ana (2)" {
			t.Fatalf("added next to the other Ana: %+v", got.Players)
		}
		if s := read(t, filepath.Join(dst.Dir, "profiles", "p1", "local", "d1", "slot1.sav")); s != "ana's newest save" {
			t.Fatalf("her files go to the game here, under its own path id: %q", s)
		}
		if got.Player("p1").Favorites[0] != "game:local" || got.Player("p1").Favorites[1] != "steam:620" || !got.Player("p1").Avatar {
			t.Fatalf("library keys follow the game here: %+v", got.Player("p1"))
		}
		if e := dst.LoadShelf()["p1"]["game:local"]; e.Status != "finished" {
			t.Fatalf("shelf: %+v", dst.LoadShelf())
		}
		if pt := dst.LoadPlaytime()["game:local"]; pt["p1"] != 7200 || pt["p2"] != 0 {
			t.Fatalf("only her play time: %+v", pt)
		}

		// Importing again updates her (no third Ana), and a game that
		// isn't here is added.
		got2, res2, err := dst.ImportPlayer(ob, got)
		if err != nil || !res2.Replaced || len(got2.Players) != 2 {
			t.Fatalf("%+v %+v %v", got2.Players, res2, err)
		}
		empty := &Store{Dir: t.TempDir()}
		got3, res3, err := empty.ImportPlayer(ob, DefaultConfig())
		if err != nil || res3.Added != 1 || len(got3.Games) != 1 || got3.Games[0].ID != "g1" {
			t.Fatalf("%+v %+v %v", got3, res3, err)
		}
		if !strings.Contains(read(t, filepath.Join(empty.Dir, "profiles", "p1", "g1", "saves", "slot1.sav")), "newest") {
			t.Fatal("saves for an added game")
		}
	}
}
