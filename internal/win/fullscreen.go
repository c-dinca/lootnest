//go:build windows

package win

import "unsafe"

var (
	procGetWindowLongPtrW  = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW  = user32.NewProc("SetWindowLongPtrW")
	procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
	procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
	procMonitorFromPoint   = user32.NewProc("MonitorFromPoint")
)

type windowPlacement struct {
	length           uint32
	flags            uint32
	showCmd          uint32
	ptMinPosition    struct{ x, y int32 }
	ptMaxPosition    struct{ x, y int32 }
	rcNormalPosition rect
}

var (
	fsOn        bool
	fsStyle     uintptr
	fsPlacement windowPlacement
)

// SetFullscreen fills the screen the window is on with it, without a frame
// (as a console's screen), or puts it back as it was. Call it on the
// window's thread.
func SetFullscreen(hwnd uintptr, on bool) {
	if on == fsOn {
		return
	}
	if on {
		mon, _, _ := procMonitorFromWindow.Call(hwnd, 2) // MONITOR_DEFAULTTONEAREST
		fillMonitor(hwnd, mon)
		return
	}
	const (
		swpNoZOrder          = 0x0004
		swpNoOwnerZOrder     = 0x0200
		swpFrameChanged      = 0x0020
		swpNoMove, swpNoSize = 0x0002, 0x0001
	)
	procSetWindowLongPtrW.Call(hwnd, gwlStyle, fsStyle)
	procSetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&fsPlacement)))
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoOwnerZOrder|swpFrameChanged)
	fsOn = false
}

// FullscreenPrimary fills the primary display with the window, without a
// frame, wherever it was: TV mode, as a streaming host makes its virtual
// display the primary one. Called again (the displays changed), it fits
// the window to the primary display as it is now. SetFullscreen(false)
// puts it back.
func FullscreenPrimary(hwnd uintptr) {
	mon, _, _ := procMonitorFromPoint.Call(0, 1) // (0,0), MONITOR_DEFAULTTOPRIMARY
	fillMonitor(hwnd, mon)
}

// FullscreenLike fills the monitor another window is on (the primary one
// when that window is gone), borderless.
func FullscreenLike(hwnd, other uintptr) {
	if other == 0 || !IsWindow(other) {
		FullscreenPrimary(hwnd)
		return
	}
	mon, _, _ := procMonitorFromWindow.Call(other, 2) // MONITOR_DEFAULTTONEAREST
	fillMonitor(hwnd, mon)
}

const gwlStyle = ^uintptr(15) // GWL_STYLE (-16)

func fillMonitor(hwnd, mon uintptr) {
	const (
		wsOverlappedWindow = 0x00CF0000
		swpNoOwnerZOrder   = 0x0200
		swpFrameChanged    = 0x0020
	)
	if !fsOn {
		fsStyle, _, _ = procGetWindowLongPtrW.Call(hwnd, gwlStyle)
		fsPlacement = windowPlacement{}
		fsPlacement.length = uint32(unsafe.Sizeof(fsPlacement))
		procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&fsPlacement)))
	}
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	procSetWindowLongPtrW.Call(hwnd, gwlStyle, fsStyle&^wsOverlappedWindow)
	r := mi.rcMonitor
	procSetWindowPos.Call(hwnd, 0, uintptr(r.left), uintptr(r.top), uintptr(r.right-r.left), uintptr(r.bottom-r.top), swpNoOwnerZOrder|swpFrameChanged)
	fsOn = true
}
