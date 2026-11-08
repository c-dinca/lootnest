package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Remove a folder game from this PC: its saves are copied to Lootnest's
// backups first (backups\removed\<game>\<time>), read back, then the app
// moves the folder to the Recycle Bin. Nothing is deleted for good here.
// Added again later, the copies ledger offers its kept saves back.
// ---------------------------------------------------------------------------

// RemovePlan is what removing a game does.
type RemovePlan struct {
	Name  string     `json:"name"`
	Dir   string     `json:"dir"`
	Keep  string     `json:"keep"` // where its saves are copied
	Saves []KeptSave `json:"saves"`
	Size  int64      `json:"size"` // of the folder that goes to the Recycle Bin
}

// KeptSave is one save location copied before the folder goes.
type KeptSave struct {
	From string `json:"from"`
	To   string `json:"to"`
	Rel  string `json:"rel,omitempty"` // inside the game's folder, with "/"; "" for one elsewhere
	Size int64  `json:"size"`
}

// PlanRemove checks a folder game can go and lists the saves to copy.
// It refuses while the game runs, and for any folder that holds more than
// one game's files: a drive, Program Files, the user's home or Desktop.
func (e *Env) PlanRemove(name, dir string, saves []SaveLocation, backups string, running bool, now time.Time) (RemovePlan, error) {
	if running {
		return RemovePlan{}, fmt.Errorf("%s is running. Close it first.", name)
	}
	if dir == "" || !filepath.IsAbs(dir) {
		return RemovePlan{}, errors.New("This game has no folder of its own to remove.")
	}
	dir = filepath.Clean(dir)
	if !isDir(dir) {
		return RemovePlan{}, errors.New("Its folder isn't on this PC any more.")
	}
	if err := e.CheckSafe(dir); err != nil {
		return RemovePlan{}, fmt.Errorf("Lootnest won't remove %s: %v.", dir, err)
	}
	if tooBroadDir(dir) {
		return RemovePlan{}, fmt.Errorf("Lootnest won't remove %s: it holds more than one game.", dir)
	}
	if backups == "" || isWithin(backups, dir) {
		return RemovePlan{}, errors.New("Lootnest's backups are inside this game's folder, so it can't be removed from here.")
	}
	key := CoverKey(name)
	if key == "" {
		key = "game"
	}
	p := RemovePlan{Name: name, Dir: dir, Keep: filepath.Join(backups, "removed", key, now.Format("20060102-150405")), Saves: []KeptSave{}, Size: Size(dir)}
	for i, l := range saves {
		if !HasFiles(l.Resolved) || samePath(l.Resolved, dir) {
			continue
		}
		k := KeptSave{From: l.Resolved, Size: Size(l.Resolved)}
		if rel, err := filepath.Rel(dir, l.Resolved); err == nil && isWithin(l.Resolved, dir) {
			k.Rel = filepath.ToSlash(rel)
			k.To = filepath.Join(p.Keep, "folder", rel)
		} else {
			k.To = filepath.Join(p.Keep, "other", strconv.Itoa(i+1)+" "+filepath.Base(l.Resolved))
		}
		p.Saves = append(p.Saves, k)
	}
	return p, nil
}

// CopySaves copies every save to the plan's Keep folder, each read back
// (Mirror), and writes removed.json there, which says where each came
// from. On an error the folder must stay.
func (p RemovePlan) CopySaves() error {
	for _, k := range p.Saves {
		if err := Mirror(k.From, k.To); err != nil {
			return fmt.Errorf("couldn't copy %s: %w", k.From, err)
		}
	}
	if err := os.MkdirAll(p.Keep, 0o755); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(p.Keep, "removed.json"), p)
}

// KeptSaves are the saves inside the folder, by their place in it, as
// the ledger keeps them for a copy added again.
func (p RemovePlan) KeptSaves() map[string]string {
	out := map[string]string{}
	for _, k := range p.Saves {
		if k.Rel != "" && !strings.HasPrefix(k.Rel, "..") {
			out[k.Rel] = k.To
		}
	}
	return out
}
