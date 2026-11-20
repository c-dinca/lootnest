package core

import (
	"math"
	"strconv"
	"strings"
)

// Turning a PlayStation controller into an Xbox one for games that only
// know Xbox pads (app/padbridge.go). Lootnest reads the pad's own input
// reports and hands a virtual Xbox 360 pad the same state, through
// ViGEmBus when that driver is already on the PC. This file is the part
// that needs no Windows: what a report says, and what the Xbox pad shows.

// PadState is a controller as games read an Xbox one: sticks from -32768
// to 32767 with up and right positive, triggers from 0 to 255, and the
// buttons as XInput's bits.
type PadState struct {
	Buttons        uint16
	LT, RT         uint8
	LX, LY, RX, RY int16
}

// XInput's button bits (XINPUT_GAMEPAD_*).
const (
	XUp     = 0x0001
	XDown   = 0x0002
	XLeft   = 0x0004
	XRight  = 0x0008
	XStart  = 0x0010
	XBack   = 0x0020
	XLThumb = 0x0040
	XRThumb = 0x0080
	XLB     = 0x0100
	XRB     = 0x0200
	XA      = 0x1000
	XB      = 0x2000
	XX      = 0x4000
	XY      = 0x8000
)

// Sony's vendor id, and the pads Lootnest reads.
const SonyVendor = 0x054C

// PSPad says whether a product id is a DualSense (and Edge) or a
// DualShock 4 (both versions, and its USB wireless adaptor), and which.
func PSPad(pid uint16) (dualsense, ok bool) {
	switch pid {
	case 0x0CE6, 0x0DF2:
		return true, true
	case 0x05C4, 0x09CC, 0x0BA0:
		return false, true
	}
	return false, false
}

// PSPadPath reads a HID device's path as Windows names it: a PlayStation
// pad's has Sony's vendor id and the pad's product id, written
// "vid_054c&pid_0ce6" on USB and "vid&0002054c_pid&0ce6" over Bluetooth.
// ok is false for any other device.
func PSPadPath(path string) (pid uint16, bluetooth, ok bool) {
	p := strings.ToLower(path)
	for _, f := range []struct {
		mark string
		bt   bool
	}{{"vid_054c&pid_", false}, {"vid&0002054c_pid&", true}} {
		i := strings.Index(p, f.mark)
		if i < 0 || len(p) < i+len(f.mark)+4 {
			continue
		}
		n, err := strconv.ParseUint(p[i+len(f.mark):i+len(f.mark)+4], 16, 16)
		if err != nil {
			continue
		}
		if _, known := PSPad(uint16(n)); known {
			return uint16(n), f.bt, true
		}
	}
	return 0, false, false
}

// psLayout is where a report keeps the sticks, the triggers and the three
// button bytes, counted from the report id at 0.
type psLayout struct{ sticks, lt, rt, b0 int }

// psReport finds the layout of one input report. A DualSense on USB sends
// 0x01 with the triggers before the buttons; over Bluetooth it sends the
// short 0x01 (the DualShock 4's layout) until something asks for the long
// 0x31, which is USB's layout one byte further on. A DualShock 4 sends
// 0x01 on USB and 0x11 over Bluetooth, two bytes further on. Windows
// hands every report in a buffer as long as the pad's longest, so the
// length says nothing: how the pad is connected does.
func psReport(dualsense, bluetooth bool, r []byte) (psLayout, bool) {
	if len(r) < 12 {
		return psLayout{}, false
	}
	switch {
	case dualsense && !bluetooth && r[0] == 0x01:
		return psLayout{sticks: 1, lt: 5, rt: 6, b0: 8}, true
	case dualsense && bluetooth && r[0] == 0x31:
		return psLayout{sticks: 2, lt: 6, rt: 7, b0: 9}, true
	case !dualsense && bluetooth && r[0] == 0x11:
		return psLayout{sticks: 3, b0: 7, lt: 10, rt: 11}, true
	case r[0] == 0x01:
		return psLayout{sticks: 1, b0: 5, lt: 8, rt: 9}, true
	}
	return psLayout{}, false
}

// stick turns a byte (0 left or up, 128 centre, 255 right or down) into
// XInput's range; flip is for the vertical axis, where XInput's up is
// positive.
func stick(v byte, flip bool) int16 {
	n := (int(v) - 128) * 257
	if n > 32767 {
		n = 32767
	}
	if flip {
		n = -n
		if n > 32767 {
			n = 32767
		}
	}
	if n < -32768 {
		n = -32768
	}
	return int16(n)
}

// The D-pad is a hat: 0 is up, then clockwise in eighths; 8 is nothing.
var hatBits = [8]uint16{XUp, XUp | XRight, XRight, XRight | XDown, XDown, XDown | XLeft, XLeft, XLeft | XUp}

// PSToXbox reads one input report of a PlayStation pad as an Xbox pad's
// state: cross is A, circle B, square X, triangle Y, L1 and R1 the
// bumpers, L2 and R2 the triggers, Create or Share is Back (View), Options
// is Start (Menu). The PS button isn't passed on: in Big Picture and TV
// mode it is Lootnest's Home button, read on its own. ok is false for a
// report Lootnest doesn't know.
func PSToXbox(dualsense, bluetooth bool, report []byte) (PadState, bool) {
	l, ok := psReport(dualsense, bluetooth, report)
	if !ok {
		return PadState{}, false
	}
	s := PadState{
		LX: stick(report[l.sticks], false), LY: stick(report[l.sticks+1], true),
		RX: stick(report[l.sticks+2], false), RY: stick(report[l.sticks+3], true),
		LT: report[l.lt], RT: report[l.rt],
	}
	b0, b1 := report[l.b0], report[l.b0+1]
	if hat := b0 & 0x0F; hat < 8 {
		s.Buttons |= hatBits[hat]
	}
	for _, m := range [...]struct {
		in  byte
		bit byte
		out uint16
	}{
		{b0, 0x10, XX}, {b0, 0x20, XA}, {b0, 0x40, XB}, {b0, 0x80, XY},
		{b1, 0x01, XLB}, {b1, 0x02, XRB}, {b1, 0x10, XBack}, {b1, 0x20, XStart}, {b1, 0x40, XLThumb}, {b1, 0x80, XRThumb},
	} {
		if m.in&m.bit != 0 {
			s.Buttons |= m.out
		}
	}
	return s, true
}

// XUSBReport is a pad's state as ViGEmBus takes it for an Xbox 360 pad
// (XUSB_REPORT: buttons, the two triggers, four axes, little-endian).
func (s PadState) XUSBReport() [12]byte {
	le := func(v uint16) (byte, byte) { return byte(v), byte(v >> 8) }
	var b [12]byte
	b[0], b[1] = le(s.Buttons)
	b[2], b[3] = s.LT, s.RT
	b[4], b[5] = le(uint16(s.LX))
	b[6], b[7] = le(uint16(s.LY))
	b[8], b[9] = le(uint16(s.RX))
	b[10], b[11] = le(uint16(s.RY))
	return b
}

// MergePads is several controllers as one, as the page reads them while
// it can't read pads itself (app/padfeed.go): every button any of them
// holds, and each stick and trigger from the one pushed furthest.
func MergePads(pads ...PadState) PadState {
	var out PadState
	far := func(a, b int16) int16 {
		if absPad(b) > absPad(a) {
			return b
		}
		return a
	}
	for _, p := range pads {
		out.Buttons |= p.Buttons
		out.LX, out.LY = far(out.LX, p.LX), far(out.LY, p.LY)
		out.RX, out.RY = far(out.RX, p.RX), far(out.RY, p.RY)
		out.LT, out.RT = max(out.LT, p.LT), max(out.RT, p.RT)
	}
	return out
}

func absPad(v int16) int32 {
	if v < 0 {
		return -int32(v)
	}
	return int32(v)
}

// PadAxis is a stick's position as the Gamepad API gives it, from -1 to 1
// in steps of 0.05, so a stick resting a little off centre doesn't send
// the page a change every frame. down is for the vertical axis: XInput's
// up is positive, the Gamepad API's is negative.
func PadAxis(v int16, down bool) float64 {
	f := float64(v) / 32767
	if down {
		f = -f
	}
	f = math.Round(f*20) / 20
	return max(-1, min(1, f)) + 0 // + 0 turns -0 into 0
}
