package plat

import (
	"sync/atomic"

	"lootnest/internal/desk"
)

// Lootnest's window on Linux. The app passes its window handle (a.hwnd) to
// plat as it does on Windows; here that handle is MainWindow, given by
// SetMainWindow, and the functions below reach the GTK window through it.
// What GTK can't tell or do on every desktop (the stacking order, another
// program's window, which window has the keyboard on Wayland) stays in the
// stubs.

// Window is what plat needs of the window (internal/webkit's WebView).
// Its state is read from any goroutine; what changes it runs on the GTK
// thread (desk.Later).
type Window interface {
	Show()
	Present()
	Hide()
	SetFullscreen(on bool)
	FullscreenPrimary()
	Visible() bool
	Active() bool
	Maximized() bool
}

// MainWindow is the handle of Lootnest's window on Linux.
const MainWindow uintptr = 1

var mainWin atomic.Pointer[Window]

// SetMainWindow makes w the window MainWindow names, and returns it.
func SetMainWindow(w Window) uintptr {
	mainWin.Store(&w)
	return MainWindow
}

func windowOf(hwnd uintptr) Window {
	if hwnd != MainWindow {
		return nil
	}
	if p := mainWin.Load(); p != nil {
		return *p
	}
	return nil
}

func onWindow(hwnd uintptr, f func(Window)) {
	if w := windowOf(hwnd); w != nil {
		desk.Later(func() { f(w) })
	}
}

// Present shows the window in front.
func Present(hwnd uintptr) { onWindow(hwnd, Window.Present) }

// ForceForeground and SetForeground bring the window to the front, as far
// as the desktop lets a program do so (Wayland may only mark it).
func ForceForeground(hwnd uintptr) { onWindow(hwnd, Window.Present) }
func SetForeground(hwnd uintptr)   { onWindow(hwnd, Window.Present) }

// ShowWindow hides the window for SW_HIDE and shows it for anything else.
func ShowWindow(hwnd uintptr, cmd int) {
	onWindow(hwnd, func(w Window) {
		if cmd == SW_HIDE {
			w.Hide()
		} else {
			w.Show()
		}
	})
}

// IsVisible tells whether the window is shown.
func IsVisible(hwnd uintptr) bool {
	w := windowOf(hwnd)
	return w != nil && w.Visible()
}

// IsZoomed tells whether the window is maximized.
func IsZoomed(hwnd uintptr) bool {
	w := windowOf(hwnd)
	return w != nil && w.Maximized()
}

// SetFullscreen fills the window's monitor with it, or puts it back.
func SetFullscreen(hwnd uintptr, on bool) {
	onWindow(hwnd, func(w Window) { w.SetFullscreen(on) })
}

// FullscreenPrimary fills the primary monitor with the window.
func FullscreenPrimary(hwnd uintptr) { onWindow(hwnd, Window.FullscreenPrimary) }

// Foreground is Lootnest's window while it has the keyboard, else 0: GTK
// can't name another program's window.
func Foreground() uintptr {
	if w := windowOf(MainWindow); w != nil && w.Active() {
		return MainWindow
	}
	return 0
}
