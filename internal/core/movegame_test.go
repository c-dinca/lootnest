package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func moveEnv(t *testing.T) (*Env, string) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "ana")
	data := filepath.Join(home, "AppData", "Roaming", "Lootnest")
	return &Env{Home: home, DataDir: data, Getenv: func(k string) string { return map[string]string{"USERPROFILE": home}[k] }}, root
}

func TestMoveGamePlanRefuses(t *testing.T) {
	e, root := moveEnv(t)
	game := filepath.Join(root, "Games", "Hollow Tale")
	write(t, filepath.Join(game, "HollowTale.exe"), "1234567890")
	write(t, filepath.Join(root, "E", "Games", "Taken", "x.txt"), "x")
	dst := filepath.Join(root, "E", "Games")

	p, err := e.MoveGamePlan("Hollow Tale", "folder", game, dst, false, -1)
	if err != nil || p.To != filepath.Join(dst, "Hollow Tale") || p.Size != 10 || p.Files != 1 {
		t.Fatalf("plan %+v %v", p, err)
	}
	taken := filepath.Join(root, "Games", "Taken")
	write(t, filepath.Join(taken, "Taken.exe"), "x")

	for _, c := range []struct {
		name, source, src, dst string
		running                bool
		free                   int64
		want                   string
	}{
		{"steam", "steam", game, dst, false, -1, "Its store keeps track"},
		{"hydra", "hydra", game, dst, false, -1, "Its store keeps track"},
		{"shortcut", "shortcut", game, dst, false, -1, "shortcut"},
		{"running", "folder", game, dst, true, -1, "is running"},
		{"no room", "added", game, dst, false, 10, "isn't enough room"},
		{"inside the game", "folder", game, filepath.Join(game, "moved"), false, -1, "inside the game's own folder"},
		{"the game itself", "folder", game, game, false, -1, "inside the game's own folder"},
		{"a drive", "folder", filepath.VolumeName(root) + string(filepath.Separator), dst, false, -1, "more than one game"},
		{"a folder of games", "folder", filepath.Join(root, "Games"), dst, false, -1, "more than one game"},
		{"already there", "folder", taken, dst, false, -1, "already there"},
		{"into the data folder", "folder", game, e.DataDir, false, -1, "data folder"},
		{"nowhere", "folder", game, "", false, -1, "Choose where"},
	} {
		if _, err := e.MoveGamePlan("Hollow Tale", c.source, c.src, c.dst, c.running, c.free); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	// Exactly the game and a tenth more is enough.
	if _, err := e.MoveGamePlan("Hollow Tale", "folder", game, dst, false, 11); err != nil {
		t.Fatalf("room for it: %v", err)
	}
}

func TestMoveGameCopiesAndChecks(t *testing.T) {
	e, root := moveEnv(t)
	game := filepath.Join(root, "Games", "Hollow Tale")
	write(t, filepath.Join(game, "HollowTale.exe"), "exe")
	write(t, filepath.Join(game, "saves", "slot1.sav"), "slot one")
	write(t, filepath.Join(game, "data", "pak0.pak"), strings.Repeat("p", 3<<20))
	dst := filepath.Join(root, "E", "Games")

	p, err := e.MoveGamePlan("Hollow Tale", "folder", game, dst, false, -1)
	if err != nil {
		t.Fatal(err)
	}
	var last int64
	if err := MoveGame(p, func(n int64) { last = n }, nil); err != nil {
		t.Fatal(err)
	}
	if last != p.Size {
		t.Fatalf("progress ended at %d of %d", last, p.Size)
	}
	if b, _ := os.ReadFile(filepath.Join(p.To, "saves", "slot1.sav")); string(b) != "slot one" {
		t.Fatalf("copied save %q", b)
	}
	if err := sameTree(game, p.To); err != nil {
		t.Fatal(err)
	}
	if !isFile(filepath.Join(game, "HollowTale.exe")) {
		t.Fatal("the original must stay until the app sends it to the Recycle Bin")
	}
	if _, err := os.Stat(p.To + partSuffix); !os.IsNotExist(err) {
		t.Fatal("the part folder is left")
	}

	// The game changed while it was copied: the copy doesn't match what was
	// counted, and is removed.
	_ = os.RemoveAll(p.To)
	orig := filepath.Join(game, "saves", "slot1.sav")
	n := 0
	if err := MoveGame(p, func(int64) {
		if n++; n == 1 {
			_ = os.WriteFile(orig, []byte("slot two, longer"), 0o644) // the game wrote while it was copied
		}
	}, nil); err == nil {
		t.Fatal("a game that changed while it was copied moved anyway")
	}
	if _, err := os.Stat(p.To); !os.IsNotExist(err) {
		t.Fatal("a failed copy is left at the destination")
	}
	write(t, orig, "slot one")

	// Cancelled: nothing at the destination, the game where it was.
	if err := MoveGame(p, nil, func() bool { return true }); err != ErrMoveCancelled {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := os.Stat(p.To); !os.IsNotExist(err) {
		t.Fatal("a cancelled copy is left")
	}
	if _, err := os.Stat(p.To + partSuffix); !os.IsNotExist(err) {
		t.Fatal("a cancelled part folder is left")
	}
	if !isFile(filepath.Join(game, "saves", "slot1.sav")) {
		t.Fatal("cancelling touched the game")
	}
}
