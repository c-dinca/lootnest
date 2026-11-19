//go:build windows

package win

import (
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procPeekMessageW = user32.NewProc("PeekMessageW")

// VK_Q is the Q key.
const VK_Q = 0x51

// EmergencyKey is the key WatchEmergencyKey holds, in words.
const EmergencyKey = "Ctrl+Shift+Q"

// WatchEmergencyKey holds Ctrl+Shift+Q on a thread of its own while armed
// says so, and calls fire on a new goroutine when it's pressed. It never
// waits on Lootnest's window, so it answers when that window doesn't.
// held is told whether the key could be held each time armed changes
// (false when another program has it).
func WatchEmergencyKey(armed func() bool, fire func(), held func(bool)) {
	go func() {
		runtime.LockOSThread()
		const id = 0x4C51 // "LQ"
		const pmRemove = 1
		on := false
		var m msgT
		for {
			if want := armed(); want != on {
				on = want
				if on {
					r, _, _ := procRegisterHotKey.Call(0, id, ModControl|ModShift|ModNoRepeat, VK_Q)
					held(r != 0)
				} else {
					procUnregisterHotKey.Call(0, id)
				}
			}
			for {
				r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, WM_HOTKEY, WM_HOTKEY, pmRemove)
				if r == 0 {
					break
				}
				if m.message == WM_HOTKEY && m.wParam == id {
					go fire()
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// KillPID ends a process at once, without asking it.
func KillPID(pid uint32) bool {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	return windows.TerminateProcess(h, 1) == nil
}
