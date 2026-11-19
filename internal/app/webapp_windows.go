package app

import (
	"os"
	"path/filepath"
	"time"

	webview2 "github.com/jchv/go-webview2"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// RunWebApp shows the site with this id until its window is closed, on
// the monitor the window `on` is on (Lootnest's own), and returns the
// process's exit code.
func RunWebApp(id string, on uintptr) int {
	site, ok := core.WebAppByID(id)
	if !ok {
		return 2
	}
	if site.ID == "stremio" {
		if stop := startStremioServer(); stop != nil {
			defer stop()
		}
	}
	if site.Loopback {
		// Read by WebView2 when it starts this window's browser.
		os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", core.WebAppBrowserArgs)
	}
	plat.EnableDPIAwareness()
	dpi := plat.SystemDPI()
	asMouse := false
	// An ordinary site is magnified to be read from a sofa.
	var zoom float64
	if site.Width > 0 {
		_, w, _ := plat.ScreenOf(on)
		zoom = core.WebAppZoom(site.Width, int(w), dpi)
	}
	// A voice chat's own pages get the microphone; nothing else does.
	var mic func(string) bool
	if site.Mic {
		mic = site.Owns
	}
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus:  true,
		Microphone: mic,
		DataPath:   filepath.Join(DataDir(), "apps", site.ID),
		UserAgent:  site.UserAgent,
		Zoom:       zoom,
		// Only the site's own pages may ask for a key press.
		AllowMessage: site.Owns,
		WindowOptions: webview2.WindowOptions{
			Title:      site.Name,
			IconId:     1,
			ClassName:  webAppClass,
			Hidden:     true,
			Background: 0x0F0F0F,
			OnCreate:   func(hwnd uintptr) { plat.SetWindowIcons(hwnd, dpi) },
			// Lootnest says when the controller is a mouse over this window
			// (pointer.go), so its presses aren't also keys here.
			MessageHook: func(_, msg, wp, _ uintptr) (uintptr, bool) {
				if msg == wmPointer {
					asMouse = wp != 0
					return 0, true
				}
				return 0, false
			},
		},
	})
	if w == nil {
		return 1
	}
	hwnd := uintptr(w.Window())
	// A key is pressed only for this window: with another in front (the
	// Home menu, a notification) the controller isn't this site's.
	var second time.Time
	var pressed int
	_ = w.Bind("ln_key", func(k string) {
		vk, ok := webAppKeys[k]
		if !ok || asMouse || plat.Foreground() != hwnd {
			return
		}
		if now := time.Now(); now.Sub(second) >= time.Second {
			second, pressed = now, 0
		}
		if pressed++; pressed > webAppPace {
			return
		}
		plat.TapKey(vk)
	})
	if site.Nav == "spatial" {
		w.Init(padSpatial)
	} else {
		w.Init(padKeys)
	}
	plat.FullscreenLike(hwnd, on)
	plat.ForceForeground(hwnd)
	w.RefreshView()
	w.Navigate(site.URL)
	w.Run()
	return 0
}
