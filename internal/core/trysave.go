package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	if st.IsDir() && liveIsFile {
		return fmt.Errorf("%s keeps that save as a single file; choose a file, not a folder", g.Name)
	}
	if st.IsDir() {
		src = importRoot(src, target.live)
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

// importRoot is the folder inside src that matches a save folder: an
// archive often holds one folder around the saves ("MySave/slot1.sav").
// It goes down through single folders the live folder has no entry for.
func importRoot(src, live string) string {
	have := map[string]bool{}
	if es, err := os.ReadDir(live); err == nil {
		for _, e := range es {
			have[strings.ToLower(e.Name())] = true
		}
	}
	for i := 0; i < 4; i++ {
		es, err := os.ReadDir(src)
		if err != nil || len(es) != 1 || !es[0].IsDir() || have[strings.ToLower(es[0].Name())] {
			return src
		}
		src = filepath.Join(src, es[0].Name())
	}
	return src
}
