package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A Steam install with a second library on another "drive".
func moverFixture(t *testing.T) (*fixture, string) {
	f := newFixture(t)
	lib2 := filepath.Join(f.root, "D", "SteamLibrary")
	write(t, filepath.Join(f.steam, "steamapps", "libraryfolders.vdf"), `"libraryfolders"
{
	"0" { "path" "`+strings.ReplaceAll(f.steam, `\`, `\\`)+`" }
	"1" { "path" "`+strings.ReplaceAll(lib2, `\`, `\\`)+`" }
}
`)
	_ = os.MkdirAll(filepath.Join(lib2, "steamapps"), 0o755)
	apps := filepath.Join(f.steam, "steamapps")
	write(t, filepath.Join(apps, "appmanifest_1326470.acf"), `"AppState"
{
	"appid"		"1326470"
	"name"		"Sons Of The Forest"
	"StateFlags"		"4"
	"installdir"		"Sons Of The Forest"
}
`)
	game := filepath.Join(apps, "common", "Sons Of The Forest")
	write(t, filepath.Join(game, "SonsOfTheForest.exe"), "exe")
	write(t, filepath.Join(game, "Data", "level0"), strings.Repeat("x", 3000))
	write(t, filepath.Join(game, "Config", "settings.cfg"), "fov=90")
	write(t, filepath.Join(apps, "workshop", "content", "1326470", "99", "mod.pak"), "mod")
	write(t, filepath.Join(apps, "workshop", "appworkshop_1326470.acf"), `"AppWorkshop" { "appid" "1326470" }`)
	return f, lib2
}

func TestMoveSteamGame(t *testing.T) {
	f, lib2 := moverFixture(t)
	in, err := FindSteamInstall(f.steam, 1326470)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(in.Library, f.steam) || in.InstallDir != "Sons Of The Forest" {
		t.Fatalf("wrong install: %+v", in)
	}
	files, bytes, err := TreeSize(in.Dir())
	if err != nil || files != 3 || bytes != 3+3000+6 {
		t.Fatalf("TreeSize = %d files, %d bytes, %v", files, bytes, err)
	}
	if n, err := in.Size(); err != nil || n != bytes+3 {
		t.Fatalf("Size should count the workshop items too: %d, %v", n, err)
	}
	var last int64
	res, err := MoveSteamGame(in, lib2, func(n int64) { last = n }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Leftover != "" || last < bytes {
		t.Fatalf("leftover %q, progress %d", res.Leftover, last)
	}
	apps2 := filepath.Join(lib2, "steamapps")
	for _, p := range []string{
		filepath.Join(apps2, "common", "Sons Of The Forest", "Data", "level0"),
		filepath.Join(apps2, "common", "Sons Of The Forest", "Config", "settings.cfg"),
		filepath.Join(apps2, "appmanifest_1326470.acf"),
		filepath.Join(apps2, "workshop", "content", "1326470", "99", "mod.pak"),
		filepath.Join(apps2, "workshop", "appworkshop_1326470.acf"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s should be in the new library: %v", p, err)
		}
	}
	apps := filepath.Join(f.steam, "steamapps")
	for _, p := range []string{
		filepath.Join(apps, "common", "Sons Of The Forest"),
		filepath.Join(apps, "appmanifest_1326470.acf"),
		filepath.Join(apps, "workshop", "content", "1326470"),
		filepath.Join(apps, "workshop", "appworkshop_1326470.acf"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s should be gone from the old library", p)
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(apps2, "common", "*"+partSuffix)); len(matches) != 0 {
		t.Fatalf("no partial copy should be left: %v", matches)
	}
	again, err := FindSteamInstall(f.steam, 1326470)
	if err != nil || !samePath(again.Library, lib2) {
		t.Fatalf("Steam should now find the game in the new library: %+v %v", again, err)
	}
}

// A failed copy leaves the game, and Steam's records, exactly where they were.
func TestMoveSteamGameKeepsGameOnFailure(t *testing.T) {
	f, lib2 := moverFixture(t)
	in, _ := FindSteamInstall(f.steam, 1326470)
	// The new library already has the workshop folder: refused before copying.
	write(t, filepath.Join(lib2, "steamapps", "workshop", "content", "1326470", "x"), "x")
	if _, err := MoveSteamGame(in, lib2, nil, nil); err == nil {
		t.Fatal("a clash in the new library should refuse the move")
	}
	if _, err := os.Stat(filepath.Join(in.Dir(), "Data", "level0")); err != nil {
		t.Fatal("the game must stay where it was")
	}
	if _, err := os.Stat(in.Manifest); err != nil {
		t.Fatal("Steam's record must stay where it was")
	}
	if _, err := os.Stat(filepath.Join(lib2, "steamapps", "common", "Sons Of The Forest")); !os.IsNotExist(err) {
		t.Fatal("nothing should be copied when the move is refused")
	}

	// A folder with the game's name already in the new library is refused too.
	_ = os.RemoveAll(filepath.Join(lib2, "steamapps", "workshop"))
	write(t, filepath.Join(lib2, "steamapps", "common", "Sons Of The Forest", "other"), "keep me")
	if _, err := MoveSteamGame(in, lib2, nil, nil); err == nil {
		t.Fatal("an existing folder in the new library must not be overwritten")
	}
	if b, _ := os.ReadFile(filepath.Join(lib2, "steamapps", "common", "Sons Of The Forest", "other")); string(b) != "keep me" {
		t.Fatal("the existing folder must be left alone")
	}

	// The same library, or a folder that isn't a library, is refused.
	if _, err := MoveSteamGame(in, f.steam, nil, nil); err == nil {
		t.Fatal("moving into the same library should be refused")
	}
	if _, err := MoveSteamGame(in, filepath.Join(f.root, "E"), nil, nil); err == nil {
		t.Fatal("a folder without steamapps isn't a library")
	}
}

// If Steam or the game starts while the game is being copied, nothing in the
// old library is touched and the copy is removed.
func TestMoveSteamGameCancelledByCheck(t *testing.T) {
	f, lib2 := moverFixture(t)
	in, _ := FindSteamInstall(f.steam, 1326470)
	_, err := MoveSteamGame(in, lib2, nil, func() error { return errors.New("Steam was started during the move") })
	if err == nil {
		t.Fatal("the move should stop")
	}
	if _, err := os.Stat(in.Manifest); err != nil {
		t.Fatal("Steam's record must stay where it was")
	}
	if _, err := os.Stat(filepath.Join(in.Dir(), "Data", "level0")); err != nil {
		t.Fatal("the game must stay where it was")
	}
	for _, p := range []string{
		filepath.Join(lib2, "steamapps", "common", "Sons Of The Forest"),
		filepath.Join(lib2, "steamapps", "workshop", "content", "1326470"),
		filepath.Join(lib2, "steamapps", "appmanifest_1326470.acf"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s should be removed again", p)
		}
	}
}

func TestFindSteamInstallRefusesPartialInstall(t *testing.T) {
	f, _ := moverFixture(t)
	write(t, filepath.Join(f.steam, "steamapps", "appmanifest_1326470.acf"), `"AppState"
{
	"appid"		"1326470"
	"name"		"Sons Of The Forest"
	"StateFlags"		"1026"
	"installdir"		"Sons Of The Forest"
}
`)
	if _, err := FindSteamInstall(f.steam, 1326470); err == nil {
		t.Fatal("a game Steam is still downloading must not be moved")
	}
	if _, err := FindSteamInstall(f.steam, 730); err == nil {
		t.Fatal("a game that isn't installed can't be found")
	}
}

func TestRetargetPaths(t *testing.T) {
	f, lib2 := moverFixture(t)
	from := filepath.Join(f.steam, "steamapps", "common", "Sons Of The Forest")
	to := filepath.Join(lib2, "steamapps", "common", "Sons Of The Forest")
	g := &Game{Paths: []PathEntry{
		{ID: "cfg", Path: "{STEAM}/steamapps/common/Sons Of The Forest/Config"},
		{ID: "saves", Path: "%APPDATA%/Forest"},
	}}
	if !f.env.RetargetPaths(g, from, to) {
		t.Fatal("the path inside the game folder should change")
	}
	if !samePath(g.Paths[0].Path, filepath.Join(to, "Config")) {
		t.Fatalf("got %s", g.Paths[0].Path)
	}
	if g.Paths[1].Path != "%APPDATA%/Forest" {
		t.Fatalf("paths outside the game folder stay as they are, got %s", g.Paths[1].Path)
	}
	if f.env.RetargetPaths(g, from, to) {
		t.Fatal("nothing left to change")
	}
}
