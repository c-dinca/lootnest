package core

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// ---------------------------------------------------------------------------
// Hydra Launcher is one more launcher, like itch or GOG: Lootnest lists the
// games it installed on this PC. Hydra keeps its library in a LevelDB
// database, %APPDATA%\hydralauncher\hydra-db, with one JSON entry per game
// under the "games" sublevel ("!games!<shop>:<objectId>"). Lootnest reads a
// copy of it, so Hydra can be running, and never writes to it.
// ---------------------------------------------------------------------------

type hydraGame struct {
	Title          string  `json:"title"`
	ObjectID       string  `json:"objectId"`
	Shop           string  `json:"shop"`
	ExecutablePath *string `json:"executablePath"`
	IsDeleted      bool    `json:"isDeleted"`
}

// HydraGames lists the games Hydra installed on this PC whose exe is still
// there. dbDir is Hydra's hydra-db folder; tmp is a folder Lootnest may
// use and remove. The error says why a library that's there couldn't be
// read.
func HydraGames(dbDir, tmp string, containers []string) ([]FoundGame, error) {
	if !isDir(dbDir) {
		return nil, nil
	}
	defer os.RemoveAll(tmp)
	if err := copyLevelDB(dbDir, tmp); err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(tmp, &opt.Options{ErrorIfMissing: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()
	it := db.NewIterator(util.BytesPrefix([]byte("!games!")), nil)
	defer it.Release()
	var out []FoundGame
	seen := map[string]bool{}
	for it.Next() {
		var g hydraGame
		if json.Unmarshal(it.Value(), &g) != nil || g.IsDeleted || g.ExecutablePath == nil {
			continue
		}
		exe := filepath.Clean(strings.TrimSpace(*g.ExecutablePath))
		if !filepath.IsAbs(exe) || !strings.EqualFold(filepath.Ext(exe), ".exe") || !isFile(exe) {
			continue
		}
		root, name, process := exeRoot(exe, containers)
		if tooBroadDir(root) || seen[strings.ToLower(root)] {
			continue
		}
		seen[strings.ToLower(root)] = true
		if t := strings.TrimSpace(g.Title); t != "" {
			name = t
		}
		f := FoundGame{Key: "dir:" + strings.ToLower(root), Name: name, Source: "hydra", Dir: root, Exe: exe, ProcessName: process}
		if g.Shop == "steam" {
			f.SteamID, _ = strconv.Atoi(g.ObjectID)
		}
		out = append(out, f)
	}
	return out, nil
}

// copyLevelDB copies a LevelDB folder's files, except its lock, so the copy
// can be opened while the program that owns it is running. A file the
// program holds open for itself (its newest log, on Windows) is left out:
// the copy then lacks only the latest changes, and the Library keeps what
// the last scan found.
func copyLevelDB(src, dst string) error {
	_ = os.RemoveAll(dst)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() || strings.EqualFold(e.Name(), "LOCK") {
			continue
		}
		in, err := os.Open(filepath.Join(src, e.Name()))
		if err != nil {
			if e.Name() == "CURRENT" || strings.HasPrefix(e.Name(), "MANIFEST-") {
				return err
			}
			continue
		}
		out, err := os.Create(filepath.Join(dst, e.Name()))
		if err == nil {
			_, err = io.Copy(out, io.LimitReader(in, 256<<20))
			if cerr := out.Close(); err == nil {
				err = cerr
			}
		}
		in.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
