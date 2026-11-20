//go:build windows

package win

import (
	"errors"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// The PC's own displays while TV mode is on (app/tvwindows.go): a window
// that opens on the TV's display (Windows' primary while a stream runs)
// is moved to the PC's, and Windows' taskbar shows on every display, with
// every app on each.

var (
	procIsZoomed        = user32.NewProc("IsZoomed")
	procGetClassNameW   = user32.NewProc("GetClassNameW")
	procMonitorFromRect = user32.NewProc("MonitorFromRect")
)

// WindowBox is a window's area {left, top, right, bottom}.
func WindowBox(hwnd uintptr) [4]int {
	var r rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return [4]int{int(r.left), int(r.top), int(r.right), int(r.bottom)}
}

// WindowClass is a window's class name.
func WindowClass(hwnd uintptr) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf[:n])
}

// ProcessName is a process's program name, lower case without ".exe" (as
// CloseProcessWindows takes it), or "" when Windows won't say.
func ProcessName(pid uint32) string {
	p := processPath(pid)
	if p == "" {
		return ""
	}
	return strings.TrimSuffix(strings.ToLower(filepath.Base(p)), ".exe")
}

// WorkAreaOf is the work area (without its taskbar) of the display that
// holds most of area.
func WorkAreaOf(area [4]int) [4]int {
	r := rect{int32(area[0]), int32(area[1]), int32(area[2]), int32(area[3])}
	mon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&r)), 2 /*MONITOR_DEFAULTTONEAREST*/)
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	w := mi.rcWork
	return [4]int{int(w.left), int(w.top), int(w.right), int(w.bottom)}
}

// MoveWindowTo puts a window at box without giving it the keyboard. A
// maximised window is maximised again where it lands, on that display.
func MoveWindowTo(hwnd uintptr, box [4]int) {
	const (
		swRestore   = 9
		swMaximize  = 3
		swpNoZOrder = 0x0004
		swpNoActive = 0x0010
	)
	zoomed, _, _ := procIsZoomed.Call(hwnd)
	if zoomed != 0 {
		procShowWindow.Call(hwnd, swRestore)
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(int32(box[0])), uintptr(int32(box[1])),
		uintptr(int32(box[2]-box[0])), uintptr(int32(box[3]-box[1])), swpNoZOrder|swpNoActive)
	if zoomed != 0 {
		procShowWindow.Call(hwnd, swMaximize)
	}
}

const taskbarKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`

// TaskbarSettings reads Windows' choices for the taskbar on several
// displays: Show my taskbar on all displays (all: MMTaskbarEnabled) and
// Show my taskbar apps on (mode: MMTaskbarMode; 0 all taskbars, 1 the main
// taskbar and the taskbar where the window is open, 2 the taskbar where
// the window is open). -1 is a value that isn't set.
func TaskbarSettings() (all, mode int) {
	all, mode = -1, -1
	k, err := registry.OpenKey(registry.CURRENT_USER, taskbarKey, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	if v, _, err := k.GetIntegerValue("MMTaskbarEnabled"); err == nil {
		all = int(v)
	}
	if v, _, err := k.GetIntegerValue("MMTaskbarMode"); err == nil {
		mode = int(v)
	}
	return
}

// SetTaskbarSettings writes those choices (-1 removes a value, Windows'
// own default) and tells Explorer, which redraws its taskbars as the
// Settings app makes it do.
func SetTaskbarSettings(all, mode int) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, taskbarKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	for _, v := range []struct {
		name string
		n    int
	}{{"MMTaskbarEnabled", all}, {"MMTaskbarMode", mode}} {
		if v.n < 0 {
			if err := k.DeleteValue(v.name); err != nil && !errors.Is(err, registry.ErrNotExist) {
				return err
			}
			continue
		}
		if err := k.SetDWordValue(v.name, uint32(v.n)); err != nil {
			return err
		}
	}
	const (
		hwndBroadcast     = 0xFFFF
		wmSettingChange   = 0x001A
		smtoAbortIfHung   = 0x0002
		settingChangeWait = 2000
	)
	what, _ := windows.UTF16PtrFromString("TraySettings")
	var result uintptr
	procSendMessageTimeoutW.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(what)), smtoAbortIfHung, settingChangeWait, uintptr(unsafe.Pointer(&result)))
	return nil
}
