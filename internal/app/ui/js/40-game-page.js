  // ------------------------------------------------------------ game page --
  const reduced = () => window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  // Movement (not only fades) is on: no reduced motion, and Motion is Full.
  const moving = () => !reduced() && !document.documentElement.classList.contains("motion-less") && !document.documentElement.classList.contains("motion-off");
  const ms = (token) => parseFloat(getComputedStyle(document.documentElement).getPropertyValue(token)) || 0;
  // A page change: the new page is drawn at once, under a still copy of
  // the one you leave (the same nodes, where they were scrolled), and comes
  // in as that copy fades out, so no frame is empty. Art already on this
  // PC is decoded first, for at most --dur-1, so the page arrives whole.
  // A second change while one is under way finishes the first at once.
  let swapping = null;
  function flushSwap() { if (swapping) swapping.done(); }
  function swapPage(update) {
    flushSwap();
    const v = $("#view"), old = v.firstElementChild, top = v.scrollTop;
    if (reduced() || !old) { update(); return; }
    const ghost = document.createElement("div"), r = v.getBoundingClientRect();
    ghost.className = "view page-ghost";
    ghost.inert = true;
    ghost.style.setProperty("--sec", getComputedStyle(v).getPropertyValue("--sec")); // the page you leave keeps its colour
    Object.assign(ghost.style, { left: r.left + "px", top: r.top + "px", width: r.width + "px", height: r.height + "px" });
    let shown = false;
    const show = () => {
      if (shown || swapping !== sw) return;
      shown = true;
      v.classList.remove("is-arriving");
      if (v.firstElementChild) enter(v.firstElementChild);
      ghost.classList.add("out"); // starts in the same frame as the entrance: a slow first frame delays both
      Promise.all(ghost.getAnimations().map((a) => a.finished)).then(sw.done, sw.done);
    };
    const sw = swapping = {
      // The page drawn again before its entrance started: the copy starts with it.
      sync(at) { ghost.getAnimations().forEach((a) => { if (a.startTime == null) a.startTime = at; }); },
      done() { if (!shown) { shown = true; v.classList.remove("is-arriving"); } ghost.remove(); if (swapping === sw) swapping = null; },
    };
    v.classList.add("is-arriving");
    try { update(); } catch (e) { sw.done(); throw e; }
    old.classList.remove("enter");
    old.querySelectorAll("[id]").forEach((e) => e.removeAttribute("id"));
    ghost.append(old);
    v.parentElement.append(ghost);
    ghost.scrollTop = top;
    const vr = v.getBoundingClientRect();
    const art = [...v.querySelectorAll("img[src]")].filter((i) => { const b = i.getBoundingClientRect(); return b.bottom > vr.top && b.top < vr.bottom; }).slice(0, 24);
    Promise.race([Promise.all(art.map((i) => i.decode().catch(() => {}))), new Promise((ok) => setTimeout(ok, ms("--dur-1")))]).then(show);
  }
  // A page arriving, once: its head settles, the labels of the sections
  // in view wipe in one after another (up to five), the rest fades in (the
  // order is in app.css, motion). A redraw while it arrives (a cover coming
  // in) carries on from where it was (at, when the entrance of the page it
  // replaces started); while a change waits for its art, the page waits too.
  let arrival = 0;
  const ENTRANCE = ["page-in", "head-in", "label-in", "part-in"];
  function enter(page, at) {
    if (reduced() || swapping && $("#view").classList.contains("is-arriving")) return;
    const vr = $("#view").getBoundingClientRect();
    let i = 0;
    page.querySelectorAll(".ln-section-head:not(.ln-section-head--minor)").forEach((h) => {
      const r = h.getBoundingClientRect();
      if (i < 5 && r.height && r.bottom > vr.top && r.top < vr.bottom) { h.classList.add("wipe"); h.style.setProperty("--i", i++); }
    });
    page.classList.add("enter");
    const runs = page.getAnimations({ subtree: true }).filter((a) => ENTRANCE.includes(a.animationName));
    if (at != null) { runs.forEach((a) => (a.startTime = at)); if (swapping) swapping.sync(at); }
    Promise.allSettled(runs.map((a) => a.finished)).then(() => page.classList.remove("enter"));
  }

  // Profiles for this game, as a pill that switches: on swaps files per
  // profile; off asks first (turning off moves the profiles aside).
  function profPill(on, attrs) {
    const label = solo() ? "Runs" : "Profiles";
    return `<button class="prof-pill" role="switch" aria-checked="${on}" ${attrs} ${S.adding ? "disabled" : ""} title="${on ? `${label} are on for this game` : `Turn on ${label.toLowerCase()} for this game`}">${GI.profiles}<span>${label}</span><i aria-hidden="true"></i></button>`;
  }
  function openSheet(kind, id, fromEl) {
    if (S.view === "edit" && !confirmLeaveEdit()) return;
    if (S.dialog && S.dialog.kind !== "welcome") S.dialog = null;
    const P = { kind, id, cloud: null };
    const from = fromEl && coverFrom(fromEl);
    if (from) requestAnimationFrame(() => openCover(from));
    swapPage(() => {
      if (S.view !== "game") S.from = S.view === "home" ? "home" : "library";
      enterView("game", P); // the timeline is read again: a session may have ended since
      S.animateView = true;
      render(); $("#view").scrollTop = 0;
      $("#view").focus({ preventScroll: true });
    });
    askCloud(P);
  }

  // The cover you click opens into the game: a copy of it grows from where
  // it was to fill the window, over the game's own art, and fades as the
  // page settles. Only with Motion on Full, and only for a cover on screen.
  function coverFrom(el) {
    if (!moving()) return null;
    // The biggest picture showing in what was clicked: the art, not a logo over it.
    const area = (i) => { const b = i.getBoundingClientRect(); return b.width * b.height; };
    const img = [...el.querySelectorAll("img")].filter((i) => i.complete && i.naturalWidth).sort((a, b) => area(b) - area(a))[0];
    const r = img && img.getBoundingClientRect();
    if (!r || r.width < 40 || r.bottom < 0 || r.top > innerHeight) return null;
    return { src: img.currentSrc || img.src, r, radius: getComputedStyle(img).borderRadius };
  }
  function openCover(f) {
    const c = document.createElement("img");
    c.className = "cover-fly";
    c.alt = "";
    c.src = f.src;
    Object.assign(c.style, { left: f.r.left + "px", top: f.r.top + "px", width: f.r.width + "px", height: f.r.height + "px", borderRadius: f.radius });
    document.body.append(c);
    const sx = innerWidth / f.r.width, sy = innerHeight / f.r.height, s = Math.max(sx, sy);
    const dx = innerWidth / 2 - (f.r.left + f.r.width / 2), dy = innerHeight / 2 - (f.r.top + f.r.height / 2);
    const a = c.animate([
      { transform: "translate(0, 0) scale(1)", opacity: 1, filter: "blur(0) brightness(1)" },
      { transform: `translate(${dx * 0.55}px, ${dy * 0.55}px) scale(${1 + (s - 1) * 0.35})`, opacity: 0.85, filter: "blur(0) brightness(0.9)", offset: 0.4 },
      { transform: `translate(${dx}px, ${dy}px) scale(${s})`, opacity: 0, filter: "blur(18px) brightness(0.6)" },
    ], { duration: ms("--dur-4") * 1.25, easing: "cubic-bezier(.3, .7, .2, 1)" });
    a.finished.then(() => c.remove(), () => c.remove());
  }

  function askCloud(P) {
    const g = P.kind === "game" && game(P.id);
    if (g && g.launch.type === "steam") {
      api.cloudStatus(P.id).then((st) => { P.cloud = st; if (S.page === P && S.view === "game") render(); }).catch(() => {});
    }
  }

  // The game page's back: to the page before it, named, or to the Library.
  function pageBack() { if (hist.back.length) goHist(true); else { uiSound("back"); actions.nav({ dataset: { view: "library" } }); } }
  function backLabel() {
    const to = hist.back[hist.back.length - 1] || { view: "library" };
    const it = to.page && everyItem().find((x) => x.kind === to.page.kind && x.id === to.page.id);
    const name = it ? it.name : to.player && (player(to.player) || {}).name;
    if (name) return name.length > 24 ? name.slice(0, 23) + "…" : name;
    return PAGE_NAMES[to.view] || "Back";
  }

  // When something was last played, in words.
  function ago(t) {
    const m = Math.round((Date.now() - new Date(t).getTime()) / 60000);
    if (!(m >= 0)) return "";
    if (m < 2) return "just now";
    if (m < 60) return m + " minutes ago";
    const h = Math.round(m / 60);
    if (h < 24) return h === 1 ? "an hour ago" : h + " hours ago";
    const d = Math.round(h / 24);
    if (d < 2) return "yesterday";
    if (d < 30) return d + " days ago";
    return new Date(t).toLocaleDateString(undefined, { day: "numeric", month: "long" });
  }
  const recentKey = (it) => (it.kind === "game" ? "game:" + it.id : it.id);
  function lastPlayed(it) {
    const r = (S.d.recent || {})[recentKey(it)];
    if (!r) return "";
    const when = ago(r.at);
    return r.player && player(r.player) ? `${pname(r.player)} played ${when}` : `Played ${when}`;
  }

  // Wide art and logos: from Steam's cache on this PC, or downloaded when
  // "Download covers and artwork" is on.
  function artHTML(it) {
    const h = artFor(it, "hero"), c = h.url || S.icons[coverKey(it)] || "";
    if (h.from === "hero") return `<div class="art" style="background-image:url('${h.url}')"></div>`;
    if (c) return `<div class="art soft" style="background-image:url('${c}')"></div>`;
    return `<div class="art none"></div>`;
  }

  const artWanted = new Set();
  async function loadArt(it) {
    const key = coverKey(it);
    if (!it || artWanted.has(key)) return;
    artWanted.add(key);
    try {
      const [h, l] = await Promise.all([api.art(it.appId, it.name, key, "hero"), api.art(it.appId, it.name, key, "logo")]);
      if (h) S.art["hero:" + key] = h;
      if (l) S.art["logo:" + key] = l;
      if (h || l) render();
    } catch (e) { /* no art */ }
  }



  const day = (d) => new Date(d).toLocaleDateString(undefined, { day: "numeric", month: "short", year: new Date(d).getFullYear() === new Date().getFullYear() ? undefined : "numeric" });

  // An update or announcement, read inside Lootnest. Its body is HTML that
  // Lootnest made safe in Go (core.NewsHTML): only known elements, images
  // from Steam's servers, links that open in the browser.
  function viewReader() {
    const D = S.dialog;
    const n = ((S.news[D.key] || {}).items || [])[D.i];
    if (!n) return "";
    const it = everyItem().find((x) => coverKey(x) === D.key);
    return `<div class="scrim" data-action="scrim"><article class="dialog reader" role="dialog" aria-modal="true" aria-label="${esc(n.title)}">
      <header class="reader-head"><div>
        <p class="reader-meta">${n.update ? '<span class="dot"></span>Update' : "News"}<span>${esc(new Date(n.date).toLocaleDateString(undefined, { day: "numeric", month: "long", year: "numeric" }))}</span>${it ? `<span>${esc(it.name)}</span>` : ""}</p>
        <h2>${esc(n.title)}</h2></div>
        <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></header>
      <div class="reader-body selectable">${n.body || `<p>${esc(n.summary)}</p>`}</div>
      <footer class="dialog-foot"><button class="btn quiet" data-action="open-news" data-url="${esc(n.url)}">Open on Steam</button>
        <button class="btn" data-action="dialog-cancel">Close</button></footer>
    </article></div>`;
  }

  // Updates and news from Steam's feed for the game (opt-in).
  const storeKept = (src) => src === "steam" || src === "epic" || src === "gog";
  // The news as a short list, two to a row. items: [{ it, n, i }].
  function newsSection(title, items, showGame) {
    if (!items.length) return "";
    const pic = (it, n) => n.image || artFor(it, "hero").url;
    const meta = (it, n) => `<span class="news-meta"><time class="ln-num">${esc(day(n.date))}</time>${n.update ? '<span class="ln-tag ln-tag--strong">Update</span>' : ""}${showGame ? `<span>${esc(it.name)}</span>` : ""}</span>`;
    return `<section class="news" aria-label="${esc(title)}"><div class="ln-section-head"><div><h2>${esc(title)}</h2></div></div><ul class="news-list">${items.map(({ it, n, i }) => { const p = pic(it, n); return `<li><button data-action="read-news" data-key="${esc(coverKey(it))}" data-i="${i}">
      <span class="news-thumb"${p ? ` style="background-image:url('${esc(p)}')"` : ""}></span>
      <span class="news-body">${meta(it, n)}<b>${esc(n.title)}</b></span></button></li>`; }).join("")}</ul></section>`;
  }
  function newsHTML(it) {
    const title = "Updates and news";
    const head = `<div class="ln-section-head"><div><h2>${title}</h2></div></div>`;
    if (!cfg().settings.gameNews) {
      return `<section class="news">${head}${banner("Updates and patch notes", "From Steam's news feed. Only the game's Steam id or name is sent.", bBtn("secondary", 'data-action="news-on"', "Show updates and news"))}</section>`;
    }
    const key = coverKey(it);
    const N = S.news[key];
    if (!N) {
      S.news[key] = { loading: true };
      api.gameNews(key, it.appId, it.name, storeKept(it.source) ? "" : it.exe).catch(() => {});
    }
    // Nothing yet, or nothing to show: no News tab.
    if (!N || N.loading || N.none) return "";
    if (N.error) return `<section class="news">${head}${lnInline("error", esc(N.error))}</section>`;
    const items = (N.items || []).slice(0, 6).map((n, i) => ({ it, n, i }));
    // From Updates' Patch notes: open that post once the news is here.
    if (S.newsOpen && S.newsOpen.key === key) {
      const at = Math.max(0, (N.items || []).findIndex((n) => n.title === S.newsOpen.title));
      S.newsOpen = null;
      if ((N.items || []).length) setTimeout(() => actions["read-news"]({ dataset: { key, i: String(at) } }), 260);
    }
    const uk = updKey(it);
    const offer = it.kind === "found" && (S.copies || []).find((o) => o.new.key === it.id);
    const newer = N.newer ? banner("A newer version is out", `${esc(N.newer.title)}, ${esc(day(N.newer.date))}`,
      uk ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="upd-start" data-key="${esc(uk)}" data-name="${esc(it.name)}">Apply an update</button>` : "") : "";
    const copyBanner = offer ? `<ul class="ln-rows news-newer">${copyCard(offer, true)}</ul>` : "";
    if (!items.length && !newer && !offer) return "";
    if (!items.length) return `${copyBanner}<section class="news">${head}${newer}<p class="news-none">No news yet.</p></section>`;
    return copyBanner + (newer ? `<div class="news-newer">${newer}</div>` : "") + newsSection(title, items, false);
  }

  // ------------------------------------------------------------ game page --
  // Built on system.css (DESIGN.md): the game's art under a glass play bar.
  // With profiles on, the player whose files are loaded is the one Play and
  // everyone else is a chip; the actions below are grouped by intent.
  const gsvg = (d, fill) => `<svg class="ln-icon" viewBox="0 0 20 20" aria-hidden="true"${fill ? ' style="fill:currentColor;stroke:none"' : ""}>${d}</svg>`;
  // Icons by role: Phosphor's (js/01-icons.js), named for what they do here.
  const GI = {
    refresh: ph("arrow-clockwise"),
    more: ph("dots-three"),
    back: ph("caret-left"),
    next: ph("caret-right"),
    swap: ph("arrows-left-right"),
    tick: ph("check"),
    shield: ph("shield"),
    lock: ph("lock"),
    trophy: ph("trophy"),
    play: ph("play", "fill"),
    load: ph("download-simple"),
    ask: ph("question"),
    edit: ph("pencil-simple"),
    folder: ph("folder"),
    save: ph("floppy-disk"),
    clip: ph("video-camera"),
    camera: ph("camera"),
    history: ph("clock-counter-clockwise"),
    check: ph("shield-check"),
    drive: ph("hard-drive"),
    desktop: ph("desktop"),
    trash: ph("trash"),
    power: ph("power"),
    hide: ph("eye-slash"),
    copy: ph("copy"),
    sliders: ph("sliders-horizontal"),
    person: ph("user"),
    profiles: ph("users"),
    search: ph("magnifying-glass"),
    x: ph("x"),
    plus: ph("plus"),
    flask: ph("flask"),
    globe: ph("globe-simple"),
    cloud: ph("cloud"),
    info: ph("info"),
    alert: ph("warning"),
    image: ph("image"),
    launch: ph("rocket-launch"),
    pad: ph("game-controller"),
    puzzle: ph("puzzle-piece"),
    star: ph("star"),
  };

  // A player's avatar in the system's style: their picture, a game's cover,
  // or their initial on their colour.
  const lnAvatar = (p, size) => {
    // A run wears its person's picture and colour, with a ring.
    const par = parentOf(p), who = par || p;
    const u = (S.d.avatars || {})[who.id];
    const g = !u && who.avatarGame && itemByKey(who.avatarGame);
    if (g) ensureCover(g);
    const gu = g && (artFor(g, "cover").url || S.icons[coverKey(g)]);
    const inner = u ? `<img src="${u}" alt="">` : gu ? `<img src="${gu}" alt="">` : esc((who.name || "?").trim().charAt(0).toUpperCase());
    return `<span class="ln-avatar ln-player${size ? " ln-avatar--" + size : ""}${par ? " is-run" : ""}" style="--player:${esc(who.color || p.color)}" aria-hidden="true">${inner}</span>`;
  };
  const gpHead = (t) => `<div class="ln-section-head ln-section-head--minor"><h2>${t}</h2></div>`;
  const gpRow = (attrs, ic, title, sub, opt = {}) => `<button class="ln-row${opt.danger ? " ln-row--danger" : ""}" ${attrs}${opt.disabled ? " disabled" : ""}>${ic}<span><b>${title}</b>${sub ? `<span>${sub}</span>` : ""}</span></button>`;
  const gpGroup = (title, rows) => (rows.filter(Boolean).length ? `<section class="gpage-group" data-g="${title}">${gpHead(title)}<div class="ln-list">${rows.filter(Boolean).join("")}</div></section>` : "");
  const lnInline = (kind, title, text) => `<div class="ln-state ln-state--inline${kind === "error" ? " ln-state--error" : ""}">${kind === "error" ? GI.alert : '<span class="ln-spinner" aria-hidden="true"></span>'}<div><h3>${title}</h3>${text ? `<p>${text}</p>` : ""}</div></div>`;
  const gpStat = (value, label, p) => `<div class="ln-meta__item"><dd class="ln-num">${value}</dd><dt>${p ? `<span class="ln-mark ln-player" style="--player:${esc(p.color)}"></span>` : ""}${label}</dt></div>`;
  const hoursValue = (sec) => { const t = hours(sec), m = t.match(/^([\d.]+) (h|min)$/); return m ? `${m[1]}<small>${m[2]}</small>` : esc(t); };

  function viewGame() {
    const P = S.page;
    const it = P && (P.kind === "owned" ? ownedItem(P.id) : everyItem().find((x) => x.kind === P.kind && x.id === P.id));
    if (!it) { S.view = "library"; return viewLibrary(); }
    return it.kind === "owned" ? ownedView(it) : gameView(it, true);
  }
  // The game page (page = true), or just its art and play bar, as on Home.
  function gameView(it, page) {
    const P = page ? S.page : {};
    loadArt(it); ensureCover(it);
    const key = coverKey(it);
    const h = artFor(it, "hero"), icon = S.icons[key], logo = artFor(it, "logo").url;
    const art = h.url ? artImg(h, "hero", "gpage-art")
      : icon ? `<img class="gpage-art icon" src="${icon}" alt="">` : `<div class="gpage-plate" aria-hidden="true"><span>${esc(initial(it.name))}</span></div>`;
    const real = h.url;
    const title = logo ? `<img class="gpage-logo" src="${logo}" alt="${esc(it.name)}">` : esc(it.name);
    const rec = (S.d.recent || {})[recentKey(it)];
    const meta = [esc(sourceLabel(it.source, it.f))];
    const installed = page && installOf(groupOf(it) || it); // every copy, as in the Library
    if (installed) meta.push(`<span class="ln-num">${esc(about(installed))}</span> on this PC`);
    if (isOffline(it)) meta.push(`${GI.globe}Kept offline`);
    const st = steamTime(it);
    let bar, note = "", side, news, head;
    const details = page ? "" : `<button class="ln-button ln-button--ghost" data-action="open-sheet" ${itemAttr(it)}>Details</button>`;
    // Everything else you can do with the game opens in a panel at the side.
    const manage = page ? `<button class="ln-button ln-button--secondary ln-button--sm gpage-manage-btn" data-action="manage" ${itemAttr(it)} aria-haspopup="dialog">${GI.sliders}Manage</button>` : "";

    if (it.kind === "found") {
      const f = it.f;
      meta.push("Profiles off");
      const stats = [st ? gpStat(hoursValue(st), "Steam account") : ""].join("");
      const turnOn = () => profPill(false, `data-action="found-add" data-key="${esc(f.key)}"`);
      const t = totalPlayed(it) || st;
      // A game from more than one store: which copy starts, and the others.
      const grp = page && groupOf(it);
      const startWith = grp ? `<button class="ln-button ln-button--secondary" data-action="start-with" data-key="${esc(f.key)}" aria-haspopup="menu">${GI.swap}Start with</button>` : "";
      const also = grp ? `<p class="gpage-note">${GI.copy}<span>Also in ${andList(grp.copies.filter((c) => c.id !== it.id).map((c) => `<button class="gpage-link" data-action="open-sheet" data-key="${esc(c.id)}" title="${esc(c.f.dir || "")}">${esc(sourceLabel(c.source, c.f))}</button>`))}.</span></p>` : "";
      // Home: Play, when it was played, and the way to turn profiles on.
      if (!page) bar = `<section class="gpage-bar" aria-label="Play">
        <button class="ln-button ln-button--primary ln-button--lg" data-action="play-found" data-key="${esc(f.key)}">${GI.play}Play</button>
        <div class="gpage-state"><b>${rec ? esc(lastPlayed(it)) : "Not played yet"}</b>${t ? `<span>${esc(hours(t))} played</span>` : ""}</div>
        <div class="gpage-others">${turnOn()}</div></section>`;
      else bar = `<section class="gpage-bar" aria-label="Play">
        <button class="ln-button ln-button--primary ln-button--lg" data-action="play-found" data-key="${esc(f.key)}">${GI.play}Play</button>
        <div class="gpage-state"><b>${solo() ? "One set of saves" : "Profiles off"}</b>${rec ? `<span>${esc(lastPlayed(it))}</span>` : t ? `<span>${esc(hours(t))} played</span>` : ""}</div>
        <div class="gpage-others">${startWith}${turnOn()}</div>
        ${stats ? `<dl class="ln-meta gpage-stats">${stats}</dl>` : ""}</section>`;
      head = { play: `<button class="ln-button ln-button--primary" data-action="play-found" data-key="${esc(f.key)}">${GI.play}Play</button>`, state: solo() ? "One set of saves" : "Profiles off" };
      note = page ? also + dlNote(it) + storeNote(ITCH, it) + storeNote(GOG, it) + antiCheatNote(it) : "";
      side = page ? playersPanel(it) + willRunBlock(it) + collectionsBlock(it) : "";
      news = page ? newsHTML(it) : "";
    } else {
      const g = it.g, s = session(g.id), owner = S.d.active[g.id];
      const ps = cfg().players;
      const withProfile = ps.filter((p) => hasProfile(g.id, p.id));
      meta.push(!withProfile.length ? "Profiles on" : withProfile.length === 1 ? "1 profile" : withProfile.length + " profiles");
      if (rec && !page) meta.push(esc(lastPlayed(it)));
      // The one Play: whoever's files are loaded, else whoever played last.
      const lead = seated() || player(owner) || (rec && player(rec.player)) || withProfile[0] || ps[0];
      const pt = playtimeOf(it);
      let play, state;
      const playAs = (size) => `<button class="ln-button ln-button--play${size} ln-player" style="--player:${esc(lead.color)}" data-action="play-as" data-game="${esc(g.id)}" data-player="${esc(lead.id)}">${GI.play}Play as ${esc(lead.name)}</button>`;
      if (s) {
        play = `<span class="gpage-session ln-player" style="--player:${esc(color(s.playerId))}" role="status">${s.phase === "playing" ? '<span class="ln-mark ln-player" style="--player:' + esc(color(s.playerId)) + '"></span>' : '<span class="ln-spinner" aria-hidden="true"></span>'}${esc(sessionText(s))}</span>`;
        state = `<div class="gpage-state"><b>${esc(possessive(pname(s.playerId)))} files are loaded</b></div>`
          + (s.phase === "playing" ? `<div class="gpage-others"><button class="ln-button ln-button--secondary" data-action="close-game" data-game="${esc(g.id)}">${GI.x}Close the game</button></div>` : "");
      } else {
        play = lead ? playAs(" ln-button--lg") : "";
        const sub = owner && lastBy(it, owner) ? `Played ${esc(ago(lastBy(it, owner)))}` : owner && pt[owner] ? `${esc(hours(pt[owner]))} played` : owner ? "Loaded without playing" : "Not played yet";
        state = `<div class="gpage-state"><b>${owner && player(owner) ? `${esc(possessive(pname(owner)))} files are loaded` : "Nobody's files yet"}</b><span>${sub}</span></div>`;
      }
      const other = !s && ps.length > 1 ? `<button class="ln-button ln-button--secondary" data-action="play-pick" data-game="${esc(g.id)}" aria-haspopup="dialog">${solo() ? "Choose a run" : "Someone else"}</button>` : "";
      // A run of whoever's playing: its own saves, from the start or a copy.
      const newRun = !s && lead && ps.length < MAX_PROFILES ? `<button class="ln-button ln-button--secondary" data-action="new-run" data-game="${esc(g.id)}" data-person="${esc(personOf(lead).id)}">${GI.plus}New run</button>` : "";
      const pill = page && !s ? profPill(true, `data-action="remove-game" data-game="${esc(g.id)}"`) : "";
      bar = `<section class="gpage-bar" aria-label="Play">${play}${state}${other || newRun || pill ? `<div class="gpage-others">${other}${newRun}${pill}</div>` : ""}${details}</section>`;
      const loaded = s ? s.playerId : player(owner) && owner;
      head = { play: other + (s ? play.replace(' role="status"', "") : lead ? playAs("") : ""), state: loaded ? `${possessive(pname(loaded))} files are loaded` : "Nobody's files yet" };
      note = page ? conflictBanner(g) + dlNote(it) + storeNote(ITCH, it) + storeNote(GOG, it) + antiCheatNote(it) + oneDriveNote(it, "gpage-note") : "";
      const cloud = P.cloud && {
        none: "Doesn't sync the files profiles swap.",
        elsewhere: "Syncs other files of this game, not the ones profiles swap.",
        overlap: `Syncs ${P.cloud.count === 1 ? "1 of the files" : P.cloud.count + " of the files"} profiles swap. Checked at every start.`,
      }[P.cloud.state];
      side = page ? playersPanel(it) + willRunBlock(it) + collectionsBlock(it) + (cloud ? `<section class="gpage-block">${gpHead("Steam Cloud")}<p>${esc(cloud)}</p></section>` : "") : "";
      news = page ? newsHTML(it) : "";
    }
    const back = page ? `<button class="ln-button ln-button--ghost ln-button--sm gpage-back" data-action="back-library">${GI.back}${esc(backLabel())}</button>` : "";
    const top = `<section class="gpage-hero">${art}${back}${manage}
        <div class="gpage-title"><h1 class="${it.name.length > 22 && !logo ? "long" : ""}">${page ? title : `<button class="gpage-title-link" data-action="open-sheet" ${itemAttr(it)}>${title}</button>`}</h1>
          <p class="gpage-meta">${meta.map((m) => `<span>${m}</span>`).join("")}</p></div>
      </section>`;
    if (!page) return { top, bar, note, real };
    const T = gameTabs(it, side, news), tabs = T.list.length > 1;
    const small = artFor(it, "cover").url || icon;
    return `<div class="gpage ln-root${real ? "" : " no-art"}${docked.bar ? " no-bar" : ""}${docked.tabs ? " no-tabs" : ""}">
      <div class="gp-dock"><div class="gp-head">${small ? `<img class="gp-head-art" src="${small}" alt="">` : `<span class="gp-head-art" aria-hidden="true">${esc(initial(it.name))}</span>`}
        <div class="gp-head-id"><b>${esc(it.name)}</b><span>${esc(head.state)}</span></div>${tabs ? tabRow(it, T, true) : ""}<div class="gp-head-play">${head.play}</div></div></div>
      ${top}
      <div class="gpage-body">${bar}${note}${tabs ? tabRow(it, T, false) : ""}
        <div class="gp-panel" id="gp-panel"${tabs ? ` role="tabpanel" aria-labelledby="gp-tab-${T.cur}"` : ""}>${T.html}</div>
      </div></div>`;
  }

  // The page in tabs, each shown only when it has something in it. The tab
  // is kept per game (S.gameTab); switching isn't a stop for Back, but the
  // page's place in the history carries the tab it was left on.
  const TABS = [["overview", "Overview"], ["achievements", "Achievements"], ["saves", "Saves"], ["mods", "Mods"], ["timeline", "Timeline"], ["news", "News"]];
  const pageKey = (P) => P.kind + ":" + P.id;
  const wide = (html) => `<div class="gpage-main gpage-main--wide"><div class="gpage-col">${html}</div></div>`;
  const cols = (main, side) => (side ? `<div class="gpage-main"><div class="gpage-col">${main}</div><aside class="gpage-side">${side}</aside></div>` : wide(main));
  function gameTabs(it, side, news) {
    const ach = achievementsBlock(it), tl = timelineBlock(it), points = savePointsBlock(it), saves = gameSavesBlock(it), other = otherSavesBlock(it), mods = modsBlock(it);
    const newsOn = cfg().settings.gameNews;
    const panes = {
      overview: cols(activityLine(it, !!tl) + detailsBlock(it) + achievementsBlock(it, true) + latestPoint(it) + latestCopy(it) + (newsOn ? "" : news) + communityBlock(it), side),
      achievements: ach && wide(ach),
      saves: points ? cols(points, saves + other) : (saves || other) && wide(saves + other),
      mods: mods && wide(mods),
      timeline: tl && wide(tl),
      news: newsOn && news && wide(news),
    };
    const want = S.gameTab[pageKey(S.page)];
    const cur = panes[want] ? want : "overview";
    return { list: TABS.filter(([k]) => panes[k]), cur, html: panes[cur] };
  }
  const tabRow = (it, T, inHead) => `<div class="ln-tabs ${inHead ? "gp-head-tabs" : "gp-tabs"}" role="tablist" aria-label="${esc(it.name)}">${T.list.map(([k, l]) => `<button role="tab"${inHead ? "" : ` id="gp-tab-${k}"`} aria-selected="${k === T.cur}" aria-controls="gp-panel" tabindex="${k === T.cur ? 0 : -1}" data-action="game-tab" data-tab="${k}">${l}</button>`).join("")}</div>`;
  // Changing tabs keeps the scroll, unless the new tab would start above
  // the view: then it starts just under the compact head.
  function setGameTab(tab) {
    if (!S.page) return;
    const was = S.gameTab[pageKey(S.page)] || "overview";
    if (was !== tab) uiSound("tab");
    S.gameTab[pageKey(S.page)] = tab;
    render();
    // The panel comes in from the side of the tab you picked.
    const order = TABS.map((t) => t[0]), step = Math.sign(order.indexOf(tab) - order.indexOf(was)), pane = $("#gp-panel");
    if (step && pane && moving()) pane.animate([{ opacity: 0, transform: `translateX(${step * 18}px)` }, { opacity: 1, transform: "none" }], { duration: ms("--dur-3"), easing: "cubic-bezier(.16, 1, .3, 1)" });
    const v = $("#view"), panel = $("#gp-panel"), head = $(".gp-head"), row = $(".gp-tabs");
    if (!panel || !head || !row) return;
    const top = v.getBoundingClientRect().top + head.offsetHeight, over = panel.getBoundingClientRect().top - top;
    if (over < 0) v.scrollTop += over;
    dock("bar", $(".gpage-bar").getBoundingClientRect().bottom < top);
    dock("tabs", row.getBoundingClientRect().top < top);
    const t = $(`${docked.tabs ? ".gp-head-tabs" : ".gp-tabs"} [aria-selected="true"]`);
    if (t) t.focus({ preventScroll: true });
  }
  // The arrows, Home and End in a tab row, and Ctrl+Tab, move between tabs.
  function stepTab(key) {
    const tabs = [...document.querySelectorAll(".gp-tabs [role=tab]")], n = tabs.length;
    const i = tabs.findIndex((b) => b.getAttribute("aria-selected") === "true");
    const j = { ArrowRight: i + 1, ArrowLeft: i - 1 + n, Home: 0, End: n - 1 }[key] % n;
    if (n > 1 && j !== i) setGameTab(tabs[j].dataset.tab);
  }
  // The compact head: shown once the play bar has scrolled up under it,
  // and its tabs once the page's own have.
  const docked = { bar: false, tabs: false };
  let headIO = null;
  function dock(k, on) { docked[k] = on; const g = $("#view > .gpage:not(.home)"); if (g) g.classList.toggle("no-" + k, on); }
  function stickHead() {
    const v = $("#view");
    if (!headIO) {
      const h = getComputedStyle(document.documentElement).getPropertyValue("--gp-head-h").trim();
      // The bar docks once it's all under the head, the tabs once any of them is.
      headIO = new IntersectionObserver((es) => es.forEach((e) => {
        const tabs = e.target.classList.contains("gp-tabs");
        dock(tabs ? "tabs" : "bar", (tabs ? e.intersectionRatio < 1 : !e.isIntersecting) && e.boundingClientRect.top < e.rootBounds.top);
      }), { root: v, rootMargin: `-${h} 0px 0px 0px`, threshold: [0, 1] });
    }
    headIO.disconnect();
    if (S.view === "game") v.querySelectorAll(".gpage-bar, .gp-tabs").forEach((el) => headIO.observe(el));
  }
  // The Manage panel: what you can do with a game besides playing it,
  // grouped by intent. Each row closes the panel and does its one thing.
  // Keep it offline: Windows Firewall rules for games Lootnest starts itself.
  const isOffline = (it) => !!(cfg().offline || {})[recentKey(it)];
  const offlineWorth = (it) => (it.kind === "found" ? !it.appId && !(it.f && it.f.uri) && !storeKept(it.source) : it.g.launch.type === "exe");
  function offlineRow(it) {
    if (!offlineWorth(it)) return "";
    const n = ((cfg().offline || {})[recentKey(it)] || []).length;
    return isOffline(it) ? gpRow(`data-action="offline-toggle" ${itemAttr(it)} data-on="0"`, GI.globe, "Let it go online again", `Windows Firewall blocks ${n === 1 ? "its program" : n + " of its programs"}`)
      : gpRow(`data-action="offline-toggle" ${itemAttr(it)} data-on="1"`, GI.globe, "Keep it offline", "");
  }
  const detailsRow = (it) => (cfg().settings.gameDetails ? gpRow(`data-action="details-edit" ${itemAttr(it)}`, GI.edit, "Edit details", "", { disabled: !!(S.details[recentKey(it)] || {}).loading }) : "");
  // Cover and art: a picture of the user's own for each kind, copied into
  // Lootnest; their file stays where it is.
  const ART_KINDS = [["cover", "Change cover", "About 2 by 3", "cover"], ["hero", "Change wide art", "", "wide art"],
    ["logo", "Change logo", "PNG or WebP, see-through", "logo"]];
  const artWhat = (kind) => ART_KINDS.find((k) => k[0] === kind)[3];
  function artRows(it) {
    const mine = (S.d.customArt || {})[recentKey(it)] || {};
    return ART_KINDS.flatMap(([k, title, sub, what]) => [gpRow(`data-action="art-set" ${itemAttr(it)} data-kind="${k}"`, GI.image, title, sub),
      mine[k] ? gpRow(`data-action="art-clear" ${itemAttr(it)} data-kind="${k}"`, GI.history, `Remove your ${what}`, "") : ""]);
  }
  function manageGroups(it) {
    if (it.kind === "found") {
      const f = it.f, grp = groupOf(it);
      return gpGroup("Game", [gpRow(`data-action="game-settings" data-key="${esc(f.key)}"`, GI.sliders, "Name in the Library", ""),
        launchRow(it),
        detailsRow(it),
        grp ? gpRow(`data-action="copies-apart" data-group="${esc(grp.key)}"`, GI.copy, "Show copies separately", "") : ""])
        + gpGroup("Cover and art", artRows(it))
        + gpGroup("Game files", [
          gpRow(`data-action="open-game-folder" data-key="${esc(f.key)}"`, GI.folder, "Open the game's folder", ""),
          ...storeActs(it).filter((a) => a === "verify" || a === "open").map((a) => storeRow(it, a)),
          f.source === "steam" && f.appId ? gpRow(`data-action="move-open" data-app="${f.appId}" data-name="${esc(f.name)}"`, GI.drive, "Move to another drive", "") : "",
          movable(it) ? gpRow(`data-action="movefolder-open" data-key="${esc(f.key)}"`, GI.drive, "Move to another drive", "") : "",
        ])
        + gpGroup("Online", [offlineRow(it)])
        + gpGroup("Shortcuts", [gpRow(`data-action="found-shortcut" data-key="${esc(f.key)}"`, GI.desktop, "Create desktop shortcut", "")])
        + gpGroup("Danger", [storeActs(it).includes("uninstall") ? storeRow(it, "uninstall", { danger: true }) : "",
          removable(it) ? gpRow(`data-action="remove-pc" data-key="${esc(f.key)}"`, GI.trash, "Remove from this PC", "Saves are copied first", { danger: true }) : "",
          gpRow(`data-action="found-hide" data-key="${esc(f.key)}"`, GI.hide, f.added ? "Remove from the Library" : "Hide from the Library", removable(it) ? "The game's folder stays where it is" : "", { danger: true })]);
    }
    const g = it.g, s = session(g.id);
    return gpGroup("Profiles", [
      gpRow(`data-action="load-only" data-game="${esc(g.id)}"`, GI.load, "Load someone's files", "", { disabled: s }),
      g.launch.type === "steam" && !g.steamDirect ? gpRow(`data-action="steam-setup" data-game="${esc(g.id)}"`, GI.ask, "Ask who's playing in Steam", "") : "",
      gpRow(`data-action="edit-game" data-game="${esc(g.id)}"`, GI.edit, "Edit game", "", { disabled: s }),
      gpRow(`data-action="try-save" data-game="${esc(g.id)}"`, GI.flask, "Try a save", "", { disabled: s || cfg().players.length >= MAX_PROFILES }),
    ]) + gpGroup("Game", [launchRow(it), detailsRow(it)]) + gpGroup("Cover and art", artRows(it)) + gpGroup("Game files", [
      gpRow(`data-action="open-game-folder" data-game="${esc(g.id)}"`, GI.folder, "Open the game's folder", ""),
      gpRow(`data-action="open-path" data-path="${esc((g.paths[0] || {}).path || "")}"`, GI.save, "Open the save folder", ""),
      gpRow(`data-action="copies" data-game="${esc(g.id)}"`, GI.history, "Copies before swaps", ""),
      g.launch.type === "steam" ? gpRow(`data-action="steam-verify" data-app="${g.launch.appId}"`, GI.check, "Verify files with Steam", "", { disabled: s }) : "",
      g.launch.type === "steam" ? gpRow(`data-action="move-open" data-app="${g.launch.appId}" data-name="${esc(g.name)}"`, GI.drive, "Move to another drive", "", { disabled: s }) : "",
      movable(it) ? gpRow(`data-action="movefolder-open" data-game="${esc(g.id)}"`, GI.drive, "Move to another drive", "", { disabled: s }) : "",
    ]) + gpGroup("Online", [offlineRow(it)])
      + gpGroup("Shortcuts", [gpRow(`data-action="shortcut" data-game="${esc(g.id)}"`, GI.desktop, "Create desktop shortcut", "")])
      + gpGroup("Danger", [gpRow(`data-action="remove-game" data-game="${esc(g.id)}"`, GI.power, "Turn off profiles", "", { danger: true })]);
  }
  // A panel from the right. menu: each action in it closes it first.
  const drawer = (o) => `<div class="scrim ln-drawer-scrim" data-action="scrim"><aside class="ln-drawer ln-root${o.wide ? " ln-drawer--wide" : ""}" tabindex="-1" role="dialog" aria-modal="true" aria-label="${o.label}"${o.menu ? " data-menu" : ""}>
      <header class="ln-drawer__head"><div><h2>${o.title}</h2>${o.sub ? `<p>${o.sub}</p>` : ""}</div><button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="dialog-cancel" aria-label="Close">${GI.x}</button></header>
      <div class="ln-drawer__body">${o.body}</div>${o.foot ? `<footer class="ln-drawer__foot">${o.foot}</footer>` : ""}</aside></div>`;
  const dialogItem = (D) => everyItem().find((x) => x.kind === D.ik && x.id === D.iid) || null;
  // Try a save: someone else's save in a new run, never over anyone's files.
  function viewTrySave(D) {
    const g = game(D.gameId);
    if (!g) return "";
    const froms = cfg().players.filter((p) => hasProfile(g.id, p.id));
    const from = player(D.from) || froms[0];
    const per = from ? personOf(from) : null;
    const locs = (g.paths || []).filter((p) => !String(p.id || "").startsWith("ach-"));
    const loc = locs.find((l) => l.id === D.loc) || locs[0];
    if (!per || !loc) return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Try a save">
      <div><h2>Try a save in ${esc(g.name)}</h2><p class="sub">Play it once with profiles first.</p></div>
      <div class="dialog-foot"><button class="btn primary" data-action="dialog-cancel">OK</button></div></div></div>`;
    const sel = (id, list, cur, label) => list.length > 1 ? `<label class="try-field"><span>${label}</span><select class="ln-input" id="${id}">${list.map((x) => `<option value="${esc(x.id)}"${x.id === cur ? " selected" : ""}>${esc(x.label)}</option>`).join("")}</select></label>` : "";
    const chosen = D.src ? `<p class="try-src"><span class="selectable" title="${esc(D.src)}">${esc(D.src.split(/[\\/]/).pop())}</span><span class="gpage-hint">${esc(D.src)}</span></p>` : "";
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Try a save">
      <div><h2>Try a save in ${esc(g.name)}</h2><p class="sub">Goes into a new run. ${esc(possessive(from.name))} own saves stay as they are.</p></div>
      ${sel("try-from", froms.map((p) => ({ id: p.id, label: p.name })), from.id, "Start from")}
      ${sel("try-loc", locs.map((l) => ({ id: l.id, label: l.path })), loc.id, "Where the save goes")}
      <div class="try-pick"><button class="ln-button ln-button--secondary ln-button--sm" data-action="try-browse" data-kind="trySaveDir">Choose a folder</button>
        <button class="ln-button ln-button--ghost ln-button--sm" data-action="try-browse" data-kind="trySaveFile">Choose a file or archive</button></div>
      ${chosen}
      <label class="try-field"><span>The run's name</span><input type="text" class="ln-input" id="try-name" maxlength="24" placeholder="${esc(per.name)} trying" value="${esc(D.name || "")}"></label>
      <div class="dialog-foot"><span class="error-text grow">${esc(D.error || "")}</span><button class="btn quiet" data-action="dialog-cancel">Cancel</button>
      <button class="btn primary" data-action="try-go" ${D.busy || !D.src ? "disabled" : ""}>${D.busy ? "Starting the run" : "Start the run"}</button></div></div></div>`;
  }
  // The dialog's choices, kept across a re-render.
  function tryKeep(D) {
    const f = $("#try-from"), l = $("#try-loc"), n = $("#try-name");
    if (f) D.from = f.value;
    if (l) D.loc = l.value;
    if (n) D.name = n.value;
  }
  function viewManage() {
    const it = dialogItem(S.dialog);
    if (!it) return "";
    return drawer({ label: `Manage ${esc(it.name)}`, title: esc(it.name), sub: "", body: `<div class="gpage-manage-list">${manageGroups(it)}</div>`, menu: true });
  }
  // Achievement Doctor: games whose achievements a tool keeps on this PC.
  const docWorth = (it) => !(it.kind === "found" && (storeKept(it.source) || it.appId > 0));
  function docDir(it) { return it.f ? it.f.dir : it.exe; }
  function viewAchDoctor() {
    const it = dialogItem(S.dialog);
    if (!it) return "";
    const key = recentKey(it), R = S.doctor[key];
    const shell = (body, foot) => drawer({ label: `Check ${esc(possessive(it.name))} achievements`, title: "Achievement check", sub: esc(it.name), body, foot });
    if (!R) return shell(lnInline("", "Checking"));
    if (R.error) return shell(lnInline("error", esc(R.error)));
    const tone = { ok: "ok", warn: "mid", fail: "low", info: "" };
    const steps = (R.steps || []).map((x) => `<li class="doc-step run-card${tone[x.state] ? " run-card--" + tone[x.state] : ""}"><div class="run-head"><span class="run-dot"></span><div><b>${esc(x.title)}</b><p>${esc(x.detail)}</p>
        ${x.path ? `<p class="doc-path selectable">${esc(x.path)}</p>` : ""}</div></div></li>`).join("");
    const bad = (R.steps || []).find((x) => x.state === "fail") || (R.steps || []).find((x) => x.state === "warn");
    const N = R.nameInGame;
    const seat = (S.seat && player(S.seat)) || null;
    const nameBlock = N ? `<div class="gpage-sub">${gpHead("The name in the game")}<p class="gpage-hint">In the game: ${N.name ? `<b class="selectable">${esc(N.name)}</b>` : "the tool's default name"}.${seat ? " Its settings are copied before a change." : " Sit down at the top right to use yours."}</p>
        ${seat && seat.name !== N.name ? `<div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="doc-name" ${itemAttr(it)} data-player="${esc(seat.id)}">Use ${esc(possessive(seat.name))} name</button></div>` : ""}</div>` : "";
    return shell(`<p class="gpage-hint">${bad ? `Most likely: ${esc(bad.title.charAt(0).toLowerCase() + bad.title.slice(1))}.` : "Everything Lootnest checks is in place."}</p><ol class="doc-steps">${steps}</ol>${nameBlock}`,
      `<button class="ln-button ln-button--ghost ln-button--sm" data-action="show-log-nav">Show recent activity</button><span class="grow"></span><button class="ln-button ln-button--secondary ln-button--sm" data-action="ach-doctor" ${itemAttr(it)}>Check again</button>`);
  }
  // Sources: what the last scan found in each store, launcher
  // and folder, and why it found nothing there.
  function viewSources() {
    const R = S.sources;
    const read = R && R.at && !R.at.startsWith("0001") ? ` Last looked ${ago(R.at)}.` : "";
    const scanBtn = (kind) => `<button class="ln-button ln-button--${kind} ln-button--sm" data-action="sources-rescan" ${S.found.refreshing ? 'aria-busy="true"' : ""}>${S.found.refreshing ? '<span class="ln-spinner" aria-hidden="true"></span>Scanning' : "Scan again"}</button>`;
    const shell = (body) => drawer({ label: "Sources", title: "Sources", sub: read.trim(), body,
      foot: `<button class="ln-button ln-button--ghost ln-button--sm" data-action="show-log-nav">Show recent activity</button><span class="grow"></span>${scanBtn("secondary")}` });
    if (!R) return shell(lnInline("", "Reading"));
    if (!R.sources.length) return shell(`<p class="gpage-hint">${S.found.refreshing ? "Looking for games now" : "Nothing read yet"}</p>`);
    const tone = { ok: "ok", warn: "mid", info: "" };
    const name = (src) => ({ folder: "Game folders", shortcut: "Shortcuts" })[src] || sourceLabel(src);
    const away = R.sources.filter((x) => !x.installed).map((x) => name(x.source));
    return shell(`<ol class="doc-steps">${R.sources.filter((x) => x.installed).map((x) => `<li class="doc-step run-card${tone[x.state] ? " run-card--" + tone[x.state] : ""}"><div class="run-head"><span class="run-dot"></span>
      <div class="grow"><b>${esc(name(x.source))}</b>${x.reason ? `<p>${esc(x.reason)}</p>` : ""}${x.path ? `<p class="doc-path selectable">${esc(x.path)}</p>` : ""}</div>
      ${x.found ? `<span class="src-n ln-num" aria-label="${x.found} ${x.found === 1 ? "game" : "games"}">${x.found}</span>` : ""}</div>
      ${x.state === "warn" ? `<div>${scanBtn("secondary")}</div>` : ""}</li>`).join("")}</ol>${away.length ? `<p class="gpage-hint">Not on this PC: ${esc(away.join(", "))}.</p>` : ""}`);
  }
  async function loadSources() {
    try { S.sources = await api.sources(); } catch (e) { S.sources = { sources: [] }; }
    if (S.dialog && S.dialog.kind === "sources") renderOverlay();
  }
  function hookBanner() {
    if (!S.hooks || S.hideHookBanner) return "";
    const missing = S.hooks.games.filter((h) => !h.detected && !h.direct && game(h.gameId));
    const leaving = S.hooks.games.filter((h) => h.detected && h.direct && game(h.gameId));
    if (!missing.length && leaving.length) {
      const what = leaving.length === 1 ? esc(leaving[0].name) : leaving.length + " games";
      const text = S.hooks.steamRunning
        ? `So that Steam starts ${what} by itself, Steam has to restart once (about 10 seconds) while Lootnest takes itself out of the launch options.`
        : `Lootnest takes itself out of the launch options for ${what}. Steam is closed, so this takes a second.`;
      return banner(`Let Steam start ${what}`, text, bBtn("ghost", 'data-action="hide-banner"', "Later")
        + bBtn("secondary", `data-action="hooks-apply" ${S.hooksBusy ? 'aria-busy="true"' : ""}`, (S.hooksBusy ? '<span class="ln-spinner" aria-hidden="true"></span>' : "") + (S.hooksBusy ? "Updating" : S.hooks.steamRunning ? "Restart Steam and update" : "Update now")));
    }
    if (!missing.length) return "";
    const what = missing.length === 1 ? esc(missing[0].name) : missing.length + " games";
    const text = S.hooks.steamRunning
      ? `So that Steam's Play button asks who's playing ${what}, Steam has to restart once (about 10 seconds). Lootnest does it for you.`
      : `Lootnest can set Steam up so its Play button asks who's playing ${what}.`;
    return banner("Make Steam ask who's playing", text, bBtn("ghost", 'data-action="hide-banner"', "Later")
      + bBtn("secondary", `data-action="hooks-apply" ${S.hooksBusy ? 'aria-busy="true"' : ""}`, (S.hooksBusy ? '<span class="ln-spinner" aria-hidden="true"></span>' : "") + (S.hooksBusy ? "Setting up" : S.hooks.steamRunning ? "Restart Steam and set up" : "Set up")));
  }

  async function loadFound() {
    try { S.found = await api.found(); } catch (e) { /* not ready */ }
    if (S.rescanning && !S.found.refreshing) {
      const n = (S.found.games || []).filter((f) => !S.rescanning.has(f.key)).length;
      S.rescanning = null;
      toast(n ? `Found ${n} new game${n === 1 ? "" : "s"}` : "No new games found");
    }
    if (!S.picker && !S.dialog && ["home", "library", "game"].includes(S.view)) render();
    else { $("#lib-count").textContent = libraryItems().length || ""; renderSideGames(); const w = $("#wl-scan"); if (w) welcomeScan(w); }
  }

