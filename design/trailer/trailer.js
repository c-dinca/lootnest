// The Lootnest trailer: one page whose every frame is a function of the
// time. render.mjs asks for seek(t) frame by frame and screenshots it;
// nothing here runs on a clock, so the film is the same every time.
// The windows and the TV play frames that record.mjs took from the real UI.
(() => {
  "use strict";
  const FPS = 60, LEN = 70;
  const Q = new URLSearchParams(location.search);
  const SRC = Q.get("src") === "preview" ? "preview" : "footage", STEP = SRC === "preview" ? 5 : 1;
  const TAKES = { "win-lib": 696, "win-saves": 384, "win-ach": 540, "tv-main": 696, "tv-menu": 312, "tv-apps": 216 };
  const ART = "../harness/.cache/";
  const P1 = "#6C9BD2", P2 = "#D9738E";
  const stage = document.getElementById("stage");

  // ---------------------------------------------------------------- maths --
  const clamp = (v, a = 0, b = 1) => Math.min(b, Math.max(a, v));
  const mix = (a, b, k) => a + (b - a) * k;
  const E = {
    lin: (k) => k,
    out: (k) => 1 - Math.pow(1 - k, 3),
    expo: (k) => (k >= 1 ? 1 : 1 - Math.pow(2, -10 * k)),
    in: (k) => k * k * k,
    io: (k) => (k < 0.5 ? 4 * k * k * k : 1 - Math.pow(-2 * k + 2, 3) / 2),
  };
  // A damped spring from 0 to 1, sec after it was let go.
  function spring(sec, visual = 0.5, bounce = 0.25) {
    if (sec <= 0) return 0;
    const w = (2 * Math.PI) / visual, z = 1 - bounce;
    if (z >= 1) return 1 - Math.exp(-w * sec) * (1 + w * sec);
    const wd = w * Math.sqrt(1 - z * z);
    return 1 - Math.exp(-z * w * sec) * (Math.cos(wd * sec) + ((z * w) / wd) * Math.sin(wd * sec));
  }
  const ramp = (t, t0, d) => clamp((t - t0) / d);
  // Keyframes: kf(t, [t0, v0], [t1, v1, ease], ...)
  function kf(t, ...ks) {
    if (t <= ks[0][0]) return ks[0][1];
    for (let i = 1; i < ks.length; i++) {
      if (t <= ks[i][0]) { const [t0, v0] = ks[i - 1], [t1, v1, e = E.io] = ks[i]; return mix(v0, v1, e((t - t0) / (t1 - t0))); }
    }
    return ks[ks.length - 1][1];
  }

  // ------------------------------------------------------------------ dom --
  function add(parent, html) { const d = document.createElement("div"); d.innerHTML = html.trim(); const el = d.firstChild; parent.appendChild(el); return el; }
  function show(el, on) { const v = on ? "" : "none"; if (el.style.display !== v) el.style.display = v; }
  // Places an element: x, y, scale, rotation, opacity. Hidden at opacity 0.
  function put(el, o) {
    const op = o.o == null ? 1 : o.o;
    if (op <= 0.001) { show(el, false); return; }
    show(el, true);
    el.style.opacity = op >= 0.999 ? "" : op.toFixed(3);
    el.style.transform = `translate(${(o.x || 0).toFixed(2)}px,${(o.y || 0).toFixed(2)}px)${o.c ? " translate(-50%,-50%)" : ""}${o.r ? ` rotate(${o.r.toFixed(3)}deg)` : ""}${o.s != null && o.s !== 1 ? ` scale(${o.s.toFixed(4)})` : ""}`;
  }
  const ic = (name, weight) => { const i = window.PH[name] || {}; return `<svg viewBox="0 0 256 256">${i[weight] || i.regular || ""}</svg>`; };
  // Words that rise out of their line.
  function split(el, whole) {
    const txt = el.textContent; el.textContent = "";
    const out = [], parts = whole ? [txt] : txt.split(" ");
    parts.forEach((wd, i) => {
      const w = document.createElement("span"); w.className = "w";
      const wi = document.createElement("span"); wi.className = "wi"; wi.textContent = wd;
      w.appendChild(wi); el.appendChild(w); out.push(wi);
      if (i < parts.length - 1) { const sp = document.createElement("span"); sp.className = "sp"; el.appendChild(sp); }
    });
    return out;
  }
  function rise(ws, t, tin, tout, st = 0.06, dur = 0.7) {
    ws.forEach((w, i) => {
      const a = E.expo(ramp(t, tin + i * st, dur)), b = tout == null ? 0 : E.in(ramp(t, tout + i * 0.03, 0.3));
      w.style.transform = `translateY(${((1 - a) * 130 - b * 130).toFixed(2)}%) rotate(${((1 - a) * 4).toFixed(2)}deg)`;
      w.style.visibility = a <= 0 || b >= 1 ? "hidden" : "";
    });
  }
  // Frames from the real UI.
  const waits = [];
  function film(img, take, sec) {
    let f = clamp(Math.round(sec * FPS), 0, TAKES[take] - 1); f -= f % STEP;
    const src = `out/${SRC}/${take}/f${String(f).padStart(5, "0")}.jpg`;
    if (img.dataset.src !== src) { img.dataset.src = src; img.src = src; waits.push(img.decode().catch(() => {})); }
  }

  // ----------------------------------------------------------- background --
  const bg = add(stage, '<div id="bg"></div>');
  const TINT = [
    [0, [40, 72, 140], [130, 52, 30], 0.4], [5.6, [40, 72, 140], [130, 52, 30], 0.4],
    [6.4, [150, 88, 14], [30, 50, 120], 0.46], [10.8, [150, 88, 14], [30, 50, 120], 0.46],
    [11.6, [34, 70, 140], [20, 96, 120], 0.46], [17.2, [34, 70, 140], [20, 96, 120], 0.46],
    [17.8, [108, 155, 210], [217, 115, 142], 0.26], [23.8, [108, 155, 210], [217, 115, 142], 0.26],
    [24.4, [44, 60, 140], [90, 44, 130], 0.4], [31.8, [44, 60, 140], [90, 44, 130], 0.4],
    [32.4, [150, 110, 30], [34, 70, 140], 0.4], [39.6, [150, 110, 30], [34, 70, 140], 0.4],
    [40.2, [26, 36, 70], [70, 34, 24], 0.5], [57.8, [26, 36, 70], [70, 34, 24], 0.5],
    [58.3, [150, 24, 40], [30, 40, 90], 0.36], [LEN, [150, 24, 40], [30, 40, 90], 0.36],
  ];
  function drawBg(t) {
    let i = 1; while (i < TINT.length - 1 && TINT[i][0] < t) i++;
    const A = TINT[i - 1], B = TINT[i], k = ramp(t, A[0], B[0] - A[0] || 1);
    const c = (n) => [0, 1, 2].map((j) => Math.round(mix(A[n][j], B[n][j], k))).join(","), a = mix(A[3], B[3], k).toFixed(3);
    const x1 = 28 + Math.sin(t * 0.21) * 8, y1 = 30 + Math.cos(t * 0.17) * 8, x2 = 76 + Math.cos(t * 0.19) * 8, y2 = 72 + Math.sin(t * 0.23) * 8;
    bg.style.background = `radial-gradient(1300px 900px at ${x1.toFixed(2)}% ${y1.toFixed(2)}%, rgba(${c(1)},${a}), transparent 70%), radial-gradient(1200px 950px at ${x2.toFixed(2)}% ${y2.toFixed(2)}%, rgba(${c(2)},${a}), transparent 70%), #111217`;
  }

  // ------------------------------------------------------------- the hook --
  const LOGO = '<svg viewBox="0 0 20 20"><path d="M2.75 10.25c0 3.7 3.25 6.25 7.25 6.25s7.25-2.55 7.25-6.25"/><path d="M5.75 10.25c0 2.2 1.9 3.5 4.25 3.5s4.25-1.3 4.25-3.5"/><circle cx="10" cy="6.25" r="2.5"/></svg>';
  function makeLogo(parent) {
    const el = add(parent, `<div class="logo ctr">${LOGO}<b>Lootnest</b></div>`);
    const arcs = [...el.querySelectorAll("path")], gem = el.querySelector("circle"), word = split(el.querySelector("b"), true);
    arcs.forEach((p) => { p.dataset.len = p.getTotalLength(); p.style.strokeDasharray = p.dataset.len; });
    return (t, t0) => {
      gem.setAttribute("r", Math.max(0, 2.5 * spring(t - t0, 0.5, 0.42)).toFixed(3));
      arcs.forEach((p, i) => { const k = E.out(ramp(t, t0 + 0.12 + i * 0.1, 0.55)); p.style.opacity = k > 0 ? 1 : 0; p.style.strokeDashoffset = (p.dataset.len * (1 - k)).toFixed(3); });
      rise(word, t, t0 + 0.2, null, 0, 0.8);
      return el;
    };
  }
  const hook = add(stage, '<div class="scene"></div>');
  const hk1 = split(add(hook, '<div class="hk-line" style="top:306px">One game.</div>'));
  const hk2 = split(add(hook, '<div class="hk-line" style="top:478px">Many lives.</div>'));
  const WHO = [["Player 1", P1, 0], ["hardcore", P1, 1], ["100%", P1, 1], ["Player 2", P2, 0]];
  const cards = WHO.map(([name, c, run]) => add(hook, `<div class="card" style="--c:${c}${run ? `;--ring:${c}` : ""}"><img src="${ART}1086940-cover.jpg"><span class="tag"><i></i>${name}</span><span class="bar"></span></div>`));
  const hookLogo = makeLogo(hook);
  function drawHook(t) {
    rise(hk1, t, 0.7, 4.25); rise(hk2, t, 2.5, 4.3);
    cards.forEach((el, i) => {
      const a = spring(t - 0.35, 0.7, 0.28), f = spring(t - 2.5 - i * 0.04, 0.65, 0.3), e = E.in(ramp(t, 4.3 + i * 0.03, 0.55)), d = i - 1.5;
      const x = mix(1440, 1420, f) + d * 198 * f, y = 540 + (1 - a) * 160 + d * d * 11 * f + Math.sin(t * 1.3 + i) * 5;
      const seen = i === 3 ? ramp(t, 0.35, 0.25) : t >= 2.45 ? 1 : 0;
      put(el, { x: mix(x, 960, e), y: mix(y, 530, e), r: d * 5 * f * (1 - e), s: (0.8 + 0.2 * a) * mix(1, 0.82, f) * mix(1, 0.1, e), o: seen * (1 - ramp(t, 4.72, 0.2)) });
      const tag = el.querySelector(".tag"), k = ramp(t, 2.85 + i * 0.07, 0.3);
      tag.style.opacity = k; tag.style.transform = `translateY(${(1 - E.out(k)) * 12}px)`;
      el.querySelector(".bar").style.opacity = ramp(t, 2.6, 0.3);
    });
    const lg = hookLogo(t, 4.85), k = E.in(ramp(t, 5.78, 0.26));
    put(lg, { x: 960, y: 530, c: 1, s: 1 + 0.07 * k, o: t < 4.8 ? 0 : 1 - k });
  }

  // -------------------------------------------------------------- windows --
  const world = add(stage, '<div id="world"></div>');
  const scrim = add(stage, '<div id="scrim"></div>');
  function makeWin() { const el = add(world, '<div class="win"><img alt=""></div>'); return { el, img: el.firstChild }; }
  // The point (fx, fy) of the window sits at (cx, cy) on the stage.
  function putWin(w, o) {
    const op = o.o == null ? 1 : o.o;
    if (op <= 0.001) { show(w.el, false); return; }
    show(w.el, true);
    w.el.style.opacity = op >= 0.999 ? "" : op.toFixed(3);
    const fx = o.fx == null ? 720 : o.fx, fy = o.fy == null ? 405 : o.fy;
    w.el.style.transform = `translate(${o.cx.toFixed(2)}px,${o.cy.toFixed(2)}px) rotateX(${(o.rx || 0).toFixed(3)}deg) rotateY(${(o.ry || 0).toFixed(3)}deg) scale(${o.s.toFixed(4)}) translate(${(-fx).toFixed(2)}px,${(-fy).toFixed(2)}px)`;
  }
  const w1 = makeWin(), w2 = makeWin(), w3 = makeWin();
  const s1 = add(stage, '<div class="scene"></div>');
  const s1a = split(add(s1, '<div class="s1-h">Every game</div>'));
  const s1b = split(add(s1, '<div class="s1-h" style="top:430px">on your PC.</div>'));
  const s1rule = add(s1, '<div class="s1-rule"></div>');
  const STORES = ["Steam", "Epic", "GOG", "Xbox", "EA", "Ubisoft", "Battle.net", "itch.io", "And the ones in a folder."];
  const s1tick = add(s1, '<div class="s1-tick"></div>');
  const ticks = STORES.map((n) => { const sp = document.createElement("span"); sp.textContent = n; s1tick.appendChild(sp); return split(sp, true)[0]; });
  function drawW1(t) {
    // the caption beside the window
    show(s1, t >= 5.9 && t < 10.2);
    rise(s1a, t, 6.05, 9.6); rise(s1b, t, 6.2, 9.64);
    s1rule.style.transform = `scaleX(${(E.expo(ramp(t, 6.5, 0.6)) * (1 - E.in(ramp(t, 9.6, 0.25)))).toFixed(3)})`;
    ticks.forEach((w, i) => {
      const t0 = 6.6 + i * 0.27, t1 = i === STORES.length - 1 ? 9.66 : t0 + 0.27;
      const a = E.expo(ramp(t, t0, 0.24)), b = E.in(ramp(t, t1, 0.14));
      w.style.transform = `translateY(${((1 - a) * 114 - b * 114).toFixed(2)}%)`;
    });
    // the window: in from the right at an angle, then square to the camera
    const k = spring(t - 5.9, 0.9, 0.12), out = E.in(ramp(t, 17.3, 0.3));
    putWin(w1, {
      cx: kf(t, [9.7, 1345], [10.8, 960]) + (1 - k) * 1100, cy: kf(t, [9.7, 530], [10.8, 445]),
      s: kf(t, [5.9, 0.83], [9.7, 0.86, E.lin], [10.8, 1], [11.9, 1], [12.7, 1.28], [13.25, 1.28], [13.8, 1.22], [15.35, 1.3, E.lin], [15.9, 1.45], [17.3, 1.5, E.lin]) * (1 - 0.06 * out),
      ry: kf(t, [9.7, -16], [10.8, 0]) - (1 - k) * 16, rx: kf(t, [9.7, 3], [10.8, 0]),
      fx: kf(t, [11.9, 720], [12.7, 1010], [13.25, 1010], [13.8, 720]), fy: kf(t, [11.9, 405], [12.7, 470], [13.25, 470], [13.8, 385], [15.35, 385], [15.9, 345]),
      o: t < 5.9 ? 0 : 1 - out,
    });
    if (t >= 5.9) film(w1.img, "win-lib", t - 5.75);
  }
  function drawW2(t) {
    const a = spring(t - 26.05, 0.7, 0.15), out = E.in(ramp(t, 31.72, 0.28));
    putWin(w2, {
      cx: 960 - out * 520, cy: kf(t, [27.2, 445], [28, 452]) + (1 - a) * 46,
      s: kf(t, [27.2, 1], [28, 1.5], [31.6, 1.56, E.lin]) * (0.93 + 0.07 * a),
      fx: kf(t, [27.2, 720], [28, 610]), fy: kf(t, [27.2, 405], [28, 585]),
      o: ramp(t, 26.05, 0.22) * (1 - out),
    });
    if (t >= 25.9) film(w2.img, "win-saves", t - 25.6);
  }
  function drawW3(t) {
    const a = E.expo(ramp(t, 31.95, 0.55)), out = E.in(ramp(t, 39.72, 0.28));
    putWin(w3, {
      cx: 960 + (1 - a) * 640, cy: 445,
      s: kf(t, [33.5, 1], [34.15, 1.3], [35.05, 1.3], [35.6, 1.2], [37.2, 1.2], [37.95, 1.62], [40, 1.7, E.lin]) * (1 - 0.05 * out),
      fx: kf(t, [33.5, 720], [34.15, 980], [35.05, 980], [35.6, 740], [37.2, 740], [37.9, 720]),
      fy: kf(t, [33.5, 405], [34.15, 300], [35.05, 300], [35.6, 480], [37.2, 480], [37.9, 405]),
      o: ramp(t, 31.95, 0.2) * (1 - out),
    });
    if (t >= 31.9) film(w3.img, "win-ach", t - 31.7);
  }

  // ------------------------------------------------------------- the swap --
  const swap = add(stage, '<div class="scene" style="transform:scale(1.42);transform-origin:960px 400px"></div>');
  const KINDS = [["Saves", "floppy-disk"], ["Settings", "sliders-horizontal"], ["Achievements", "trophy"]];
  const PT = 232, socks = [0, 1, 2].map((i) => `<span class="sock" style="top:${110 + i * 64}px"></span>`).join("");
  const person = (name, c, small) => add(swap, `<div class="panel" style="--c:${c}${small ? `;--ring:${c}` : ""}"><div class="hd"><span class="av">P</span><span>${small ? `<small>${small}</small>` : ""}${name}</span></div>${socks}<span class="glow"></span></div>`);
  const pLeft = person("Player 2", P2), pRight = person("Player 1", P1), pRun = person("hardcore", P1, "Player 1's run");
  const pGame = add(swap, `<div class="panel game" style="--c:${P1}"><div class="hd"><img src="${ART}1086940-hero.jpg"><span>Baldur's Gate 3</span></div>${socks.replace(/left:20px/g, "")}<span class="glow"></span></div>`);
  pGame.querySelectorAll(".sock").forEach((s) => { s.style.left = "40px"; s.style.right = "40px"; });
  const chip = (i, c, ghost) => add(swap, `<div class="chip${ghost ? " ghost" : ""}" style="--c:${c}">${ic(KINDS[i][1], "fill")}${KINDS[i][0]}${ghost ? ic("check-circle", "fill").replace("<svg", '<svg class="ck"') : ""}</div>`);
  const pink = [0, 1, 2].map((i) => chip(i, P2)), copies = [0, 1, 2].map((i) => chip(i, P2, true)), blue = [0, 1, 2].map((i) => chip(i, P1)), runs = [0, 1, 2].map((i) => chip(i, P1));
  const shelfL = add(swap, '<div class="shelf-l">Copies, kept before every swap</div>');
  function drawSwap(t) {
    const m = E.io(ramp(t, 21.5, 0.75)), out = ramp(t, 23.75, 0.25);
    const LX = mix(500, 250, m), GX = mix(960, 500, m), RX = mix(1420, 960, m), UX = 1420;
    const pop = (t0) => spring(t - t0, 0.6, 0.3);
    const panel = (el, x, t0, o = 1) => { const a = pop(t0); put(el, { x, y: PT + (1 - a) * 60, s: 0.9 + 0.1 * a, o: ramp(t, t0, 0.2) * o * (1 - out) }); return a; };
    panel(pLeft, LX, 17.6, 1 - m); panel(pGame, GX, 17.7); panel(pRight, RX, 17.8); panel(pRun, UX, 21.95);
    pGame.querySelector(".glow").style.opacity = (0.75 * ramp(t, 21.05, 0.3)).toFixed(3);
    const at = (x, i) => [x - 130, PT + 110 + i * 64];
    const fly = (el, from, to, k, o, lift = 60) => put(el, { x: mix(from[0], to[0], k), y: mix(from[1], to[1], k) - Math.sin(Math.PI * k) * lift, o: o * (1 - out) });
    for (let i = 0; i < 3; i++) {
      const seen = ramp(t, 17.75 + i * 0.05, 0.2);
      // a copy first
      const c = E.out(ramp(t, 18.5 + i * 0.08, 0.6));
      fly(copies[i], at(GX, i), [960 + (i - 1) * 284 - 130, 590], c, (t >= 18.5 + i * 0.08 ? 1 : 0) * (1 - ramp(t, 21.3, 0.3)), 0);
      // the last player's files go home
      fly(pink[i], at(GX, i), at(LX, i), E.io(ramp(t, 19.5 + i * 0.1, 0.7)), seen * (1 - m));
      // the next one's go in
      fly(blue[i], at(RX, i), at(GX, i), E.io(ramp(t, 20.5 + i * 0.1, 0.7)), ramp(t, 17.9 + i * 0.05, 0.2));
      // a run has its own
      const r = spring(t - 22.2 - i * 0.09, 0.5, 0.35);
      put(runs[i], { x: UX - 130, y: PT + 110 + i * 64 + (1 - r) * 20, s: 0.8 + 0.2 * r, o: ramp(t, 22.2 + i * 0.09, 0.15) * (1 - out) });
    }
    put(shelfL, { x: 960, y: 668, c: 1, o: ramp(t, 18.7, 0.3) * (1 - ramp(t, 21.3, 0.3)) });
  }

  // ------------------------------------------------------------- the keys --
  const keys = add(stage, '<div class="scene"></div>');
  const kb = add(keys, '<div class="kb ctr"><span><span class="kcap">Ctrl</span></span><span class="kplus">+</span><span><span class="kcap">Shift</span></span><span class="kplus">+</span><span><span class="kcap">F5</span></span></div>');
  const kwrap = [...kb.children].filter((e) => !e.classList.contains("kplus")), kplus = [...kb.querySelectorAll(".kplus")];
  const kring = add(keys, '<div class="kring"></div>');
  const kpill = add(keys, `<div class="pill ctr">${ic("check-circle", "fill")}Save point made</div>`);
  let f5x = 0;
  function drawKeys(t) {
    const out = E.in(ramp(t, 25.8, 0.24));
    put(kb, { x: 960, y: 420 + out * 30, c: 1, s: 1.3, o: 1 - out });
    if (!f5x) { const r = kwrap[2].getBoundingClientRect(), s = stage.getBoundingClientRect(); f5x = r.left - s.left + r.width / 2; }
    const downs = [24.72, 24.97, 25.22];
    kwrap.forEach((el, i) => {
      const a = spring(t - 24 - i * 0.1, 0.6, 0.3);
      el.style.transform = `translateY(${((1 - a) * -170).toFixed(2)}px)`; el.style.opacity = ramp(t, 24 + i * 0.1, 0.18);
      el.firstChild.style.setProperty("--p", kf(t, [downs[i], 0], [downs[i] + 0.07, 1, E.out], [25.66, 1], [25.8, 0, E.out]).toFixed(3));
    });
    kplus.forEach((el, i) => { el.style.opacity = ramp(t, 24.25 + i * 0.1, 0.25); });
    const r = ramp(t, 25.27, 0.55);
    put(kring, { x: f5x, y: 420, s: 0.5 + 1.9 * E.out(r), o: t < 25.27 ? 0 : 0.75 * (1 - r) });
    const p = spring(t - 25.38, 0.5, 0.32);
    put(kpill, { x: 960, y: 680 + (1 - p) * 36 + out * 20, c: 1, s: 1.15 * (0.9 + 0.1 * p), o: ramp(t, 25.38, 0.15) * (1 - out) });
  }

  // --------------------------------------------------------------- the TV --
  const tvs = add(stage, '<div class="scene"></div>');
  const pc = add(tvs, '<div class="pc"><div class="pc-neck"></div><div class="pc-base"></div><div class="pc-body"><img alt=""></div></div>');
  const dots = [0, 1, 2, 3, 4, 5, 6].map(() => add(tvs, '<div class="dot"></div>'));
  const tv = add(tvs, '<div class="tv"><img class="tv-glow" alt=""><div class="tv-shelf"></div><div class="tv-foot" style="left:-480px"></div><div class="tv-foot" style="left:330px"></div><div class="tv-body"><div class="tv-screen"><img alt=""></div></div></div>');
  const tvGlow = tv.querySelector(".tv-glow"), tvImg = tv.querySelector(".tv-screen img");
  const lblPC = add(tvs, '<div class="lbl">Your PC</div>'), lblTV = add(tvs, '<div class="lbl">Your TV</div>');
  const pad = add(stage, '<div class="pad"><i class="d" style="left:51px;top:41px"></i><i class="d" data-b="U" style="left:51px;top:19px"></i><i class="d" data-b="D" style="left:51px;top:63px"></i><i class="d" data-b="L" style="left:29px;top:41px"></i><i class="d" data-b="R" style="left:73px;top:41px"></i><i class="g" data-b="G"></i><i class="f" data-b="A" style="left:176px;top:66px"></i><i class="f" style="left:200px;top:40px"></i><i class="f" style="left:152px;top:40px"></i><i class="f" style="left:176px;top:14px"></i></div>');
  const PADS = [[40.7, "R"], [41.3, "L"], [42, "A"], [42.9, "R"], [43.4, "R"], [43.9, "R"], [45, "A"], [46.5, "D"], [46.95, "R"], [47.4, "R"], [48.2, "U"], [48.9, "A"], [51.5, "G"], [52.8, "D"], [53.6, "A"], [55.2, "D"], [56, "R"], [56.6, "R"]];
  const padBtn = [...pad.querySelectorAll("[data-b]")];
  function drawTV(t) {
    const a = spring(t - 40, 0.8, 0.15), out = E.in(ramp(t, 57.7, 0.3));
    const s = kf(t, [42.4, 1], [43.4, 1.22], [48.95, 1.22], [49.75, 0.74], [51.15, 0.74], [51.9, 1.22]) * (0.9 + 0.1 * a) * (1 - 0.05 * out);
    const x = kf(t, [48.95, 960], [49.75, 1380], [51.15, 1380], [51.9, 960]), y = kf(t, [42.4, 470], [43.4, 468], [48.95, 468], [49.75, 450], [51.15, 450], [51.9, 468]);
    const o = ramp(t, 39.9, 0.35) * (1 - out);
    put(tv, { x, y, s, o });
    const take = t < 51.3 ? ["tv-main", t - 40] : t < 55 ? ["tv-menu", t - 51.3] : ["tv-apps", t - 54.8];
    film(tvImg, take[0], take[1]); film(tvGlow, take[0], take[1]);
    tvImg.style.opacity = (1 - 0.7 * (1 - clamp(Math.abs(t - 55) / 0.12))).toFixed(3);
    // the PC it streams from
    const p = spring(t - 49.35, 0.6, 0.2), po = ramp(t, 49.35, 0.2) * (1 - ramp(t, 51.1, 0.25));
    put(pc, { x: 420 - (1 - p) * 120, y: 430, s: 1.2, o: po });
    if (po > 0) film(pc.querySelector("img"), "win-lib", 0.3);
    put(lblPC, { x: 420, y: 770, o: po }); lblPC.style.transform += " translateX(-50%)";
    put(lblTV, { x: 1380, y: 770, o: po }); lblTV.style.transform += " translateX(-50%)";
    dots.forEach((d, i) => { const k = (t * 0.7 + i / dots.length) % 1; put(d, { x: mix(752, 884, k), y: 440 + Math.sin(k * Math.PI * 2 + t) * 5, s: 0.5 + 0.5 * Math.sin(Math.PI * k), o: po * Math.sin(Math.PI * k) * ramp(t, 49.7, 0.3) }); });
    // the controller, lighting up with each press
    const ho = ramp(t, 40.4, 0.3) * (1 - ramp(t, 49, 0.25) * (1 - ramp(t, 51.6, 0.25))) * (1 - ramp(t, 57.5, 0.25));
    put(pad, { x: 1580, y: 936, o: ho });
    padBtn.forEach((b) => {
      let last = -9; for (const [at, name] of PADS) if (name === b.dataset.b && at <= t) last = at;
      const k = 1 - ramp(t, last, 0.3), v = Math.round(mix(58, 255, k));
      b.style.background = `rgb(${v},${Math.round(mix(63, 255, k))},${Math.round(mix(75, 255, k))})`; b.style.transform = `scale(${(1 + 0.3 * k).toFixed(3)})`;
    });
  }

  // ------------------------------------------------- the list, the nevers --
  const also = add(stage, '<div class="scene"></div>');
  const ALSO = [["Mods", "puzzle-piece"], ["Updates, with Undo", "arrow-circle-up"], ["Your own cloud", "cloud"], ["itch.io and GOG installs", "download-simple"], ["Playnite import", "squares-four"], ["A timeline per game", "clock-counter-clockwise"], ["Will it run here?", "gauge"], ["Plugins", "cube"]];
  const alsoL = add(also, '<div class="also-l">Also in it</div>');
  const alsoN = add(also, `<div class="also-n">${ALSO.map(() => "<i><b></b></i>").join("")}</div>`);
  const alsoI = ALSO.map(([name, icon]) => { const el = add(also, `<div class="also-i">${name}</div>`); const w = split(el, true)[0]; const svg = add(also, ic(icon, "duotone").replace("<svg", '<svg class="also-ic"')); return { el, w, svg, fit: 1 }; });
  let fitted = false;
  function drawAlso(t) {
    if (!fitted) { fitted = true; alsoI.forEach((it) => { it.fit = Math.min(1, 1240 / it.el.scrollWidth); it.el.style.transform = `scale(${it.fit.toFixed(4)})`; }); }
    const out = ramp(t, 61.86, 0.14);
    alsoL.style.opacity = ramp(t, 58, 0.25) * (1 - out); alsoN.style.opacity = ramp(t, 58, 0.25) * (1 - out);
    alsoI.forEach((it, i) => {
      const t0 = 58 + i * 0.5, t1 = t0 + (i === ALSO.length - 1 ? 0.36 : 0.5), a = E.expo(ramp(t, t0, 0.3)), b = E.in(ramp(t, t1, 0.16));
      show(it.el, t >= t0 && t < t1 + 0.2);
      it.w.style.transform = `translateY(${((1 - a) * 114 - b * 114).toFixed(2)}%)`;
      const k = spring(t - t0, 0.45, 0.35), o = ramp(t, t0, 0.12) * (1 - ramp(t, t1, 0.12));
      show(it.svg, o > 0.001); it.svg.style.opacity = o; it.svg.style.transform = `scale(${(0.8 + 0.2 * k).toFixed(3)}) rotate(${((1 - k) * -8).toFixed(2)}deg)`;
      alsoN.children[i].firstChild.style.transform = `scaleX(${ramp(t, t0, 0.5).toFixed(3)})`;
    });
  }
  const never = add(stage, '<div class="scene"></div>');
  const NEVER = ["No account.", "No telemetry.", "No file moved without a copy."].map((s, i) => { const el = add(never, `<div class="nv" style="top:${300 + i * 150}px">${s}</div>`); return { el, w: split(el) }; });
  function drawNever(t) {
    NEVER.forEach((n, i) => { rise(n.w, t, 62 + i, 64.75, 0.05, 0.6); n.el.style.opacity = i < 2 ? mix(1, 0.38, ramp(t, 63 + i, 0.3)) : 1; });
  }

  // -------------------------------------------------------------- the end --
  const fin = add(stage, '<div class="scene"></div>');
  const fin1 = split(add(fin, '<div class="fin-h" style="top:330px">Your saves,</div>'));
  const fin2 = split(add(fin, '<div class="fin-h" style="top:504px">your slots.</div>'));
  const finLogo = makeLogo(fin);
  const finKey = add(fin, '<div class="fin-key ctr">Free for Windows</div>');
  const finUrl = add(fin, '<div class="fin-url ctr">c-dinca.github.io/lootnest</div>');
  const finNote = add(fin, '<div class="fin-note ctr">Open source. No account. In beta.</div>');
  function drawFin(t) {
    rise(fin1, t, 65.05, 66.7); rise(fin2, t, 65.2, 66.75);
    put(finLogo(t, 67), { x: 960, y: 380, c: 1, o: t < 66.95 ? 0 : 1 });
    const k = spring(t - 67.7, 0.55, 0.3);
    put(finKey, { x: 960, y: 650 + (1 - k) * 40, c: 1, s: 0.9 + 0.1 * k, o: ramp(t, 67.7, 0.18) });
    put(finUrl, { x: 960, y: 790 + (1 - E.expo(ramp(t, 68, 0.6))) * 20, c: 1, o: ramp(t, 68, 0.35) });
    put(finNote, { x: 960, y: 858 + (1 - E.expo(ramp(t, 68.15, 0.6))) * 20, c: 1, o: ramp(t, 68.15, 0.35) });
  }

  // ------------------------------------------------------------- captions --
  const capsEl = add(stage, '<div class="scene"></div>');
  const CAPS = [
    [11.8, 13.4, "A slot for everyone.", "Their own saves, settings and achievements."],
    [13.65, 15.3, "Press Play. Pick who's playing."],
    [15.55, 17.25, "Their files go in before the game starts."],
    [17.75, 19.35, "Someone else's turn.", "A copy is made first.", 18.5],
    [19.55, 21.35, "The last player's files go home.", "The next one's go in.", 20.5],
    [21.6, 23.75, "A second run of your own.", "Go hardcore without touching your main save."],
    [24.15, 26.35, "Save points, mid-game."],
    [26.9, 29.35, "Name the moment."],
    [29.6, 31.7, "Go back to any of them."],
    [32.4, 35.3, "Achievements as they unlock.", "With how rare each one is.", 34],
    [35.55, 37.15, "For every profile and every run."],
    [37.4, 39.7, "For games from outside the stores too."],
    [40.3, 42.5, "On the couch.", null, 0, 1],
    [43, 48.7, "Big Picture, for a controller.", null, 0, 1],
    [49.15, 51.3, "Streamed to your TV.", "LG TVs, over your home network."],
    [51.7, 54.8, "The Home button, over any game.", null, 0, 1],
    [55.15, 57.6, "YouTube, Spotify and Discord too.", null, 0, 1],
  ].map(([tin, tout, title, sub, subAt, onTV]) => {
    const te = add(capsEl, `<div class="cap cap-t${onTV ? " on-tv" : ""}">${title}</div>`), se = sub ? add(capsEl, `<div class="cap cap-s">${sub}</div>`) : null;
    return { tin, tout, te, tw: split(te), se, subAt: subAt || tin + 0.25 };
  });
  function drawCaps(t) {
    for (const c of CAPS) {
      const on = t >= c.tin - 0.05 && t < c.tout + 0.5;
      show(c.te, on); if (c.se) show(c.se, on);
      if (!on) continue;
      rise(c.tw, t, c.tin, c.tout, 0.045, 0.6);
      if (c.se) { const a = E.expo(ramp(t, c.subAt, 0.6)), b = ramp(t, c.tout, 0.22); c.se.style.opacity = (ramp(t, c.subAt, 0.3) * (1 - b)).toFixed(3); c.se.style.transform = `translateY(${((1 - a) * 16).toFixed(2)}px)`; }
    }
  }

  add(stage, '<div id="vig"></div>');
  const fade = add(stage, '<div id="fade"></div>');
  const within = (el, t, a, b, draw) => { const on = t >= a && t < b; show(el, on); if (on) draw(t); return on; };
  window.LEN = LEN; window.FPS = FPS;
  window.seek = async (t) => {
    waits.length = 0;
    drawBg(t);
    within(hook, t, 0, 6.06, drawHook);
    if (t >= 5.85 && t < 17.62) drawW1(t); else { show(w1.el, false); show(s1, false); }
    within(swap, t, 17.5, 24.02, drawSwap);
    within(keys, t, 23.95, 26.2, drawKeys);
    if (t >= 25.95 && t < 32.02) drawW2(t); else show(w2.el, false);
    if (t >= 31.9 && t < 40.02) drawW3(t); else show(w3.el, false);
    if (within(tvs, t, 39.85, 58.02, drawTV) === false) show(pad, false);
    within(also, t, 57.95, 62.08, drawAlso);
    within(never, t, 61.95, 65.1, drawNever);
    within(fin, t, 64.95, LEN + 1, drawFin);
    drawCaps(t);
    // the bottom of a window fades under its caption
    scrim.style.opacity = (ramp(t, 10.6, 0.5) * (1 - ramp(t, 17.3, 0.3)) + ramp(t, 26, 0.3) * (1 - ramp(t, 39.8, 0.2))).toFixed(3);
    fade.style.opacity = (1 - ramp(t, 0.05, 0.35) + ramp(t, 69.45, 0.5)).toFixed(3);
    await Promise.all(waits);
  };
  window.ready = Promise.all([document.fonts.load('60px "Tilt Warp"'), document.fonts.load('600 30px "Atkinson"'), document.fonts.load('600 30px "Figtree"', "0123"), ...[...document.images].filter((i) => i.getAttribute("src")).map((i) => i.decode().catch(() => {}))]).then(() => document.fonts.ready);
})();
