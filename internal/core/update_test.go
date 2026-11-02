package core

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestNewerVersion(t *testing.T) {
	for _, c := range []struct {
		cur, latest string
		want        bool
	}{
		{"1.3.0", "v1.4.0", true},
		{"1.4.0", "1.4.0", false},
		{"1.4.0", "v1.3.9", false},
		{"1.4.9", "1.10.0", true},
		{"1.4", "1.4.1", true},
		{"1.4.0", "nightly", false},
		{"dev", "1.4.0", false},
	} {
		if got := NewerVersion(c.cur, c.latest); got != c.want {
			t.Errorf("NewerVersion(%q, %q) = %v", c.cur, c.latest, got)
		}
	}
}

func TestParseRelease(t *testing.T) {
	r, err := ParseRelease([]byte(`{"tag_name":"v1.4.0","html_url":"https://github.com/c-dinca/lootnest/releases/tag/v1.4.0","draft":false,"prerelease":false,
		"assets":[{"name":"Lootnest.exe","browser_download_url":"x","size":1},
		{"name":"Lootnest-Setup.exe","browser_download_url":"https://github.com/c-dinca/lootnest/releases/download/v1.4.0/Lootnest-Setup.exe","size":7287808,"digest":"sha256:ABCDEF"}]}`))
	if err != nil || r.Version != "1.4.0" || r.Size != 7287808 || r.SHA256 != "abcdef" || r.SetupURL == "x" || r.Page == "" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := ParseRelease([]byte(`{"tag_name":"v2.0.0","prerelease":true}`)); err == nil {
		t.Fatal("pre-releases aren't offered")
	}
}

func TestVerifyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "setup.exe")
	data := []byte("lootnest setup")
	_ = os.WriteFile(p, data, 0o644)
	sum := sha256.Sum256(data)
	good := hex.EncodeToString(sum[:])
	if err := VerifyFile(p, int64(len(data)), good); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFile(p, int64(len(data))+1, ""); err == nil {
		t.Fatal("a short download must fail")
	}
	if err := VerifyFile(p, 0, "00"+good[2:]); err == nil {
		t.Fatal("a wrong checksum must fail")
	}
}
