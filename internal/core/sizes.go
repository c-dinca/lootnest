package core

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Disk space: how much each game's folder, its saves, its profiles and
// Lootnest's own copies take. Sizes are counted in the background and kept
// in sizes.json by folder, so a Library of big games isn't walked again
// until a folder changes or a week has gone by. They're approximate: the
// count stops at sizeFileCap files and never follows a link out of the
// folder.
// ---------------------------------------------------------------------------

const (
	sizeFileCap = 400_000
	sizeMaxAge  = 7 * 24 * time.Hour
)

// DirSize counts the bytes and files in dir. Links and junctions inside it
// are skipped, not followed; dir itself may be one. The count stops at
// sizeFileCap files.
func DirSize(dir string) (bytes int64, files int, err error) {
	st, err := os.Stat(dir)
	if err != nil {
		return 0, 0, err
	}
	if !st.IsDir() {
		return st.Size(), 1, nil
	}
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		dir = r // a walk doesn't go into a root that's a link
	}
	stop := errors.New("cap")
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == dir {
				return err
			}
			return nil // a folder Windows won't open counts as empty
		}
		if p != dir && d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := os.Lstat(p)
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		bytes += info.Size()
		if files++; files >= sizeFileCap {
			return stop
		}
		return nil
	})
	if errors.Is(err, stop) {
		err = nil
	}
	return bytes, files, err
}

// SizedDir is one folder's count in sizes.json.
type SizedDir struct {
	Bytes int64     `json:"bytes"`
	Files int       `json:"files"`
	Mod   time.Time `json:"mod"` // the folder's own modification time then
	At    time.Time `json:"at"`
}

// SizeCache is sizes.json: counts by folder.
type SizeCache struct {
	Dirs map[string]SizedDir `json:"dirs"`
}

func (s *Store) SizesPath() string { return filepath.Join(s.Dir, "sizes.json") }

func (s *Store) LoadSizes() SizeCache {
	c := SizeCache{}
	if b, err := os.ReadFile(s.SizesPath()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	if c.Dirs == nil {
		c.Dirs = map[string]SizedDir{}
	}
	return c
}

// SaveSizes writes the cache, without the counts older than a week.
func (s *Store) SaveSizes(c SizeCache, now time.Time) error {
	for k, d := range c.Dirs {
		if now.Sub(d.At) > sizeMaxAge {
			delete(c.Dirs, k)
		}
	}
	return writeJSONAtomic(s.SizesPath(), c)
}

func sizeKey(dir string) string { return strings.ToLower(filepath.Clean(dir)) }

// Size is dir's count: the kept one while the folder's modification time
// is the same and it's less than a week old, else a new one. A folder
// that isn't there is 0.
func (c *SizeCache) Size(dir string, now time.Time) (SizedDir, error) {
	st, err := os.Stat(dir)
	if err != nil {
		delete(c.Dirs, sizeKey(dir))
		return SizedDir{}, nil
	}
	if d, ok := c.Dirs[sizeKey(dir)]; ok && d.Mod.Equal(st.ModTime()) && now.Sub(d.At) <= sizeMaxAge {
		return d, nil
	}
	b, n, err := DirSize(dir)
	if err != nil {
		return SizedDir{}, err
	}
	d := SizedDir{Bytes: b, Files: n, Mod: st.ModTime(), At: now}
	c.Dirs[sizeKey(dir)] = d
	return d, nil
}

// ProfileSize is what a profile keeps of a game in Lootnest's data folder.
func (s *Store) ProfileSize(playerID, gameID string) int64 {
	return Size(s.ProfileDir(playerID, gameID))
}
