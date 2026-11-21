package core

import (
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// A "mod" that is someone's saved game (a 100% save, a prologue skip) goes
// in the game's save folder, never its program folder, where the game
// doesn't look for it. ModLooksLikeSave tells such a download apart, so the
// Mods tab hands it to Try a save, which puts it in a run of its own.
// ---------------------------------------------------------------------------

// saveExts are extensions games give their save files.
var saveExts = map[string]bool{
	".sav": true, ".save": true, ".savegame": true, ".sl2": true, // many; Dark Souls, Elden Ring
	".ess": true, ".fos": true, ".lsv": true, // Skyrim, Fallout 4, Baldur's Gate 3
}

// saveNameRe matches save file names that carry no telling extension:
// Rockstar's (SRDR30000 for Red Dead Redemption 2, SGTA50000 for GTA V)
// and the common savegame1, save_02, slot3.
var saveNameRe = regexp.MustCompile(`^(srdr3\d{4,}|sgta5\d{4,}|sgtav\d{4,}|savegame[_-]?\d*|save[_-]?\d+|slot[_-]?\d+|savedata\d*)(\.[a-z0-9]{1,8})?$`)

// savedGamesRe matches Nexus Mods' category for saves ("Saved Games").
var savedGamesRe = regexp.MustCompile(`\bsave(d)?[ -]?games?\b`)

// besideSave are files a save's archive often carries next to it.
var besideSave = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".bmp": true, ".gif": true}

// saveShape is a file name with its digits as #, so SRDR30003 and SRDR30014
// have one shape.
func saveShape(name string) string {
	b := []byte(strings.ToLower(name))
	for i, c := range b {
		if c >= '0' && c <= '9' {
			b[i] = '#'
		}
	}
	return string(b)
}

// SaveNames lists the file names in a game's save folders (at most three
// folders down and 5,000 names), for ModLooksLikeSave.
func SaveNames(dirs []string) []string {
	var out []string
	for _, d := range dirs {
		d = filepath.Clean(d)
		_ = filepath.WalkDir(d, func(p string, e fs.DirEntry, err error) error {
			if len(out) >= 5000 {
				return fs.SkipAll
			}
			if err != nil {
				return filepath.SkipDir
			}
			if e.IsDir() {
				if rel, _ := filepath.Rel(d, p); strings.Count(rel, string(filepath.Separator)) >= 3 {
					return filepath.SkipDir
				}
				return nil
			}
			out = append(out, e.Name())
			return nil
		})
	}
	return out
}

// ModLooksLikeSave says whether a mod is a saved game rather than a mod.
// category is the mod's on Nexus Mods, or ""; saves are the file names in
// the game's save folders (SaveNames). Only a mod that matched nothing in
// the game's folder (AtRoot) can be one. Then it's a save when Nexus files
// it under saved games, or when every file but pictures is named like a
// save: a save's extension, a save's usual name, or the shape of a file in
// the game's save folders with a number in it (SRDR30000 beside SRDR30002).
func ModLooksLikeSave(m *Mod, category string, saves []string) bool {
	if m == nil || !m.AtRoot || len(m.Files) == 0 {
		return false
	}
	if savedGamesRe.MatchString(strings.ToLower(category)) {
		return true
	}
	shapes := map[string]bool{}
	for _, n := range saves {
		if s := saveShape(n); strings.Contains(s, "#") {
			shapes[s] = true
		}
	}
	found := false
	for _, f := range m.Files {
		name := strings.ToLower(path.Base(f.Rel))
		ext := path.Ext(name)
		switch {
		case besideSave[ext]:
			continue
		case saveExts[ext], saveNameRe.MatchString(name), shapes[saveShape(name)]:
			found = true
		default:
			return false
		}
	}
	return found
}
