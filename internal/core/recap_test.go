package core

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodePNGDataURL(t *testing.T) {
	png := append(append([]byte{}, pngMagic...), 1, 2, 3)
	b, err := DecodePNGDataURL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))
	if err != nil || len(b) != len(png) {
		t.Fatalf("a PNG was refused: %v", err)
	}
	for _, bad := range []string{
		"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(png),
		"data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("<svg>")),
		"data:image/png;base64,!!!",
		"https://example.com/x.png",
	} {
		if _, err := DecodePNGDataURL(bad); err == nil {
			t.Errorf("accepted %q", bad[:20])
		}
	}
}

func TestRecapFileName(t *testing.T) {
	if got := RecapFileName("Player 1", 2026); got != "Player 1 in 2026.png" {
		t.Fatal(got)
	}
	got := RecapFileName(`a\b/c:d*?"<>|`, 2026)
	if strings.ContainsAny(strings.TrimSuffix(got, ".png"), `\/:*?"<>|`) {
		t.Fatalf("unsafe name %q", got)
	}
	if RecapFileName("  ", 2026) != "Lootnest in 2026.png" {
		t.Fatal(RecapFileName("  ", 2026))
	}
}
