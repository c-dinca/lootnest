//go:build windows

package win

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procRegisterClipboardFormat = user32.NewProc("RegisterClipboardFormatW")

// SetClipboardImage puts a PNG on the Windows clipboard twice over: as
// "PNG" (what Discord, browsers and chat apps paste with transparency) and
// as a device-independent bitmap (what everything else pastes).
func SetClipboardImage(hwnd uintptr, pngData []byte) error {
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return errors.New("that isn't a picture")
	}
	dib := DIBFromImage(img)
	if r, _, _ := procOpenClipboard.Call(hwnd); r == 0 {
		return errors.New("the clipboard is busy; try again")
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	const cfDIB = 8
	if err := putClip(cfDIB, dib); err != nil {
		return err
	}
	name, _ := windows.UTF16PtrFromString("PNG")
	if f, _, _ := procRegisterClipboardFormat.Call(uintptr(unsafe.Pointer(name))); f != 0 {
		_ = putClip(f, pngData) // the bitmap is already there if this fails
	}
	return nil
}

func putClip(format uintptr, data []byte) error {
	const gmemMoveable = 0x2
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(data)))
	if h == 0 {
		return errors.New("out of memory")
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return errors.New("couldn't lock clipboard memory")
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(p)), len(data)), data)
	procGlobalUnlock.Call(h)
	if r, _, _ := procSetClipboardData.Call(format, h); r == 0 {
		procGlobalFree.Call(h)
		return errors.New("couldn't put the picture on the clipboard")
	}
	return nil
}

// DIBFromImage is a CF_DIB: a BITMAPINFOHEADER, then 32-bit BGRA rows
// from the bottom up.
func DIBFromImage(img image.Image) []byte {
	b := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	w, h := b.Dx(), b.Dy()
	out := make([]byte, 40+w*h*4)
	le := binary.LittleEndian
	le.PutUint32(out[0:], 40)
	le.PutUint32(out[4:], uint32(int32(w)))
	le.PutUint32(out[8:], uint32(int32(h)))
	le.PutUint16(out[12:], 1)  // planes
	le.PutUint16(out[14:], 32) // bits per pixel
	le.PutUint32(out[20:], uint32(w*h*4))
	px := out[40:]
	for y := 0; y < h; y++ {
		src := rgba.Pix[(h-1-y)*rgba.Stride:]
		row := px[y*w*4:]
		for x := 0; x < w; x++ {
			r, g, bl, a := src[x*4], src[x*4+1], src[x*4+2], src[x*4+3]
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = bl, g, r, a
		}
	}
	return out
}
