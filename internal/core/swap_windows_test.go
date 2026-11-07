//go:build windows

package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// These run on Windows itself (CI's windows job): what a real PC does that a
// Linux test can't show. A game holding its save open, saves marked
// read-only, and paths longer than Windows' old 260-character limit.

// startTwoProfiles gives p1 and p2 their own files for the fixture's game,
// with p1's loaded.
func startTwoProfiles(t *testing.T, f *fixture) {
	t.Helper()
	write(t, f.cfgFile(), "res=2560x1440")
	write(t, f.optFile(), "sens=1.2")
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{FirstOwner: "p1", NewProfile: NewFresh}); err != nil {
		t.Fatal(err)
	}
	write(t, f.optFile(), "sens=3.0")
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.optFile()) != "sens=1.2" {
		t.Fatalf("p1's files should be in: %q", read(t, f.optFile()))
	}
}

// A game that still holds its save open (no sharing at all, as some do):
// the swap must stop before it changes anything, and lose nothing.
func TestSwapWithSaveHeldOpen(t *testing.T) {
	f := newFixture(t)
	startTwoProfiles(t, f)
	p, _ := windows.UTF16PtrFromString(f.optFile())
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{})
	windows.CloseHandle(h)
	if err == nil {
		// If Windows let it through, the swap must at least be complete.
		if read(t, f.optFile()) != "sens=3.0" {
			t.Fatalf("swap went through but p2's files aren't in: %q", read(t, f.optFile()))
		}
	} else {
		if read(t, f.optFile()) != "sens=1.2" || read(t, f.cfgFile()) != "res=2560x1440" {
			t.Fatalf("a refused swap changed the live files: %q %q", read(t, f.optFile()), read(t, f.cfgFile()))
		}
	}
	// Once the game lets go, swapping works and both profiles are whole.
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.optFile()) != "sens=3.0" {
		t.Fatalf("p2's files: %q", read(t, f.optFile()))
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.optFile()) != "sens=1.2" || read(t, f.cfgFile()) != "res=2560x1440" {
		t.Fatalf("p1's files after the lock: %q %q", read(t, f.optFile()), read(t, f.cfgFile()))
	}
	noLeftovers(t, f.root)
}

// Saves marked read-only (some games and cloud apps do it).
func TestSwapReadOnlySaves(t *testing.T) {
	f := newFixture(t)
	startTwoProfiles(t, f)
	if err := os.Chmod(f.optFile(), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{}); err != nil {
		t.Fatalf("read-only save stopped the swap: %v", err)
	}
	if read(t, f.optFile()) != "sens=3.0" {
		t.Fatalf("p2's files: %q", read(t, f.optFile()))
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.optFile()) != "sens=1.2" {
		t.Fatalf("p1's read-only save came back as %q", read(t, f.optFile()))
	}
	noLeftovers(t, f.root)
}

// A save folder deeper than 260 characters.
func TestSwapLongPaths(t *testing.T) {
	f := newFixture(t)
	deep := filepath.Join(f.appdata, "Game", strings.Repeat("a-very-long-folder-name-", 6), strings.Repeat("and-another-one-", 6), strings.Repeat("x", 60))
	if len(deep) < 270 {
		deep = filepath.Join(deep, strings.Repeat("y", 280-len(deep)))
	}
	g := f.cfg.Game("cs2")
	rel, _ := filepath.Rel(f.appdata, deep)
	g.Paths = append(g.Paths, PathEntry{ID: "deep", Path: "%APPDATA%/" + filepath.ToSlash(rel)})
	write(t, filepath.Join(deep, "slot1.sav"), "p1 deep")
	startTwoProfiles(t, f)
	if read(t, filepath.Join(deep, "slot1.sav")) != "p1 deep" {
		t.Fatalf("p1's deep save: %q", read(t, filepath.Join(deep, "slot1.sav")))
	}
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(deep, "slot1.sav"), "p2 deep")
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(deep, "slot1.sav")) != "p1 deep" {
		t.Fatalf("p1's deep save after a round trip: %q", read(t, filepath.Join(deep, "slot1.sav")))
	}
}

func noLeftovers(t *testing.T, root string) {
	t.Helper()
	_ = filepath.Walk(root, func(p string, _ os.FileInfo, _ error) error {
		if strings.HasSuffix(p, tmpSuffix) || strings.HasSuffix(p, oldSuffix) {
			t.Errorf("leftover %s", p)
		}
		return nil
	})
}
