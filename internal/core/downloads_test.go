package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMatchDownloads(t *testing.T) {
	dl := t.TempDir()
	big := make([]byte, 80<<10)
	old := time.Now().Add(-time.Hour)
	putAt := func(rel string, at time.Time) {
		p := filepath.Join(dl, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, big, 0o644); err != nil {
			t.Fatal(err)
		}
		_ = os.Chtimes(p, at, at)
		if d := filepath.Dir(p); d != dl {
			_ = os.Chtimes(d, old, old)
		}
	}
	put := func(rel string) { putAt(rel, old) }
	put("Hollow Tale v1.3/Game.exe")
	put("HollowTale.v1.2.zip")
	put("Other.Game.v2.0.zip")
	put("HollowTale.v1.4.rar.part")
	put("Hollow.Tale.v1.5.part1.rar")
	put("Hollow.Tale.v1.5.part2.rar")
	put("Hollow.Tale.2.v1.0.zip")
	put("tiny/HollowTale.v1.6.zip")
	if err := os.WriteFile(filepath.Join(dl, "tiny", "HollowTale.v1.6.zip"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	put("Updates/HollowTale.v1.7.7z")
	put("Hollow Tale copy/HollowTale.exe")
	put("setup_hollow_tale_1.9_(64bit).exe")
	putAt("Hollow Tale v1.10/Game.exe", time.Now().AddDate(0, 0, -60)) // extracted now, with the archive's file times
	put("Hollow.Tale.v1.11.part1.rar")
	put("Hollow.Tale.v1.11.part2.rar.crdownload")
	put("Hollow Tale v2.0/Game.exe") // another Library game's folder

	games := []UpdateTarget{{Key: "found:hollow", Name: "Hollow Tale", Dir: `D:\Games\Hollow Tale`, Version: "1.2"},
		{Key: "found:hollow2", Name: "Hollow Tale Remake", Dir: filepath.Join(dl, "Hollow Tale v2.0")}}
	got := map[string]DownloadMatch{}
	for _, m := range MatchDownloads(dl, games, time.Now().AddDate(0, 0, -30)) {
		if _, twice := got[m.Name]; twice {
			t.Errorf("%s matched twice", m.Name)
		}
		got[m.Name] = m
	}
	want := map[string]struct {
		kind, state string
		newer       bool
	}{
		"Hollow Tale v1.3":                  {"folder", "ok", true},
		"HollowTale.v1.2.zip":               {"zip", "same", false},
		"Hollow.Tale.v1.5.part1.rar":        {"rar", "ok", true},
		"HollowTale.v1.7.7z":                {"7z", "ok", true},
		"setup_hollow_tale_1.9_(64bit).exe": {"exe", "ok", true},
		"Hollow Tale v1.10":                 {"folder", "ok", true},
	}
	if len(got) != len(want) {
		t.Errorf("matched %d, want %d: %v", len(got), len(want), got)
	}
	for name, w := range want {
		m, ok := got[name]
		if !ok {
			t.Errorf("%s didn't match", name)
			continue
		}
		if m.Key != "found:hollow" || m.Kind != w.kind || m.Check.State != w.state || m.Newer != w.newer {
			t.Errorf("%s = %+v, want %+v", name, m, w)
		}
	}
	if m := got["Hollow Tale v1.3"]; m.Check.To != "1.3" || m.Size != int64(len(big)) || m.Game != "Hollow Tale" {
		t.Errorf("folder match = %+v", m)
	}
	if fs := got["Hollow.Tale.v1.5.part1.rar"].Files(); len(fs) != 2 || filepath.Base(fs[1]) != "Hollow.Tale.v1.5.part2.rar" {
		t.Errorf("the set's files = %v", fs)
	}
	if m := got["Hollow.Tale.v1.5.part1.rar"]; m.Size != 2*int64(len(big)) {
		t.Errorf("the set's size = %d, want both parts", m.Size)
	}
	if fs := got["HollowTale.v1.2.zip"].Files(); len(fs) != 1 {
		t.Errorf("a zip's files = %v", fs)
	}

	for _, m := range MatchDownloads(dl, games, time.Now().Add(-time.Minute)) {
		if m.Kind != "folder" { // a folder counts from when it was made, where the PC knows
			t.Errorf("since: %s is older than it", m.Name)
		}
	}
	put("Hollow Tale v1.8.zip")
	now := time.Now()
	_ = os.Chtimes(filepath.Join(dl, "Hollow Tale v1.8.zip"), now, now)
	for _, m := range MatchDownloads(dl, games, time.Time{}) {
		if m.Name == "Hollow Tale v1.8.zip" {
			t.Error("a file written seconds ago matched")
		}
	}
	inside := []UpdateTarget{{Key: "found:dl", Name: "Hollow Tale", Dir: filepath.Join(dl, "Hollow Tale v1.3"), Version: "1.3"}}
	for _, m := range MatchDownloads(dl, inside, time.Time{}) {
		if m.Name == "Hollow Tale v1.3" {
			t.Error("the game's own folder matched as its update")
		}
	}
}

func TestNameFits(t *testing.T) {
	for _, c := range []struct {
		name, game string
		want       bool
	}{
		{"Hollow.Tale.Update.v1.3-GROUP.rar", "Hollow Tale", true},
		{"setup_hollow_tale_1.3_(64bit).exe", "Hollow Tale", true},
		{"Hollow Tale 1.3.zip", "Hollow Tale", true},
		{"Hollow Tale 2 v1.0.zip", "Hollow Tale", false},
		{"Hollow Tale II v1.0.zip", "Hollow Tale", false},
		{"Hollow Tale 2 v1.0.zip", "Hollow Tale 2", true},
		{"Hollow.zip", "Hollow Tale", false},
		{"My Hollow Tale v1.3.zip", "Hollow Tale", false},
	} {
		_, to := UpdateVersions(c.name)
		if got := nameFits(nameTokens(c.name), norm(c.game), to); got != c.want {
			t.Errorf("nameFits(%q, %q) = %v, want %v", c.name, c.game, got, c.want)
		}
	}
}

func TestDownloadsSeen(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	f := filepath.Join(t.TempDir(), "HollowTale.v1.3.zip")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	here := DownloadMatch{Path: f, Size: 1}.Stamp()
	gone := DownloadMatch{Path: f + ".old", Size: 1}.Stamp()
	if err := s.UpdateDownloadsSeen(func(d *DownloadsSeen) { d.Announced[here], d.Announced[gone], d.Ignored[gone] = true, true, true }); err != nil {
		t.Fatal(err)
	}
	var got DownloadsSeen
	_ = s.UpdateDownloadsSeen(func(d *DownloadsSeen) { got = *d })
	if !got.Announced[here] || got.Announced[gone] || got.Ignored[gone] {
		t.Fatalf("seen = %+v; want only the file still there", got)
	}
}
