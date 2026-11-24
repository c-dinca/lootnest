package core

import (
	"archive/zip"
	"errors"
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
	write(t, filepath.Join(folder, "b.txt"), "y")
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

func zipOf(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	for name, body := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = out.Close()
}

// A zip with one file is that file: into a location that is one file, and
// joining the person's files in a folder.
func TestTrySaveArchiveOfOneFile(t *testing.T) {
	f := trySaveFixture(t)
	dir := t.TempDir()
	zipOf(t, filepath.Join(dir, "opt.zip"), map[string]string{"Friend/opt.ini": "sens=9"})
	staged, err := StageUpdate(filepath.Join(dir, "opt.zip"), filepath.Join(dir, "a"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r1", "p1", "opt", staged); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(f.eng.Store.ProfileDir("r1", "cs2"), "opt")) != "sens=9" {
		t.Fatal("the one file should be the location")
	}

	zipOf(t, filepath.Join(dir, "slot.zip"), map[string]string{"slot2.sav": "friend slot"})
	staged, err = StageUpdate(filepath.Join(dir, "slot.zip"), filepath.Join(dir, "b"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r2", "p2", "cfg", staged); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(f.eng.Store.ProfileDir("r2", "cs2"), "cfg")
	if read(t, filepath.Join(cfg, "video.txt")) != "p2 video" || read(t, filepath.Join(cfg, "slot2.sav")) != "friend slot" {
		t.Fatal("the one file should join p2's files, not replace them")
	}
}

// A live folder with slot folders: one folder in the import is a slot. A
// wrapper around what the live folder has is still stepped through.
func TestImportRootSlot(t *testing.T) {
	for _, c := range []struct {
		live, imp []string
		want      string
	}{
		{[]string{"Slot_1/data.sav"}, []string{"Slot_3/data.sav"}, ""},
		{[]string{"76561198000000000/ER0000.sl2"}, []string{"EldenRing/76561198000000000/ER0000.sl2"}, "EldenRing"},
		{[]string{"save1.sav", "backup/old.sav"}, []string{"MySave/save1.sav", "MySave/save2.sav"}, "MySave"},
	} {
		dir := t.TempDir()
		live, src := filepath.Join(dir, "live"), filepath.Join(dir, "import")
		for _, p := range c.live {
			write(t, filepath.Join(live, p), "mine")
		}
		for _, p := range c.imp {
			write(t, filepath.Join(src, p), "friend")
		}
		if got := importRoot(src, live); got != filepath.Join(src, c.want) {
			t.Errorf("live %v, import %v: importRoot = %s", c.live, c.imp, got)
		}
	}
}

func TestStageMax(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "big.zip")
	zipOf(t, zp, map[string]string{"save.dat": "more than ten bytes"})
	out := filepath.Join(dir, "staged")
	if _, err := StageUpdateMax(zp, out, 10); !errors.Is(err, ErrTooBig) {
		t.Fatalf("err %v, want ErrTooBig", err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("a refused archive left files behind")
	}
	if _, err := StageUpdateMax(zp, out, 1<<20); err != nil {
		t.Fatal(err)
	}
}

// A save brought for Red Dead Redemption 2 (SRDR30000, in Profiles\<id>):
// the run holds it and none of the person's other slots, so the game's
// Continue can only load it; the person's own files keep every slot.
func TestTrySaveSlotsLikeRDR2(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string // brought, relative to the download
	}{
		{"one file", map[string]string{"SRDR30000": "skip prologue"}},
		{"wrapped files", map[string]string{"Prologue Skip/SRDR30000": "skip prologue", "Prologue Skip/SRDR30001": "skip prologue b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := trySaveFixture(t)
			live := filepath.Dir(f.cfgFile())
			prof := filepath.Join(live, "A1B2C3D4")
			write(t, filepath.Join(prof, "SRDR30000"), "p1 chapter 2")
			write(t, filepath.Join(prof, "SRDR30003"), "p1 chapter 4")
			write(t, filepath.Join(prof, "SRDR30014"), "p1 autosave")
			dl := filepath.Join(f.root, "Downloads", "mod")
			for rel, body := range tc.files {
				write(t, filepath.Join(dl, filepath.FromSlash(rel)), body)
			}
			src := dl
			if len(tc.files) == 1 {
				src = filepath.Join(dl, "SRDR30000")
			}
			if err := f.eng.ProfileFromImport(f.cfg.Game("cs2"), "r1", "p1", "cfg", src); err != nil {
				t.Fatal(err)
			}
			run := filepath.Join(f.eng.Store.ProfileDir("r1", "cs2"), "cfg", "A1B2C3D4")
			if read(t, filepath.Join(run, "SRDR30000")) != "skip prologue" {
				t.Fatal("the run doesn't hold the brought save")
			}
			for _, n := range []string{"SRDR30003", "SRDR30014"} {
				if _, err := os.Stat(filepath.Join(run, n)); err == nil {
					t.Errorf("the run kept p1's %s, which the game would continue from", n)
				}
			}
			if len(tc.files) == 2 && read(t, filepath.Join(run, "SRDR30001")) != "skip prologue b" {
				t.Error("the second brought save isn't beside the first")
			}
			if _, err := os.Stat(filepath.Join(f.eng.Store.ProfileDir("r1", "cs2"), "cfg", "SRDR30000")); err == nil {
				t.Error("a brought save landed at the folder's top, where the game doesn't look")
			}
			if read(t, filepath.Join(run, "..", "video.txt")) != "p1 video" {
				t.Error("the run lost p1's other files")
			}
			// p1's own files keep every slot.
			if read(t, filepath.Join(prof, "SRDR30003")) != "p1 chapter 4" || read(t, filepath.Join(prof, "SRDR30000")) != "p1 chapter 2" {
				t.Error("p1's saves changed")
			}
		})
	}
}
