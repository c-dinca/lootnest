package core

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const ludusaviSample = `---
"Hollow Knight":
  files:
    "<winLocalAppDataLow>/Team Cherry/Hollow Knight/*.dat":
      tags:
        - save
      when:
        - os: windows
    "<home>/.config/unity3d/Team Cherry/Hollow Knight":
      when:
        - os: linux
  installDir:
    Hollow Knight: {}
  steam:
    id: 367520
"Some Indie":
  files:
    "<base>/Saves":
      tags:
        - save
    "<winAppData>/SomeIndie/**/settings.ini":
      tags:
        - config
    "<root>/userdata/<storeUserId>/1/remote":
      tags:
        - save
  installDir:
    SomeIndie: {}
"No Saves Listed":
  steam:
    id: 1
"Logs Only":
  files:
    "<winAppData>/X/log.txt":
      tags:
        - log
`

func TestSaveDB(t *testing.T) {
	db, err := ParseSaveDB(strings.NewReader(ludusaviSample))
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Games) != 2 {
		t.Fatalf("only games with Windows save or settings locations: %+v", db.Games)
	}
	path := filepath.Join(t.TempDir(), "savedb.json.gz")
	if err := SaveDBFile(path, db); err != nil {
		t.Fatal(err)
	}
	db, err = LoadSaveDB(path)
	if err != nil {
		t.Fatal(err)
	}
	hk := db.Find(367520)
	if hk == nil || hk.Name != "Hollow Knight" || len(hk.Files) != 1 {
		t.Fatalf("by Steam id, Windows only: %+v", hk)
	}
	si := db.Find(0, "some indie")
	if si == nil || len(si.Files) != 2 {
		t.Fatalf("by name, without <root>: %+v", si)
	}
	if db.Find(0, "SomeIndie") == nil || db.Find(0, "Unknown Game") != nil {
		t.Fatal("by install folder; unknown games aren't found")
	}

	// On this PC.
	f := newFixture(t)
	base := filepath.Join(f.root, "Games", "SomeIndie")
	write(t, filepath.Join(base, "Saves", "slot1"), "s")
	write(t, filepath.Join(f.appdata, "SomeIndie", "a", "b", "settings.ini"), "x")
	locs := f.env.FindSaves(si, base)
	if len(locs) != 2 {
		t.Fatalf("%+v", locs)
	}
	for _, l := range locs {
		if strings.HasSuffix(l.Resolved, "settings.ini") && !strings.HasPrefix(l.Path, "%APPDATA%") {
			t.Fatalf("stored tokenized: %+v", l)
		}
	}
	write(t, filepath.Join(f.home, "AppData", "LocalLow", "Team Cherry", "Hollow Knight", "user1.dat"), "d")
	write(t, filepath.Join(f.home, "AppData", "LocalLow", "Team Cherry", "Hollow Knight", "user2.dat"), "d")
	if locs := f.env.FindSaves(hk, ""); len(locs) != 2 {
		t.Fatalf("globs match each file: %+v", locs)
	}
}

func TestGameVersions(t *testing.T) {
	f := newFixture(t)
	root := t.TempDir()
	c := &Cloud{Root: root, Computer: "PC-A"}
	saves := filepath.Join(f.appdata, "Indie", "Saves")
	write(t, filepath.Join(saves, "slot1.sav"), "level 1")
	locs := []SaveLocation{{Path: f.env.Tokenize(saves), Resolved: saves}}
	key := GameSavesKey("Indie", "")
	h1, _ := SavesHash(locs)
	if _, err := c.SaveGameVersion(key, "Indie", locs, h1, 2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond) // versions are named by the second
	write(t, filepath.Join(saves, "slot1.sav"), "level 2")
	h2, _ := SavesHash(locs)
	if h1 == h2 {
		t.Fatal("changed saves, changed hash")
	}
	if _, err := c.SaveGameVersion(key, "Indie", locs, h2, 2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	write(t, filepath.Join(saves, "slot1.sav"), "level 3")
	h3, _ := SavesHash(locs)
	_, _ = c.SaveGameVersion(key, "Indie", locs, h3, 2)
	vs := c.GameVersions(key)
	if len(vs) != 2 || vs[0].Hash != h3 || vs[1].Hash != h2 {
		t.Fatalf("the newest two, newest first: %+v", vs)
	}
	// Restore the older one on another PC where the game has never run:
	// the folder comes from the version's own path.
	g := newFixture(t)
	other := &Cloud{Root: root, Computer: "PC-B"}
	skipped, err := other.RestoreGameVersion(key, vs[1].ID, nil, func(p string) (string, error) {
		return g.env.Resolve(strings.Replace(p, f.env.Tokenize(f.appdata), "%APPDATA%", 1))
	}, filepath.Join(t.TempDir(), "tmp"))
	if err != nil || skipped != 0 {
		t.Fatal(skipped, err)
	}
	if read(t, filepath.Join(g.appdata, "Indie", "Saves", "slot1.sav")) != "level 2" {
		t.Fatal("restored on the other PC")
	}
	if _, err := other.RestoreGameVersion(key, "../../x", nil, nil, t.TempDir()); err == nil {
		t.Fatal("an id with a path is refused")
	}
}

func TestInFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Game")
	locs := []SaveLocation{
		{Path: "a", Resolved: filepath.Join(dir, "game", "saves")},
		{Path: "b", Resolved: filepath.Join(filepath.Dir(dir), "AppData", "Game")},
		{Path: "c", Resolved: dir},
	}
	got := InFolder(locs, dir)
	if len(got) != 1 || got[0].Path != "a" {
		t.Fatalf("InFolder = %+v", got)
	}
	if InFolder(locs, "") != nil {
		t.Fatal("no folder, no locations")
	}
}

func TestFolderSaves(t *testing.T) {
	f := newFixture(t)
	game := filepath.Join(f.root, "Games", "Tale")
	write(t, filepath.Join(game, "game", "saves", "1-1-LT1.save"), "s")
	write(t, filepath.Join(game, "Tale", "Saved", "SaveGames", "slot.sav"), "u")
	write(t, filepath.Join(game, "data", "x.bin"), "not a save")
	got := map[string]bool{}
	for _, l := range f.env.FolderSaves(game) {
		got[l.Resolved] = true
	}
	if len(got) != 2 || !got[filepath.Join(game, "game", "saves")] || !got[filepath.Join(game, "Tale", "Saved", "SaveGames")] {
		t.Fatalf("FolderSaves = %v", got)
	}
	if f.env.FolderSaves("") != nil || f.env.FolderSaves("relative") != nil {
		t.Fatal("needs a folder")
	}
}

func TestVersionTargetsKeepsEveryLocation(t *testing.T) {
	f := newFixture(t)
	c := &Cloud{Root: t.TempDir(), Computer: "PC-A"}
	a := filepath.Join(f.appdata, "Indie", "Saves")
	b := filepath.Join(f.appdata, "Indie", "Config")
	write(t, filepath.Join(a, "slot1.sav"), "a old")
	write(t, filepath.Join(b, "keys.ini"), "b old")
	both := []SaveLocation{{Path: f.env.Tokenize(a), Resolved: a}, {Path: f.env.Tokenize(b), Resolved: b}}
	key := GameSavesKey("Indie", "")
	h, _ := SavesHash(both)
	v, err := c.SaveGameVersion(key, "Indie", both, h, 2)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(b, "keys.ini"), "b now")
	// The game knows only a here; the version also has b.
	targets, err := c.VersionTargets(key, v.ID, both[:1], f.env.Resolve)
	if err != nil || len(targets) != 2 || targets[1].Resolved != b {
		t.Fatalf("%+v %v", targets, err)
	}
	if _, err := c.RestoreGameVersion(key, v.ID, both[:1], f.env.Resolve, filepath.Join(t.TempDir(), "tmp")); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(b, "keys.ini")) != "b old" {
		t.Fatal("b restored")
	}
}

func TestReadSavesWrongKeyLeavesNoTmp(t *testing.T) {
	f := newFixture(t)
	root := t.TempDir()
	key, err := CloudKey(root, "cloud password")
	if err != nil {
		t.Fatal(err)
	}
	c := &Cloud{Root: root, Key: key, Computer: "PC-A"}
	saves := filepath.Join(f.appdata, "Indie", "Saves")
	write(t, filepath.Join(saves, "slot1.sav"), "level 1")
	locs := []SaveLocation{{Path: f.env.Tokenize(saves), Resolved: saves}}
	game := GameSavesKey("Indie", "")
	v, err := c.SaveGameVersion(game, "Indie", locs, "h", 2)
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "tmp")
	paths, at := locationsAsPaths(locs)
	wrong := make([]byte, len(key))
	if _, err := readSaves(filepath.Join(c.savesDir(game), v.ID+".lnbak"), wrong, paths, at, tmp); err == nil {
		t.Fatal("the wrong key is refused")
	}
	if exists(tmp) {
		t.Fatal("the decrypted copy is removed")
	}
}
