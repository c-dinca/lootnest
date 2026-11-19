package plat

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lootnest/internal/core"
)

// Recycle moves files and folders to the Trash, as the freedesktop.org
// Trash specification says, so the file manager shows them there and can
// put them back: the home Trash ($XDG_DATA_HOME/Trash) for what's on the
// home's drive, else that drive's own ($topdir/.Trash/$uid when the
// administrator made it, else $topdir/.Trash-$uid). The .trashinfo is
// written first; a move that fails takes it back, so nothing is lost.
func Recycle(hwnd uintptr, paths ...string) error {
	for _, p := range paths {
		if err := trashOne(p, time.Now()); err != nil {
			return err
		}
	}
	return nil
}

// trashHome is the home Trash; tests replace it.
var trashHome = func() string { return filepath.Join(core.XDGDataHome(home(), os.Getenv), "Trash") }

func trashOne(p string, now time.Time) error {
	abs, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	st, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	trash, rel := trashHome(), abs
	if dev(abs, st) != devOf(filepath.Dir(trash)) {
		top := mountTop(abs)
		if top == "" {
			return fmt.Errorf("no Trash for %s", abs)
		}
		trash, rel = topTrash(top), strings.TrimPrefix(strings.TrimPrefix(abs, top), "/")
	}
	files, info := filepath.Join(trash, "files"), filepath.Join(trash, "info")
	if err := os.MkdirAll(files, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(info, 0o700); err != nil {
		return err
	}
	base := filepath.Base(abs)
	for i := 0; i < 1000; i++ {
		name := base
		if i > 0 {
			ext := filepath.Ext(base)
			name = strings.TrimSuffix(base, ext) + "." + strconv.Itoa(i+1) + ext
		}
		infoPath := filepath.Join(info, name+".trashinfo")
		f, err := os.OpenFile(infoPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return err
		}
		_, werr := fmt.Fprintf(f, "[Trash Info]\nPath=%s\nDeletionDate=%s\n", trashPath(rel), now.Format("2006-01-02T15:04:05"))
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			os.Remove(infoPath)
			return werr
		}
		if _, err := os.Lstat(filepath.Join(files, name)); err == nil {
			os.Remove(infoPath) // a leftover without its info file
			continue
		}
		if err := os.Rename(abs, filepath.Join(files, name)); err != nil {
			os.Remove(infoPath)
			return err
		}
		return nil
	}
	return fmt.Errorf("the Trash has too many files named %s", base)
}

// trashPath is the Path= value: escaped as a URL path, absolute for the
// home Trash, relative to the drive's top for a drive's own Trash.
func trashPath(p string) string { return (&url.URL{Path: p}).EscapedPath() }

func dev(p string, st os.FileInfo) uint64 {
	if s, ok := st.Sys().(*syscall.Stat_t); ok {
		return uint64(s.Dev)
	}
	return 0
}

func devOf(p string) uint64 {
	for ; ; p = filepath.Dir(p) {
		if st, err := os.Stat(p); err == nil {
			return dev(p, st)
		}
		if p == filepath.Dir(p) {
			return 0
		}
	}
}

// mountTop is the top folder of the file system p is on.
func mountTop(p string) string {
	d := devOf(p)
	top := p
	for {
		up := filepath.Dir(top)
		if up == top || devOf(up) != d {
			return top
		}
		top = up
	}
}

// topTrash is a drive's Trash for this user.
func topTrash(top string) string {
	uid := strconv.Itoa(os.Getuid())
	shared := filepath.Join(top, ".Trash")
	if st, err := os.Lstat(shared); err == nil && st.IsDir() && st.Mode()&os.ModeSticky != 0 && st.Mode()&os.ModeSymlink == 0 {
		return filepath.Join(shared, uid)
	}
	return filepath.Join(top, ".Trash-"+uid)
}
