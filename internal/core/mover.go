package core

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Game Mover. Moves an installed Steam game to another of the PC's Steam
// libraries, the way Steam's own Storage settings do, while Steam is closed.
//
// Nothing is deleted until the whole game is copied and checked: the copy
// goes into "<folder>.lootnest-part" in the new library, is compared file by
// file with the original, and only then gets its real name. Steam's records
// for the game move next, and the old copy is removed last.
// ---------------------------------------------------------------------------

// SteamInstall is where an installed Steam game lives.
type SteamInstall struct {
	AppID      int
	Name       string
	Library    string // the library folder (contains steamapps)
	InstallDir string // folder name under steamapps\common
	Manifest   string // steamapps\appmanifest_<id>.acf
}

// Dir is the game's folder.
func (in SteamInstall) Dir() string {
	return filepath.Join(in.Library, "steamapps", "common", in.InstallDir)
}

// Size is the bytes the game and its workshop items take up.
func (in SteamInstall) Size() (int64, error) {
	_, n, err := TreeSize(in.Dir())
	if err != nil {
		return 0, err
	}
	ws := filepath.Join(in.Library, "steamapps", "workshop", "content", strconv.Itoa(in.AppID))
	if st, err := os.Stat(ws); err == nil && st.IsDir() {
		_, w, err := TreeSize(ws)
		if err != nil {
			return 0, err
		}
		n += w
	}
	return n, nil
}

const partSuffix = ".lootnest-part"

// FindSteamInstall finds the library that holds an installed Steam game.
func FindSteamInstall(steamPath string, appID int) (SteamInstall, error) {
	for _, lib := range SteamLibraries(steamPath) {
		mf := filepath.Join(lib, "steamapps", "appmanifest_"+strconv.Itoa(appID)+".acf")
		b, err := os.ReadFile(mf)
		if err != nil {
			continue
		}
		m, err := ParseVDF(string(b))
		if err != nil {
			continue
		}
		st := vdfMap(m, "AppState")
		dir := vdfStr(st, "installdir")
		if st == nil || dir == "" || strings.ContainsAny(dir, `/\`) || dir == "." || dir == ".." {
			continue
		}
		flags, _ := strconv.Atoi(vdfStr(st, "StateFlags"))
		if flags&4 == 0 {
			return SteamInstall{}, fmt.Errorf("%s isn't fully installed. Let Steam finish downloading or updating it, then move it", vdfStr(st, "name"))
		}
		return SteamInstall{AppID: appID, Name: vdfStr(st, "name"), Library: lib, InstallDir: dir, Manifest: mf}, nil
	}
	return SteamInstall{}, errors.New("Steam doesn't list this game as installed on this PC")
}

// TreeSize counts the files and bytes in a folder. Links are refused: a
// game folder with links in it isn't one Lootnest can safely copy.
func TreeSize(dir string) (files int, bytes int64, err error) {
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 || d.Type()&fs.ModeIrregular != 0 {
			return fmt.Errorf("%s is a link, which Lootnest doesn't move", p)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, bytes, err
}

// moveCopyTree copies src into dst (which must not exist yet), flushing every
// file to disk and reading it back. done is called with the bytes copied so
// far; stop, if set, cancels the copy when it says so.
func moveCopyTree(src, dst string, done func(int64), stop func() bool) error {
	var total int64
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		if d.Type()&fs.ModeSymlink != 0 || d.Type()&fs.ModeIrregular != 0 {
			return fmt.Errorf("%s is a link, which Lootnest doesn't move", p)
		}
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		n, err := copyFileSynced(p, to, info, func(n int64) {
			if done != nil {
				done(total + n)
			}
		}, stop)
		total += n
		if err != nil {
			return err
		}
		if n != info.Size() {
			return fmt.Errorf("%s changed while it was being copied", p)
		}
		return nil
	})
}

func copyFileSynced(from, to string, info fs.FileInfo, progress func(int64), stop func() bool) (int64, error) {
	in, err := os.Open(from)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	var n int64
	h := sha1.New()
	buf := make([]byte, 1<<20)
	for {
		if stop != nil && stop() {
			out.Close()
			return n, ErrMoveCancelled
		}
		k, rerr := in.Read(buf)
		if k > 0 {
			h.Write(buf[:k])
			if _, werr := out.Write(buf[:k]); werr != nil {
				out.Close()
				return n, werr
			}
			n += int64(k)
			progress(n)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return n, rerr
		}
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return n, err
	}
	if err := out.Close(); err != nil {
		return n, err
	}
	if err := verifyCopy(to, n, h.Sum(nil)); err != nil {
		return n, err
	}
	_ = os.Chtimes(to, info.ModTime(), info.ModTime())
	return n, nil
}

// sameTree checks that b holds the same files, with the same sizes, as a.
func sameTree(a, b string) error {
	sizes := map[string]int64{}
	err := filepath.WalkDir(a, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(a, p)
		sizes[strings.ToLower(rel)] = info.Size()
		return nil
	})
	if err != nil {
		return err
	}
	err = filepath.WalkDir(b, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(b, p)
		want, ok := sizes[strings.ToLower(rel)]
		if !ok || want != info.Size() {
			return fmt.Errorf("the copy of %s doesn't match the original", rel)
		}
		delete(sizes, strings.ToLower(rel))
		return nil
	})
	if err != nil {
		return err
	}
	for rel := range sizes {
		return fmt.Errorf("%s is missing from the copy", rel)
	}
	return nil
}

// copyChecked copies src to dst through dst+".lootnest-part" and checks the
// copy before giving it its real name.
func copyChecked(src, dst string, done func(int64), stop func() bool) error {
	part := dst + partSuffix
	_ = os.RemoveAll(part) // left over from a move that was interrupted
	if err := os.MkdirAll(filepath.Dir(part), 0o755); err != nil {
		return err
	}
	if err := moveCopyTree(src, part, done, stop); err != nil {
		_ = os.RemoveAll(part)
		return err
	}
	if err := sameTree(src, part); err != nil {
		_ = os.RemoveAll(part)
		return err
	}
	if err := os.Rename(part, dst); err != nil {
		_ = os.RemoveAll(part)
		return err
	}
	return nil
}

// MoveResult says what a move left behind.
type MoveResult struct {
	From, To string
	// Leftover is set when the game moved but some of the old files couldn't
	// be removed (a program had one open). The game itself is fine.
	Leftover string
}

// MoveSteamGame moves an installed Steam game into another Steam library.
// Steam must be closed. done reports bytes copied so far. check, if set,
// runs once the copy is done and before anything in the old library is
// touched; an error from it (Steam or the game was started meanwhile)
// cancels the move.
func MoveSteamGame(in SteamInstall, toLib string, done func(int64), check func() error) (MoveResult, error) {
	res := MoveResult{From: in.Dir()}
	if samePath(in.Library, toLib) {
		return res, errors.New("the game is already in that library")
	}
	toApps := filepath.Join(toLib, "steamapps")
	if st, err := os.Stat(toApps); err != nil || !st.IsDir() {
		return res, fmt.Errorf("%s isn't a Steam library", toLib)
	}
	id := strconv.Itoa(in.AppID)
	dst := filepath.Join(toApps, "common", in.InstallDir)
	res.To = dst
	newManifest := filepath.Join(toApps, "appmanifest_"+id+".acf")
	if _, err := os.Lstat(dst); err == nil {
		return res, fmt.Errorf("that library already has a folder named %s (%s). Remove or rename it, then try again", in.InstallDir, dst)
	}
	if _, err := os.Lstat(newManifest); err == nil {
		return res, fmt.Errorf("Steam already lists this game in %s", toLib)
	}
	manifest, err := os.ReadFile(in.Manifest)
	if err != nil {
		return res, err
	}

	// Workshop items for the game, if it has any.
	fromApps := filepath.Join(in.Library, "steamapps")
	wsDir := filepath.Join(fromApps, "workshop", "content", id)
	wsDst := filepath.Join(toApps, "workshop", "content", id)
	wsACF := filepath.Join(fromApps, "workshop", "appworkshop_"+id+".acf")
	wsACFDst := filepath.Join(toApps, "workshop", "appworkshop_"+id+".acf")
	hasWS := false
	if st, err := os.Stat(wsDir); err == nil && st.IsDir() {
		if _, err := os.Lstat(wsDst); err == nil {
			return res, fmt.Errorf("that library already has workshop items for this game (%s). Remove them, then try again", wsDst)
		}
		hasWS = true
	}
	wsManifest, wsErr := os.ReadFile(wsACF)

	// 1. Copy and check. Nothing is changed in the old library yet.
	var gameBytes int64
	if err := copyChecked(in.Dir(), dst, func(n int64) {
		gameBytes = n
		if done != nil {
			done(n)
		}
	}, nil); err != nil {
		return res, fmt.Errorf("copying the game failed (%v). It stays where it was", err)
	}
	undo := func() {
		_ = os.RemoveAll(dst)
		if hasWS {
			_ = os.RemoveAll(wsDst)
		}
		_ = os.Remove(newManifest)
		_ = os.Remove(wsACFDst)
	}
	if hasWS {
		if err := copyChecked(wsDir, wsDst, func(n int64) {
			if done != nil {
				done(gameBytes + n)
			}
		}, nil); err != nil {
			undo()
			return res, fmt.Errorf("copying the game's workshop items failed (%v). The game stays where it was", err)
		}
	}

	if check != nil {
		if err := check(); err != nil {
			undo()
			return res, fmt.Errorf("%w. The game stays where it was", err)
		}
	}

	// 2. Steam's records: write the new ones, then remove the old manifest,
	// so Steam never sees the game in neither library.
	if err := writeFileAtomic(newManifest, manifest); err != nil {
		undo()
		return res, fmt.Errorf("writing Steam's record for the game failed (%v). It stays where it was", err)
	}
	if wsErr == nil {
		if err := writeFileAtomic(wsACFDst, wsManifest); err != nil {
			undo()
			return res, fmt.Errorf("writing Steam's record for the workshop items failed (%v). The game stays where it was", err)
		}
	}
	if err := os.Remove(in.Manifest); err != nil {
		undo()
		return res, fmt.Errorf("updating Steam's records failed (%v). The game stays where it was", err)
	}
	if wsErr == nil {
		_ = os.Remove(wsACF)
	}

	// 3. The old copy goes last.
	if err := os.RemoveAll(in.Dir()); err != nil {
		res.Leftover = in.Dir()
	}
	if hasWS {
		if err := os.RemoveAll(wsDir); err != nil && res.Leftover == "" {
			res.Leftover = wsDir
		}
	}
	return res, nil
}

func writeFileAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".lootnest-tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// RetargetPaths points a game's swapped locations that were inside from at
// the same place inside to. It reports whether anything changed.
func (e *Env) RetargetPaths(g *Game, from, to string) bool {
	changed := false
	for i, pe := range g.Paths {
		p, err := e.expand(pe.Path)
		if err != nil || !isWithin(p, from) {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(from), filepath.Clean(p))
		if err != nil {
			continue
		}
		g.Paths[i].Path = e.Tokenize(filepath.Join(to, rel))
		changed = true
	}
	return changed
}
