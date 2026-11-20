//go:build windows

package win

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"lootnest/internal/core"
)

// The in-game counter, in this test's own process: the DLL Lootnest ships
// is loaded as it would be into a game, then frames are presented with
// Direct3D 11 (WARP), Direct3D 9 and OpenGL, and the shared block must say
// they were counted and the number drawn on them.

func comCall(obj uintptr, idx int, args ...uintptr) uintptr {
	vt := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vt + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func testWindow(t *testing.T) uintptr {
	cls, _ := windows.UTF16PtrFromString("STATIC")
	const wsOverlappedWindow, wsVisible = 0x00CF0000, 0x10000000
	h, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), 0, wsOverlappedWindow|wsVisible, 0, 0, 640, 360, 0, 0, 0, 0)
	if h == 0 {
		t.Fatalf("no window: %v", err)
	}
	return h
}

// pump keeps the window answering while frames are made.
func pump() {
	var m msgT
	const pmRemove = 1
	peek := user32.NewProc("PeekMessageW")
	for {
		r, _, _ := peek.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove)
		if r == 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// frames presents until the block's drawn count passes from, or 10 s.
func frames(t *testing.T, s *OverlayShare, from int32, present func()) {
	t.Helper()
	end := time.Now().Add(10 * time.Second)
	for time.Now().Before(end) {
		present()
		pump()
		if s.Int32(core.OverDrawn) > from+5 {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("not drawn: api %d, drawn %d, fps %d, hooked %d, error %d (%s)", s.Int32(core.OverAPI), s.Int32(core.OverDrawn),
		s.Int32(core.OverFPS), s.Int32(core.OverHooked), s.Int32(core.OverError), core.OverlayErrors[s.Int32(core.OverError)])
}

func TestOverlayDLLDraws(t *testing.T) {
	dll, err := filepath.Abs(filepath.Join("..", "app", "overlay", "lootnest-overlay64.dll"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dll); err != nil {
		t.Fatal(err)
	}
	s, err := NewOverlayShare(uint32(os.Getpid()), core.OverlayMagic, core.OverlayVersion, core.OverlaySize)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.SetInt32(core.OverOn, 1)
	s.SetInt32(core.OverCorner, 3)
	// The graphics libraries first, as a game has them when Lootnest comes in.
	d3d11 := windows.NewLazySystemDLL("d3d11.dll")
	d3d9 := windows.NewLazySystemDLL("d3d9.dll")
	gl := windows.NewLazySystemDLL("opengl32.dll")
	windows.NewLazySystemDLL("dxgi.dll").Load()
	for _, d := range []*windows.LazyDLL{d3d11, d3d9, gl} {
		if err := d.Load(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := windows.LoadLibrary(dll); err != nil {
		t.Fatalf("the DLL didn't load: %v", err)
	}
	for i := 0; i < 100 && s.Int32(core.OverHooked) != 7; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("hooked %d, error %d", s.Int32(core.OverHooked), s.Int32(core.OverError))

	t.Run("Direct3D 11", func(t *testing.T) {
		runtime.LockOSThread() // the window, its device and OpenGL's context live on one thread
		defer runtime.UnlockOSThread()
		w := testWindow(t)
		defer procDestroyWindow.Call(w)
		type swapDesc struct {
			Width, Height, RefreshN, RefreshD, Format, Scanline, Scaling uint32
			SampleCount, SampleQuality                                   uint32
			Usage, Count                                                 uint32
			Window                                                       uintptr
			Windowed, Effect, Flags                                      uint32
		}
		d := swapDesc{Width: 640, Height: 360, Format: 28, SampleCount: 1, Usage: 0x20, Count: 1, Window: w, Windowed: 1}
		var sc, dev, ctx uintptr
		create := d3d11.NewProc("D3D11CreateDeviceAndSwapChain")
		hr, _, _ := create.Call(0, 5 /*WARP*/, 0, 0, 0, 0, 7, uintptr(unsafe.Pointer(&d)),
			uintptr(unsafe.Pointer(&sc)), uintptr(unsafe.Pointer(&dev)), 0, uintptr(unsafe.Pointer(&ctx)))
		if int32(hr) < 0 || sc == 0 {
			t.Fatalf("no Direct3D 11 device: %#x", uint32(hr))
		}
		defer comCall(sc, 2)
		defer comCall(dev, 2)
		defer comCall(ctx, 2)
		frames(t, s, s.Int32(core.OverDrawn), func() { comCall(sc, 8, 0, 0) })
		if s.Int32(core.OverFPS) <= 0 {
			t.Errorf("no frames counted")
		}
		if api := s.Int32(core.OverAPI); api != 1 {
			t.Errorf("api %d, want 1 (Direct3D 11)", api)
		}
	})

	t.Run("Direct3D 9", func(t *testing.T) {
		runtime.LockOSThread() // the window, its device and OpenGL's context live on one thread
		defer runtime.UnlockOSThread()
		w := testWindow(t)
		defer procDestroyWindow.Call(w)
		var d3d uintptr
		if hr, _, _ := d3d9.NewProc("Direct3DCreate9Ex").Call(32, uintptr(unsafe.Pointer(&d3d))); int32(hr) < 0 || d3d == 0 {
			t.Skipf("no Direct3D 9Ex here: %#x", uint32(hr))
		}
		defer comCall(d3d, 2)
		type presentParams struct {
			Width, Height, Format, Count, MultiSample, MultiQuality, Effect uint32
			Window                                                          uintptr
			Windowed, AutoDepth, DepthFormat, Flags, Refresh, Interval      uint32
		}
		pp := presentParams{Effect: 1 /*DISCARD*/, Window: w, Windowed: 1}
		var dev uintptr
		const hal, softwareVP = 1, 0x20
		if hr := comCall(d3d, 20, 0, hal, w, softwareVP, uintptr(unsafe.Pointer(&pp)), 0, uintptr(unsafe.Pointer(&dev))); int32(hr) < 0 || dev == 0 {
			t.Skipf("no Direct3D 9 device here: %#x", uint32(hr))
		}
		defer comCall(dev, 2)
		frames(t, s, s.Int32(core.OverDrawn), func() { comCall(dev, 17, 0, 0, 0, 0) })
	})

	t.Run("OpenGL", func(t *testing.T) {
		runtime.LockOSThread() // the window, its device and OpenGL's context live on one thread
		defer runtime.UnlockOSThread()
		w := testWindow(t)
		defer procDestroyWindow.Call(w)
		gdi := windows.NewLazySystemDLL("gdi32.dll")
		dc, _, _ := user32.NewProc("GetDC").Call(w)
		var pfd [40]byte
		pfd[0], pfd[2] = 40, 1
		*(*uint32)(unsafe.Pointer(&pfd[4])) = 0x4 | 0x20 | 0x1 // draw to window, OpenGL, double buffer
		pfd[9] = 32
		pf, _, _ := gdi.NewProc("ChoosePixelFormat").Call(dc, uintptr(unsafe.Pointer(&pfd[0])))
		if pf == 0 {
			t.Skip("no pixel format")
		}
		gdi.NewProc("SetPixelFormat").Call(dc, pf, uintptr(unsafe.Pointer(&pfd[0])))
		rc, _, _ := gl.NewProc("wglCreateContext").Call(dc)
		if rc == 0 {
			t.Skip("no OpenGL context")
		}
		defer gl.NewProc("wglDeleteContext").Call(rc)
		gl.NewProc("wglMakeCurrent").Call(dc, rc)
		defer gl.NewProc("wglMakeCurrent").Call(0, 0)
		swap := gdi.NewProc("SwapBuffers")
		frames(t, s, s.Int32(core.OverDrawn), func() { swap.Call(dc) })
	})

	// Off: frames are still counted, nothing is drawn.
	s.SetInt32(core.OverOn, 0)
	if s.Int32(core.OverError) != 0 {
		t.Errorf("the DLL said: %s", core.OverlayErrors[s.Int32(core.OverError)])
	}
}
