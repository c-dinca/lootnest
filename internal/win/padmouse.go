//go:build windows

package win

import (
	"syscall"
	"unsafe"
)

// The controller as a mouse (app/pointer.go): XInput's sticks and
// buttons, and a pointer, buttons and wheel moved as the mouse's own.

// XPad is what XInput's controllers hold now, as one: any button any of
// them holds, and the stick furthest over.
type XPad struct {
	Buttons        uint16
	LX, LY, RX, RY int16
}

// XInput's button flags.
const (
	XUp    = 0x0001
	XDown  = 0x0002
	XLeft  = 0x0004
	XRight = 0x0008
	XA     = 0x1000
	XB     = 0x2000
	XX     = 0x4000
	XY     = 0x8000
)

// XInputRead reads every XInput controller; false when there's none.
func XInputRead() (XPad, bool) {
	p := xinputStateEx()
	if p == 0 {
		return XPad{}, false
	}
	var out XPad
	var any bool
	far := func(a, b int16) int16 {
		abs := func(v int16) int32 {
			if v < 0 {
				return -int32(v)
			}
			return int32(v)
		}
		if abs(b) > abs(a) {
			return b
		}
		return a
	}
	i16 := func(b []byte) int16 { return int16(uint16(b[0]) | uint16(b[1])<<8) }
	for i := 0; i < 4; i++ {
		var state [24]byte // XINPUT_STATE, with room to spare
		if r, _, _ := syscall.SyscallN(p, uintptr(i), uintptr(unsafe.Pointer(&state[0]))); r != 0 {
			continue
		}
		any = true
		out.Buttons |= uint16(state[4]) | uint16(state[5])<<8
		out.LX, out.LY = far(out.LX, i16(state[8:])), far(out.LY, i16(state[10:]))
		out.RX, out.RY = far(out.RX, i16(state[12:])), far(out.RY, i16(state[14:]))
	}
	return out, any
}

// mouseInput is INPUT with its MOUSEINPUT, on 64-bit Windows.
type mouseInput struct {
	typ   uint32
	_     uint32
	dx    int32
	dy    int32
	data  uint32
	flags uint32
	time  uint32
	_     uint32
	extra uintptr
}

func sendMouse(dx, dy int32, data uint32, flags uint32) {
	in := mouseInput{typ: 0, dx: dx, dy: dy, data: data, flags: flags}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

// MouseMove moves the pointer by this many pixels, right and down positive.
func MouseMove(dx, dy int) {
	if dx != 0 || dy != 0 {
		sendMouse(int32(dx), int32(dy), 0, 0x0001) // MOUSEEVENTF_MOVE
	}
}

// MouseButton presses (down) or lets go of the left button, or the right.
func MouseButton(right, down bool) {
	flags := map[[2]bool]uint32{{false, true}: 0x0002, {false, false}: 0x0004, {true, true}: 0x0008, {true, false}: 0x0010}[[2]bool{right, down}]
	sendMouse(0, 0, 0, flags)
}

// MouseWheel turns the wheel: 120 is a notch, up positive.
func MouseWheel(delta int) {
	if delta != 0 {
		sendMouse(0, 0, uint32(int32(delta)), 0x0800) // MOUSEEVENTF_WHEEL
	}
}
