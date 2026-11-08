//go:build windows

package win

import (
	"errors"
	"image"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// ---------------------------------------------------------------------------
// Captures from other apps (core/captures.go): where Game Bar keeps them,
// whether it's on, and Windows' own thumbnail of a clip or picture.
// ---------------------------------------------------------------------------

var (
	folderCaptures           = &windows.KNOWNFOLDERID{Data1: 0xEDC0FE71, Data2: 0x98D8, Data3: 0x4F4A, Data4: [8]byte{0xB9, 0x20, 0xC8, 0xDC, 0x13, 0x3C, 0xB1, 0x65}}
	iidShellItemImageFactory = mustGUID("{bcc18b79-ba16-442f-80c4-8a59c30c463b}")
)

func Videos() string { return KnownFolder(windows.FOLDERID_Videos) }

// GameBarCaptures is the folder Game Bar saves to ("" if Windows names
// none), and whether it's turned off in Windows' Settings, Gaming, Captures.
func GameBarCaptures() (dir string, off bool) {
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\GameDVR`, registry.QUERY_VALUE); err == nil {
		v, _, err := k.GetIntegerValue("AppCaptureEnabled")
		off = err == nil && v == 0
		k.Close()
	}
	return KnownFolder(folderCaptures), off
}

// ShellThumbnail is the thumbnail Explorer shows for a file, about size
// pixels on its longer side, read from Windows' thumbnail cache or made by
// the file's thumbnail handler. Never an icon: a file with no thumbnail is
// an error.
func ShellThumbnail(path string, size int) (*image.RGBA, error) {
	var img *image.RGBA
	err := withCOM(func() error {
		var f *comObj
		r, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(wstr(path))), 0, uintptr(unsafe.Pointer(iidShellItemImageFactory)), uintptr(unsafe.Pointer(&f)))
		if err := hr(r); err != nil {
			return err
		}
		if f == nil {
			return errors.New("no thumbnail")
		}
		defer f.release()
		const siigbfBiggerSizeOK, siigbfThumbnailOnly = 0x1, 0x8
		var hbm windows.Handle
		// GetImage, with the SIZE passed by value in one register.
		sz := uintptr(uint32(size)) | uintptr(uint32(size))<<32
		if err := hr(f.call(3, sz, siigbfBiggerSizeOK|siigbfThumbnailOnly, uintptr(unsafe.Pointer(&hbm)))); err != nil {
			return err
		}
		defer procDeleteObject.Call(uintptr(hbm))
		var bm bitmapObj
		if r, _, _ := procGetObjectW.Call(uintptr(hbm), unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); r == 0 {
			return errors.New("couldn't read the thumbnail's size")
		}
		w, h := int(bm.bmWidth), int(bm.bmHeight)
		if h < 0 {
			h = -h
		}
		if w <= 0 || h <= 0 || w > 4096 || h > 4096 {
			return errors.New("unexpected thumbnail size")
		}
		px, err := bitmapPixels(hbm, w, h)
		if err != nil {
			return err
		}
		img = image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < w*h; i++ {
			img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = px[i*4+2], px[i*4+1], px[i*4], 255
		}
		return nil
	})
	return img, err
}
