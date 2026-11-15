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
func altTap() {
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
	const vkMenu, keyUp = 0x12, 0x0002
	in := [2]keyInput{{typ: 1, vk: vkMenu}, {typ: 1, vk: vkMenu, flags: keyUp}}
	procSendInput.Call(2, uintptr(unsafe.Pointer(&in[0])), unsafe.Sizeof(in[0]))
}
