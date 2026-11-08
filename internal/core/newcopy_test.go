package core

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCopyMatches(t *testing.T) {
	l := CopyLedger{Games: map[string]SeenGame{}, Dismissed: map[string]bool{}, Carried: map[string]string{}}
	there := map[string]bool{`D:\Games\PBZ-v1`: true, `D:\Games\Other`: true}
	dirThere := func(d string) bool { return there[d] }
	t0 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	l.NoteSeen([]FoundGame{
		{Key: "dir:d:\\games\\pbz-v1", Name: "Phantom Blade Zero", Dir: `D:\Games\PBZ-v1`, SteamID: 4115450},
		{Key: "dir:d:\\games\\other", Name: "Other Game", Dir: `D:\Games\Other`},
	}, t0, dirThere)
	if len(l.Matches()) != 0 {
		t.Fatal("no new copies yet")
	}

	// A full new version lands in a new folder; the old one is still there.
	there[`D:\Games\PBZ-v2`] = true
	l.NoteSeen([]FoundGame{
		{Key: "dir:d:\\games\\pbz-v1", Name: "Phantom Blade Zero", Dir: `D:\Games\PBZ-v1`, SteamID: 4115450},
		{Key: "dir:d:\\games\\pbz-v2", Name: "Phantom Blade Zero v2", Dir: `D:\Games\PBZ-v2`, SteamID: 4115450},
		{Key: "dir:d:\\games\\other", Name: "Other Game", Dir: `D:\Games\Other`},
	}, t0.Add(time.Hour), dirThere)
	// Later scans don't change which is the old one.
	l.NoteSeen([]FoundGame{
		{Key: "dir:d:\\games\\pbz-v2", Name: "Phantom Blade Zero v2", Dir: `D:\Games\PBZ-v2`, SteamID: 4115450},
		{Key: "dir:d:\\games\\pbz-v1", Name: "Phantom Blade Zero", Dir: `D:\Games\PBZ-v1`, SteamID: 4115450},
	}, t0.Add(2*time.Hour), dirThere)
	m := l.Matches()
	if len(m) != 1 || m[0].Old.Dir != `D:\Games\PBZ-v1` || m[0].New.Dir != `D:\Games\PBZ-v2` || m[0].OldGone {
		t.Fatalf("matches = %+v", m)
	}

	// The old folder is deleted: still a match, now with the old one gone.
	delete(there, `D:\Games\PBZ-v1`)
	l.NoteSeen([]FoundGame{{Key: "dir:d:\\games\\pbz-v2", Name: "Phantom Blade Zero v2", Dir: `D:\Games\PBZ-v2`, SteamID: 4115450}}, t0.Add(3*time.Hour), dirThere)
	if m := l.Matches(); len(m) != 1 || !m[0].OldGone {
		t.Fatalf("after deleting the old folder: %+v", m)
	}
	// Carried over, or dismissed: no more asking.
	l.Carried["dir:d:\\games\\pbz-v1"] = "dir:d:\\games\\pbz-v2"
	if len(l.Matches()) != 0 {
		t.Fatal("a carried copy isn't offered again")
	}
	delete(l.Carried, "dir:d:\\games\\pbz-v1")
	l.Dismissed["dir:d:\\games\\pbz-v1|dir:d:\\games\\pbz-v2"] = true
	if len(l.Matches()) != 0 {
		t.Fatal("a dismissed pair isn't offered again")
	}
}

func TestRekeyGameAndCarrySaves(t *testing.T) {
	f := newFixture(t)
	s := f.eng.Store
	at := time.Date(2026, 11, 1, 20, 0, 0, 0, time.Local)
	_ = s.RecordPlay("old", "p1", at)
	_ = s.AddPlaytime("old", "p1", 2*time.Hour)
	_ = s.AddPlaytime("new", "p1", time.Hour)
	_ = s.AddActivity("old", "p1", at, at.Add(2*time.Hour))
	_ = s.SetShelf("p1", "old", ShelfEntry{Status: "playing", Note: "Act 2", Offset: 5 * 3600}, at)
	_ = s.SetShelf("p1", "new", ShelfEntry{Rating: 4, Offset: 3600}, at.Add(time.Hour))
	_ = writeJSONAtomic(s.CapturesPath(), captureIndex{Captures: []Capture{{Path: "a.png", Key: "old", Manual: true}}})
	if err := s.RekeyGame("old", "new"); err != nil {
		t.Fatal(err)
	}
	if pt := s.LoadPlaytime(); pt["new"]["p1"] != 3*3600 || pt["old"] != nil {
		t.Fatalf("playtime = %+v", pt)
	}
	if r := s.LoadRecent(); r["new"].Player != "p1" || !r["new"].At.Equal(at) {
		t.Fatalf("recent = %+v", r)
	}
	if e := s.LoadShelf()["p1"]["new"]; e.Status != "playing" || e.Note != "Act 2" || e.Rating != 4 || e.Offset != 6*3600 {
		t.Fatalf("shelf = %+v", e)
	}
	if a := s.LoadActivity(); a.Longest["p1"].Key != "new" {
		t.Fatalf("activity = %+v", a.Longest)
	}
	if c := s.LoadCaptures(); len(c) != 1 || c[0].Key != "new" {
		t.Fatalf("captures = %+v", c)
	}
	cfg := &Config{LibraryPrefs: map[string]LibraryPref{"old": {Name: "PBZ"}}, Tags: map[string][]string{"old": {"Souls"}, "new": {"New"}}}
	if !RekeyConfig(cfg, "old", "new") || cfg.LibraryPrefs["new"].Name != "PBZ" || len(cfg.Tags["new"]) != 2 {
		t.Fatalf("config = %+v", cfg)
	}

	// Saves inside the old folder go into the new one; the new copy's own
	// are kept first.
	oldDir, newDir := filepath.Join(f.root, "old"), filepath.Join(f.root, "new")
	write(t, filepath.Join(oldDir, "ProjectPBZ", "Saved", "SaveGames", "slot.sav"), "my progress")
	write(t, filepath.Join(newDir, "ProjectPBZ", "Saved", "SaveGames", "slot.sav"), "a fresh save")
	keep := filepath.Join(f.root, "keep")
	got, err := CarrySaves(newDir, map[string]string{"ProjectPBZ/Saved": filepath.Join(oldDir, "ProjectPBZ", "Saved")}, keep)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Replaced || read(t, filepath.Join(newDir, "ProjectPBZ", "Saved", "SaveGames", "slot.sav")) != "my progress" ||
		read(t, filepath.Join(keep, "ProjectPBZ", "Saved", "SaveGames", "slot.sav")) != "a fresh save" {
		t.Fatalf("carried = %+v", got)
	}
	if _, err := CarrySaves(newDir, map[string]string{"../escape": oldDir}, keep); err == nil {
		t.Fatal("a save location outside the game's folder is refused")
	}
}

func TestReseat(t *testing.T) {
	f := newFixture(t)
	write(t, f.optFile(), "p1 settings")
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{FirstOwner: "p1"}); err != nil {
		t.Fatal(err)
	}
	write(t, f.optFile(), "a new copy's defaults")
	who, err := f.eng.Reseat(f.cfg, "cs2", 10)
	if err != nil || who != "p1" || read(t, f.optFile()) != "p1 settings" || len(f.eng.Backups("cs2")) != 1 {
		t.Fatalf("reseat = %q, %v, %q", who, err, read(t, f.optFile()))
	}
}

func TestNoteMovedIsNoNewCopy(t *testing.T) {
	l := CopyLedger{Games: map[string]SeenGame{}, Dismissed: map[string]bool{}, Carried: map[string]string{}}
	there := map[string]bool{`D:\Games\Hollow Tale`: true}
	dirThere := func(d string) bool { return there[d] }
	t0 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	l.NoteSeen([]FoundGame{{Key: `dir:d:\games\hollow tale`, Name: "Hollow Tale", Dir: `D:\Games\Hollow Tale`}}, t0, dirThere)

	// Moved to E: the old folder goes, the same copy is seen in the new one.
	l.NoteMoved(`dir:d:\games\hollow tale`, `dir:e:\games\hollow tale`, `E:\Games\Hollow Tale`)
	there = map[string]bool{`E:\Games\Hollow Tale`: true}
	l.NoteSeen([]FoundGame{{Key: `dir:e:\games\hollow tale`, Name: "Hollow Tale", Dir: `E:\Games\Hollow Tale`}}, t0.Add(time.Hour), dirThere)
	if m := l.Matches(); len(m) != 0 {
		t.Fatalf("a moved game is offered as a new copy of itself: %+v", m)
	}
	if g := l.Games[`dir:e:\games\hollow tale`]; !g.First.Equal(t0) || g.Dir != `E:\Games\Hollow Tale` || len(l.Games) != 1 {
		t.Fatalf("ledger %+v", l.Games)
	}
}
