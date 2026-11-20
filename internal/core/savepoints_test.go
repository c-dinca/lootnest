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

// Going back makes the point's saves the newest, keeping their order, so a
// game's Continue and Steam Cloud take them.
func TestRestoreMakesSavesNewest(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "saves")
	write(t, filepath.Join(live, "slot1.sav"), "old slot")
	write(t, filepath.Join(live, "sub", "slot2.sav"), "the boss")
	t1 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(90 * time.Minute)
	_ = os.Chtimes(filepath.Join(live, "slot1.sav"), t1, t1)
	_ = os.Chtimes(filepath.Join(live, "sub", "slot2.sav"), t2, t2)
	locs := []SaveLoc{{ID: "a", Live: live}}
	root := filepath.Join(dir, "pts")
	p, err := MakeSavePoint(root, locs, SavePoint{Name: "Boss"}, t2, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Played on: a newer save, then going back.
	write(t, filepath.Join(live, "sub", "slot2.sav"), "after the boss")
	write(t, filepath.Join(live, "slot3.sav"), "later")
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	if err := RestoreSavePointAt(root, p.ID, locs, now); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(live, "sub", "slot2.sav")) != "the boss" || exists(filepath.Join(live, "slot3.sav")) {
		t.Fatal("the point's saves, and only those")
	}
	st1, _ := os.Stat(filepath.Join(live, "slot1.sav"))
	st2, _ := os.Stat(filepath.Join(live, "sub", "slot2.sav"))
	if !st2.ModTime().Equal(now) {
		t.Fatalf("newest save is %v, want now", st2.ModTime())
	}
	if d := st2.ModTime().Sub(st1.ModTime()); d != 90*time.Minute {
		t.Fatalf("order kept: %v apart", d)
	}
	// The point itself keeps its own times.
	in, _ := os.Stat(filepath.Join(root, p.ID, "files", "a", "sub", "slot2.sav"))
	if !in.ModTime().Equal(t2) {
		t.Fatalf("point's copy changed: %v", in.ModTime())
	}
	// A single-file location, and nothing newer than now is moved back.
	f := filepath.Join(dir, "one.sav")
	write(t, f, "x")
	future := now.Add(time.Hour)
	_ = os.Chtimes(f, future, future)
	if err := MakeNewest([]string{f, filepath.Join(dir, "gone")}, now); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(f); !st.ModTime().Equal(future) {
		t.Fatal("a newer file must stay")
	}
}

// Only the oldest of the points kept when the game saved go; the user's and
// Lootnest's other points stay, and a renamed game point becomes the user's.
func TestPruneGamePoints(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "saves")
	write(t, filepath.Join(live, "a.sav"), "x")
	locs := []SaveLoc{{ID: "a", Live: live}}
	root := filepath.Join(dir, "pts")
	at := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	mk := func(p SavePoint) SavePoint {
		at = at.Add(time.Minute)
		p, err := MakeSavePoint(root, locs, p, at, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	mine := mk(SavePoint{Name: "Before the boss"})
	g1 := mk(SavePoint{Game: true})
	auto := mk(SavePoint{Name: "Before the update", Auto: true})
	g2 := mk(SavePoint{Game: true})
	g3 := mk(SavePoint{Game: true})
	g4 := mk(SavePoint{Game: true})
	if err := RenameSavePoint(root, g2.ID, "The good ending"); err != nil {
		t.Fatal(err)
	}
	n, err := PruneGamePoints(root, 1)
	if err != nil || n != 2 {
		t.Fatalf("pruned %d %v", n, err)
	}
	left := map[string]bool{}
	for _, p := range SavePoints(root) {
		left[p.ID] = true
	}
	for _, keep := range []SavePoint{mine, auto, g2, g4} {
		if !left[keep.ID] {
			t.Fatalf("%q must stay", keep.Name)
		}
	}
	if left[g1.ID] || left[g3.ID] {
		t.Fatal("the oldest game saves go")
	}
}
