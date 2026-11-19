package app

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
//
// A on a text field (a search box, a message box) opens a keyboard drawn
// in the page, since a TV has no keyboard and Windows' own doesn't come
// up for a controller (the owner, 19 Nov 2026, on Stremio's search): the
// D-pad moves over its keys, A types, X deletes, Y is a space, B closes
// it, and Start (or its Done key) closes it and presses Enter in the
// field, which searches. Letters go in as typed text (insertText), so the
// site sees them as it would a keyboard's.
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
  // A field that takes typed words.
  function textField(el) {
    if (!el) return false;
    if (el.isContentEditable || el.getAttribute("role") === "textbox") return true;
    if (el.tagName === "TEXTAREA") return !el.readOnly;
    if (el.tagName !== "INPUT" || el.readOnly) return false;
    return /^(text|search|email|url|tel|password|number|)$/.test((el.getAttribute("type") || "").toLowerCase());
  }
  // The keyboard: in a shadow root, so the site's styles don't reach it.
  var kb = (function () {
    var ROWS = [
      ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"],
      ["q", "w", "e", "r", "t", "y", "u", "i", "o", "p"],
      ["a", "s", "d", "f", "g", "h", "j", "k", "l", "@"],
      ["z", "x", "c", "v", "b", "n", "m", ",", ".", "-"],
      ["shift", "space", "del", "done"]
    ];
    var WORDS = { shift: "Shift", space: "Space", del: "Delete", done: "Done" };
    var host = null, keysEl = null, line = null, field = null, at = [1, 0], upper = false;
    function build() {
      host = document.createElement("div");
      host.setAttribute("data-ln-keyboard", "");
      host.style.cssText = "position:fixed;left:0;right:0;bottom:0;z-index:2147483647;";
      var root = host.attachShadow({ mode: "open" });
      var css = document.createElement("style");
      css.textContent = [
        ".pad{background:#181a21;border-top:1px solid rgba(255,255,255,.08);padding:22px 0 26px;display:flex;flex-direction:column;align-items:center;gap:14px;font:600 22px/1.2 system-ui,'Segoe UI',sans-serif;color:#dfe5eb;box-shadow:0 -20px 60px rgba(0,0,0,.5)}",
        ".line{min-width:min(900px,90vw);max-width:90vw;box-sizing:border-box;padding:12px 18px;border-radius:10px;background:#111217;color:#dfe5eb;font-weight:500;white-space:pre;overflow:hidden;text-overflow:ellipsis}",
        ".line:empty::before{content:'Type with the controller';color:#8c94a0}",
        ".keys{display:flex;flex-direction:column;gap:10px}",
        ".row{display:flex;gap:10px;justify-content:center}",
        ".k{min-width:68px;height:62px;padding:0 14px;box-sizing:border-box;border:0;border-radius:10px;background:#1f222b;color:#dfe5eb;font:inherit;box-shadow:inset 0 1px 0 rgba(255,255,255,.08);cursor:pointer}",
        ".k.wide{min-width:150px}.k.space{min-width:330px}",
        ".k.done{background:#ef233c;color:#111217}",
        ".k.on{background:#dfe5eb;color:#111217;transform:scale(1.06)}",
        ".k.done.on{outline:3px solid #dfe5eb;outline-offset:3px;background:#ef233c}",
        ".k.lit{box-shadow:inset 0 0 0 2px #dfe5eb}",
        ".hint{color:#8c94a0;font-weight:500;font-size:17px}"
      ].join("");
      var pad = document.createElement("div");
      pad.className = "pad";
      line = document.createElement("div");
      line.className = "line";
      keysEl = document.createElement("div");
      keysEl.className = "keys";
      var hint = document.createElement("div");
      hint.className = "hint";
      hint.textContent = "A types, X deletes, Y adds a space, B closes, Start searches";
      pad.appendChild(line);
      pad.appendChild(keysEl);
      pad.appendChild(hint);
      root.appendChild(css);
      root.appendChild(pad);
      // A mouse works too, without taking the focus from the field.
      keysEl.addEventListener("mousedown", function (e) { e.preventDefault(); });
      keysEl.addEventListener("click", function (e) {
        var b = e.target.closest(".k");
        if (!b) return;
        at = [+b.dataset.r, +b.dataset.c];
        paint();
        press();
      });
    }
    function paint() {
      keysEl.textContent = "";
      ROWS.forEach(function (row, r) {
        var d = document.createElement("div");
        d.className = "row";
        row.forEach(function (k, c) {
          var b = document.createElement("button");
          b.className = "k" + (WORDS[k] ? " wide " + k : "") + (r === at[0] && c === at[1] ? " on" : "") + (k === "shift" && upper ? " lit" : "");
          b.dataset.r = r;
          b.dataset.c = c;
          b.textContent = WORDS[k] || (upper ? k.toUpperCase() : k);
          d.appendChild(b);
        });
        keysEl.appendChild(d);
      });
      line.textContent = valueOf();
    }
    function valueOf() {
      if (!field) return "";
      var v = field.isContentEditable || field.getAttribute("role") === "textbox" ? field.innerText.replace(/\n$/, "") : field.value;
      return field.type === "password" ? v.replace(/./g, "•") : v;
    }
    function focusField() {
      try { field.focus({ preventScroll: true }); } catch (e) {}
    }
    function setValue(v) {
      var proto = field.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
      var d = Object.getOwnPropertyDescriptor(proto, "value");
      d.set.call(field, v);
      field.dispatchEvent(new Event("input", { bubbles: true }));
    }
    function type(ch) {
      focusField();
      var before = valueOf();
      if (!document.execCommand("insertText", false, ch) || valueOf() === before) {
        if (field.value !== undefined && !field.isContentEditable) setValue(field.value + ch);
      }
    }
    function erase() {
      focusField();
      var before = valueOf();
      if (!document.execCommand("delete", false) || valueOf() === before) {
        if (field.value !== undefined && !field.isContentEditable) setValue(field.value.slice(0, -1));
      }
    }
    function press() {
      var k = ROWS[at[0]][at[1]];
      if (k === "shift") upper = !upper;
      else if (k === "space") type(" ");
      else if (k === "del") erase();
      else if (k === "done") { done(); return; }
      else { type(upper ? k.toUpperCase() : k); if (upper && /[a-z]/.test(k)) upper = false; }
      paint();
    }
    function step(dir) {
      var r = at[0], c = at[1];
      if (dir === "left") c = Math.max(0, c - 1);
      else if (dir === "right") c = Math.min(ROWS[r].length - 1, c + 1);
      else {
        var to = dir === "up" ? r - 1 : r + 1;
        if (to < 0 || to >= ROWS.length) return;
        // The key in the next row nearest across, by where they sit.
        var rows = keysEl.children, from = rows[r].children[c].getBoundingClientRect(), x = from.left + from.width / 2, best = 0, bd = Infinity;
        Array.prototype.forEach.call(rows[to].children, function (b, i) {
          var q = b.getBoundingClientRect(), d = Math.abs(q.left + q.width / 2 - x);
          if (d < bd) { bd = d; best = i; }
        });
        r = to;
        c = best;
      }
      at = [r, c];
      paint();
    }
    function close() {
      if (host) host.remove();
      if (field && field.isConnected) mark(field);
      field = null;
    }
    function done() {
      var f = field;
      close();
      if (f && f.isConnected) { try { f.focus({ preventScroll: true }); } catch (e) {} window.ln_key("ok"); }
    }
    return {
      shown: function () { return !!(host && host.isConnected && field); },
      open: function (el) {
        if (!host) build();
        field = el;
        at = [1, 0];
        upper = false;
        focusField();
        (document.body || document.documentElement).appendChild(host);
        paint();
      },
      act: function (k) {
        if (field && !field.isConnected) { close(); return; }
        if (k === "up" || k === "down" || k === "left" || k === "right") step(k);
        else if (k === "ok") press();
        else if (k === "back") close();
        else if (k === "play") { erase(); paint(); }
        else if (k === "y") { type(" "); paint(); }
        else if (k === "start") done();
      }
    };
  })();
  function act(k) {
    if (k === "up" || k === "down" || k === "left" || k === "right") { move(k); return; }
    if (k === "ok") {
      if (!cur || !cur.isConnected) { move("down"); return; }
      if (textField(cur)) { kb.open(cur); return; }
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
    if (k === "play" || k === "next" || k === "prev") window.ln_key(k);
  }
  var MAP = { 12: "up", 13: "down", 14: "left", 15: "right", 0: "ok", 1: "back", 2: "play", 3: "y", 4: "prev", 5: "next", 9: "start" };
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
      var go = kb.shown() ? kb.act : act;
      if (!held[k]) { go(k); next[k] = t + 400; }
      else if ((dir || (kb.shown() && k === "play")) && t >= next[k]) { go(k); next[k] = t + 140; }
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
