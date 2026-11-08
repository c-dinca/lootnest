//go:build windows

package app

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Screenshots (core/screenshots.go): Ctrl+Shift+F12 while a game runs saves
// a picture of its window to Pictures\Lootnest\Screenshots\<game>, and a
// save point made while its game runs keeps one beside it. The picture is
// taken from outside the game (win/capture.go), never in its folder.
// ---------------------------------------------------------------------------

const hotkeyShot = 0x4C53 // "LS"

func picturesDir() string {
	if d := win.Pictures(); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Pictures")
}

// errBlack is a picture that came out black: a game in exclusive fullscreen.
var errBlack = errors.New("the picture came out black")

// captureGame takes a picture of the window of the game whose process is
// proc. A window that can't draw itself is copied from the screen when it's
// in front.
func captureGame(proc string) (*image.RGBA, error) {
	hwnd := win.GameWindow(proc)
	if hwnd == 0 {
		return nil, errors.New("Lootnest couldn't find the game's window")
	}
	img, err := win.CaptureWindow(hwnd)
	if err != nil || core.IsNearBlack(img) {
		if s, serr := win.CaptureScreen(hwnd); serr == nil {
			img, err = s, nil
		}
	}
	if err != nil {
		return nil, err
	}
	if core.IsNearBlack(img) {
		return nil, errBlack
	}
	return img, nil
}

// libraryName is a Library game's name, or "" if it's gone.
func (a *App) libraryName(key string) string {
	if id, ok := strings.CutPrefix(key, "game:"); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		if g := a.cfg.Game(id); g != nil {
			return g.Name
		}
		return ""
	}
	f, _ := a.foundByKey(key)
	return f.Name
}

// shotThumb is a screenshot's thumbnail as a data: URL, made the first time
// it's asked for.
func (a *App) shotThumb(key, dir, name string) string {
	dst := filepath.Join(a.store.ShotThumbsDir(key), name)
	if u := dataURL(dst); u != "" {
		return u
	}
	if err := core.MakeThumb(filepath.Join(dir, name), dst); err != nil {
		return ""
	}
	return dataURL(dst)
}

// apiPicture is one picture at full size, for the window's lightbox: a save
// point's ("point:<id>") or a screenshot's ("shot:<file name>").
func (a *App) apiPicture(key, ref string) (string, error) {
	var p string
	if id, ok := strings.CutPrefix(ref, "point:"); ok {
		var err error
		if p, err = core.SavePointFile(a.store.SavePointsDir(key), id, core.PointPicture); err != nil {
			return "", err
		}
	} else if name, ok := strings.CutPrefix(ref, "shot:"); ok && name == filepath.Base(name) && !strings.HasPrefix(name, ".") && strings.EqualFold(filepath.Ext(name), ".png") {
		game := a.libraryName(key)
		if game == "" {
			return "", errFoundGone
		}
		p = filepath.Join(core.ScreenshotsDir(picturesDir(), game), name)
	} else {
		return "", errors.New("unknown picture")
	}
	if u := dataURLMax(p, 64<<20); u != "" {
		return u, nil
	}
	return "", errors.New("That picture is gone, or too big to show.")
}

// quickShot is the hotkey: a picture of the game running now.
func (a *App) quickShot() {
	r, color, ok := a.runningNow()
	if !ok {
		return
	}
	go func() {
		img, err := captureGame(r.proc)
		if errors.Is(err, errBlack) {
			win.ShowToast(win.Toast{Kicker: "No screenshot", Title: "The picture came out black", Line: "Switch the game to borderless windowed.", Accent: color})
			a.logf("Screenshot of %s came out black (exclusive fullscreen?)", r.name)
			return
		}
		var p string
		if err == nil {
			p, err = core.SaveNewScreenshot(img, picturesDir(), r.name, time.Now())
		}
		if err != nil {
			win.ShowToast(win.Toast{Kicker: "No screenshot", Title: r.name, Line: sentence(err), Accent: color})
			a.logf("Couldn't take a screenshot of %s: %v", r.name, err)
			return
		}
		if err := core.SaveScreenshotPNG(core.Thumbnail(img, core.ThumbWidth), filepath.Join(a.store.ShotThumbsDir(r.key), filepath.Base(p))); err != nil {
			a.logf("Couldn't keep a thumbnail of %s: %v", p, err)
		}
		win.ShowToast(win.Toast{Kicker: "Screenshot saved", Title: filepath.Base(p), Line: r.name, Accent: color})
		a.logf("Screenshot of %s: %s", r.name, p)
		a.apiGameTimeline(r.key)
	}()
}
