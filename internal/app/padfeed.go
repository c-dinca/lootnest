package app

import (
	"runtime"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// The controller for the page while it can't read it itself. In TV mode
// and Big Picture, someone may be working on the PC's other screens while
// someone else plays on the TV. The window with the keyboard is then on
// the PC, and WebView2 gives the page no gamepad input (and its :focus no
// longer matches), so only the Home button worked, read here by raw input
// and XInput. While Lootnest's window is the front one on its own screen,
// Go reads every pad as one and sends it to the page as "padFeed" events,
// on change only; the page uses them only while it doesn't have the
// keyboard, or sees no pad (25-bigpicture.js, padFeed).

// psSeen is a PlayStation pad's latest report, from raw input.
type psSeen struct {
	st core.PadState
	at time.Time
}

var (
	psPadsMu sync.Mutex
	psPads   = map[uintptr]psSeen{}
)

// psStale is how long a pad that sends nothing more counts: a streaming
// host's virtual pad reports only when something changes, a real one all
// the time, so a pad unplugged mid-press doesn't hold its buttons forever.
const psStale = 5 * time.Second

// notePSPad keeps a PlayStation pad's report from WM_INPUT (app.go).
func notePSPad(r plat.PSReport) {
	dualsense, ok := core.PSPad(r.Product)
	if !ok {
		return
	}
	st, ok := core.PSToXbox(dualsense, r.Bluetooth, r.Data)
	if !ok {
		return
	}
	psPadsMu.Lock()
	psPads[r.Device] = psSeen{st, time.Now()}
	psPadsMu.Unlock()
}

// padsNow is every controller as one, and whether a PlayStation pad is
// among them (for the page's button signs).
func padsNow() (st core.PadState, ps, any bool) {
	var all []core.PadState
	if x, ok := plat.XInputRead(); ok {
		all = append(all, core.PadState{Buttons: x.Buttons &^ 0x0400, LX: x.LX, LY: x.LY, RX: x.RX, RY: x.RY}) // 0x0400 is Guide: Lootnest's Home, read on its own
		any = true
	}
	psPadsMu.Lock()
	for dev, p := range psPads {
		if time.Since(p.at) > psStale {
			delete(psPads, dev)
			continue
		}
		all = append(all, p.st)
		ps, any = true, true
	}
	psPadsMu.Unlock()
	return core.MergePads(all...), ps, any
}

// padOwned says whether the controller is Lootnest's: its window shows and
// is the front one on its own screen, so no game or app covers it there,
// whichever window has the keyboard.
func (a *App) padOwned() bool {
	if a.hwnd == 0 || !plat.IsVisible(a.hwnd) {
		return false
	}
	if runtime.GOOS != "windows" {
		// GTK can't see the stacking order on every desktop: the pads are
		// Lootnest's while its window has the keyboard.
		return plat.Foreground() == a.hwnd
	}
	screen, _, _ := plat.ScreenOf(a.hwnd)
	for _, w := range plat.TopWindows() {
		if w.Screen == screen {
			return w.HWND == a.hwnd
		}
	}
	return false
}

// padFeed runs while TV mode or Big Picture is on (startGuide).
func (a *App) padFeed() {
	type sent struct {
		b      uint16
		x, y   float64
		ps, on bool
	}
	var (
		last   sent
		own    bool
		looked time.Time
	)
	for {
		time.Sleep(16 * time.Millisecond)
		a.mu.Lock()
		on := a.tv || a.bigPicture
		if !on {
			a.padFeedOn = false
		}
		a.mu.Unlock()
		if !on {
			if last.on {
				a.emit("padFeed", map[string]any{"on": false})
			}
			return
		}
		if time.Since(looked) >= 250*time.Millisecond {
			looked = time.Now()
			if o := a.padOwned(); o != own {
				own = o
				a.logf("Controller for Lootnest's page from Go: %v", own)
			}
		}
		var now sent
		if own {
			if st, ps, any := padsNow(); any {
				now = sent{b: st.Buttons, x: core.PadAxis(st.LX, false), y: core.PadAxis(st.LY, true), ps: ps, on: true}
			}
		}
		if now == last {
			continue
		}
		last = now
		if !now.on {
			a.emit("padFeed", map[string]any{"on": false})
			continue
		}
		a.emit("padFeed", map[string]any{"on": true, "b": now.b, "x": now.x, "y": now.y, "ps": now.ps})
	}
}
