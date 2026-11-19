package plat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSecrets(t *testing.T) {
	dir := t.TempDir()
	keyring := ""
	working := true
	keyStore.lookup = func() (string, error) {
		if !working {
			return "", errors.New("no Secret Service")
		}
		return keyring, nil
	}
	keyStore.store = func(v string) error { keyring = v; return nil }
	keyStore.file = func() string { return filepath.Join(dir, "secret.key") }
	reset := func() { secretKey = nil }
	reset()

	enc, err := Protect([]byte("steam-api-key"))
	if err != nil {
		t.Fatal(err)
	}
	if keyring == "" || strings.Contains(string(enc), "steam-api-key") {
		t.Fatal("the key goes to the keyring, the text is encrypted")
	}
	reset() // a new start reads the key back from the keyring
	if dec, err := Unprotect(enc); err != nil || string(dec) != "steam-api-key" {
		t.Fatalf("round trip: %q %v", dec, err)
	}
	if _, err := Unprotect(append(enc[:len(enc)-1:len(enc)-1], enc[len(enc)-1]^1)); err == nil {
		t.Fatal("a changed byte is refused")
	}
	if _, err := Unprotect([]byte("DPAPI blob from Windows")); err == nil {
		t.Fatal("Windows' DPAPI isn't read here")
	}

	// No keyring: a file only this user can read.
	working, keyring = false, ""
	reset()
	enc2, err := Protect([]byte("token"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filepath.Join(dir, "secret.key"))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("the key file: %v %v", st, err)
	}
	reset()
	if dec, err := Unprotect(enc2); err != nil || string(dec) != "token" {
		t.Fatalf("from the key file: %q %v", dec, err)
	}
	// The keyring comes back later: the file's key still opens what it made.
	working = true
	reset()
	if dec, err := Unprotect(enc2); err != nil || string(dec) != "token" {
		t.Fatalf("keyring back, file key kept: %q %v", dec, err)
	}
}

func TestTrash(t *testing.T) {
	root := t.TempDir()
	trash := filepath.Join(root, "share", "Trash")
	trashHome = func() string { return trash }
	_ = os.MkdirAll(filepath.Join(root, "share"), 0o755)
	game := filepath.Join(root, "Games", "Old Game ü")
	_ = os.MkdirAll(filepath.Join(game, "data"), 0o755)
	_ = os.WriteFile(filepath.Join(game, "data", "x"), []byte("x"), 0o644)
	again := filepath.Join(root, "Other", "Old Game ü")
	_ = os.MkdirAll(again, 0o755)

	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.Local)
	if err := trashOne(game, now); err != nil {
		t.Fatal(err)
	}
	if err := trashOne(again, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(game); !os.IsNotExist(err) {
		t.Fatal("the folder left its place")
	}
	if _, err := os.Stat(filepath.Join(trash, "files", "Old Game ü", "data", "x")); err != nil {
		t.Fatal("it's in the Trash with its files")
	}
	info, err := os.ReadFile(filepath.Join(trash, "info", "Old Game ü.trashinfo"))
	if err != nil || !strings.Contains(string(info), "Path="+trashPath(game)) || !strings.Contains(string(info), "DeletionDate=2026-11-19T12:00:00") {
		t.Fatalf("its info: %s %v", info, err)
	}
	if !strings.Contains(trashPath(game), "%C3%BC") || strings.Contains(trashPath(game), " ") {
		t.Fatalf("the path is escaped: %s", trashPath(game))
	}
	if _, err := os.Stat(filepath.Join(trash, "info", "Old Game ü.2.trashinfo")); err != nil {
		t.Fatal("a second one with the same name gets its own")
	}
	if err := trashOne(filepath.Join(root, "missing"), now); err == nil {
		t.Fatal("nothing to move")
	}
	entries, _ := os.ReadDir(filepath.Join(trash, "info"))
	if len(entries) != 2 {
		t.Fatalf("a failed move leaves no info file: %d", len(entries))
	}
}

func TestAutostart(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, on := Autostart("Lootnest"); on {
		t.Fatal("off at first")
	}
	if err := SetAutostart("Lootnest", `"/opt/lootnest/lootnest" --tray`, true); err != nil {
		t.Fatal(err)
	}
	if cmd, on := Autostart("Lootnest"); !on || cmd != `"/opt/lootnest/lootnest" --tray` {
		t.Fatalf("on: %q %v", cmd, on)
	}
	if err := SetAutostart("Lootnest", "", false); err != nil {
		t.Fatal(err)
	}
	if _, on := Autostart("Lootnest"); on {
		t.Fatal("off again")
	}
	if err := SetAutostart("Lootnest", "", false); err != nil {
		t.Fatal("turning off twice is fine")
	}
}
