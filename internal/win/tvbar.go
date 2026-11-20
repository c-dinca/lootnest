//go:build windows

package win

import (
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------------------------------------------------------------------------
// TV mode's bar on the PC: while Lootnest is on the TV, a small window of its
// own sits at the bottom right of each of the PC's screens, always on top, saying so,
// with a key that leaves TV mode. It never takes the focus, so whatever the
// PC is doing keeps the keyboard. Drawn with GDI on its own thread, as the
// toasts are.
// ---------------------------------------------------------------------------

// TVBar is what the bar shows.
type TVBar struct {
	Title string // "Lootnest is on the TV"
	Line  string // under it, smaller
	Key   string // the key's label
	Busy  string // the key's label once pressed
}

var (
	procRoundRect       = gdi32.NewProc("RoundRect")
	procInvalidateRect  = user32.NewProc("InvalidateRect")
	procLoadCursorW     = user32.NewProc("LoadCursorW")
	procSetCursor       = user32.NewProc("SetCursor")
	procTrackMouseEvent = user32.NewProc("TrackMouseEvent")
	procGetWindowRect   = user32.NewProc("GetWindowRect")
)

var (
	barMu     sync.Mutex
	barWant   bool
	barShows  TVBar
	barOn     [][4]int // the PC's displays, one bar on each
	barLeave  func()
	barThread bool // the bars' thread runs
)

// barWin is one bar's own state. Only the bars' thread uses it.
type barWin struct {
	dpi          int
	hover, press bool
	drawn        TVBar
	at           [4]int
}

// Only the bars' thread uses these.
var (
	bars       = map[uintptr]*barWin{}
	barOrder   []uintptr // bars[barOrder[i]] is on barOn[i]
	barPressed bool      // the key was used on one of them; all show Busy until they go
)

// ShowTVBar puts a bar on each display area (left, top, right, bottom),
// or updates what they show. leave runs, on another goroutine, when a
// bar's key is pressed.
func ShowTVBar(areas [][4]int, b TVBar, leave func()) {
	barMu.Lock()
	defer barMu.Unlock()
	barWant, barShows, barOn, barLeave = len(areas) > 0, b, append([][4]int(nil), areas...), leave
	if barWant && !barThread {
		barThread = true
		go barLoop()
	}
}

// HideTVBar takes the bars away.
func HideTVBar() {
	barMu.Lock()
	barWant = false
	barMu.Unlock()
}

const (
	barW, barH       = 560, 72
	barMargin        = 24
	barKeyW, barKeyH = 148, 40
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmMouseLeave     = 0x02A3
	wmSetCursor      = 0x0020
	wmMouseActivate  = 0x0021
)

func barLoop() {
	runtime.LockOSThread()
	defer func() {
		barMu.Lock()
		barThread = false
		again := barWant
		if again {
			barThread = true
		}
		barMu.Unlock()
		if again { // shown again while this one was closing
			go barLoop()
		}
	}()
	hinst, _ := getModuleHandle()
	class, _ := windows.UTF16PtrFromString("LootnestTVBar")
	wc := wndClassEx{lpfnWndProc: windows.NewCallback(barWndProc), hInstance: hinst, lpszClassName: class}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))) // fails harmlessly when already registered
	bars, barOrder, barPressed = map[uintptr]*barWin{}, nil, false
	// One timer for the thread (no window): it keeps one bar per display.
	timer, _, _ := procSetTimer.Call(0, 0, 150, 0)
	defer procKillTimer.Call(0, timer)
	if !barSync(hinst, class) {
		return
	}
	var m msgT
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		if m.hwnd == 0 && m.message == wmTimer {
			if !barSync(hinst, class) {
				return
			}
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// barSync makes one bar for each display asked for, places each and
// redraws what changed; false once no bar is wanted, with every bar gone.
// Called on the bars' thread.
func barSync(hinst uintptr, class *uint16) bool {
	barMu.Lock()
	want, on, shows := barWant, barOn, barShows
	barMu.Unlock()
	if !want {
		on = nil
	}
	for len(barOrder) > len(on) {
		h := barOrder[len(barOrder)-1]
		barOrder = barOrder[:len(barOrder)-1]
		procDestroyWindow.Call(h)
	}
	for len(barOrder) < len(on) {
		h := newBar(hinst, class)
		if h == 0 {
			break
		}
		bars[h] = &barWin{dpi: 96}
		barOrder = append(barOrder, h)
	}
	if len(barOrder) == 0 {
		barMu.Lock()
		if len(on) > 0 { // a bar couldn't be made: don't try forever
			barWant = false
		}
		barMu.Unlock()
		return false
	}
	for i, h := range barOrder {
		barPlace(h, bars[h], on[i], shows)
	}
	return true
}

func newBar(hinst uintptr, class *uint16) uintptr {
	const (
		wsPopup        = 0x80000000
		wsExTopmost    = 0x00000008
		wsExToolWindow = 0x00000080
		wsExLayered    = 0x00080000
		wsExNoActivate = 0x08000000
	)
	title, _ := windows.UTF16PtrFromString("Lootnest TV mode")
	hwnd, _, _ := procCreateWindowExW.Call(wsExTopmost|wsExToolWindow|wsExLayered|wsExNoActivate,
		uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), wsPopup, 0, 0, 1, 1, 0, 0, hinst, 0)
	if hwnd == 0 {
		return 0
	}
	if procDwmSetWindowAttribute.Find() == nil {
		corner := int32(2) // DWMWCP_ROUND, on Windows 11
		procDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&corner)), 4)
	}
	procSetLayeredWindowAttributes.Call(hwnd, 0, 248, 2 /*LWA_ALPHA*/)
	return hwnd
}

// barPlace moves a bar to the bottom right of its display's work area
// (above a taskbar there), on top, shows it, and redraws it when what it
// shows changed. Called on the bars' thread.
func barPlace(hwnd uintptr, b *barWin, on [4]int, shows TVBar) {
	cx, cy := int32((on[0]+on[2])/2), int32((on[1]+on[3])/2)
	mon, _, _ := procMonitorFromPoint.Call(uintptr(uint32(cx))|uintptr(uint32(cy))<<32, 2 /*MONITOR_DEFAULTTONEAREST*/)
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	b.dpi = monitorDPI(mon)
	px := func(n int) int32 { return int32(n * b.dpi / 96) }
	w, h := px(barW), px(barH)
	x, y := mi.rcWork.right-w-px(barMargin), mi.rcWork.bottom-h-px(barMargin)
	at := [4]int{int(x), int(y), int(w), int(h)}
	const swpNoActivate, swpShowWindow = 0x10, 0x40
	hwndTopmost := ^uintptr(0)
	if at != b.at {
		b.at = at
		procSetWindowPos.Call(hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoActivate|swpShowWindow)
	} else {
		const swpNoMove, swpNoSize = 0x2, 0x1
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
	}
	if shows != b.drawn {
		b.drawn = shows
		procInvalidateRect.Call(hwnd, 0, 1)
	}
}

// barKey is the key's area in a bar.
func barKey(dpi int) rect {
	px := func(n int) int32 { return int32(n * dpi / 96) }
	w, h := px(barW), px(barH)
	return rect{w - px(16) - px(barKeyW), (h - px(barKeyH)) / 2, w - px(16), (h + px(barKeyH)) / 2}
}

func inRect(r rect, lp uintptr) bool {
	x, y := int32(int16(lp&0xFFFF)), int32(int16(lp>>16&0xFFFF))
	return x >= r.left && x < r.right && y >= r.top && y < r.bottom
}

func barWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	b := bars[hwnd]
	if b == nil {
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
		return r
	}
	switch msg {
	case wmMouseActivate:
		return 3 // MA_NOACTIVATE: the PC's window in front keeps the keyboard
	case wmSetCursor:
		var pt struct{ x, y int32 }
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		var wr rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
		k := barKey(b.dpi)
		if x, y := pt.x-wr.left, pt.y-wr.top; x >= k.left && x < k.right && y >= k.top && y < k.bottom && !barPressed {
			c, _, _ := procLoadCursorW.Call(0, 32649 /*IDC_HAND*/)
			procSetCursor.Call(c)
			return 1
		}
	case wmMouseMove:
		in := inRect(barKey(b.dpi), lp)
		if in != b.hover {
			b.hover = in
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		tme := struct {
			size, flags uint32
			hwnd        uintptr
			hover       uint32
		}{flags: 0x2 /*TME_LEAVE*/, hwnd: hwnd}
		tme.size = uint32(unsafe.Sizeof(tme))
		procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
		return 0
	case wmMouseLeave:
		if b.hover || b.press {
			b.hover, b.press = false, false
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case wmLButtonDown:
		b.press = inRect(barKey(b.dpi), lp) && !barPressed
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0
	case wmLButtonUp:
		if b.press && inRect(barKey(b.dpi), lp) {
			barPressed = true
			barMu.Lock()
			leave := barLeave
			barMu.Unlock()
			if leave != nil {
				go leave()
			}
			for h := range bars { // every bar says it's leaving
				procInvalidateRect.Call(h, 0, 0)
			}
		}
		b.press = false
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0
	case wmPaint:
		paintBar(hwnd, b)
		return 0
	case wmDestroy:
		delete(bars, hwnd)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

func paintBar(hwnd uintptr, bw *barWin) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	px := func(n int) int32 { return int32(n * bw.dpi / 96) }
	w, h := px(barW), px(barH)
	b := bw.drawn

	fill := func(r rect, rgb uint32) {
		br, _, _ := procCreateSolidBrush.Call(uintptr(colorRef(rgb)))
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), br)
		procDeleteObject.Call(br)
	}
	fill(rect{0, 0, w, h}, toastEdge)
	fill(rect{1, 1, w - 1, h - 1}, toastGround)

	// The gem: Lootnest's red diamond.
	gx, gy, gr := px(28), h/2, px(9)
	type pt struct{ x, y int32 }
	gem := []pt{{gx, gy - gr}, {gx + gr, gy}, {gx, gy + gr}, {gx - gr, gy}}
	pen, _, _ := procCreatePen.Call(0, 1, uintptr(colorRef(toastRed)))
	brush, _, _ := procCreateSolidBrush.Call(uintptr(colorRef(toastRed)))
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	procPolygon.Call(hdc, uintptr(unsafe.Pointer(&gem[0])), uintptr(len(gem)))
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	procDeleteObject.Call(pen)
	procDeleteObject.Call(brush)

	face, _ := windows.UTF16PtrFromString("Segoe UI")
	procSetBkMode.Call(hdc, 1 /*TRANSPARENT*/)
	text := func(s string, r rect, size, weight int, rgb uint32, center bool) {
		if s == "" {
			return
		}
		f, _, _ := procCreateFontW.Call(uintptr(uint32(-px(size))), 0, 0, 0, uintptr(weight), 0, 0, 0,
			1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, uintptr(unsafe.Pointer(face)))
		old, _, _ := procSelectObject.Call(hdc, f)
		procSetTextColor.Call(hdc, uintptr(colorRef(rgb)))
		u, _ := windows.UTF16FromString(s)
		const dtSingle, dtEllipsis, dtNoPrefix, dtVCenter, dtCenter = 0x20, 0x8000, 0x800, 0x4, 0x1
		flags := uintptr(dtSingle | dtEllipsis | dtNoPrefix | dtVCenter)
		if center {
			flags |= dtCenter
		}
		procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&r)), flags)
		procSelectObject.Call(hdc, old)
		procDeleteObject.Call(f)
	}
	k := barKey(bw.dpi)
	l, r := px(52), k.left-px(12)
	text(b.Title, rect{l, px(14), r, px(36)}, 15, 600, toastText, false)
	text(b.Line, rect{l, px(36), r, px(58)}, 12, 400, toastMuted, false)

	// The key: Racing Red with near-black text (the one primary key), a
	// little lighter under the pointer, a pixel down while pressed.
	ground, label := uint32(toastRed), b.Key
	switch {
	case barPressed:
		ground, label = toastBadge, b.Busy
	case bw.hover:
		ground = 0xF2465C
	}
	kr := k
	if bw.press {
		kr.top, kr.bottom = kr.top+1, kr.bottom+1
	}
	kp, _, _ := procCreatePen.Call(0, 1, uintptr(colorRef(ground)))
	kb, _, _ := procCreateSolidBrush.Call(uintptr(colorRef(ground)))
	oldPen, _, _ = procSelectObject.Call(hdc, kp)
	oldBrush, _, _ = procSelectObject.Call(hdc, kb)
	procRoundRect.Call(hdc, uintptr(kr.left), uintptr(kr.top), uintptr(kr.right), uintptr(kr.bottom), uintptr(px(10)), uintptr(px(10)))
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	procDeleteObject.Call(kp)
	procDeleteObject.Call(kb)
	ink := uint32(0x111217) // Obsidian on the red
	if barPressed {
		ink = toastMuted
	}
	text(label, kr, 13, 700, ink, true)
}
