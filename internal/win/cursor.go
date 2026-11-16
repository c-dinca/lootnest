//go:build windows

package win

import (
	"sync"
	"syscall"
	"unsafe"
)

// The cursor and the displays, for keeping the PC's mouse off the TV in TV
// mode (core.CursorKeep). Boxes are {left, top, right, bottom} in desktop
// coordinates, right and bottom exclusive.

var (
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procSetCursorPos        = user32.NewProc("SetCursorPos")
	procClipCursor          = user32.NewProc("ClipCursor")
	procGetClipCursor       = user32.NewProc("GetClipCursor")
)

var (
	monitorsMu  sync.Mutex
	monitorsOut [][4]int
	monitorsCB  = syscall.NewCallback(func(mon, _, r, _ uintptr) uintptr {
		b := (*rect)(unsafe.Pointer(r))
		monitorsOut = append(monitorsOut, [4]int{int(b.left), int(b.top), int(b.right), int(b.bottom)})
		return 1
	})
)

// Displays lists every display's area.
func Displays() [][4]int {
	monitorsMu.Lock()
	defer monitorsMu.Unlock()
	monitorsOut = nil
	procEnumDisplayMonitors.Call(0, 0, monitorsCB, 0)
	out := monitorsOut
	monitorsOut = nil
	return out
}

// PrimaryDisplay is the primary display's area: the TV's in TV mode, as
// the streaming host makes its virtual display the primary one.
func PrimaryDisplay() [4]int {
	mon, _, _ := procMonitorFromPoint.Call(0, 1) // (0,0), MONITOR_DEFAULTTOPRIMARY
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	r := mi.rcMonitor
	return [4]int{int(r.left), int(r.top), int(r.right), int(r.bottom)}
}

// CursorPos is where the cursor is.
func CursorPos() (x, y int, ok bool) {
	var pt struct{ x, y int32 }
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return int(pt.x), int(pt.y), r != 0
}

// MoveCursor puts the cursor at a point.
func MoveCursor(x, y int) {
	procSetCursorPos.Call(uintptr(int32(x)), uintptr(int32(y)))
}

// CursorClip is the area the cursor is held in (the whole desktop when
// nothing holds it).
func CursorClip() [4]int {
	var r rect
	procGetClipCursor.Call(uintptr(unsafe.Pointer(&r)))
	return [4]int{int(r.left), int(r.top), int(r.right), int(r.bottom)}
}

// ClipCursor holds the cursor inside an area, or lets it go (nil).
// Windows lets it go by itself at times (another window in front), so a
// caller that wants it held sets it again.
func ClipCursor(b *[4]int) {
	if b == nil {
		procClipCursor.Call(0)
		return
	}
	r := rect{int32(b[0]), int32(b[1]), int32(b[2]), int32(b[3])}
	procClipCursor.Call(uintptr(unsafe.Pointer(&r)))
}
