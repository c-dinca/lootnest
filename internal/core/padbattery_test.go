package core

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func psReportWith(id byte, at int, status byte) []byte {
	r := make([]byte, 78)
	r[0] = id
	r[at] = status
	return r
}

func TestPSBattery(t *testing.T) {
	for _, c := range []struct {
		name   string
		ds, bt bool
		r      []byte
		ok     bool
		want   PadBattery
	}{
		{"DualSense USB, 75% on battery", true, false, psReportWith(0x01, 53, 0x07), true, PadBattery{Percent: 75}},
		{"DualSense USB, charging at 35%", true, false, psReportWith(0x01, 53, 0x13), true, PadBattery{Percent: 35, Charging: true}},
		{"DualSense USB, full", true, false, psReportWith(0x01, 53, 0x2A), true, PadBattery{Percent: 100, Full: true}},
		{"DualSense BT long, 15%", true, true, psReportWith(0x31, 54, 0x01), true, PadBattery{Percent: 15}},
		{"DualSense BT, 10 caps at 100", true, true, psReportWith(0x31, 54, 0x0A), true, PadBattery{Percent: 100}},
		{"DualSense BT short report: no battery", true, true, psReportWith(0x01, 53, 0x07), false, PadBattery{}},
		{"DualSense, temperature error", true, false, psReportWith(0x01, 53, 0xB5), false, PadBattery{}},
		{"DualShock 4 USB, on the cable, charging", false, false, psReportWith(0x01, 30, 0x15), true, PadBattery{Percent: 55, Charging: true}},
		{"DualShock 4 USB, on the cable, full", false, false, psReportWith(0x01, 30, 0x1B), true, PadBattery{Percent: 100, Full: true}},
		{"DualShock 4 BT long, 25%", false, true, psReportWith(0x11, 32, 0x02), true, PadBattery{Percent: 25}},
		{"DualShock 4 BT short report: no battery", false, true, psReportWith(0x01, 30, 0x02), false, PadBattery{}},
		{"too short", true, false, []byte{0x01, 0, 0}, false, PadBattery{}},
	} {
		got, ok := PSBattery(c.ds, c.bt, c.r)
		if ok != c.ok || got != c.want {
			t.Errorf("%s: %+v %v, want %+v %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestXInputBattery(t *testing.T) {
	if _, ok := XInputBattery(0, 3); ok {
		t.Fatal("a disconnected pad says something")
	}
	if b, ok := XInputBattery(1, 0); !ok || !b.Wired {
		t.Fatalf("wired: %+v", b)
	}
	if b, ok := XInputBattery(3, 1); !ok || b.Level != "low" || b.Percent > BatteryLow {
		t.Fatalf("NiMH low: %+v", b)
	}
	if b, _ := XInputBattery(2, 0); b.Percent > BatteryEmpty {
		t.Fatalf("alkaline empty: %+v", b)
	}
}

func TestBatteryWarn(t *testing.T) {
	at := func(p int) PadBattery { return PadBattery{Percent: p} }
	say, w := BatteryWarn(PadBattery{}, at(45), false, 0)
	if say != "" || w != 0 {
		t.Fatalf("45%%: %q %d", say, w)
	}
	if say, w = BatteryWarn(at(45), at(15), true, w); say != "low" || w != 1 {
		t.Fatalf("15%%: %q %d", say, w)
	}
	if say, w = BatteryWarn(at(15), at(15), true, w); say != "" || w != 1 {
		t.Fatalf("15%% again: %q %d", say, w)
	}
	if say, w = BatteryWarn(at(15), at(5), true, w); say != "empty" || w != 2 {
		t.Fatalf("5%%: %q %d", say, w)
	}
	// Plugged in, then full: one "charged", and the warnings start over.
	if say, w = BatteryWarn(at(5), PadBattery{Percent: 15, Charging: true}, true, w); say != "" || w != 0 {
		t.Fatalf("charging: %q %d", say, w)
	}
	if say, _ = BatteryWarn(PadBattery{Percent: 95, Charging: true}, PadBattery{Percent: 100, Full: true}, true, 0); say != "charged" {
		t.Fatalf("full: %q", say)
	}
	if say, _ = BatteryWarn(PadBattery{Percent: 100, Full: true}, PadBattery{Percent: 100, Full: true}, true, 0); say != "" {
		t.Fatalf("still full: %q", say)
	}
	// The first look at a pad that's already low warns.
	if say, _ = BatteryWarn(PadBattery{}, at(5), false, 0); say != "empty" {
		t.Fatalf("first look at 5%%: %q", say)
	}
	if say, _ = BatteryWarn(PadBattery{}, PadBattery{Percent: -1, Wired: true}, false, 0); say != "" {
		t.Fatalf("wired: %q", say)
	}
}

func TestSaverWanted(t *testing.T) {
	low, mid, chg := PadBattery{Percent: 15}, PadBattery{Percent: 25}, PadBattery{Percent: 15, Charging: true}
	if SaverWanted("", low, true) || !SaverWanted("on", PadBattery{Percent: 90}, false) {
		t.Fatal("off or on")
	}
	if !SaverWanted("low", low, false) || SaverWanted("low", mid, false) {
		t.Fatal("low: from 20%")
	}
	if !SaverWanted("low", mid, true) || SaverWanted("low", PadBattery{Percent: 35}, true) {
		t.Fatal("low: stays on to 30%")
	}
	if SaverWanted("low", chg, true) {
		t.Fatal("low: off while charging")
	}
}

func TestPSSaver(t *testing.T) {
	b, err := PSSaver(true, false, 48, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if b[0] != 0x02 || b[1] != 0 || b[2] != 0x16 || b[3] != 0 || b[4] != 0 || b[10] != 0x04 || b[39] != 0 || b[44] != 0 || b[47] != 0 {
		t.Fatalf("DualSense USB on: % x", b)
	}
	if b, _ = PSSaver(true, false, 48, 0, false); b[2] != 0x06 || b[10] != 0 || b[47] != 0x40 {
		t.Fatalf("DualSense USB off: % x", b)
	}
	b, _ = PSSaver(true, true, 78, 2, true)
	if b[0] != 0x31 || b[4] != 0x16 || b[12] != 0x04 {
		t.Fatalf("DualSense BT on: % x", b[:14])
	}
	h := crc32.NewIEEE()
	h.Write([]byte{0xA2})
	h.Write(b[:74])
	if binary.LittleEndian.Uint32(b[74:]) != h.Sum32() {
		t.Fatal("DualSense BT: the CRC wasn't written again")
	}
	if b, _ = PSSaver(false, false, 32, 0, true); b[0] != 0x05 || b[1] != 0x02 || b[4] != 0 || b[8] != 0 {
		t.Fatalf("DualShock 4 USB on: % x", b[:10])
	}
	if b, _ = PSSaver(false, true, 78, 0, false); b[0] != 0x11 || b[3] != 0x02 || b[10] != 0x40 {
		t.Fatalf("DualShock 4 BT off: % x", b[:12])
	}
}
