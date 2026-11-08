package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDirSize(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Games", "Hollow Tale")
	write(t, filepath.Join(game, "HollowTale.exe"), "12345")
	write(t, filepath.Join(game, "data", "pak0.pak"), "1234567890")
	outside := filepath.Join(root, "Elsewhere")
	write(t, filepath.Join(outside, "big.bin"), "a lot of bytes that aren't the game's")
	if err := os.Symlink(outside, filepath.Join(game, "linked")); err != nil {
		t.Logf("no symlink here: %v", err)
	}
	b, n, err := DirSize(game)
	if err != nil || b != 15 || n != 2 {
		t.Fatalf("DirSize = %d bytes, %d files, %v", b, n, err)
	}
	link := filepath.Join(root, "Moved")
	if err := os.Symlink(game, link); err == nil {
		if b, n, err := DirSize(link); err != nil || b != 15 || n != 2 {
			t.Fatalf("DirSize of a link to the game = %d bytes, %d files, %v", b, n, err)
		}
	}
	if _, _, err := DirSize(filepath.Join(root, "missing")); err == nil {
		t.Fatal("a missing folder has no size")
	}
}

func TestSizeCache(t *testing.T) {
	root := t.TempDir()
	s := &Store{Dir: filepath.Join(root, "data")}
	game := filepath.Join(root, "Games", "Hollow Tale")
	write(t, filepath.Join(game, "HollowTale.exe"), "12345")
	now := time.Now()

	c := s.LoadSizes()
	d, err := c.Size(game, now)
	if err != nil || d.Bytes != 5 || d.Files != 1 {
		t.Fatalf("first count %+v %v", d, err)
	}
	if err := s.SaveSizes(c, now); err != nil {
		t.Fatal(err)
	}

	// A file deep inside changes: the kept count stands until a week is up.
	write(t, filepath.Join(game, "HollowTale.exe"), "1234567890")
	st, _ := os.Stat(game)
	_ = os.Chtimes(game, st.ModTime(), st.ModTime())
	c = s.LoadSizes()
	if d, _ := c.Size(game, now.Add(time.Hour)); d.Bytes != 5 {
		t.Fatalf("kept count %+v", d)
	}
	if d, _ := c.Size(game, now.Add(8*24*time.Hour)); d.Bytes != 10 {
		t.Fatalf("a week later %+v", d)
	}

	// A new file in the folder changes its time: counted again.
	write(t, filepath.Join(game, "readme.txt"), "hi")
	later := st.ModTime().Add(time.Minute)
	_ = os.Chtimes(game, later, later)
	if d, _ := c.Size(game, now.Add(8*24*time.Hour)); d.Bytes != 12 {
		t.Fatalf("after a new file %+v", d)
	}

	// Gone: 0, and forgotten.
	if err := os.RemoveAll(game); err != nil {
		t.Fatal(err)
	}
	if d, _ := c.Size(game, now); d.Bytes != 0 || len(c.Dirs) != 0 {
		t.Fatalf("gone %+v %v", d, c.Dirs)
	}

	// Counts older than a week aren't written.
	c.Dirs["old"] = SizedDir{Bytes: 1, At: now.Add(-8 * 24 * time.Hour)}
	if err := s.SaveSizes(c, now); err != nil {
		t.Fatal(err)
	}
	if got := s.LoadSizes(); len(got.Dirs) != 0 {
		t.Fatalf("old counts kept: %v", got.Dirs)
	}
}

func TestProfileSize(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	write(t, filepath.Join(s.ProfileDir("ana", "g1"), "p1", "slot1.sav"), "123")
	write(t, filepath.Join(s.ProfileDir("ana", "g1"), "p2", "settings.ini"), "4567")
	if n := s.ProfileSize("ana", "g1"); n != 7 {
		t.Fatalf("ProfileSize = %d", n)
	}
	if n := s.ProfileSize("ben", "g1"); n != 0 {
		t.Fatalf("no profile: %d", n)
	}
}
