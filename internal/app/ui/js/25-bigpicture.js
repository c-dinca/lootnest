  // ----------------------------------------------------------- big picture --
  // Big Picture: Lootnest as a console's screen, on the whole screen, for a
  // controller from the sofa. Tabs along the top (LB and RB), rows of games
  // to move through with the stick or the D-pad, A to choose, B to go back,
  // Y to play as someone else, Start to leave. The keyboard works the same:
  // arrows, Enter, Escape, Q and E.
  const BP = { on: false, tab: "home", tag: "", detail: null, choose: null, focus: "", from: "", keep: false, timer: 0, sounds: pref("bpSounds", "1") === "1" };
  const BP_TABS = [["home", "Home"], ["library", "Library"], ["achievements", "Achievements"], ["profiles", "Profiles"]];
  const BP_SEC = { home: "home", library: "library", achievements: "ach", profiles: "profiles" };
  const bpItem = (k) => libraryItems().find((x) => x.kind + ":" + x.id === k) || null;
  const bpKey = (it) => it.kind + ":" + it.id;
  function bpOpen() {
    if (BP.on) return;
    BP.on = true; BP.tab = "home"; BP.tag = ""; BP.detail = null; BP.choose = null; BP.focus = ""; BP.from = "";
    closeContext(); gqClose();
    if (!S.allAch) api.allAchievements().catch(() => {});
    api.bigPicture(true).catch(() => {});
    renderBP(true);
    sfx("open");
  }
  function bpClose() {
    if (!BP.on) return;
    BP.on = false;
    const el = $("#bp");
    if (el) { el.removeAttribute("id"); el.classList.add("out"); setTimeout(() => el.remove(), 320); }
    api.bigPicture(false).catch(() => {});
    sfx("close");
    if (behindBP) render();
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
  function bpArt(it, wide) {
    loadArt(it); ensureCover(it);
    return artFor(it, wide ? "hero" : "cover");
  }
  function bpTile(it, wide) {
    const a = bpArt(it, wide), kind = wide ? "hero" : "cover";
    const s = it.g && session(it.g.id);
    return `<button class="bp-tile${wide ? " bp-tile--wide" : ""}" data-bp data-bpid="t:${esc(bpKey(it))}:${wide ? "w" : "c"}" data-k="${esc(bpKey(it))}" data-go="open">
      <span class="bp-tile-art">${a.url ? artImg(a, kind) : typeCover(it, "", { icon: true })}</span>
      ${s ? `<span class="bp-live">${esc(sessionText(s))}</span>` : ""}<span class="bp-tile-name">${esc(it.name)}</span></button>`;
  }
  // A collection too small for a row of its own: one wide tile with its first three covers and its name.
  function bpTagTile(t, list) {
    const covers = list.slice(0, 3).map((it) => {
      const a = bpArt(it, false);
      return `<span class="bp-tag-cover">${a.url ? artImg(a, "cover") : plate(S.icons[coverKey(it)], "", "")}</span>`;
    }).join("");
    return `<button class="bp-tile bp-tile--wide bp-tile--tag" data-bp data-bpid="c:${esc(t)}" data-go="tag" data-tag="${esc(t)}">
      <span class="bp-tile-art">${covers}<span class="bp-tag-name"><b>${esc(t)}</b>${list.length} ${list.length === 1 ? "game" : "games"}</span></span></button>`;
  }
  const bpRow = (id, title, list, wide) => (list.length ? `<section class="bp-row"><h2><span class="ln-label">${esc(title)}</span></h2><div class="bp-row-track" data-row="${esc(id)}">${list.map((it) => bpTile(it, wide)).join("")}</div></section>` : "");
  function bpScreen() {
    const all = libraryItems();
    if (BP.choose) {
      const it = bpItem(BP.choose);
      const ps = cfg().players;
      return `<div class="bp-choose"><p class="bp-kicker">${esc(it ? it.name : "")}</p><h1>Who's playing?</h1>
        <div class="bp-people">${ps.map((p) => `<button class="bp-person ln-player" style="--player:${esc(p.color)}" data-bp data-bpid="who:${esc(p.id)}" data-go="play-as" data-p="${esc(p.id)}">${lnAvatar(p, "96")}<b>${esc(p.name)}</b><span>${it && it.g && S.d.active[it.g.id] === p.id ? "Files loaded" : it && it.g && hasProfile(it.g.id, p.id) ? "Has a profile" : "New profile"}</span></button>`).join("")}</div></div>`;
    }
    if (BP.detail) {
      const it = bpItem(BP.detail);
      if (!it) { BP.detail = null; return bpScreen(); }
      const key = recentKey(it), t = totalPlayed(it) || steamTime(it), seat = seated();
      const A = S.ach[key], det = cfg().settings.gameDetails && (S.details[key] || {}).details;
      achFor(it); // reads its achievements if it hasn't yet
      const logo = artFor(it, "logo").url;
      const as = it.g && seat;
      return `<div class="bp-detail">
        <div class="bp-detail-text">${logo ? `<img class="bp-logo" src="${logo}" alt="${esc(it.name)}">` : `<h1>${esc(it.name)}</h1>`}
          <p class="bp-meta">${[esc(storeNames(it)), playedAt(it) ? "Played " + esc(ago(new Date(playedAt(it)))) : "", t ? esc(hours(t)) + " played" : "", A && A.total ? `${A.unlocked} of ${A.total} achievements` : "", det && CTL_SAYS[det.controller]].filter(Boolean).map((x) => `<span>${x}</span>`).join("")}</p>
          <div class="bp-actions"><button class="bp-btn ${as ? `bp-btn--play ln-player" style="--player:${esc(seat.color)}"` : 'bp-btn--primary"'} data-bp data-bpid="play" data-go="play" data-k="${esc(bpKey(it))}">${GI.play}${as ? `Play as ${esc(seat.name)}` : "Play"}</button>
            ${it.g && cfg().players.length > 1 ? `<button class="bp-btn" data-bp data-bpid="playas" data-go="choose" data-k="${esc(bpKey(it))}">${glyph("Y")}Play as someone else</button>` : ""}
            ${bpRunning(it) ? `<button class="bp-btn" data-bp data-bpid="close" data-go="close-game" data-k="${esc(bpKey(it))}">${glyph("X")}Close the game</button>` : ""}
            <button class="bp-btn" data-bp data-bpid="back" data-go="back">${glyph("B")}Back</button></div></div></div>`;
    }
    if (!all.length && (BP.tab === "library" || BP.tab === "home")) {
      return `<div class="bp-empty"><h2>No games yet</h2>
        <button class="bp-btn" data-bp data-bpid="exit" data-go="exit">${glyph("Start")}Leave Big Picture</button></div>`;
    }
    if (BP.tab === "library") {
      const list = BP.tag ? all.filter((it) => tagsOf(it).includes(BP.tag)) : all;
      return `<div class="bp-grid-wrap"><h2 class="bp-h"><span class="ln-label">${BP.tag ? esc(BP.tag) : "All games"}<span class="ln-count ln-num">${list.length}</span></span></h2><div class="bp-grid">${list.map((it) => bpTile(it, false)).join("")}</div></div>`;
    }
    if (BP.tab === "profiles") {
      const ps = cfg().players, seat = seated(), alone = solo();
      return `<div class="bp-choose"><h1>${alone ? "Playing as" : "Who's here?"}</h1>
        <div class="bp-people">${ps.map((p) => `<button class="bp-person ln-player${seat && seat.id === p.id ? " is-seated" : ""}" style="--player:${esc(p.color)}" data-bp data-bpid="seat:${esc(p.id)}" data-go="seat" data-p="${esc(p.id)}">${lnAvatar(p, "96")}<b>${esc(p.name)}</b><span>${seat && seat.id === p.id ? (alone ? "Playing" : "Sitting here") : alone ? "Play as this" : "Sit down"}</span></button>`).join("")}</div></div>`;
    }
    if (BP.tab === "achievements") {
      const rows = (S.allAch ? S.allAch.games : []).map((g) => ({ g, it: itemByKey(g.key) })).filter((r) => r.it);
      if (!rows.length) return `<div class="bp-empty"><h2>${S.allAch ? "No achievements found yet" : "Reading achievements"}</h2></div>`;
      const who = (seated() || cfg().players[0] || {}).id;
      return `<div class="bp-grid-wrap"><h2 class="bp-h"><span class="ln-label">Achievements</span></h2><div class="bp-ach">${rows.map(({ g, it }) => {
        const own = g.players && g.players[who], u = own ? own.unlocked : g.unlocked, pct = Math.round((u / g.total) * 100);
        return `<button class="bp-achcard" data-bp data-bpid="a:${esc(bpKey(it))}" data-k="${esc(bpKey(it))}" data-go="open"><span class="ach-gring">${ring(pct, 64)}</span><span><b>${esc(g.name)}</b><span>${u} of ${g.total}</span></span></button>`; }).join("")}</div></div>`;
    }
    const recent = recentItems(all, false).slice(0, 12);
    const tags = {};
    all.forEach((it) => tagsOf(it).forEach((t) => (tags[t] = tags[t] || []).push(it)));
    const names = allTags().filter((t) => tags[t]); // the sidebar's order
    const big = names.filter((t) => tags[t].length >= 5).slice(0, 3);
    const small = names.filter((t) => !big.includes(t));
    return `<div class="bp-home">${bpRow("recent", "Continue playing", recent.length ? recent : all.slice(0, 12), true)}
      ${big.map((t) => bpRow("tag:" + t, t, tags[t], false)).join("")}
      ${small.length ? `<section class="bp-row"><h2><span class="ln-label">Collections</span></h2><div class="bp-row-track" data-row="tags">${small.map((t) => bpTagTile(t, tags[t])).join("")}</div></section>` : ""}
      ${bpRow("all", "Your games", all.slice(0, 40), false)}</div>`;
  }
  let bpSeat = "", bpFootX = false, bpFam = "";
  function renderBP(first) {
    let el = $("#bp");
    if (!el) { el = document.createElement("div"); el.id = "bp"; el.className = "bp ln-root"; el.setAttribute("role", "application"); el.setAttribute("aria-label", "Big Picture"); document.body.appendChild(el); }
    // Keep where each row and the page were scrolled, and what had focus.
    const rows = {};
    el.querySelectorAll("[data-row]").forEach((r) => (rows[r.dataset.row] = r.scrollLeft));
    let main = el.querySelector(".bp-main");
    const top = main ? main.scrollTop : 0;
    const a = document.activeElement;
    if (!BP.keep && a && a.dataset && a.dataset.bpid && el.contains(a)) BP.focus = a.dataset.bpid;
    BP.keep = false;
    const seat = seated();
    const seatHTML = seat ? `<span class="bp-seat ln-player" style="--player:${esc(seat.color)}">${lnAvatar(seat, "32")}${esc(seat.name)}</span>` : "";
    const now = new Date().toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
    const sub = BP.detail || BP.choose;
    const sound = BP.sounds ? "Sounds on" : "Sounds off";
    const fresh = !main, fam = bpFam !== pad.fam;
    bpFam = pad.fam;
    if (fresh) {
      el.innerHTML = `<div class="bp-bg" aria-hidden="true"><div class="bp-bg-l"></div><div class="bp-bg-l"></div></div>
      <header class="bp-top"><div class="bp-brand">${BRAND_MARK}<span class="bp-brand-name">Lootnest</span></div>
        <nav class="bp-tabs"><span class="bp-ind" aria-hidden="true"></span>${glyph("LB")}${BP_TABS.map(([k, l]) => `<button class="bp-tab" data-go="tab" data-t="${k}" tabindex="-1">${l}</button>`).join("")}${glyph("RB")}</nav>
        <div class="bp-right">${seatHTML}<time class="bp-clock ln-num"></time>
          <button class="bp-exit" data-go="exit" tabindex="-1" title="Leave Big Picture">${glyph("Start")}Leave</button></div></header>
      <main class="bp-main"></main>
      <footer class="bp-foot"></footer>`;
      main = el.querySelector(".bp-main");
    } else {
      if (seatHTML !== bpSeat) {
        el.querySelectorAll(".bp-right .bp-seat").forEach((x) => x.remove());
        el.querySelector(".bp-right").insertAdjacentHTML("afterbegin", seatHTML);
      }
      if (fam) {
        const [lb, rb] = el.querySelectorAll(".bp-tabs > .bp-g");
        lb.outerHTML = glyph("LB"); rb.outerHTML = glyph("RB");
        el.querySelector(".bp-exit > .bp-g").outerHTML = glyph("Start");
      }
    }
    bpSeat = seatHTML;
    const closeHint = !!BP.detail && bpRunning(bpItem(BP.detail));
    if (fresh || fam || closeHint !== bpFootX) el.querySelector(".bp-foot").innerHTML = `${glyph("A")}Select<span></span>${glyph("B")}Back<span></span>${closeHint ? `${glyph("X")}Close the game<span></span>` : ""}${glyph("Y")}Play as someone else<span></span>${glyph("LB")}${glyph("RB")}Tabs<span></span>${glyph("Start")}Leave<button class="bp-sound" data-go="sound" tabindex="-1">${glyph("View")}<span></span></button>`;
    bpFootX = closeHint;
    el.querySelector(".bp-tabs").classList.toggle("is-dim", !!sub);
    el.querySelectorAll(".bp-tab").forEach((t) => t.setAttribute("aria-current", String(BP.tab === t.dataset.t)));
    el.dataset.sec = BP_SEC[BP.tab];
    slideInd(el.querySelector(".bp-ind"), el.querySelector('.bp-tab[aria-current="true"]'));
    el.querySelector(".bp-clock").textContent = now;
    const snd = el.querySelector(".bp-sound");
    snd.setAttribute("aria-pressed", String(BP.sounds));
    snd.lastElementChild.textContent = sound;
    main.classList.toggle("bp-arrive", !!first);
    main.innerHTML = bpScreen();
    el.querySelectorAll("[data-row]").forEach((r) => { if (rows[r.dataset.row]) r.scrollLeft = rows[r.dataset.row]; });
    if (!first) main.scrollTop = top;
    let f = BP.focus && el.querySelector(`[data-bpid="${CSS.escape(BP.focus)}"]`);
    if (!f) f = el.querySelector("[data-bp]");
    if (f && !S.dialog && !S.picker) f.focus({ preventScroll: !first });
    bpArtFor(f, fresh);
  }
  // The background follows the game in focus: its art, large and soft.
  let bpBgFront = 0, bpBgUrl = "";
  function bpArtFor(el, now) {
    const it = el && el.dataset.k && bpItem(el.dataset.k);
    const url = it ? bpArt(it, true).url : BP.detail ? bpArt(bpItem(BP.detail), true).url : "";
    if (url === bpBgUrl && !now) return;
    bpBgUrl = url;
    const ls = document.querySelectorAll("#bp .bp-bg-l");
    if (ls.length < 2) return;
    const next = ls[bpBgFront ^ 1];
    if (url) next.style.backgroundImage = `url("${url}")`;
    next.classList.toggle("on", !!url); ls[bpBgFront].classList.remove("on");
    bpBgFront ^= 1;
  }
  // Moving the focus to the nearest thing in a direction, as on a console.
  function bpMove(dir) {
    const els = [...document.querySelectorAll("#bp [data-bp]")];
    if (!els.length) return;
    const cur = document.activeElement && document.activeElement.closest && document.activeElement.closest("#bp [data-bp]");
    if (!cur) { ((BP.focus && document.querySelector(`#bp [data-bpid="${CSS.escape(BP.focus)}"]`)) || els[0]).focus(); return; }
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
    if (!best) { sfx("edge"); return; }
    best.focus({ preventScroll: true });
    best.scrollIntoView({ block: "nearest", inline: "center", behavior: reduced() ? "auto" : "smooth" });
    BP.focus = best.dataset.bpid;
    bpArtFor(best);
    sfx("move");
  }
  function bpTab(step) {
    if (BP.detail || BP.choose) return;
    const i = BP_TABS.findIndex((t) => t[0] === BP.tab);
    BP.tab = BP_TABS[(i + step + BP_TABS.length) % BP_TABS.length][0];
    BP.tag = ""; BP.focus = ""; BP.keep = true;
    sfx("tab");
    renderBP(true);
  }
  function bpPlay(it, pid) {
    if (!it) return;
    sfx("go");
    if (it.kind === "found") { actions["play-found"]({ dataset: { key: it.id } }); return; }
    const p = pid || (seated() || {}).id;
    if (!p) { BP.choose = bpKey(it); BP.focus = ""; BP.keep = true; renderBP(true); return; }
    BP.choose = null; BP.focus = BP.detail ? "play" : BP.from; BP.keep = true;
    renderBP(false);
    actions["play-as"]({ dataset: { game: it.id, player: p } });
  }
  const bpRunning = (it) => { const s = it && it.g && session(it.g.id); return !!s && s.phase === "playing"; };
  function bpCloseGame(k) {
    const it = bpItem(k);
    if (!bpRunning(it)) return;
    sfx("go");
    actions["close-game"]({ dataset: { game: it.g.id } });
  }
  function bpX() {
    const el = document.activeElement;
    bpCloseGame((el && el.dataset && el.dataset.k) || BP.detail);
  }
  function bpSelect(el) {
    if (!el) return;
    const go = el.dataset.go;
    if (go === "open") { BP.detail = el.dataset.k; BP.from = el.dataset.bpid; BP.focus = "play"; BP.keep = true; sfx("go"); renderBP(true); }
    else if (go === "play") bpPlay(bpItem(el.dataset.k));
    else if (go === "tag") { BP.tab = "library"; BP.tag = el.dataset.tag; BP.focus = ""; BP.keep = true; sfx("go"); renderBP(true); }
    else if (go === "choose") { BP.choose = el.dataset.k; BP.focus = ""; sfx("go"); renderBP(true); }
    else if (go === "close-game") bpCloseGame(el.dataset.k);
    else if (go === "play-as") bpPlay(bpItem(BP.choose), el.dataset.p);
    else if (go === "back") bpBack();
    else if (go === "seat") { actions.seat({ dataset: { player: el.dataset.p } }); sfx("go"); renderBP(false); }
    else if (go === "tab") { BP.tab = el.dataset.t; BP.tag = ""; BP.detail = null; BP.choose = null; BP.focus = ""; sfx("tab"); renderBP(true); }
    else if (go === "exit") bpClose();
    else if (go === "sound") { BP.sounds = !BP.sounds; setPref("bpSounds", BP.sounds ? "1" : "0"); sfx("toggle"); renderBP(false); }
  }
  function bpBack() {
    if (BP.choose) { BP.choose = null; BP.focus = BP.detail ? "play" : BP.from; }
    else if (BP.detail) { BP.focus = BP.from; BP.detail = null; }
    else if (BP.tag) { BP.focus = "c:" + BP.tag; BP.tab = "home"; BP.tag = ""; }
    else { sfx("edge"); return; }
    BP.keep = true;
    sfx("back");
    renderBP(true);
  }
  function bpY() {
    const el = document.activeElement;
    const k = (el && el.dataset && el.dataset.k) || BP.detail;
    const it = k && bpItem(k);
    if (it && it.g && cfg().players.length > 1) { if (!BP.detail) BP.from = el.dataset.bpid || ""; BP.choose = k; BP.focus = ""; sfx("go"); renderBP(true); }
  }
  document.addEventListener("click", (ev) => {
    const el = BP.on && ev.target.closest("#bp [data-go]");
    if (!el) return;
    ev.preventDefault(); ev.stopPropagation();
    bpSelect(el);
  }, true);
  document.addEventListener("focusin", (ev) => { if (BP.on && ev.target.closest && ev.target.closest("#bp [data-bp]")) { BP.focus = ev.target.dataset.bpid || BP.focus; bpArtFor(ev.target); } });
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
  document.addEventListener("keydown", (ev) => {
    if (ev.key === "F11") { ev.preventDefault(); if (BP.on) bpClose(); else bpOpen(); return; }
    if (!BP.on || S.picker || S.dialog) return;
    const k = ev.key;
    const dirs = { ArrowLeft: "left", ArrowRight: "right", ArrowUp: "up", ArrowDown: "down" };
    if (dirs[k]) { ev.preventDefault(); bpMove(dirs[k]); }
    else if (k === "Enter" || k === " ") { ev.preventDefault(); bpSelect(document.activeElement.closest && document.activeElement.closest("#bp [data-go]")); }
    else if (k === "Escape" || k === "Backspace") { ev.preventDefault(); if (BP.detail || BP.choose || BP.tag) bpBack(); else bpClose(); }
    else if (k === "q" || k === "PageUp") { ev.preventDefault(); bpTab(-1); }
    else if (k === "e" || k === "PageDown") { ev.preventDefault(); bpTab(1); }
    else if (k === "y") { ev.preventDefault(); bpY(); }
    else if (k === "x") { ev.preventDefault(); bpX(); }
    else if (k === "F5") ev.preventDefault();
    ev.stopImmediatePropagation();
  }, true);
  setInterval(() => { const c = BP.on && $("#bp .bp-clock"); if (c) c.textContent = new Date().toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" }); }, 15000);

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
  const padRole = (fam, i) => (fam === "nintendo" && i < 2 ? (i ? "A" : "B") : i === 17 && fam !== "playstation" ? "" : PAD_ROLES[i] || "");
  const padDir = (g) => {
    const b = (i) => !!(g.buttons[i] && g.buttons[i].pressed), ax = g.axes[0] || 0, ay = g.axes[1] || 0;
    return b(12) || ay < -0.55 ? "up" : b(13) || ay > 0.55 ? "down" : b(14) || ax < -0.55 ? "left" : b(15) || ax > 0.55 ? "right" : "";
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
    if (document.hasFocus()) {
      if (fam !== pad.fam) { pad.fam = fam; if (BP.on) renderBP(false); }
      if (roles.has("Start")) { if (BP.on) bpClose(); else bpOpen(); }
      if (BP.on && !S.picker && !S.dialog) {
        if (roles.has("A")) bpSelect(document.activeElement && document.activeElement.closest && document.activeElement.closest("#bp [data-go]"));
        if (roles.has("B")) bpBack();
        if (roles.has("X")) bpX();
        if (roles.has("Y")) bpY();
        if (roles.has("LB")) bpTab(-1);
        if (roles.has("RB")) bpTab(1);
        if (roles.has("View")) bpSelect({ dataset: { go: "sound" } });
        if (dir && (dir !== pad.held || t >= pad.next)) { bpMove(dir); pad.next = t + (dir !== pad.held ? 380 : 120); }
      } else if (BP.on && S.dialog && !S.picker) {
        const o = $("#overlay"), a = document.activeElement;
        if (roles.has("A") && a !== o && o.contains(a)) a.click();
        if (roles.has("B") && S.dialog.kind !== "welcome") closeDialog(false);
        const step = !dir || dir === pad.held ? 0 : dir === "up" || dir === "left" ? -1 : 1;
        const bs = step ? [...o.querySelectorAll("button:not([disabled])")] : [];
        if (bs.length) bs[(Math.max(bs.indexOf(a), step < 0 ? 0 : -1) + step + bs.length) % bs.length].focus();
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

