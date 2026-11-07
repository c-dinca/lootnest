//go:build windows

package win

import (
	"encoding/binary"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// This PC's hardware, for comparing with a game's requirements. Everything
// is read locally: the registry and two kernel32 calls.

var (
	procGlobalMemoryStatusEx     = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetUserDefaultGeoName    = kernel32.NewProc("GetUserDefaultGeoName")
	procGetUserDefaultLocaleName = kernel32.NewProc("GetUserDefaultLocaleName")
)

// CPUName is the processor's name, such as "AMD Ryzen 7 5800X 8-Core
// Processor".
func CPUName() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, _ := k.GetStringValue("ProcessorNameString")
	return strings.Join(strings.Fields(v), " ")
}

// GPU is the graphics card with the most memory, and that memory in bytes
// (0 when the driver doesn't say). Windows' basic display driver is left
// out.
func GPU() (string, int64) {
	const class = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, class, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return "", 0
	}
	defer root.Close()
	subs, _ := root.ReadSubKeyNames(64)
	best, bestMem := "", int64(-1)
	for _, s := range subs {
		if _, err := strconv.Atoi(s); err != nil {
			continue // "Properties" and the like
		}
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, class+`\`+s, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		name, _, _ := k.GetStringValue("DriverDesc")
		mem := int64(0)
		if v, _, err := k.GetIntegerValue("HardwareInformation.qwMemorySize"); err == nil {
			mem = int64(v)
		} else if b, _, err := k.GetBinaryValue("HardwareInformation.MemorySize"); err == nil && len(b) >= 4 {
			mem = int64(binary.LittleEndian.Uint32(b))
		} else if v, _, err := k.GetIntegerValue("HardwareInformation.MemorySize"); err == nil {
			mem = int64(v)
		}
		k.Close()
		name = strings.Join(strings.Fields(name), " ")
		if name == "" || strings.Contains(strings.ToLower(name), "basic display") || strings.Contains(strings.ToLower(name), "basic render") {
			continue
		}
		if mem > bestMem {
			best, bestMem = name, mem
		}
	}
	if bestMem < 0 {
		bestMem = 0
	}
	return best, bestMem
}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// TotalRAM is the PC's memory in bytes, or 0.
func TotalRAM() int64 {
	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	if r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return 0
	}
	return int64(m.TotalPhys)
}

// WindowsName is "Windows 11" or "Windows 10" (from the build number,
// since Windows 11 still calls itself Windows 10 in the registry).
func WindowsName() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "Windows"
	}
	defer k.Close()
	build, _, _ := k.GetStringValue("CurrentBuild")
	if n, err := strconv.Atoi(build); err == nil && n >= 22000 {
		return "Windows 11"
	}
	if name, _, err := k.GetStringValue("ProductName"); err == nil && strings.HasPrefix(name, "Windows ") {
		f := strings.Fields(name)
		return f[0] + " " + f[1]
	}
	return "Windows"
}

// UserCountry is the two-letter country set in Windows (Region), such as
// "RO", or "" when Windows doesn't say.
func UserCountry() string {
	buf := make([]uint16, 16)
	if procGetUserDefaultGeoName.Find() == nil {
		if n, _, _ := procGetUserDefaultGeoName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); n > 1 {
			if s := windows.UTF16ToString(buf); len(s) == 2 {
				return strings.ToUpper(s)
			}
		}
	}
	buf = make([]uint16, 85)
	if n, _, _ := procGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); n > 1 {
		if _, cc, ok := strings.Cut(windows.UTF16ToString(buf), "-"); ok && len(cc) == 2 {
			return strings.ToUpper(cc)
		}
	}
	return ""
}

// MostFree is the fixed drive with the most free space, and that space.
func MostFree() (string, int64) {
	best, most := "", int64(-1)
	for _, d := range FixedDrives() {
		if f := DiskFree(d); f > most {
			best, most = strings.TrimSuffix(d, `\`), f
		}
	}
	if most < 0 {
		most = 0
	}
	return best, most
}
