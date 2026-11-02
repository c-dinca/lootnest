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
