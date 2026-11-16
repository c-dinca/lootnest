//go:build windows

package win

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// Reading a PlayStation controller's own input reports, for the pad
// Lootnest makes of it (vigem.go). The pad is opened for reading and
// shared, and only read: nothing is sent to it, and no feature report is
// asked for, so a Bluetooth DualSense stays in the mode it's in.

// GUID_DEVINTERFACE_HID.
var hidGUID = windows.GUID{Data1: 0x4D1E55B2, Data2: 0xF16F, Data3: 0x11CF, Data4: [8]byte{0x88, 0xCB, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30}}

// HIDPaths are the device paths of the HID devices present, which name
// each one's vendor and product (core.PSPadPath reads them).
func HIDPaths() []string {
	l, err := windows.CM_Get_Device_Interface_List("", &hidGUID, cmPresent)
	if err != nil {
		return nil
	}
	return l
}

// PadReader is a controller opened for reading.
type PadReader struct{ h windows.Handle }

// OpenPad opens a HID device by its path, shared with whoever else reads it.
func OpenPad(path string) (*PadReader, error) {
	p16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p16, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("Windows wouldn't open the controller (another program may hold it for itself): %w", err)
	}
	return &PadReader{h: h}, nil
}

// Read waits for the pad's next input report and copies it into buf, the
// report's id first. It fails when the pad goes, or once Close is called.
func (r *PadReader) Read(buf []byte) (int, error) {
	var n uint32
	if err := windows.ReadFile(r.h, buf, &n, nil); err != nil {
		return 0, err
	}
	return int(n), nil
}

// Close ends a Read that's waiting and closes the pad. It may be called
// from another goroutine than the one reading, once.
func (r *PadReader) Close() {
	if r.h == 0 {
		return
	}
	_ = windows.CancelIoEx(r.h, nil)
	windows.CloseHandle(r.h)
	r.h = 0
}
