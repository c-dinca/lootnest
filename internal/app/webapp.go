//go:build windows

package app

import (
	"path/filepath"

	webview2 "github.com/jchv/go-webview2"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// A site as an app (YouTube's TV interface): `lootnest.exe --app <id>` is
// a small program of its own that shows the site full screen in a
// WebView2 window, with its own sign-in kept under Lootnest's data and
// nothing of the user's browser in it. It's a separate process from the
// Lootnest that swaps files, so TV mode follows and closes it as any
// other app (tv.go, apiLaunchApp).
//
// A browser started with a TV's name wasn't enough: on the owner's PC
// Edge opened the desktop site, and its kiosk window forgets the sign-in.
// Here the name is set through WebView2's own setting.

// webAppClass is the window class of a site shown as an app.
const webAppClass = "LootnestApp"

// padKeys is added to every page of the site: the TV interface reads
// arrows, Enter and Escape and knows nothing of controllers, so the
// controller's presses are asked of Lootnest as real key presses (ln_key).
// Every pad counts as one (a PlayStation pad and its Xbox twin would
// press twice otherwise). A held direction repeats; OK and Back don't.
const padKeys = `(function () {
  if (window.top !== window || window.__lnPad) return;
  window.__lnPad = true;
  var MAP = { 12: "up", 13: "down", 14: "left", 15: "right", 0: "ok", 1: "back" };
  var held = {}, next = {};
  function tick(t) {
    var now = {}, pads = navigator.getGamepads ? navigator.getGamepads() : [];
    for (var i = 0; i < pads.length; i++) {
      var p = pads[i];
      if (!p || !p.connected) continue;
      for (var b in MAP) if (p.buttons[b] && p.buttons[b].pressed) now[MAP[b]] = true;
      var x = p.axes[0] || 0, y = p.axes[1] || 0;
      if (Math.abs(x) > 0.6 || Math.abs(y) > 0.6) now[Math.abs(x) > Math.abs(y) ? (x > 0 ? "right" : "left") : (y > 0 ? "down" : "up")] = true;
    }
    for (var k in now) {
      if (!held[k]) { window.ln_key(k); next[k] = t + 400; }
      else if (k !== "ok" && k !== "back" && t >= next[k]) { window.ln_key(k); next[k] = t + 120; }
    }
    held = now;
    requestAnimationFrame(tick);
  }
  requestAnimationFrame(tick);
})();`

// webAppKeys are the only keys a page can ask for.
var webAppKeys = map[string]uint16{"up": 0x26, "down": 0x28, "left": 0x25, "right": 0x27, "ok": 0x0D, "back": 0x1B}

// RunWebApp shows the site with this id until its window is closed, on
// the monitor the window `on` is on (Lootnest's own), and returns the
// process's exit code.
func RunWebApp(id string, on uintptr) int {
	site, ok := core.WebAppByID(id)
	if !ok {
		return 2
	}
	win.EnableDPIAwareness()
	dpi := win.SystemDPI()
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(DataDir(), "apps", site.ID),
		UserAgent: site.UserAgent,
		// Only the site's own pages may ask for a key press.
		AllowMessage: site.Owns,
		WindowOptions: webview2.WindowOptions{
			Title:      site.Name,
			IconId:     1,
			ClassName:  webAppClass,
			Hidden:     true,
			Background: 0x0F0F0F,
			OnCreate:   func(hwnd uintptr) { win.SetWindowIcons(hwnd, dpi) },
		},
	})
	if w == nil {
		return 1
	}
	hwnd := uintptr(w.Window())
	// A key is pressed only for this window: with another in front (the
	// Home menu, a notification) the controller isn't this site's.
	_ = w.Bind("ln_key", func(k string) {
		if vk, ok := webAppKeys[k]; ok && win.Foreground() == hwnd {
			win.TapKey(vk)
		}
	})
	w.Init(padKeys)
	win.FullscreenLike(hwnd, on)
	win.ForceForeground(hwnd)
	w.RefreshView()
	w.Navigate(site.URL)
	w.Run()
	return 0
}
