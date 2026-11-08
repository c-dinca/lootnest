package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// testdata/playnite/library holds files written by LiteDB 4.1.4 (the
// version Playnite ships) with Playnite's mapper settings: 158 games, one
// of them deleted again, one updated past its first data block, one with
// a note long enough for two extend pages, and three index pages.

func playniteFixture(t *testing.T) []PlayniteGame {
	t.Helper()
	t.Setenv("LOCALAPPDATA", "")
	lib, games, err := ReadPlaynite(filepath.Join("testdata", "playnite"))
	if err != nil {
		t.Fatal(err)
	}
	if lib != filepath.Join("testdata", "playnite", "library") {
		t.Fatalf("library folder %q", lib)
	}
	return games
}

func playniteNamed(games []PlayniteGame, name string) *PlayniteGame {
	for i := range games {
		if games[i].Name == name {
			return &games[i]
		}
	}
	return nil
}

func TestReadPlaynite(t *testing.T) {
	games := playniteFixture(t)
	if len(games) != 157 {
		t.Fatalf("%d games, want 157", len(games))
	}
	if playniteNamed(games, "Deleted Game") != nil {
		t.Fatal("a deleted game was read")
	}
	h := playniteNamed(games, "Hades")
	want := PlayniteGame{Name: "Hades", Store: "steam", GameID: "1145360", Playtime: 43200, LastActivity: time.Date(2026, 10, 1, 18, 30, 0, 0, time.UTC),
		Favorite: true, Status: "Beaten", Categories: []string{"Co-op"}, Tags: []string{"Backlog 2026"}, UserScore: 80, Notes: "Heat 8 next.",
		InstallDir: `C:\Program Files (x86)\Steam\steamapps\common\Hades`}
	if h == nil || !reflect.DeepEqual(*h, want) {
		t.Fatalf("Hades:\n got %+v\nwant %+v", h, want)
	}
	if g := playniteNamed(games, "Alan Wake 2"); g == nil || g.Store != "epic" || g.GameID != "dc9d2e595d0e4650b35d659f90d41059" || g.Status != "Playing" || g.UserScore != 95 {
		t.Fatalf("Alan Wake 2: %+v", g)
	}
	w := playniteNamed(games, "The Witcher 3: Wild Hunt")
	if w == nil || w.Store != "gog" || w.Status != "Completed" || !reflect.DeepEqual(w.Categories, []string{"Soulslike", "Co-op"}) || w.InstallDir != "" {
		t.Fatalf("Witcher 3, {PlayniteDir} unknown: %+v", w)
	}
	if g := playniteNamed(games, "Hollow Knight"); g == nil || !strings.HasPrefix(g.Notes, "Grew past one block. xxx") || len(g.Notes) != 21+3000 || g.Status != "Abandoned" {
		t.Fatalf("Hollow Knight, updated: %+v", g)
	}
	if g := playniteNamed(games, "Outer Wilds"); g == nil || g.Notes != strings.Repeat("n", 6000) || g.Status != "On Hold" {
		t.Fatal("Outer Wilds' note, on extend pages, didn't read back")
	}
	if g := playniteNamed(games, "Celeste™"); g == nil || g.Status != "Plan to Play" || g.Store != "" {
		t.Fatalf("Celeste: %+v", g)
	}
	// The library folder itself works too.
	if _, again, err := ReadPlaynite(filepath.Join("testdata", "playnite", "library")); err != nil || len(again) != 157 {
		t.Fatalf("from the library folder: %d, %v", len(again), err)
	}
}

// A portable Playnite's {PlayniteDir} is the folder beside its library.
func TestReadPlaynitePortableDir(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "library"), os.DirFS(filepath.Join("testdata", "playnite", "library"))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Playnite.DesktopApp.exe"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, games, err := ReadPlaynite(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w := playniteNamed(games, "The Witcher 3: Wild Hunt"); w == nil || w.InstallDir != dir+`\Games\Witcher 3` {
		t.Fatalf("Witcher 3: %+v", w)
	}
}

func TestReadPlayniteRefuses(t *testing.T) {
	good, err := os.ReadFile(filepath.Join("testdata", "playnite", "library", "games.db"))
	if err != nil {
		t.Fatal(err)
	}
	lib := func(t *testing.T, games []byte) string {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "library"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "library", "games.db"), games, 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	if _, _, err := ReadPlaynite(t.TempDir()); err == nil || !strings.Contains(err.Error(), "no Playnite library") {
		t.Fatalf("an empty folder: %v", err)
	}
	old := t.TempDir()
	_ = os.MkdirAll(filepath.Join(old, "library", "games"), 0o755)
	if _, _, err := ReadPlaynite(old); err == nil || !strings.Contains(err.Error(), "Playnite 9") {
		t.Fatalf("a library from before Playnite 9: %v", err)
	}
	if _, _, err := ReadPlaynite(lib(t, []byte("not a database"))); err == nil {
		t.Fatal("a file that isn't LiteDB was read")
	}
	enc := bytes.Clone(good)
	enc[90] = 7
	if _, _, err := ReadPlaynite(lib(t, enc)); err == nil || !strings.Contains(err.Error(), "encrypted") {
		t.Fatalf("an encrypted file: %v", err)
	}
	v5 := bytes.Clone(good)
	v5[52] = 8
	if _, _, err := ReadPlaynite(lib(t, v5)); err == nil {
		t.Fatal("another file version was read")
	}
	if _, _, err := ReadPlaynite(lib(t, good[:3*litePage])); err == nil {
		t.Fatal("a cut-short file was read without a word")
	}
	// A file Lootnest can't open (Playnite holds it while it runs) says so.
	if _, err := readLiteDB(t.TempDir()); err == nil || !strings.Contains(err.Error(), "close it") {
		t.Fatalf("an unreadable file: %v", err)
	}
	// Damage anywhere is refused or read around, never a panic.
	for i := 0; i < len(good); i += 53 {
		bad := bytes.Clone(good)
		bad[i] ^= 0xA5
		_, _ = liteDocs(bad)
		_, _ = liteDocs(good[:i])
	}
}

func TestPlayniteStars(t *testing.T) {
	for score, want := range map[int]int{0: 0, -3: 0, 1: 1, 20: 1, 21: 2, 50: 3, 80: 4, 81: 5, 95: 5, 100: 5} {
		if got := PlayniteStars(score); got != want {
			t.Errorf("score %d: %d stars, want %d", score, got, want)
		}
	}
}

func TestPlayniteImportMatching(t *testing.T) {
	lib := []FoundGame{
		{Key: "steam:1145360", Name: "Hades", Source: "steam", AppID: 1145360, Dir: `C:\Steam\steamapps\common\Hades`},
		{Key: "dir:d:\\epic\\alanwake2", Name: "Alan Wake II", Source: "epic", StoreID: EpicID("ns", "item", "DC9D2E595D0E4650B35D659F90D41059"), Dir: `D:\Epic\AlanWake2`},
		{Key: "dir:c:\\gog\\w3", Name: "Witcher 3", Source: "gog", StoreID: "1207664663"},
		{Key: "dir:e:\\games\\hollow knight", Name: "HK", Source: "folder", Dir: `E:\Games\Hollow Knight\`},
		{Key: "dir:x:\\celeste", Name: "Celeste", Source: "folder"},
		{Key: "dir:x:\\hydra\\tunic", Name: "TUNIC", Source: "hydra", SteamID: 553420},
		{Key: "game:p1", Name: "Hades", Source: "folder", Dir: `D:\Other\Hades`},
	}
	games := []PlayniteGame{
		{Name: "Hades", Store: "steam", GameID: "1145360", InstallDir: `D:\Other\Hades`},
		{Name: "Alan Wake 2", Store: "epic", GameID: "dc9d2e595d0e4650b35d659f90d41059"},
		{Name: "The Witcher 3: Wild Hunt", Store: "gog", GameID: "1207664663"},
		{Name: "Hollow Knight", InstallDir: "e:/games/hollow knight"},
		{Name: "Celeste™", Store: "steam", GameID: "504230"},
		{Name: "Tunic", Store: "steam", GameID: "553420"},
		{Name: "Hades", InstallDir: `D:\Other\Hades`},
		{Name: "Not On This PC", Store: "steam", GameID: "999999"},
		{Name: "Hades", Store: "steam", GameID: "1145360"},
	}
	for i := range games {
		games[i].Playtime = 3600
	}
	p := PlayniteImport(games, lib, PlayniteHave{})
	got := map[string]string{}
	for _, c := range p.Changes {
		got[c.Key] = c.From + " by " + c.By
	}
	want := map[string]string{
		"steam:1145360":                "Hades by store",
		"dir:d:\\epic\\alanwake2":      "Alan Wake 2 by store",
		"dir:c:\\gog\\w3":              "The Witcher 3: Wild Hunt by store",
		"dir:e:\\games\\hollow knight": "Hollow Knight by folder",
		"dir:x:\\celeste":              "Celeste™ by name",
		"dir:x:\\hydra\\tunic":         "Tunic by store",
		"game:p1":                      "Hades by folder",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matches:\n got %v\nwant %v", got, want)
	}
	// Each Library game is matched once: the second Steam Hades has none left.
	if p.Matched != 7 || !reflect.DeepEqual(p.Unmatched, []string{"Not On This PC", "Hades"}) {
		t.Fatalf("matched %d, unmatched %v", p.Matched, p.Unmatched)
	}
}

func TestPlayniteImportFields(t *testing.T) {
	lib := []FoundGame{{Key: "steam:1", Name: "One", Source: "steam", AppID: 1}, {Key: "steam:2", Name: "Two", Source: "steam", AppID: 2}}
	last := time.Date(2026, 10, 1, 18, 0, 0, 0, time.UTC)
	statuses := map[string]string{"Playing": "playing", "Beaten": "finished", "Completed": "complete", "Abandoned": "stopped", "Plan to Play": "next", "On Hold": "next", "Not Played": "", "Played": ""}
	for from, to := range statuses {
		p := PlayniteImport([]PlayniteGame{{Name: "One", Store: "steam", GameID: "1", Status: from}}, lib, PlayniteHave{})
		got := ""
		if len(p.Changes) == 1 {
			got = p.Changes[0].Status
		}
		if got != to {
			t.Errorf("status %q: %q, want %q", from, got, to)
		}
	}
	g := PlayniteGame{Name: "One", Store: "steam", GameID: "1", Playtime: 10 * 3600, LastActivity: last, Favorite: true, Status: "Beaten",
		Categories: []string{"Co-op", "RPG"}, Tags: []string{"rpg", "Backlog"}, UserScore: 80, Notes: "Line one\r\nline two"}
	p := PlayniteImport([]PlayniteGame{g}, lib, PlayniteHave{})
	want := PlayniteChange{Key: "steam:1", Name: "One", From: "One", By: "store", Add: 10 * 3600, Total: 10 * 3600, LastPlayed: last, Status: "finished", Stars: 4,
		Note: "Line one\nline two", Favourite: true, Categories: []string{"Co-op", "RPG"}, Tags: []string{"Backlog"}}
	if len(p.Changes) != 1 || !reflect.DeepEqual(p.Changes[0], want) {
		t.Fatalf("change:\n got %+v\nwant %+v", p.Changes, want)
	}
	// What the profile has stays: its status, rating, note, collections,
	// a later last played, and the time an earlier import brought in.
	have := PlayniteHave{
		Shelf:    map[string]ShelfEntry{"steam:1": {Status: "playing", Rating: 2, Note: "Mine"}},
		Imported: map[string]int64{"steam:1": 9 * 3600},
		Played:   map[string]time.Time{"steam:1": last.Add(time.Hour)},
		Tags:     map[string][]string{"steam:1": {"Favourites", "co-op"}},
	}
	p = PlayniteImport([]PlayniteGame{g}, lib, have)
	want = PlayniteChange{Key: "steam:1", Name: "One", From: "One", By: "store", Add: 3600, Total: 10 * 3600, Categories: []string{"RPG"}, Tags: []string{"Backlog"}}
	if len(p.Changes) != 1 || !reflect.DeepEqual(p.Changes[0], want) {
		t.Fatalf("over what's there:\n got %+v\nwant %+v", p.Changes, want)
	}
	// Nothing new: matched, but no change listed.
	have.Imported["steam:1"] = 10 * 3600
	have.Tags["steam:1"] = append(have.Tags["steam:1"], "RPG", "Backlog")
	if p = PlayniteImport([]PlayniteGame{g}, lib, have); p.Matched != 1 || len(p.Changes) != 0 {
		t.Fatalf("a second import: %+v", p)
	}
	long := PlayniteImport([]PlayniteGame{{Name: "Two", Store: "steam", GameID: "2", Notes: strings.Repeat("é", 600)}}, lib, PlayniteHave{})
	if n := len([]rune(long.Changes[0].Note)); n != 500 {
		t.Fatalf("a long note is %d characters", n)
	}
	only := want.Only(PlayniteOptions{Categories: true})
	if only.Add != 0 || only.Total != 0 || len(only.Tags) != 0 || !reflect.DeepEqual(only.Categories, []string{"RPG"}) {
		t.Fatalf("Only: %+v", only)
	}
}

func TestApplyPlayniteAndUndo(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if err := s.SetShelf("p1", "steam:1", ShelfEntry{Status: "playing", Offset: 600}, now); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPlay("steam:1", "p2", now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	read := func(n string) []byte { b, _ := os.ReadFile(filepath.Join(s.Dir, n)); return b }
	before := map[string][]byte{}
	for _, n := range importFiles {
		before[n] = read(n)
	}
	id, err := s.BackupBeforeImport(now)
	if err != nil {
		t.Fatal(err)
	}

	lib := []FoundGame{{Key: "steam:1", Name: "One", Source: "steam", AppID: 1}, {Key: "steam:2", Name: "Two", Source: "steam", AppID: 2}}
	games := []PlayniteGame{
		{Name: "One", Store: "steam", GameID: "1", Playtime: 7200, LastActivity: now.Add(-2 * time.Hour), Status: "Beaten", UserScore: 60, Favorite: true, Categories: []string{"Co-op"}},
		{Name: "Two", Store: "steam", GameID: "2", Playtime: 3600, LastActivity: now.Add(-time.Minute), Notes: "Hi"},
	}
	plan := func() PlaynitePlan {
		sh := s.LoadShelf()["p1"]
		played := map[string]time.Time{}
		for k, p := range s.LoadRecent() {
			played[k] = p.Players["p1"]
		}
		return PlayniteImport(games, lib, PlayniteHave{Shelf: sh, Imported: s.PlayniteImported("p1"), Played: played, Tags: cfg.Tags})
	}
	p := plan()
	if err := s.ApplyPlaynite("p1", p.Changes, now); err != nil {
		t.Fatal(err)
	}
	if skipped := cfg.ApplyPlaynite(p.Changes); skipped != 0 {
		t.Fatalf("%d collections skipped", skipped)
	}
	sh := s.LoadShelf()["p1"]
	if e := sh["steam:1"]; e.Status != "playing" || e.Rating != 3 || e.Offset != 600+7200 {
		t.Fatalf("One on the shelf: %+v", e)
	}
	if e := sh["steam:2"]; e.Note != "Hi" || e.Offset != 3600 {
		t.Fatalf("Two on the shelf: %+v", e)
	}
	rec := s.LoadRecent()
	if r := rec["steam:1"]; r.Player != "p2" || !r.Players["p1"].Equal(now.Add(-2*time.Hour)) {
		t.Fatalf("One's last played: %+v", r)
	}
	if r := rec["steam:2"]; r.Player != "p1" || !r.At.Equal(now.Add(-time.Minute)) {
		t.Fatalf("Two's last played: %+v", r)
	}
	if !reflect.DeepEqual(cfg.Tags["steam:1"], []string{"Favourites", "Co-op"}) {
		t.Fatalf("collections: %v", cfg.Tags)
	}

	// Again, and again after an hour more in Playnite: only the new hour.
	if p := plan(); p.Matched != 2 || len(p.Changes) != 0 {
		t.Fatalf("a second import changes %+v", p.Changes)
	}
	games[0].Playtime += 3600
	p = plan()
	if len(p.Changes) != 1 || p.Changes[0].Add != 3600 {
		t.Fatalf("after an hour more: %+v", p.Changes)
	}

	if err := s.UndoImport(id); err != nil {
		t.Fatal(err)
	}
	for _, n := range importFiles {
		if !bytes.Equal(read(n), before[n]) {
			t.Errorf("%s isn't back as it was", n)
		}
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "playnite.json")); !os.IsNotExist(err) {
		t.Fatal("the import's own playnite.json stayed after Undo")
	}
	if b, err := os.ReadFile(filepath.Join(s.importDir(id), "undone", "playnite.json")); err != nil || len(b) == 0 {
		t.Fatal("what Undo replaced wasn't kept")
	}
	if err := s.UndoImport("../x"); err == nil {
		t.Fatal("an id outside backups\\import was taken")
	}
}

// Reading Playnite's library and applying an import never writes to it.
func TestPlayniteLeavesItsFilesAlone(t *testing.T) {
	pn := t.TempDir()
	if err := os.CopyFS(filepath.Join(pn, "library"), os.DirFS(filepath.Join("testdata", "playnite", "library"))); err != nil {
		t.Fatal(err)
	}
	snap := func() map[string]string {
		out := map[string]string{}
		_ = filepath.WalkDir(pn, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				st, _ := d.Info()
				b, _ := os.ReadFile(p)
				h := sha256.Sum256(b)
				out[p] = st.ModTime().String() + string(h[:])
			}
			return nil
		})
		return out
	}
	was := snap()
	_, games, err := ReadPlaynite(pn)
	if err != nil {
		t.Fatal(err)
	}
	s := &Store{Dir: t.TempDir()}
	p := PlayniteImport(games, []FoundGame{{Key: "steam:1145360", Name: "Hades", Source: "steam", AppID: 1145360}}, PlayniteHave{})
	if len(p.Changes) != 1 {
		t.Fatalf("%+v", p)
	}
	if _, err := s.BackupBeforeImport(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyPlaynite("p1", p.Changes, time.Now()); err != nil {
		t.Fatal(err)
	}
	if now := snap(); !reflect.DeepEqual(now, was) {
		t.Fatal("Playnite's folder changed")
	}
	b, _ := json.Marshal(p.Changes[0])
	if !strings.Contains(string(b), `"lastPlayed":"2026-10-01T18:30:00Z"`) {
		t.Fatalf("change as sent to the page: %s", b)
	}
}
