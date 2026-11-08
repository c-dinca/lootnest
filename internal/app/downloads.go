//go:build windows

package app

import (
	"errors"
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
// click. Lootnest looks nowhere else and fetches nothing.
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
	if dir := win.Downloads(); !off && dir != "" {
		ts := a.updTargets()
		targets := make([]core.UpdateTarget, len(ts))
		for i, t := range ts {
			targets[i] = core.UpdateTarget{Key: t.Key, Name: t.Name, Dir: t.Dir, Version: core.CopyVersion(t.Dir, core.GameUpdates(a.updatesDir(t)))}
			byKey[t.Key] = t
		}
		found = core.MatchDownloads(dir, targets, time.Now().AddDate(0, 0, -30))
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
