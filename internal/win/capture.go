//go:build windows

package win

import (
	"errors"
	"image"
	"sync"
	"syscall"
	"unsafe"
)

// ---------------------------------------------------------------------------
// A picture of a game's window, taken from outside it with GDI, as any
// screenshot tool does: nothing goes into the game. A game in exclusive
// fullscreen can come out black; the caller says so.
// ---------------------------------------------------------------------------

var (
	procPrintWindow        = user32.NewProc("PrintWindow")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procClientToScreen     = user32.NewProc("ClientToScreen")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procGdiFlush           = gdi32.NewProc("GdiFlush")
)

const (
	pwClientOnly              = 0x1
	pwRenderFullContent       = 0x2
	srcCopy                   = 0x00CC0020
	wsExToolWindow            = 0x80
	gwlExStyle                = ^uintptr(19) // GWL_EXSTYLE (-20)
	captureMaxSide      int32 = 16384
)

// One callback for the whole program, like closeCB; findMu guards its state.
var (
	findMu   sync.Mutex
	findPIDs map[uint32]bool
	findBest uintptr
	findArea int64
	findCB   = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if !findPIDs[pid] {
			return 1
		}
		if vis, _, _ := procIsWindowVisible.Call(hwnd); vis == 0 {
			return 1
		}
		if ex, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle); ex&wsExToolWindow != 0 {
			return 1
		}
		var r rect
		procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		if a := int64(r.right-r.left) * int64(r.bottom-r.top); a > findArea {
			findBest, findArea = hwnd, a
		}
		return 1
	})
)

// GameWindow is the largest visible top-level window (not a tool window) of
// the processes named proc (lower case, without .exe), or 0.
func GameWindow(proc string) uintptr {
	pids := processIDs(proc)
	if len(pids) == 0 {
		return 0
	}
	findMu.Lock()
	defer findMu.Unlock()
	findPIDs, findBest, findArea = pids, 0, 0
	procEnumWindows.Call(findCB, 0)
	findPIDs = nil
	return findBest
}

// clientArea is a window's client area on the screen.
func clientArea(hwnd uintptr) (x, y, w, h int32, err error) {
	if r, _, _ := procIsIconic.Call(hwnd); r != 0 {
		return 0, 0, 0, 0, errors.New("the game's window is minimized")
	}
	var r rect
	if ok, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return 0, 0, 0, 0, errors.New("the game's window is gone")
	}
	pt := struct{ x, y int32 }{}
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
	w, h = r.right-r.left, r.bottom-r.top
	if w <= 0 || h <= 0 || w > captureMaxSide || h > captureMaxSide {
		return 0, 0, 0, 0, errors.New("the game's window has no picture to take")
	}
	return pt.x, pt.y, w, h, nil
}

// grab draws into a w by h 32-bit DIB with paint and returns its pixels.
func grab(w, h int32, paint func(mem, screen uintptr) bool) (*image.RGBA, error) {
	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return nil, errors.New("no device context")
	}
	defer procReleaseDC.Call(0, screen)
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	if mem == 0 {
		return nil, errors.New("no memory device context")
	}
	defer procDeleteDC.Call(mem)
	bih := bitmapInfoHeader{biWidth: w, biHeight: -h, biPlanes: 1, biBitCount: 32}
	bih.biSize = uint32(unsafe.Sizeof(bih))
	var bits unsafe.Pointer
	hbm, _, _ := procCreateDIBSection.Call(screen, uintptr(unsafe.Pointer(&bih)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		return nil, errors.New("not enough memory for the picture")
	}
	defer procDeleteObject.Call(hbm)
	old, _, _ := procSelectObject.Call(mem, hbm)
	ok := paint(mem, screen)
	procGdiFlush.Call()
	procSelectObject.Call(mem, old)
	if !ok {
		return nil, errors.New("Windows couldn't copy the game's window")
	}
	src := unsafe.Slice((*byte)(bits), int(w)*int(h)*4)
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	for i := 0; i < len(src); i += 4 { // BGRA to RGBA, opaque
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = src[i+2], src[i+1], src[i], 255
	}
	return img, nil
}

// CaptureWindow asks a window to draw its client area (PrintWindow with
// PW_RENDERFULLCONTENT, which takes DirectX content too), even behind
// other windows.
func CaptureWindow(hwnd uintptr) (*image.RGBA, error) {
	_, _, w, h, err := clientArea(hwnd)
	if err != nil {
		return nil, err
	}
	return grab(w, h, func(mem, _ uintptr) bool {
		r, _, _ := procPrintWindow.Call(hwnd, mem, pwClientOnly|pwRenderFullContent)
		return r != 0
	})
}

// CaptureScreen copies what the screen shows over a window's client area,
// for when the window can't draw itself. Only for the window in front, so
// it never takes a picture of something else.
func CaptureScreen(hwnd uintptr) (*image.RGBA, error) {
	if foreground() != hwnd {
		return nil, errors.New("the game isn't the window in front")
	}
	x, y, w, h, err := clientArea(hwnd)
	if err != nil {
		return nil, err
	}
	return grab(w, h, func(mem, screen uintptr) bool {
		r, _, _ := procBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), screen, uintptr(x), uintptr(y), srcCopy)
		return r != 0
	})
}
