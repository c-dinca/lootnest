package core

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// What's in a copy of saves, and the names the user gives copies. The Saves
// page lists a copy's files so it can be recognised, and any copy can be
// named ("Before the boss"): save points keep their own name, copies before
// a swap and the safety net's copies keep theirs in copynames.json.
// ---------------------------------------------------------------------------

// CopyFile is one file kept in a copy.
type CopyFile struct {
	Path string    `json:"path"` // relative, with forward slashes
	Size int64     `json:"size"`
	Mod  time.Time `json:"mod"`
}

// CopyFilesMax is how many files a copy's list shows at most.
const CopyFilesMax = 200

// CopyFiles lists the files in a copy, a folder or a .zip, newest first, at
// most max of them, with how many there are in all. It only reads.
func CopyFiles(path string, max int) ([]CopyFile, int, error) {
	var all []CopyFile
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		r, err := zip.OpenReader(path)
		if err != nil {
			return nil, 0, err
		}
		defer r.Close()
		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				continue
			}
			all = append(all, CopyFile{Path: strings.TrimPrefix(filepath.ToSlash(f.Name), "/"), Size: int64(f.UncompressedSize64), Mod: f.Modified})
		}
	} else {
		st, err := os.Stat(path)
		if err != nil {
			return nil, 0, err
		}
		if !st.IsDir() {
			return nil, 0, errors.New("not a copy Lootnest can read")
		}
		err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Type()&fs.ModeSymlink != 0 {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(path, p)
			all = append(all, CopyFile{Path: filepath.ToSlash(rel), Size: info.Size(), Mod: info.ModTime()})
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if !all[i].Mod.Equal(all[j].Mod) {
			return all[i].Mod.After(all[j].Mod)
		}
		return all[i].Path < all[j].Path
	})
	n := len(all)
	if max > 0 && n > max {
		all = all[:max]
	}
	return all, n, nil
}

// CopyIDOK: a copy's id is one plain name, never a path.
func CopyIDOK(id string) bool {
	return safeComponent(id) && !strings.ContainsAny(id, `/\:`)
}

// CopyNameMax is how long a copy's name can be, in characters.
const CopyNameMax = 60

func copyNamesFile(dir string) string { return filepath.Join(dir, "copynames.json") }

func copyNameKey(key, kind, id string) string { return key + "|" + kind + "|" + id }

// LoadCopyNames reads the names given to copies, by game key, kind and id.
func LoadCopyNames(dir string) map[string]string {
	out := map[string]string{}
	if b, err := os.ReadFile(copyNamesFile(dir)); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// CopyNameOf is the name given to one copy, or "".
func CopyNameOf(names map[string]string, key, kind, id string) string {
	return names[copyNameKey(key, kind, id)]
}

// SetCopyName names a copy before a swap or a safety net copy; an empty
// name takes the name away. Save points are named with RenameSavePoint.
func SetCopyName(dir, key, kind, id, name string) error {
	if kind != "swap" && kind != "net" {
		return errors.New("only copies before a swap and safety net copies are named here")
	}
	if key == "" || !CopyIDOK(id) {
		return errors.New("unknown copy")
	}
	name = strings.Join(strings.Fields(name), " ")
	if utf8.RuneCountInString(name) > CopyNameMax {
		return errors.New("a name can be up to 60 characters")
	}
	names := LoadCopyNames(dir)
	if name == "" {
		delete(names, copyNameKey(key, kind, id))
	} else {
		names[copyNameKey(key, kind, id)] = name
	}
	return writeJSONAtomic(copyNamesFile(dir), names)
}
