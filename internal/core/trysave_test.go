package core

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// p1's files loaded, p2's in their profile.
func trySaveFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	write(t, f.cfgFile(), "p1 video")
	write(t, f.optFile(), "sens=1.2")
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{FirstOwner: "p1", NewProfile: NewFresh}); err != nil {
		t.Fatal(err)
	}
	write(t, f.cfgFile(), "p2 video")
	write(t, f.optFile(), "sens=3.0")
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	f.cfg.Players = append(f.cfg.Players,
		Player{ID: "r1", Name: "Player 1 trying", Color: PlayerColors[0], Of: "p1"},
		Player{ID: "r2", Name: "Player 2 trying", Color: PlayerColors[1], Of: "p2"})
	return f
}

func TestTrySaveFolder(t *testing.T) {
	f := trySaveFixture(t)
	src := filepath.Join(f.root, "Downloads", "friend")
	write(t, filepath.Join(src, "FriendSave", "video.txt"), "friend video")

	if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r1", "p1", "cfg", src); err != nil {
		t.Fatal(err)
	}
	// Nothing loaded changed, and p1's files are whole.
	if read(t, f.cfgFile()) != "p1 video" || read(t, f.optFile()) != "sens=1.2" {
		t.Fatalf("trying a save changed the live files: %q %q", read(t, f.cfgFile()), read(t, f.optFile()))
	}
	// The run plays the friend's save with p1's settings, then p1 comes back.
	if err := f.eng.Activate(f.cfg, "cs2", "r1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "friend video" || read(t, f.optFile()) != "sens=1.2" {
		t.Fatalf("the run's files: %q %q", read(t, f.cfgFile()), read(t, f.optFile()))
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "p1 video" {
		t.Fatalf("p1's save after the run: %q", read(t, f.cfgFile()))
	}
	if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r1", "p1", "cfg", src); err == nil {
		t.Fatal("a profile that has files was started again")
	}
}

// One file into a save folder joins the person's own files, from their
// profile when they aren't loaded.
func TestTrySaveOneFile(t *testing.T) {
	f := trySaveFixture(t)
	src := filepath.Join(f.root, "Downloads", "slot2.sav")
	write(t, src, "friend slot")
	if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r2", "p2", "cfg", src); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(f.eng.Store.ProfileDir("r2", "cs2"), "cfg")
	if read(t, filepath.Join(dir, "video.txt")) != "p2 video" || read(t, filepath.Join(dir, "slot2.sav")) != "friend slot" {
		t.Fatal("the run should have p2's files and the friend's slot")
	}
	if read(t, filepath.Join(f.eng.Store.ProfileDir("r2", "cs2"), "opt")) != "sens=3.0" {
		t.Fatal("the run should have p2's settings")
	}
	if _, err := os.Stat(filepath.Join(f.eng.Store.ProfileDir("p2", "cs2"), "cfg", "slot2.sav")); err == nil {
		t.Fatal("p2's own profile got the friend's slot")
	}
}

func TestTrySaveRefusals(t *testing.T) {
	f := trySaveFixture(t)
	folder := filepath.Join(f.root, "Downloads", "x")
	write(t, filepath.Join(folder, "a.txt"), "x")
	if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r1", "p1", "opt", folder); err == nil {
		t.Fatal("a folder went into a location that is one file")
	}
	if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r1", "p1", "nope", folder); err == nil {
		t.Fatal("an unknown location was accepted")
	}
	if f.eng.HasProfile("r1", "cs2") {
		t.Fatal("a refused try left a profile behind")
	}
}

func TestStageZip(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "save.zip")
	out, _ := os.Create(zp)
	zw := zip.NewWriter(out)
	w, _ := zw.Create("Saves/slot1.sav")
	_, _ = w.Write([]byte("slot one"))
	_ = zw.Close()
	_ = out.Close()
	got, err := StageUpdate(zp, filepath.Join(dir, "staged"))
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(got, "Saves", "slot1.sav")) != "slot one" {
		t.Fatal("the zip wasn't extracted")
	}
	if importRoot(got, filepath.Join(dir, "live")) != filepath.Join(got, "Saves") {
		t.Fatal("importRoot should go into the archive's one folder")
	}
}
