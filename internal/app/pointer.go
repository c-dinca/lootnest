//go:build windows

package app

import (
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// The controller as a mouse, in Big Picture and TV mode: for a program in
// front of Lootnest that only knows a mouse (a store's launcher, Spotify's
// or Stremio's own window, a web page). It's for the one program it was
// turned on for, from the Home menu or by Lootnest for an app that needs
// it, works only while that program is in front (never over Lootnest or
// its Home menu), and ends when the program closes.
//
// The left stick moves the pointer, A is the left button (held, it
// drags), X the right one, the right stick the wheel, the D-pad the
// arrows, B is Escape, and Y opens or closes Windows' on-screen keyboard.
// It reads XInput, so a PlayStation pad counts when it reaches the PC as
// an Xbox one (from the TV, or with "as an Xbox one" on).

// wmPointer tells a site's window (webapp.go) that the controller is a
// mouse now (wParam 1) or its own again (0), so it doesn't also press keys.
const wmPointer = 0x8000 + 7 // WM_APP + 7

// pointerSet turns the controller into a mouse for the program running
// over Lootnest under this key, or back; "" is whatever runs now.
func (a *App) pointerSet(key string, on bool) {
	a.mu.Lock()
	if key == "" {
		key = a.conGame
	}
	was := a.pointerFor
	if on {
		a.pointerFor = key
	} else {
		a.pointerFor = ""
	}
	start := on && key != "" && !a.pointerRun
	if start {
		a.pointerRun = true
	}
	a.mu.Unlock()
	if key == "" {
		return
	}
	if on != (was == key) {
		a.logf("The controller is %s for %s", map[bool]string{true: "a mouse", false: "its own again"}[on], key)
	}
	a.pointerTell(key, on)
	if start {
		go a.pointerLoop(key)
	}
}

// pointerTell lets one of Lootnest's own site windows know.
func (a *App) pointerTell(key string, on bool) {
	if pid := a.pidOf(key); pid != 0 {
		if h := win.WindowOfPID(pid); h != 0 {
			var wp uintptr
			if on {
				wp = 1
			}
			win.PostMessage(h, wmPointer, wp, 0)
		}
	}
}

// apiTVPointer is "Use the controller as a mouse" in the Home menu.
func (a *App) apiTVPointer(on bool) { a.pointerSet("", on) }

// pointerLoop moves the mouse from the controller until it's turned off
// or the program it's for has closed.
func (a *App) pointerLoop(key string) {
	defer func() {
		a.mu.Lock()
		a.pointerRun = false
		a.mu.Unlock()
	}()
	const frame = 8 * time.Millisecond
	var (
		p       core.Pointer
		prev    uint16 = 0xFFFF // nothing held when it starts counts as newly pressed
		held    bool            // the left button is down
		seen    bool            // the program was in front once
		said    bool
		gone    time.Time
		dirAt   time.Time
		dirHeld uint16
		last    = time.Now()
		arrows  = [4][2]uint16{{win.XUp, 0x26}, {win.XDown, 0x28}, {win.XLeft, 0x25}, {win.XRight, 0x27}}
		letGo   = func() {
			if held {
				win.MouseButton(false, false)
				held = false
			}
		}
		started = time.Now()
	)
	defer letGo()
	for {
		time.Sleep(frame)
		now := time.Now()
		dt := now.Sub(last)
		last = now
		a.mu.Lock()
		on, playing, menu, hwnd := a.pointerFor == key, a.conGame, a.tvMenuOn, a.hwnd
		a.mu.Unlock()
		if !on {
			return
		}
		// The program closed (a while after it was last there), or never came.
		if playing == key {
			seen, gone = true, time.Time{}
		} else if gone.IsZero() {
			gone = now
		}
		if playing != key && ((seen && now.Sub(gone) > 3*time.Second) || (!seen && now.Sub(started) > 2*time.Minute)) {
			a.mu.Lock()
			if a.pointerFor == key {
				a.pointerFor = ""
			}
			a.mu.Unlock()
			return
		}
		pad, ok := win.XInputRead()
		if !ok || playing != key || menu || win.Foreground() == hwnd {
			letGo()
			prev = 0xFFFF
			continue
		}
		if !said {
			said = true
			win.ShowToast(win.Toast{Kicker: "Controller", Title: "The controller is a mouse now",
				Line: "The left stick moves, A clicks, the right stick scrolls. The Home button's menu turns it off."})
		}
		_, _, screenH := win.ScreenOf(win.Foreground())
		dx, dy, wheel := p.Step(core.PadSticks{LX: core.StickUnit(pad.LX), LY: core.StickUnit(pad.LY), RY: core.StickUnit(pad.RY)}, dt, int(screenH))
		win.MouseMove(dx, dy)
		win.MouseWheel(wheel)
		pressed := pad.Buttons &^ prev
		// A is the left button: down when it's newly pressed, up when let go.
		if down := pad.Buttons&win.XA != 0; down != held && (!down || prev&win.XA == 0) {
			win.MouseButton(false, down)
			held = down
		}
		if pressed&win.XX != 0 {
			win.MouseButton(true, true)
			win.MouseButton(true, false)
		}
		if pressed&win.XB != 0 {
			win.TapKey(0x1B) // Escape
		}
		if pressed&win.XY != 0 {
			screenKeyboard()
		}
		// The D-pad is the arrows; held, it repeats.
		dirs := pad.Buttons & (win.XUp | win.XDown | win.XLeft | win.XRight)
		if dirs != 0 && (dirs != dirHeld || now.After(dirAt)) {
			for _, d := range arrows {
				if dirs&d[0] != 0 {
					win.TapKey(d[1])
				}
			}
			if dirs != dirHeld {
				dirAt = now.Add(400 * time.Millisecond)
			} else {
				dirAt = now.Add(120 * time.Millisecond)
			}
		}
		dirHeld = dirs
		prev = pad.Buttons
	}
}

// screenKeyboard opens Windows' on-screen keyboard, or closes it.
func screenKeyboard() {
	if procs, err := win.ProcessNames(); err == nil && procs["osk"] {
		win.CloseProcessWindows("osk")
		return
	}
	_ = win.ShellOpen("osk.exe")
}
