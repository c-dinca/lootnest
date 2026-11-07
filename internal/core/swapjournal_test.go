package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// twoProfiles sets up cs2 with p1's files in place and p2's in its profile.
func twoProfiles(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	write(t, f.cfgFile(), "p2 cfg")
	write(t, f.optFile(), "p2 opt")
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{FirstOwner: "p2"}); err != nil {
		t.Fatal(err)
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{NewProfile: NewFresh}); err != nil {
		t.Fatal(err)
	}
	write(t, f.cfgFile(), "p1 cfg")
	write(t, f.optFile(), "p1 opt")
	if err := f.eng.FinishSession(f.cfg, "cs2", "p1"); err != nil {
		t.Fatal(err)
	}
	return f
}

// interrupt leaves cs2 as a swap to p2 would if the PC turned off halfway:
// p1's files saved, the journal written, one location already p2's, and
// state.json still naming p1.
func interrupt(t *testing.T, f *fixture) {
	t.Helper()
	g := f.cfg.Game("cs2")
	rs, err := f.eng.resolveAll(g)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.eng.Store.setSwap("cs2", &PendingSwap{To: "p2"}); err != nil {
		t.Fatal(err)
	}
	if err := Mirror(filepath.Join(f.eng.Store.ProfileDir("p2", "cs2"), rs[0].entry.ID), rs[0].live); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "p2 cfg" || read(t, f.optFile()) != "p1 opt" {
		t.Fatal("setup: expected a mix of two profiles in place")
	}
}

func TestInterruptedSwapIsFinished(t *testing.T) {
	f := twoProfiles(t)
	interrupt(t, f)

	// Nothing may save the mix over anyone's profile meanwhile.
	if err := f.eng.SaveLiveTo(f.cfg.Game("cs2"), "p1"); err == nil {
		t.Fatal("saving the files in place should be refused while a swap is unfinished")
	}
	if err := f.eng.FinishSession(f.cfg, "cs2", "p1"); err == nil {
		t.Fatal("finishing a session should be refused while a swap is unfinished")
	}

	done, err := f.eng.RecoverSwaps(f.cfg, 10)
	if err != nil || len(done) != 1 || done[0] != "cs2" {
		t.Fatalf("RecoverSwaps = %v, %v", done, err)
	}
	if read(t, f.cfgFile()) != "p2 cfg" || read(t, f.optFile()) != "p2 opt" {
		t.Fatalf("p2's files should be in place: %q %q", read(t, f.cfgFile()), read(t, f.optFile()))
	}
	if got := f.eng.Store.LoadState()["cs2"]; got != "p2" {
		t.Fatalf("owner = %q, want p2", got)
	}
	if len(f.eng.Store.PendingSwaps()) != 0 {
		t.Fatal("the journal should be empty")
	}
	if _, err := os.Stat(f.eng.Store.SwapsPath()); !os.IsNotExist(err) {
		t.Fatal("an empty journal should leave no file")
	}

	// p1's own files were never touched.
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "p1 cfg" || read(t, f.optFile()) != "p1 opt" {
		t.Fatalf("p1's profile was damaged: %q %q", read(t, f.cfgFile()), read(t, f.optFile()))
	}
}

func TestActivateFinishesAnInterruptedSwapFirst(t *testing.T) {
	f := twoProfiles(t)
	interrupt(t, f)
	// Without a restart in between, the next Play does it first.
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "p1 cfg" || read(t, f.optFile()) != "p1 opt" {
		t.Fatalf("p1's profile was damaged: %q %q", read(t, f.cfgFile()), read(t, f.optFile()))
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "p2 cfg" || read(t, f.optFile()) != "p2 opt" {
		t.Fatalf("p2's profile was damaged: %q %q", read(t, f.cfgFile()), read(t, f.optFile()))
	}
}

func TestInterruptedFreshStart(t *testing.T) {
	f := newFixture(t)
	write(t, f.cfgFile(), "p1 cfg")
	write(t, f.optFile(), "p1 opt")
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{FirstOwner: "p1"}); err != nil {
		t.Fatal(err)
	}
	// p2 starts fresh: p1's saved, the journal written, one location cleared.
	if err := f.eng.SaveLiveTo(f.cfg.Game("cs2"), "p1"); err != nil {
		t.Fatal(err)
	}
	_ = f.eng.Store.setSwap("cs2", &PendingSwap{To: "p2", Fresh: true})
	_ = os.Remove(f.optFile())

	if _, err := f.eng.RecoverSwaps(f.cfg, 10); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "<missing>" || read(t, f.optFile()) != "<missing>" {
		t.Fatal("p2's fresh start should leave no files in place")
	}
	if !f.eng.HasProfile("p2", "cs2") || f.eng.Store.LoadState()["cs2"] != "p2" {
		t.Fatal("p2 should own an empty profile")
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.cfgFile()) != "p1 cfg" || read(t, f.optFile()) != "p1 opt" {
		t.Fatal("p1's profile was damaged")
	}
}

func TestInterruptedSwapToDeletedProfile(t *testing.T) {
	f := twoProfiles(t)
	interrupt(t, f)
	f.eng.DeletePlayerData("p2")
	before := len(f.eng.Backups("cs2"))
	if _, err := f.eng.RecoverSwaps(f.cfg, 10); err != nil {
		t.Fatal(err)
	}
	if len(f.eng.Backups("cs2")) != before+1 {
		t.Fatal("the files in place should be kept as a backup")
	}
	if f.eng.Store.LoadState()["cs2"] != "" {
		t.Fatal("nobody should own the files in place")
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); !errors.Is(err, ErrNeedOwner) {
		t.Fatalf("want ErrNeedOwner, got %v", err)
	}
}

func TestMirrorRefusesADamagedCopy(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, filepath.Join(src, "save.dat"), "new progress")
	write(t, filepath.Join(dst, "save.dat"), "old progress")
	afterCopy = func(p string) { _ = os.WriteFile(p, []byte("new"), 0o644) }
	defer func() { afterCopy = nil }()
	if err := Mirror(src, dst); !errors.Is(err, ErrCopyMismatch) {
		t.Fatalf("want ErrCopyMismatch, got %v", err)
	}
	if read(t, filepath.Join(dst, "save.dat")) != "old progress" {
		t.Fatal("a damaged copy must not replace anything")
	}
	if _, err := os.Stat(dst + tmpSuffix); !os.IsNotExist(err) {
		t.Fatal("the damaged copy should be cleaned up")
	}
}

func TestVanishedSavesAreKept(t *testing.T) {
	f := twoProfiles(t) // p1's files in place
	var warned string
	f.eng.Warn = func(title, _ string) { warned = title }
	before := len(f.eng.Backups("cs2"))

	// The game is reinstalled: its settings file is gone, the folder empty.
	_ = os.Remove(f.optFile())
	_ = os.Remove(f.cfgFile())
	if err := f.eng.FinishSession(f.cfg, "cs2", "p1"); err != nil {
		t.Fatal(err)
	}
	bs := f.eng.Backups("cs2")
	if len(bs) != before+1 || bs[0].Owner != "p1" {
		t.Fatalf("p1's last copy should be kept as p1's backup: %+v", bs)
	}
	if read(t, filepath.Join(bs[0].Dir, "opt")) != "p1 opt" || read(t, filepath.Join(bs[0].Dir, "cfg", "video.txt")) != "p1 cfg" {
		t.Fatal("the kept copy should hold p1's files")
	}
	if warned == "" {
		t.Fatal("the user should be told")
	}

	// Once the profile is empty too, nothing more is kept.
	warned = ""
	if err := f.eng.FinishSession(f.cfg, "cs2", "p1"); err != nil {
		t.Fatal(err)
	}
	if len(f.eng.Backups("cs2")) != before+1 || warned != "" {
		t.Fatal("an empty profile has nothing to keep")
	}
	if !HasFiles(bs[0].Dir) || HasFiles(filepath.Join(f.root, "nope")) {
		t.Fatal("HasFiles")
	}
}
