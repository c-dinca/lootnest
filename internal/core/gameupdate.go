package core

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Applying an update to a game in a folder. The user brings the update (a
// folder or a .zip, from wherever they got it); Lootnest only copies its
// files into the game's folder, safely:
//
//  1. the plan says which files are added and which replaced, and where
//     the update's files go (an update often comes inside a folder of its
//     own, which is skipped when the game's folder doesn't have it);
//  2. every file to be replaced is copied to the update's own backup first;
//  3. each new file is written next to its place, read back, then moved in;
//  4. any failure puts everything back as it was.
//
// An applied update can be undone: its added files go and the replaced ones
// come back. Each update keeps <backup>\update.json and the old files in
// <backup>\files.
// ---------------------------------------------------------------------------

// UpdateFile is one file an update writes, relative to the game's folder.
type UpdateFile struct {
	Rel      string `json:"rel"` // with "/"
	Size     int64  `json:"size"`
	Replaces bool   `json:"replaces"`
	src      string // in a folder: its path; in a zip: its name
}

// UpdatePlan is what applying an update will do.
type UpdatePlan struct {
	Source  string       `json:"source"`
	Kind    string       `json:"kind"`            // "folder" or "zip"
	Skipped string       `json:"skipped"`         // the update's own top folder, not copied as a folder
	Files   []UpdateFile `json:"files,omitempty"` // every file, sorted
	Added   int          `json:"added"`
	Replace int          `json:"replace"`
	Bytes   int64        `json:"bytes"`            // all the update's files
	Kept    int64        `json:"kept"`             // the game's files kept aside first
	Where   []UpdateArea `json:"where"`            // the folders the files go into, most first
	Readme  []string     `json:"readme,omitempty"` // text files at the update's top, shown, not copied
}

// UpdateArea is a top-level folder of the game the update writes into.
type UpdateArea struct {
	Path  string `json:"path"` // "" for the game's own folder
	Files int    `json:"files"`
}

// AppliedUpdate is the record of one update.
type AppliedUpdate struct {
	ID       string    `json:"id"`
	Game     string    `json:"game"`
	Source   string    `json:"source"` // the file or folder's name
	At       time.Time `json:"at"`
	Added    []string  `json:"added"`
	Replaced []string  `json:"replaced"`
	Dirs     []string  `json:"dirs,omitempty"` // folders it made, removed on undo when empty
	Bytes    int64     `json:"bytes"`
	State    string    `json:"state"` // "applying", "applied", "undone"
	Dir      string    `json:"-"`
}

var errUnsafePath = errors.New("the update has a file outside the game's folder")

// updateRel cleans a path inside an update, or refuses one that leaves it.
func updateRel(name string) (string, error) {
	n := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(n, "/") || strings.Contains(n, ":") {
		return "", errUnsafePath
	}
	n = filepath.ToSlash(filepath.Clean(filepath.FromSlash(n)))
	if n == "." || n == ".." || strings.HasPrefix(n, "../") {
		return "", errUnsafePath
	}
	for _, part := range strings.Split(n, "/") {
		if part == "" || part == ".." {
			return "", errUnsafePath
		}
	}
	return n, nil
}

// readmeName: notes that come with an update, not part of the game.
func readmeName(rel string) bool {
	if strings.Contains(rel, "/") {
		return false
	}
	l := strings.ToLower(rel)
	ext := filepath.Ext(l)
	return ext == ".txt" || ext == ".nfo" || ext == ".url" || ext == ".md" || strings.HasPrefix(l, "readme")
}

// PlanUpdate reads an update (a folder or a .zip) and works out what it
// does to the game in gameDir.
func PlanUpdate(gameDir, src string) (*UpdatePlan, error) {
	st, err := os.Stat(gameDir)
	if err != nil || !st.IsDir() {
		return nil, errors.New("the game's folder isn't there")
	}
	ss, err := os.Stat(src)
	if err != nil {
		return nil, errors.New("Lootnest can't find that update")
	}
	if samePath(src, gameDir) || isWithin(src, gameDir) || isWithin(gameDir, src) {
		return nil, errors.New("choose the update, not the game's own folder")
	}
	p := &UpdatePlan{Source: src}
	var files []UpdateFile
	switch {
	case ss.IsDir():
		p.Kind = "folder"
		err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&fs.ModeSymlink != 0 || d.Type()&fs.ModeIrregular != 0 {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil // never follow links out of the update
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			r, err := updateRel(rel)
			if err != nil {
				return err
			}
			files = append(files, UpdateFile{Rel: r, Size: info.Size(), src: path})
			return nil
		})
	case strings.EqualFold(filepath.Ext(src), ".zip"):
		p.Kind = "zip"
		var zr *zip.ReadCloser
		if zr, err = zip.OpenReader(src); err != nil {
			return nil, errors.New("that .zip can't be opened; it may be damaged")
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			r, rerr := updateRel(f.Name)
			if rerr != nil {
				err = rerr
				break
			}
			files = append(files, UpdateFile{Rel: r, Size: int64(f.UncompressedSize64), src: f.Name})
		}
		zr.Close()
	default:
		return nil, errors.New("choose the update's folder or its .zip; other archives need to be extracted first")
	}
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("the update has no files")
	}

	// Where its files go: skip the update's own top folders when that fits
	// the game's folder better ("Update\Binaries\..." into the game's
	// Binaries).
	prefix := bestPrefix(gameDir, files)
	p.Skipped = strings.TrimSuffix(prefix, "/")
	areas := map[string]int{}
	seen := map[string]bool{}
	for _, f := range files {
		rel := strings.TrimPrefix(f.Rel, prefix)
		if !strings.HasPrefix(f.Rel, prefix) || rel == "" {
			continue // outside the folder that's copied (notes beside it)
		}
		if readmeName(rel) {
			p.Readme = append(p.Readme, rel)
			continue
		}
		key := strings.ToLower(rel)
		if seen[key] {
			continue
		}
		seen[key] = true
		f.Rel = rel
		target := filepath.Join(gameDir, filepath.FromSlash(rel))
		if ts, err := os.Stat(target); err == nil {
			if ts.IsDir() {
				return nil, fmt.Errorf("the update has a file, %s, where the game has a folder", rel)
			}
			f.Replaces = true
			p.Replace++
			p.Kept += ts.Size()
		} else {
			p.Added++
		}
		p.Bytes += f.Size
		area := ""
		if i := strings.Index(rel, "/"); i > 0 {
			area = rel[:i]
		}
		areas[area]++
		p.Files = append(p.Files, f)
	}
	if len(p.Files) == 0 {
		return nil, errors.New("the update has no game files, only notes")
	}
	sort.Slice(p.Files, func(i, j int) bool { return p.Files[i].Rel < p.Files[j].Rel })
	for a, n := range areas {
		p.Where = append(p.Where, UpdateArea{Path: a, Files: n})
	}
	sort.Slice(p.Where, func(i, j int) bool {
		if p.Where[i].Files != p.Where[j].Files {
			return p.Where[i].Files > p.Where[j].Files
		}
		return p.Where[i].Path < p.Where[j].Path
	})
	return p, nil
}

// bestPrefix is the top folder(s) of the update to skip, "" for none: the
// one whose contents match the game's folder best.
func bestPrefix(gameDir string, files []UpdateFile) string {
	have := map[string]bool{}
	if ents, err := os.ReadDir(gameDir); err == nil {
		for _, e := range ents {
			have[strings.ToLower(e.Name())] = true
		}
	}
	score := func(prefix string) (n int) {
		tops := map[string]bool{}
		for _, f := range files {
			if rel, ok := strings.CutPrefix(f.Rel, prefix); ok {
				top := strings.ToLower(strings.SplitN(rel, "/", 2)[0])
				if !tops[top] && have[top] {
					tops[top] = true
					n++
				}
			}
		}
		return n
	}
	best, bestScore := "", score("")
	prefix := ""
	for depth := 0; depth < 2; depth++ {
		// The update's only top folder at this level, if it has one.
		top := ""
		for _, f := range files {
			rel, ok := strings.CutPrefix(f.Rel, prefix)
			if !ok {
				continue
			}
			i := strings.Index(rel, "/")
			if i <= 0 {
				if readmeName(rel) {
					continue
				}
				top = ""
				break
			}
			if top == "" {
				top = rel[:i]
			} else if !strings.EqualFold(top, rel[:i]) {
				top = ""
				break
			}
		}
		if top == "" || have[strings.ToLower(top)] {
			break // files at this level, several folders, or the game has this folder too
		}
		prefix += top + "/"
		if s := score(prefix); s > bestScore || strings.EqualFold(top, filepath.Base(gameDir)) {
			best, bestScore = prefix, s
		}
	}
	return best
}

// ApplyUpdate applies a plan to the game in gameDir, keeping the files it
// replaces in keepDir. If anything fails, the game's folder is put back as
// it was and the error says what failed.
func ApplyUpdate(gameDir string, p *UpdatePlan, keepDir, game string, now time.Time) (*AppliedUpdate, error) {
	u := &AppliedUpdate{ID: filepath.Base(keepDir), Game: game, Source: filepath.Base(p.Source), At: now.UTC(), State: "applying", Dir: keepDir,
		Added: []string{}, Replaced: []string{}}
	if err := os.MkdirAll(filepath.Join(keepDir, "files"), 0o755); err != nil {
		return nil, err
	}
	for _, f := range p.Files {
		if f.Replaces {
			u.Replaced = append(u.Replaced, f.Rel)
		} else {
			u.Added = append(u.Added, f.Rel)
		}
		u.Bytes += f.Size
	}
	// Folders the update makes, so undoing it can take them away again.
	dirSeen := map[string]bool{}
	for _, rel := range u.Added {
		for d := filepath.ToSlash(filepath.Dir(rel)); d != "." && !dirSeen[d]; d = filepath.ToSlash(filepath.Dir(d)) {
			dirSeen[d] = true
			if !exists(filepath.Join(gameDir, filepath.FromSlash(d))) {
				u.Dirs = append(u.Dirs, d)
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(u.Dirs))) // deepest first
	if err := saveUpdate(u); err != nil {
		return nil, err
	}

	// 1. Every file it replaces, kept aside first.
	for _, rel := range u.Replaced {
		from := filepath.Join(gameDir, filepath.FromSlash(rel))
		info, err := os.Stat(from)
		if err != nil {
			return nil, failUpdate(u, gameDir, err)
		}
		if err := copyFile(from, filepath.Join(keepDir, "files", filepath.FromSlash(rel)), info); err != nil {
			return nil, failUpdate(u, gameDir, fmt.Errorf("couldn't keep a copy of %s first: %w", rel, err))
		}
	}

	// 2. The update's files, each read back before it goes in.
	var zr *zip.ReadCloser
	byName := map[string]*zip.File{}
	if p.Kind == "zip" {
		var err error
		if zr, err = zip.OpenReader(p.Source); err != nil {
			return nil, failUpdate(u, gameDir, err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			byName[f.Name] = f
		}
	}
	for _, f := range p.Files {
		target := filepath.Join(gameDir, filepath.FromSlash(f.Rel))
		var err error
		if p.Kind == "zip" {
			zf := byName[f.src]
			if zf == nil {
				err = fmt.Errorf("%s is missing from the .zip", f.src)
			} else {
				err = writeZipFile(zf, target)
			}
		} else {
			var info fs.FileInfo
			if info, err = os.Stat(f.src); err == nil {
				err = placeFile(f.src, target, info)
			}
		}
		if err != nil {
			return nil, failUpdate(u, gameDir, fmt.Errorf("%s: %w", f.Rel, err))
		}
	}
	u.State = "applied"
	return u, saveUpdate(u)
}

// placeFile copies src next to dst, checked, then moves it over dst.
func placeFile(src, dst string, info fs.FileInfo) error {
	tmp := dst + tmpSuffix
	if err := copyFile(src, tmp, info); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return moveInto(tmp, dst)
}

func writeZipFile(zf *zip.File, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := zf.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + tmpSuffix
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	h := sha1.New()
	n, err := io.Copy(out, io.TeeReader(in, h)) // the zip checks its own CRC as it's read
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = verifyCopy(tmp, n, h.Sum(nil))
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chtimes(tmp, zf.Modified, zf.Modified)
	return moveInto(tmp, dst)
}

// moveInto replaces dst with tmp. A read-only game file is made writable
// first, as an update installer would.
func moveInto(tmp, dst string) error {
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Chmod(dst, 0o666)
		if err2 := os.Rename(tmp, dst); err2 != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("%s is in use; close the game and try again (%w)", filepath.Base(dst), err)
		}
	}
	return nil
}

// failUpdate puts the game's folder back after a failed update and returns
// the error that stopped it.
func failUpdate(u *AppliedUpdate, gameDir string, cause error) error {
	if err := revertFiles(u, gameDir); err != nil {
		u.State = "applying" // left as it is: Undo can try again
		_ = saveUpdate(u)
		return fmt.Errorf("%v; putting the files back also failed (%v), so the update's copy of them is kept", cause, err)
	}
	u.State = "undone"
	_ = saveUpdate(u)
	return cause
}

// revertFiles takes the update's added files away and puts the replaced
// ones back from its copy. Files it never got to are left alone.
func revertFiles(u *AppliedUpdate, gameDir string) error {
	var first error
	for _, rel := range u.Added {
		p := filepath.Join(gameDir, filepath.FromSlash(rel))
		_ = os.Remove(p + tmpSuffix)
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) && first == nil {
			first = err
		}
	}
	for _, rel := range u.Replaced {
		kept := filepath.Join(u.Dir, "files", filepath.FromSlash(rel))
		p := filepath.Join(gameDir, filepath.FromSlash(rel))
		_ = os.Remove(p + tmpSuffix)
		info, err := os.Stat(kept)
		if err != nil {
			continue // not kept yet: this file was never touched
		}
		if err := placeFile(kept, p, info); err != nil && first == nil {
			first = err
		}
	}
	for _, d := range u.Dirs {
		_ = os.Remove(filepath.Join(gameDir, filepath.FromSlash(d))) // only when empty
	}
	return first
}

// UndoUpdate takes an applied update back out of the game's folder.
func UndoUpdate(gameDir, keepDir string) (*AppliedUpdate, error) {
	u, err := loadUpdate(keepDir)
	if err != nil {
		return nil, err
	}
	if u.State == "undone" {
		return u, nil
	}
	if err := revertFiles(u, gameDir); err != nil {
		return u, err
	}
	u.State = "undone"
	return u, saveUpdate(u)
}

func saveUpdate(u *AppliedUpdate) error {
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(u.Dir, "update.json"), b)
}

func loadUpdate(dir string) (*AppliedUpdate, error) {
	b, err := os.ReadFile(filepath.Join(dir, "update.json"))
	if err != nil {
		return nil, errors.New("that update's record isn't there any more")
	}
	var u AppliedUpdate
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, err
	}
	u.Dir = dir
	return &u, nil
}

// GameUpdates lists the updates applied to a game, newest first.
func GameUpdates(root string) []AppliedUpdate {
	ents, _ := os.ReadDir(root)
	var out []AppliedUpdate
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if u, err := loadUpdate(filepath.Join(root, e.Name())); err == nil {
			out = append(out, *u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out
}
