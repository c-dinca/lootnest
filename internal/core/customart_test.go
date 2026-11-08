package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writePicture(t *testing.T, path, format string, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	switch format {
	case "png":
		_ = png.Encode(&buf, img)
	case "jpeg":
		_ = jpeg.Encode(&buf, img, nil)
	case "webp": // a lossless WebP's header: enough for its format and size
		buf.WriteString("RIFF\x00\x00\x00\x00WEBPVP8L\x00\x00\x00\x00\x2f")
		_ = binary.Write(&buf, binary.LittleEndian, uint32(w-1)|uint32(h-1)<<14)
		buf.Write(make([]byte, 32))
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSetCustomArtCopiesAndKeepsTheSource(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	src := filepath.Join(t.TempDir(), "My Cover.PNG")
	want := writePicture(t, src, "png", 60, 90)
	old := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	_ = os.Chtimes(src, old, old)

	w, h, err := s.SetCustomArt("dir:d:\\games\\pbz", ArtCover, src)
	if err != nil || w != 60 || h != 90 {
		t.Fatalf("SetCustomArt = %d, %d, %v", w, h, err)
	}
	if b, _ := os.ReadFile(src); !bytes.Equal(b, want) {
		t.Fatal("the chosen file changed")
	}
	if st, _ := os.Stat(src); !st.ModTime().Equal(old) {
		t.Fatal("the chosen file was touched")
	}
	p := s.CustomArtPath("dir:d:\\games\\pbz", ArtCover)
	if filepath.Dir(p) != s.customArtDir() || !strings.HasSuffix(p, "-cover.png") {
		t.Fatalf("copy at %q", p)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, want) {
		t.Fatal("the copy differs")
	}

	// A new cover replaces the earlier copy, even in another format.
	jpg := filepath.Join(t.TempDir(), "other.jpeg")
	writePicture(t, jpg, "jpeg", 30, 45)
	if _, _, err := s.SetCustomArt("dir:d:\\games\\pbz", ArtCover, jpg); err != nil {
		t.Fatal(err)
	}
	if exists(p) || !strings.HasSuffix(s.CustomArtPath("dir:d:\\games\\pbz", ArtCover), "-cover.jpg") {
		t.Fatal("the earlier copy is still there")
	}
	if !exists(src) {
		t.Fatal("the first chosen file is gone")
	}

	webp := filepath.Join(t.TempDir(), "logo.webp")
	writePicture(t, webp, "webp", 400, 120)
	if w, h, err := s.SetCustomArt("dir:d:\\games\\pbz", ArtLogo, webp); err != nil || w != 400 || h != 120 {
		t.Fatalf("webp logo = %d, %d, %v", w, h, err)
	}
	if m := s.CustomArt()["dir:d:\\games\\pbz"]; len(m) != 2 {
		t.Fatalf("index = %+v", m)
	}
}

func TestSetCustomArtRefuses(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	dir := t.TempDir()
	jpgAsPNG := filepath.Join(dir, "cover.png")
	writePicture(t, jpgAsPNG, "jpeg", 10, 10)
	text := filepath.Join(dir, "notes.jpg")
	_ = os.WriteFile(text, []byte(strings.Repeat("not a picture ", 200)), 0o644)
	gif := filepath.Join(dir, "cover.gif")
	_ = os.WriteFile(gif, []byte("GIF89a"), 0o644)
	jpgLogo := filepath.Join(dir, "logo.jpg")
	writePicture(t, jpgLogo, "jpeg", 10, 10)
	big := filepath.Join(dir, "huge.png")
	f, _ := os.Create(big)
	_ = f.Truncate(CustomArtMax + 1)
	f.Close()

	for _, c := range []struct{ kind, path, want string }{
		{ArtCover, jpgAsPNG, "isn't a PNG"},
		{ArtCover, text, "isn't a JPG"},
		{ArtCover, gif, "PNG, JPEG or WebP"},
		{ArtLogo, jpgLogo, "a logo needs"},
		{ArtCover, big, "over 20 MB"},
		{"banner", jpgLogo, "no such kind"},
	} {
		if _, _, err := s.SetCustomArt("game:x", c.kind, c.path); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s as %s: %v, want %q", filepath.Base(c.path), c.kind, err, c.want)
		}
	}
	if len(s.CustomArt()) != 0 || exists(s.customArtDir()) && len(must(os.ReadDir(s.customArtDir()))) != 0 {
		t.Fatal("a refused picture left something behind")
	}
}

func must[T any](v T, _ error) T { return v }

// Custom art sits beside the art Lootnest fetched, never over it, so Use
// the original again brings that back; clearing deletes only Lootnest's copy.
func TestCustomArtBesideFetchedArt(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	covers := filepath.Join(s.Dir, "covers")
	fetched := filepath.Join(covers, "app4115450.jpg")
	_ = os.MkdirAll(covers, 0o755)
	writePicture(t, fetched, "jpeg", 60, 90)
	src := filepath.Join(t.TempDir(), "mine.png")
	writePicture(t, src, "png", 60, 90)

	if _, _, err := s.SetCustomArt("found:pbz", ArtCover, src); err != nil {
		t.Fatal(err)
	}
	custom := s.CustomArtPath("found:pbz", ArtCover)
	if custom == "" || SavedCover(covers, "Phantom Blade Zero", 4115450) != fetched {
		t.Fatalf("custom %q, fetched %q", custom, SavedCover(covers, "Phantom Blade Zero", 4115450))
	}
	if s.CustomArtPath("found:pbz", ArtHero) != "" || s.CustomArtPath("found:other", ArtCover) != "" {
		t.Fatal("custom art for a kind or game that has none")
	}

	if err := s.ClearCustomArt("found:pbz", ArtCover); err != nil {
		t.Fatal(err)
	}
	if exists(custom) || s.CustomArtPath("found:pbz", ArtCover) != "" || len(s.CustomArt()) != 0 {
		t.Fatal("the copy is still there")
	}
	if !exists(src) || !exists(fetched) {
		t.Fatal("clearing deleted more than Lootnest's copy")
	}
	if err := s.ClearCustomArt("found:pbz", ArtCover); err != nil {
		t.Fatal("clearing twice:", err)
	}
}

func TestRekeyGameMovesCustomArt(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	dir := t.TempDir()
	cover, hero, newHero := filepath.Join(dir, "c.png"), filepath.Join(dir, "h.png"), filepath.Join(dir, "nh.png")
	writePicture(t, cover, "png", 6, 9)
	writePicture(t, hero, "png", 16, 9)
	writePicture(t, newHero, "png", 32, 9)
	_, _, _ = s.SetCustomArt("old", ArtCover, cover)
	_, _, _ = s.SetCustomArt("old", ArtHero, hero)
	_, _, _ = s.SetCustomArt("new", ArtHero, newHero)
	oldCover := s.CustomArtPath("old", ArtCover)

	if err := s.RekeyGame("old", "new"); err != nil {
		t.Fatal(err)
	}
	if exists(oldCover) || s.CustomArtPath("old", ArtCover) != "" {
		t.Fatal("the cover stayed under the old key")
	}
	if b, _ := os.ReadFile(s.CustomArtPath("new", ArtCover)); len(b) == 0 {
		t.Fatal("the cover didn't move")
	}
	// The new key's own hero is kept; the old one stays where it was.
	if c, _, _ := image.DecodeConfig(bytes.NewReader(must(os.ReadFile(s.CustomArtPath("new", ArtHero))))); c.Width != 32 {
		t.Fatalf("new hero is %d wide", c.Width)
	}
	if s.CustomArtPath("old", ArtHero) == "" {
		t.Fatal("the old hero was dropped")
	}
}

func TestRekeyGameWithCustomArtGone(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	src := filepath.Join(t.TempDir(), "c.png")
	writePicture(t, src, "png", 6, 9)
	if _, _, err := s.SetCustomArt("old", ArtCover, src); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(s.CustomArtPath("old", ArtCover)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.PlaytimePath(), []byte(`{"old":{"p":100}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.RekeyGame("old", "new"); err != nil {
		t.Fatal(err)
	}
	pt := map[string]map[string]int64{}
	_ = json.Unmarshal(must(os.ReadFile(s.PlaytimePath())), &pt)
	if pt["new"]["p"] != 100 || pt["old"] != nil {
		t.Fatalf("playtime didn't move: %v", pt)
	}
	if len(s.CustomArt()) != 0 {
		t.Fatalf("the missing copy is still listed: %v", s.CustomArt())
	}
}
