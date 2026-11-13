  // ---------------------------------------------------------------- render --
  // Back and Forward: the pages visited, kept in memory (a page, its game or
  // profile, a game page's tab, and how far down it was scrolled). A render that lands on
  // another page records the one it left; a re-render records nothing.
  const hist = { back: [], fwd: [], at: null, moving: false };
  function here() {
    const page = S.view === "game" && S.page ? { kind: S.page.kind, id: S.page.id } : null;
    const player = S.view === "player" ? S.playerPage : "";
    const tab = page ? S.gameTab[pageKey(page)] || "" : "";
    return { view: S.view, page, player, tab, from: S.from, key: [S.view, page ? page.kind + ":" + page.id : "", player].join("|") };
  }
  function noteVisit(scroll) {
    const at = hist.at;
    if (!at || hist.moving || at.view === "edit" || at.key === here().key) return;
    hist.back.push({ ...at, scroll });
    if (hist.back.length > 50) hist.back.shift();
    hist.fwd = [];
  }
  // Goes back (or forward) a page, to where it was scrolled. The editor
  // isn't a stop: its unsaved state is gone once it's left.
  function goHist(back) {
    flushSwap(); // a page still on its way is visited first
    const from = back ? hist.back : hist.fwd, to = back ? hist.fwd : hist.back;
    if (!from.length || BP.on || S.picker || S.dialog || (S.view === "edit" && !confirmLeaveEdit())) return;
    uiSound(back ? "back" : "tab");
    const loc = from.pop();
    // A page not drawn yet (a quick second Back) keeps the scroll it had.
    if (hist.at.view !== "edit") to.push({ ...hist.at, scroll: hist.at.scroll ?? $("#view").scrollTop });
    hist.at = loc;
    const P = loc.page && { ...loc.page, cloud: null };
    swapPage(() => {
      enterView(loc.view, P); S.from = loc.from; S.animateView = true;
      if (P) S.gameTab[pageKey(P)] = loc.tab;
      if (loc.player) S.playerPage = loc.player;
      hist.moving = true; render(); hist.moving = false;
      $("#view").scrollTop = loc.scroll || 0;
      $("#view").focus({ preventScroll: true });
    });
    if (P) askCloud(P);
  }
  // What arriving on a page resets, and the game page's game.
  function enterView(view, P) {
    S.view = view; S.menuFor = null; S.edit = null; S.svCloud = false;
    if (P) {
      S.page = P; S.selected = { kind: P.kind, id: P.id }; tlAsked.clear(); docked.bar = docked.tabs = false;
      for (const k in S.details) if (S.details[k].error) delete S.details[k]; // asked again on this visit
    }
    if (view === "saves") { loadBackupInfo(); loadSaves(); askSaves(); }
    if (view === "settings") { ctlAsked = false; S.pluginError = ""; }
    if (view !== "players") S.players = null;
    if (view !== "library") S.libSel.clear();
  }

  function render() {
    if (!S.d) return;
    document.documentElement.style.setProperty("--p1", (cfg().players[0] || {}).color || "#6C9BD2");
    document.documentElement.style.setProperty("--p2", (cfg().players[1] || {}).color || "#D9738E");
    const n = libraryItems().length;
    $("#lib-count").textContent = n ? n : "";
    renderSeat();
    renderSideUpdate();
    renderSideGames();
    const v = $("#view");
    noteVisit(v.scrollTop);
    document.documentElement.dataset.sec = secOf();
    applyAppearance();
    renderBar();
    const keepScroll = v.scrollTop;
    // A page still arriving carries on from where its entrance was.
    const was = v.firstElementChild, run = was && was.classList.contains("enter") && was.getAnimations().find((a) => a.animationName === "page-in");
    // Read before the page goes (it stops then); one not started yet starts now.
    const runAt = run ? run.startTime ?? document.timeline.currentTime : undefined;
    // A field being typed in keeps its text, focus and caret.
    const ae = document.activeElement, typing = ae && ["lib-q", "find-q", "gp-note", "gp-hours", "pt-rename"].includes(ae.id) && { id: ae.id, value: ae.value, at: ae.selectionStart };
    if (S.view === "home") v.innerHTML = viewHome();
    else if (S.view === "library") v.innerHTML = viewLibrary();
    else if (S.view === "game") v.innerHTML = viewGame();
    else if (S.view === "edit") v.innerHTML = viewEdit();
    else if (S.view === "players") v.innerHTML = viewPlayers();
    else if (S.view === "player") v.innerHTML = viewPlayer();
    else if (S.view === "settings") v.innerHTML = viewSettings();
    else if (S.view === "saves") v.innerHTML = viewSaves();
    else if (S.view === "achievements") v.innerHTML = viewAchievements();
    else if (S.view === "find") v.innerHTML = viewFind();
    else if (S.view === "updates") v.innerHTML = viewUpdates();
    const first = v.firstElementChild;
    if (S.animateView && first) {
      // "kids": the page is already here, its numbers have just arrived.
      if (S.animateView !== "kids") { arrival++; enter(first); }
      arriveOnce(v);
    } else if (first && run) enter(first, runAt);
    S.animateView = false;
    hist.at = here();
    v.scrollTop = keepScroll;
    if (S.view === "saves" && S.svCloud) svToCloud();
    if (S.view === "game") fitDetails();
    stickHead();
    if (typing) { const q = $("#" + typing.id); if (q) { q.focus(); q.value = typing.value; if (typing.at != null) q.setSelectionRange(typing.at, typing.at); } }
    renderOverlay();
    renderRoom();
    loadCovers();
    if (BP.on) bpSoon();
  }

  // ------------------------------------------------------------- top bar --
  // The sections as tabs. The current one is a cartridge that slides from
  // tab to tab; Profiles and Settings light theirs. Narrow windows keep the
  // first three tabs and put the rest under More.
  const PAGE_NAMES = { home: "Home", library: "Library", find: "Store", updates: "Updates", achievements: "Achievements", saves: "Saves", players: "Profiles", settings: "Settings" };
  // Each section's colour (data-sec on the window, --sec in CSS). A game's
  // page is the Library's; a Settings category has one only when it is a
  // section's own.
  const SEC = { home: "home", library: "library", game: "library", edit: "library", find: "store", updates: "updates", achievements: "ach", saves: "saves", players: "profiles", player: "profiles" };
  const SET_SEC = { games: "library", online: "store", updates: "updates", cloud: "saves" };
  const secOf = () => (S.view === "settings" ? SET_SEC[S.setTab] : SEC[S.view]) || "neutral";
  function renderBar(to) {
    const top = to || (S.view === "game" ? S.from : S.view === "edit" ? "library" : S.view === "player" ? "players" : S.view);
    document.querySelectorAll("#side [data-view]").forEach((t) => (t.dataset.view === top ? t.setAttribute("aria-current", "page") : t.removeAttribute("aria-current")));
    $("#hist-back").disabled = !hist.back.length;
    $("#hist-fwd").disabled = !hist.fwd.length;
    placeInd(false);
  }
  function placeInd(still) {
    const ind = $("#bar-ind"), el = $('.bar-tab[aria-current="page"]');
    if (!el || !el.offsetParent) { ind.style.opacity = "0"; return; }
    // The sidebar's light: it slides down or up to the current section.
    const jump = still || ind.style.opacity !== "1";
    if (jump) ind.style.transition = "none";
    ind.style.transform = `translateY(${el.offsetTop}px)`;
    ind.style.height = el.offsetHeight + "px";
    ind.style.opacity = "1";
    if (jump) { void ind.offsetWidth; ind.style.transition = ""; }
  }
  // A cartridge under the current tab: it slides to the next one, and
  // jumps when it first shows or the bar is resized.
  function slideInd(ind, el, still) {
    const jump = still || ind.style.opacity !== "1";
    if (jump) ind.style.transition = "none";
    ind.style.transform = `translateX(${el.offsetLeft}px) scaleX(${el.offsetWidth})`;
    ind.style.opacity = "1";
    if (jump) { void ind.offsetWidth; ind.style.transition = ""; }
  }
  new ResizeObserver(() => placeInd(true)).observe($("#bar-tabs"));

  // ------------------------------------------------------------- the room --
  // The art of the game in focus fills the window behind everything (Home's
  // game, a game's page; the Library keeps the last one, blurred and dark);
  // pages without a game of their own sit on flat obsidian. One art gives
  // way to the next over --room-fade.
  let roomKey = "";
  function roomItem() {
    if (S.view === "game" && S.page) return S.page.kind === "owned" ? ownedItem(S.page.id) : everyItem().find((x) => x.kind === S.page.kind && x.id === S.page.id);
    if (S.view === "home") return S.roomPick || recentItems(libraryItems(), false)[0] || null;
    // The Library: the game under the pointer, else the last played, dark.
    if (S.view === "library") return S.libFocus || recentItems(libraryItems(), false).find((x) => artFor(x, "hero").from === "hero") || null;
    // Editing a game: that game, dark, as behind the Library.
    if (S.view === "edit" && S.edit && !S.edit.isNew) return everyItem().find((x) => x.kind === "game" && x.id === S.edit.game.id) || null;
    return null;
  }
  function renderRoom() {
    const room = $("#room"), art = room.querySelector(".room-art");
    veil();
    const it = roomItem();
    const a = it && artFor(it, "hero"), url = a && a.from === "hero" ? a.url : "";
    room.classList.toggle("is-dim", S.view === "library" || S.view === "edit");
    room.classList.toggle("is-flat", !url);
    if (!url || url === roomKey) return;
    roomKey = url;
    const img = new Image();
    img.alt = ""; img.decoding = "async"; img.src = url;
    const show = () => {
      art.querySelectorAll("img").forEach((o) => { o.classList.remove("on"); setTimeout(() => o.remove(), 900); });
      art.appendChild(img);
      requestAnimationFrame(() => img.classList.add("on"));
    };
    img.decode ? img.decode().then(show, show) : show();
  }
  // Scrolling a page down veils the room: fully once the first screen has
  // scrolled away, so text further down never sits on bright art.
  let veilQueued = false;
  function veil() {
    const v = $("#view"), f = Math.min(1, v.scrollTop / Math.max(1, v.clientHeight * 0.6));
    $("#room").style.setProperty("--room-veil", (f * (S.view === "library" || S.view === "edit" ? 0.35 : 0.9)).toFixed(3));
  }
  $("#view").addEventListener("scroll", () => {
    if (veilQueued) return;
    veilQueued = true;
    requestAnimationFrame(() => { veilQueued = false; veil(); });
  }, { passive: true });
  // In the Library, resting on a cover brings that game's art into the room.
  let focusT = 0;
  $("#view").addEventListener("pointerover", (ev) => {
    if (S.view !== "library") return;
    const host = ev.target.closest && ev.target.closest(".ln-tile, .lib-cap, .lib-row");
    if (!host) return;
    clearTimeout(focusT);
    focusT = setTimeout(() => { const it = itemFor(ev.target) || itemFor(host.querySelector("[data-game], [data-key]")); if (it && it !== S.libFocus) { S.libFocus = it; renderRoom(); } }, 220);
  });
  // Settings, Appearance: see-through panels, the art behind pages, motion.
  function applyAppearance() {
    const s = cfg().settings || {}, c = document.documentElement.classList;
    c.toggle("glass-off", !!s.glassOff);
    c.toggle("room-off", !!s.roomOff);
    c.toggle("motion-less", s.motion === "less");
    c.toggle("motion-off", s.motion === "off");
  }
  function moreMenu(el) {
    const hid = [...document.querySelectorAll(".bar-tab[data-view]")].filter((t) => !t.offsetParent);
    toggleMenu(el, hid.map((t) => `<button data-action="nav" data-view="${t.dataset.view}" role="menuitem"${t.hasAttribute("aria-current") ? ' aria-current="page"' : ""}><span class="grow">${PAGE_NAMES[t.dataset.view]}</span>${(t.querySelector(".count") || {}).textContent || ""}</button>`).join(""));
  }

  const artOf = (it) => (it && (artFor(it, "hero").url || S.icons[coverKey(it)])) || "";
  function itemFor(el) {
    const host = el && el.closest("[data-game], [data-key]");
    if (!host) return null;
    const id = host.dataset.game || host.dataset.key, kind = host.dataset.game ? "game" : "found";
    if (isOwnedKey(id)) return ownedItem(id);
    return everyItem().find((x) => x.kind === kind && x.id === id) || null;
  }

  // ---------------------------------------------------------- right-click --
  function contextItems(it, host) {
    const g = it.g, f = it.f;
    const attr = itemAttr(it);
    if (it.kind === "owned") return [storeItem(it, "install"), `<button data-action="open-sheet" ${attr}>Open the game's page</button>`, "<hr>", ...collectionItems(it, host)];
    const s = g && session(g.id);
    const items = [];
    if (!s) {
      if (g) cfg().players.forEach((p) => items.push(`<button data-action="play-as" data-game="${esc(g.id)}" data-player="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span>Play as ${esc(p.name)}</button>`));
      else items.push(`<button data-action="play-found" ${attr}>${icon.play}Play</button>`);
    }
    items.push(`<button data-action="open-sheet" ${attr}>Open the game's page</button>`, ...collectionItems(it, host), "<hr>");
    if (g) items.push(`<button data-action="edit-game" ${attr} ${s ? "disabled" : ""}>Edit game</button>`);
    else items.push(`<button data-action="game-settings" ${attr}>Game settings</button>`);
    items.push(`<button data-action="open-game-folder" ${attr}>Open the game's folder</button>`,
      `<button data-action="${g ? "shortcut" : "found-shortcut"}" ${attr}>Create desktop shortcut</button>`);
    if (g && it.appId) items.push(`<button data-action="steam-verify" data-app="${it.appId}" ${s ? "disabled" : ""}>Verify files with Steam</button>`);
    if (f) items.push(...storeActs(it).filter((a) => a === "verify" || a === "open").map((a) => storeItem(it, a)));
    items.push("<hr>");
    if (g) return [...items, `<button class="danger" data-action="remove-game" ${attr}>Turn off profiles</button>`];
    return [...items, `<button class="danger" data-action="found-hide" ${attr}>${f.added ? "Remove from the Library" : "Hide from the Library"}</button>`,
      storeActs(it).includes("uninstall") ? storeItem(it, "uninstall") : "", removable(it) ? `<button class="danger" data-action="remove-pc" ${attr}>Remove from this PC</button>` : ""];
  }
  function openContext(ev, it, host) { ctxAt(ev.clientX, ev.clientY, `<div class="ctx-title">${esc(it.name)}</div>${contextItems(it, host).join("")}`); }
  function ctxAt(x, y, html) {
    const m = $("#ctx");
    if (ctxFrom) closeContext();
    m.innerHTML = html;
    m.hidden = false;
    m.style.left = Math.max(8, Math.min(x, innerWidth - m.offsetWidth - 8)) + "px";
    m.style.top = Math.max(8, Math.min(y, innerHeight - m.offsetHeight - 8)) + "px";
    m.style.transformOrigin = "0 0";
    const first = m.querySelector("button:not(:disabled)");
    if (first) first.focus({ preventScroll: true });
  }
  function closeContext(refocus) {
    const m = $("#ctx");
    if (!m || m.hidden) return;
    m.hidden = true; m.innerHTML = "";
    if (ctxFrom && ctxFrom.hasAttribute("aria-haspopup")) ctxFrom.setAttribute("aria-expanded", "false");
    if (refocus && ctxFrom) ctxFrom.focus({ preventScroll: true });
    ctxFrom = null;
  }
  document.addEventListener("contextmenu", (ev) => {
    const folder = ev.target.closest("#side-games .sg-folder:not(.is-fav) .sg-headrow");
    if (folder && folder.querySelector(".sg-head")) { ev.preventDefault(); ctxAt(ev.clientX, ev.clientY, tagMenuHTML(folder.parentNode.dataset.tag)); uiSound("open"); return; }
    const host = ev.target.closest(".card, .lib-cap, .lib-row, .sg-row, .ln-tile, .tn-card, .now-card");
    if (!host || S.picker || BP.on) return;
    const it = itemFor(host.matches("[data-game], [data-key]") ? host : host.querySelector("[data-game], [data-key]"));
    if (!it) return;
    ev.preventDefault();
    openContext(ev, it, host);
    uiSound("open");
  });
  document.addEventListener("pointerdown", (ev) => { if (!ev.target.closest("#ctx") && !(ctxFrom && ctxFrom.contains(ev.target))) closeContext(); }, true);
  document.addEventListener("focusin", (ev) => { if (!ev.target.closest("#ctx") && !(ctxFrom && ctxFrom.contains(ev.target))) closeContext(); });
  window.addEventListener("blur", () => closeContext());
  document.addEventListener("scroll", (ev) => { if (!(ev.target.closest && ev.target.closest("#ctx"))) closeContext(); }, true);

  // The seat: who's at the PC, until someone switches or Lootnest quits.
  // Every Play is theirs; "or play as…" and the picker are for anyone else.
  const seated = () => (S.seat && player(S.seat)) || null;
  let seatShown = null; // who the seat showed last: a new face pops in
  function renderSeat() {
    // Just you: someone is always seated, so Play is yours.
    if (solo() && !seated() && people()[0]) { S.seat = people()[0].id; api.setSeat(S.seat).catch(() => {}); }
    const seat = seated(), b = $("#bar-me"), sat = seat && seatShown !== null && seat.id !== seatShown;
    seatShown = seat ? seat.id : "";
    const s = seat && (S.d.sessions || []).find((x) => x.phase === "playing" && x.playerId === seat.id), g = s && game(s.gameId);
    const label = seat ? `${solo() ? "Playing as" : "At the PC:"} ${seat.name}${g ? `, playing ${g.name}` : ""}` : "Nobody's at the PC";
    b.classList.toggle("playing", !!g);
    b.innerHTML = seat ? `${lnAvatar(seat)}<span class="bar-label">${esc(seat.name)}</span>` : `<span class="bar-nobody">${GI.person}</span><span class="bar-label">Sit down</span>`;
    if (sat && !reduced()) b.querySelector(".ln-avatar").classList.add("sat");
    $("#room").style.setProperty("--seat-c", seat ? seat.color : "transparent");
    b.setAttribute("aria-label", label + ". Who's here");
    b.title = label;
    applyTheme();
  }
  // A menu in #ctx under the button that opened it, growing from it. The
  // same button closes it again.
  let ctxFrom = null;
  function menuUnder(el, html) {
    const m = $("#ctx");
    m.innerHTML = html;
    m.hidden = false;
    const b = el.getBoundingClientRect(), w = m.offsetWidth, h = m.offsetHeight; // the size before it grows in
    const left = Math.max(8, Math.min(b.left, innerWidth - w - 8));
    m.style.left = left + "px";
    m.style.top = (b.bottom + 4 + h > innerHeight - 8 ? Math.max(8, b.top - h - 4) : b.bottom + 4) + "px";
    m.style.transformOrigin = `${Math.round(b.left + b.width / 2 - left)}px 0`;
    ctxFrom = el;
    if (el.hasAttribute("aria-haspopup")) el.setAttribute("aria-expanded", "true");
    return m;
  }
  // above: over the button instead, for a bar at the bottom.
  function toggleMenu(el, html, above) {
    if (!$("#ctx").hidden && ctxFrom === el) { closeContext(); return; }
    const m = menuUnder(el, html);
    if (above) {
      m.style.top = Math.max(8, el.getBoundingClientRect().top - m.offsetHeight - 4) + "px";
      m.style.transformOrigin = m.style.transformOrigin.replace(/ 0$/, " 100%");
    }
    const first = m.querySelector('[aria-checked="true"], [aria-current="page"]') || m.querySelector("button:not(:disabled)");
    if (first) first.focus({ preventScroll: true });
  }
  // The seat's menu: who's here (1 to 9 seat someone), a run and Profiles.
  // Just you: your profile and its runs, and a run.
  function meMenu(el) {
    const seat = seated(), alone = solo();
    const playing = {};
    (S.d.sessions || []).forEach((s) => { if (s.phase === "playing") playing[s.playerId] = game(s.gameId); });
    const rows = grouped().map((p, i) => {
      const on = !!(seat && seat.id === p.id), g = playing[p.id];
      return `<button class="me-who${parentOf(p) ? " me-run" : ""}${g ? " playing" : ""}" data-action="seat-set" data-player="${esc(p.id)}" role="menuitemradio" aria-checked="${on}">${lnAvatar(p)}<span class="grow">${esc(p.name)}${g ? `<small>Playing ${esc(g.name)}</small>` : ""}</span>${on ? GI.tick : i < 9 ? `<kbd>${i + 1}</kbd>` : ""}</button>`;
    }).join("");
    const run = seat && cfg().players.length < MAX_PROFILES ? `<button data-action="add-run" data-player="${esc(personOf(seat).id)}" role="menuitem">${GI.plus}Add a run</button>` : "";
    const more = run + (alone ? "" : `<button data-action="nav" data-view="players" role="menuitem">${GI.profiles}Profiles</button>`);
    toggleMenu(el, `<div class="ctx-title">${alone ? "Playing as" : "Who's at the PC?"}</div><div role="group">${rows}
      ${alone ? "" : `<button data-action="seat-set" data-player="" role="menuitemradio" aria-checked="${!seat}"><span class="grow">Nobody: ask who's playing</span>${!seat ? GI.tick : ""}</button>`}</div>${more ? "<hr>" + more : ""}`);
  }
  function setSeat(pid) {
    S.seat = pid && player(pid) ? pid : "";
    api.setSeat(S.seat).catch(() => {});
    render();
    toast(S.seat ? `${solo() ? "" : `${pname(S.seat)} is at the PC. `}Play starts as ${pname(S.seat)}.` : "Nobody's at the PC. Play asks who's playing.");
  }

  // A labelled section, as every page draws them.
  const labelSec = (title, n, body, cls = "") => `<section${cls ? ` class="${cls}"` : ""} aria-label="${title}"><div class="ln-section-head"><div><h2>${title}${n ? `<span class="ln-count ln-num">${n}</span>` : ""}</h2></div></div>${body}</section>`;
  // A row of what needs a look: its lamp, what happened, one short line, its key.
  const lookRow = (title, line, key, lamp = "warn") => `<li class="look-row"><span class="ln-lamp${lamp ? " ln-lamp--" + lamp : ""}" aria-hidden="true"></span><div><b>${title}</b>${line ? `<p>${line}</p>` : ""}</div>${key ? `<div class="look-key">${key}</div>` : ""}</li>`;

  // The Lootnest mark, as in the top bar.
  const BRAND_MARK = '<svg class="brand-mark" viewBox="0 0 20 20" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path pathLength="1" d="M2.75 10.25c0 3.7 3.25 6.25 7.25 6.25s7.25-2.55 7.25-6.25"/><path pathLength="1" d="M5.75 10.25c0 2.2 1.9 3.5 4.25 3.5s4.25-1.3 4.25-3.5"/><circle cx="10" cy="6.25" r="2.5"/></svg>';


