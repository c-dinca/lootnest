// Lootnest's site: two scroll stories, each on one stage. Scrolling moves a
// clock from 0 to 5, one unit per caption; each actor has keyframes on it.
// Nothing moves on its own: with reduced motion the stage still follows
// the scroll, without the transitions.
(() => {
  "use strict";
  const root = document.documentElement;

  // ------------------------------------------------------------ engine ---
  const lerp = (a, b, f) => a + (b - a) * f;
  const ease = (f) => (f < 0.5 ? 4 * f * f * f : 1 - Math.pow(-2 * f + 2, 3) / 2);
  const clamp = (v, a = 0, b = 1) => Math.min(b, Math.max(a, v));
  // keys: [[t, {x, y, o, s}], ...]; later frames inherit what they don't set
  function at(keys, t) {
    let prev = { x: 0, y: 0, o: 1, s: 1, ...keys[0][1] };
    if (t <= keys[0][0]) return prev;
    for (let i = 1; i < keys.length; i++) {
      const next = { ...prev, ...keys[i][1] };
      if (t <= keys[i][0]) {
        const f = ease((t - keys[i - 1][0]) / (keys[i][0] - keys[i - 1][0])), r = {};
        for (const k in next) r[k] = lerp(prev[k], next[k], f);
        return r;
      }
      prev = next;
    }
    return prev;
  }
  const put = (el, v) => {
    el.style.transform = `translate(${v.x}px, ${v.y}px) scale(${v.s})`;
    el.style.opacity = v.o;
    el.style.visibility = v.o < 0.01 ? "hidden" : "";
    const em = el.querySelector(":scope > em");
    if (em) em.style.opacity = v.y < -60 ? 0 : 1;
  };
  const between = (t, a, b) => t >= a && t < b;

  function stage(section, build) {
    const world = section.querySelector(".world"), scene = section.querySelector(".scene");
    const caps = [...section.querySelectorAll(".cap")], bars = [...section.querySelectorAll(".steps b")];
    const mk = (cls, html, style) => {
      const d = document.createElement("div");
      d.className = "el " + cls; d.innerHTML = html;
      if (style) d.setAttribute("style", style);
      world.appendChild(d);
      return d;
    };
    const render = build(mk);
    let step = -1;
    return {
      section,
      fit() {
        scene.style.height = "";
        const w = scene.clientWidth, h = scene.clientHeight || innerHeight * 0.6;
        const k = Math.min(w / (innerWidth <= 900 ? 600 : 660), h / 640, 1.1);
        world.style.setProperty("--k", k);
        if (innerWidth <= 900) scene.style.height = Math.round(640 * k) + "px";
      },
      // returns the colour of whoever is in the game, or null
      tick() {
        const r = section.getBoundingClientRect();
        const p = clamp(-r.top / (r.height - innerHeight));
        const t = Math.min(5, p * 5.2);
        const s = Math.min(4, Math.floor(t));
        if (s !== step) { step = s; caps.forEach((c, i) => c.classList.toggle("on", i === s)); }
        bars.forEach((b, i) => b.style.setProperty("--f", clamp(t - i)));
        return { color: render(t), active: r.top < innerHeight * 0.5 && r.bottom > innerHeight * 0.5 };
      },
    };
  }
  const bump = (el, text) => { if (el.textContent !== text) { el.textContent = text; el.classList.remove("bump"); void el.offsetWidth; el.classList.add("bump"); } };

  // ----------------------------------------------- story 1: profiles ---
  const P1 = "#6c9bd2", P2 = "#e88ca0";
  const saves = (mk) => {
    const GAMES = [["Steam", "Elden Ring"], ["Epic", "Alan Wake 2"], ["GOG", "The Witcher 3"], ["Folder", "Hollow Knight"],
      ["Xbox", "Starfield"], ["Steam", "Baldur's Gate 3"], ["EA", "It Takes Two"], ["Steam", "Hades II"],
      ["Ubisoft", "Assassin's Creed"], ["itch.io", "Celeste"], ["Battle.net", "Diablo IV"], ["Folder", "Stardew Valley"]];
    const tiles = GAMES.map(([s, n], i) => ({ el: mk("tile", `<div><small>${s}</small>${n}</div>`), x: (i % 4 - 1.5) * 128, y: (Math.floor(i / 4) - 1) * 166, hero: i === 5 }));
    const cart = mk("cart", `<div class="socket"></div><h3>Baldur's Gate 3</h3><div class="who">Nobody's files yet</div>
      <div class="rows"><div class="row"><span>Save</span><b data-k="save">None</b></div><div class="row"><span>Settings</span><b data-k="set">None</b></div><div class="row"><span>Achievements</span><b data-k="ach">None</b></div></div>`);
    const thread = mk("thread", `<svg width="400" height="400" viewBox="0 0 400 400"><path d="" fill="none" stroke="#3a3e4a" stroke-width="2" stroke-dasharray="4 5"/></svg>`);
    const g1 = mk("gem", `P1<em>Player 1</em>`, `--pc:${P1}`), g2 = mk("gem", `P2<em>Player 2</em>`, `--pc:${P2}`), gr = mk("gem run", `H<em>hardcore</em>`, `--pc:${P1}`);
    const tray = mk("tray", `<span class="tr"></span><span class="tt"></span>`);
    const SOCKET = [0, -175], H1 = [-220, 150], H2 = [220, 150], RUN = [-150, 250];
    const WHO = {
      p1: { c: P1, who: "Player 1's files", save: "Act 3, Lower City", set: "Mouse, 1440p", ach: "7 of 54" },
      p2: { c: P2, who: "Player 2's files", save: "Act 1, the Grove", set: "Controller", ach: "2 of 54" },
      run: { c: P1, who: "Player 1, hardcore", save: "Act 2, Honour Mode", set: "Mouse, 1440p", ach: "11 of 54" },
    };
    const K = {
      g1: [[0.9, { x: H1[0], y: H1[1] + 60, o: 0, s: 0.6 }], [1.4, { y: H1[1], o: 1, s: 1 }], [2.1, {}], [2.6, { x: SOCKET[0], y: SOCKET[1] }], [3.05, {}], [3.4, { x: H1[0], y: H1[1] }]],
      g2: [[1.2, { x: H2[0], y: H2[1] + 60, o: 0, s: 0.6 }], [1.7, { y: H2[1], o: 1, s: 1 }], [3.4, {}], [3.85, { x: SOCKET[0], y: SOCKET[1] }], [4.05, {}], [4.35, { x: H2[0], y: H2[1] }]],
      gr: [[4.0, { x: H1[0], y: H1[1], o: 0, s: 0.4 }], [4.3, { x: RUN[0], y: RUN[1], o: 1, s: 1 }], [4.45, {}], [4.85, { x: SOCKET[0], y: SOCKET[1], s: 1.2 }]],
      cart: [[0.35, { o: 0, s: 0.45 }], [0.85, { o: 1, s: 1 }]],
      tray: [[3.0, { o: 0 }], [3.2, { o: 1 }]],
    };
    let who;
    return (t) => {
      tiles.forEach((g) => {
        const f = clamp((t - 0.2) / 0.6), e = ease(f);
        if (g.hero) put(g.el, { x: lerp(g.x, 0, e), y: lerp(g.y, 0, e), s: lerp(1, 2, e), o: 1 - clamp(f * 1.6) });
        else put(g.el, { x: g.x * (1 + e * 0.6), y: g.y * (1 + e * 0.6), s: lerp(1, 0.7, e), o: 1 - e });
        g.el.style.zIndex = g.hero ? 2 : 1;
      });
      put(cart, at(K.cart, t)); put(g1, at(K.g1, t)); put(g2, at(K.g2, t)); put(tray, at(K.tray, t));
      const r = at(K.gr, t); put(gr, r);
      thread.querySelector("path").setAttribute("d", r.o > 0.05 ? `M ${H1[0] + 200} ${H1[1] + 200} Q ${H1[0] + 190} ${r.y + 240} ${r.x + 200} ${r.y + 200}` : "");
      thread.style.opacity = r.o;
      const w = t >= 4.85 ? "run" : between(t, 3.85, 4.35) ? "p2" : between(t, 2.6, 3.25) ? "p1" : null;
      if (w !== who) {
        who = w;
        const d = w && WHO[w];
        cart.classList.toggle("lit", !!d);
        cart.style.setProperty("--c", d ? d.c : "var(--line-strong)");
        cart.querySelector(".who").textContent = d ? d.who : "Nobody's files yet";
        cart.querySelectorAll("b[data-k]").forEach((b) => { b.textContent = d ? d[b.dataset.k] : "None"; });
      }
      // a copy at every swap; scrolling back takes them away again
      const want = (t >= 3.25 ? 1 : 0) + (t >= 4.35 ? 1 : 0), tr = tray.querySelector(".tr");
      while (tr.children.length > want) tr.lastChild.remove();
      while (tr.children.length < want) tr.appendChild(document.createElement("s"));
      tray.querySelector(".tt").textContent = want ? `${want} ${want === 1 ? "copy" : "copies"} kept` : "";
      return w ? WHO[w].c : null;
    };
  };

  // ------------------------------------------ story 2: around the game ---
  const more = (mk) => {
    const cart = mk("cart lit", `<div class="socket"></div><h3>Baldur's Gate 3</h3><div class="who"><b class="ver" style="font-weight:600">Version 4.1.1</b></div>
      <div class="rows"><div class="row"><span>Achievements</span><b class="ach">11 of 54</b></div><div class="row"><span>Save points</span><b class="pts">2</b></div><div class="row"><span>Mods</span><b class="mods">None</b></div></div>`, `--c:${P1}`);
    const gem = mk("gem", `P1<em>Player 1</em>`, `--pc:${P1}`);
    const toast = mk("toast", `<span class="badge"><svg viewBox="0 0 20 20"><path d="M6 3.5h8v3.2a4 4 0 0 1-8 0zM6 5H3.5v1a2.5 2.5 0 0 0 2.6 2.5M14 5h2.5v1a2.5 2.5 0 0 1-2.6 2.5M10 10.7v3M7 16.5h6M8 13.7h4l.5 2.8h-5z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round" stroke-linecap="round"/></svg></span>
      <div><small>Achievement unlocked</small><b>Critical hit</b><span>Rare, 4.1% of players</span></div>`);
    const keys = [["Ctrl", -96], ["Shift", -8], ["F5", 74]].map(([k, x]) => ({ el: mk("kc", k), x }));
    const line = mk("line", "");
    const pins = [["Act 1", -180], ["Act 2", -60], ["Before the boss", 110]].map(([n, x], i) => ({ el: mk("pin" + (i === 2 ? " new" : ""), `<i></i><em>${n}</em>`), x, fresh: i === 2 }));
    const mods = [["Better camera", -86], ["Bigger party", -30], ["Fast travel", 26]].map(([n, y], i) => ({ el: mk("mod", `${n}<span>${i + 1}</span>`), y, i }));
    const undo = mk("chipline", `<span>Old files kept</span><b>Undo</b>`);
    const tv = mk("tv", "");
    const pads = mk("pads", `<span><i>A</i>Play</span><span><i>Y</i>Someone else</span><span><i class="wide">Start</i>Big Picture</span>`);
    const fadeOut = (a, b) => [[a, { o: 1 }], [b, { o: 0 }]];
    return (t) => {
      // the game, with Player 1 in it; on the couch it goes onto a TV
      put(cart, at([[3.9, { s: 1 }], [4.45, { s: 0.78, y: -20 }]], t));
      put(gem, at([[3.9, { y: -175, s: 1 }], [4.45, { y: -20 - 175 * 0.78, s: 0.78 }]], t));
      // 0: an achievement unlocks
      put(toast, at([[0.05, { x: 260, y: -195, o: 0 }], [0.3, { x: 160, o: 1 }], [0.85, {}], [1.05, { y: -235, o: 0 }]], t));
      bump(cart.querySelector(".ach"), t >= 0.3 ? "12 of 54" : "11 of 54");
      // 1: Ctrl+Shift+F5, and a new point on the timeline
      keys.forEach((k, i) => {
        put(k.el, at([[0.95, { x: k.x, y: 245, o: 0 }], [1.15, { y: 225, o: 1 }], [1.9, {}], [2.05, { o: 0 }]], t));
        k.el.classList.toggle("down", between(t, 1.3 + i * 0.06, 1.6));
      });
      put(line, at([[1.0, { y: 292, o: 0, s: 0.6 }], [1.2, { o: 1, s: 1 }], [1.9, {}], [2.05, { o: 0 }]], t));
      pins.forEach((p) => put(p.el, at(p.fresh ? [[1.5, { x: p.x, y: 262, o: 0 }], [1.65, { y: 292, o: 1 }], [1.9, {}], [2.05, { o: 0 }]]
        : [[1.05, { x: p.x, y: 292, o: 0 }], [1.2, { o: 1 }], [1.9, {}], [2.05, { o: 0 }]], t)));
      bump(cart.querySelector(".pts"), t >= 1.6 ? "3" : "2");
      // 2: three mods go in
      mods.forEach((m) => put(m.el, at([[2.0 + m.i * 0.05, { x: 300, y: m.y, o: 0 }], [2.2 + m.i * 0.05, { x: 215, o: 1 }], [2.45 + m.i * 0.08, {}], [2.8 + m.i * 0.08, { x: 0, y: 60, s: 0.3, o: 0 }]], t)));
      bump(cart.querySelector(".mods"), t >= 3.0 ? "3 on" : "None");
      // 3: an update you bring, with the old files kept
      bump(cart.querySelector(".ver"), t >= 3.35 ? "Version 4.1.2" : "Version 4.1.1");
      put(undo, at([[3.25, { y: 250, o: 0 }], [3.45, { y: 232, o: 1 }], [3.85, {}], [3.95, { o: 0 }]], t));
      // 4: on the couch
      put(tv, at([[3.9, { y: -20, o: 0, s: 1.25 }], [4.45, { o: 1, s: 1 }]], t));
      put(pads, at([[4.3, { y: 262, o: 0 }], [4.6, { y: 242, o: 1 }]], t));
      return P1;
    };
  };

  // ------------------------------------------------------------- clock ---
  const stages = [];
  document.querySelectorAll(".story").forEach((s) => stages.push(stage(s, s.dataset.story === "more" ? more : saves)));
  function frame() {
    let color = null;
    for (const st of stages) { const r = st.tick(); if (r.active) color = r.color; }
    root.style.setProperty("--c", color || "#3a3e4a");
  }
  let queued = false;
  addEventListener("scroll", () => { if (!queued) { queued = true; requestAnimationFrame(() => { queued = false; frame(); }); } }, { passive: true });
  addEventListener("resize", () => { stages.forEach((s) => s.fit()); frame(); });
  stages.forEach((s) => s.fit());
  frame();

  // -------------------------------------------------- latest version ---
  const ver = document.getElementById("ver");
  if (ver) {
    fetch("https://api.github.com/repos/c-dinca/lootnest/releases/latest", { headers: { Accept: "application/vnd.github+json" } })
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        const tag = j && typeof j.tag_name === "string" ? j.tag_name.replace(/^v/, "") : "";
        if (/^\d+\.\d+\.\d+$/.test(tag)) ver.textContent = `Version ${tag}, in beta.`;
      })
      .catch(() => {});
  }
})();
