package core

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Steam Cloud. Profiles work with Steam Cloud left on; Lootnest only needs
// to know which of a game's swapped files Steam syncs, and to notice when
// Steam replaced the files it just loaded with the cloud's copy.
//
// Steam lists a game's synced files in userdata\<id>\<appid>\remotecache.vdf,
// with each file's SHA-1 as Steam last synced it. Root "0" files live in
// userdata\<id>\<appid>\remote; other roots (Steam Auto-Cloud) are folders
// such as Documents or AppData, matched here by their path's tail.
// ---------------------------------------------------------------------------

type CloudFile struct {
	Path string // as listed, lower case, with "/"
	Root string
	SHA  string // lower-case hex SHA-1
}

// CloudMatch is a synced file that is also one of the game's swapped files.
type CloudMatch struct {
	CloudFile
	Local string
}

// CloudFiles reads the files Steam Cloud syncs for a game, or none when the
// game doesn't use it (or never synced on this PC).
func CloudFiles(steamUserDir string, appID int) []CloudFile {
	if steamUserDir == "" || appID <= 0 {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(steamUserDir, strconv.Itoa(appID), "remotecache.vdf"))
	if err != nil {
		return nil
	}
	m, err := ParseVDF(string(b))
	if err != nil {
		return nil
	}
	app := vdfMap(m, strconv.Itoa(appID))
	var out []CloudFile
	for name, v := range app {
		f, ok := v.(map[string]any)
		if !ok || vdfStr(f, "sha") == "" {
			continue
		}
		out = append(out, CloudFile{Path: strings.ToLower(filepath.ToSlash(name)), Root: vdfStr(f, "root"), SHA: strings.ToLower(vdfStr(f, "sha"))})
	}
	return out
}

// CloudMatches returns the synced files that lie inside the game's swapped
// locations (roots, as resolved live paths).
func CloudMatches(files []CloudFile, steamUserDir string, appID int, roots []string) []CloudMatch {
	if len(files) == 0 || len(roots) == 0 {
		return nil
	}
	within := func(p string) bool {
		for _, r := range roots {
			if isWithin(p, r) {
				return true
			}
		}
		return false
	}
	// Every file under the swapped locations, by lower-case "/" path.
	var local []string
	for _, r := range roots {
		_ = filepath.WalkDir(r, func(p string, d fs.DirEntry, err error) error {
			if err != nil || len(local) > 20000 {
				return filepath.SkipDir
			}
			if !d.IsDir() {
				local = append(local, p)
			}
			return nil
		})
	}
	remote := filepath.Join(steamUserDir, strconv.Itoa(appID), "remote")
	var out []CloudMatch
	for _, f := range files {
		if f.Root == "0" {
			p := filepath.Join(remote, filepath.FromSlash(f.Path))
			if within(p) {
				out = append(out, CloudMatch{f, p})
			}
			continue
		}
		for _, p := range local {
			if strings.HasSuffix(strings.ToLower(filepath.ToSlash(p)), "/"+f.Path) {
				out = append(out, CloudMatch{f, p})
				break
			}
		}
	}
	return out
}

// HashFile is a file's lower-case hex SHA-1, or "" if it can't be read.
func HashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// HashMatches records the current content of each matched file.
func HashMatches(ms []CloudMatch) map[string]string {
	out := map[string]string{}
	for _, m := range ms {
		out[m.Local] = HashFile(m.Local)
	}
	return out
}

// CloudReplaced lists the matched files that Steam replaced with the cloud's
// copy after Lootnest loaded a player's files: their content is now exactly
// the cloud version, and that is not what was loaded. A file the game itself
// changed matches neither, so it isn't reported.
func CloudReplaced(ms []CloudMatch, loaded map[string]string) []string {
	var out []string
	for _, m := range ms {
		was, ok := loaded[m.Local]
		if !ok || was == m.SHA {
			continue
		}
		if now := HashFile(m.Local); now != "" && now == m.SHA {
			out = append(out, m.Local)
		}
	}
	return out
}

// RevertSession is used when Steam Cloud replaced a player's files during a
// session: the live files are kept as a backup, and the player's profile
// (untouched by this session) goes back into the live locations, so the
// player's own settings and saves are what's loaded again.
func (e *Engine) RevertSession(cfg *Config, gameID, playerID string, keep int) (string, error) {
	g := cfg.Game(gameID)
	if g == nil {
		return "", nil
	}
	rs, err := e.resolveAll(g)
	if err != nil {
		return "", err
	}
	dir, err := e.Backup(g, keep)
	if err != nil {
		return "", err
	}
	if !e.HasProfile(playerID, gameID) {
		return dir, nil // nothing to put back; the files stay as they are
	}
	return dir, e.swapIn(g, playerID, rs, false)
}

// LivePaths resolves a game's swapped locations.
func (e *Engine) LivePaths(g *Game) ([]string, error) {
	rs, err := e.resolveAll(g)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.live
	}
	return out, nil
}

// SplitSteamAppFolder fixes a game set up (by earlier versions) to swap
// Steam's whole folder for it, Steam\userdata\<account>\<appid>, which also
// swapped Steam Cloud's records. That entry is replaced by one entry per
// folder inside it (local, remote, ...), and each player's saved copy of
// those folders is moved, not copied, to the new entries, so nothing is lost
// and nobody's files change. It reports whether g changed; the caller saves
// the config.
func (e *Engine) SplitSteamAppFolder(cfg *Config, g *Game) (bool, error) {
	if g.Launch.Type != "steam" {
		return false, nil
	}
	changed := false
	var paths []PathEntry
	for _, p := range g.Paths {
		abs, err := e.Env.expand(p.Path)
		if err != nil || e.Env.SteamAppFolder(abs) != g.Launch.AppID {
			paths = append(paths, p)
			continue
		}
		subs, err := os.ReadDir(abs)
		if err != nil {
			return false, err
		}
		var add []PathEntry
		for _, d := range subs {
			if d.IsDir() {
				add = append(add, PathEntry{ID: p.ID + "-" + strings.ToLower(d.Name()), Path: p.Path + "/" + d.Name()})
			}
		}
		if len(add) == 0 {
			paths = append(paths, p) // nothing to split into; left for the user to fix
			continue
		}
		for _, pl := range cfg.Players {
			dir := e.Store.ProfileDir(pl.ID, g.ID)
			for _, a := range add {
				sub := filepath.Join(dir, p.ID, filepath.Base(filepath.FromSlash(a.Path)))
				if exists(sub) && !exists(filepath.Join(dir, a.ID)) {
					if err := os.Rename(sub, filepath.Join(dir, a.ID)); err != nil {
						return false, err
					}
				}
			}
		}
		e.logf("%s swapped Steam's own folder for it; it now swaps the folders inside it instead", g.Name)
		paths = append(paths, add...)
		changed = true
	}
	if changed {
		g.Paths = paths
	}
	return changed, nil
}
