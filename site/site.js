// Lootnest's site: two scroll stories, each played in a Lootnest window
// drawn in HTML and CSS (the launcher's own layout, in the site's palette).
// Scrolling moves a clock from 0 to 5, one unit per caption. The pointer
// and the moving parts follow keyframes on that clock; the rest are states
// the clock switches on and off, so scrolling back undoes them. Apart from
// a caret and the scroll hint, nothing plays on its own.
(() => {
  "use strict";
  const root = document.documentElement;
  const P1 = "#6c9bd2", P2 = "#e88ca0";

  // ------------------------------------------------------------ engine ---
  const lerp = (a, b, f) => a + (b - a) * f;
  const ease = (f) => (f < 0.5 ? 4 * f * f * f : 1 - Math.pow(-2 * f + 2, 3) / 2);
  const clamp = (v, a = 0, b = 1) => Math.min(b, Math.max(a, v));
  const within = (t, a, b) => t >= a && t < b;
  // keys: [[t, {x, y, o}], ...]; a frame keeps what it doesn't set
  function at(keys, t) {
    let prev = { x: 0, y: 0, o: 1, ...keys[0][1] };
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
  const tog = (el, cls, on) => el && el.classList.toggle(cls, !!on);
  const txt = (el, s) => { if (el && el.textContent !== s) el.textContent = s; };

  // ------------------------------------------------------------ pieces ---
  const art = (n) => `<img src="img/art/${n}.webp" alt="" loading="lazy" decoding="async">`;
  const GEM = `<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M2.75 10.25c0 3.7 3.25 6.25 7.25 6.25s7.25-2.55 7.25-6.25"/><path d="M5.75 10.25c0 2.2 1.9 3.5 4.25 3.5s4.25-1.3 4.25-3.5"/><circle cx="10" cy="6.25" r="2.5" stroke="#ef233c"/></svg>`;
  const PLAY = `<svg viewBox="0 0 20 20"><path d="M5 3.5v13l11-6.5z" fill="currentColor"/></svg>`;
  const TROPHY = `<svg viewBox="0 0 20 20"><path d="M6 3.5h8v3.2a4 4 0 0 1-8 0zM6 5H3.5v1a2.5 2.5 0 0 0 2.6 2.5M14 5h2.5v1a2.5 2.5 0 0 1-2.6 2.5M10 10.7v3M7 16.5h6M8 13.7h4l.5 2.8h-5z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round" stroke-linecap="round"/></svg>`;
  const FLAG = `<svg viewBox="0 0 20 20"><path d="M5 17V3.5m0 0h9l-2 3.25 2 3.25H5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round"/></svg>`;
  const ARROW = `<svg viewBox="0 0 22 22"><path d="M3 2l15 9-6.5 1.3L15 19l-3 1.4-3.4-6.8L3 18z" fill="#fff" stroke="#111217" stroke-width="1.4" stroke-linejoin="round"/></svg>`;

  const bar = `<div class="a-bar"><span class="a-brand">${GEM}Lootnest</span>
    <div class="a-tabs"><span data-tab="home">Home</span><span data-tab="library">Library</span><span data-tab="store">Store</span><span data-tab="updates">Updates</span><span data-tab="ach">Achievements</span><span data-tab="saves">Saves</span></div>
    <span class="a-search">Search<kbd>Ctrl K</kbd></span><span class="a-seat">P1</span></div>`;
  const si = (n, name, pc, on) => `<div class="a-si${on ? " on" : ""}"><i>${art(n)}</i>${name}${pc ? `<b style="--pc:${pc}"></b>` : ""}</div>`;
  const side = `<div class="a-side"><h6>All games</h6><h6>Recent</h6>
    ${si("bg3", "Baldur's Gate 3", P1, true)}${si("cs2", "Counter-Strike 2", P2)}${si("ittakestwo", "It Takes Two", P2)}${si("hollow", "Hollow Knight")}
    <h6>Favourites</h6>${si("stardew", "Stardew Valley")}${si("hades2", "Hades II")}
    <h6>Co-op</h6>${si("ittakestwo", "It Takes Two", P2)}${si("forza", "Forza Horizon 5")}${si("gtav", "Grand Theft Auto V")}</div>`;
  const cursor = `<div class="a-cur">${ARROW}</div>`;

  const prof = (cls, letter, name, sub, h, pc) => `<div class="a-prof ${cls}" style="--pc:${pc}"><span class="a-av">${letter}</span><div><b>${name}</b><small>${sub}</small></div><em>${h}</em></div>`;
  const gamePage = (left) => `<section class="a-page" data-page="game">
      <div class="a-hero">${art("bg3-wide")}<h4>Baldur's Gate 3</h4><small>Steam&nbsp;&nbsp;&nbsp;About 66 GB on this PC</small></div>
      <div class="a-play"><span class="a-key go" data-k="play">${PLAY}<span>Play as Player 1</span></span>
        <div class="info"><b data-k="loaded">Player 1's files are loaded</b><span data-k="when"><i></i><span>Played 2 days ago</span></span></div>
        <span class="sp"></span><span class="a-key" data-k="else">Someone else</span><span class="a-key" data-k="newrun">+ New run</span></div>
      <div class="a-gtabs"><span data-gt="overview">Overview</span><span>Achievements</span><span>Saves</span><span data-gt="mods">Mods</span><span>Timeline</span></div>
      <div class="a-gbody"><div class="a-pane">${left}</div>
        <div><p class="a-label">Profiles</p>
          ${prof("p1", "P1", "Player 1", "Played 2 days ago", "44 h", P1)}
          ${prof("p2", "P2", "Player 2", "Played 3 days ago", "88 h", P2)}
          ${prof("run", "H", "hardcore", "Player 2's run", "0 h", P2)}</div></div>
    </section>`;
  const facts = `<div class="a-sub on" data-sub="overview"><p class="a-label">Details</p><div class="a-facts">
      <div><span>Developer</span>Larian Studios</div><div><span>Released</span>3 Aug 2023</div><div><span>Controller</span>Full support</div><div><span>Achievements</span>11 of 54</div></div></div>`;

  // ----------------------------------------------- story 1: profiles ---
  const COVERS = [["cs2", "Counter-Strike 2", "Steam"], ["ittakestwo", "It Takes Two", "EA"], ["cyberpunk", "Cyberpunk 2077", "GOG"], ["bg3", "Baldur's Gate 3", "Steam"],
    ["stardew", "Stardew Valley", "GOG"], ["hollow", "Hollow Knight", "Folder"], ["hades2", "Hades II", "Epic"], ["forza", "Forza Horizon 5", "Xbox"],
    ["gtav", "Grand Theft Auto V", "Rockstar"], ["isaac", "The Binding of Isaac", "Steam"], ["forest", "The Forest", "Steam"], [null, "Celeste", "itch.io"]];
  const pk = (cls, letter, name, sub, pc) => `<div class="a-pk ${cls}" style="--pc:${pc}"><span class="a-av">${letter}</span><b>${name}</b><small>${sub}</small></div>`;
  function story1(app) {
    app.innerHTML = bar + side + `<div class="a-view">
      <section class="a-page a-lib" data-page="lib"><div class="a-h"><h4>Library</h4><span>12 games</span></div><div class="a-grid">
        ${COVERS.map(([n, name, src]) => `<div class="a-cov"${n === "bg3" ? ' data-k="bg3"' : ""}><div class="a-art">${n ? art(n) : `<div class="a-plate">${name}</div>`}</div><p>${name}</p><small>${src}</small></div>`).join("")}
      </div></section>
      ${gamePage(facts)}</div>
      <div class="a-over a-pick"><h5>Who's playing Baldur's Gate 3?</h5><div class="row">
        ${pk("", "P1", "Player 1", "Played 2 days ago", P1)}${pk("", "P1", "100%", "Player 1's run", P1)}${pk("", "P2", "Player 2", "Played 3 days ago", P2)}${pk("new", "+", "New profile", "Someone new, or a run", "#a6aeb9")}
      </div><p>Press 1 to 4, or Esc to cancel</p></div>
      <div class="a-over a-swap"><div class="box"><span class="a-av" data-k="swapav">P1</span><h5 data-k="swaph">Loading Player 1's files</h5>
        <div class="a-step"><i></i><span data-k="s0"></span></div><div class="a-step"><i></i><span>A copy is made and read back</span></div><div class="a-step"><i></i><span data-k="s2"></span></div>
        <div class="a-bar2"><i></i></div></div></div>
      <div class="a-over a-dlg"><div class="box"><h5>New run of Player 2</h5><label>Name</label><div class="a-field"><span data-k="typed"></span><i></i></div>
        <div class="a-radio on"><i></i>Start from a copy of Player 2's files</div><div class="a-radio"><i></i>Start fresh</div>
        <div class="foot"><span class="a-key">Cancel</span><span class="a-key red" data-k="start">Start the run</span></div></div></div>
      ${cursor}`;
    const $ = (s) => app.querySelector(s), $$ = (s) => [...app.querySelectorAll(s)];
    const k = (n) => $(`[data-k="${n}"]`);
    const covers = $$(".a-cov"), lib = $('[data-page="lib"]'), game = $('[data-page="game"]');
    const pick = $(".a-pick"), pks = $$(".a-pk"), swap = $(".a-swap"), dlg = $(".a-dlg"), cur = $(".a-cur");
    tog($('[data-tab="library"]'), "on", true); tog($('[data-gt="overview"]'), "on", true);
    const SEATS = { p1: { pc: P1, name: "Player 1", key: "Play as Player 1", av: "P1" }, p2: { pc: P2, name: "Player 2", key: "Play as Player 2", av: "P2" }, run: { pc: P2, name: "Player 2, hardcore", key: "Play as hardcore", av: "H" } };
    return (t, spot) => {
      tog(lib, "on", t < 0.95); tog(game, "on", t >= 0.95);
      covers.forEach((c, i) => tog(c, "in", t > 0.04 + i * 0.035));
      tog(k("bg3"), "press", within(t, 0.82, 0.92));
      tog($(".a-prof.p1"), "in", t >= 1.12); tog($(".a-prof.p2"), "in", t >= 1.28); tog($(".a-prof.run"), "in", t >= 4.74);
      const seat = SEATS[t >= 4.72 ? "run" : t >= 3.96 ? "p2" : "p1"];
      app.style.setProperty("--pc", seat.pc); app.style.setProperty("--seat", seat.pc);
      k("play").style.setProperty("--pc", seat.pc);
      txt(k("play").lastChild, seat.key);
      txt($(".a-seat"), seat.av);
      const playing = within(t, 2.95, 3.12);
      txt(k("loaded"), playing ? "Player 1 is playing" : t >= 4.72 ? "A copy of Player 2's files" : `${seat.name}'s files are loaded`);
      txt(k("when").lastChild, playing ? "Baldur's Gate 3 is running" : t >= 4.72 ? "New run, ready" : "Played 2 days ago");
      tog(k("when"), "live", playing);
      tog(k("play"), "down", within(t, 2.15, 2.24)); tog(k("else"), "down", within(t, 3.17, 3.24)); tog(k("newrun"), "down", within(t, 4.14, 4.2));
      // Who's playing
      tog(pick, "on", within(t, 3.25, 3.62));
      tog(pks[2], "hot", t >= 3.42); tog(pks[2], "down", within(t, 3.48, 3.56));
      // the swap, twice
      const first = within(t, 2.25, 2.95), second = within(t, 3.62, 4.0), on = first || second;
      tog(swap, "on", on);
      if (on) {
        const [a, len] = first ? [2.25, 0.7] : [3.62, 0.38], p = clamp((t - a) / len);
        const to = first ? SEATS.p1 : SEATS.p2;
        $(".a-swap .a-av").style.setProperty("--pc", to.pc); txt(k("swapav"), to.av);
        txt(k("swaph"), `Loading ${to.name}'s files`);
        txt(k("s0"), first ? "What's in the game goes back to its slot" : "Player 1's files go back to their slot");
        txt(k("s2"), `${to.name}'s files go in`);
        $$(".a-swap .a-step").forEach((s, i) => tog(s, "done", p > 0.22 + i * 0.28));
        $(".a-swap .a-bar2 i").style.setProperty("--p", p);
      }
      // New run
      tog(dlg, "on", within(t, 4.22, 4.7));
      txt(k("typed"), "hardcore".slice(0, Math.max(0, Math.floor((t - 4.28) / 0.022))));
      tog(k("start"), "down", within(t, 4.6, 4.66));
      // the pointer
      const P = (n) => spot(n);
      const c = at([[0.4, { ...P("corner"), o: 0 }], [0.5, { o: 1 }], [0.8, P("bg3")], [1.0, {}], [1.55, P("profiles")], [2.05, P("play")], [2.24, {}], [2.3, { o: 0 }],
        [2.95, { o: 0 }], [3.02, { o: 1 }], [3.15, P("else")], [3.3, {}], [3.45, P("pk2")], [3.58, {}], [3.64, { o: 0 }], [3.98, { o: 0 }], [4.02, { o: 1 }],
        [4.12, P("newrun")], [4.26, {}], [4.5, P("start")], [4.68, {}], [4.85, P("corner")]], t);
      cur.style.transform = `translate(${c.x}px, ${c.y}px)`; cur.style.opacity = c.o;
      tog(cur, "click", [[0.82, 0.9], [2.15, 2.23], [3.17, 3.24], [3.48, 3.55], [4.14, 4.2], [4.6, 4.66]].some(([a, b]) => within(t, a, b)));
      return seat.pc;
    };
  }
  const spots1 = { bg3: '[data-k="bg3"] .a-art', play: '[data-k="play"]', else: '[data-k="else"]', newrun: '[data-k="newrun"]', profiles: ".a-prof.p2", pk2: ".a-pk:nth-child(3)", start: '[data-k="start"]' };

  // ------------------------------------------ story 2: around the game ---
  const mod = (name, files) => `<div class="a-modrow"><div><b>${name}</b><small>${files}</small></div><span class="a-sw"></span></div>`;
  const urow = (th, name, src, have, out, act) => `<div class="a-urow"><span class="th">${th}</span><div class="nm"><b>${name}</b><small>${src}</small></div>
      <div class="vr"><b data-u="${name}">${have}</b><span>${out}</span></div><div class="act">${act}</div></div>`;
  function story2(app) {
    app.innerHTML = bar + side + `<div class="a-view">
      ${gamePage(`<div class="a-sub" data-sub="mods"><p class="a-label">Player 1's mods</p>
        ${mod("Better camera", "12 files")}${mod("Bigger party", "3 files")}${mod("Fast travel anywhere", "1 file")}
        <div class="a-modfoot"><span class="a-key red" data-k="apply">Apply</span><span data-k="modst">3 changes</span></div></div>`)}
      <section class="a-page a-upd" data-page="upd"><div class="a-h"><h4>Updates</h4><span>For games no launcher updates</span></div>
        ${urow(art("hollow"), "Hollow Knight", "Folder", "1.5.68", "1.5.78 is out", `<span class="a-key red" data-k="upd">Apply update</span><span class="a-bar2 hide" data-k="updbar"><i></i></span><span class="done hide" data-k="done"><span class="a-chip">Old files kept</span><span class="a-key">Undo</span></span>`)}
        ${urow(art("stardew"), "Stardew Valley", "GOG, installed by Lootnest", "1.6.15", "Up to date", `<span class="a-lamp"></span>`)}
        ${urow(`<div class="a-plate" style="font-size:11px;padding:5px">Celeste</div>`, "Celeste", "itch.io, installed by Lootnest", "1.4.0.0", "Up to date", `<span class="a-lamp"></span>`)}
      </section></div>
      <div class="a-game">${art("bg3-wide")}
        <div class="a-toast" data-k="ach"><span class="badge">${TROPHY}</span><div><small>Achievement unlocked</small><b>Critical hit</b><span>Rare, 4.1% of players</span></div></div>
        <div class="a-toast" data-k="point" style="top:100px"><span class="badge plain">${FLAG}</span><div><small>Save point made</small><b>Before the boss</b><span class="plain">Act 3, Lower City</span></div></div>
        <div class="a-keys"><span class="a-kc">Ctrl</span><span class="a-kc">Shift</span><span class="a-kc">F5</span></div></div>
      <div class="a-bp"><div class="tp"><span class="a-brand">${GEM}Lootnest</span><span class="on">Home</span><span>Library</span><span>Achievements</span><span>Profiles</span></div>
        <h6>Continue playing</h6><div class="wides">${["bg3-wide", "cs2-wide", "ittakestwo-wide", "hollow-wide", "cyberpunk-wide"].map((n) => `<span class="wide">${art(n)}</span>`).join("")}</div>
        <h6>Weekend</h6><div class="covs">${["stardew", "hades2", "gtav", "forza", "isaac", "forest", "cs2", "hollow"].map((n) => `<i>${art(n)}</i>`).join("")}</div>
        <div class="hints"><span><i>A</i>Select</span><span><i>B</i>Back</span><span><i>Y</i>Play as someone else</span></div></div>
      ${cursor}`;
    const $ = (s) => app.querySelector(s), $$ = (s) => [...app.querySelectorAll(s)];
    const k = (n) => $(`[data-k="${n}"]`);
    const game = $('[data-page="game"]'), upd = $('[data-page="upd"]'), screen = $(".a-game"), bp = $(".a-bp"), cur = $(".a-cur");
    $$(".a-prof.p1, .a-prof.p2").forEach((p) => p.classList.add("in"));
    tog($('[data-gt="mods"]'), "on", true); tog($('[data-sub="overview"]'), "on", false);
    app.style.setProperty("--pc", P1);
    return (t, spot) => {
      // 0 and 1: in the game
      tog(screen, "on", t < 2.0);
      tog(k("ach"), "on", within(t, 0.22, 0.95));
      tog($(".a-keys"), "on", within(t, 1.02, 1.9));
      $$(".a-kc").forEach((kc, i) => tog(kc, "down", within(t, 1.25 + i * 0.05, 1.55)));
      tog(k("point"), "on", within(t, 1.45, 1.95));
      // 2: Mods
      tog(game, "on", within(t, 2.0, 3.0)); tog($('[data-sub="mods"]'), "on", true);
      $$(".a-modrow").forEach((r, i) => { tog(r, "in", t >= 2.08 + i * 0.06); tog(r.querySelector(".a-sw"), "on", t >= 2.3 + i * 0.08); });
      tog(k("apply"), "down", within(t, 2.64, 2.7));
      txt(k("modst"), t >= 2.72 ? "3 mods on. Put the game back any time." : "3 changes");
      // 3: Updates
      tog(upd, "on", within(t, 3.0, 4.0));
      tog($('[data-tab="library"]'), "on", t < 3.0 && t >= 2.0); tog($('[data-tab="updates"]'), "on", within(t, 3.0, 4.0));
      const p = clamp((t - 3.3) / 0.3), done = t >= 3.62, going = within(t, 3.3, 3.62);
      tog(k("upd"), "down", within(t, 3.22, 3.28)); tog(k("upd"), "hide", going || done);
      tog(k("updbar"), "hide", !going); k("updbar").firstElementChild.style.setProperty("--p", p);
      tog(k("done"), "hide", !done);
      txt($('[data-u="Hollow Knight"]'), done ? "1.5.78" : "1.5.68");
      txt($('[data-u="Hollow Knight"]').nextElementSibling, done ? "Up to date" : "1.5.78 is out");
      // 4: Big Picture
      tog(bp, "on", t >= 4.0); tog(app, "is-bp", t >= 4.0);
      const f = clamp(Math.floor((t - 4.25) / 0.17), 0, 3);
      $$(".a-bp .wide").forEach((w, i) => tog(w, "focus", t >= 4.1 && i === f));
      $(".a-bp .wides").style.transform = `translateX(${-Math.max(0, f - 1) * 266}px)`;
      // the pointer, for Apply and Apply update
      const c = at([[2.45, { ...spot("corner"), o: 0 }], [2.5, { o: 1 }], [2.62, spot("apply")], [2.8, {}], [3.12, spot("upd")], [3.3, {}], [3.36, { o: 0 }]], t);
      cur.style.transform = `translate(${c.x}px, ${c.y}px)`; cur.style.opacity = c.o;
      tog(cur, "click", within(t, 2.64, 2.7) || within(t, 3.22, 3.28));
      return P1;
    };
  }
  const spots2 = { apply: '[data-k="apply"]', upd: '[data-k="upd"]' };

  // ------------------------------------------------------------- stage ---
  function stage(section, build, spots) {
    const world = section.querySelector(".world"), scene = section.querySelector(".scene");
    const caps = [...section.querySelectorAll(".cap")], bars = [...section.querySelectorAll(".steps b")];
    const app = document.createElement("div"); app.className = "app"; world.appendChild(app);
    const render = build(app);
    let k = 1, W = 1000, step = -1;
    // where a part of the window is, in the window's own pixels
    const spot = (name) => {
      if (name === "corner") return { x: W - 60, y: 560 };
      const el = app.querySelector(spots[name]); if (!el) return { x: 500, y: 300 };
      const a = app.getBoundingClientRect(), r = el.getBoundingClientRect();
      return { x: (r.left - a.left + r.width * 0.55) / k, y: (r.top - a.top + r.height * 0.6) / k };
    };
    return {
      fit() {
        scene.style.height = "";
        W = innerWidth <= 700 ? 640 : 1000;
        app.classList.toggle("compact", W < 1000);
        world.style.setProperty("--ww", W + "px");
        const w = scene.clientWidth, h = scene.clientHeight || innerHeight * 0.6;
        k = Math.min(w / W, h / 640, 1.15);
        world.style.setProperty("--k", k);
        if (innerWidth <= 900) scene.style.height = Math.round(640 * k) + "px";
      },
      tick() {
        const r = section.getBoundingClientRect();
        const near = r.bottom > -innerHeight && r.top < innerHeight * 2;
        const t = Math.min(5, clamp(-r.top / (r.height - innerHeight)) * 5.2);
        const s = Math.min(4, Math.floor(t));
        if (s !== step) { step = s; caps.forEach((c, i) => c.classList.toggle("on", i === s)); }
        bars.forEach((b, i) => b.style.setProperty("--f", clamp(t - i)));
        const color = near ? render(t, spot) : null;
        return { color, active: r.top < innerHeight * 0.5 && r.bottom > innerHeight * 0.5 };
      },
    };
  }

  const stages = [];
  document.querySelectorAll(".story").forEach((s) => stages.push(s.dataset.story === "more" ? stage(s, story2, spots2) : stage(s, story1, spots1)));
  function frame() {
    let color = null;
    for (const st of stages) { const r = st.tick(); if (r.active && r.color) color = r.color; }
    root.style.setProperty("--c", color || "#3a3e4a");
  }
  let queued = false;
  addEventListener("scroll", () => { if (!queued) { queued = true; requestAnimationFrame(() => { queued = false; frame(); }); } }, { passive: true });
  addEventListener("resize", () => { stages.forEach((s) => s.fit()); frame(); });
  stages.forEach((s) => s.fit());
  frame();
  document.fonts && document.fonts.ready.then(frame);

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
