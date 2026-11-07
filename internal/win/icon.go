//go:build windows

package win

import (
	"errors"
	"image"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------------------------------------------------------------------------
// A program's own icon, as an image: the Library shows it for games that
// have no cover, so every game has a picture without going online.
// ---------------------------------------------------------------------------

var (
	gdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procPrivateExtractIconsW = user32.NewProc("PrivateExtractIconsW")
	procGetIconInfo          = user32.NewProc("GetIconInfo")
	procDestroyIcon          = user32.NewProc("DestroyIcon")
	procGetDC                = user32.NewProc("GetDC")
	procReleaseDC            = user32.NewProc("ReleaseDC")
	procGetDIBits            = gdi32.NewProc("GetDIBits")
	procGetObjectW           = gdi32.NewProc("GetObjectW")
	procDeleteObject         = gdi32.NewProc("DeleteObject")
)

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  windows.Handle
	hbmColor windows.Handle
}

type bitmapObj struct {
	bmType       int32
	bmWidth      int32
	bmHeight     int32
	bmWidthBytes int32
	bmPlanes     uint16
	bmBitsPixel  uint16
	bmBits       uintptr
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

// bitmapPixels reads a bitmap as top-down 32-bit BGRA.
func bitmapPixels(hbm windows.Handle, w, h int) ([]byte, error) {
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, errors.New("no device context")
	}
	defer procReleaseDC.Call(0, hdc)
	bih := bitmapInfoHeader{biWidth: int32(w), biHeight: -int32(h), biPlanes: 1, biBitCount: 32}
	bih.biSize = uint32(unsafe.Sizeof(bih))
	// BITMAPINFO is the header plus a colour table, unused at 32 bits; leave room anyway.
	info := make([]byte, int(bih.biSize)+16)
	*(*bitmapInfoHeader)(unsafe.Pointer(&info[0])) = bih
	buf := make([]byte, w*h*4)
	r, _, _ := procGetDIBits.Call(hdc, uintptr(hbm), 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&info[0])), 0)
	if r == 0 {
		return nil, errors.New("couldn't read the icon's pixels")
	}
	return buf, nil
}

// ExeIcon returns the icon of an .exe at about size pixels (Windows picks
// the closest size the file has).
func ExeIcon(path string, size int) (image.Image, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var hicon windows.Handle
	var id uint32
	n, _, _ := procPrivateExtractIconsW.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(size), uintptr(size),
		uintptr(unsafe.Pointer(&hicon)), uintptr(unsafe.Pointer(&id)), 1, 0)
	if n == 0 || n == 0xFFFFFFFF || hicon == 0 {
		return nil, errors.New("the file has no icon")
	}
	defer procDestroyIcon.Call(uintptr(hicon))
	var ii iconInfo
	if r, _, _ := procGetIconInfo.Call(uintptr(hicon), uintptr(unsafe.Pointer(&ii))); r == 0 {
		return nil, errors.New("couldn't read the icon")
	}
	if ii.hbmMask != 0 {
		defer procDeleteObject.Call(uintptr(ii.hbmMask))
	}
	if ii.hbmColor == 0 {
		return nil, errors.New("the icon is black and white")
	}
	defer procDeleteObject.Call(uintptr(ii.hbmColor))
	var bm bitmapObj
	if r, _, _ := procGetObjectW.Call(uintptr(ii.hbmColor), unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); r == 0 {
		return nil, errors.New("couldn't read the icon's size")
	}
	w, h := int(bm.bmWidth), int(bm.bmHeight)
	if w <= 0 || h <= 0 || w > 1024 || h > 1024 {
		return nil, errors.New("unexpected icon size")
	}
	px, err := bitmapPixels(ii.hbmColor, w, h)
	if err != nil {
		return nil, err
	}
	hasAlpha := false
	for i := 3; i < len(px); i += 4 {
		if px[i] != 0 {
			hasAlpha = true
			break
		}
	}
	var mask []byte
	if !hasAlpha && ii.hbmMask != 0 {
		mask, _ = bitmapPixels(ii.hbmMask, w, h) // white in the mask = transparent
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		b, g, r, a := px[i*4], px[i*4+1], px[i*4+2], px[i*4+3]
		if !hasAlpha {
			a = 255
			if mask != nil && mask[i*4] != 0 {
				a = 0
			}
		}
		img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = r, g, b, a
	}
	return img, nil
}
