//go:build windows

package win

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A short buzz on one controller, so the person naming pads in Settings,
// Controllers can tell which is which (the owner, 23 Nov 2026: vibration,
// not the light or the speaker). Only from that button: a PlayStation pad
// is opened and sent one output report to start its motors and one to
// stop them (core.PSRumble); an Xbox pad goes through XInput. A Bluetooth
// DualSense or DualShock 4 switches to its full reports once it's sent
// one, until it's turned off, as it does with Steam or DS4Windows.

var (
	hidDLL                 = windows.NewLazySystemDLL("hid.dll")
	procHidDGetPreparsed   = hidDLL.NewProc("HidD_GetPreparsedData")
	procHidDFreePreparsed  = hidDLL.NewProc("HidD_FreePreparsedData")
	procHidPGetCaps        = hidDLL.NewProc("HidP_GetCaps")
	procCMGetInterfaceProp = cfgmgr32.NewProc("CM_Get_Device_Interface_PropertyW")

	// DEVPKEY_Device_InstanceId.
	keyInstanceID = windows.DEVPROPKEY{FmtID: windows.DEVPROPGUID{Data1: 0x78C34FC8, Data2: 0x104A, Data3: 0x4ACA, Data4: [8]byte{0x9E, 0xA4, 0x52, 0x4D, 0x52, 0x99, 0x6E, 0x57}}, PID: 256}

	errNoOutput       = errors.New("the controller takes no output reports")
	errXInputNotThere = errors.New("that Xbox-style controller isn't connected")
	errNoXInput       = errors.New("XInput isn't on this PC")
)

const hidpStatusSuccess = 0x00110000

// HIDInstance is the device instance id of a HID interface path, as
// PadDevice.Chain[0] has it; "" when Windows doesn't say.
func HIDInstance(path string) string {
	p16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var typ windows.DEVPROPTYPE
	buf := make([]uint16, 512)
	size := uint32(len(buf) * 2)
	r, _, _ := procCMGetInterfaceProp.Call(uintptr(unsafe.Pointer(p16)), uintptr(unsafe.Pointer(&keyInstanceID)), uintptr(unsafe.Pointer(&typ)),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0)
	if r != 0 || typ != windows.DEVPROP_TYPE_STRING {
		return ""
	}
	return windows.UTF16ToString(buf)
}

// PadWriter is a controller opened to send it output reports.
type PadWriter struct {
	h    windows.Handle
	size int
}

// OpenPadWrite opens a HID device by its path to write to it, shared with
// whoever else reads it, and reads how long its output reports are.
func OpenPadWrite(path string) (*PadWriter, error) {
	p16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p16, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("Windows wouldn't open the controller (another program may hold it for itself): %w", err)
	}
	w := &PadWriter{h: h}
	var pre uintptr
	if r, _, _ := procHidDGetPreparsed.Call(uintptr(h), uintptr(unsafe.Pointer(&pre))); r == 0 {
		w.Close()
		return nil, errNoOutput
	}
	defer procHidDFreePreparsed.Call(pre)
	var caps [64]byte // HIDP_CAPS
	if r, _, _ := procHidPGetCaps.Call(pre, uintptr(unsafe.Pointer(&caps[0]))); r != hidpStatusSuccess {
		w.Close()
		return nil, errNoOutput
	}
	w.size = int(caps[6]) | int(caps[7])<<8 // OutputReportByteLength
	if w.size == 0 {
		w.Close()
		return nil, errNoOutput
	}
	return w, nil
}

// Size is how long every output report written must be.
func (w *PadWriter) Size() int { return w.size }

// Write sends one output report, its id first, Size bytes long.
func (w *PadWriter) Write(report []byte) error {
	var n uint32
	if err := windows.WriteFile(w.h, report, &n, nil); err != nil {
		return fmt.Errorf("the controller didn't take it: %w", err)
	}
	return nil
}

// Close closes the controller.
func (w *PadWriter) Close() {
	if w == nil || w.h == 0 {
		return
	}
	windows.CloseHandle(w.h)
	w.h = 0
}

// XInputRumble sets an XInput slot's motors (0 to 65535; strong is the
// left one, weak the right).
func XInputRumble(slot int, strong, weak uint16) error {
	set := xinput().NewProc("XInputSetState")
	if set.Find() != nil {
		return errNoXInput
	}
	v := [2]uint16{strong, weak} // XINPUT_VIBRATION
	if r, _, _ := set.Call(uintptr(slot), uintptr(unsafe.Pointer(&v[0]))); r != 0 {
		return errXInputNotThere
	}
	return nil
}
