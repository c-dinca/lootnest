package app

import (
	"path/filepath"
	"testing"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// A game started through a launcher runs as another program from its
// folder: that one is the game, a helper only when nothing else is there.
func TestInGameFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Red Dead Redemption 2")
	helpers := helperNames(core.LaunchOptions{StartBefore: []core.Program{{Path: filepath.Join(dir, "Launcher.exe")}}})
	apps := []plat.RunningApp{
		{Name: "Discord.exe", Path: filepath.Join(filepath.Dir(dir), "Discord", "Discord.exe")},
		{Name: "Launcher.exe", Path: filepath.Join(dir, "Launcher.exe")},
		{Name: "RDR2.exe", Path: filepath.Join(dir, "RDR2.exe")},
	}
	if n, helper := inGameFolder(apps, dir, helpers); n != "rdr2" || helper {
		t.Fatalf("the game, not its launcher: %q %v", n, helper)
	}
	if n, helper := inGameFolder(apps[:2], dir, helpers); n != "launcher" || !helper {
		t.Fatalf("only the launcher so far: %q %v", n, helper)
	}
	if n, _ := inGameFolder(apps[:1], dir, helpers); n != "" {
		t.Fatalf("nothing of the game: %q", n)
	}
	if n, _ := inGameFolder(apps, "", helpers); n != "" {
		t.Fatal("no folder, no guess")
	}
	// A sibling folder whose name starts the same isn't inside.
	if underDir(filepath.Join(dir+" Mods", "x.exe"), dir) || !underDir(filepath.Join(dir, "x64", "Game.exe"), dir) {
		t.Fatal("underDir")
	}
}
