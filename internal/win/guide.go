//go:build windows

package win

import (
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The Home button of a controller, for TV mode's menu over a game: an
// Xbox pad's Guide (and the Xbox pad a streaming host makes for a TV's
// controller) through XInput, and a PlayStation pad's PS button through
// raw input. Neither opens the device or sends it anything, so a
// Bluetooth DualSense keeps its mode (as in controllers.go).

const WM_INPUT = 0x00FF

var xinputStateEx = sync.OnceValue(func() uintptr {
	d := windows.NewLazySystemDLL("xinput1_4.dll")
	if d.Load() != nil {
		return 0
	}
	// XInputGetStateEx, ordinal 100: XInputGetState without the Guide
	// button masked out.
	p, err := windows.GetProcAddressByOrdinal(windows.Handle(d.Handle()), 100)
	if err != nil {
		return 0
	}
	return p
})

// XInputGuideDown says whether any XInput pad holds its Guide button.
func XInputGuideDown() bool {
	p := xinputStateEx()
	if p == 0 {
		return false
	}
	for i := 0; i < 4; i++ {
		var state [24]byte // XINPUT_STATE, with room to spare
		if r, _, _ := syscall.SyscallN(p, uintptr(i), uintptr(unsafe.Pointer(&state[0]))); r != 0 {
			continue
		}
		buttons := uint16(state[4]) | uint16(state[5])<<8
		if buttons&0x0400 != 0 {
			return true
		}
	}
	return false
}

var (
	procRegisterRawInputDevices = user32.NewProc("RegisterRawInputDevices")
	procGetRawInputData         = user32.NewProc("GetRawInputData")
	procGetRawInputDeviceInfoW  = user32.NewProc("GetRawInputDeviceInfoW")
)

type rawInputDevice struct {
	usagePage, usage uint16
	flags            uint32
	target           uintptr
}

// WatchPadRawInput asks for game pads' and joysticks' raw input at hwnd,
// even while another window is in front (on), or stops it (off).
func WatchPadRawInput(hwnd uintptr, on bool) bool {
	const (
		ridevRemove      = 0x00000001
		ridevInputSink   = 0x00000100
		usagePageGeneric = 0x01
		usageJoystick    = 0x04
		usageGamePad     = 0x05
	)
	flags, target := uint32(ridevInputSink), hwnd
	if !on {
		flags, target = ridevRemove, 0
	}
	devs := [2]rawInputDevice{
		{usagePageGeneric, usageGamePad, flags, target},
		{usagePageGeneric, usageJoystick, flags, target},
	}
	r, _, _ := procRegisterRawInputDevices.Call(uintptr(unsafe.Pointer(&devs[0])), 2, unsafe.Sizeof(devs[0]))
	return r != 0
}

var (
	padKindsMu sync.Mutex
	padKinds   = map[uintptr]uint32{} // device -> vendor<<16 | product
)

// RawPSButton reads a WM_INPUT's lParam: whether it's a PlayStation pad's
// report (ok) and whether its PS button is down in it.
func RawPSButton(lp uintptr) (down, ok bool) {
	const ridInput, rimTypeHID = 0x10000003, 2
	hdr := unsafe.Sizeof(rawInputHeader{})
	var size uint32
	procGetRawInputData.Call(lp, ridInput, 0, uintptr(unsafe.Pointer(&size)), hdr)
	if size == 0 || size > 4096 {
		return false, false
	}
	buf := make([]byte, size)
	if r, _, _ := procGetRawInputData.Call(lp, ridInput, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), hdr); r == ^uintptr(0) || r == 0 {
		return false, false
	}
	h := (*rawInputHeader)(unsafe.Pointer(&buf[0]))
	if h.typ != rimTypeHID || uintptr(len(buf)) < hdr+8 {
		return false, false
	}
	kind := padKind(h.device)
	if kind>>16 != 0x054c { // Sony
		return false, false
	}
	sizeHid := *(*uint32)(unsafe.Pointer(&buf[hdr]))
	count := *(*uint32)(unsafe.Pointer(&buf[hdr+4]))
	data := buf[hdr+8:]
	if count == 0 || sizeHid == 0 || uint64(sizeHid)*uint64(count) > uint64(len(data)) {
		return false, false
	}
	return PSButtonIn(uint16(kind), data[:sizeHid]), true
}

type rawInputHeader struct {
	typ, size uint32
	device    uintptr
	wParam    uintptr
}

func padKind(dev uintptr) uint32 {
	padKindsMu.Lock()
	defer padKindsMu.Unlock()
	if k, ok := padKinds[dev]; ok {
		return k
	}
	const ridiDeviceInfo = 0x2000000b
	var info struct {
		cbSize, typ          uint32
		vendor, product, ver uint32
		usagePage, usage     uint16
		_                    [8]byte
	}
	info.cbSize = uint32(unsafe.Sizeof(info))
	size := info.cbSize
	k := uint32(0)
	if r, _, _ := procGetRawInputDeviceInfoW.Call(dev, ridiDeviceInfo, uintptr(unsafe.Pointer(&info)), uintptr(unsafe.Pointer(&size))); int32(r) > 0 {
		k = info.vendor<<16 | info.product&0xFFFF
	}
	padKinds[dev] = k
	return k
}

// PSButtonIn says whether a Sony pad's input report has its PS button
// down. product is the USB product id; report starts with its report id.
func PSButtonIn(product uint16, report []byte) bool {
	at := -1
	dualSense := product == 0x0ce6 || product == 0x0df2
	switch {
	case len(report) == 0:
		return false
	case report[0] == 0x31 && dualSense: // DualSense, Bluetooth, full report
		at = 11
	case report[0] == 0x11 && !dualSense: // DualShock 4, Bluetooth, full report
		at = 10
	case report[0] == 0x01 && dualSense && len(report) >= 64: // DualSense, USB
		at = 10
	case report[0] == 0x01: // DualShock 4 over USB, or either's Bluetooth simple report
		at = 7
	}
	return at >= 0 && at < len(report) && report[at]&0x01 != 0
}
