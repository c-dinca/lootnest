//go:build windows

package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Captures from other apps (core/captures.go): the screenshots and clips
// Game Bar, Steam, NVIDIA, AMD and OBS made are indexed when the Library is
// scanned and when a session ends, at most once every 10 minutes, and shown
// on each game's Timeline. They're only read: assigning one to a game by
// hand writes the index, nothing else.
// ---------------------------------------------------------------------------

const (
	captureEvery   = 10 * time.Minute
	capturesShown  = 200
	capturePicMax  = 64 << 20
	captureDecoded = 100_000_000 // pixels a picture may have to be shrunk here
)

var (
	capMu      sync.Mutex
	capLast    time.Time
	capPending bool
)

// scanCapturesSoon indexes the captures now, or once 10 minutes have gone
// by since the last scan. Never in safe mode, or with captures off.
func (a *App) scanCapturesSoon() {
	a.mu.Lock()
	off := a.cfg.Settings.CapturesOff
	a.mu.Unlock()
	if a.safe || off {
		return
	}
	capMu.Lock()
	defer capMu.Unlock()
	if capPending {
		return
	}
	capPending = true
	time.AfterFunc(max(0, time.Until(capLast.Add(captureEvery))), func() {
		capMu.Lock()
		capPending = false
		capMu.Unlock()
		a.scanCaptures()
	})
}

// captureGames is every Library game, as captures are matched to them.
func (a *App) captureGames() []core.CaptureGame {
	var out []core.CaptureGame
	a.mu.Lock()
	for _, g := range a.cfg.Games {
		id := 0
		if g.Launch.Type == "steam" {
			id = g.Launch.AppID
		}
		out = append(out, core.CaptureGame{Key: "game:" + g.ID, Name: g.Name, AppID: id})
	}
	a.mu.Unlock()
	for _, f := range a.visibleFound(a.libraryGames()) {
		id := f.AppID
		if id == 0 {
			id = f.SteamID
		}
		out = append(out, core.CaptureGame{Key: f.Key, Name: f.Name, AppID: id})
	}
	return out
}

func (a *App) captureSources(games []core.CaptureGame) []core.CaptureSource {
	names := make([]string, len(games))
	for i, g := range games {
		names[i] = g.Name
	}
	gameBar, _ := win.GameBarCaptures()
	return core.CaptureSources(core.CaptureEnv{Videos: win.Videos(), Pictures: picturesDir(), SteamPath: a.env.SteamPath, GameBar: gameBar, Games: names})
}

func (a *App) scanCaptures() {
	a.mu.Lock()
	off := a.cfg.Settings.CapturesOff
	a.mu.Unlock()
	if off {
		return
	}
	start := time.Now()
	capMu.Lock()
	capLast = start
	capMu.Unlock()
	games := a.captureGames()
	sessions, _, err := a.store.ReadSessions("", time.Time{})
	if err != nil {
		a.logf("Couldn't read the session log: %v", err)
	}
	list, err := a.store.UpdateCaptures(core.ScanCaptures(a.captureSources(games), a.store.LoadCaptures()), games, sessions)
	if err != nil {
		a.logf("ERROR saving the index of captures: %v", err)
		return
	}
	unsorted := 0
	for _, c := range list {
		if c.Key == "" {
			unsorted++
		}
	}
	a.logf("Captures from other apps: %d indexed, %d unsorted, in %s", len(list), unsorted, time.Since(start).Round(100*time.Millisecond))
	a.emit("capturesIndexed", nil)
}

// CaptureItem is a capture as the window shows it.
type CaptureItem struct {
	Path   string    `json:"path"`
	Kind   string    `json:"kind"`
	Source string    `json:"source"`
	At     time.Time `json:"at"`
	Thumb  string    `json:"thumb,omitempty"` // a data: URL; none for a clip Windows has no thumbnail of
}

// sendCaptures emits the newest 200 captures in an event at once, with the
// thumbnails already made, then again once the missing ones are made.
func (a *App) sendCaptures(event string, list []core.Capture, fields map[string]any) {
	list = list[:min(len(list), capturesShown)]
	for _, create := range []bool{false, true} {
		items, missing := []CaptureItem{}, false
		for _, c := range list {
			it := CaptureItem{Path: c.Path, Kind: c.Kind, Source: c.Source, At: c.At, Thumb: a.captureThumb(c, create)}
			missing = missing || it.Thumb == ""
			items = append(items, it)
		}
		fields["items"] = items
		a.emit(event, fields)
		if !missing {
			return
		}
	}
}

// apiGameCaptures answers with a "captures" event: a game's captures from
// other apps, newest first, with how many each app made. Lootnest's own
// screenshots in the game's folder are on the timeline already.
func (a *App) apiGameCaptures(key string) {
	go func() {
		var mine []core.Capture
		counts := map[string]int{}
		own := core.ScreenshotsDir(picturesDir(), a.libraryName(key))
		for _, c := range a.store.LoadCaptures() {
			if c.Key == key && !(c.Source == core.CapLootnest && filepath.Dir(c.Path) == own) {
				counts[c.Source]++
				mine = append(mine, c)
			}
		}
		_, gameBarOff := win.GameBarCaptures()
		a.sendCaptures("captures", mine, map[string]any{"key": key, "counts": counts, "gameBarOff": gameBarOff})
	}()
}

// CaptureFolder is a folder captures are read from, for Settings.
type CaptureFolder struct {
	Source string `json:"source"`
	Dir    string `json:"dir"`
}

// apiUnsortedCaptures answers with an "unsortedCaptures" event: the
// captures matched to no game, and the folders they're read from.
func (a *App) apiUnsortedCaptures() {
	go func() {
		var list []core.Capture
		for _, c := range a.store.LoadCaptures() {
			if c.Key == "" {
				list = append(list, c)
			}
		}
		folders := []CaptureFolder{}
		seen := map[string]bool{}
		for _, s := range a.captureSources(a.captureGames()) {
			dir := s.Dir // one game's folder: the folder that holds them
			switch {
			case s.AppID > 0:
				dir = filepath.Dir(filepath.Dir(dir))
			case s.Game != "":
				dir = filepath.Dir(dir)
			}
			if k := s.Name + "|" + dir; !seen[k] {
				seen[k] = true
				folders = append(folders, CaptureFolder{s.Name, dir})
			}
		}
		a.sendCaptures("unsortedCaptures", list, map[string]any{"total": len(list), "folders": folders})
	}()
}

// indexedCapture is the capture at path, only if it's in the index: the
// window can't name any other file.
func (a *App) indexedCapture(path string) (core.Capture, error) {
	for _, c := range a.store.LoadCaptures() {
		if c.Path == path {
			return c, nil
		}
	}
	return core.Capture{}, errors.New("That capture isn't on the list anymore.")
}

func (a *App) apiAssignCapture(path, key string) error {
	if key != core.CaptureNoGame && a.libraryName(key) == "" {
		return errFoundGone
	}
	if err := a.store.AssignCapture(path, key); err != nil {
		return errors.New("That capture isn't on the list anymore.")
	}
	a.emit("capturesIndexed", nil)
	return nil
}

// apiOpenCapture opens a capture in the app Windows opens it with.
func (a *App) apiOpenCapture(path string) error {
	c, err := a.indexedCapture(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(c.Path); err != nil {
		return errors.New("That file is gone from its folder.")
	}
	return win.ShellOpen(c.Path)
}

// capturePicture is a capture at full size, for the lightbox.
func (a *App) capturePicture(path string) (string, error) {
	c, err := a.indexedCapture(path)
	if err != nil {
		return "", err
	}
	if c.Kind == "picture" {
		if u := dataURLMax(c.Path, capturePicMax); u != "" {
			return u, nil
		}
	}
	return "", errors.New("That picture is gone, or too big to show.")
}

// captureThumb is a capture's thumbnail as a data: URL, kept in the data
// folder by the file's path, size and time, and made there if create is set.
func (a *App) captureThumb(c core.Capture, create bool) string {
	h := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%d", strings.ToLower(c.Path), c.Size, c.At.UnixNano()))
	dst := filepath.Join(a.store.Dir, "capture-thumbs", hex.EncodeToString(h[:12])+".png")
	if u := dataURL(dst); u != "" || !create {
		return u
	}
	img, err := decodeSmall(c)
	if err != nil {
		if img, err = win.ShellThumbnail(c.Path, core.ThumbWidth); err != nil {
			return ""
		}
	}
	if err := core.SaveScreenshotPNG(core.Thumbnail(img, core.ThumbWidth), dst); err != nil {
		a.logf("Couldn't keep a thumbnail of %s: %v", c.Path, err)
		return ""
	}
	return dataURL(dst)
}

// decodeSmall reads a PNG or JPEG capture that isn't too big to shrink here.
func decodeSmall(c core.Capture) (image.Image, error) {
	if c.Kind != "picture" || c.Size > capturePicMax {
		return nil, errors.New("not a picture to read here")
	}
	f, err := os.Open(c.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if cfg, _, err := image.DecodeConfig(f); err != nil || cfg.Width*cfg.Height > captureDecoded {
		return nil, errors.New("not a picture to read here")
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	return img, err
}
