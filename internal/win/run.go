//go:build windows

package win

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procShellExecuteExW = shell32.NewProc("ShellExecuteExW")

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hIcon        uintptr
	hProcess     windows.Handle
}

// ErrCancelledByUser: the user said no to Windows' "allow this app to make
// changes" prompt.
var ErrCancelledByUser = errors.New("the installer was cancelled at Windows' prompt")

// RunAndWait starts a program the way Explorer would (so an installer that
// needs administrator rights gets Windows' prompt), waits for it to close,
// and returns its exit code.
func RunAndWait(owner uintptr, exe, dir string) (uint32, error) {
	verb, _ := windows.UTF16PtrFromString("open")
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return 0, err
	}
	var d *uint16
	if dir != "" {
		d, _ = windows.UTF16PtrFromString(dir)
	}
	const seeMaskNoCloseProcess = 0x00000040
	info := shellExecuteInfo{fMask: seeMaskNoCloseProcess, hwnd: owner, lpVerb: verb, lpFile: file, lpDirectory: d, nShow: windows.SW_SHOWNORMAL}
	info.cbSize = uint32(unsafe.Sizeof(info))
	if r, _, e := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info))); r == 0 {
		if errors.Is(e, windows.ERROR_CANCELLED) {
			return 0, ErrCancelledByUser
		}
		return 0, e
	}
	if info.hProcess == 0 {
		return 0, nil // handed to a process that was already running
	}
	defer windows.CloseHandle(info.hProcess)
	if _, err := windows.WaitForSingleObject(info.hProcess, windows.INFINITE); err != nil {
		return 0, err
	}
	var code uint32
	_ = windows.GetExitCodeProcess(info.hProcess, &code)
	return code, nil
}
