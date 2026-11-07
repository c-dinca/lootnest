//go:build windows

package win

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Protect encrypts b for the signed-in Windows user (DPAPI): only this
// user, on this PC, can read it back. Used for keys the user saves in
// Settings, so config.json never holds them in plain text.
func Protect(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("nothing to protect")
	}
	in := windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

// Unprotect decrypts what Protect made.
func Unprotect(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("nothing to read")
	}
	in := windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
