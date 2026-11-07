package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Putting back a copy made before a swap. Each copy is the files that were
// in place then, and whose they were (owner.txt). Putting one back gives
// them back to that profile: into the game's folders if that profile's
// files are loaded, else into the profile. Whatever is there now is copied
// first, so putting a copy back can itself be undone.
// ---------------------------------------------------------------------------

// BackupID is a copy's folder name, as Backups lists it.
func BackupID(b BackupInfo) string { return filepath.Base(b.Dir) }

// RestoreBackup puts the copy id of gameID back for the profile it belongs
// to, and reports whose it was and where it went ("live" or "profile").
func (e *Engine) RestoreBackup(cfg *Config, gameID, id string) (owner, where string, err error) {
	g := cfg.Game(gameID)
	if g == nil {
		return "", "", errors.New("unknown game")
	}
	if id == "" || id != filepath.Base(id) || strings.HasPrefix(id, ".") {
		return "", "", errors.New("unknown copy")
	}
	dir := filepath.Join(e.Store.BackupsDir(), gameID, id)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", "", errors.New("that copy is gone")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "owner.txt"))
	owner = strings.TrimSpace(string(b))
	if owner == "" || cfg.Player(owner) == nil {
		return "", "", errors.New("Lootnest doesn't know whose files that copy holds")
	}
	if _, pending := e.Store.PendingSwaps()[g.ID]; pending {
		return "", "", fmt.Errorf("the last swap of %s didn't finish; start Lootnest again first", g.Name)
	}
	rs, err := e.resolveAll(g)
	if err != nil {
		return "", "", err
	}
	if e.Store.LoadState()[g.ID] == owner {
		// Their files are in place: the files there now become a copy too.
		// No pruning here: it could take the very copy being put back.
		if _, err := e.Backup(g, 1<<30); err != nil {
			return "", "", err
		}
		for _, r := range rs {
			if err := Mirror(filepath.Join(dir, r.entry.ID), r.live); err != nil {
				return owner, "live", err
			}
		}
		e.logf("Put back %s's files for %s from the copy of %s", e.name(owner), g.Name, id)
		return owner, "live", nil
	}
	// Their files are in their profile: keep the profile as it is now first.
	prof := e.Store.ProfileDir(owner, g.ID)
	if HasFiles(prof) {
		kept := filepath.Join(e.Store.BackupsDir(), g.ID, e.now().Format("20060102-150405")+"-profile")
		for n := 2; exists(kept); n++ {
			kept = filepath.Join(e.Store.BackupsDir(), g.ID, fmt.Sprintf("%s-profile-%d", e.now().Format("20060102-150405"), n))
		}
		for _, r := range rs {
			if err := Mirror(filepath.Join(prof, r.entry.ID), filepath.Join(kept, r.entry.ID)); err != nil {
				return owner, "profile", err
			}
		}
		_ = os.WriteFile(filepath.Join(kept, "owner.txt"), []byte(owner), 0o644)
	}
	if err := os.MkdirAll(prof, 0o755); err != nil {
		return owner, "profile", err
	}
	for _, r := range rs {
		if err := Mirror(filepath.Join(dir, r.entry.ID), filepath.Join(prof, r.entry.ID)); err != nil {
			return owner, "profile", err
		}
	}
	if err := os.WriteFile(filepath.Join(prof, initMarker), []byte(e.now().Format(time.RFC3339)), 0o644); err != nil {
		return owner, "profile", err
	}
	e.logf("Put back %s's profile for %s from the copy of %s", e.name(owner), g.Name, id)
	return owner, "profile", nil
}
