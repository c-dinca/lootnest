//go:build windows

package win

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Game controllers as Windows lists them. Only SetupAPI, the device tree
// and XInput's state: no device is opened, and no HID report is read or
// written, since that can switch a Bluetooth DualSense into another mode
// and break games that expect the simple one.

var (
	cfgmgr32                 = windows.NewLazySystemDLL("cfgmgr32.dll")
	procCMGetParent          = cfgmgr32.NewProc("CM_Get_Parent")
	procCMGetDeviceID        = cfgmgr32.NewProc("CM_Get_Device_IDW")
	procCMGetDevNodeProperty = cfgmgr32.NewProc("CM_Get_DevNode_PropertyW")

	guidHID  = windows.GUID{Data1: 0x4D1E55B2, Data2: 0xF16F, Data3: 0x11CF, Data4: [8]byte{0x88, 0xCB, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30}}
	guidXUSB = windows.GUID{Data1: 0xEC87F1E3, Data2: 0xC13B, Data3: 0x4100, Data4: [8]byte{0xB5, 0xF7, 0x8B, 0x84, 0xD5, 0x42, 0x60, 0xCB}}

	keyBusDesc = windows.DEVPROPKEY{FmtID: windows.DEVPROPGUID{Data1: 0x540B947E, Data2: 0x8B40, Data3: 0x45BC, Data4: [8]byte{0xA8, 0xA2, 0x6A, 0x0B, 0x89, 0x4C, 0xBD, 0xA2}}, PID: 4}
	keyName    = windows.DEVPROPKEY{FmtID: windows.DEVPROPGUID{Data1: 0xB725F130, Data2: 0x47EF, Data3: 0x101A, Data4: [8]byte{0xA5, 0xF1, 0x02, 0x60, 0x8C, 0x9E, 0xBA, 0xC0}}, PID: 10}
)

// PadDevice is one present HID or Xbox 360 pad device: its instance id and
// its parents' (up to four), its hardware and compatible ids, and a name.
type PadDevice struct {
	Chain       []string
	HardwareIDs []string
	CompatIDs   []string
	Name        string
	XUSB        bool
}

// PadDevices lists every present HID device, then every Xbox 360 (XUSB)
// pad, which has no HID device of its own.
func PadDevices() []PadDevice {
	out := padDevices(&guidHID, false)
	return append(out, padDevices(&guidXUSB, true)...)
}

func padDevices(guid *windows.GUID, xusb bool) []PadDevice {
	set, err := windows.SetupDiGetClassDevsEx(guid, "", 0, windows.DIGCF_PRESENT|windows.DIGCF_DEVICEINTERFACE, 0, "")
	if err != nil {
		return nil
	}
	defer set.Close()
	var out []PadDevice
	for i := 0; ; i++ {
		info, err := set.EnumDeviceInfo(i)
		if err == windows.ERROR_NO_MORE_ITEMS {
			break
		}
		if err != nil {
			continue
		}
		d := PadDevice{XUSB: xusb, HardwareIDs: regStrings(set, info, windows.SPDRP_HARDWAREID), CompatIDs: regStrings(set, info, windows.SPDRP_COMPATIBLEIDS)}
		for n, dev := 0, info.DevInst; n < 4; n++ {
			id := deviceID(dev)
			if id == "" {
				break
			}
			d.Chain = append(d.Chain, id)
			// The pad's own name is on the USB or Bluetooth device above
			// a HID one ("DualSense Wireless Controller"), not on the HID
			// device ("HID-compliant game controller").
			if n <= 1 && d.Name == "" {
				d.Name = devString(dev, &keyBusDesc)
			}
			if n == 1 && d.Name == "" {
				d.Name = devString(dev, &keyName)
			}
			var parent windows.DEVINST
			if r, _, _ := procCMGetParent.Call(uintptr(unsafe.Pointer(&parent)), uintptr(dev), 0); r != 0 {
				break
			}
			dev = parent
		}
		if d.Name == "" {
			d.Name = devString(info.DevInst, &keyName)
		}
		out = append(out, d)
	}
	return out
}

func regStrings(set windows.DevInfo, info *windows.DevInfoData, p windows.SPDRP) []string {
	v, err := set.DeviceRegistryProperty(info, p)
	if err != nil {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case string:
		return []string{s}
	}
	return nil
}

func deviceID(dev windows.DEVINST) string {
	buf := make([]uint16, 512)
	if r, _, _ := procCMGetDeviceID.Call(uintptr(dev), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0); r != 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}

// devString reads one string property of a device node, "" when it has none.
func devString(dev windows.DEVINST, key *windows.DEVPROPKEY) string {
	var typ windows.DEVPROPTYPE
	buf := make([]uint16, 256)
	size := uint32(len(buf) * 2)
	r, _, _ := procCMGetDevNodeProperty.Call(uintptr(dev), uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0)
	if r != 0 || typ != windows.DEVPROP_TYPE_STRING {
		return ""
	}
	return windows.UTF16ToString(buf)
}

// XInputPad is one connected XInput slot (Index 0 to 3).
type XInputPad struct {
	Index    int
	SubType  uint8
	Wireless bool
}

var xinput = sync.OnceValue(func() *windows.LazyDLL {
	if d := windows.NewLazySystemDLL("xinput1_4.dll"); d.Load() == nil {
		return d
	}
	return windows.NewLazySystemDLL("xinput9_1_0.dll")
})

// XInputPads lists the XInput slots that have a pad in them. XInput only
// asks the Xbox pad driver for its state; it sends nothing to the pad.
func XInputPads() []XInputPad {
	dll := xinput()
	getState, getCaps := dll.NewProc("XInputGetState"), dll.NewProc("XInputGetCapabilities")
	if getState.Find() != nil || getCaps.Find() != nil {
		return nil
	}
	var out []XInputPad
	for i := 0; i < 4; i++ {
		var state [16]byte // XINPUT_STATE
		if r, _, _ := getState.Call(uintptr(i), uintptr(unsafe.Pointer(&state[0]))); r != 0 {
			continue
		}
		var caps struct { // XINPUT_CAPABILITIES
			Type, SubType uint8
			Flags         uint16
			_             [16]byte
		}
		p := XInputPad{Index: i}
		if r, _, _ := getCaps.Call(uintptr(i), 0, uintptr(unsafe.Pointer(&caps))); r == 0 {
			p.SubType, p.Wireless = caps.SubType, caps.Flags&0x0002 != 0
		}
		out = append(out, p)
	}
	return out
}

// ServiceState says whether a service is installed, and running, asking the
// service manager for its status only.
func ServiceState(name string) (installed, running bool) {
	m, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return false, false
	}
	defer windows.CloseServiceHandle(m)
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return false, false
	}
	s, err := windows.OpenService(m, n, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return false, false
	}
	defer windows.CloseServiceHandle(s)
	var st windows.SERVICE_STATUS
	if windows.QueryServiceStatus(s, &st) != nil {
		return true, false
	}
	return true, st.CurrentState == windows.SERVICE_RUNNING
}
