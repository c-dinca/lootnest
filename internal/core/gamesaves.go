package core

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Cloud saves for games without profiles (the ones not on Steam: Steam has
// its own cloud): a game's saves, kept as versions in the cloud folder, like
// a game launcher's cloud saves. A new version is made when the saves
// changed; any version can be restored, on this PC or another.
//
//	<cloud>/Lootnest/saves/<game>/<time> <PC>.zip   (.lnbak encrypted)
//	<cloud>/Lootnest/saves/<game>/<time> <PC>.json  SaveVersion
// ---------------------------------------------------------------------------

// SaveVersion is one version of a game's saves in the cloud.
type SaveVersion struct {
	ID        string    `json:"id"` // file name without extension
	Game      string    `json:"game"`
	Computer  string    `json:"computer"`
	Created   time.Time `json:"created"`
	Size      int64     `json:"size"`
	Hash      string    `json:"hash"`
	Encrypted bool      `json:"encrypted"`
	Paths     []string  `json:"paths"` // the save locations in it
}

// GameSavesKey names a game outside Steam the same way on every PC.
func GameSavesKey(name, uri string) string {
	return GameSyncKey(&Game{Name: name, Launch: Launch{URI: uri}})
}

func (c *Cloud) savesDir(game string) string {
	return filepath.Join(c.dir(), "saves", filepath.Base(game))
}

// locationsAsPaths makes PathEntries for SyncHash and the saves format.
func locationsAsPaths(locs []SaveLocation) ([]PathEntry, func(PathEntry) string) {
	paths := make([]PathEntry, len(locs))
	at := map[string]string{}
	for i, l := range locs {
		paths[i] = PathEntry{ID: l.Path, Path: l.Path}
		at[l.Path] = l.Resolved
	}
	return paths, func(pe PathEntry) string { return at[pe.Path] }
}

// SavesHash is the hash of a game's saves, the same on every PC.
func SavesHash(locs []SaveLocation) (string, error) {
	paths, dir := locationsAsPaths(locs)
	return SyncHash(&Game{Paths: paths}, dir)
}

// SaveGameVersion puts a new version of a game's saves in the cloud and
// keeps the newest keep versions.
func (c *Cloud) SaveGameVersion(game, name string, locs []SaveLocation, hash string, keep int) (*SaveVersion, error) {
	if len(locs) == 0 {
		return nil, errors.New("Lootnest doesn't know where this game keeps its saves.")
	}
	dir := c.savesDir(game)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := now.Format("2006-01-02 150405") + " " + safeFileName(c.Computer)
	ext := ".zip"
	if c.Key != nil {
		ext = ".lnbak"
	}
	opt, err := c.archiveOptions()
	if err != nil {
		return nil, err
	}
	paths, at := locationsAsPaths(locs)
	size, err := writeArchiveFile(filepath.Join(dir, id+ext), func(w io.Writer) error { return writeSaves(w, paths, at) }, opt)
	if err != nil {
		return nil, err
	}
	v := &SaveVersion{ID: id, Game: name, Computer: c.Computer, Created: now, Size: size, Hash: hash, Encrypted: c.Key != nil}
	for _, l := range locs {
		v.Paths = append(v.Paths, l.Path)
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	if err := writeFileAtomic(filepath.Join(dir, id+".json"), b); err != nil {
		return nil, err
	}
	c.pruneVersions(game, keep)
	return v, nil
}

// GameVersions lists a game's versions in the cloud, newest first.
func (c *Cloud) GameVersions(game string) []SaveVersion {
	ents, _ := os.ReadDir(c.savesDir(game))
	var out []SaveVersion
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(c.savesDir(game), e.Name()))
		if err != nil {
			continue
		}
		var v SaveVersion
		if json.Unmarshal(b, &v) != nil || v.ID != strings.TrimSuffix(e.Name(), ".json") || !safeComponent(v.ID) {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

func (c *Cloud) pruneVersions(game string, keep int) {
	if keep < 1 {
		keep = 10
	}
	vs := c.GameVersions(game)
	for _, v := range vs[min(keep, len(vs)):] {
		base := filepath.Join(c.savesDir(game), v.ID)
		_ = os.Remove(base + ".zip")
		_ = os.Remove(base + ".lnbak")
		_ = os.Remove(base + ".json")
	}
}

// RestoreGameVersion puts a version's saves in place over locs (the game's
// locations here, or the ones the version names when the game has none
// here yet: resolve turns a tokenized path into this PC's).
func (c *Cloud) RestoreGameVersion(game, id string, locs []SaveLocation, resolve func(string) (string, error), tmp string) (skipped int, err error) {
	if !safeComponent(id) {
		return 0, errors.New("unknown version")
	}
	var v *SaveVersion
	for _, x := range c.GameVersions(game) {
		if x.ID == id {
			x := x
			v = &x
		}
	}
	if v == nil {
		return 0, errors.New("That version isn't in the cloud folder any more.")
	}
	file := filepath.Join(c.savesDir(game), id+".zip")
	if v.Encrypted {
		file = filepath.Join(c.savesDir(game), id+".lnbak")
	}
	// Every location the version has, here: the ones found here, else the
	// version's own paths resolved on this PC.
	all := append([]SaveLocation{}, locs...)
	for _, p := range v.Paths {
		found := false
		for _, l := range all {
			found = found || strings.EqualFold(l.Path, p)
		}
		if !found && resolve != nil {
			if r, err := resolve(p); err == nil {
				all = append(all, SaveLocation{Path: p, Resolved: r})
			}
		}
	}
	paths, at := locationsAsPaths(all)
	return readSaves(file, c.Key, paths, at, tmp)
}
