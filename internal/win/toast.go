//go:build windows

package win

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------------------------------------------------------------------------
// A toast over a game: a small window of Lootnest's own, always on top, that
// never takes the focus and lets clicks through. It shows over windowed and
// borderless games, and over most fullscreen ones on Windows 10 and 11; a
// game in exclusive fullscreen draws over it. Nothing touches the game.
// ---------------------------------------------------------------------------

// Toast is one message: a small first line, a name and a line under it,
// with an accent colour ("#RRGGBB") down its left edge.
type Toast struct {
	Kicker string
	Title  string
	Line   string
	Accent string
	Badge  string // the trophy's colour ("#RRGGBB"), the achievement's tier; "" for none
}

var (
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procDestroyWindow              = user32.NewProc("DestroyWindow")
	procPostQuitMessage            = user32.NewProc("PostQuitMessage")
	procBeginPaint                 = user32.NewProc("BeginPaint")
	procEndPaint                   = user32.NewProc("EndPaint")
	procSetTimer                   = user32.NewProc("SetTimer")
	procKillTimer                  = user32.NewProc("KillTimer")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procMonitorFromWindow          = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW            = user32.NewProc("GetMonitorInfoW")
	procGetDpiForMonitor           = windows.NewLazySystemDLL("shcore.dll").NewProc("GetDpiForMonitor")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procFillRect                   = user32.NewProc("FillRect")
	procDrawTextW                  = user32.NewProc("DrawTextW")
	procCreateSolidBrush           = gdi32.NewProc("CreateSolidBrush")
	procCreateFontW                = gdi32.NewProc("CreateFontW")
	procSelectObject               = gdi32.NewProc("SelectObject")
	procSetTextColor               = gdi32.NewProc("SetTextColor")
	procSetBkMode                  = gdi32.NewProc("SetBkMode")
	procCreatePen                  = gdi32.NewProc("CreatePen")
	procEllipse                    = gdi32.NewProc("Ellipse")
	procPolygon                    = gdi32.NewProc("Polygon")
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type rect struct{ left, top, right, bottom int32 }

type msgT struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
	private uint32
}

type paintStruct struct {
	hdc      uintptr
	fErase   int32
	rcPaint  rect
	fRestore int32
	fIncUpd  int32
	reserved [32]byte
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

const (
	wmDestroy = 0x0002
	wmPaint   = 0x000F
	wmTimer   = 0x0113

	toastShowMs  = 5200
	toastFadeMs  = 16
	toastAlpha   = 245
	timerFade    = 1
	timerHide    = 2
	timerFadeOut = 3
)

var (
	toastOnce  sync.Once
	toastQueue = make(chan Toast, 16)
	toastClass *uint16
	toastProc  uintptr
	classErr   error // why the window class couldn't be registered, if it couldn't

	// What the window on screen shows; only the toast thread uses them.
	cur      Toast
	curDPI   = 96
	curAlpha = 0
	curX     int32 // where the toast comes to rest
	curY     int32
	curSlide float64 // how far it still is from there, in pixels (it slides in from the right)
)

// ToastFailed, when set, is told when a toast couldn't be shown (the
// window couldn't be made), with the toast, so it can go another way.
var ToastFailed func(t Toast, err error)

// toastShown, in tests, is told each toast window that went on screen.
var toastShown func(hwnd uintptr)

// ShowToast queues a toast; toasts show one after another. It returns at
// once. When many arrive together, the ones past the queue's room are
// dropped rather than kept for minutes.
func ShowToast(t Toast) {
	toastOnce.Do(func() { go toastLoop() })
	select {
	case toastQueue <- t:
	default:
	}
}

func toastLoop() {
	runtime.LockOSThread() // the window and its messages live on this thread
	hinst, _ := getModuleHandle()
	toastClass, _ = windows.UTF16PtrFromString("LootnestToast")
	toastProc = windows.NewCallback(toastWndProc)
	wc := wndClassEx{lpfnWndProc: toastProc, hInstance: hinst, lpszClassName: toastClass}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		classErr = e
	}
	for t := range toastQueue {
		showOne(hinst, t)
	}
}

func showOne(hinst uintptr, t Toast) {
	cur, curAlpha = t, 0
	// On the screen of the window in front (the game), at its bottom right.
	mon, _, _ := procMonitorFromWindow.Call(foreground(), 2 /*MONITOR_DEFAULTTONEAREST*/)
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	curDPI = monitorDPI(mon)
	px := func(n int) int32 { return int32(n * curDPI / 96) }
	w, h := px(380), px(84)
	x, y := mi.rcWork.right-w-px(24), mi.rcWork.bottom-h-px(24)
	if mi.rcWork.right == 0 { // no monitor info: the primary screen
		sw, _, _ := procGetSystemMetrics.Call(0)
		sh, _, _ := procGetSystemMetrics.Call(1)
		x, y = int32(sw)-w-px(24), int32(sh)-h-px(72)
	}
	const (
		wsPopup         = 0x80000000
		wsExTopmost     = 0x00000008
		wsExTransparent = 0x00000020
		wsExToolWindow  = 0x00000080
		wsExLayered     = 0x00080000
		wsExNoActivate  = 0x08000000
	)
	curX, curY, curSlide = x, y, float64(px(48))
	title, _ := windows.UTF16PtrFromString("Lootnest")
	hwnd, _, e := procCreateWindowExW.Call(wsExTopmost|wsExTransparent|wsExToolWindow|wsExLayered|wsExNoActivate,
		uintptr(unsafe.Pointer(toastClass)), uintptr(unsafe.Pointer(title)), wsPopup,
		uintptr(x+int32(curSlide)), uintptr(y), uintptr(w), uintptr(h), 0, 0, hinst, 0)
	if hwnd == 0 {
		if ToastFailed != nil {
			err := classErr
			if err == nil {
				err = e
			}
			ToastFailed(t, err)
		}
		return
	}
	if procDwmSetWindowAttribute.Find() == nil {
		corner := int32(3) // DWMWCP_ROUNDSMALL, on Windows 11
		procDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&corner)), 4)
	}
	procSetLayeredWindowAttributes.Call(hwnd, 0, 0, 2 /*LWA_ALPHA*/)
	const swShowNoActivate = 4
	procShowWindow.Call(hwnd, swShowNoActivate)
	if toastShown != nil {
		toastShown(hwnd)
	}
	const swpNoMove, swpNoSize, swpNoActivate = 0x2, 0x1, 0x10
	hwndTopmost := ^uintptr(0) // (HWND)-1
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
	procSetTimer.Call(hwnd, timerFade, toastFadeMs, 0)
	procSetTimer.Call(hwnd, timerHide, toastShowMs, 0)
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

func slideTo(hwnd uintptr) {
	const swpNoSize, swpNoZOrder, swpNoActivate = 0x1, 0x4, 0x10
	procSetWindowPos.Call(hwnd, 0, uintptr(curX+int32(curSlide)), uintptr(curY), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
}

func foreground() uintptr {
	h, _, _ := procGetForegroundWindow.Call()
	return h
}

func monitorDPI(mon uintptr) int {
	if procGetDpiForMonitor.Find() == nil {
		var dx, dy uint32
		if r, _, _ := procGetDpiForMonitor.Call(mon, 0, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy))); r == 0 && dx > 0 {
			return int(dx)
		}
	}
	return SystemDPI()
}

func toastWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmTimer:
		switch wp {
		case timerFade:
			// In: it fades up and slides the last few pixels into place,
			// fast at first and settling (each step takes 35% of what's left).
			curAlpha += 34
			if curAlpha > toastAlpha {
				curAlpha = toastAlpha
			}
			curSlide *= 0.65
			if curSlide < 0.5 {
				curSlide = 0
			}
			slideTo(hwnd)
			if curAlpha == toastAlpha && curSlide == 0 {
				procKillTimer.Call(hwnd, timerFade)
			}
			procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(curAlpha), 2)
		case timerHide:
			procKillTimer.Call(hwnd, timerHide)
			procSetTimer.Call(hwnd, timerFadeOut, toastFadeMs, 0)
		case timerFadeOut:
			curSlide += float64(curDPI) / 24 // out: back the way it came
			slideTo(hwnd)
			curAlpha -= 30
			if curAlpha <= 0 {
				procKillTimer.Call(hwnd, timerFadeOut)
				procDestroyWindow.Call(hwnd)
				return 0
			}
			procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(curAlpha), 2)
		}
		return 0
	case wmPaint:
		paintToast(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

// hexRGB reads "#RRGGBB" as 0xRRGGBB, or fallback.
func hexRGB(hex string, fallback uint32) uint32 {
	h := strings.TrimPrefix(hex, "#")
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil || len(h) != 6 {
		return fallback
	}
	return uint32(v)
}

// colorRef turns 0xRRGGBB into a GDI COLORREF (0x00BBGGRR).
func colorRef(rgb uint32) uint32 { return (rgb&0xFF)<<16 | (rgb & 0xFF00) | (rgb>>16)&0xFF }

// The brand's ground and text (DESIGN.md): a raised obsidian, Ice Grey, and
// a muted grey that keeps WCAG AA on that ground.
const (
	toastGround = 0x1A1C23
	toastEdge   = 0x2C2F39
	toastText   = 0xDFE5EB
	toastMuted  = 0xA3ACB7
	toastRed    = 0xEF233C
)

func paintToast(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	px := func(n int) int32 { return int32(n * curDPI / 96) }
	w, h := px(380), px(84)

	fill := func(r rect, rgb uint32) {
		b, _, _ := procCreateSolidBrush.Call(uintptr(colorRef(rgb)))
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), b)
		procDeleteObject.Call(b)
	}
	fill(rect{0, 0, w, h}, toastEdge) // a hairline round the edge
	fill(rect{1, 1, w - 1, h - 1}, toastGround)
	fill(rect{0, 0, px(4), h}, hexRGB(cur.Accent, toastRed))

	face, _ := windows.UTF16PtrFromString("Segoe UI")
	font := func(size, weight int) uintptr {
		f, _, _ := procCreateFontW.Call(uintptr(uint32(-px(size))), 0, 0, 0, uintptr(weight), 0, 0, 0,
			1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, uintptr(unsafe.Pointer(face)))
		return f
	}
	procSetBkMode.Call(hdc, 1 /*TRANSPARENT*/)
	text := func(s string, r rect, size, weight int, rgb uint32) {
		if s == "" {
			return
		}
		f := font(size, weight)
		old, _, _ := procSelectObject.Call(hdc, f)
		procSetTextColor.Call(hdc, uintptr(colorRef(rgb)))
		u, _ := windows.UTF16FromString(s)
		const dtSingle, dtEllipsis, dtNoPrefix, dtVCenter = 0x20, 0x8000, 0x800, 0x4
		procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&r)), dtSingle|dtEllipsis|dtNoPrefix|dtVCenter)
		procSelectObject.Call(hdc, old)
		procDeleteObject.Call(f)
	}
	l, r := px(20), w-px(16)
	if cur.Badge != "" {
		drawTrophy(hdc, px(20), h/2, px(22), hexRGB(cur.Badge, toastText), px)
		l = px(20) + px(44) + px(14)
	}
	text(cur.Kicker, rect{l, px(12), r, px(30)}, 12, 400, toastMuted)
	text(cur.Title, rect{l, px(30), r, px(52)}, 15, 600, toastText)
	lineColor := uint32(toastMuted)
	if cur.Badge != "" {
		lineColor = hexRGB(cur.Badge, toastMuted) // the tier's colour, as in the app
	}
	text(cur.Line, rect{l, px(52), r, px(72)}, 12, 600, lineColor)
}

// drawTrophy draws a round badge with a trophy in it, in the tier's colour:
// x is its left edge, cy its middle, rad its radius.
func drawTrophy(hdc uintptr, x, cy, rad int32, rgb uint32, px func(int) int32) {
	ring, _, _ := procCreatePen.Call(0 /*PS_SOLID*/, uintptr(px(2)), uintptr(colorRef(rgb)))
	face, _, _ := procCreateSolidBrush.Call(uintptr(colorRef(0x23252D)))
	oldPen, _, _ := procSelectObject.Call(hdc, ring)
	oldBrush, _, _ := procSelectObject.Call(hdc, face)
	procEllipse.Call(hdc, uintptr(x), uintptr(cy-rad), uintptr(x+2*rad), uintptr(cy+rad))
	// The cup, its stem and its base, drawn on a 20-unit grid.
	cx := x + rad
	u := func(n int) int32 { return int32(n) * rad / 11 }
	type pt struct{ x, y int32 }
	cup := []pt{{-7, -8}, {7, -8}, {6, -2}, {3, 2}, {1, 3}, {1, 5}, {4, 5}, {4, 8}, {-4, 8}, {-4, 5}, {-1, 5}, {-1, 3}, {-3, 2}, {-6, -2}}
	pts := make([]pt, len(cup))
	for i, p := range cup {
		pts[i] = pt{cx + u(int(p.x)), cy + u(int(p.y))}
	}
	cupPen, _, _ := procCreatePen.Call(0, 1, uintptr(colorRef(rgb)))
	cupBrush, _, _ := procCreateSolidBrush.Call(uintptr(colorRef(rgb)))
	procSelectObject.Call(hdc, cupPen)
	procSelectObject.Call(hdc, cupBrush)
	procPolygon.Call(hdc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	for _, o := range []uintptr{ring, face, cupPen, cupBrush} {
		procDeleteObject.Call(o)
	}
}
