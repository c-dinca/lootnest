package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDecideSync(t *testing.T) {
	m := func(h string) *SyncMeta { return &SyncMeta{Hash: h} }
	rec := SyncRecord{Local: "a", Cloud: "a"}
	for name, c := range map[string]struct {
		local string
		cloud *SyncMeta
		rec   SyncRecord
		seen  bool
		want  SyncAction
	}{
		"same":                  {"a", m("a"), rec, true, SyncNothing},
		"nothing anywhere":      {"", nil, SyncRecord{}, false, SyncNothing},
		"first upload":          {"a", nil, SyncRecord{}, false, SyncUp},
		"nothing here yet":      {"", m("b"), SyncRecord{}, false, SyncDown},
		"played here":           {"b", m("a"), rec, true, SyncUp},
		"played on the other":   {"a", m("b"), rec, true, SyncDown},
		"played on both":        {"b", m("c"), rec, true, SyncConflict},
		"both, never synced":    {"b", m("c"), SyncRecord{}, false, SyncConflict},
		"both changed the same": {"c", m("c"), rec, true, SyncNothing},
	} {
		if got := DecideSync(c.local, c.cloud, c.rec, c.seen); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestCloudSyncTwoPCs(t *testing.T) {
	for _, enc := range []bool{false, true} {
		root := t.TempDir()
		var key []byte
		if enc {
			k, err := CloudKey(root, "cloud password")
			if err != nil {
				t.Fatal(err)
			}
			key = k
			// The other PC makes the same key from the same password...
			k2, err := CloudKey(root, "cloud password")
			if err != nil || string(k2) != string(k) || !CloudKeyMatches(root, k2) {
				t.Fatalf("same key on both PCs: %v", err)
			}
			// ...and another password is refused.
			if _, err := CloudKey(root, "something else"); err == nil {
				t.Fatal("a different password is refused")
			}
		}
		a := &Cloud{Root: root, Key: key, Computer: "PC-A"}
		b := &Cloud{Root: root, Key: key, Computer: "PC-B"}
		p := &Player{ID: "p1", Name: "Ana", Color: "#112233"}
		// The same Steam game, added on each PC with its own ids.
		gA := &Game{ID: "ga", Name: "Game", Launch: Launch{Type: "steam", AppID: 10}, Paths: []PathEntry{{ID: "x1", Path: "%APPDATA%\\Game"}, {ID: "x2", Path: "{DOCUMENTS}\\Game\\settings.ini"}}}
		gB := &Game{ID: "gb", Name: "Game", Launch: Launch{Type: "steam", AppID: 10}, Paths: []PathEntry{{ID: "y2", Path: "{DOCUMENTS}\\Game\\settings.ini"}, {ID: "y1", Path: "%APPDATA%\\Game"}}}
		if GameSyncKey(gA) != GameSyncKey(gB) {
			t.Fatal("one key on both PCs")
		}
		dA, dB := t.TempDir(), t.TempDir()
		dirA := func(pe PathEntry) string { return filepath.Join(dA, pe.ID) }
		dirB := func(pe PathEntry) string { return filepath.Join(dB, pe.ID) }
		write(t, filepath.Join(dA, "x1", "slot1.sav"), "level 7")
		write(t, filepath.Join(dA, "x2"), "fov=90")

		hA, err := SyncHash(gA, dirA)
		if err != nil || hA == "" {
			t.Fatal(hA, err)
		}
		if DecideSync(hA, a.Meta("p1", GameSyncKey(gA)), SyncRecord{}, false) != SyncUp {
			t.Fatal("PC A uploads first")
		}
		meta, err := a.Upload("p1", p, gA, dirA, hA, 1)
		if err != nil || meta.Computer != "PC-A" || meta.Encrypted != enc {
			t.Fatalf("%+v %v", meta, err)
		}
		if ps := b.Players(); len(ps) != 1 || ps[0].Name != "Ana" {
			t.Fatalf("%+v", ps)
		}

		// PC B has nothing yet: it downloads, into its own path ids.
		hB, _ := SyncHash(gB, dirB)
		cm := b.Meta("p1", GameSyncKey(gB))
		if DecideSync(hB, cm, SyncRecord{}, false) != SyncDown {
			t.Fatal("PC B downloads")
		}
		skipped, err := b.Download("p1", gB, dirB, filepath.Join(t.TempDir(), "tmp"))
		if err != nil || skipped != 0 {
			t.Fatal(skipped, err)
		}
		if read(t, filepath.Join(dB, "y1", "slot1.sav")) != "level 7" || read(t, filepath.Join(dB, "y2")) != "fov=90" {
			t.Fatal("files land in PC B's save locations")
		}
		hB, _ = SyncHash(gB, dirB)
		if hB != hA {
			t.Fatal("the same files have the same hash on both PCs")
		}
		recB := SyncRecord{Local: hB, Cloud: cm.Hash}

		// Played on B: up. Then A, which hasn't played since, comes down.
		write(t, filepath.Join(dB, "y1", "slot1.sav"), "level 8")
		hB2, _ := SyncHash(gB, dirB)
		if DecideSync(hB2, b.Meta("p1", GameSyncKey(gB)), recB, true) != SyncUp {
			t.Fatal("B uploads its new save")
		}
		if _, err := b.Upload("p1", p, gB, dirB, hB2, 2); err != nil {
			t.Fatal(err)
		}
		recA := SyncRecord{Local: hA, Cloud: hA}
		if DecideSync(hA, a.Meta("p1", GameSyncKey(gA)), recA, true) != SyncDown {
			t.Fatal("A downloads B's save")
		}
		if _, err := a.Download("p1", gA, dirA, filepath.Join(t.TempDir(), "tmp")); err != nil {
			t.Fatal(err)
		}
		if read(t, filepath.Join(dA, "x1", "slot1.sav")) != "level 8" {
			t.Fatal("A has level 8")
		}
		// Both played without syncing: they ask.
		write(t, filepath.Join(dA, "x1", "slot1.sav"), "level 9 on A")
		hA3, _ := SyncHash(gA, dirA)
		recA = SyncRecord{Local: hB2, Cloud: hB2}
		write(t, filepath.Join(dB, "y1", "slot1.sav"), "level 9 on B")
		hB3, _ := SyncHash(gB, dirB)
		_, _ = b.Upload("p1", p, gB, dirB, hB3, 3)
		if DecideSync(hA3, a.Meta("p1", GameSyncKey(gA)), recA, true) != SyncConflict {
			t.Fatal("both changed: conflict")
		}
		// Without the key, an encrypted cloud copy can't be read.
		if enc {
			nokey := &Cloud{Root: root, Computer: "PC-C"}
			if _, err := nokey.Download("p1", gB, dirB, filepath.Join(t.TempDir(), "tmp")); err == nil {
				t.Fatal("no key, no download")
			}
		}
	}
}

func TestLinkPlayers(t *testing.T) {
	st := SyncState{Records: map[string]map[string]SyncRecord{}, Links: map[string]string{}}
	here := []Player{{ID: "local1", Name: "Ana"}, {ID: "local2", Name: "Dan"}, {ID: "p9", Name: "Eve"}}
	cloud := []CloudPlayer{{ID: "p1", Name: "ana"}, {ID: "p9", Name: "Eve"}, {ID: "p5", Name: "Zed"}}
	if !LinkPlayers(&st, here, cloud) || st.CloudPlayerID("local1") != "p1" || st.CloudPlayerID("local2") != "local2" || st.CloudPlayerID("p9") != "p9" {
		t.Fatalf("%+v", st.Links)
	}
	if LinkPlayers(&st, here, cloud) {
		t.Fatal("nothing more to link")
	}
}

func TestSafeSyncName(t *testing.T) {
	for n, ok := range map[string]bool{"paths.json": true, "p/0": true, "p/1/a/b.sav": true, "p/x/a": false, "p/01/a": false,
		"p/1/../../x": false, "../p/1": false, "other.json": false, "p/1/CON": false, `p/1/a\b`: false} {
		if safeSyncName(n) != ok {
			t.Errorf("safeSyncName(%q) = %v", n, !ok)
		}
	}
}

func TestSavesChanged(t *testing.T) {
	root := t.TempDir()
	dir := func(pe PathEntry) string { return filepath.Join(root, pe.ID) }
	g := &Game{Paths: []PathEntry{{ID: "a"}, {ID: "b"}, {ID: "gone"}}}
	if got := SavesChanged(g, dir); !got.IsZero() {
		t.Fatalf("nothing there: got %v", got)
	}
	old, newer := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), time.Date(2026, 3, 2, 3, 4, 5, 0, time.UTC)
	write(t, filepath.Join(root, "a", "slot1.sav"), "x")
	write(t, filepath.Join(root, "b"), "y")
	for _, p := range []string{filepath.Join(root, "a"), filepath.Join(root, "a", "slot1.sav")} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(filepath.Join(root, "b"), newer, newer); err != nil {
		t.Fatal(err)
	}
	if got := SavesChanged(g, dir); !got.Equal(newer) {
		t.Fatalf("got %v, want %v", got, newer)
	}
	// A copy keeps the files' times; its new folder's time doesn't count.
	if err := Mirror(filepath.Join(root, "a"), filepath.Join(root, "c")); err != nil {
		t.Fatal(err)
	}
	if got := SavesChanged(&Game{Paths: []PathEntry{{ID: "c"}}}, dir); !got.Equal(old) {
		t.Fatalf("copy: got %v, want %v", got, old)
	}
}
