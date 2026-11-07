//go:build windows

package win

import "unsafe"

var (
	procGetWindowLongPtrW  = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW  = user32.NewProc("SetWindowLongPtrW")
	procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
	procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
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
	const (
		gwlStyle             = ^uintptr(15) // GWL_STYLE (-16)
		wsOverlappedWindow   = 0x00CF0000
		swpNoZOrder          = 0x0004
		swpNoOwnerZOrder     = 0x0200
		swpFrameChanged      = 0x0020
		swpNoMove, swpNoSize = 0x0002, 0x0001
	)
	if on == fsOn {
		return
	}
	if on {
		fsStyle, _, _ = procGetWindowLongPtrW.Call(hwnd, gwlStyle)
		fsPlacement = windowPlacement{}
		fsPlacement.length = uint32(unsafe.Sizeof(fsPlacement))
		procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&fsPlacement)))
		mon, _, _ := procMonitorFromWindow.Call(hwnd, 2)
		mi := monitorInfo{}
		mi.cbSize = uint32(unsafe.Sizeof(mi))
		procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		procSetWindowLongPtrW.Call(hwnd, gwlStyle, fsStyle&^wsOverlappedWindow)
		r := mi.rcMonitor
		procSetWindowPos.Call(hwnd, 0, uintptr(r.left), uintptr(r.top), uintptr(r.right-r.left), uintptr(r.bottom-r.top), swpNoOwnerZOrder|swpFrameChanged)
	} else {
		procSetWindowLongPtrW.Call(hwnd, gwlStyle, fsStyle)
		procSetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&fsPlacement)))
		procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoOwnerZOrder|swpFrameChanged)
	}
	fsOn = on
}
