//go:build windows

package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// In-app updates. Lootnest asks GitHub for its latest release only when the
// user presses "Check now" or turned on "Check for updates automatically";
// "Update now" downloads the release's installer, checks it, runs it with
// --update (no window, same options) and quits.
// ---------------------------------------------------------------------------

var (
	updateMu    sync.Mutex
	updateFound core.Release // newest release found, when newer than this one
	updateBusy  bool
	lastCheck   time.Time
)

func updatesDir() string { return filepath.Join(DataDir(), "updates") }

// checkUpdate asks GitHub for the latest release and reports it to the UI
// with an "update" event. manual is true for "Check now".
func (a *App) checkUpdate(manual bool) {
	updateMu.Lock()
	lastCheck = time.Now()
	updateMu.Unlock()
	ev := map[string]any{"manual": manual, "current": Version}
	rel, err := core.LatestRelease(fetchURL)
	if err != nil {
		a.logf("Couldn't check for updates: %v", err)
		if manual {
			ev["error"] = "Couldn't check for updates. Check your internet connection, then try again."
			a.emit("update", ev)
		}
		return
	}
	if core.NewerVersion(Version, rel.Version) && rel.SetupURL != "" {
		updateMu.Lock()
		updateFound = rel
		updateMu.Unlock()
		a.logf("Lootnest %s is available (this is %s)", rel.Version, Version)
		ev["available"], ev["version"], ev["page"] = true, rel.Version, rel.Page
	} else {
		ev["upToDate"] = true
	}
	a.emit("update", ev)
}

func (a *App) apiCheckUpdate() { go a.checkUpdate(true) }

// watchUpdates checks once a day while "Check for updates automatically" is on.
func (a *App) watchUpdates() {
	_ = os.RemoveAll(updatesDir()) // the installer of the last update, if any
	time.Sleep(20 * time.Second)
	for {
		a.mu.Lock()
		on := a.cfg.Settings.CheckUpdates
		a.mu.Unlock()
		updateMu.Lock()
		due := time.Since(lastCheck) > 24*time.Hour
		updateMu.Unlock()
		if on && due {
			a.checkUpdate(false)
		}
		time.Sleep(time.Hour)
	}
}

// apiInstallUpdate downloads and starts the newer release's installer, then
// quits so the installer can replace Lootnest. Progress and errors arrive as
// "updateStep" events.
func (a *App) apiInstallUpdate() error {
	updateMu.Lock()
	rel, busy := updateFound, updateBusy
	if rel.SetupURL != "" && !busy {
		updateBusy = true
	}
	updateMu.Unlock()
	if rel.SetupURL == "" {
		return errors.New("Check for updates first.")
	}
	if busy {
		return nil
	}
	a.mu.Lock()
	for _, s := range a.sessions {
		if s.Phase == "playing" || s.Phase == "launching" || s.Phase == "saving" {
			name := a.gameName(s.GameID)
			a.mu.Unlock()
			updateMu.Lock()
			updateBusy = false
			updateMu.Unlock()
			return fmt.Errorf("%s is running. Update after it closes, so its progress is saved first.", name)
		}
	}
	a.mu.Unlock()

	go func() {
		fail := func(err error) {
			a.logf("ERROR updating to %s: %v", rel.Version, err)
			updateMu.Lock()
			updateBusy = false
			updateMu.Unlock()
			a.emit("updateStep", map[string]any{"error": "Couldn't update: " + err.Error()})
		}
		a.emit("updateStep", map[string]any{"step": "downloading", "version": rel.Version})
		path := filepath.Join(updatesDir(), fmt.Sprintf("Lootnest-Setup-%s.exe", rel.Version))
		if err := download(rel.SetupURL, path); err != nil {
			fail(err)
			return
		}
		if err := core.VerifyFile(path, rel.Size, rel.SHA256); err != nil {
			_ = os.Remove(path)
			fail(err)
			return
		}
		a.logf("Updating to Lootnest %s", rel.Version)
		a.emit("updateStep", map[string]any{"step": "installing", "version": rel.Version})
		if err := win.StartDetached(path, "--update", filepath.Dir(path)); err != nil {
			fail(err)
			return
		}
		time.Sleep(500 * time.Millisecond)
		a.ui(func() { a.quit(true) })
	}()
	return nil
}

var downloadHTTP = &http.Client{Timeout: 5 * time.Minute}

// download saves url to path through a temporary file.
func download(url, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	resp, err := downloadHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the download failed (%s)", resp.Status)
	}
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, io.LimitReader(resp.Body, 200<<20))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
