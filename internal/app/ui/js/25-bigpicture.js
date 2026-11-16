  // ----------------------------------------------------------- big picture --
  // Big Picture: Lootnest as a console's screen, on the whole screen, for a
  // controller or a TV's remote from the sofa. This part holds its state, TV
  // mode, the controller and the sounds; its screens and how the focus moves
  // are in 26-bp-screens.js. The stick or the D-pad moves, A chooses, B goes
  // back a step, Y plays as someone else, LB and RB change section, Start
  // opens the menu (it leaves, outside TV mode). The keyboard works the
  // same: arrows, Enter, Escape, Q and E. Everything Y, X, LB, RB and Start
  // do can also be reached with the arrows alone.
  const BP = { on: false, tab: "home", tag: "", tagFrom: "", detail: null, choose: null, focus: "", from: "", timer: 0, sounds: pref("bpSounds", "1") === "1", tv: false, menu: false, menuWhy: "", menuFrom: "", who: false, mem: {}, sx: {} };
  const BP_TABS = [["home", "Home"], ["library", "Library"], ["achievements", "Achievements"], ["profiles", "Profiles"]];
  const BP_SEC = { home: "home", library: "library", achievements: "ach", profiles: "profiles" };
  const bpItem = (k) => libraryItems().find((x) => x.kind + ":" + x.id === k) || null;
  const bpKey = (it) => it.kind + ":" + it.id;
  function bpOpen() {
    if (BP.on) return;
    BP.on = true; BP.tab = "home"; BP.tag = ""; BP.tagFrom = ""; BP.detail = null; BP.points = false; BP.choose = null; BP.focus = ""; BP.from = ""; BP.menu = false; BP.ask = null; BP.mem = {}; BP.sx = {};
    closeContext(); gqClose();
    if (!S.allAch) api.allAchievements().catch(() => {});
    api.bigPicture(true).catch(() => {});
    if (!pad.raf && padsNow().length) pad.raf = requestAnimationFrame(padLoop); // a pad that was there before the page: its signs show at once
    renderBP(true);
    sfx("open");
  }
  function bpClose() {
    if (!BP.on) return;
    if (BP.tv) { bpMenu(!BP.menu); return; } // TV mode stays until Exit TV mode
    bpShut();
  }
  function bpShut() {
    if (!BP.on) return;
    BP.on = false;
    const el = $("#bp");
    if (el) { el.removeAttribute("id"); el.classList.add("out"); setTimeout(() => el.remove(), 320); }
    bpShown = "";
    api.bigPicture(false).catch(() => {});
    sfx("close");
    if (behindBP) render();
  }
  // TV mode (Lootnest Link): Big Picture on a TV for a streaming host,
  // which ends the stream when TV mode ends. Who's playing comes first when
  // more than one person uses the PC; Start opens a menu instead of leaving.
  function tvOpen() {
    const was = BP.tv;
    BP.tv = true; BP.menu = false;
    document.documentElement.classList.add("is-tv");
    if (BP.on) { if (!was) { const el = $("#bp"); if (el) el.innerHTML = ""; } }
    else bpOpen();
    if (!was && !solo() && cfg().players.length > 1) { BP.tab = "profiles"; BP.who = true; BP.detail = null; BP.choose = null; BP.focus = ""; }
    renderBP(true);
  }
  function tvClose() {
    if (!BP.tv) return;
    BP.tv = false; BP.menu = false; BP.who = false; BP.launch = null; drawLaunch(); BP.tvMenu = null; drawTVMenu(); BP.playing = "";
    document.documentElement.classList.remove("is-tv");
    bpShut();
  }
  // A game starting from TV mode: its art fills the screen while it loads
  // (Lootnest stays behind it, app/tv.go), and fades back when it closes.
  function tvLaunch(it) {
    if (!BP.tv || !it) return;
    BP.launch = { key: bpKey(it), name: it.name, art: launchArt(it), phase: "starting", at: Date.now() };
    drawLaunch();
    launchWatch();
    // Art still loading: in once it's there.
    if (!BP.launch.art) setTimeout(() => {
      const L = BP.launch, el = $("#bp-launch");
      if (!L || L.key !== bpKey(it) || !el) return;
      const a = launchArt(it);
      if (a) { L.art = a; el.querySelector(".bp-launch-art").innerHTML = artImg(a, "hero"); }
    }, 700);
  }
  const launchArt = (it) => { const w = bpArt(it, true); if (w.url) return w; const c = bpArt(it, false); return c.url ? c : null; };
  // A game Lootnest can't follow (no process name) never says it closed:
  // Big Picture comes back when it has the focus again a while after.
  let launchWatching = false;
  function launchWatch() {
    if (launchWatching) return;
    launchWatching = true;
    window.addEventListener("focus", () => { if (BP.launch && Date.now() - BP.launch.at > 15000) { tvReturn(); if (BP.on) renderBP(false); } });
    setInterval(() => { if (BP.launch && BP.launch.phase === "starting" && document.hasFocus() && Date.now() - BP.launch.at > 120000) tvReturn(); }, 5000);
  }
  function tvGame(ev) {
    if (!BP.tv) return;
    if (!BP.launch || (ev.key && BP.launch.key && !BP.launch.key.endsWith(ev.key) && !ev.key.endsWith(BP.launch.key))) {
      const it = everyItem().find((x) => bpKey(x) === ev.key || (x.g && "game:" + x.g.id === ev.key) || (x.g && x.g.id === ev.key) || x.id === ev.key);
      if (it) { tvLaunch(it); }
      else { BP.launch = { key: ev.key, name: ev.name || "", art: null, phase: "starting", at: Date.now() }; drawLaunch(); launchWatch(); }
    }
    BP.launch.phase = ev.phase || BP.launch.phase;
    if (ev.phase === "playing") BP.playing = ev.key;
    drawLaunch();
  }
  function drawLaunch() {
    const L = BP.launch;
    let el = $("#bp-launch");
    if (!L) { if (el) { el.classList.add("out"); setTimeout(() => el.remove(), 420); el.removeAttribute("id"); } return; }
    if (!el) {
      el = document.createElement("div"); el.id = "bp-launch"; el.className = "bp-launch ln-root";
      el.setAttribute("role", "status"); el.setAttribute("aria-live", "polite");
      el.innerHTML = `<div class="bp-launch-art">${L.art ? artImg(L.art, "hero") : ""}</div><div class="bp-launch-veil"></div>
        <div class="bp-launch-body"><div class="bp-launch-name"></div><div class="bp-launch-state"><span class="bp-launch-bar"></span><span class="bp-launch-word"></span></div></div>`;
      document.body.appendChild(el);
    }
    el.dataset.phase = L.phase;
    el.querySelector(".bp-launch-name").textContent = L.name;
    el.querySelector(".bp-launch-word").textContent = L.phase === "playing" ? "Playing" : "Starting";
  }
  const sameGame = (it, key) => !!it && !!key && (bpKey(it) === key || it.id === key || (!!it.g && (it.g.id === key || "game:" + it.g.id === key)));
  // TV mode's menu over a game, on the controller's Home or PS button.
  const TV_MENU = [["resume", "Resume game"], ["home", "Lootnest"], ["close", "Close the game"]];
  function tvMenu(ev) {
    if (!ev.on) { BP.tvMenu = null; drawTVMenu(); return; }
    const it = everyItem().find((x) => sameGame(x, ev.key));
    BP.tvMenu = { key: ev.key, name: it ? it.name : (BP.launch && BP.launch.name) || "", at: 0, sure: false };
    sfx("open");
    drawTVMenu();
  }
  function drawTVMenu() {
    const M = BP.tvMenu;
    let el = $("#tv-menu");
    if (!M) { if (el) { el.classList.add("out"); el.removeAttribute("id"); setTimeout(() => el.remove(), 320); } return; }
    if (!el) { el = document.createElement("div"); el.id = "tv-menu"; el.className = "tv-menu ln-root"; el.setAttribute("role", "menu"); document.body.appendChild(el); }
    el.innerHTML = `<div class="tv-menu-panel"><div class="tv-menu-name">${esc(M.name)}</div>
      ${TV_MENU.map(([k, l], i) => `<button class="tv-menu-item${i === M.at ? " on" : ""}" role="menuitem" data-tvm="${k}" tabindex="-1">${k === "close" && M.sure ? "Press again to close it" : esc(l)}</button>`).join("")}
      <div class="tv-menu-foot">${glyph("A")}Select<span></span>${glyph("B")}Resume</div></div>`;
    const on = el.querySelector(".tv-menu-item.on");
    if (on) on.focus({ preventScroll: true });
  }
  function tvMenuGo(k) {
    const M = BP.tvMenu;
    if (!M) return;
    if (k === "resume") { sfx("back"); BP.tvMenu = null; drawTVMenu(); api.tvResume().catch(() => {}); return; }
    if (k === "home") { sfx("go"); BP.tvMenu = null; drawTVMenu(); tvReturn(); api.tvStay().catch(() => {}); if (BP.on) renderBP(false); return; }
    if (k === "close") {
      if (!M.sure) { M.sure = true; sfx("toggle"); drawTVMenu(); return; }
      sfx("go"); BP.tvMenu = null; drawTVMenu();
      api.tvCloseGame().catch((e) => toast(errText(e), "error"));
    }
  }
  document.addEventListener("click", (e) => { const b = e.target.closest && e.target.closest("#tv-menu [data-tvm]"); if (b) tvMenuGo(b.dataset.tvm); });
  // The game closed: Big Picture comes back from behind its art.
  function tvReturn() {
    if (!BP.launch) return false;
    BP.launch = null;
    drawLaunch();
    return true;
  }
  // The menu, over whatever screen is up: Start in TV mode, the key at the
  // end of the bar, and Back on Home, which asks before leaving.
  function bpMenu(on, why) {
    if (on && (S.picker || S.dialog || BP.ask)) return;
    if (on === BP.menu) return;
    if (on) BP.menuFrom = BP.focus;
    BP.menu = on; BP.menuWhy = on ? why || "" : ""; BP.focus = on ? "menu:back" : BP.menuFrom || "";
    sfx(on ? "open" : "back");
    renderBP(false);
  }
  function bpSoon() { if (BP.timer) return; BP.timer = requestAnimationFrame(() => { BP.timer = 0; if (BP.on) renderBP(false); }); }
  // The buttons, named by what they do here as on an Xbox pad, the way each
  // kind of pad labels them. Face buttons by position: A bottom, B right, X
  // left, Y top; a Nintendo pad confirms with its right-hand A (padRole).
  const PADS = {
    xbox: { A: "A", B: "B", X: "X", Y: "Y", LB: "LB", RB: "RB", Start: "Menu", View: "View" },
    playstation: { A: "cross", B: "circle", X: "square", Y: "triangle", LB: "L1", RB: "R1", Start: "Options", View: "Create" },
    nintendo: { A: "A", B: "B", X: "Y", Y: "X", LB: "L", RB: "R", Start: "+", View: "\u2212" },
    generic: { A: "bottom", B: "right", X: "left", Y: "top", LB: "L", RB: "R", Start: "Start", View: "Select" },
  };
  const dot = (x, y, on) => `<circle cx="${x}" cy="${y}" r="${on ? 3 : 1.5}" fill="currentColor" stroke="none"/>`;
  const PAD_SHAPES = {
    cross: '<path d="M5.5 5.5l9 9M14.5 5.5l-9 9"/>', circle: '<circle cx="10" cy="10" r="5"/>',
    square: '<rect x="5.25" y="5.25" width="9.5" height="9.5" rx=".5"/>', triangle: '<path d="M10 4.75 15.5 14.5h-11z"/>',
    ...Object.fromEntries(["top", "right", "bottom", "left"].map((k) => [k, dot(10, 4.5, k === "top") + dot(15.5, 10, k === "right") + dot(10, 15.5, k === "bottom") + dot(4.5, 10, k === "left")])),
  };
  const pad = { prev: [], held: "", next: 0, raf: 0, fam: "", hello: 0, joined: [] };
  const glyph = (b) => {
    const f = pad.fam || "xbox", v = PADS[f][b], face = b.length === 1;
    const cls = f === "xbox" || !face ? " bp-g--" + b.toLowerCase() : "";
    return `<span class="bp-g${cls}" aria-hidden="true">${PAD_SHAPES[v] ? gsvg(PAD_SHAPES[v]) : esc(v)}</span>`;
  };
  // The same over the picker or a dialog above Big Picture: its keys, the
  // main choices first.
  function overlayMove(dir) {
    const o = $("#overlay"), a = document.activeElement;
    const els = [...o.querySelectorAll("button:not([disabled]), input, select")].filter((e) => e.getClientRects().length);
    if (!els.length) return;
    const cur = a && a !== o && o.contains(a) && els.includes(a) ? a : null;
    const best = cur ? nearest(els, cur, dir) : o.querySelector(".slot:not([disabled]), .choice, .ln-button--primary") || els[0];
    if (!best) { sfx("edge"); return; }
    best.focus({ preventScroll: true });
    best.scrollIntoView({ block: "nearest", inline: "nearest" });
    sfx("move");
  }
  // The nearest element in a direction, as on a console: in the same row first.
  function nearest(els, cur, dir) {
    const r = cur.getBoundingClientRect(), cx = r.left + r.width / 2, cy = r.top + r.height / 2;
    const horiz = dir === "left" || dir === "right";
    let best = null, bs = Infinity;
    for (const el of els) {
      if (el === cur) continue;
      const q = el.getBoundingClientRect(), x = q.left + q.width / 2, y = q.top + q.height / 2;
      const dx = x - cx, dy = y - cy;
      const main = { left: -dx, right: dx, up: -dy, down: dy }[dir];
      if (main <= 4) continue;
      // In the same row first: the line it's on counts more than distance.
      const cross = horiz ? Math.abs(dy) : Math.abs(dx);
      const sameLine = horiz ? Math.abs(dy) < r.height / 2 : true;
      const score = main + cross * (horiz ? 4 : 1.1) - (sameLine && horiz ? 1000 : 0);
      if (score < bs) { bs = score; best = el; }
    }
    return best;
  }
  // Tab stays inside what's on top: the picker, a dialog or drawer, or Big Picture.
  document.addEventListener("keydown", (ev) => {
    if (ev.key !== "Tab") return;
    const root = S.picker ? $("#overlay .picker") : S.dialog ? $("#overlay .dialog, #overlay .welcome, #overlay .ln-drawer") : BP.on ? $("#bp") : null;
    if (!root) return;
    const els = [...root.querySelectorAll("button, input, select, textarea, [tabindex]")].filter((e) => !e.disabled && e.tabIndex >= 0 && e.getClientRects().length);
    const a = document.activeElement, first = els[0], last = els[els.length - 1];
    if (!els.length) ev.preventDefault();
    else if (!root.contains(a)) { ev.preventDefault(); (ev.shiftKey ? last : first).focus(); }
    else if (ev.shiftKey && (a === first || a === root)) { ev.preventDefault(); last.focus(); }
    else if (!ev.shiftKey && a === last) { ev.preventDefault(); first.focus(); }
  }, true);

  // The controller: read every frame while one is connected. Start (or the
  // PS button) opens Big Picture from anywhere in Lootnest; in it, the stick
  // and the D-pad move (held, they repeat), A chooses, B goes back. The
  // glyphs follow the pad that last sent a press.
  // Vendor ids first: many pads, Xbox ones included, call themselves "Wireless Controller".
  const PAD_IDS = [[/vendor: ?045e/, "xbox"], [/vendor: ?054c/, "playstation"], [/vendor: ?057e/, "nintendo"],
    [/xbox|xinput/, "xbox"], [/dualsense|dualshock|playstation/, "playstation"], [/pro controller|joy-con|nintendo/, "nintendo"]];
  const padFamily = (id) => { const s = (id || "").toLowerCase(), m = PAD_IDS.find(([re]) => re.test(s)); return m ? m[1] : "generic"; };
  const realPad = (g) => padFamily(g.id) !== "xbox";
  // The standard mapping's buttons; 16 is the PS or Home button, 17 the touchpad.
  const PAD_ROLES = { 0: "A", 1: "B", 2: "X", 3: "Y", 4: "LB", 5: "RB", 8: "View", 9: "Start", 16: "Start", 17: "View" };
  // In TV mode the Home or PS button is Lootnest's own (app/tv.go tvHome).
  const padRole = (fam, i) => (BP.tv && i === 16 ? "" : fam === "nintendo" && i < 2 ? (i ? "A" : "B") : i === 17 && fam !== "playstation" ? "" : PAD_ROLES[i] || "");
  const padDir = (g) => {
    const b = (i) => !!(g.buttons[i] && g.buttons[i].pressed), ax = g.axes[0] || 0, ay = g.axes[1] || 0;
    return b(12) || ay < -0.5 ? "up" : b(13) || ay > 0.5 ? "down" : b(14) || ax < -0.5 ? "left" : b(15) || ax > 0.5 ? "right" : "";
  };
  const padsNow = () => (navigator.getGamepads ? [...navigator.getGamepads()].filter(Boolean) : []);
  function padLoop(t) {
    pad.raf = 0;
    const pads = padsNow();
    if (!pads.length) return;
    // DS4Windows, Steam Input and x360ce add an Xbox twin of another pad.
    // While that pad holds a button, an Xbox pad's presses are its twin's echo
    // and are left out; each role acts once a frame. The glyphs stay the real
    // pad's, as the twin may be the only one that sends input.
    const real = pads.filter(realPad);
    const echo = real.some((g) => g.buttons.some((x) => x.pressed));
    const roles = new Set();
    let fam = pad.fam || padFamily((real[0] || pads[0]).id), dir = "";
    for (const g of pads) {
      const f = padFamily(g.id), now = g.buttons.map((x) => x.pressed), was = pad.prev[g.index] || [];
      pad.prev[g.index] = now;
      if (f === "xbox" && echo) continue;
      now.forEach((p, i) => { if (p && !was[i]) { if (f !== "xbox" || !real.length) fam = f; const r = padRole(f, i); if (r) roles.add(r); } });
      dir = dir || padDir(g);
    }
    if (document.hasFocus() && BP.tvMenu) {
      const M = BP.tvMenu;
      if (roles.has("B")) tvMenuGo("resume");
      else if (roles.has("A")) tvMenuGo(TV_MENU[M.at][0]);
      else if ((dir === "up" || dir === "down") && dir !== pad.held) { M.at = (M.at + (dir === "down" ? 1 : TV_MENU.length - 1)) % TV_MENU.length; M.sure = false; sfx("move"); drawTVMenu(); }
    } else if (document.hasFocus() && BP.launch) {
      // While a game starts, B comes back to Big Picture; nothing else acts.
      if (roles.has("B") && BP.launch.phase === "starting") { sfx("back"); tvReturn(); }
    } else if (document.hasFocus()) {
      if (fam !== pad.fam) { pad.fam = fam; if (BP.on) renderBP(false); }
      if (roles.has("Start") && !(BP.tv && (S.picker || S.dialog))) { if (BP.on) bpClose(); else bpOpen(); }
      if (BP.on && !S.picker && !S.dialog) {
        if (roles.has("A")) bpSelect(document.activeElement && document.activeElement.closest && document.activeElement.closest("#bp [data-go]"));
        if (roles.has("B")) bpBack();
        if (roles.has("X")) bpX();
        if (roles.has("Y")) bpY();
        if (roles.has("LB")) bpTab(-1);
        if (roles.has("RB")) bpTab(1);
        if (roles.has("View")) bpSelect({ dataset: { go: "sound" } });
        if (dir && (dir !== pad.held || t >= pad.next)) { bpMove(dir); pad.next = t + (dir !== pad.held ? 400 : 120); }
      } else if (BP.on) { // the picker (Who's playing, its questions, the swap) or a dialog above Big Picture
        const o = $("#overlay"), a = document.activeElement;
        if (roles.has("A") && a !== o && o.contains(a)) a.click();
        if (roles.has("B")) {
          if (S.picker) { if (!S.picker.working) { const back = o.querySelector('[data-action="picker-back"]'); if (back) back.click(); else closePicker(); } }
          else if (S.dialog.kind !== "welcome") closeDialog(false);
        }
        if (dir && (dir !== pad.held || t >= pad.next)) { overlayMove(dir); pad.next = t + (dir !== pad.held ? 400 : 120); }
      }
      pad.held = dir;
    }
    pad.raf = requestAnimationFrame(padLoop);
  }
  window.addEventListener("gamepadconnected", (ev) => {
    if (!pad.raf) pad.raf = requestAnimationFrame(padLoop);
    // The next frame picks the glyphs again. A twin connects with its pad:
    // one note for both, named for the real one.
    pad.fam = "";
    if (ev.gamepad) pad.joined.push(ev.gamepad);
    clearTimeout(pad.hello);
    pad.hello = setTimeout(() => {
      const js = pad.joined, g = js.find(realPad) || js[js.length - 1];
      pad.joined = [];
      if (g) toast(`Controller connected. Press the ${PADS[padFamily(g.id)].Start} button for Big Picture.`);
    }, 300);
  });
  window.addEventListener("focus", () => { if (!pad.raf && padsNow().length) pad.raf = requestAnimationFrame(padLoop); });

  // Kenney's Interface Sounds (CC0), inlined at start as {name: base64}
  // like the fonts. Moving and choosing in Big Picture are off with the
  // button at its bottom; unlocks, save points and swaps with Settings.
  const SOUNDS = /*@SOUNDS*/{};
  const SFX = {
    move: ["tick_001", 0.3], edge: ["click_004", 0.2, 0.8], tab: ["select_007", 0.3], go: ["select_002", 0.35], back: ["back_002", 0.35],
    open: ["maximize_008", 0.3], close: ["minimize_008", 0.3], toggle: ["toggle_002", 0.2],
    unlock: ["confirmation_002", 0.3], rare: ["glass_001", 0.25], point: ["drop_002", 0.3], swap: ["confirmation_001", 0.3],
  };
  let actx = null, master = null;
  const sfxBufs = {}, sfxAt = {};
  function playSound(kind, scale = 1) {
    const [name, gain, rate] = SFX[kind] || [], b64 = name && SOUNDS[name];
    const now = performance.now();
    if (!b64 || now - (sfxAt[kind] || -1e9) < 45) return;
    sfxAt[kind] = now;
    try {
      if (!actx) { actx = new AudioContext(); master = actx.createGain(); master.gain.value = 0.7; master.connect(actx.destination); }
      if (actx.state === "suspended") actx.resume().catch(() => {});
      sfxBufs[name] = sfxBufs[name] || actx.decodeAudioData(Uint8Array.from(atob(b64), (c) => c.charCodeAt(0)).buffer);
      sfxBufs[name].then((buf) => {
        const src = actx.createBufferSource(), g = actx.createGain();
        src.buffer = buf; src.playbackRate.value = rate || 1; g.gain.value = gain * scale;
        src.connect(g).connect(master); src.start();
      }).catch(() => {});
    } catch (e) { /* no sound */ }
  }
  function sfx(kind) { if (BP.sounds && !document.hidden && !cfg().settings.soundsOff) playSound(kind); }
  // The window's own sounds, at half Big Picture's volume: one for each
  // thing you do (a page, a tab, a menu, Play), never while a game runs.
  // Moving between covers with the keys sounds at most every 120 ms.
  let uiAt = -1e9;
  function uiSound(kind) {
    const now = performance.now();
    if (BP.on || document.hidden || cfg().settings.soundsOff || now - uiAt < 80 || (kind === "move" && now - (sfxAt.move || -1e9) < 120)) return;
    if ((S.d.sessions || []).some((s) => s.phase === "playing")) return;
    uiAt = now;
    playSound(kind, 0.5);
  }
  // Go sends no unlocks for a game with an anti-cheat.
  function unlockSound(list) {
    if (cfg().settings.soundsOff) return;
    playSound("unlock");
    if (list.some((a) => tierOf(a.percent) === "ultra")) setTimeout(() => playSound("rare"), 120);
  }
  // Save points and swaps: never for a game with an anti-cheat, as with the
  // notes over games. A Steam profile game's folder is its scanned one's.
  function cue(kind, key) {
    if (cfg().settings.soundsOff) return;
    const it = everyItem().find((x) => recentKey(x) === key);
    const dir = it && (it.f ? it.f.dir : it.exe || ((S.found.games || []).find((f) => it.appId && f.appId === it.appId) || {}).dir);
    if (!dir) { playSound(kind); return; }
    const f = S.facts[key];
    (f ? Promise.resolve(f) : api.gameFacts(dir).then((x) => (S.facts[key] = x || {})))
      .then((x) => { if (!x.antiCheat) playSound(kind); }).catch(() => {});
  }

