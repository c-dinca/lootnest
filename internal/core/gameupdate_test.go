package core

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gameWithUpdate makes a game folder and an update for it, inside a folder
// of its own as updates often come.
func gameWithUpdate(t *testing.T) (game, upd string) {
	t.Helper()
	root := t.TempDir()
	game = filepath.Join(root, "Games", "Phantom Blade Zero")
	write(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe"), "old exe")
	write(t, filepath.Join(game, "ProjectPBZ", "Content", "Paks", "a.pak"), "old pak")
	write(t, filepath.Join(game, "ProjectPBZ", "Saved", "slot.sav"), "my save")
	upd = filepath.Join(root, "Downloads", "PBZ Update v1.0302256")
	write(t, filepath.Join(upd, "PBZ Update v1.0302256", "ProjectPBZ", "Binaries", "Win64", "PBZ.exe"), "new exe")
	write(t, filepath.Join(upd, "PBZ Update v1.0302256", "ProjectPBZ", "Content", "Paks", "b.pak"), "new pak")
	write(t, filepath.Join(upd, "PBZ Update v1.0302256", "README.txt"), "copy to the game folder")
	return game, upd
}

func TestPlanApplyUndoUpdate(t *testing.T) {
	game, upd := gameWithUpdate(t)
	p, err := PlanUpdate(game, upd)
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != "folder" || p.Skipped != "PBZ Update v1.0302256" || p.Added != 1 || p.Replace != 1 || len(p.Readme) != 1 {
		t.Fatalf("plan = %+v", p)
	}
	if len(p.Where) != 1 || p.Where[0].Path != "ProjectPBZ" || p.Where[0].Files != 2 {
		t.Fatalf("where = %+v", p.Where)
	}
	keep := filepath.Join(t.TempDir(), "20261107-180000")
	u, err := ApplyUpdate(game, p, keep, "Phantom Blade Zero", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe")) != "new exe" ||
		read(t, filepath.Join(game, "ProjectPBZ", "Content", "Paks", "b.pak")) != "new pak" ||
		read(t, filepath.Join(game, "ProjectPBZ", "Saved", "slot.sav")) != "my save" {
		t.Fatal("the update's files should be in place, and the save untouched")
	}
	if exists(filepath.Join(game, "README.txt")) || exists(filepath.Join(game, "PBZ Update v1.0302256")) {
		t.Fatal("notes and the update's own folder aren't copied")
	}
	if u.State != "applied" || len(GameUpdates(filepath.Dir(keep))) != 1 {
		t.Fatalf("record = %+v", u)
	}

	if _, err := UndoUpdate(game, keep); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe")) != "old exe" || exists(filepath.Join(game, "ProjectPBZ", "Content", "Paks", "b.pak")) {
		t.Fatal("undo should put the old exe back and take the new pak away")
	}
	if GameUpdates(filepath.Dir(keep))[0].State != "undone" {
		t.Fatal("the record should say undone")
	}
}

func TestUpdateFromZip(t *testing.T) {
	game, _ := gameWithUpdate(t)
	zp := filepath.Join(t.TempDir(), "update.zip")
	makeZip(t, zp, map[string]string{
		`ProjectPBZ\Binaries\Win64\PBZ.exe`: "zip exe", // Windows zips use backslashes
		"ProjectPBZ/NewFolder/c.bin":        "new",
	})
	p, err := PlanUpdate(game, zp)
	if err != nil || p.Kind != "zip" || p.Skipped != "" || p.Added != 1 || p.Replace != 1 {
		t.Fatalf("plan = %+v, %v", p, err)
	}
	keep := filepath.Join(t.TempDir(), "u1")
	if _, err := ApplyUpdate(game, p, keep, "PBZ", time.Now()); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe")) != "zip exe" {
		t.Fatal("the zip's exe should be in place")
	}
	if _, err := UndoUpdate(game, keep); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(game, "ProjectPBZ", "NewFolder")) {
		t.Fatal("undo should take away the folder the update made")
	}

	bad := filepath.Join(t.TempDir(), "bad.zip")
	makeZip(t, bad, map[string]string{"../../evil.dll": "x"})
	if _, err := PlanUpdate(game, bad); !errors.Is(err, errUnsafePath) {
		t.Fatalf("a path out of the game's folder must be refused, got %v", err)
	}
	if _, err := PlanUpdate(game, game); err == nil {
		t.Fatal("the game's own folder isn't an update")
	}
}

func TestFailedUpdatePutsEverythingBack(t *testing.T) {
	game, upd := gameWithUpdate(t)
	p, err := PlanUpdate(game, upd)
	if err != nil {
		t.Fatal(err)
	}
	// The second file's copy comes out damaged.
	afterCopy = func(dst string) {
		if strings.HasSuffix(dst, "b.pak"+tmpSuffix) {
			_ = os.WriteFile(dst, []byte("x"), 0o644)
		}
	}
	defer func() { afterCopy = nil }()
	keep := filepath.Join(t.TempDir(), "u1")
	if _, err := ApplyUpdate(game, p, keep, "PBZ", time.Now()); !errors.Is(err, ErrCopyMismatch) {
		t.Fatalf("want ErrCopyMismatch, got %v", err)
	}
	// p.Files is sorted: the exe went in first, then the pak failed.
	if read(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe")) != "old exe" ||
		exists(filepath.Join(game, "ProjectPBZ", "Content", "Paks", "b.pak")) ||
		exists(filepath.Join(game, "ProjectPBZ", "Content", "Paks", "b.pak"+tmpSuffix)) {
		t.Fatal("a failed update must leave the game as it was")
	}
	if u := GameUpdates(filepath.Dir(keep)); len(u) != 1 || u[0].State != "undone" {
		t.Fatalf("records = %+v", u)
	}
}

func makeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestInstallerUpdateCanBeUndone(t *testing.T) {
	game, _ := gameWithUpdate(t)
	full := filepath.Join(t.TempDir(), "full")
	if err := Mirror(game, full); err != nil {
		t.Fatal(err)
	}
	before, _ := SnapshotDir(game)
	// The installer changes the exe, adds a pak and removes an old one.
	time.Sleep(10 * time.Millisecond)
	write(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe"), "installed exe")
	write(t, filepath.Join(game, "ProjectPBZ", "Content", "Paks", "new", "c.pak"), "new pak")
	_ = os.Remove(filepath.Join(game, "ProjectPBZ", "Content", "Paks", "a.pak"))
	after, _ := SnapshotDir(game)
	added, changed, removed := DiffSnapshots(before, after)
	if len(added) != 1 || len(changed) != 1 || len(removed) != 1 {
		t.Fatalf("diff = %v %v %v", added, changed, removed)
	}
	keep := filepath.Join(t.TempDir(), "u1")
	u, err := RecordInstallerUpdate(game, full, keep, "PBZ", "PBZ-Update-v1.0.3-setup.exe", added, changed, removed, time.Now())
	if err != nil || u.NoUndo || u.To != "1.0.3" || !u.Verify {
		t.Fatalf("record = %+v, %v", u, err)
	}
	_ = os.RemoveAll(full) // the full copy isn't needed any more
	if _, err := UndoUpdate(game, keep); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(game, "ProjectPBZ", "Binaries", "Win64", "PBZ.exe")) != "old exe" ||
		read(t, filepath.Join(game, "ProjectPBZ", "Content", "Paks", "a.pak")) != "old pak" ||
		exists(filepath.Join(game, "ProjectPBZ", "Content", "Paks", "new")) ||
		read(t, filepath.Join(game, "ProjectPBZ", "Saved", "slot.sav")) != "my save" {
		t.Fatal("undo should put the folder back as it was, saves untouched")
	}

	// Without a full copy, it can't be undone, and says so.
	u2, _ := RecordInstallerUpdate(game, "", filepath.Join(t.TempDir(), "u2"), "PBZ", "setup.exe", nil, nil, nil, time.Now())
	if !u2.NoUndo {
		t.Fatal("no copy, no undo")
	}
	if _, err := UndoUpdate(game, u2.Dir); err == nil {
		t.Fatal("undo without a copy must be refused")
	}
}
