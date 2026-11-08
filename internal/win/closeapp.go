//go:build windows

package win

import (
	"os"
	"path/filepath"
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

// ForegroundIs reports whether the window in front belongs to a process
// named name (lower case, without .exe).
func ForegroundIs(name string) bool {
	var pid uint32
	procGetWindowThreadProcessId.Call(foreground(), uintptr(unsafe.Pointer(&pid)))
	return pid != 0 && processIDs(name)[pid]
}

// ProcessPaths returns the full paths of the running processes named name
// (lower case, without .exe), each once.
func ProcessPaths(name string) []string {
	var out []string
	seen := map[string]bool{}
	for pid := range processIDs(name) {
		if p := processPath(pid); p != "" && !seen[strings.ToLower(p)] {
			seen[strings.ToLower(p)] = true
			out = append(out, p)
		}
	}
	return out
}

func processPath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

// KillProcesses ends the processes named name (lower case, without .exe)
// at once, without asking them, and returns how many it ended.
func KillProcesses(name string) int {
	n := 0
	for pid := range processIDs(name) {
		h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
		if err != nil {
			continue
		}
		if windows.TerminateProcess(h, 1) == nil {
			n++
		}
		windows.CloseHandle(h)
	}
	return n
}

// RunningApp is a program with a window open.
type RunningApp struct {
	Name string `json:"name"` // its file name, "Discord.exe"
	Path string `json:"path"`
}

var (
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	appsMu                   sync.Mutex
	appsPIDs                 map[uint32]bool
	appsCB                   = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if vis, _, _ := procIsWindowVisible.Call(hwnd); vis != 0 {
			if n, _, _ := procGetWindowTextLengthW.Call(hwnd); n > 0 {
				var pid uint32
				procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
				appsPIDs[pid] = true
			}
		}
		return 1
	})
)

// RunningApps lists the programs with a visible, titled window, each once,
// leaving out Windows' own (in the Windows folder) and this one.
func RunningApps() []RunningApp {
	appsMu.Lock()
	appsPIDs = map[uint32]bool{}
	procEnumWindows.Call(appsCB, 0)
	pids := appsPIDs
	appsPIDs = nil
	appsMu.Unlock()
	sys := strings.ToLower(os.Getenv("SystemRoot")) + `\`
	self := uint32(os.Getpid())
	seen := map[string]bool{}
	out := []RunningApp{}
	for pid := range pids {
		p := processPath(pid)
		low := strings.ToLower(p)
		if p == "" || pid == self || strings.HasPrefix(low, sys) || seen[low] {
			continue
		}
		seen[low] = true
		out = append(out, RunningApp{Name: filepath.Base(p), Path: p})
	}
	return out
}
