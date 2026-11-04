package core

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func sha1hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// A game whose saves Steam Auto-Cloud syncs from %APPDATA%\Forest, plus a
// remote-folder file Lootnest doesn't swap.
func cloudFixture(t *testing.T) (*fixture, string, string) {
	f := newFixture(t)
	f.cfg.Games = []Game{{ID: "forest", Name: "Sons Of The Forest", ProcessName: "SonsOfTheForest",
		Launch: Launch{Type: "steam", AppID: 1326470}, Paths: []PathEntry{{ID: "saves", Path: "%APPDATA%/Forest"}}}}
	user := filepath.Join(f.steam, "userdata", "12345")
	slot := filepath.Join(f.appdata, "Forest", "Saves", "Slot1.sav")
	write(t, slot, "mihai's save")
	write(t, filepath.Join(user, "1326470", "remotecache.vdf"), `"1326470"
{
	"ChangeNumber"		"42"
	"Saves/Slot1.sav"
	{
		"root"		"4"
		"size"		"14"
		"sha"		"`+sha1hex("cristian's save")+`"
	}
	"achievements.dat"
	{
		"root"		"0"
		"sha"		"`+sha1hex("x")+`"
	}
}
`)
	return f, user, slot
}

func TestCloudMatches(t *testing.T) {
	f, user, slot := cloudFixture(t)
	files := CloudFiles(user, 1326470)
	if len(files) != 2 {
		t.Fatalf("want 2 synced files, got %+v", files)
	}
	roots, err := f.eng.LivePaths(f.cfg.Game("forest"))
	if err != nil {
		t.Fatal(err)
	}
	ms := CloudMatches(files, user, 1326470, roots)
	if len(ms) != 1 || ms[0].Local != slot {
		t.Fatalf("only the swapped save should match, got %+v", ms)
	}
	if CloudFiles(user, 730) != nil {
		t.Fatal("a game without remotecache.vdf has no synced files")
	}
}

func TestCloudReplaced(t *testing.T) {
	f, user, slot := cloudFixture(t)
	roots, _ := f.eng.LivePaths(f.cfg.Game("forest"))
	ms := CloudMatches(CloudFiles(user, 1326470), user, 1326470, roots)
	loaded := HashMatches(ms) // Mihai's files were just loaded

	// The game changed the save itself: not Steam Cloud.
	write(t, slot, "mihai's save, later")
	if r := CloudReplaced(ms, loaded); len(r) != 0 {
		t.Fatalf("a game's own write isn't a cloud replace: %v", r)
	}
	// Steam downloaded the cloud copy (Cristian's) over it.
	write(t, slot, "cristian's save")
	if r := CloudReplaced(ms, loaded); len(r) != 1 || r[0] != slot {
		t.Fatalf("cloud replace not seen: %v", r)
	}
	// Loading the cloud version itself is never a replace.
	if r := CloudReplaced(ms, HashMatches(ms)); len(r) != 0 {
		t.Fatalf("loaded == cloud must not be reported: %v", r)
	}
}

func TestRevertSession(t *testing.T) {
	f, _, slot := cloudFixture(t)
	if err := f.eng.SaveLiveTo(f.cfg.Game("forest"), "p2"); err != nil { // Mihai's profile
		t.Fatal(err)
	}
	_ = f.eng.Store.SetActive("forest", "p2")
	write(t, slot, "cristian's save") // Steam Cloud replaced it during the session
	dir, err := f.eng.RevertSession(f.cfg, "forest", "p2", 10)
	if err != nil || dir == "" {
		t.Fatalf("revert: %q %v", dir, err)
	}
	if b, _ := os.ReadFile(slot); string(b) != "mihai's save" {
		t.Fatalf("Mihai's save should be back, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "saves", "Saves", "Slot1.sav")); string(b) != "cristian's save" {
		t.Fatalf("the replaced files should be in the backup, got %q", b)
	}
	if f.eng.Store.LoadState()["forest"] != "p2" {
		t.Fatal("Mihai's files are still the loaded ones")
	}
	if p, _ := os.ReadFile(filepath.Join(f.eng.Store.ProfileDir("p2", "forest"), "saves", "Saves", "Slot1.sav")); string(p) != "mihai's save" {
		t.Fatalf("Mihai's profile must be untouched, got %q", p)
	}
}

func TestSplitSteamAppFolder(t *testing.T) {
	f := newFixture(t)
	user := filepath.Join(f.steam, "userdata", "12345")
	app := filepath.Join(user, "381210")
	write(t, filepath.Join(app, "remotecache.vdf"), "steam's records")
	write(t, filepath.Join(app, "remote", "save.dat"), "live save")
	write(t, filepath.Join(app, "local", "cfg.ini"), "live cfg")
	f.cfg.Games = []Game{{ID: "dbd", Name: "Dead by Daylight", ProcessName: "dbd", Launch: Launch{Type: "steam", AppID: 381210},
		Paths: []PathEntry{{ID: "s", Path: "{STEAMUSER}/381210"}, {ID: "c", Path: "%APPDATA%/DBD"}}}}
	// Mihai's profile, saved by an older version: the whole folder.
	prof := f.eng.Store.ProfileDir("p2", "dbd")
	write(t, filepath.Join(prof, "s", "remotecache.vdf"), "old records")
	write(t, filepath.Join(prof, "s", "remote", "save.dat"), "mihai's save")
	write(t, filepath.Join(prof, "s", "local", "cfg.ini"), "mihai's cfg")

	g := f.cfg.Game("dbd")
	changed, err := f.eng.SplitSteamAppFolder(f.cfg, g)
	if err != nil || !changed {
		t.Fatalf("split: %v %v", changed, err)
	}
	got := map[string]string{}
	for _, p := range g.Paths {
		got[p.ID] = p.Path
	}
	if len(got) != 3 || got["s-remote"] != "{STEAMUSER}/381210/remote" || got["s-local"] != "{STEAMUSER}/381210/local" || got["c"] == "" {
		t.Fatalf("paths: %+v", g.Paths)
	}
	if err := f.eng.CheckPaths(g); err != nil {
		t.Fatalf("the new paths must be valid: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(prof, "s-remote", "save.dat")); string(b) != "mihai's save" {
		t.Fatalf("Mihai's save must move with the entry, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(prof, "s-local", "cfg.ini")); string(b) != "mihai's cfg" {
		t.Fatalf("Mihai's cfg must move with the entry, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(app, "remote", "save.dat")); string(b) != "live save" {
		t.Fatal("the live files must not change")
	}
	if again, _ := f.eng.SplitSteamAppFolder(f.cfg, g); again {
		t.Fatal("a second run must change nothing")
	}
}
