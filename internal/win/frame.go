//go:build windows

package win

import "unsafe"

// Lootnest draws its own title bar inside the page, as Steam and other
// apps do: the window keeps its frame (resizing at the edges, snapping,
// the shadow, Windows 11's rounded corners and border), but not the
// caption. FrameSize answers WM_NCCALCSIZE for that; the page's bar calls
// DragWindow, ResizeWindow and WindowKey.

const (
	WM_NCCALCSIZE = 0x0083
	WM_SIZE       = 0x0005

	wmNCLButtonDown = 0x00A1
	wsCaption       = 0x00C00000
	swMinimize      = 6
	swMaximize      = 3
)

var procReleaseCapture = user32.NewProc("ReleaseCapture")

// FrameSize answers WM_NCCALCSIZE for a window whose caption is drawn by
// its page: the system's frame at the sides and the bottom, none at the
// top. Maximized, the window hangs past the screen by the frame's
// thickness on every side, so the top is kept in by as much as the left.
// A window without a caption (full screen) is left to the system.
func FrameSize(hwnd, wp, lp uintptr) (uintptr, bool) {
	if wp == 0 || lp == 0 || !HasCaption(hwnd) {
		return 0, false
	}
	r := (*rect)(unsafe.Pointer(lp)) // NCCALCSIZE_PARAMS starts with the proposed window rect
	top, left := r.top, r.left
	procDefWindowProcW.Call(hwnd, WM_NCCALCSIZE, wp, lp)
	r.top = top
	if IsZoomed(hwnd) {
		r.top += r.left - left
	}
	return 0, true
}

// RefreshFrame makes the system ask WM_NCCALCSIZE again (after the
// window's hook is in place).
func RefreshFrame(hwnd uintptr) {
	const swpNoMove, swpNoSize, swpNoZOrder, swpNoActivate, swpFrameChanged = 0x2, 0x1, 0x4, 0x10, 0x20
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
}

// HasCaption: the window has its normal frame (not full screen).
func HasCaption(hwnd uintptr) bool {
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	return style&wsCaption == wsCaption
}

// IsZoomed: the window is maximized.
func IsZoomed(hwnd uintptr) bool {
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}

// hitOf is the system's hit-test code for a part of the frame the page's
// bar stands in for.
var hitOf = map[string]uintptr{
	"caption": 2, "left": 10, "right": 11, "top": 12, "topleft": 13, "topright": 14,
}

// DragWindow lets the system move (part "caption") or resize ("top",
// "topleft", "topright") the window from a press in the page, as if that
// part of a real frame had been pressed: snapping, a maximized window
// coming down as it's dragged, and the system's own loop. Call it on the
// window's thread while the mouse button is down.
func DragWindow(hwnd uintptr, part string) {
	hit, ok := hitOf[part]
	if !ok || !HasCaption(hwnd) {
		return
	}
	if hit != 2 && IsZoomed(hwnd) {
		return
	}
	var pt struct{ x, y int32 }
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procReleaseCapture.Call()
	procPostMessageW.Call(hwnd, wmNCLButtonDown, hit, uintptr(uint16(pt.x))|uintptr(uint16(pt.y))<<16)
}

// WindowKey does what a caption's key does: "min", "max" (maximize, or
// restore when maximized) or "close" (WM_CLOSE, so Close to tray holds).
func WindowKey(hwnd uintptr, key string) {
	switch key {
	case "min":
		procShowWindow.Call(hwnd, swMinimize)
	case "max":
		if !HasCaption(hwnd) {
			return
		}
		if IsZoomed(hwnd) {
			procShowWindow.Call(hwnd, SW_RESTORE)
		} else {
			procShowWindow.Call(hwnd, swMaximize)
		}
	case "close":
		procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
	}
}
