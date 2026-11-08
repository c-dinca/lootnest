package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanRemove(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "ana")
	pf := filepath.Join(root, "Program Files")
	data := filepath.Join(home, "AppData", "Roaming", "Lootnest")
	e := &Env{Home: home, DataDir: data, ExtraProtected: []string{pf},
		Getenv: func(k string) string { return map[string]string{"USERPROFILE": home}[k] }}
	game := filepath.Join(root, "Games", "Hollow Tale")
	write(t, filepath.Join(game, "HollowTale.exe"), "exe")
	write(t, filepath.Join(game, "saves", "slot1.sav"), "slot one")
	other := filepath.Join(home, "Saved Games", "Hollow Tale")
	write(t, filepath.Join(other, "settings.ini"), "vsync=1")
	empty := filepath.Join(game, "empty")
	_ = os.MkdirAll(empty, 0o755)
	backups := filepath.Join(data, "backups")
	at := time.Date(2026, 11, 8, 20, 15, 0, 0, time.Local)
	locs := []SaveLocation{{Resolved: filepath.Join(game, "saves")}, {Resolved: other}, {Resolved: empty}, {Resolved: game}}

	p, err := e.PlanRemove("Hollow Tale", game, locs, backups, false, at)
	if err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(backups, "removed", "hollowtale", "20261108-201500")
	if p.Keep != keep || p.Dir != game || p.Size == 0 {
		t.Fatalf("plan %+v", p)
	}
	if len(p.Saves) != 2 || p.Saves[0].Rel != "saves" || p.Saves[0].To != filepath.Join(keep, "folder", "saves") ||
		p.Saves[1].Rel != "" || p.Saves[1].To != filepath.Join(keep, "other", "2 Hollow Tale") {
		t.Fatalf("saves %+v", p.Saves)
	}
	if err := p.CopySaves(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(keep, "folder", "saves", "slot1.sav")); string(b) != "slot one" {
		t.Errorf("in-folder save not copied: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(keep, "other", "2 Hollow Tale", "settings.ini")); string(b) != "vsync=1" {
		t.Errorf("save elsewhere not copied: %q", b)
	}
	if _, err := os.Stat(filepath.Join(keep, "removed.json")); err != nil {
		t.Error("no removed.json")
	}
	if _, err := os.Stat(filepath.Join(game, "saves", "slot1.sav")); err != nil {
		t.Error("CopySaves touched the game's own files")
	}
	if k := p.KeptSaves(); len(k) != 1 || k["saves"] != filepath.Join(keep, "folder", "saves") {
		t.Errorf("kept %v", k)
	}

	refuse := map[string]string{
		"running":         "running",
		"drive":           "whole drive",
		"program files":   "too broad",
		"home":            "too broad",
		"desktop":         "too broad",
		"games root":      "more than one game",
		"missing":         "isn't on this PC",
		"relative":        "no folder",
		"in lootnest":     "Lootnest's own data",
		"holds lootnest":  "too broad",
		"backups in game": "backups are inside",
	}
	desktop := filepath.Join(home, "Desktop")
	gamesRoot := filepath.Join(root, "D", "Games")
	inData := filepath.Join(data, "x", "y")
	for _, d := range []string{pf, desktop, gamesRoot, inData} {
		_ = os.MkdirAll(d, 0o755)
	}
	for what, want := range refuse {
		dir, run, bk := game, false, backups
		switch what {
		case "running":
			run = true
		case "drive":
			dir = filepath.VolumeName(root) + string(filepath.Separator)
		case "program files":
			dir = pf
		case "home":
			dir = home
		case "desktop":
			dir = desktop
		case "games root":
			dir = gamesRoot
		case "missing":
			dir = filepath.Join(root, "Games", "Gone")
		case "relative":
			dir = "Games/Hollow Tale"
		case "in lootnest":
			dir = inData
		case "holds lootnest":
			dir = filepath.Join(home, "AppData")
		case "backups in game":
			bk = filepath.Join(game, "Lootnest", "backups")
		}
		if _, err := e.PlanRemove("Hollow Tale", dir, locs, bk, run, at); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s (%s): got %v, want an error with %q", what, dir, err, want)
		}
	}
}

// A game removed from this PC and added again, in the same folder or
// another, is offered its kept saves back.
func TestRemovedCopyOfferedBack(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dir := `D:\Games\Hollow Tale`
	f := FoundGame{Key: "dir:" + strings.ToLower(dir), Name: "Hollow Tale", Dir: dir}
	there := map[string]bool{dir: true}
	dirThere := func(d string) bool { return there[d] }
	kept := map[string]string{"saves": `C:\Lootnest\backups\removed\hollowtale\x\folder\saves`}

	for _, again := range []string{dir, `E:\Hollow Tale v2`} {
		l := CopyLedger{Games: map[string]SeenGame{}, Dismissed: map[string]bool{}, Carried: map[string]string{}}
		l.NoteSeen([]FoundGame{f}, t0, dirThere)
		l.NoteRemoved(f, kept, t0.Add(time.Hour))
		there[dir] = false
		l.NoteSeen(nil, t0.Add(2*time.Hour), dirThere)
		if len(l.Matches()) != 0 {
			t.Fatalf("a match before it's back: %v", l.Matches())
		}
		nf := FoundGame{Key: "dir:" + strings.ToLower(again), Name: "Hollow Tale", Dir: again}
		there[again] = true
		l.NoteSeen([]FoundGame{nf}, t0.Add(3*time.Hour), dirThere)
		m := l.Matches()
		if len(m) != 1 || m[0].New.Key != nf.Key || m[0].Old.Kept["saves"] != kept["saves"] || !m[0].OldGone {
			t.Fatalf("added again in %s: %+v", again, m)
		}
		if m[0].Old.Key == m[0].New.Key {
			t.Fatalf("same key for both copies: %+v", m[0])
		}
		l.Carried[m[0].Old.Key] = m[0].New.Key
		if len(l.Matches()) != 0 {
			t.Errorf("still offered once carried over")
		}
	}

	// Removed and added again a second time in the same folder: offered
	// again. Still listed while its folder is gone (a game added by hand):
	// not offered.
	l := CopyLedger{Games: map[string]SeenGame{}, Dismissed: map[string]bool{}, Carried: map[string]string{}}
	there[dir] = true
	l.NoteSeen([]FoundGame{f}, t0, dirThere)
	for round := 1; round <= 2; round++ {
		at := t0.Add(time.Duration(round*10) * time.Hour)
		l.NoteRemoved(f, kept, at)
		there[dir] = false
		l.NoteSeen([]FoundGame{f}, at.Add(time.Hour), dirThere)
		if len(l.Matches()) != 0 || !l.Games[f.Key].Gone || len(l.Games[f.Key].Kept) == 0 {
			t.Fatalf("round %d: listed with its folder gone: %+v", round, l)
		}
		there[dir] = true
		l.NoteSeen([]FoundGame{f}, at.Add(2*time.Hour), dirThere)
		m := l.Matches()
		if len(m) != 1 || m[0].New.Key != f.Key || len(m[0].Old.Kept) == 0 {
			t.Fatalf("round %d: added again: %+v", round, m)
		}
		if round == 1 {
			l.Carried[m[0].Old.Key] = m[0].New.Key
		} else {
			l.Dismissed[m[0].Old.Key+"|"+m[0].New.Key] = true
		}
	}
}
