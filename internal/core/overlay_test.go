package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInGameAllowed(t *testing.T) {
	story := t.TempDir()
	os.MkdirAll(filepath.Join(story, "Binaries", "Win64"), 0o755)
	os.WriteFile(filepath.Join(story, "Binaries", "Win64", "Game.exe"), []byte("MZ"), 0o644)
	if ok, why := InGameAllowed(InGameFacts{Dirs: []string{story}, Name: "Hollow Tale"}); !ok {
		t.Fatalf("a story game was refused: %s", why)
	}
	// An anti-cheat three levels down, which AntiCheat alone doesn't see.
	ac := t.TempDir()
	os.MkdirAll(filepath.Join(ac, "Game", "Binaries", "Win64", "EasyAntiCheat"), 0o755)
	if AntiCheat(ac) != "" {
		t.Fatal("test setup: AntiCheat already sees it")
	}
	if ok, why := InGameAllowed(InGameFacts{Dirs: []string{ac}, Name: "Some Game"}); ok || why == "" {
		t.Fatal("a game with Easy Anti-Cheat deep in its folder was allowed")
	}
	be := t.TempDir()
	os.WriteFile(filepath.Join(be, "BEClient_x64.dll"), nil, 0o644)
	if ok, _ := InGameAllowed(InGameFacts{Dirs: []string{be}}); ok {
		t.Fatal("BattlEye's client was missed")
	}
	for _, c := range []struct {
		f    InGameFacts
		want bool
	}{
		{InGameFacts{Dirs: []string{story}, AppID: 271590, Name: "Some name"}, false},   // GTA V by id
		{InGameFacts{Dirs: []string{story}, Name: "Red Dead Redemption 2"}, false},      // by name
		{InGameFacts{Dirs: []string{story}, Name: "Call of Duty®: Black Ops 6"}, false}, // by words
		{InGameFacts{Dirs: []string{story}, Name: "Rusty Lake Hotel"}, true},            // not "rust"
		{InGameFacts{Dirs: []string{story}, Name: "Hollow", SteamOnline: "Valve Anti-Cheat"}, false},
		{InGameFacts{Dirs: []string{story}, Name: "Hollow", Running: map[string]bool{"beservice": true}}, false},
		{InGameFacts{Dirs: []string{story}, Name: "Cyberpunk 2077", Running: map[string]bool{"explorer": true}}, true},
	} {
		if ok, why := InGameAllowed(c.f); ok != c.want {
			t.Errorf("%+v: allowed %v (%s), want %v", c.f.Name, ok, why, c.want)
		}
	}
}

func TestOnlineByCategories(t *testing.T) {
	for _, c := range []struct {
		cats map[int]bool
		want string
	}{
		{map[int]bool{2: true, 8: true}, "Valve Anti-Cheat"},
		{map[int]bool{1: true, 36: true}, "online only"},
		{map[int]bool{1: true, 2: true}, ""}, // a story with co-op
		{nil, ""},
	} {
		if got := onlineByCategories(c.cats); got != c.want {
			t.Errorf("%v: %q, want %q", c.cats, got, c.want)
		}
	}
}

func TestOverlayDLLName(t *testing.T) {
	if n := OverlayDLLName("31230816656d2c2e31890a42"); n != "lootnest-overlay64-31230816656d.dll" {
		t.Fatal(n)
	}
	dir := t.TempDir()
	for _, n := range []string{"lootnest-overlay64-aaa.dll", "lootnest-overlay64-bbb.dll", "other.dll"} {
		os.WriteFile(filepath.Join(dir, n), nil, 0o644)
	}
	PruneOverlayDLLs(dir, "lootnest-overlay64-bbb.dll")
	es, _ := os.ReadDir(dir)
	if len(es) != 2 {
		t.Fatalf("%d files left", len(es))
	}
	if FPSCornerIndex("br") != 3 || FPSCornerIndex("x") != 0 {
		t.Fatal("corners")
	}
}
