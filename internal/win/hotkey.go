//go:build windows

package win

import (
	"time"
	"unsafe"
)

var (
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
)

// Hot key modifiers and keys (RegisterHotKey).
const (
	ModAlt      = 0x0001
	ModControl  = 0x0002
	ModShift    = 0x0004
	ModNoRepeat = 0x4000
	VK_F5       = 0x74
	VK_F10      = 0x79
	VK_F11      = 0x7A
	VK_F12      = 0x7B
	WM_HOTKEY   = 0x0312
)

// RegisterHotKey asks Windows to send hwnd WM_HOTKEY (wParam id) for a key,
// wherever the focus is. False when another program has it. Call it on the
// window's thread.
func RegisterHotKey(hwnd uintptr, id int, mods uint32, vk uint32) bool {
	r, _, _ := procRegisterHotKey.Call(hwnd, uintptr(id), uintptr(mods), uintptr(vk))
	return r != 0
}

func UnregisterHotKey(hwnd uintptr, id int) {
	procUnregisterHotKey.Call(hwnd, uintptr(id))
}

var procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")

// ShiftDown says whether a Shift key is held right now.
func ShiftDown() bool {
	r, _, _ := procGetAsyncKeyState.Call(0x10) // VK_SHIFT
	return r&0x8000 != 0
}

// ModifiersDown says whether Ctrl, Shift, Alt or a Windows key is held:
// a key pressed for a game while one is held would reach it as a shortcut
// (Ctrl+Shift+F5 rather than F5).
func ModifiersDown() bool {
	for _, vk := range []uintptr{0x10, 0x11, 0x12, 0x5B, 0x5C} { // Shift, Ctrl, Alt, left and right Windows
		if r, _, _ := procGetAsyncKeyState.Call(vk); r&0x8000 != 0 {
			return true
		}
	}
	return false
}

var procMapVirtualKey = user32.NewProc("MapVirtualKeyW")

// TapGameKey presses a key for the window in front the way a keyboard
// does for a game: by its scan code, which games reading raw or
// DirectInput keys need, and held for a few frames so a game that looks
// once a frame sees it.
func TapGameKey(vk uint16) {
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
	const scanCode, keyUp = 0x0008, 0x0002
	sc, _, _ := procMapVirtualKey.Call(uintptr(vk), 0) // MAPVK_VK_TO_VSC
	down := keyInput{typ: 1, vk: vk, scan: uint16(sc), flags: scanCode}
	up := down
	up.flags |= keyUp
	procSendInput.Call(1, uintptr(unsafe.Pointer(&down)), unsafe.Sizeof(down))
	time.Sleep(80 * time.Millisecond)
	procSendInput.Call(1, uintptr(unsafe.Pointer(&up)), unsafe.Sizeof(up))
}
