//go:build windows

package win

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procEnumWindows = user32.NewProc("EnumWindows")

// processIDs returns the ids of running processes named name (lower case,
// without .exe).
func processIDs(name string) map[uint32]bool {
	out := map[uint32]bool{}
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return out
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Process32First(snap, &e); err != nil {
		return out
	}
	for {
		n := strings.TrimSuffix(strings.ToLower(windows.UTF16ToString(e.ExeFile[:])), ".exe")
		if n == name {
			out[e.ProcessID] = true
		}
		if err := windows.Process32Next(snap, &e); err != nil {
			break
		}
	}
	return out
}

// One callback for the whole program (Windows limits how many can be made);
// closeMu guards the state it reads.
var (
	closeMu   sync.Mutex
	closePIDs map[uint32]bool
	closeN    int
	closeCB   = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if closePIDs[pid] {
			if vis, _, _ := procIsWindowVisible.Call(hwnd); vis != 0 {
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
				closeN++
			}
		}
		return 1 // keep going
	})
)

// CloseProcessWindows asks every visible window of the processes named name
// (lower case, without .exe) to close, as if its X was clicked, so the
// program can save and quit the way it normally does. It returns how many
// windows were asked; 0 means the program has no window to close.
func CloseProcessWindows(name string) int {
	pids := processIDs(name)
	if len(pids) == 0 {
		return 0
	}
	closeMu.Lock()
	defer closeMu.Unlock()
	closePIDs, closeN = pids, 0
	procEnumWindows.Call(closeCB, 0)
	n := closeN
	closePIDs = nil
	return n
}
