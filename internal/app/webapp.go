//go:build windows

package app

import (
	"path/filepath"
	"time"

	webview2 "github.com/jchv/go-webview2"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// A site as an app (YouTube's TV interface, Spotify's web player, Discord):
// `lootnest.exe --app <id>` is
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

// padSpatial is added to every page of an ordinary site (Spotify's web
// player), which has no way to be driven without a mouse: the D-pad or
// stick moves a focus of Lootnest's own to the nearest thing to choose in
// that direction, scrolling the list it's in when there's more that way.
// A chooses it: a real Enter for a link or a button (the user's own key
// press, so sound may start), a click for the rest. B closes what's on
// top, else goes back a page. X plays or pauses, LB and RB change track,
// through the keyboard's media keys.
const padSpatial = `(function () {
  if (window.top !== window || window.__lnPad) return;
  window.__lnPad = true;
  var SEL = 'a[href], button, input, select, textarea, [role="button"], [role="link"], [role="tab"], [role="menuitem"], [role="option"], [role="switch"], [role="checkbox"], [tabindex]:not([tabindex="-1"])';
  var cur = null, last = null;
  var st = document.createElement("style");
  st.textContent = "[data-ln-focus]{outline:4px solid #fff !important;outline-offset:3px !important;box-shadow:0 0 0 8px rgba(0,0,0,.55) !important;border-radius:6px;}";
  // The page isn't there yet when this runs; the style goes in with the first focus.
  function styled() { var root = document.head || document.documentElement; if (root && !st.isConnected) root.appendChild(st); }
  function rectOf(el) {
    if (!el.isConnected || el.disabled || el.getAttribute("aria-hidden") === "true") return null;
    var r = el.getBoundingClientRect();
    if (r.width < 6 || r.height < 6 || r.bottom <= 0 || r.top >= innerHeight || r.right <= 0 || r.left >= innerWidth) return null;
    var s = getComputedStyle(el);
    if (s.visibility === "hidden" || s.pointerEvents === "none" || +s.opacity === 0) return null;
    // Covered by something else (a bar over the list): not there to choose.
    var x = Math.min(Math.max(r.left + r.width / 2, 1), innerWidth - 1), y = Math.min(Math.max(r.top + r.height / 2, 1), innerHeight - 1);
    var top = document.elementFromPoint(x, y);
    if (top && top !== el && !el.contains(top) && !top.contains(el)) return null;
    return r;
  }
  // What's on top: an open dialog or menu keeps the focus inside it.
  function scope() {
    var tops = document.querySelectorAll('[role="dialog"], [role="menu"], [role="listbox"]');
    for (var i = tops.length - 1; i >= 0; i--) if (tops[i].getClientRects().length && tops[i].querySelector(SEL)) return tops[i];
    return document;
  }
  function things() {
    var out = [], els = scope().querySelectorAll(SEL);
    for (var i = 0; i < els.length; i++) { var r = rectOf(els[i]); if (r) out.push({ el: els[i], r: r }); }
    // A link wrapped round a card and the button inside it: keep the inner ones too; they're separate things.
    return out;
  }
  function mark(el) {
    if (cur && cur !== el) cur.removeAttribute("data-ln-focus");
    cur = el;
    if (!el) return;
    styled();
    el.setAttribute("data-ln-focus", "");
    last = el.getBoundingClientRect();
    if (!el.matches("a[href], button, input, select, textarea, [tabindex]")) el.setAttribute("tabindex", "-1");
    try { el.focus({ preventScroll: true }); } catch (e) {}
    el.scrollIntoView({ block: "nearest", inline: "nearest" });
  }
  function scrollerOf(el) {
    for (var n = el; n && n !== document.documentElement; n = n.parentElement) {
      var s = getComputedStyle(n);
      if (/(auto|scroll)/.test(s.overflowY) && n.scrollHeight > n.clientHeight + 4) return n;
    }
    return document.scrollingElement || document.documentElement;
  }
  function move(dir, again) {
    var list = things();
    if (!list.length) return;
    var from = cur && cur.isConnected && rectOf(cur) ? cur.getBoundingClientRect() : null;
    if (!from) {
      // Nothing in focus yet (or it went): the thing nearest where the focus was, else the first in the middle.
      var px = last ? last.left + last.width / 2 : innerWidth / 2, py = last ? last.top + last.height / 2 : innerHeight / 3;
      var b0 = null, d0 = Infinity;
      list.forEach(function (t) { var d = Math.hypot(t.r.left + t.r.width / 2 - px, t.r.top + t.r.height / 2 - py); if (d < d0) { d0 = d; b0 = t; } });
      mark(b0.el);
      return;
    }
    var horiz = dir === "left" || dir === "right";
    var cx = from.left + from.width / 2, cy = from.top + from.height / 2, best = null, bs = Infinity;
    // Up and down stay in the list being scrolled while it has more that
    // way; only at its end does the focus leave it for a bar beside it.
    var sc = scrollerOf(cur), more = false;
    if (!horiz) more = dir === "down" ? sc.scrollTop + sc.clientHeight < sc.scrollHeight - 4 : sc.scrollTop > 4;
    if (more) list = list.filter(function (t) { return sc.contains(t.el); });
    list.forEach(function (t) {
      if (t.el === cur || t.el.contains(cur)) return;
      var r = t.r, x = r.left + r.width / 2, y = r.top + r.height / 2;
      var main = dir === "left" ? from.left - r.right : dir === "right" ? r.left - from.right : dir === "up" ? from.top - r.bottom : r.top - from.bottom;
      if (main < -Math.min(horiz ? from.width : from.height, horiz ? r.width : r.height) / 2) return; // not in that direction
      if (main < 0) main = 0;
      // On the same line counts most: overlap across the direction.
      var over = horiz ? Math.min(from.bottom, r.bottom) - Math.max(from.top, r.top) : Math.min(from.right, r.right) - Math.max(from.left, r.left);
      var cross = horiz ? Math.abs(y - cy) : Math.abs(x - cx);
      if (main === 0 && over <= 0) return;
      var score = main + (over > 0 ? cross * 0.3 : cross * 2.5 + 400);
      if (cur.contains(t.el)) score += 30; // inside the same card comes after the neighbours on its line
      if (score < bs) { bs = score; best = t; }
    });
    if (best) { mark(best.el); return; }
    // Nothing further that way on the screen: bring more of the list in.
    var by = Math.round(innerHeight * 0.5);
    if (horiz || again) return;
    sc.scrollBy({ top: dir === "down" ? by : -by });
    setTimeout(function () { move(dir, true); }, 260); // and on to what came in
  }
  function act(k) {
    if (k === "up" || k === "down" || k === "left" || k === "right") { move(k); return; }
    if (k === "ok") {
      if (!cur || !cur.isConnected) { move("down"); return; }
      // A real Enter for what takes one (it counts as the user's, so sound may start); a click for the rest.
      if (cur.matches("a[href], button, input, select, textarea")) window.ln_key("ok");
      else cur.click();
      return;
    }
    if (k === "back") {
      if (scope() !== document) window.ln_key("back"); // closes the dialog or menu on top
      else if (!window.navigation || window.navigation.canGoBack) history.back();
      return;
    }
    window.ln_key(k); // play, next, prev
  }
  var MAP = { 12: "up", 13: "down", 14: "left", 15: "right", 0: "ok", 1: "back", 2: "play", 4: "prev", 5: "next" };
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
      var dir = k === "up" || k === "down" || k === "left" || k === "right";
      if (!held[k]) { act(k); next[k] = t + 400; }
      else if (dir && t >= next[k]) { act(k); next[k] = t + 140; }
    }
    held = now;
    requestAnimationFrame(tick);
  }
  requestAnimationFrame(tick);
})();`

// webAppKeys are the only keys a page can ask for: the arrows, Enter,
// Escape, and the keyboard's media keys.
var webAppKeys = map[string]uint16{"up": 0x26, "down": 0x28, "left": 0x25, "right": 0x27, "ok": 0x0D, "back": 0x1B,
	"play": 0xB3, "next": 0xB0, "prev": 0xB1}

// webAppPace is the most key presses a page gets in a second: more than
// a held direction's repeat, far fewer than a page gone wrong would ask.
const webAppPace = 30

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
	win.EnableDPIAwareness()
	dpi := win.SystemDPI()
	asMouse := false
	// An ordinary site is magnified to be read from a sofa.
	var zoom float64
	if site.Width > 0 {
		_, w, _ := win.ScreenOf(on)
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
			OnCreate:   func(hwnd uintptr) { win.SetWindowIcons(hwnd, dpi) },
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
		if !ok || asMouse || win.Foreground() != hwnd {
			return
		}
		if now := time.Now(); now.Sub(second) >= time.Second {
			second, pressed = now, 0
		}
		if pressed++; pressed > webAppPace {
			return
		}
		win.TapKey(vk)
	})
	if site.Nav == "spatial" {
		w.Init(padSpatial)
	} else {
		w.Init(padKeys)
	}
	win.FullscreenLike(hwnd, on)
	win.ForceForeground(hwnd)
	w.RefreshView()
	w.Navigate(site.URL)
	w.Run()
	return 0
}
