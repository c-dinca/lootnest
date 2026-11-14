package core

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCopyFilesFolderAndZip(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "copy", "slot"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "copy", "slot", "save1.sav"), []byte("12345"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "copy", "settings.ini"), []byte("x"), 0o644))
	old := time.Now().Add(-time.Hour)
	must(os.Chtimes(filepath.Join(dir, "copy", "settings.ini"), old, old))
	files, n, err := CopyFiles(filepath.Join(dir, "copy"), 10)
	if err != nil || n != 2 || files[0].Path != "slot/save1.sav" || files[0].Size != 5 || files[1].Path != "settings.ini" {
		t.Fatalf("folder: %+v %d %v", files, n, err)
	}
	if files, n, _ := CopyFiles(filepath.Join(dir, "copy"), 1); len(files) != 1 || n != 2 {
		t.Fatalf("at most max files, with the count of all: %d %d", len(files), n)
	}
	zp := filepath.Join(dir, "v.zip")
	f, err := os.Create(zp)
	must(err)
	zw := zip.NewWriter(f)
	w, _ := zw.Create("loc0/a.sav")
	_, _ = w.Write([]byte("abc"))
	_, _ = zw.Create("loc0/")
	must(zw.Close())
	must(f.Close())
	files, n, err = CopyFiles(zp, 10)
	if err != nil || n != 1 || files[0].Path != "loc0/a.sav" || files[0].Size != 3 {
		t.Fatalf("zip: %+v %d %v", files, n, err)
	}
	if _, _, err := CopyFiles(filepath.Join(dir, "gone"), 10); err == nil {
		t.Fatal("a missing copy is an error")
	}
}

func TestCopyNames(t *testing.T) {
	dir := t.TempDir()
	if err := SetCopyName(dir, "game:bg3", "swap", "20261114-101500", "  Before   the boss "); err != nil {
		t.Fatal(err)
	}
	names := LoadCopyNames(dir)
	if CopyNameOf(names, "game:bg3", "swap", "20261114-101500") != "Before the boss" {
		t.Fatalf("names %v", names)
	}
	if err := SetCopyName(dir, "game:bg3", "point", "x", "a"); err == nil {
		t.Fatal("save points are named elsewhere")
	}
	if err := SetCopyName(dir, "game:bg3", "net", "../x", "a"); err == nil {
		t.Fatal("an id with a path in it is refused")
	}
	long := ""
	for i := 0; i < 61; i++ {
		long += "a"
	}
	if err := SetCopyName(dir, "game:bg3", "net", "v1", long); err == nil {
		t.Fatal("a name over 60 characters is refused")
	}
	if err := SetCopyName(dir, "game:bg3", "swap", "20261114-101500", ""); err != nil || len(LoadCopyNames(dir)) != 0 {
		t.Fatalf("an empty name takes it away: %v %v", err, LoadCopyNames(dir))
	}
}
