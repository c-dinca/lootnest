package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSavePointsMakeRestoreBranch(t *testing.T) {
	f := newFixture(t)
	g := f.cfg.Game("cs2")
	write(t, f.optFile(), "before the boss")
	write(t, f.cfgFile(), "video=high")
	locs, err := f.eng.SaveLocs(g)
	if err != nil || len(locs) != 2 {
		t.Fatalf("locs %v %v", locs, err)
	}
	root := f.eng.Store.SavePointsDir("game:cs2")
	now := time.Date(2026, 11, 7, 20, 0, 0, 0, time.Local)
	p, err := MakeSavePoint(root, locs, SavePoint{Name: "  Before\nthe boss  ", Player: "p1"}, now, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Before the boss" || len(p.Locs) != 2 || p.Size == 0 {
		t.Fatalf("point %+v", p)
	}
	// Same second: a second point gets its own id.
	p2, err := MakeSavePoint(root, locs, SavePoint{}, now, 1, 0)
	if err != nil || p2.ID == p.ID || p2.Name == "" {
		t.Fatalf("second %+v %v", p2, err)
	}
	list := SavePoints(root)
	if len(list) != 2 {
		t.Fatalf("list %d", len(list))
	}

	write(t, f.optFile(), "after the boss, lost")
	if err := RestoreSavePoint(root, p.ID, locs); err != nil {
		t.Fatal(err)
	}
	if read(t, f.optFile()) != "before the boss" {
		t.Fatal("restore")
	}
	if err := RenameSavePoint(root, p.ID, "Gate"); err != nil || SavePoints(root)[len(list)-1].Name == "" {
		t.Fatal(err)
	}

	// A new run from the point.
	if err := f.eng.ProfileFromSavePoint(root, p.ID, g, "p9"); err != nil {
		t.Fatal(err)
	}
	if !f.eng.HasProfile("p9", "cs2") {
		t.Fatal("profile made")
	}
	if b, _ := os.ReadFile(filepath.Join(f.eng.Store.ProfileDir("p9", "cs2"), "opt")); string(b) != "before the boss" {
		t.Fatalf("profile files %q", b)
	}
	if err := f.eng.ProfileFromSavePoint(root, p.ID, g, "p9"); err == nil {
		t.Fatal("must not overwrite a profile")
	}

	if err := DeleteSavePoint(root, p2.ID); err != nil || len(SavePoints(root)) != 1 {
		t.Fatal("delete")
	}
	for _, bad := range []string{"", "..", "../x", `a\b`, ".hidden"} {
		if err := DeleteSavePoint(root, bad); err == nil {
			t.Fatalf("%q must be refused", bad)
		}
	}
}

func TestSavePointNothingToKeep(t *testing.T) {
	root := t.TempDir()
	_, err := MakeSavePoint(filepath.Join(root, "pts"), []SaveLoc{{ID: "a", Live: filepath.Join(root, "nothing")}}, SavePoint{}, time.Now(), 1, 0)
	if err == nil {
		t.Fatal("no saves: no point")
	}
	if len(SavePoints(filepath.Join(root, "pts"))) != 0 {
		t.Fatal("nothing left behind")
	}
}

// A branch that can't copy a point's files leaves no profile folder.
func TestSavePointBranchFails(t *testing.T) {
	f := newFixture(t)
	g := f.cfg.Game("cs2")
	write(t, f.optFile(), "before the boss")
	locs, err := f.eng.SaveLocs(g)
	if err != nil {
		t.Fatal(err)
	}
	root := f.eng.Store.SavePointsDir("game:cs2")
	p, err := MakeSavePoint(root, locs, SavePoint{Player: "p1"}, time.Now(), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(root, p.ID, "files", "opt")
	if err := os.RemoveAll(bad); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(bad, bad); err != nil {
		t.Skip("no symlinks here:", err)
	}
	if err := f.eng.ProfileFromSavePoint(root, p.ID, g, "p9"); err == nil {
		t.Fatal("a point whose files can't be read made a run")
	}
	if _, err := os.Stat(f.eng.Store.ProfileDir("p9", "cs2")); !os.IsNotExist(err) {
		t.Fatalf("the run's folder was left behind: %v", err)
	}
}
