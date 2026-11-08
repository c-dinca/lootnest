//go:build windows

package win

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
