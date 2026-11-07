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
	if st.IsDir() {
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
		case st.IsDir() || liveIsFile:
			if err := Mirror(src, out); err != nil {
				return fail(err)
			}
		default:
			// One file into a folder: the person's folder, with that file
			// added or replaced.
			if c := fromCopy(r); c != "" {
				if err := Mirror(c, out); err != nil {
					return fail(err)
				}
			}
			if err := os.MkdirAll(out, 0o755); err != nil {
				return fail(err)
			}
			if err := Mirror(src, filepath.Join(out, filepath.Base(src))); err != nil {
				return fail(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(dst, initMarker), []byte(e.now().Format(time.RFC3339)), 0o644); err != nil {
		return fail(err)
	}
	e.logf("Started %s's files for %s from %s, with %s in %s", e.name(playerID), g.Name, e.name(from), filepath.Base(src), target.entry.Path)
	return nil
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
