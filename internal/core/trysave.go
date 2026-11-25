package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Try a save safely: a save from a friend or a mod site goes into a run of
// its own (a new profile, Player.Of the person), never over anyone's files.
// The run starts as a copy of the person's files, with one save location
// holding the save they brought instead. Playing the person again is one
// click back; the run can be kept, or deleted like any profile.

// ProfileFromImport starts playerID's profile for a game as a copy of
// from's files, with the save location locID holding src instead: a folder
// (already extracted from an archive), or a single file, which replaces or
// joins the files of from's copy of that folder. Nothing of from's changes.
func (e *Engine) ProfileFromImport(g *Game, playerID, from, locID, src string) error {
	_, err := e.ProfileFromImportTo(g, playerID, from, locID, src)
	return err
}

// TriedSave says where a brought save went, for the person: Where is the
// folder on this PC the game will find it in while the run plays, Files
// the brought files put there (none when a whole folder was), and Left the
// person's saves of that kind left out of the run so the game can't
// continue from them.
type TriedSave struct {
	Where string   `json:"where"`
	Files []string `json:"files,omitempty"`
	Left  []string `json:"left,omitempty"`
}

// ProfileFromImportTo is ProfileFromImport, saying where the save went.
func (e *Engine) ProfileFromImportTo(g *Game, playerID, from, locID, src string) (TriedSave, error) {
	var told TriedSave
	err := e.profileFromImport(g, playerID, from, locID, src, &told)
	return told, err
}

func (e *Engine) profileFromImport(g *Game, playerID, from, locID, src string, told *TriedSave) error {
	if e.HasProfile(playerID, g.ID) {
		return errors.New("that profile already has files for this game")
	}
	if strings.HasPrefix(locID, AchEntryPrefix) {
		return errors.New("choose where the game keeps its saves, not its achievements")
	}
	rs, err := e.resolveAll(g)
	if err != nil {
		return err
	}
	if locID == "" {
		if locID, err = e.saveLocFor(g, from, rs, src); err != nil {
			return err
		}
	}
	var target *resolved
	for i := range rs {
		if rs[i].entry.ID == locID {
			target = &rs[i]
		}
	}
	if target == nil {
		return errors.New("that save location isn't one of the game's any more")
	}
	st, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("the save to try isn't there any more: %w", err)
	}
	liveIsFile := looksLikeFile(target.entry.Path)
	if lst, err := os.Stat(target.live); err == nil {
		liveIsFile = !lst.IsDir()
	}
	// Loose save files (an archive of SRDR30000 and SRDR30001) for a folder
	// that keeps saves like them deeper (Profiles\<id>) join that folder,
	// as one file does.
	var loose []string
	if st.IsDir() && !liveIsFile {
		loose = looseSaves(src, fromSaves(e, g, from, target))
	}
	if len(loose) == 1 {
		src, loose = loose[0], nil
	}
	if st.IsDir() && loose == nil {
		if one := onlyFile(src); one != "" && (liveIsFile || filepath.Dir(one) == importRoot(src, target.live)) {
			src = one
		} else if liveIsFile {
			return fmt.Errorf("%s keeps that save as a single file; choose a file, not a folder", g.Name)
		} else {
			src = importRoot(src, target.live)
		}
		if st, err = os.Stat(src); err != nil {
			return err
		}
	}

	active := e.Store.LoadState()[g.ID]
	fromLive := from != "" && from == active
	dst := e.Store.ProfileDir(playerID, g.ID)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	fail := func(err error) error {
		_ = os.RemoveAll(dst)
		return err
	}
	fromCopy := func(r resolved) string {
		if fromLive {
			return r.live
		}
		if from != "" && e.HasProfile(from, g.ID) {
			return filepath.Join(e.Store.ProfileDir(from, g.ID), r.entry.ID)
		}
		return ""
	}
	for _, r := range rs {
		out := filepath.Join(dst, r.entry.ID)
		if r.entry.ID != locID {
			if c := fromCopy(r); c != "" {
				if err := Mirror(c, out); err != nil {
					return fail(err)
				}
			}
			continue
		}
		switch {
		case (st.IsDir() && loose == nil) || liveIsFile:
			if err := Mirror(src, out); err != nil {
				return fail(err)
			}
			told.Where = target.live
		default:
			// Save files into a folder: the person's folder, with the
			// brought saves beside the saves like them. The person's other
			// saves of that kind (slots, autosaves) stay out of the run, so
			// the game's Continue can only pick the brought one; the
			// person's own profile keeps all of them.
			files := loose
			if files == nil {
				files = []string{src}
			}
			if c := fromCopy(r); c != "" {
				if err := Mirror(c, out); err != nil {
					return fail(err)
				}
			}
			if err := os.MkdirAll(out, 0o755); err != nil {
				return fail(err)
			}
			home := saveHome(out, filepath.Base(files[0]))
			told.Left = dropLikeSaves(home, files)
			for _, f := range files {
				if err := Mirror(f, filepath.Join(home, filepath.Base(f))); err != nil {
					return fail(err)
				}
				told.Files = append(told.Files, filepath.Base(f))
			}
			told.Where = target.live
			if rel, err := filepath.Rel(out, home); err == nil && rel != "." {
				told.Where = filepath.Join(target.live, rel)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(dst, initMarker), []byte(e.now().Format(time.RFC3339)), 0o644); err != nil {
		return fail(err)
	}
	e.logf("Started %s's files for %s from %s, with %s in %s (left out of the run: %s)", e.name(playerID), g.Name, e.name(from), filepath.Base(src), told.Where, orNoneList(told.Left))
	return nil
}

// ErrWhichSaveFolder: the game has several folders and none holds a save
// like the brought one, so where it goes is the person's choice.
var ErrWhichSaveFolder = errors.New("no folder holds a save like it")

// saveLocFor is the save location a brought save goes to when the person
// didn't choose one: the folder that already holds saves like it (same
// name, then shape, then extension), wherever it is in the game's list.
// Red Dead Redemption 2's Settings folder may be listed before the folder
// its saves are in, and a save put there is never read. With no folder
// like it, the game's only folder, else ErrWhichSaveFolder.
func (e *Engine) saveLocFor(g *Game, from string, rs []resolved, src string) (string, error) {
	name := broughtName(src)
	var folders []*resolved
	best, rank := "", 0
	for i := range rs {
		r := &rs[i]
		if strings.HasPrefix(r.entry.ID, AchEntryPrefix) {
			continue
		}
		isFile := looksLikeFile(r.entry.Path)
		if st, err := os.Stat(r.live); err == nil {
			isFile = !st.IsDir()
		}
		if isFile {
			if name != "" && strings.EqualFold(filepath.Base(r.live), name) && rank < 3 {
				best, rank = r.entry.ID, 3
			}
			continue
		}
		folders = append(folders, r)
		if name == "" {
			continue
		}
		if _, n := saveHomeRank(fromSaves(e, g, from, r), name); n > rank {
			best, rank = r.entry.ID, n
		}
	}
	switch {
	case best != "":
		return best, nil
	case len(folders) == 1:
		return folders[0].entry.ID, nil
	case len(folders) == 0:
		return "", errors.New("choose where the game keeps its saves")
	}
	what := "a save like " + name
	if name == "" {
		what = "a save like it"
	}
	return "", whichFolder(fmt.Sprintf("None of %s's folders has %s yet, so Lootnest can't tell where it goes. Choose the folder under Where the save goes.", g.Name, what))
}

func orNoneList(v []string) string {
	if len(v) == 0 {
		return "none"
	}
	return strings.Join(v, ", ")
}

// whichFolder is ErrWhichSaveFolder said for one game and save.
type whichFolder string

func (w whichFolder) Error() string        { return string(w) }
func (w whichFolder) Is(target error) bool { return target == ErrWhichSaveFolder }

// broughtName is the name of the brought save's first file: the file
// itself, or the first one in the folder, through wrapping folders.
func broughtName(src string) string {
	st, err := os.Stat(src)
	if err != nil {
		return ""
	}
	if !st.IsDir() {
		return filepath.Base(src)
	}
	name := ""
	_ = filepath.WalkDir(throughWrappers(src), func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			name = d.Name()
			return fs.SkipAll
		}
		return nil
	})
	return name
}

// saveHome is the folder in dir, at most three folders down, that holds
// saves like name: one with the same name, else the same shape
// (SRDR30005 beside SRDR30000 in Red Dead Redemption 2's Profiles\<id>),
// else the same extension. With none, dir itself.
func saveHome(dir, name string) string {
	home, _ := saveHomeRank(dir, name)
	return home
}

// saveHomeRank is saveHome with how alike its saves are: 3 the same name,
// 2 the same shape, 1 the same extension, 0 none (dir itself).
func saveHomeRank(dir, name string) (string, int) {
	want, shape, ext := strings.ToLower(name), saveShape(name), strings.ToLower(filepath.Ext(name))
	best, rank := dir, 0
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if rel, _ := filepath.Rel(dir, p); strings.Count(rel, string(filepath.Separator)) >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		n := strings.ToLower(d.Name())
		r := 0
		switch {
		case n == want:
			r = 3
		case strings.Contains(shape, "#") && saveShape(n) == shape:
			r = 2
		case ext != "" && filepath.Ext(n) == ext:
			r = 1
		}
		if r > rank {
			best, rank = filepath.Dir(p), r
		}
		return nil
	})
	return best, rank
}

// fromSaves is the folder holding from's copy of the save location: their
// profile's, or the live one when theirs are the files in place.
func fromSaves(e *Engine, g *Game, from string, target *resolved) string {
	if from != "" && e.Store.LoadState()[g.ID] == from {
		return target.live
	}
	if from != "" && e.HasProfile(from, g.ID) {
		return filepath.Join(e.Store.ProfileDir(from, g.ID), target.entry.ID)
	}
	return target.live
}

// looseSaves lists the files at the top of src when it holds only files
// (no folders) and the person's save folder keeps files shaped like the
// first one in a folder below its top, as Red Dead Redemption 2 keeps
// SRDR30000 in Profiles\<id>. Otherwise nil: the folder is taken as it is.
func looseSaves(src, person string) []string {
	src = throughWrappers(src)
	es, err := os.ReadDir(src)
	if err != nil || len(es) == 0 {
		return nil
	}
	var files []string
	for _, en := range es {
		if !en.Type().IsRegular() {
			return nil
		}
		files = append(files, filepath.Join(src, en.Name()))
	}
	if person == "" {
		return nil
	}
	if home := saveHome(person, filepath.Base(files[0])); filepath.Clean(home) == filepath.Clean(person) {
		return nil
	}
	return files
}

// throughWrappers goes down through folders that hold nothing but one
// folder ("Prologue Skip/Saves/SRDR30000" becomes ".../Saves").
func throughWrappers(dir string) string {
	for i := 0; i < 8; i++ {
		es, err := os.ReadDir(dir)
		if err != nil || len(es) != 1 || !es[0].IsDir() {
			return dir
		}
		dir = filepath.Join(dir, es[0].Name())
	}
	return dir
}

// dropLikeSaves takes out of a run's folder the saves shaped like the
// brought ones (SRDR30003 beside a brought SRDR30000) that aren't brought
// themselves, with their copies the game keeps beside them (SRDR30003.bak),
// which it could load instead. Only names with digits count, so settings
// files stay.
func dropLikeSaves(home string, brought []string) (left []string) {
	shapes, names := map[string]bool{}, map[string]bool{}
	for _, f := range brought {
		n := strings.ToLower(filepath.Base(f))
		names[n] = true
		if s := saveShape(n); strings.Contains(s, "#") {
			shapes[s] = true
		}
	}
	es, err := os.ReadDir(home)
	if err != nil {
		return nil
	}
	for _, en := range es {
		n := strings.ToLower(en.Name())
		if !en.Type().IsRegular() || names[n] {
			continue
		}
		bare := strings.TrimSuffix(n, filepath.Ext(n))
		if shapes[saveShape(n)] || (bare != n && shapes[saveShape(bare)]) {
			if os.Remove(filepath.Join(home, en.Name())) == nil {
				left = append(left, en.Name())
			}
		}
	}
	return left
}

// looksLikeFile says whether a save path names a file (a name with an
// extension such as .ini or .sav) rather than a folder.
func looksLikeFile(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext != "" && len(ext) <= 6 && !strings.ContainsAny(ext, " ")
}

// onlyFile is the path of the one regular file under dir, at any depth,
// or "" when dir holds none or more than one.
func onlyFile(dir string) string {
	one := ""
	n := 0
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			one, n = p, n+1
		}
		if n > 1 {
			return fs.SkipAll
		}
		return nil
	})
	if n != 1 {
		return ""
	}
	return one
}

// importRoot is the folder inside src that matches a save folder: an
// archive often holds one folder around the saves ("MySave/slot1.sav").
// It goes down through single folders the live folder has no entry for.
// When the live folder holds only folders (save slots), it goes down only
// into a folder holding something the live folder has, so a lone slot
// stays a slot.
func importRoot(src, live string) string {
	have := map[string]bool{}
	slots := false
	if es, err := os.ReadDir(live); err == nil {
		slots = len(es) > 0
		for _, e := range es {
			slots = slots && e.IsDir()
			have[strings.ToLower(e.Name())] = true
		}
	}
	known := func(e os.DirEntry) bool { return have[strings.ToLower(e.Name())] }
	for i := 0; i < 4; i++ {
		es, err := os.ReadDir(src)
		if err != nil || len(es) != 1 || !es[0].IsDir() || known(es[0]) {
			return src
		}
		next := filepath.Join(src, es[0].Name())
		if kids, _ := os.ReadDir(next); slots && !slices.ContainsFunc(kids, known) {
			return src
		}
		src = next
	}
	return src
}
