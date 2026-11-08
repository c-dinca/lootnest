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
    S.view = view; S.menuFor = null; S.edit = null;
    if (P) {
      S.page = P; S.selected = { kind: P.kind, id: P.id }; tlAsked.clear(); docked.bar = docked.tabs = false;
      for (const k in S.details) if (S.details[k].error) delete S.details[k]; // asked again on this visit
    }
    if (view === "cloud") { loadBackupInfo(); loadSaves(); }
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
    if (!swapping) renderBar(); // nav has marked the tab it goes to
    const keepScroll = v.scrollTop;
    // A field being typed in keeps its text, focus and caret.
    const ae = document.activeElement, typing = ae && ["lib-q", "find-q", "gp-note", "gp-hours", "pt-rename"].includes(ae.id) && { id: ae.id, value: ae.value, at: ae.selectionStart };
    if (S.view === "home") v.innerHTML = viewHome();
    else if (S.view === "library") v.innerHTML = viewLibrary();
    else if (S.view === "game") v.innerHTML = viewGame();
    else if (S.view === "edit") v.innerHTML = viewEdit();
    else if (S.view === "players") v.innerHTML = viewPlayers();
    else if (S.view === "player") v.innerHTML = viewPlayer();
    else if (S.view === "settings") v.innerHTML = viewSettings();
    else if (S.view === "cloud") v.innerHTML = viewCloud();
    else if (S.view === "achievements") v.innerHTML = viewAchievements();
    else if (S.view === "find") v.innerHTML = viewFind();
    else if (S.view === "updates") v.innerHTML = viewUpdates();
    const first = v.firstElementChild, since = performance.now() - enteredAt;
    if (S.animateView && first) {
      // "kids": the page is already here, its numbers have just arrived.
      if (S.animateView !== "kids") { arrival++; enter(first, 0); }
      arriveOnce(v);
    } else if (first && swapping) first.classList.add("leaving");
    else if (first && since < ms("--dur-3") + 5 * ms("--stagger")) enter(first, since);
    S.animateView = false;
    hist.at = here();
    v.scrollTop = keepScroll;
    if (S.view === "game") fitDetails();
    stickHead();
    if (typing) { const q = $("#" + typing.id); if (q) { q.focus(); q.value = typing.value; if (typing.at != null) q.setSelectionRange(typing.at, typing.at); } }
    renderOverlay();
    loadCovers();
    backdropForView();
    if (BP.on) bpSoon();
  }

  // ------------------------------------------------------------- top bar --
  // The sections as tabs, with one line under the current section that
  // slides from tab to tab. Profiles and Settings mark theirs with a fill.
  // Narrow windows keep the first three tabs and put the rest under More.
  const PAGE_NAMES = { home: "Home", library: "Library", find: "Store", updates: "Updates", achievements: "Achievements", cloud: "Cloud saves", players: "Profiles", settings: "Settings" };
  function renderBar(to) {
    const top = to || (S.view === "game" ? S.from : S.view === "edit" ? "library" : S.view === "player" ? "players" : S.view);
    document.querySelectorAll(".topbar [data-view]").forEach((t) => (t.dataset.view === top ? t.setAttribute("aria-current", "page") : t.removeAttribute("aria-current")));
    $("#hist-back").disabled = !hist.back.length;
    $("#hist-fwd").disabled = !hist.fwd.length;
    placeInd(false);
  }
  function placeInd(still) {
    const ind = $("#bar-ind"), more = $("#bar-more"), cur = $('.bar-tab[aria-current="page"]');
    const el = !cur ? null : cur.offsetParent ? cur : more;
    more.classList.toggle("is-current", el === more);
    if (!el) { ind.style.opacity = "0"; return; }
    const cs = getComputedStyle(el), l = parseFloat(cs.paddingLeft), w = el.offsetWidth - l - parseFloat(cs.paddingRight);
    const jump = still || ind.style.opacity !== "1";
    if (jump) ind.style.transition = "none";
    ind.style.transform = `translateX(${el.offsetLeft + l}px) scaleX(${w})`;
    ind.style.opacity = "1";
    if (jump) { void ind.offsetWidth; ind.style.transition = ""; }
  }
  new ResizeObserver(() => placeInd(true)).observe($("#bar-tabs"));
  function moreMenu(el) {
    const hid = [...document.querySelectorAll(".bar-tab[data-view]")].filter((t) => !t.offsetParent);
    toggleMenu(el, hid.map((t) => `<button data-action="nav" data-view="${t.dataset.view}" role="menuitem"${t.hasAttribute("aria-current") ? ' aria-current="page"' : ""}><span class="grow">${PAGE_NAMES[t.dataset.view]}</span>${(t.querySelector(".count") || {}).textContent || ""}</button>`).join(""));
  }

  // ------------------------------------------------------------ backdrop --
  // The window takes on the colours of the game in focus: its art, blurred
  // far out of shape, behind everything. It follows the game the pointer
  // rests on in the Library, and the game whose page is open.
  let bdShown = "", bdFront = 0, bdTimer = 0;
  function setBackdrop(url) {
    if (url === bdShown) return;
    bdShown = url;
    const layers = document.querySelectorAll("#backdrop .bd");
    if (layers.length < 2) return;
    const next = layers[bdFront ^ 1], cur = layers[bdFront];
    if (url) next.style.backgroundImage = `url("${url}")`;
    next.classList.toggle("on", !!url);
    cur.classList.remove("on");
    bdFront ^= 1;
  }
  const artOf = (it) => (it && (artFor(it, "hero").url || S.icons[coverKey(it)])) || "";
  function itemFor(el) {
    const host = el && el.closest("[data-game], [data-key]");
    if (!host) return null;
    const id = host.dataset.game || host.dataset.key, kind = host.dataset.game ? "game" : "found";
    if (isOwnedKey(id)) return ownedItem(id);
    return everyItem().find((x) => x.kind === kind && x.id === id) || null;
  }
  // The colours change when a game is chosen, not as the pointer passes:
  // Home takes the game played last, a game's page that game, and the
  // Library the game opened last.
  function defaultBackdrop() {
    const find = (p) => p && everyItem().find((x) => x.kind === p.kind && x.id === p.id);
    if (S.view === "game" && S.page) return artOf(find(S.page));
    if (S.view === "home") return artOf(recentItems(libraryItems(), false)[0]);
    if (S.view === "library") return artOf(find(S.selected));
    return "";
  }
  function backdropForView() { clearTimeout(bdTimer); setBackdrop(defaultBackdrop()); }

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
  function renderSeat() {
    // Just you: someone is always seated, so Play is yours.
    if (solo() && !seated() && people()[0]) { S.seat = people()[0].id; api.setSeat(S.seat).catch(() => {}); }
    const seat = seated(), b = $("#bar-me");
    const s = seat && (S.d.sessions || []).find((x) => x.phase === "playing" && x.playerId === seat.id), g = s && game(s.gameId);
    const label = seat ? `${solo() ? "Playing as" : "At the PC:"} ${seat.name}${g ? `, playing ${g.name}` : ""}` : "Nobody's at the PC";
    b.classList.toggle("playing", !!g);
    b.innerHTML = seat ? `${lnAvatar(seat)}<span class="bar-label">${esc(seat.name)}</span>` : `<span class="bar-nobody">${GI.person}</span><span class="bar-label">Sit down</span>`;
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
    m.style.top = Math.min(b.bottom + 4, innerHeight - h - 8) + "px";
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

  // The Lootnest mark, as in the top bar.
  const BRAND_MARK = '<svg class="brand-mark" viewBox="0 0 20 20" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M2.75 10.25c0 3.7 3.25 6.25 7.25 6.25s7.25-2.55 7.25-6.25"/><path d="M5.75 10.25c0 2.2 1.9 3.5 4.25 3.5s4.25-1.3 4.25-3.5"/><circle cx="10" cy="6.25" r="2.5"/></svg>';


