package plat

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Controllers on Linux, read from the kernel's event devices
// (/dev/input/event*), the way SDL and WebKitGTK's libmanette read them.
// Every pad answers XInputRead as one, as XInput's do on Windows; its Home
// button (the Xbox or PS button, BTN_MODE) is XInputGuideDown. The page
// reads pads itself through the Gamepad API while the window has the
// keyboard; this is for the Home button over a game and for the page when
// it can't (padfeed.go). Nothing is ever written to a pad, and none is
// grabbed: games read them as before.
//
// systemd gives the person at the PC read access to game controllers
// (uaccess); a pad Lootnest can't open is left out.

const (
	evSyn = 0x00
	evKey = 0x01
	evAbs = 0x03

	synDropped = 3

	btnSouth  = 0x130
	btnEast   = 0x131
	btnNorth  = 0x133 // BTN_X: xpad's X, the spec's top button
	btnWest   = 0x134 // BTN_Y: xpad's Y, the spec's left button
	btnTL     = 0x136
	btnTR     = 0x137
	btnSelect = 0x13a
	btnStart  = 0x13b
	btnMode   = 0x13c
	btnThumbL = 0x13d
	btnThumbR = 0x13e
	btnDUp    = 0x220
	btnDDown  = 0x221
	btnDLeft  = 0x222
	btnDRight = 0x223
	keyMax    = 0x2ff

	absX    = 0x00
	absY    = 0x01
	absRX   = 0x03
	absRY   = 0x04
	absHatX = 0x10
	absHatY = 0x11
	absMax  = 0x3f

	busUSB       = 0x03
	busBluetooth = 0x05
	busVirtual   = 0x06
)

// XInput's flags for the buttons that have no name in plat's constants.
const (
	xStart  = 0x0010
	xBack   = 0x0020
	xThumbL = 0x0040
	xThumbR = 0x0080
	xLB     = 0x0100
	xRB     = 0x0200
	xGuide  = 0x0400
)

// ioctl numbers from linux/input.h: _IOC(read, 'E', nr, size).
func eviocRead(nr, size uintptr) uintptr { return 2<<30 | size<<16 | 'E'<<8 | nr }

func ioctl(fd int, req uintptr, p unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(p)); e != 0 {
		return e
	}
	return nil
}

func evBits(fd int, ev uintptr, n int) []byte {
	b := make([]byte, (n+8)/8)
	if ioctl(fd, eviocRead(0x20+ev, uintptr(len(b))), unsafe.Pointer(&b[0])) != nil {
		return nil
	}
	return b
}

func evString(fd int, nr uintptr) string {
	b := make([]byte, 256)
	if ioctl(fd, eviocRead(nr, uintptr(len(b))), unsafe.Pointer(&b[0])) != nil {
		return ""
	}
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimSpace(string(b))
}

func bitOn(b []byte, n int) bool { return n/8 < len(b) && b[n/8]&(1<<(n%8)) != 0 }

// absInfo is struct input_absinfo.
type absInfo struct{ Value, Min, Max, Fuzz, Flat, Res int32 }

// evPad is one controller's event device.
type evPad struct {
	path                 string
	f                    *os.File
	bus, vendor, product uint16
	name, uniq, phys     string
	spec                 bool // its face buttons follow the kernel's layout (north is the top one)
	abs                  map[uint16]absInfo
	keys                 map[uint16]bool
	axes                 map[uint16]int32
}

// isPad says whether an event device is a controller: it has the south
// face button (BTN_GAMEPAD) and a left stick or a D-pad. A DualSense's
// motion sensors and touchpad are devices of their own and have neither.
func isPad(fd int) bool {
	keys, abs := evBits(fd, evKey, keyMax), evBits(fd, evAbs, absMax)
	return bitOn(keys, btnSouth) && (bitOn(abs, absX) || bitOn(abs, absHatX) || bitOn(keys, btnDUp))
}

func openPad(path string) (*evPad, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	var p *evPad
	withFd(f, func(fd int) {
		if !isPad(fd) {
			return
		}
		var id [4]uint16 // struct input_id: bustype, vendor, product, version
		ioctl(fd, eviocRead(0x02, 8), unsafe.Pointer(&id[0]))
		p = &evPad{path: path, f: f, bus: id[0], vendor: id[1], product: id[2],
			name: evString(fd, 0x06), phys: evString(fd, 0x07), uniq: evString(fd, 0x08)}
		// Sony's and Nintendo's kernel drivers follow the kernel's layout;
		// xpad and most others send BTN_X for the left button.
		p.spec = p.vendor == 0x054c || p.vendor == 0x057e
		p.sync(fd)
	})
	if p == nil {
		f.Close()
	}
	return p, nil
}

// withFd runs f with the file's descriptor without taking it out of Go's
// poller (os.File.Fd would make reads block a thread).
func withFd(file *os.File, f func(fd int)) {
	if c, err := file.SyscallConn(); err == nil {
		c.Control(func(fd uintptr) { f(int(fd)) })
	}
}

// sync reads the pad's whole state: at the start, and after the kernel
// dropped events (SYN_DROPPED).
func (p *evPad) sync(fd int) {
	have := evBits(fd, evAbs, absMax)
	p.abs, p.axes = map[uint16]absInfo{}, map[uint16]int32{}
	for _, c := range []uint16{absX, absY, absRX, absRY, absHatX, absHatY} {
		if !bitOn(have, int(c)) {
			continue
		}
		var ai absInfo
		if ioctl(fd, eviocRead(0x40+uintptr(c), unsafe.Sizeof(ai)), unsafe.Pointer(&ai)) == nil {
			p.abs[c] = ai
			p.axes[c] = ai.Value
		}
	}
	p.keys = map[uint16]bool{}
	down := make([]byte, (keyMax+8)/8)
	if ioctl(fd, eviocRead(0x18, uintptr(len(down))), unsafe.Pointer(&down[0])) == nil {
		for k := 0; k <= keyMax; k++ {
			if bitOn(down, k) {
				p.keys[uint16(k)] = true
			}
		}
	}
}

// stick turns an axis into XInput's range, up positive when flip.
func (p *evPad) stick(code uint16, flip bool) int16 {
	ai, ok := p.abs[code]
	if !ok || ai.Max <= ai.Min {
		return 0
	}
	v := (float64(p.axes[code]) - float64(ai.Min+ai.Max)/2) / (float64(ai.Max-ai.Min) / 2)
	if flip {
		v = -v
	}
	v = max(-1, min(1, v))
	return int16(v * 32767)
}

// state is the pad's buttons as XInput's flags and its sticks.
func (p *evPad) state() XPad {
	var b uint16
	set := func(on bool, flag uint16) {
		if on {
			b |= flag
		}
	}
	k := p.keys
	set(k[btnSouth], XA)
	set(k[btnEast], XB)
	if p.spec {
		set(k[btnNorth], XY)
		set(k[btnWest], XX)
	} else {
		set(k[btnNorth], XX)
		set(k[btnWest], XY)
	}
	set(k[btnTL], xLB)
	set(k[btnTR], xRB)
	set(k[btnSelect], xBack)
	set(k[btnStart], xStart)
	set(k[btnMode], xGuide)
	set(k[btnThumbL], xThumbL)
	set(k[btnThumbR], xThumbR)
	hx, hy := p.axes[absHatX], p.axes[absHatY]
	set(k[btnDUp] || hy < 0, XUp)
	set(k[btnDDown] || hy > 0, XDown)
	set(k[btnDLeft] || hx < 0, XLeft)
	set(k[btnDRight] || hx > 0, XRight)
	return XPad{Buttons: b, LX: p.stick(absX, false), LY: p.stick(absY, true), RX: p.stick(absRX, false), RY: p.stick(absRY, true)}
}

// event applies one input_event.
func (p *evPad) event(typ, code uint16, value int32) {
	switch typ {
	case evKey:
		p.keys[code] = value != 0
	case evAbs:
		p.axes[code] = value
	}
}

var (
	padsMu   sync.Mutex
	padsOpen = map[string]*evPad{}
	padsOnce sync.Once
	padsDir  = "/dev/input"
)

// padsStart opens the pads there are now and keeps looking for new ones
// every 2 seconds.
func padsStart() {
	padsOnce.Do(func() {
		padsScan()
		go func() {
			for range time.Tick(2 * time.Second) {
				padsScan()
			}
		}()
	})
}

// padsTried holds the devices that opened but aren't pads, by when they
// were looked at. One that couldn't be opened is tried again at each scan:
// its permission may come a moment after it appears.
var padsTried = map[string]time.Time{}

func padsScan() {
	paths, _ := filepath.Glob(filepath.Join(padsDir, "event*"))
	for _, path := range paths {
		padsMu.Lock()
		_, open := padsOpen[path]
		tried, was := padsTried[path]
		padsMu.Unlock()
		if open {
			continue
		}
		if fi, err := os.Stat(path); was && (err != nil || !fi.ModTime().After(tried)) {
			continue // the same device as before: a new one is a new node
		}
		p, err := openPad(path)
		padsMu.Lock()
		if err == nil && p == nil {
			padsTried[path] = time.Now()
		} else if p != nil {
			delete(padsTried, path)
			padsOpen[path] = p
		}
		padsMu.Unlock()
		if p != nil {
			go p.read()
		}
	}
}

// read follows the pad's events until it's unplugged.
func (p *evPad) read() {
	const size = 24 // struct input_event on 64-bit Linux: time 16, type 2, code 2, value 4
	buf := make([]byte, size*64)
	for {
		n, err := p.f.Read(buf)
		if err != nil {
			padsMu.Lock()
			delete(padsOpen, p.path)
			padsMu.Unlock()
			p.f.Close()
			return
		}
		padsMu.Lock()
		for i := 0; i+size <= n; i += size {
			e := buf[i : i+size]
			typ, code := binary.LittleEndian.Uint16(e[16:]), binary.LittleEndian.Uint16(e[18:])
			value := int32(binary.LittleEndian.Uint32(e[20:]))
			if typ == evSyn && code == synDropped {
				withFd(p.f, p.sync)
				continue
			}
			p.event(typ, code, value)
		}
		padsMu.Unlock()
	}
}

// XInputRead reads every controller as one: any button any of them holds,
// and the stick furthest over. False when there's none.
func XInputRead() (XPad, bool) {
	padsStart()
	padsMu.Lock()
	defer padsMu.Unlock()
	var out XPad
	far := func(a, b int16) int16 {
		if abs16(b) > abs16(a) {
			return b
		}
		return a
	}
	for _, p := range padsOpen {
		s := p.state()
		out.Buttons |= s.Buttons
		out.LX, out.LY = far(out.LX, s.LX), far(out.LY, s.LY)
		out.RX, out.RY = far(out.RX, s.RX), far(out.RY, s.RY)
	}
	return out, len(padsOpen) > 0
}

func abs16(v int16) int32 {
	if v < 0 {
		return -int32(v)
	}
	return int32(v)
}

// XInputGuideDown tells whether any controller holds its Home button.
func XInputGuideDown() bool {
	x, _ := XInputRead()
	return x.Buttons&xGuide != 0
}

// PadDevices lists the controllers, with ids shaped as Windows' so that
// core.Pads names them the same way: a USB, Bluetooth or virtual (ROOT)
// device carrying the vendor and product, under it a HID one.
func PadDevices() []PadDevice {
	padsStart()
	padsMu.Lock()
	defer padsMu.Unlock()
	out := []PadDevice{}
	for _, p := range padsOpen {
		vp := fmt.Sprintf("VID_%04X&PID_%04X", p.vendor, p.product)
		where := p.uniq
		if where == "" {
			where = p.phys
		}
		if where == "" {
			where = filepath.Base(p.path)
		}
		where = strings.ToUpper(strings.NewReplacer(":", "", "/", "-", "\\", "-").Replace(where))
		bus := "HID"
		switch p.bus {
		case busUSB:
			bus = "USB"
		case busBluetooth:
			bus = "BTHENUM"
		case busVirtual:
			bus = "ROOT"
		}
		out = append(out, PadDevice{
			Chain:       []string{`HID\` + vp + `\` + where, bus + `\` + vp + `\` + where},
			HardwareIDs: []string{`HID\` + vp},
			CompatIDs:   []string{"HID_DEVICE_SYSTEM_GAME"},
			Name:        p.name,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Chain[1] < out[j].Chain[1] })
	return out
}
