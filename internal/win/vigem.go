//go:build windows

package win

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

// A virtual Xbox 360 pad through ViGEmBus, the driver DS4Windows and
// streaming hosts install (Lootnest ships no driver: it only talks to one
// that's already there). The bus is a device with one interface; a pad is
// plugged into it with an IOCTL, fed reports with another, and unplugged
// when the handle that plugged it closes, so a Lootnest that ends, however
// it ends, leaves no pad behind.

// GUID_DEVINTERFACE_BUSENUM_VIGEM.
var vigemGUID = windows.GUID{Data1: 0x96E42B22, Data2: 0xF5E9, Data3: 0x42F8, Data4: [8]byte{0xB0, 0x43, 0xED, 0x0F, 0x93, 0x2F, 0x01, 0x4F}}

// ViGEmBus's control codes: CTL_CODE(FILE_DEVICE_BUS_EXTENDER, 0x801 + n,
// METHOD_BUFFERED, FILE_WRITE_DATA).
const (
	ioctlVigemPlug    = 0x2AA004 // n = 0x000, VIGEM_PLUGIN_TARGET
	ioctlVigemUnplug  = 0x2AA008 // n = 0x001, VIGEM_UNPLUG_TARGET
	ioctlVigemVersion = 0x2AA00C // n = 0x002, VIGEM_CHECK_VERSION
	ioctlVigemReady   = 0x2AA010 // n = 0x003, VIGEM_WAIT_DEVICE_READY (ViGEmBus 1.17 and later)
	ioctlXusbSubmit   = 0x2AA808 // n = 0x201, XUSB_SUBMIT_REPORT

	vigemVersion    = 0x0001 // VIGEM_COMMON_VERSION
	vigemXbox360    = 0      // VIGEM_TARGET_TYPE Xbox360Wired
	vigemMaxSerials = 32
	cmPresent       = 0 // CM_GET_DEVICE_INTERFACE_LIST_PRESENT
)

// vigemPaths are the bus's interfaces present on this PC.
func vigemPaths() []string {
	l, err := windows.CM_Get_Device_Interface_List("", &vigemGUID, cmPresent)
	if err != nil {
		return nil
	}
	return l
}

// ViGEmPresent says whether the ViGEmBus driver is on this PC and running.
func ViGEmPresent() bool { return len(vigemPaths()) > 0 }

// ViGEmPad is one virtual Xbox 360 pad.
type ViGEmPad struct {
	bus    windows.Handle
	serial uint32
}

func ioctl(h windows.Handle, code uint32, in []byte) error {
	var n uint32
	return windows.DeviceIoControl(h, code, &in[0], uint32(len(in)), nil, 0, &n, nil)
}

// NewViGEmPad plugs a virtual Xbox 360 pad in. Games see it as any Xbox
// pad, in the first free XInput slot.
func NewViGEmPad() (*ViGEmPad, error) {
	paths := vigemPaths()
	if len(paths) == 0 {
		return nil, errors.New("the ViGEmBus driver isn't on this PC")
	}
	var last error
	for _, path := range paths {
		p16, err := windows.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		h, err := windows.CreateFile(p16, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
		if err != nil {
			last = fmt.Errorf("Windows wouldn't open ViGEmBus: %w", err)
			continue
		}
		// The version check is ViGEmBus's own handshake; a bus that
		// refuses it may still take a pad, so it isn't the last word.
		ver := make([]byte, 8)
		binary.LittleEndian.PutUint32(ver[0:], 8)
		binary.LittleEndian.PutUint32(ver[4:], vigemVersion)
		verErr := ioctl(h, ioctlVigemVersion, ver)
		for serial := uint32(1); serial <= vigemMaxSerials; serial++ {
			plug := make([]byte, 16)
			binary.LittleEndian.PutUint32(plug[0:], 16)
			binary.LittleEndian.PutUint32(plug[4:], serial)
			binary.LittleEndian.PutUint32(plug[8:], vigemXbox360)
			binary.LittleEndian.PutUint16(plug[12:], 0x045E) // Microsoft
			binary.LittleEndian.PutUint16(plug[14:], 0x028E) // Xbox 360 Controller
			if err := ioctl(h, ioctlVigemPlug, plug); err == nil {
				// As ViGEmClient does: wait until Windows has the new pad
				// set up, else the first reports are refused as busy. An
				// older bus doesn't know the call; Submit tries again then.
				ready := make([]byte, 8)
				binary.LittleEndian.PutUint32(ready[0:], 8)
				binary.LittleEndian.PutUint32(ready[4:], serial)
				_ = ioctl(h, ioctlVigemReady, ready)
				return &ViGEmPad{bus: h, serial: serial}, nil
			} else {
				last = fmt.Errorf("ViGEmBus wouldn't add a controller: %w", err)
			}
		}
		if verErr != nil {
			last = fmt.Errorf("%w (and it didn't accept Lootnest's version: %v)", last, verErr)
		}
		windows.CloseHandle(h)
	}
	return nil, last
}

// Submit sets the pad's state from an XUSB_REPORT (core.PadState.XUSBReport).
func (p *ViGEmPad) Submit(report [12]byte) error {
	buf := make([]byte, 20)
	binary.LittleEndian.PutUint32(buf[0:], 20)
	binary.LittleEndian.PutUint32(buf[4:], p.serial)
	copy(buf[8:], report[:])
	var err error
	for try := 0; try < 10; try++ {
		// A pad just plugged in answers ERROR_BUSY until it's ready.
		if err = ioctl(p.bus, ioctlXusbSubmit, buf); !errors.Is(err, windows.ERROR_BUSY) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

// Close unplugs the pad.
func (p *ViGEmPad) Close() {
	if p == nil || p.bus == 0 {
		return
	}
	un := make([]byte, 8)
	binary.LittleEndian.PutUint32(un[0:], 8)
	binary.LittleEndian.PutUint32(un[4:], p.serial)
	_ = ioctl(p.bus, ioctlVigemUnplug, un)
	windows.CloseHandle(p.bus)
	p.bus = 0
}
