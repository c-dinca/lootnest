//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Updates already in the user's Downloads folder (core/downloads.go). Every
// two minutes, and when the window gains focus, Lootnest matches the names
// there to the games the Updates page covers, and tells about a newer
// version once. Applying one is the update drawer's job, on the user's
// click. The folder Hydra downloads into is looked in too when the user
// chose one in Hydra (the owner, 14 Nov 2026). Lootnest looks nowhere
// else and fetches nothing.
// ---------------------------------------------------------------------------

var (
	dlBusy  atomic.Bool
	dlAgain atomic.Bool // asked while a check ran: look once more after it
	dlMu    sync.Mutex
	dlLast  []core.DownloadMatch // the last check's matches: the only files Ignore and Recycle take
)

// apiDownloads looks now; the answer is a "downloadsFound" event.
func (a *App) apiDownloads() { go a.downloadsCheck() }

func (a *App) downloadsCheck() {
	dlAgain.Store(true)
	for dlAgain.Load() && dlBusy.CompareAndSwap(false, true) {
		dlAgain.Store(false)
		a.downloadsLook()
		dlBusy.Store(false)
	}
}

func (a *App) downloadsLook() {
	a.mu.Lock()
	off := a.cfg.Settings.DownloadsOff
	a.mu.Unlock()
	byKey := map[string]updTarget{}
	var found []core.DownloadMatch
	if dirs := downloadDirs(); !off && len(dirs) > 0 {
		ts := a.updTargets()
		targets := make([]core.UpdateTarget, len(ts))
		for i, t := range ts {
			targets[i] = core.UpdateTarget{Key: t.Key, Name: t.Name, Dir: t.Dir, Version: core.CopyVersion(t.Dir, core.GameUpdates(a.updatesDir(t)))}
			byKey[t.Key] = t
		}
		for _, dir := range dirs {
			found = append(found, core.MatchDownloads(dir, targets, time.Now().AddDate(0, 0, -30))...)
		}
	}
	out := []core.DownloadMatch{}
	var tell []core.DownloadMatch
	procs := map[string]bool{}
	if len(found) > 0 {
		procs, _ = win.ProcessNames()
	}
	err := a.store.UpdateDownloadsSeen(func(d *core.DownloadsSeen) {
		for _, m := range found {
			if d.Ignored[m.Stamp()] {
				continue
			}
			out = append(out, m)
			if t := byKey[m.Key]; m.Newer && !d.Announced[m.Stamp()] && !t.runningIn(procs) {
				d.Announced[m.Stamp()] = true
				tell = append(tell, m)
			}
		}
	})
	if err != nil {
		a.logf("ERROR saving downloads.json: %v", err)
		tell = nil // told next time, not every two minutes
	}
	for _, m := range tell {
		a.logf("An update for %s is in Downloads: %s", m.Game, m.Path)
		a.notify("An update for "+m.Game+" is in your Downloads", m.Name+" is ready to review on the Updates page.")
	}
	dlMu.Lock()
	dlLast = out
	dlMu.Unlock()
	a.emit("downloadsFound", map[string]any{"matches": out})
}

func lastDownload(path string) (core.DownloadMatch, error) {
	dlMu.Lock()
	defer dlMu.Unlock()
	for _, m := range dlLast {
		if m.Path == path {
			return m, nil
		}
	}
	return core.DownloadMatch{}, errors.New("That file isn't in your Downloads any more.")
}

// apiIgnoreDownload stops showing a match, until its file changes.
func (a *App) apiIgnoreDownload(path string) error {
	m, err := lastDownload(path)
	if err != nil {
		return err
	}
	if err := a.store.UpdateDownloadsSeen(func(d *core.DownloadsSeen) { d.Ignored[m.Stamp()] = true }); err != nil {
		return err
	}
	go a.downloadsCheck()
	return nil
}

// apiRecycleDownload moves a matched update (every part of a .rar set) to
// the Recycle Bin, after it's applied.
func (a *App) apiRecycleDownload(path string) error {
	m, err := lastDownload(path)
	if err != nil {
		return err
	}
	if err := win.Recycle(a.hwnd, m.Files()...); errors.Is(err, win.ErrRecycleCancelled) {
		return errors.New("It's still in your Downloads.")
	} else if err != nil {
		return errors.New("It's still in your Downloads: " + err.Error() + ".")
	}
	a.logf("Moved %s to the Recycle Bin", m.Path)
	go a.downloadsCheck()
	return nil
}

// downloadDirs are Windows' Downloads folder and, when the user set one in
// Hydra, Hydra's download folder.
func downloadDirs() []string {
	var out []string
	if d := win.Downloads(); d != "" {
		out = append(out, d)
	}
	if h := hydraDownloads(); h != "" && (len(out) == 0 || !strings.EqualFold(filepath.Clean(h), filepath.Clean(out[0]))) {
		out = append(out, h)
	}
	return out
}

// hydraDownloads reads Hydra's download folder at most every ten minutes.
var hydraDL struct {
	sync.Mutex
	dir string
	at  time.Time
}

func hydraDownloads() string {
	hydraDL.Lock()
	defer hydraDL.Unlock()
	if time.Since(hydraDL.at) < 10*time.Minute {
		return hydraDL.dir
	}
	hydraDL.at = time.Now()
	ad := os.Getenv("APPDATA")
	if ad == "" {
		return ""
	}
	dir, err := core.HydraDownloadsPath(filepath.Join(ad, "hydralauncher", "hydra-db"), filepath.Join(DataDir(), "hydra-prefs-tmp"))
	if err != nil {
		dir = ""
	}
	hydraDL.dir = dir
	return dir
}

// apiOpenHydra opens Hydra for a game that came from it, with the game's
// name on the clipboard to paste into Hydra's search: Hydra has no link to
// a game's page (only to run one), so this is as close as it gets.
func (a *App) apiOpenHydra(name string) error {
	ad := os.Getenv("APPDATA")
	if ad == "" || !isDirPath(filepath.Join(ad, "hydralauncher")) {
		return errors.New("Hydra isn't on this PC.")
	}
	if n := strings.TrimSpace(name); n != "" {
		if err := win.SetClipboardText(a.hwnd, n); err != nil {
			a.logf("Couldn't copy %q for Hydra: %v", n, err)
		}
	}
	if err := win.ShellOpen("hydralauncher://"); err != nil {
		a.logf("Couldn't open Hydra: %v", err)
		return errors.New("Couldn't open Hydra. Open it from the Start menu.")
	}
	return nil
}

func isDirPath(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
