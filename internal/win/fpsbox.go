//go:build windows

package win

import (
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------------------------------------------------------------------------
// The FPS counter on the screen: a small window of Lootnest's own in a
// corner of the screen the game is on, always on top, never taking the
// focus and letting every click through, as the toasts do. Like them it
// shows over windowed and borderless games and over most fullscreen ones on
// Windows 10 and 11; a game in exclusive fullscreen draws over it.
// ---------------------------------------------------------------------------

var (
	fpsMu     sync.Mutex
	fpsWant   bool
	fpsText   string
	fpsCorner string // "tl", "tr", "bl", "br"
	fpsThread bool

	// Only the counter's thread uses these.
	fpsDPI   = 96
	fpsDrawn string
	fpsAt    [4]int32
)

// ShowFPS puts the counter in a corner ("tl", "tr", "bl" or "br") of the
// screen the window in front is on, or changes what it says.
func ShowFPS(text, corner string) {
	fpsMu.Lock()
	defer fpsMu.Unlock()
	fpsWant, fpsText, fpsCorner = true, text, corner
	if !fpsThread {
		fpsThread = true
		go fpsLoop()
	}
}

// HideFPS takes the counter away.
func HideFPS() {
	fpsMu.Lock()
	fpsWant = false
	fpsMu.Unlock()
}

const (
	fpsW, fpsH   = 92, 36
	fpsMargin    = 12
	timerFPS     = 1
	fpsTickMs    = 200
	fpsAlphaFull = 235
)

func fpsLoop() {
	runtime.LockOSThread()
	defer func() {
		fpsMu.Lock()
		fpsThread = false
		again := fpsWant
		if again {
			fpsThread = true
		}
		fpsMu.Unlock()
		if again { // shown again while this one was closing
			go fpsLoop()
		}
	}()
	hinst, _ := getModuleHandle()
	class, _ := windows.UTF16PtrFromString("LootnestFPS")
	wc := wndClassEx{lpfnWndProc: windows.NewCallback(fpsWndProc), hInstance: hinst, lpszClassName: class}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))) // fails harmlessly when already registered
	const (
		wsPopup         = 0x80000000
		wsExTopmost     = 0x00000008
		wsExTransparent = 0x00000020
		wsExToolWindow  = 0x00000080
		wsExLayered     = 0x00080000
		wsExNoActivate  = 0x08000000
	)
	fpsDrawn, fpsAt = "", [4]int32{}
	title, _ := windows.UTF16PtrFromString("Lootnest FPS")
	hwnd, _, _ := procCreateWindowExW.Call(wsExTopmost|wsExTransparent|wsExToolWindow|wsExLayered|wsExNoActivate,
		uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), wsPopup, 0, 0, 1, 1, 0, 0, hinst, 0)
	if hwnd == 0 {
		fpsMu.Lock()
		fpsWant = false
		fpsMu.Unlock()
		return
	}
	if procDwmSetWindowAttribute.Find() == nil {
		corner := int32(3) // DWMWCP_ROUNDSMALL, on Windows 11
		procDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&corner)), 4)
	}
	procSetLayeredWindowAttributes.Call(hwnd, 0, fpsAlphaFull, 2 /*LWA_ALPHA*/)
	fpsPlace(hwnd)
	const swShowNoActivate = 4
	procShowWindow.Call(hwnd, swShowNoActivate)
	procSetTimer.Call(hwnd, timerFPS, fpsTickMs, 0)
	var m msgT
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// fpsPlace keeps the counter in its corner of the front window's screen,
// on top, and redraws it when its number changed.
func fpsPlace(hwnd uintptr) {
	fpsMu.Lock()
	text, corner := fpsText, fpsCorner
	fpsMu.Unlock()
	fg := foreground()
	var mon uintptr
	if fg != 0 && fg != hwnd {
		mon, _, _ = procMonitorFromWindow.Call(fg, 2 /*MONITOR_DEFAULTTONEAREST*/)
	} else {
		mon, _, _ = procMonitorFromPoint.Call(0, 1 /*MONITOR_DEFAULTTOPRIMARY*/)
	}
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	fpsDPI = monitorDPI(mon)
	px := func(n int) int32 { return int32(n * fpsDPI / 96) }
	w, h, m := px(fpsW), px(fpsH), px(fpsMargin)
	r := mi.rcMonitor // a game covers the taskbar, so the whole screen
	x, y := r.left+m, r.top+m
	if corner == "tr" || corner == "br" {
		x = r.right - w - m
	}
	if corner == "bl" || corner == "br" {
		y = r.bottom - h - m
	}
	at := [4]int32{x, y, w, h}
	const swpNoActivate, swpNoMove, swpNoSize = 0x10, 0x2, 0x1
	hwndTopmost := ^uintptr(0)
	if at != fpsAt {
		fpsAt = at
		procSetWindowPos.Call(hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoActivate)
	} else {
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
	}
	if text != fpsDrawn {
		fpsDrawn = text
		procInvalidateRect.Call(hwnd, 0, 0)
	}
}

func fpsWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmTimer:
		fpsMu.Lock()
		want := fpsWant
		fpsMu.Unlock()
		if !want {
			procKillTimer.Call(hwnd, timerFPS)
			procDestroyWindow.Call(hwnd)
			return 0
		}
		fpsPlace(hwnd)
		return 0
	case wmMouseActivate:
		return 3 // MA_NOACTIVATE
	case 0x0084: // WM_NCHITTEST
		return ^uintptr(0) // HTTRANSPARENT: clicks go to the game
	case wmPaint:
		paintFPS(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

func paintFPS(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	px := func(n int) int32 { return int32(n * fpsDPI / 96) }
	w, h := px(fpsW), px(fpsH)
	fill := func(r rect, rgb uint32) {
		br, _, _ := procCreateSolidBrush.Call(uintptr(colorRef(rgb)))
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), br)
		procDeleteObject.Call(br)
	}
	fill(rect{0, 0, w, h}, toastEdge) // a hairline round the edge
	fill(rect{1, 1, w - 1, h - 1}, toastGround)

	face, _ := windows.UTF16PtrFromString("Segoe UI")
	procSetBkMode.Call(hdc, 1 /*TRANSPARENT*/)
	text := func(s string, r rect, size, weight int, rgb uint32, flags uintptr) {
		f, _, _ := procCreateFontW.Call(uintptr(uint32(-px(size))), 0, 0, 0, uintptr(weight), 0, 0, 0,
			1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, uintptr(unsafe.Pointer(face)))
		old, _, _ := procSelectObject.Call(hdc, f)
		procSetTextColor.Call(hdc, uintptr(colorRef(rgb)))
		u, _ := windows.UTF16FromString(s)
		procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&r)), flags)
		procSelectObject.Call(hdc, old)
		procDeleteObject.Call(f)
	}
	const dtSingle, dtNoPrefix, dtVCenter, dtRight = 0x20, 0x800, 0x4, 0x2
	n := fpsDrawn
	if n == "" {
		n = "--"
	}
	// The number, right-aligned so it doesn't jump as digits change, and
	// a small label beside it.
	text(n, rect{px(6), 0, px(56), h}, 20, 700, toastText, dtSingle|dtNoPrefix|dtVCenter|dtRight)
	text("FPS", rect{px(61), px(2), w - px(6), h}, 11, 600, toastMuted, dtSingle|dtNoPrefix|dtVCenter)
}
