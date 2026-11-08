package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Move a folder game to another drive. The folder is copied into
// <destination>\<its name> through a ".lootnest-part" folder, every file is
// read back, and the copy is compared with the original (each file and its
// size) before it gets its real name. The app then points the Library at
// the new folder and only after that sends the old one to the Recycle Bin.
// If anything fails, the copy is removed and the game stays where it was.
// Games a store or launcher installed are moved by that launcher, which
// keeps track of where they are (Steam games by Game Mover, mover.go).
// ---------------------------------------------------------------------------

// GameMove is moving one folder game.
type GameMove struct {
	Name  string `json:"name"`
	From  string `json:"from"`
	To    string `json:"to"`
	Size  int64  `json:"size"`
	Files int    `json:"files"`
}

// ErrMoveCancelled: the user stopped the move while it copied.
var ErrMoveCancelled = errors.New("the move was cancelled")

// moveRoom is the space a move needs on the new drive: the game and a
// tenth more.
func moveRoom(size int64) int64 { return size + size/10 }

// MoveGamePlan checks a folder game can move into the folder dst (on
// another drive, usually), with free bytes there (-1 when Windows can't
// say), and counts what it copies.
func (e *Env) MoveGamePlan(name, source, src, dst string, running bool, free int64) (GameMove, error) {
	if source == "shortcut" {
		return GameMove{}, fmt.Errorf("%s starts from a shortcut, which would still point at the old folder. Move it yourself, then add it again.", name)
	}
	switch source {
	case "folder", "added", "":
	default:
		return GameMove{}, fmt.Errorf("Its store keeps track of where %s is installed, so move it there.", name)
	}
	if running {
		return GameMove{}, fmt.Errorf("%s is running. Close it first.", name)
	}
	if src == "" || !filepath.IsAbs(src) || !isDir(src) {
		return GameMove{}, errors.New("This game has no folder of its own on this PC.")
	}
	src = filepath.Clean(src)
	if filepath.Dir(src) == src || tooBroadDir(src) {
		return GameMove{}, fmt.Errorf("Lootnest won't move %s: it holds more than one game.", src)
	}
	if err := e.CheckSafe(src); err != nil {
		return GameMove{}, fmt.Errorf("Lootnest won't move %s: %v.", src, err)
	}
	if dst == "" || !filepath.IsAbs(dst) {
		return GameMove{}, errors.New("Choose where the game goes.")
	}
	dst = filepath.Clean(dst)
	if isWithin(dst, src) {
		return GameMove{}, errors.New("That folder is inside the game's own folder. Choose one somewhere else.")
	}
	to := filepath.Join(dst, filepath.Base(src))
	if e.DataDir != "" && isWithin(to, e.DataDir) {
		return GameMove{}, errors.New("That folder is inside Lootnest's own data folder. Choose another.")
	}
	if _, err := os.Lstat(to); err == nil {
		return GameMove{}, fmt.Errorf("%s is already there. Rename or remove it, or choose another folder.", to)
	}
	files, size, err := TreeSize(src)
	if err != nil {
		return GameMove{}, fmt.Errorf("Lootnest can't move this game: %v.", err)
	}
	if free >= 0 && free < moveRoom(size) {
		return GameMove{}, fmt.Errorf("There isn't enough room on %s. %s needs about %s and %s is free.", volumeOf(dst), name, aboutGB(moveRoom(size)), aboutGB(free))
	}
	return GameMove{Name: name, From: src, To: to, Size: size, Files: files}, nil
}

// MoveGame copies the game into m.To and checks the copy. done reports the
// bytes copied so far; stop, when it says so, cancels the move. On an error
// nothing is left at m.To, and the original is never touched.
func MoveGame(m GameMove, done func(int64), stop func() bool) error {
	if err := copyChecked(m.From, m.To, done, stop); err != nil {
		return err
	}
	files, size, err := TreeSize(m.To)
	if err == nil && (files != m.Files || size != m.Size) {
		err = fmt.Errorf("the copy has %d files (%s), the game %d (%s)", files, aboutGB(size), m.Files, aboutGB(m.Size))
	}
	if err != nil {
		_ = removeAll(m.To)
		return err
	}
	return nil
}

func volumeOf(p string) string {
	if v := filepath.VolumeName(p); v != "" {
		return v
	}
	return p
}

func aboutGB(n int64) string {
	if n < 1<<30 {
		return fmt.Sprintf("%d MB", max(1, n>>20))
	}
	return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
}
