//go:build windows

package app

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
)

// hubRecent is how many copies the Saves page lists under Recent copies.
const hubRecent = 30

// apiSavesOverview sends what Lootnest keeps of every game's saves as a
// "savesOverview" event: the copies before swaps, the safety net's copies
// and the save points, newest first, and their sums per game. It only reads.
func (a *App) apiSavesOverview() {
	go func() {
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		var copies []core.SaveCopy
		points := func(key string) {
			for _, p := range core.SavePoints(a.store.SavePointsDir(key)) {
				copies = append(copies, core.SaveCopy{Key: key, Kind: "point", ID: p.ID, At: p.Created, Size: p.Size, Name: p.Name, Owner: p.Player})
			}
		}
		for _, g := range cfg.Games {
			key := "game:" + g.ID
			for _, b := range a.eng.Backups(g.ID) {
				at, _ := time.Parse(time.RFC3339, b.When)
				copies = append(copies, core.SaveCopy{Key: key, Kind: "swap", ID: core.BackupID(b), At: at, Size: core.Size(b.Dir), Owner: b.Owner})
			}
			points(key)
		}
		net := netStore()
		for _, f := range a.visibleFound(a.libraryGames()) {
			if gameSavesOn(f) {
				for _, v := range net.GameVersions(gameSavesKey(f)) {
					copies = append(copies, core.SaveCopy{Key: f.Key, Kind: "net", ID: v.ID, At: v.Created, Size: v.Size})
				}
			}
			points(f.Key)
		}
		names := core.LoadCopyNames(DataDir())
		for i := range copies {
			if c := &copies[i]; c.Kind != "point" {
				c.Name = core.CopyNameOf(names, c.Key, c.Kind, c.ID)
			}
		}
		recent, games := core.SavesOverview(copies, hubRecent)
		a.emit("savesOverview", map[string]any{"recent": recent, "games": games})
	}()
}

// copyPath is where a copy's files are: a copy before a swap's folder, a
// save point's files, or a safety net copy's archive.
func (a *App) copyPath(key, kind, id string) (string, error) {
	if !core.CopyIDOK(id) {
		return "", errors.New("unknown copy")
	}
	switch kind {
	case "swap":
		gid := strings.TrimPrefix(key, "game:")
		for _, b := range a.eng.Backups(gid) {
			if core.BackupID(b) == id {
				return b.Dir, nil
			}
		}
	case "point":
		return filepath.Join(a.store.SavePointsDir(key), id, "files"), nil
	case "net":
		for _, f := range a.visibleFound(a.libraryGames()) {
			if f.Key == key && gameSavesOn(f) {
				dir := netStore().SavesDir(gameSavesKey(f))
				for _, ext := range []string{".zip", ".lnbak"} {
					if p := filepath.Join(dir, id+ext); fileExists(p) {
						if ext == ".lnbak" {
							return "", errors.New("This copy is encrypted, so its files can't be listed.")
						}
						return p, nil
					}
				}
			}
		}
	}
	return "", errors.New("That copy isn't there any more.")
}

// apiCopyFiles lists a copy's files, newest first, for the Saves page.
func (a *App) apiCopyFiles(key, kind, id string) (map[string]any, error) {
	p, err := a.copyPath(key, kind, id)
	if err != nil {
		return nil, err
	}
	files, n, err := core.CopyFiles(p, core.CopyFilesMax)
	if err != nil {
		a.logf("Saves: couldn't list copy %s %s: %v", kind, id, err)
		return nil, errors.New("Couldn't read this copy's files.")
	}
	if files == nil {
		files = []core.CopyFile{}
	}
	return map[string]any{"files": files, "count": n}, nil
}

// apiNameCopy names a copy; a save point keeps its name in the point.
func (a *App) apiNameCopy(key, kind, id, name string) error {
	if kind == "point" {
		return core.RenameSavePoint(a.store.SavePointsDir(key), id, name)
	}
	return core.SetCopyName(DataDir(), key, kind, id, name)
}
