package core

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestPSRumble(t *testing.T) {
	// DualSense on USB: 48 bytes, flags 0x03, right (weak) then left (strong), flags2 0x04.
	b, err := PSRumble(true, false, 48, 0, 200, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 48 || b[0] != 0x02 || b[1] != 0x03 || b[2] != 0 || b[3] != 100 || b[4] != 200 || b[39] != 0x04 {
		t.Fatalf("DualSense USB: % x", b[:48])
	}
	// Nothing else marked valid: the lights and triggers stay as they are.
	for i, v := range b {
		if v != 0 && i != 0 && i != 1 && i != 3 && i != 4 && i != 39 {
			t.Fatalf("DualSense USB: byte %d is %#x", i, v)
		}
	}
	// DualSense over Bluetooth: 0x31, sequence, tag, the same from 3, CRC at 74
	// (the value from Python's zlib.crc32 over 0xA2 and the first 74 bytes).
	b, err = PSRumble(true, true, 78, 1, 200, 100)
	if err != nil {
		t.Fatal(err)
	}
	if b[0] != 0x31 || b[1] != 0x10 || b[2] != 0x10 || b[3] != 0x03 || b[5] != 100 || b[6] != 200 || b[41] != 0x04 {
		t.Fatalf("DualSense BT: % x", b[:8])
	}
	if crc := binary.LittleEndian.Uint32(b[74:]); crc != 0xa75d7fcf {
		t.Fatalf("DualSense BT CRC %#x", crc)
	}
	// Windows may ask for a longer buffer over Bluetooth: the CRC stays at 74.
	if long, _ := PSRumble(true, true, 547, 1, 200, 100); len(long) != 547 || binary.LittleEndian.Uint32(long[74:]) != 0xa75d7fcf {
		t.Fatalf("DualSense BT, long buffer")
	}
	// DualShock 4 on USB.
	b, _ = PSRumble(false, false, 32, 0, 200, 100)
	if b[0] != 0x05 || b[1] != 0x01 || b[4] != 100 || b[5] != 200 {
		t.Fatalf("DualShock 4 USB: % x", b[:8])
	}
	// DualShock 4 over Bluetooth.
	b, _ = PSRumble(false, true, 78, 0, 200, 100)
	if b[0] != 0x11 || b[1] != 0xC0 || b[3] != 0x01 || b[6] != 100 || b[7] != 200 {
		t.Fatalf("DualShock 4 BT: % x", b[:8])
	}
	if crc := binary.LittleEndian.Uint32(b[74:]); crc != 0x6c291dc7 {
		t.Fatalf("DualShock 4 BT CRC %#x", crc)
	}
	// Stopping is the same report with the motors at 0.
	if b, _ = PSRumble(true, false, 48, 0, 0, 0); b[3] != 0 || b[4] != 0 || b[1] != 0x03 {
		t.Fatalf("stop: % x", b[:6])
	}
	// Too short for the layout: refused.
	for _, c := range []struct {
		ds, bt bool
		size   int
	}{{true, false, 47}, {true, true, 64}, {false, false, 20}, {false, true, 77}} {
		if _, err := PSRumble(c.ds, c.bt, c.size, 0, 1, 1); err == nil {
			t.Fatalf("size %d for %v/%v taken", c.size, c.ds, c.bt)
		}
	}
}

func TestSetPadName(t *testing.T) {
	id := `BTHENUM\{00001124-0000-1000-8000-00805F9B34FB}_VID&0002054C_PID&0CE6\7&1&0&A0AB51C4C2A6_C00000000`
	names, err := SetPadName(nil, id, "  Mihai's   pad ")
	if err != nil {
		t.Fatal(err)
	}
	if names[strings.ToLower(id)] != "Mihai's pad" {
		t.Fatalf("names: %v", names)
	}
	if _, err := SetPadName(names, id, strings.Repeat("x", PadNameMax+1)); err == nil {
		t.Fatal("a long name taken")
	}
	if _, err := SetPadName(names, id, "a\tb"); err != nil {
		t.Fatalf("a tab is a space: %v", err) // Fields joins it with a space
	}
	if _, err := SetPadName(names, "", "x"); err == nil {
		t.Fatal("no id taken")
	}
	names, err = SetPadName(names, id, "")
	if err != nil || names != nil {
		t.Fatalf("taking the last name away: %v %v", names, err)
	}
	many := map[string]string{}
	for i := 0; i < padNamesMax; i++ {
		many[strings.Repeat("a", i+1)] = "x"
	}
	if _, err := SetPadName(many, "new", "y"); err == nil {
		t.Fatal("one name too many taken")
	}
	if out, err := SetPadName(many, "a", "z"); err != nil || out["a"] != "z" {
		t.Fatalf("renaming at the limit: %v", err)
	}
}

func TestPadsIDAndBuzz(t *testing.T) {
	ds := PadDevice{
		Chain:     []string{`HID\{00001124-0000-1000-8000-00805f9b34fb}_VID&0002054c_PID&0ce6\9&2&0&0000`, `BTHENUM\{00001124-0000-1000-8000-00805f9b34fb}_VID&0002054c_PID&0ce6\8&1&0&A0AB51C4C2A6_C00000000`},
		CompatIDs: []string{`HID_DEVICE_UP:0001_U:0005`},
	}
	xb := PadDevice{Chain: []string{`HID\VID_045E&PID_0B12&IG_00\8&3&0&0000`, `USB\VID_045E&PID_0B12\3032363030`}, XUSB: false}
	pads := Pads([]PadDevice{ds, xb}, []XInputSlot{{Slot: 1}}, nil, false)
	if len(pads) != 2 {
		t.Fatalf("pads: %+v", pads)
	}
	if !strings.HasPrefix(pads[0].ID, `bthenum\`) || !pads[0].Buzz {
		t.Fatalf("DualSense: %+v", pads[0])
	}
	if pads[1].ID != `usb\vid_045e&pid_0b12\3032363030` || !pads[1].Buzz || pads[1].Slot != 1 {
		t.Fatalf("Xbox: %+v", pads[1])
	}
	// Two Xbox-style pads: Windows doesn't say whose slot is whose, so neither buzzes.
	xb2 := PadDevice{Chain: []string{`HID\VID_045E&PID_0B12&IG_00\8&4&0&0000`, `USB\VID_045E&PID_0B12\99`}}
	for _, p := range Pads([]PadDevice{xb, xb2}, []XInputSlot{{Slot: 1}, {Slot: 2}}, nil, false) {
		if p.Buzz {
			t.Fatalf("an Xbox pad with no known slot buzzes: %+v", p)
		}
	}
}
