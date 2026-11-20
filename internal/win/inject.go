//go:build windows

package win

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------------------------------------------------------------------------
// The in-game FPS counter (core/overlay.go): its shared block, and loading
// its DLL into a game, as Steam's and Discord's overlays do. app/overlay.go
// decides which games; nothing here checks.
// ---------------------------------------------------------------------------

var (
	procCreateRemoteThread = kernel32.NewProc("CreateRemoteThread")
	procVirtualAllocEx     = kernel32.NewProc("VirtualAllocEx")
	procVirtualFreeEx      = kernel32.NewProc("VirtualFreeEx")
	procWriteProcessMemory = kernel32.NewProc("WriteProcessMemory")
	procGetExitCodeThread  = kernel32.NewProc("GetExitCodeThread")
)

// ErrNot64Bit: the game is a 32-bit program, which the 64-bit DLL can't go into.
var ErrNot64Bit = errors.New("it's a 32-bit program")

// InjectDLL loads the DLL at path into process pid and waits for its
// LoadLibraryW to return (at most 10 seconds).
func InjectDLL(pid uint32, path string) error {
	const access = windows.PROCESS_CREATE_THREAD | windows.PROCESS_QUERY_INFORMATION |
		windows.PROCESS_VM_OPERATION | windows.PROCESS_VM_WRITE | windows.PROCESS_VM_READ
	h, err := windows.OpenProcess(access, false, pid)
	if err != nil {
		return fmt.Errorf("Windows didn't let Lootnest open it: %w", err)
	}
	defer windows.CloseHandle(h)
	var wow bool
	if err := windows.IsWow64Process(h, &wow); err == nil && wow {
		return ErrNot64Bit
	}
	p, err := windows.UTF16FromString(path)
	if err != nil {
		return err
	}
	size := uintptr(len(p) * 2)
	const memCommitReserve, pageReadWrite, memRelease = 0x3000, 0x04, 0x8000
	remote, _, e := procVirtualAllocEx.Call(uintptr(h), 0, size, memCommitReserve, pageReadWrite)
	if remote == 0 {
		return fmt.Errorf("no memory in it: %w", e)
	}
	defer procVirtualFreeEx.Call(uintptr(h), remote, 0, memRelease)
	var wrote uintptr
	if r, _, e := procWriteProcessMemory.Call(uintptr(h), remote, uintptr(unsafe.Pointer(&p[0])), size, uintptr(unsafe.Pointer(&wrote))); r == 0 {
		return fmt.Errorf("couldn't write the DLL's path: %w", e)
	}
	// kernel32 sits at the same address in every 64-bit process of a boot,
	// so this process's LoadLibraryW is the game's too.
	load := kernel32.NewProc("LoadLibraryW")
	if err := load.Find(); err != nil {
		return err
	}
	th, _, e := procCreateRemoteThread.Call(uintptr(h), 0, 0, load.Addr(), remote, 0, 0)
	if th == 0 {
		return fmt.Errorf("couldn't start a thread in it: %w", e)
	}
	defer windows.CloseHandle(windows.Handle(th))
	ev, _ := windows.WaitForSingleObject(windows.Handle(th), 10000)
	if ev != windows.WAIT_OBJECT_0 {
		return errors.New("its LoadLibrary didn't return in 10 seconds")
	}
	var code uint32
	procGetExitCodeThread.Call(th, uintptr(unsafe.Pointer(&code)))
	if code == 0 { // the module's handle, truncated to 32 bits; 0 is a failure
		return errors.New("it refused the DLL")
	}
	return nil
}

// Is64BitProcess says whether pid is a 64-bit program (false when it can't
// be asked).
func Is64BitProcess(pid uint32) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var wow bool
	return windows.IsWow64Process(h, &wow) == nil && !wow
}

// OverlayShare is the block "Local\LootnestOverlay-<pid>" (core.OverlaySize
// bytes, laid out as core's Over* offsets) shared with the DLL in a game.
type OverlayShare struct {
	h windows.Handle
	p uintptr
}

// NewOverlayShare makes the block for the game pid, before the DLL goes in.
func NewOverlayShare(pid uint32, magic, version uint32, size int) (*OverlayShare, error) {
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf(`Local\LootnestOverlay-%d`, pid))
	h, err := windows.CreateFileMapping(windows.InvalidHandle, nil, windows.PAGE_READWRITE, 0, uint32(size), name)
	if err != nil {
		return nil, err
	}
	p, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ|windows.FILE_MAP_WRITE, 0, 0, uintptr(size))
	if err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	s := &OverlayShare{h, p}
	if s.Int32(0) != int32(magic) { // a fresh block (an older one from the same pid is kept)
		s.SetInt32(4, int32(version))
		s.SetInt32(0, int32(magic))
	}
	return s, nil
}

// Int32 reads the int32 at off; SetInt32 writes it.
func (s *OverlayShare) Int32(off int) int32 {
	return *(*int32)(unsafe.Pointer(s.p + uintptr(off)))
}

func (s *OverlayShare) SetInt32(off int, v int32) {
	*(*int32)(unsafe.Pointer(s.p + uintptr(off))) = v
}

// Int64 reads the int64 at off.
func (s *OverlayShare) Int64(off int) int64 {
	return *(*int64)(unsafe.Pointer(s.p + uintptr(off)))
}

// Close lets the block go (the DLL keeps its own view).
func (s *OverlayShare) Close() {
	if s.p != 0 {
		windows.UnmapViewOfFile(s.p)
		windows.CloseHandle(s.h)
		s.p = 0
	}
}

// ProcessPath is a process's program, "" when Windows won't say.
func ProcessPath(pid uint32) string { return processPath(pid) }
