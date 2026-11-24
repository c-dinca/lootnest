package plat

import (
	"encoding/binary"
	"os"
	"testing"
	"time"

	"lootnest/internal/core"
)

func testPad(vendor uint16) *evPad {
	return &evPad{vendor: vendor, product: 0x0ce6, bus: busUSB, name: "Test pad", uniq: "aa:bb:cc:dd:ee:ff",
		spec: vendor == 0x054c || vendor == 0x057e,
		abs: map[uint16]absInfo{
			absX: {Min: 0, Max: 255}, absY: {Min: 0, Max: 255},
			absRX: {Min: -32768, Max: 32767}, absRY: {Min: -32768, Max: 32767},
			absHatX: {Min: -1, Max: 1}, absHatY: {Min: -1, Max: 1},
		},
		keys: map[uint16]bool{},
		axes: map[uint16]int32{absX: 128, absY: 128},
	}
}

func TestPadState(t *testing.T) {
	sony := testPad(0x054c)
	sony.event(evKey, btnSouth, 1)
	sony.event(evKey, btnNorth, 1) // the triangle: XInput's Y
	sony.event(evKey, btnMode, 1)
	sony.event(evAbs, absHatY, -1)
	sony.event(evAbs, absY, 0) // pushed up
	st := sony.state()
	if want := uint16(XA | XY | xGuide | XUp); st.Buttons != want {
		t.Fatalf("buttons %#x, want %#x", st.Buttons, want)
	}
	if st.LY < 32000 || st.LX < -300 || st.LX > 300 {
		t.Fatalf("left stick %d,%d: want centred across and up", st.LX, st.LY)
	}

	xbox := testPad(0x045e)
	xbox.event(evKey, btnNorth, 1) // xpad's X sends BTN_X
	xbox.event(evAbs, absRX, 32767)
	if st := xbox.state(); st.Buttons != XX || st.RX != 32767 {
		t.Fatalf("xpad: buttons %#x RX %d", st.Buttons, st.RX)
	}
	xbox.event(evKey, btnNorth, 0)
	if st := xbox.state(); st.Buttons != 0 {
		t.Fatalf("let go: buttons %#x", st.Buttons)
	}
}

func TestPadsAsOne(t *testing.T) {
	padsOnce.Do(func() {}) // no scanning of the test machine's devices
	padsMu.Lock()
	old := padsOpen
	a, b := testPad(0x054c), testPad(0x045e)
	b.bus, b.uniq, b.phys = busBluetooth, "", "usb-0000:00:14.0-2/input0"
	a.event(evKey, btnStart, 1)
	b.event(evKey, btnMode, 1)
	b.event(evAbs, absX, 0)
	padsOpen = map[string]*evPad{"/dev/input/event5": a, "/dev/input/event9": b}
	padsMu.Unlock()
	defer func() { padsMu.Lock(); padsOpen = old; padsMu.Unlock() }()

	x, ok := XInputRead()
	if !ok || x.Buttons != xStart|xGuide || x.LX != -32767 {
		t.Fatalf("XInputRead = %+v, %v", x, ok)
	}
	if !XInputGuideDown() {
		t.Fatal("the Home button isn't down")
	}

	var devs []core.PadDevice
	for _, d := range PadDevices() {
		devs = append(devs, core.PadDevice(d))
	}
	pads := core.Pads(devs, nil, nil, false)
	if len(pads) != 2 {
		t.Fatalf("pads %+v", pads)
	}
	for _, p := range pads {
		switch p.VID {
		case 0x054c:
			if p.Family != "playstation" || p.Bus != "usb" || p.ID == "" {
				t.Errorf("DualSense row %+v", p)
			}
		case 0x045e:
			if p.Bus != "bluetooth" {
				t.Errorf("Xbox row %+v", p)
			}
		default:
			t.Errorf("row %+v", p)
		}
	}
}

// read follows a device's events: here a pipe stands in for the device,
// and its closing for the pad being unplugged.
func TestPadRead(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	p := testPad(0x054c)
	p.path, p.f = "/dev/input/event-test", r
	padsMu.Lock()
	padsOpen[p.path] = p
	padsMu.Unlock()
	done := make(chan struct{})
	go func() { p.read(); close(done) }()

	ev := func(typ, code uint16, value int32) []byte {
		b := make([]byte, 24)
		binary.LittleEndian.PutUint16(b[16:], typ)
		binary.LittleEndian.PutUint16(b[18:], code)
		binary.LittleEndian.PutUint32(b[20:], uint32(value))
		return b
	}
	w.Write(append(append(ev(evKey, btnEast, 1), ev(evAbs, absHatX, 1)...), ev(evSyn, 0, 0)...))
	deadline := time.Now().Add(2 * time.Second)
	for {
		padsMu.Lock()
		b := p.state().Buttons
		padsMu.Unlock()
		if b == XB|XRight {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("buttons %#x after the events", b)
		}
		time.Sleep(10 * time.Millisecond)
	}
	w.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("read didn't end when the device went")
	}
	padsMu.Lock()
	_, still := padsOpen[p.path]
	padsMu.Unlock()
	if still {
		t.Fatal("an unplugged pad is still listed")
	}
}

// A device that isn't a pad (here a plain file, which answers no ioctl) is
// left out and not opened again until a new one takes its name.
func TestPadsScanSkips(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(dir+"/event0", nil, 0o644)
	old := padsDir
	padsDir = dir
	defer func() { padsDir = old }()
	padsScan()
	padsMu.Lock()
	_, open := padsOpen[dir+"/event0"]
	_, tried := padsTried[dir+"/event0"]
	padsMu.Unlock()
	if open || !tried {
		t.Fatalf("open %v, tried %v", open, tried)
	}
}
