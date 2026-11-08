package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Custom art: a picture the user chose for a game's cover, wide art or logo.
// Lootnest keeps its own copy in covers\custom, named after the Library key;
// the chosen file is only read. art.json there lists the copies by key.
// ---------------------------------------------------------------------------

// CustomArtMax is the largest picture Lootnest copies.
const CustomArtMax = 20 << 20

const customArtSide = 16384 // pixels; anything larger isn't a cover

func (s *Store) customArtDir() string   { return filepath.Join(s.Dir, "covers", "custom") }
func (s *Store) customArtIndex() string { return filepath.Join(s.customArtDir(), "art.json") }

func (s *Store) readCustomArt() map[string]map[string]string {
	m := map[string]map[string]string{}
	if b, err := os.ReadFile(s.customArtIndex()); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

// CustomArt lists the custom art kept: Library key -> kind -> file name.
func (s *Store) CustomArt() map[string]map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readCustomArt()
}

// CustomArtPath is the copy of a game's custom art of a kind, or "".
func (s *Store) CustomArtPath(key, kind string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := s.readCustomArt()[key][kind]
	if name == "" {
		return ""
	}
	p := filepath.Join(s.customArtDir(), filepath.Base(name))
	if !exists(p) {
		return ""
	}
	return p
}

// pictureFormat reads which format b is in ("png", "jpeg" or "webp") and
// its size from its header.
func pictureFormat(b []byte) (format string, w, h int) {
	if c, f, err := image.DecodeConfig(bytes.NewReader(b)); err == nil && (f == "png" || f == "jpeg") {
		return f, c.Width, c.Height
	}
	if len(b) < 30 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return "", 0, 0
	}
	c := b[12:] // the first chunk: its type, its length, then its data at c[8:]
	switch string(c[0:4]) {
	case "VP8 ":
		if c[11] == 0x9d && c[12] == 0x01 && c[13] == 0x2a {
			w, h = int(binary.LittleEndian.Uint16(c[14:])&0x3fff), int(binary.LittleEndian.Uint16(c[16:])&0x3fff)
		}
	case "VP8L":
		if c[8] == 0x2f {
			v := binary.LittleEndian.Uint32(c[9:])
			w, h = int(v&0x3fff)+1, int(v>>14&0x3fff)+1
		}
	case "VP8X":
		w = int(c[12]) | int(c[13])<<8 | int(c[14])<<16 + 1
		h = int(c[15]) | int(c[16])<<8 | int(c[17])<<16 + 1
	}
	if w == 0 || h == 0 {
		return "", 0, 0
	}
	return "webp", w, h
}

var customArtExt = map[string]string{".png": "png", ".jpg": "jpeg", ".jpeg": "jpeg", ".webp": "webp"}

// SetCustomArt copies the picture at src as a game's art of a kind (ArtCover,
// ArtHero or ArtLogo), in place of an earlier custom one of that kind. src
// is never changed. It returns the picture's size in pixels.
func (s *Store) SetCustomArt(key, kind, src string) (w, h int, err error) {
	if key == "" || steamArtURL[kind] == "" {
		return 0, 0, errors.New("no such kind of art")
	}
	format := customArtExt[strings.ToLower(filepath.Ext(src))]
	if format == "" {
		return 0, 0, errors.New("choose a PNG, JPEG or WebP picture")
	}
	st, err := os.Stat(src)
	if err != nil {
		return 0, 0, err
	}
	if !st.Mode().IsRegular() {
		return 0, 0, errors.New("choose a picture file")
	}
	if st.Size() > CustomArtMax {
		return 0, 0, errors.New("that picture is over 20 MB; choose a smaller one")
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return 0, 0, err
	}
	got, w, h := pictureFormat(b)
	switch {
	case got != format:
		return 0, 0, fmt.Errorf("%s isn't a %s picture Lootnest can read", filepath.Base(src), strings.ToUpper(strings.TrimPrefix(filepath.Ext(src), ".")))
	case w > customArtSide || h > customArtSide:
		return 0, 0, fmt.Errorf("that picture is %d by %d pixels; choose one under %d on each side", w, h, customArtSide)
	case kind == ArtLogo && format == "jpeg":
		return 0, 0, errors.New("a logo needs a PNG or WebP picture, so the art shows around it")
	}
	ext := "." + format
	if format == "jpeg" {
		ext = ".jpg"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.customArtDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, 0, err
	}
	name := gameFileKey(key) + "-" + kind + ext
	tmp := filepath.Join(dir, name+".tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		_ = os.Remove(tmp)
		return 0, 0, err
	}
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		_ = os.Remove(tmp)
		return 0, 0, err
	}
	m := s.readCustomArt()
	if m[key] == nil {
		m[key] = map[string]string{}
	}
	if old := m[key][kind]; old != "" && old != name {
		_ = os.Remove(filepath.Join(dir, filepath.Base(old)))
	}
	m[key][kind] = name
	return w, h, writeJSONAtomic(s.customArtIndex(), m)
}

// ClearCustomArt deletes Lootnest's copy of a game's custom art of a kind,
// so its own art shows again.
func (s *Store) ClearCustomArt(key, kind string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.readCustomArt()
	name := m[key][kind]
	if name == "" {
		return nil
	}
	if err := os.Remove(filepath.Join(s.customArtDir(), filepath.Base(name))); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(m[key], kind)
	if len(m[key]) == 0 {
		delete(m, key)
	}
	return writeJSONAtomic(s.customArtIndex(), m)
}

// rekeyCustomArt moves a game's custom art to its new key, each kind the
// new key has none of. A copy that's gone is dropped; one that can't be
// moved stays under the old key. The caller holds s.mu.
func (s *Store) rekeyCustomArt(from, to string) error {
	m := s.readCustomArt()
	if len(m[from]) == 0 {
		return nil
	}
	if m[to] == nil {
		m[to] = map[string]string{}
	}
	dir := s.customArtDir()
	var failed error
	for kind, name := range m[from] {
		if m[to][kind] != "" {
			continue
		}
		moved := gameFileKey(to) + "-" + kind + filepath.Ext(name)
		err := os.Rename(filepath.Join(dir, filepath.Base(name)), filepath.Join(dir, moved))
		switch {
		case err == nil:
			m[to][kind] = moved
		case os.IsNotExist(err):
		default:
			failed = err
			continue
		}
		delete(m[from], kind)
	}
	if len(m[from]) == 0 {
		delete(m, from)
	}
	if len(m[to]) == 0 {
		delete(m, to)
	}
	if err := writeJSONAtomic(s.customArtIndex(), m); err != nil {
		return err
	}
	return failed
}
