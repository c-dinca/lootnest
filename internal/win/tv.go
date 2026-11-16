//go:build windows

package win

import (
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procAttachThreadInput  = user32.NewProc("AttachThreadInput")
	procSendInput          = user32.NewProc("SendInput")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
	procIsProcessInJob     = kernel32.NewProc("IsProcessInJob")
)

// AskCommand sends a text command as SendCommand does and returns the
// number the other window answered with (0 when it didn't answer).
func AskCommand(hwnd uintptr, cmd string) uintptr {
	b := []byte(cmd)
	if len(b) == 0 {
		return 0
	}
	cds := copyDataStruct{dwData: 0x50535750, cbData: uint32(len(b)), lpData: uintptr(unsafe.Pointer(&b[0]))}
	var result uintptr
	r, _, _ := procSendMessageTimeoutW.Call(hwnd, WM_COPYDATA, 0, uintptr(unsafe.Pointer(&cds)), 0x2 /*SMTO_ABORTIFHUNG*/, 5000, uintptr(unsafe.Pointer(&result)))
	runtime.KeepAlive(b)
	if r == 0 {
		return 0
	}
	return result
}

// WaitProcess blocks until the process with this id has ended. It returns
// at once when the process can't be opened (it's gone already).
func WaitProcess(pid uint32) {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	_, _ = windows.WaitForSingleObject(h, windows.INFINITE)
}

// StartOutsideJob starts a program as StartDetached does, outside the job
// object this process runs in when that job lets it (a streaming host runs
// its apps in one and ends the whole job when the stream stops). When the
// job doesn't, it starts the program inside it and says so with inJob.
func StartOutsideJob(exe, args, dir string) (inJob bool, err error) {
	start := func(flags uint32) error {
		cmd := exec.Command(exe)
		cmd.Dir = dir
		line := syscall.EscapeArg(exe)
		if strings.TrimSpace(args) != "" {
			line += " " + args
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: line, CreationFlags: flags}
		if err := cmd.Start(); err != nil {
			return err
		}
		return cmd.Process.Release()
	}
	if !InJob() {
		return false, start(0)
	}
	if start(0x01000000 /*CREATE_BREAKAWAY_FROM_JOB*/) == nil {
		return false, nil
	}
	return true, start(0)
}

// InJob says whether this process runs inside a job object, as apps a
// streaming host starts do.
func InJob() bool {
	var in int32
	r, _, _ := procIsProcessInJob.Call(uintptr(windows.CurrentProcess()), 0, uintptr(unsafe.Pointer(&in)))
	return r != 0 && in != 0
}

// ForceForeground brings the window to the front with the keyboard focus,
// even when Windows wouldn't let this process take it (after a game closed,
// with input coming only from a controller): it joins the input of the
// window in front for a moment, and as a last resort sends a lone Alt
// press, which counts as the user's input.
func ForceForeground(hwnd uintptr) {
	Present(hwnd)
	if fg, _, _ := procGetForegroundWindow.Call(); fg == hwnd {
		return
	}
	fg, _, _ := procGetForegroundWindow.Call()
	if fg != 0 {
		fgThread, _, _ := procGetWindowThreadProcessId.Call(fg, 0)
		me, _, _ := procGetCurrentThreadId.Call()
		if fgThread != 0 && fgThread != me {
			procAttachThreadInput.Call(me, fgThread, 1)
			procBringWindowToTop.Call(hwnd)
			procSetForegroundWindow.Call(hwnd)
			procAttachThreadInput.Call(me, fgThread, 0)
		}
	}
	if now, _, _ := procGetForegroundWindow.Call(); now == hwnd {
		return
	}
	altTap()
	procSetForegroundWindow.Call(hwnd)
}

// altTap sends Alt down and up.
func altTap() { TapKey(0x12) } // VK_MENU

// TapKey presses and lets go of a key, as the keyboard would, for the
// window in front.
func TapKey(vk uint16) {
	type keyInput struct {
		typ   uint32
		_     uint32 // padding to the union on 64-bit
		vk    uint16
		scan  uint16
		flags uint32
		time  uint32
		extra uintptr
		_     [8]byte // the union is as large as MOUSEINPUT
	}
	const extended, keyUp = 0x0001, 0x0002
	var flags uint32
	if vk >= 0x25 && vk <= 0x28 { // the arrows, not the number pad's
		flags = extended
	}
	in := [2]keyInput{{typ: 1, vk: vk, flags: flags}, {typ: 1, vk: vk, flags: flags | keyUp}}
	procSendInput.Call(2, uintptr(unsafe.Pointer(&in[0])), unsafe.Sizeof(in[0]))
}

// SendToBack puts the window under every other window without hiding it or
// taking focus: TV mode stays behind a game, covering the desktop, instead
// of vanishing.
func SendToBack(hwnd uintptr) {
	const (
		hwndBottom  = 1
		swpNoSize   = 0x0001
		swpNoMove   = 0x0002
		swpNoActive = 0x0010
		swpNoOwnerZ = 0x0200
	)
	procSetWindowPos.Call(hwnd, hwndBottom, 0, 0, 0, 0, swpNoSize|swpNoMove|swpNoActive|swpNoOwnerZ)
}

// ShowTaskbar shows or hides the main taskbar (the primary display's,
// which is the TV's while a stream runs). Other displays' taskbars stay.
func ShowTaskbar(show bool) {
	h := FindWindow("Shell_TrayWnd")
	if h == 0 {
		return
	}
	if show {
		procShowWindow.Call(h, 8) // SW_SHOWNA
	} else {
		procShowWindow.Call(h, SW_HIDE)
	}
}

// AllowAnyForeground lets the next program started (a game, through its
// launcher) come to the front over this one.
func AllowAnyForeground() {
	procAllowSetForegroundWindow.Call(^uintptr(0)) // ASFW_ANY
}

// StartTracked starts a program as StartDetached does and returns its
// process id, for an app followed by its own process.
func StartTracked(exe, args, dir string) (uint32, error) {
	cmd := exec.Command(exe)
	cmd.Dir = dir
	line := syscall.EscapeArg(exe)
	if strings.TrimSpace(args) != "" {
		line += " " + args
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: line}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := uint32(cmd.Process.Pid)
	_ = cmd.Process.Release()
	return pid, nil
}

// ProcessAlive says whether the process with this id still runs.
func ProcessAlive(pid uint32) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	ev, _ := windows.WaitForSingleObject(h, 0)
	return ev == uint32(windows.WAIT_TIMEOUT)
}

// WindowOfPID is the largest visible window of the process with this id.
func WindowOfPID(pid uint32) uintptr {
	findMu.Lock()
	defer findMu.Unlock()
	findPIDs, findBest, findArea = map[uint32]bool{pid: true}, 0, 0
	procEnumWindows.Call(findCB, 0)
	findPIDs = nil
	return findBest
}

// CloseWindowsOfPID asks every window of the process with this id to
// close, as closing them by hand would; it returns how many it asked.
func CloseWindowsOfPID(pid uint32) int {
	closeMu.Lock()
	defer closeMu.Unlock()
	closePIDs, closeN = map[uint32]bool{pid: true}, 0
	procEnumWindows.Call(closeCB, 0)
	n := closeN
	closePIDs = nil
	return n
}
