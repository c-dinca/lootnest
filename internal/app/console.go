package app

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/jpeg"
	"os"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// Lootnest as a console's screen: in TV mode and in Big Picture, a game
// starts over Lootnest and Lootnest stays right behind it, so the desktop
// and other programs never show; the controller's Home button brings a
// menu over the game (go back to it, go to Lootnest, close it); and when
// the game closes, Lootnest is what's there.

// console says whether Lootnest is that screen now.
func (a *App) console() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tv || a.bigPicture
}

// conKey is a game's key as the tick knows it: a profile game's id, or a
// Library game's key.
func conKey(key string) string { return strings.TrimPrefix(key, "game:") }

// tvGameStarting runs right after a game was asked to start. In TV mode and
// Big Picture, the window shows the game starting and stays in front until
// the game's own window is up (curtain), rather than hiding.
func (a *App) tvGameStarting(key, name string) {
	if !a.console() {
		return
	}
	key = conKey(key)
	seen := map[uint64]bool{}
	for _, w := range plat.TopWindows() {
		seen[uint64(w.HWND)] = true
	}
	a.mu.Lock()
	a.tvPlaying = name
	a.conGen++
	gen := a.conGen
	a.mu.Unlock()
	plat.AllowAnyForeground()
	a.emit("tvGame", map[string]any{"phase": "starting", "key": key, "name": name})
	go a.curtain(key, name, gen, seen)
}

// gameWindowOf is the window of a game Lootnest follows by its process, or 0.
func (a *App) gameWindowOf(key string) uintptr {
	if pid := a.pidOf(key); pid != 0 {
		return plat.WindowOfPID(pid)
	}
	if proc := a.procOf(key); proc != "" {
		return plat.GameWindow(proc)
	}
	return 0
}

// front puts a game's window in front with the keyboard, and Lootnest
// right behind it. Call it on the window's thread.
func (a *App) front(gw uintptr) {
	a.mu.Lock()
	a.conWin = gw
	a.mu.Unlock()
	plat.ForceForeground(gw)
	plat.PlaceBehind(a.hwnd, gw)
}

// curtain follows a game that was just started until its window is up: a
// window opened by a program that isn't in front opens behind Lootnest, so
// Lootnest looks for it (by the game's process when it knows it, else the
// first large new window on its own display), brings it to the front and
// stays right behind it. A launcher's window may come first and the game's
// after it, so it keeps looking for half a minute. A game it can't follow
// by its process is watched by its window: when that's gone, Lootnest
// comes back.
func (a *App) curtain(key, name string, gen int, seen map[uint64]bool) {
	self := uint32(os.Getpid())
	start := time.Now()
	var last uintptr
	var found time.Time
	mine := func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return (a.tv || a.bigPicture) && a.conGen == gen
	}
	for {
		time.Sleep(300 * time.Millisecond)
		if !mine() {
			return
		}
		gw := a.gameWindowOf(key)
		if gw == 0 {
			screen, sw, sh := plat.ScreenOf(a.hwnd)
			var now []core.TopWindow
			for _, w := range plat.TopWindows() {
				if w.Screen == screen && w.HWND != a.hwnd {
					now = append(now, core.TopWindow{ID: uint64(w.HWND), PID: w.PID, W: int(w.W), H: int(w.H)})
				}
			}
			gw = uintptr(core.NewGameWindow(seen, now, self, int(sw), int(sh)))
		}
		if gw != 0 && gw != last {
			if last == 0 {
				a.logf("%s: its window is up after %s; Lootnest stays behind it", name, time.Since(start).Round(100*time.Millisecond))
			}
			last, found = gw, time.Now()
			a.ui(func() {
				if mine() {
					a.front(gw)
				}
			})
		}
		switch {
		case last == 0 && time.Since(start) > 25*time.Second:
			// Nothing Lootnest could recognise: step behind every window,
			// as before, so a game that opened behind it shows.
			a.logf("%s: no window found in 25 s; Lootnest steps behind the other windows", name)
			a.ui(func() {
				if mine() {
					plat.SendToBack(a.hwnd)
				}
			})
			return
		case last != 0 && time.Since(found) > 30*time.Second:
			a.watchGone(key, name, gen, last)
			return
		}
	}
}

// watchGone brings Lootnest back when the window of a game it can't follow
// by its process is gone. Games it follows come back from the tick.
func (a *App) watchGone(key, name string, gen int, gw uintptr) {
	if a.procOf(key) != "" || a.pidOf(key) != 0 {
		return
	}
	a.mu.Lock()
	if a.conGame == "" {
		a.conGame = key
	}
	a.mu.Unlock()
	for plat.IsWindow(gw) {
		time.Sleep(time.Second)
		a.mu.Lock()
		on := (a.tv || a.bigPicture) && a.conGen == gen
		a.mu.Unlock()
		if !on {
			return
		}
	}
	a.logf("%s: its window closed; back to Lootnest", name)
	a.ui(func() {
		a.mu.Lock()
		on := (a.tv || a.bigPicture) && a.conGen == gen
		if on {
			a.hidFor, a.conGame, a.conWin, a.tvMenuOn, a.tvPlaying = "", "", 0, false, ""
		}
		a.mu.Unlock()
		if on {
			a.tvBack()
			a.emit("bpReturn", nil)
		}
	})
}

// tvGameUp brings the game Lootnest stands behind to the front: once its
// process runs, and again from the Home menu's Resume. Its window may take
// a while after its process. Call it on the window's thread.
func (a *App) tvGameUp(key string) {
	a.emit("tvGame", map[string]any{"phase": "playing", "key": key})
	a.mu.Lock()
	known, gen := a.conWin, a.conGen
	a.mu.Unlock()
	if gw := a.gameWindowOf(key); gw != 0 {
		a.front(gw)
		return
	}
	if plat.IsWindow(known) {
		a.front(known)
		return
	}
	if a.procOf(key) == "" && a.pidOf(key) == 0 {
		return
	}
	go func() {
		for i := 0; i < 20; i++ {
			time.Sleep(time.Second)
			if gw := a.gameWindowOf(key); gw != 0 {
				a.ui(func() {
					a.mu.Lock()
					on := (a.tv || a.bigPicture) && a.conGen == gen
					a.mu.Unlock()
					if on {
						a.front(gw)
					}
				})
				return
			}
		}
	}()
}

// startGuide watches the Home button while Lootnest is a console's screen.
func (a *App) startGuide() {
	a.mu.Lock()
	if a.guideOn {
		a.mu.Unlock()
		return
	}
	a.guideOn = true
	a.mu.Unlock()
	go a.watchGuide()
}

// watchGuide follows XInput pads' Guide button in TV mode and Big Picture:
// an Xbox pad's, and the TV's controller, which reaches the PC as one
// through the streaming host. The page can't see it while a game is in
// front.
func (a *App) watchGuide() {
	was := false
	for {
		time.Sleep(60 * time.Millisecond)
		a.mu.Lock()
		on := a.tv || a.bigPicture
		if !on {
			a.guideOn = false
		}
		a.mu.Unlock()
		if !on {
			return
		}
		down := plat.XInputGuideDown()
		if down && !was {
			a.ui(a.tvHome)
		}
		was = down
	}
}

// gameShot is a picture of the game in front, for the Home menu to lie
// over, as a JPEG data URL; "" when there's none (exclusive fullscreen
// draws black). Take it before Lootnest comes to the front.
func (a *App) gameShot() string {
	a.mu.Lock()
	gw := a.conWin
	a.mu.Unlock()
	if !plat.IsWindow(gw) {
		return ""
	}
	img, err := plat.CaptureWindow(gw)
	if err != nil || core.IsNearBlack(img) {
		if s, serr := plat.CaptureScreen(gw); serr == nil {
			img, err = s, nil
		}
	}
	if err != nil || img == nil || core.IsNearBlack(img) {
		return ""
	}
	var b bytes.Buffer
	if jpeg.Encode(&b, core.Thumbnail(img, 1280), &jpeg.Options{Quality: 72}) != nil {
		return ""
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

// raise brings Lootnest to the front over a running game without
// forgetting the game, as show would.
func (a *App) raise() {
	a.mu.Lock()
	tv := a.tv
	a.mu.Unlock()
	plat.Present(a.hwnd)
	if tv {
		plat.FullscreenPrimary(a.hwnd)
	}
	plat.ForceForeground(a.hwnd)
}

// tvHome is the controller's Home button in TV mode and Big Picture. Over
// a game it opens the menu, as a console's does, with the game's picture
// behind it, and closes it again; with no game it opens Big Picture's own
// menu. Call it on the window's thread.
func (a *App) tvHome() {
	a.mu.Lock()
	con, playing, on := a.tv || a.bigPicture, a.conGame, a.tvMenuOn
	a.mu.Unlock()
	if !con {
		return
	}
	switch {
	case playing == "":
		a.emit("tvHome", map[string]any{})
	case on:
		a.tvResume()
	default:
		shot := ""
		if plat.Foreground() != a.hwnd {
			shot = a.gameShot()
		}
		a.mu.Lock()
		a.tvMenuOn = true
		a.mu.Unlock()
		a.raise()
		a.mu.Lock()
		pointer := a.pointerFor == playing
		a.mu.Unlock()
		a.emit("tvMenu", map[string]any{"on": true, "key": playing, "shot": shot, "pointer": pointer})
	}
}

// tvResume closes the Home menu and brings the game back to the front.
func (a *App) tvResume() {
	a.mu.Lock()
	playing := a.conGame
	a.tvMenuOn = false
	a.mu.Unlock()
	a.emit("tvMenu", map[string]any{"on": false})
	if playing != "" {
		a.tvGameUp(playing)
	}
}

// apiTVResume is Resume in the Home menu, and Play on the game running.
func (a *App) apiTVResume() { a.ui(a.tvResume) }

// apiTVStay is Lootnest in the Home menu: the menu closes, Lootnest stays
// in front, and the game keeps running behind it.
func (a *App) apiTVStay() {
	a.mu.Lock()
	a.tvMenuOn = false
	a.mu.Unlock()
}

// apiTVCloseGame asks the game running over Lootnest to quit, as closing
// its window would; Lootnest comes back when it's gone.
func (a *App) apiTVCloseGame() error {
	a.mu.Lock()
	playing, gw := a.conGame, a.conWin
	a.tvMenuOn = false
	a.mu.Unlock()
	if playing == "" {
		return nil
	}
	proc, pid := a.procOf(playing), a.pidOf(playing)
	closed := 0
	switch {
	case pid != 0:
		closed = plat.CloseWindowsOfPID(pid)
	case proc != "":
		closed = plat.CloseProcessWindows(proc)
	case plat.IsWindow(gw):
		plat.PostMessage(gw, plat.WM_CLOSE, 0, 0)
		closed = 1
	}
	if closed == 0 {
		return errors.New("Lootnest couldn't close the game. Quit it from its own menu")
	}
	a.logf("Asked %s to close from the Home menu", playing)
	return nil
}
