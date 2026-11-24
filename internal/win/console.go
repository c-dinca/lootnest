//go:build windows

package win

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	procIsWindow              = user32.NewProc("IsWindow")
	procIsHungAppWindow       = user32.NewProc("IsHungAppWindow")
	procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")
)

// TopWindow is a top-level window someone could be looking at: visible, not
// a tool window, not cloaked. Screen is the display it's on.
type TopWindow struct {
	HWND   uintptr
	PID    uint32
	W, H   int32
	Screen uintptr
}

// One callback for the whole program, like findCB; topMu guards its list.
var (
	topMu   sync.Mutex
	topList []TopWindow
	topCB   = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if vis, _, _ := procIsWindowVisible.Call(hwnd); vis == 0 {
			return 1
		}
		if ex, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle); ex&wsExToolWindow != 0 {
			return 1
		}
		// A cloaked window is one Windows keeps but doesn't draw (a
		// suspended Store app, a window on another virtual desktop).
		var cloaked uint32
		procDwmGetWindowAttribute.Call(hwnd, 14 /*DWMWA_CLOAKED*/, uintptr(unsafe.Pointer(&cloaked)), 4)
		if cloaked != 0 {
			return 1
		}
		var r rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		mon, _, _ := procMonitorFromWindow.Call(hwnd, 2) // MONITOR_DEFAULTTONEAREST
		topList = append(topList, TopWindow{HWND: hwnd, PID: pid, W: r.right - r.left, H: r.bottom - r.top, Screen: mon})
		return 1
	})
)

// TopWindows lists the windows on show, the front one first.
func TopWindows() []TopWindow {
	topMu.Lock()
	defer topMu.Unlock()
	topList = nil
	procEnumWindows.Call(topCB, 0)
	l := topList
	topList = nil
	return l
}

// ScreenOf is the display a window is on, and that display's size.
func ScreenOf(hwnd uintptr) (screen uintptr, w, h int32) {
	mon, _, _ := procMonitorFromWindow.Call(hwnd, 2) // MONITOR_DEFAULTTONEAREST
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	return mon, mi.rcMonitor.right - mi.rcMonitor.left, mi.rcMonitor.bottom - mi.rcMonitor.top
}

// PlaceBehind puts hwnd right behind front in the stack of windows, without
// giving it the keyboard: whatever front doesn't cover shows hwnd, not the
// desktop or another program.
func PlaceBehind(hwnd, front uintptr) {
	const (
		swpNoSize   = 0x0001
		swpNoMove   = 0x0002
		swpNoActive = 0x0010
		swpNoOwnerZ = 0x0200
	)
	if hwnd == 0 || front == 0 || hwnd == front {
		return
	}
	procSetWindowPos.Call(hwnd, front, 0, 0, 0, 0, swpNoSize|swpNoMove|swpNoActive|swpNoOwnerZ)
}

// IsWindow says whether a window still exists.
func IsWindow(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	ok, _, _ := procIsWindow.Call(hwnd)
	return ok != 0
}

// WindowHung says whether Windows takes a window for not responding (its
// thread hasn't looked at its messages for about five seconds), as the
// "(Not Responding)" in its title does. A modal loop, such as a menu or a
// window being dragged, still answers.
func WindowHung(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	r, _, _ := procIsHungAppWindow.Call(hwnd)
	return r != 0
}

// Foreground is the window that has the keyboard.
func Foreground() uintptr {
	fg, _, _ := procGetForegroundWindow.Call()
	return fg
}
