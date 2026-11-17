//go:build windows

package win

import (
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	versionDLL                  = windows.NewLazySystemDLL("version.dll")
	procGetFileVersionInfoSizeW = versionDLL.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW     = versionDLL.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = versionDLL.NewProc("VerQueryValueW")
)

// FileVersion is the file version in a program's version resource, such as
// "1.4.2" (trailing zeros dropped, at least two numbers), or "" when it has
// none. The file is only read.
func FileVersion(path string) string {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var handle uint32
	size, _, _ := procGetFileVersionInfoSizeW.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&handle)))
	if size == 0 || size > 16<<20 {
		return ""
	}
	buf := make([]byte, size)
	if ok, _, _ := procGetFileVersionInfoW.Call(uintptr(unsafe.Pointer(p)), 0, size, uintptr(unsafe.Pointer(&buf[0]))); ok == 0 {
		return ""
	}
	root, _ := windows.UTF16PtrFromString(`\`)
	var info unsafe.Pointer
	var n uint32
	if ok, _, _ := procVerQueryValueW.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(root)), uintptr(unsafe.Pointer(&info)), uintptr(unsafe.Pointer(&n))); ok == 0 || n < 16 || info == nil {
		return ""
	}
	// VS_FIXEDFILEINFO: signature, struct version, then the file version in
	// two 32-bit words.
	f := (*[4]uint32)(info)
	if f[0] != 0xFEEF04BD {
		return ""
	}
	parts := []uint32{f[2] >> 16, f[2] & 0xffff, f[3] >> 16, f[3] & 0xffff}
	for len(parts) > 2 && parts[len(parts)-1] == 0 {
		parts = parts[:len(parts)-1]
	}
	if parts[0] == 0 && parts[1] == 0 && len(parts) == 2 {
		return ""
	}
	s := make([]string, len(parts))
	for i, v := range parts {
		s[i] = strconv.FormatUint(uint64(v), 10)
	}
	return strings.Join(s, ".")
}
