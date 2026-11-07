package core

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A data folder with two players, one game, and the game's live save.
func backupFixture(t *testing.T) (data, live string, cfg *Config) {
	data, live = t.TempDir(), t.TempDir()
	cfg = DefaultConfig()
	cfg.Players = []Player{{ID: "p1", Name: "Ana", Color: "#112233"}, {ID: "p2", Name: "Dan", Color: "#445566"}}
	cfg.Games = []Game{{ID: "g1", Name: "Game", ProcessName: "game", Launch: Launch{Type: "steam", AppID: 10}, Paths: []PathEntry{{ID: "saves", Path: live}}}}
	cfg.Settings.BackupsToKeep = 7
	s := &Store{Dir: data}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(data, "shelf.json"), `{"p1":{"game:g1":{"status":"playing"}}}`)
	write(t, filepath.Join(data, "profiles", "p2", "g1", "saves", "slot1.sav"), "dan's save")
	write(t, filepath.Join(data, "profiles", "p1", "g1", "saves", "slot1.sav"), "ana's old save")
	write(t, filepath.Join(data, "avatars", "p1.png"), "png")
	write(t, filepath.Join(data, "covers", "big.jpg"), "not backed up")
	write(t, filepath.Join(live, "slot1.sav"), "ana's newest save")
	_ = s.SetActive("g1", "p1")
	return data, live, cfg
}

func TestBackupRoundTrip(t *testing.T) {
	for _, enc := range []bool{false, true} {
		data, live, _ := backupFixture(t)
		out := t.TempDir()
		opt := BackupOptions{}
		if enc {
			salt := NewBackupSalt()
			key, err := BackupKey("correct horse", salt, BackupKDFIters)
			if err != nil {
				t.Fatal(err)
			}
			opt = BackupOptions{Key: key, Salt: salt, Iters: BackupKDFIters}
		}
		file := filepath.Join(out, BackupFileName("MY-PC", time.Now(), enc))
		src := []LiveSource{{LiveEntry{GameID: "g1", EntryID: "saves", Owner: "p1"}, live}}
		n, err := WriteBackupFile(file, data, src, BackupManifest{Computer: "MY-PC", Created: time.Now()}, opt)
		if err != nil || n == 0 {
			t.Fatal(n, err)
		}
		if _, err := os.Stat(file + ".partial"); err == nil {
			t.Fatal("the .partial file is moved into place")
		}
		if isEnc, _ := IsEncryptedBackup(file); isEnc != enc {
			t.Fatalf("encrypted = %v", isEnc)
		}
		if enc {
			if _, err := OpenBackup(file, filepath.Join(out, "x0"), "", nil); !errors.Is(err, ErrNeedPassword) {
				t.Fatalf("no password: %v", err)
			}
			if _, err := OpenBackup(file, filepath.Join(out, "x1"), "wrong", nil); !errors.Is(err, ErrBadPassword) {
				t.Fatalf("wrong password: %v", err)
			}
			if _, err := os.Stat(filepath.Join(out, "x1")); err == nil {
				t.Fatal("a failed open leaves nothing behind")
			}
		}
		ob, err := OpenBackup(file, filepath.Join(out, "open"), "correct horse", nil)
		if err != nil {
			t.Fatal(err)
		}
		if ob.Manifest.Computer != "MY-PC" || len(ob.Manifest.Live) != 1 {
			t.Fatalf("%+v", ob.Manifest)
		}
		if _, err := os.Stat(filepath.Join(ob.Dir, "covers")); err == nil {
			t.Fatal("covers aren't backed up")
		}

		// Restore into another PC's data folder.
		other := &Store{Dir: t.TempDir()}
		here := DefaultConfig()
		here.Settings.BackupsToKeep = 3
		here.Settings.SteamAccountID = "this pc"
		write(t, filepath.Join(other.Dir, "profiles", "p2", "g1", "saves", "slot1.sav"), "replaced")
		cfg, err := other.Restore(ob, here)
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.Players) != 2 || len(cfg.Games) != 1 || cfg.Settings.SteamAccountID != "this pc" || cfg.Settings.BackupsToKeep != 3 {
			t.Fatalf("players and games come from the backup, settings stay: %+v", cfg)
		}
		if got := read(t, filepath.Join(other.Dir, "profiles", "p1", "g1", "saves", "slot1.sav")); got != "ana's newest save" {
			t.Fatalf("the files in place go to their owner: %q", got)
		}
		if got := read(t, filepath.Join(other.Dir, "profiles", "p2", "g1", "saves", "slot1.sav")); got != "dan's save" {
			t.Fatalf("profiles are restored: %q", got)
		}
		if !strings.Contains(read(t, filepath.Join(other.Dir, "shelf.json")), "playing") || read(t, filepath.Join(other.Dir, "avatars", "p1.png")) != "png" {
			t.Fatal("shelf and pictures are restored")
		}
		if other.LoadState()["g1"] != "" {
			t.Fatal("after a restore nobody's files are known to be in place")
		}
	}
}

func TestBackupTampering(t *testing.T) {
	data, _, _ := backupFixture(t)
	out := t.TempDir()
	salt := NewBackupSalt()
	key, _ := BackupKey("pw", salt, BackupKDFIters)
	// Big enough for several chunks.
	write(t, filepath.Join(data, "profiles", "p1", "g1", "saves", "big.bin"), strings.Repeat("x", 3*backupChunk))
	f, err := os.Create(filepath.Join(data, "profiles", "p1", "g1", "saves", "rand.bin"))
	if err != nil {
		t.Fatal(err)
	}
	r := make([]byte, 3*backupChunk)
	_, _ = rand.Read(r)
	_, _ = f.Write(r)
	f.Close()
	file := filepath.Join(out, "b.lnbak")
	if _, err := WriteBackupFile(file, data, nil, BackupManifest{}, BackupOptions{Key: key, Salt: salt, Iters: BackupKDFIters}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(file)
	if len(b) < 2*backupChunk {
		t.Fatalf("expected several chunks, got %d bytes", len(b))
	}
	header := len(backupMagic) + 1 + 4 + backupSaltLen + backupPrefix
	cases := map[string][]byte{
		"cut at a chunk": b[:header+backupChunk+16],
		"cut short":      b[:len(b)-5],
		"a byte changed": func() []byte { c := bytes.Clone(b); c[header+100] ^= 1; return c }(),
		"chunks swapped": func() []byte {
			c := bytes.Clone(b)
			n := backupChunk + 16
			copy(c[header:header+n], b[header+n:header+2*n])
			copy(c[header+n:header+2*n], b[header:header+n])
			return c
		}(),
	}
	for name, c := range cases {
		p := filepath.Join(out, "t.lnbak")
		_ = os.WriteFile(p, c, 0o644)
		if _, err := OpenBackup(p, filepath.Join(out, "o-"+strings.ReplaceAll(name, " ", "")), "pw", nil); err == nil {
			t.Errorf("%s: opened", name)
		}
	}
	// The saved key opens it without the password.
	if _, err := OpenBackup(file, filepath.Join(out, "ok"), "", key); err != nil {
		t.Fatal(err)
	}
}

func TestBackupNames(t *testing.T) {
	for name, ok := range map[string]bool{
		"config.json":                    true,
		"profiles/p1/g1/saves/a.sav":     true,
		"live/g1/saves/a.sav":            true,
		"avatars/p1.png":                 true,
		"../evil":                        false,
		"profiles/../../evil":            false,
		"/etc/passwd":                    false,
		"C:/Windows/x":                   false,
		`profiles\p1\g1\x`:               false,
		"profiles/P1!/g1/x":              false,
		"other.json":                     false,
		"avatars/sub/x.png":              false,
		"profiles/p1/g1/saves/CON":       false,
		"profiles/p1/g1/saves/nul.txt":   false,
		"profiles/p1/g1/saves/trailing.": false,
		"profiles/p1/g1":                 false,
		"live/g1":                        false,
	} {
		if safeBackupName(name) != ok {
			t.Errorf("safeBackupName(%q) = %v", name, !ok)
		}
	}
	// A zip with a path outside is refused, and nothing is written.
	dir := t.TempDir()
	p := filepath.Join(dir, "evil.zip")
	f, _ := os.Create(p)
	zw := zip.NewWriter(f)
	w, _ := zw.Create(manifestName)
	_, _ = w.Write([]byte(`{"format":1}`))
	w, _ = zw.Create("../../outside.txt")
	_, _ = w.Write([]byte("x"))
	zw.Close()
	f.Close()
	if _, err := OpenBackup(p, filepath.Join(dir, "o"), "", nil); err == nil {
		t.Fatal("a path outside the backup is refused")
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "outside.txt")); err == nil {
		t.Fatal("nothing is written outside")
	}
}

func TestPruneBackups(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		write(t, filepath.Join(dir, BackupFileName("PC", base.Add(time.Duration(i)*time.Hour), i%2 == 0)), "x")
	}
	write(t, filepath.Join(dir, BackupFileName("LAPTOP", base, false)), "x")
	write(t, filepath.Join(dir, "notes.txt"), "x")
	PruneBackups(dir, "PC", 2)
	ents, _ := os.ReadDir(dir)
	var names []string
	for _, e := range ents {
		names = append(names, e.Name())
	}
	got := strings.Join(names, "|")
	if len(names) != 4 || !strings.Contains(got, "LAPTOP") || !strings.Contains(got, "notes.txt") ||
		!strings.Contains(got, "160000") || !strings.Contains(got, "150000") {
		t.Fatalf("keeps this PC's 2 newest and everything else: %v", names)
	}
}
