package core

import (
	"fmt"
	"testing"
)

// dsUSB is a DualSense's USB report with everything at rest.
func dsUSB() []byte {
	r := make([]byte, 64)
	r[0] = 0x01
	r[1], r[2], r[3], r[4] = 128, 128, 128, 128
	r[8] = 0x08 // the D-pad at rest
	return r
}

func TestPSToXboxDualSenseUSB(t *testing.T) {
	s, ok := PSToXbox(true, false, dsUSB())
	if !ok || s != (PadState{}) {
		t.Fatalf("at rest: %+v ok=%v, want all zero", s, ok)
	}
	r := dsUSB()
	r[1], r[2] = 255, 0 // left stick right and up
	r[3], r[4] = 0, 255 // right stick left and down
	r[5], r[6] = 40, 255
	r[8] = 0x20 | 0x80 | 0x02 // cross, triangle, D-pad right
	r[9] = 0x01 | 0x20 | 0x80 // L1, Options, R3
	r[10] = 0x01              // PS: not passed on
	s, ok = PSToXbox(true, false, r)
	want := PadState{Buttons: XA | XY | XRight | XLB | XStart | XRThumb, LT: 40, RT: 255, LX: 32639, LY: 32767, RX: -32768, RY: -32639}
	if !ok || s != want {
		t.Fatalf("got %+v ok=%v\nwant %+v", s, ok, want)
	}
}

func TestPSToXboxOtherReports(t *testing.T) {
	// A DualSense over Bluetooth, short report: the DualShock 4's layout.
	// Windows pads it to the pad's longest report.
	short := make([]byte, 78)
	copy(short, []byte{0x01, 128, 128, 128, 128, 0x40 | 0x06, 0x02 | 0x10, 0x01, 9, 200})
	s, ok := PSToXbox(true, true, short)
	if want := (PadState{Buttons: XB | XLeft | XRB | XBack, LT: 9, RT: 200}); !ok || s != want {
		t.Errorf("DualSense Bluetooth short: %+v ok=%v, want %+v", s, ok, want)
	}
	// The long Bluetooth report is USB's, one byte further on.
	long := make([]byte, 78)
	long[0] = 0x31
	long[2], long[3], long[4], long[5] = 128, 128, 128, 128
	long[6] = 77
	long[9] = 0x10 | 0x00 // square, D-pad up
	long[10] = 0x40       // L3
	s, ok = PSToXbox(true, true, long)
	if want := (PadState{Buttons: XX | XUp | XLThumb, LT: 77}); !ok || s != want {
		t.Errorf("DualSense Bluetooth long: %+v ok=%v, want %+v", s, ok, want)
	}
	// A DualShock 4 over Bluetooth, two bytes further on than USB.
	ds4 := make([]byte, 78)
	ds4[0] = 0x11
	ds4[3], ds4[4], ds4[5], ds4[6] = 128, 128, 128, 128
	ds4[7] = 0x20 | 0x08 // cross, D-pad at rest
	ds4[8] = 0x20        // Options
	ds4[11] = 31
	s, ok = PSToXbox(false, true, ds4)
	if want := (PadState{Buttons: XA | XStart, RT: 31}); !ok || s != want {
		t.Errorf("DualShock 4 Bluetooth: %+v ok=%v, want %+v", s, ok, want)
	}
	for name, r := range map[string][]byte{"empty": nil, "too short": {0x01, 1, 2}, "another report": {0x05, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}} {
		if _, ok := PSToXbox(true, false, r); ok {
			t.Errorf("%s: read as a pad report", name)
		}
	}
	if _, ok := PSToXbox(true, true, ds4); ok {
		t.Errorf("a DualShock 4 report read as a DualSense's")
	}
	// A DualShock 4 on USB: the short layout.
	usb := make([]byte, 64)
	copy(usb, []byte{0x01, 128, 128, 128, 128, 0x80 | 0x04, 0x01, 0, 255, 0})
	s, ok = PSToXbox(false, false, usb)
	if want := (PadState{Buttons: XY | XDown | XLB, LT: 255}); !ok || s != want {
		t.Errorf("DualShock 4 USB: %+v ok=%v, want %+v", s, ok, want)
	}
}

func TestPadHelpers(t *testing.T) {
	if ds, ok := PSPad(0x0CE6); !ds || !ok {
		t.Errorf("DualSense not known")
	}
	if ds, ok := PSPad(0x09CC); ds || !ok {
		t.Errorf("DualShock 4 not known, or taken for a DualSense")
	}
	if _, ok := PSPad(0x028E); ok {
		t.Errorf("an Xbox 360 pad taken for a PlayStation one")
	}
	got := PadState{Buttons: 0x1234, LT: 5, RT: 6, LX: -2, LY: 3, RX: 32767, RY: -32768}.XUSBReport()
	want := [12]byte{0x34, 0x12, 5, 6, 0xFE, 0xFF, 3, 0, 0xFF, 0x7F, 0x00, 0x80}
	if got != want {
		t.Errorf("XUSBReport: % x, want % x", got, want)
	}
}

func TestPSPadPath(t *testing.T) {
	cases := []struct {
		path string
		pid  uint16
		bt   bool
		ok   bool
	}{
		{`\\?\HID#VID_054C&PID_0CE6&MI_03#8&2f5d1e9&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`, 0x0CE6, false, true},
		{`\\?\hid#{00001124-0000-1000-8000-00805f9b34fb}_vid&0002054c_pid&0ce6#9&1a2b&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`, 0x0CE6, true, true},
		{`\\?\hid#vid_054c&pid_09cc&mi_03#7&abc&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`, 0x09CC, false, true},
		{`\\?\hid#vid_045e&pid_028e&ig_00#3&1&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`, 0, false, false},
		{`\\?\hid#vid_054c&pid_1234#1#{4d1e55b2-f16f-11cf-88cb-001111000030}`, 0, false, false}, // a Sony device that isn't a pad
		{`\\?\hid#vid_054c&pid_0c`, 0, false, false},
		{``, 0, false, false},
	}
	for _, c := range cases {
		pid, bt, ok := PSPadPath(c.path)
		if pid != c.pid || bt != c.bt || ok != c.ok {
			t.Errorf("%s: got %04x %v %v, want %04x %v %v", c.path, pid, bt, ok, c.pid, c.bt, c.ok)
		}
	}
}

func TestMergePads(t *testing.T) {
	a := PadState{Buttons: XA, LX: 1000, LY: -30000, LT: 10}
	b := PadState{Buttons: XUp, LX: -20000, LY: 500, RT: 200}
	got := MergePads(a, b)
	want := PadState{Buttons: XA | XUp, LX: -20000, LY: -30000, LT: 10, RT: 200}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if MergePads() != (PadState{}) {
		t.Error("no pads should be a pad at rest")
	}
	if got := MergePads(PadState{LX: -32768}); got.LX != -32768 {
		t.Errorf("the far left was lost: %d", got.LX)
	}
}

func TestPadAxis(t *testing.T) {
	cases := []struct {
		v    int16
		down bool
		want float64
	}{
		{0, false, 0}, {0, true, 0}, {32767, false, 1}, {-32768, false, -1},
		{32767, true, -1}, // XInput's up is the Gamepad API's -1
		{-32768, true, 1},
		{800, false, 0}, // a stick resting a little off centre
		{16384, false, 0.5},
		{-22000, true, 0.65},
	}
	for _, c := range cases {
		if got := PadAxis(c.v, c.down); got != c.want {
			t.Errorf("PadAxis(%d, %v) = %v, want %v", c.v, c.down, got, c.want)
		}
	}
	if s := fmt.Sprint(PadAxis(100, true)); s != "0" {
		t.Errorf("a stick at rest reads %s, not 0", s)
	}
}
