package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestModelOf(t *testing.T) {
	cases := []struct {
		vid, pid     uint16
		name, family string
	}{
		{0x045E, 0x028E, "Xbox 360 controller", "xbox"},
		{0x045E, 0x02EA, "Xbox One S controller", "xbox"},
		{0x045E, 0x0B12, "Xbox Series controller", "xbox"},
		{0x045E, 0x0B00, "Xbox Elite Series 2 controller", "xbox"},
		{0x045E, 0x0B0A, "Xbox Adaptive Controller", "xbox"},
		{0x045E, 0x1234, "", "xbox"},
		{0x054C, 0x05C4, "DualShock 4 (first model)", "playstation"},
		{0x054C, 0x09CC, "DualShock 4 (second model)", "playstation"},
		{0x054C, 0x0CE6, "DualSense", "playstation"},
		{0x054C, 0x0DF2, "DualSense Edge", "playstation"},
		{0x057E, 0x2009, "Switch Pro Controller", "nintendo"},
		{0x057E, 0x2006, "Joy-Con (L)", "nintendo"},
		{0x057E, 0x2007, "Joy-Con (R)", "nintendo"},
		{0x057E, 0x200E, "Joy-Con pair", "nintendo"},
		{0x2DC8, 0x6101, "8BitDo controller", "generic"},
		{0x046D, 0xC21D, "Logitech F310", "xbox"},
		{0x046D, 0xC219, "Logitech F710 (DirectInput mode)", "generic"},
		{0x28DE, 0x1102, "Steam Controller", "xbox"},
		{0x28DE, 0x1205, "Steam Deck controls", "xbox"},
		{0x0079, 0x0006, "", "generic"},
	}
	for _, c := range cases {
		if n, f := modelOf(c.vid, c.pid); n != c.name || f != c.family {
			t.Errorf("modelOf(%04x, %04x) = %q, %q; want %q, %q", c.vid, c.pid, n, f, c.name, c.family)
		}
	}
}

func TestParseVIDPID(t *testing.T) {
	for id, want := range map[string][2]uint16{
		`HID\VID_054C&PID_0CE6&REV_0100&MI_03`:                                        {0x054C, 0x0CE6},
		`HID\{00001124-0000-1000-8000-00805f9b34fb}_VID&0002054c_PID&0ce6`:            {0x054C, 0x0CE6},
		`USB\VID_045E&PID_0B12\3039363431313730`:                                      {0x045E, 0x0B12},
		`HID\{00001812-0000-1000-8000-00805f9b34fb}_Dev_VID&02045e_PID&0b13_REV&0509`: {0x045E, 0x0B13},
	} {
		if v, p, ok := parseVIDPID(id); !ok || v != want[0] || p != want[1] {
			t.Errorf("parseVIDPID(%s) = %04x %04x %v", id, v, p, ok)
		}
	}
	if _, _, ok := parseVIDPID(`ROOT\SYSTEM\0001`); ok {
		t.Error("no ids in a root device")
	}
}

func TestPads(t *testing.T) {
	devs := []PadDevice{
		// A DualSense over Bluetooth.
		{Chain: []string{`HID\{00001124-0000-1000-8000-00805f9b34fb}_VID&0002054c_PID&0ce6\9&1&0000`, `BTHENUM\{00001124-0000-1000-8000-00805f9b34fb}_VID&0002054c_PID&0ce6\8&1`, `BTH\MS_BTHBRB\7&1`},
			HardwareIDs: []string{`HID\{00001124-0000-1000-8000-00805f9b34fb}_VID&0002054c_PID&0ce6`}, CompatIDs: []string{"HID_DEVICE_SYSTEM_GAME", "HID_DEVICE"}, Name: "Wireless Controller"},
		// A keyboard: left out.
		{Chain: []string{`HID\VID_046D&PID_C31C&MI_00\8&1`, `USB\VID_046D&PID_C31C&MI_00\7&1`, `USB\VID_046D&PID_C31C\6&1`},
			HardwareIDs: []string{`HID\VID_046D&PID_C31C&MI_00`}, CompatIDs: []string{"HID_DEVICE_SYSTEM_KEYBOARD"}},
		// A Steam Deck's two collections, one without a game usage: one pad.
		{Chain: []string{`HID\VID_28DE&PID_1205&MI_00\8&1`, `USB\VID_28DE&PID_1205&MI_00\7&1`, `USB\VID_28DE&PID_1205\6&1`},
			HardwareIDs: []string{`HID\VID_28DE&PID_1205&MI_00`}, CompatIDs: []string{"HID_DEVICE_SYSTEM_MOUSE"}},
		{Chain: []string{`HID\VID_28DE&PID_1205&MI_02\8&2`, `USB\VID_28DE&PID_1205&MI_02\7&2`, `USB\VID_28DE&PID_1205\6&1`},
			HardwareIDs: []string{`HID\VID_28DE&PID_1205&MI_02`}, CompatIDs: []string{"HID_DEVICE_UP:0001_U:0005"}},
		// The Xbox 360 pad ViGEmBus makes for DS4Windows.
		{Chain: []string{`USB\VID_045E&PID_028E\1&2d595ca7&0&01`, `ROOT\SYSTEM\0001`}, HardwareIDs: []string{`USB\VID_045E&PID_028E&REV_0114`}, XUSB: true},
		// An Xbox Series pad over Bluetooth LE.
		{Chain: []string{`HID\{00001812-0000-1000-8000-00805f9b34fb}_Dev_VID&02045e_PID&0b13_REV&0509_Col01\a&1`, `BTHLEDEVICE\{00001812-0000-1000-8000-00805f9b34fb}_Dev_VID&02045e_PID&0b13_REV&0509_c8e2\9&1`},
			HardwareIDs: []string{`HID\{00001812-0000-1000-8000-00805f9b34fb}_Dev_VID&02045e_PID&0b13_REV&0509_Col01`}, CompatIDs: []string{"HID_DEVICE_SYSTEM_GAME"}, Name: "Xbox Wireless Controller"},
		// A pad Windows can't name.
		{Chain: []string{`HID\VID_0079&PID_0006\7&1`, `USB\VID_0079&PID_0006\6&1`}, HardwareIDs: []string{`HID\VID_0079&PID_0006`}, CompatIDs: []string{"HID_DEVICE_UP:0001_U:0004"}, Name: " USB Gamepad "},
	}
	tools := FindControllerTools(map[string]bool{"ds4windows": true}, nil, map[string]bool{"ViGEmBus": true})
	got := Pads(devs, []XInputSlot{{Slot: 2, Kind: "gamepad"}}, tools, true)
	want := []Pad{
		{Name: "DualSense", Family: "playstation", VID: 0x054C, PID: 0x0CE6, Bus: "bluetooth"},
		{Name: "Steam Deck controls", Family: "xbox", VID: 0x28DE, PID: 0x1205, Bus: "usb"},
		{Name: "Xbox 360 controller", Family: "xbox", VID: 0x045E, PID: 0x028E, XInput: true, Slot: 2, Virtual: "vigem", MadeBy: "DS4Windows"},
		{Name: "Xbox Series controller", Family: "xbox", VID: 0x045E, PID: 0x0B13, Bus: "bluetooth"},
		{Name: "USB Gamepad", Family: "generic", VID: 0x0079, PID: 0x0006, Bus: "usb"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Pads =\n%+v\nwant\n%+v", got, want)
	}
	// A real Xbox 360 pad: a USB hub above it, not the root.
	real := Pads([]PadDevice{{Chain: []string{`USB\VID_045E&PID_028E\1`, `USB\ROOT_HUB30\4&1`}, XUSB: true}}, nil, nil, true)
	if real[0].Virtual != "" || real[0].Bus != "usb" {
		t.Fatalf("real pad = %+v", real[0])
	}
	// An Xbox-style pad from a maker the tables don't know.
	pdp := Pads([]PadDevice{{Chain: []string{`USB\VID_0E6F&PID_02A8\1`, `USB\ROOT_HUB30\4&1`}, XUSB: true}}, nil, nil, false)
	if pdp[0].Family != "xbox" {
		t.Fatalf("PDP pad = %+v", pdp[0])
	}
	vjoy := Pads([]PadDevice{{Chain: []string{`HID\HIDCLASS&COL01\1&2`, `ROOT\HIDCLASS\0000`}, HardwareIDs: []string{`HID\VID_1234&PID_BEAD&REV_0222&COL01`}}}, nil, nil, false)
	if vjoy[0].Virtual != "vjoy" || vjoy[0].MadeBy != "vJoy" || vjoy[0].Bus != "" {
		t.Fatalf("vJoy = %+v", vjoy[0])
	}
	if XInputKind(1) != "gamepad" || XInputKind(2) != "wheel" || XInputKind(99) != "controller" {
		t.Fatal("XInputKind")
	}
}

func TestControllerTools(t *testing.T) {
	appData, services := ControllerProbes()
	if !reflect.DeepEqual(appData, []string{"DS4Windows"}) || !reflect.DeepEqual(services, []string{"HidHide", "ViGEmBus"}) {
		t.Fatalf("probes = %v %v", appData, services)
	}
	names := func(ts []ControllerTool) (out []string) {
		for _, t := range ts {
			s := t.Name
			if t.Running {
				s += " running"
			}
			out = append(out, s)
		}
		return out
	}
	// Installed only by its folder; a service installed and stopped.
	ts := FindControllerTools(map[string]bool{"explorer": true}, map[string]bool{"DS4Windows": true}, map[string]bool{"ViGEmBus": false})
	if got := names(ts); !reflect.DeepEqual(got, []string{"DS4Windows", "ViGEmBus"}) {
		t.Fatalf("tools = %v", got)
	}
	if n := ControllerNotes(ts); len(n) != 0 {
		t.Fatalf("nothing runs, but notes = %v", n)
	}

	has := func(notes []string, sub string) bool {
		for _, n := range notes {
			if strings.Contains(n, sub) {
				return true
			}
		}
		return false
	}
	cases := []struct {
		procs    map[string]bool
		services map[string]bool
		want     []string // parts of the notes, in order
	}{
		// Steam alone: nothing to say.
		{map[string]bool{"steam": true}, nil, nil},
		// Steam and DS4Windows, HidHide installed.
		{map[string]bool{"steam": true, "ds4windows": true}, map[string]bool{"HidHide": true, "ViGEmBus": true},
			[]string{"Steam Input and DS4Windows both translating can make games see double input. For a game Steam starts"}},
		// DS4Windows without HidHide: two pads.
		{map[string]bool{"ds4windows": true}, map[string]bool{"ViGEmBus": true},
			[]string{"the copy DS4Windows makes, unless HidHide hides your own"}},
		// Three translating, no Steam: "all", and no Steam advice.
		{map[string]bool{"dsx": true, "rewasd": true, "x360ce": true}, map[string]bool{"HidHide": false},
			[]string{"DSX, reWASD and x360ce all translating"}},
		// Two that make a copy, without HidHide: one note naming both.
		{map[string]bool{"ds4windows": true, "x360ce": true}, map[string]bool{"ViGEmBus": true},
			[]string{"DS4Windows and x360ce both translating", "the copies DS4Windows and x360ce make"}},
		// A key mapper with a pad translator.
		{map[string]bool{"joytokey": true, "rewasdservice": true}, nil,
			[]string{"While JoyToKey runs"}},
	}
	for i, c := range cases {
		notes := ControllerNotes(FindControllerTools(c.procs, nil, c.services))
		if len(notes) != len(c.want) {
			t.Fatalf("case %d: notes = %q", i, notes)
		}
		for j, w := range c.want {
			if !strings.Contains(notes[j], w) {
				t.Errorf("case %d: note %d = %q, want %q", i, j, notes[j], w)
			}
		}
		if i == 3 && has(notes, "Steam") {
			t.Errorf("case 3: Steam advice without Steam: %q", notes)
		}
	}
	if padMaker("vigem", nil) != "an app that uses ViGEmBus" || padMaker("steam", nil) != "Steam Input" || padMaker("root", nil) != "" {
		t.Fatal("padMaker")
	}
}
