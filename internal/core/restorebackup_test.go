package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreBackup(t *testing.T) {
	f := newFixture(t)
	write(t, f.optFile(), "sens=1.2")
	// p1's files are in; p2 starts fresh: the swap's copy holds p1's.
	if err := f.eng.Activate(f.cfg, "cs2", "p2", ActivateOptions{FirstOwner: "p1", NewProfile: NewFresh}); err != nil {
		t.Fatal(err)
	}
	bs := f.eng.Backups("cs2")
	if len(bs) == 0 || bs[0].Owner != "p1" {
		t.Fatalf("backups %+v", bs)
	}
	id := BackupID(bs[0])

	// p1's profile is damaged somehow; putting the copy back restores it,
	// keeping the damaged one as a copy.
	write(t, filepath.Join(f.eng.Store.ProfileDir("p1", "cs2"), "opt"), "broken")
	owner, where, err := f.eng.RestoreBackup(f.cfg, "cs2", id)
	if err != nil || owner != "p1" || where != "profile" {
		t.Fatalf("restore: %s %s %v", owner, where, err)
	}
	if b, _ := os.ReadFile(filepath.Join(f.eng.Store.ProfileDir("p1", "cs2"), "opt")); string(b) != "sens=1.2" {
		t.Fatalf("p1's profile: %q", b)
	}
	keptBroken := false
	for _, b := range f.eng.Backups("cs2") {
		if raw, err := os.ReadFile(filepath.Join(b.Dir, "opt")); err == nil && string(raw) == "broken" {
			keptBroken = true
		}
	}
	if !keptBroken {
		t.Fatal("the profile as it was must be kept as a copy")
	}
	// Back to p1: the restored profile comes in.
	if err := f.eng.Activate(f.cfg, "cs2", "p1", ActivateOptions{}); err != nil {
		t.Fatal(err)
	}
	if read(t, f.optFile()) != "sens=1.2" {
		t.Fatalf("live: %q", read(t, f.optFile()))
	}
	// With p1 loaded, putting a copy back goes into place, after a copy.
	write(t, f.optFile(), "sens=oops")
	if _, where, err := f.eng.RestoreBackup(f.cfg, "cs2", id); err != nil || where != "live" {
		t.Fatalf("live restore: %s %v", where, err)
	}
	if read(t, f.optFile()) != "sens=1.2" {
		t.Fatalf("live after restore: %q", read(t, f.optFile()))
	}
	if _, _, err := f.eng.RestoreBackup(f.cfg, "cs2", "../x"); err == nil {
		t.Fatal("a bad id must be refused")
	}
}
