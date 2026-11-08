  // ---------------------------------------------------------------- render --
  // Back and Forward: the pages visited, kept in memory (a page, its game or
  // profile, and how far down it was scrolled). A render that lands on
  // another page records the one it left; a re-render records nothing.
  const hist = { back: [], fwd: [], at: null, moving: false };
  function here() {
    const page = S.view === "game" && S.page ? { kind: S.page.kind, id: S.page.id } : null;
    const player = S.view === "player" ? S.playerPage : "";
    return { view: S.view, page, player, from: S.from, key: [S.view, page ? page.kind + ":" + page.id : "", player].join("|") };
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
    const from = back ? hist.back : hist.fwd, to = back ? hist.fwd : hist.back;
    if (!from.length || BP.on || S.picker || S.dialog || (S.view === "edit" && !confirmLeaveEdit())) return;
    const loc = from.pop();
    // A page not drawn yet (a quick second Back) keeps the scroll it had.
    if (hist.at.view !== "edit") to.push({ ...hist.at, scroll: hist.at.scroll ?? $("#view").scrollTop });
    hist.at = loc;
    const P = loc.page && { ...loc.page, cloud: null };
    transition(() => {
      enterView(loc.view, P); S.from = loc.from;
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
      S.page = P; S.selected = { kind: P.kind, id: P.id }; tlAsked.clear();
      for (const k in S.details) if (S.details[k].error) delete S.details[k]; // asked again on this visit
    }
    if (view === "cloud") { loadBackupInfo(); loadSaves(); }
    if (view !== "players") S.players = null;
  }

  function render() {
    if (!S.d) return;
    document.documentElement.style.setProperty("--p1", (cfg().players[0] || {}).color || "#6C9BD2");
    document.documentElement.style.setProperty("--p2", (cfg().players[1] || {}).color || "#D9738E");
    const top = S.view === "game" ? S.from : S.view === "edit" ? "library" : S.view === "player" ? "players" : S.view;
    document.querySelectorAll(".nav[data-view]").forEach((t) => (t.dataset.view === top ? t.setAttribute("aria-current", "page") : t.removeAttribute("aria-current")));
    const n = libraryItems().length;
    $("#lib-count").textContent = n ? n : "";
    renderWho();
    renderSideUpdate();
    renderSideGames();
    const v = $("#view");
    noteVisit(v.scrollTop);
    const keepScroll = v.scrollTop;
    // A field being typed in keeps its text, focus and caret.
    const ae = document.activeElement, typing = ae && ["lib-q", "find-q", "gp-note", "gp-hours"].includes(ae.id) && { id: ae.id, value: ae.value, at: ae.selectionStart };
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
    if (S.animateView && v.firstElementChild) {
      // From a transition the page itself already moves.
      const first = v.firstElementChild;
      if (S.animateView !== "kids") {
        first.classList.add("enter");
        first.addEventListener("animationend", (e) => { if (e.target === first) first.classList.remove("enter"); });
      }
      arriveOnce(v);
    }
    S.animateView = false;
    hist.at = here();
    v.scrollTop = keepScroll;
    if (S.view === "game") fitDetails();
    if (typing) { const q = $("#" + typing.id); if (q) { q.focus(); q.value = typing.value; if (typing.at != null) q.setSelectionRange(typing.at, typing.at); } }
    renderOverlay();
    loadCovers();
    backdropForView();
    if (BP.on) bpSoon();
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
  const artOf = (it) => (it && (S.art["hero:" + coverKey(it)] || S.covers[coverKey(it)] || S.icons[coverKey(it)])) || "";
  function itemFor(el) {
    const host = el && el.closest("[data-game], [data-key]");
    if (!host) return null;
    const id = host.dataset.game || host.dataset.key, kind = host.dataset.game ? "game" : "found";
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
  function openContext(ev, it) {
    const g = it.g, f = it.f;
    const attr = g ? `data-game="${esc(g.id)}"` : `data-key="${esc(f.key)}"`;
    const s = g && session(g.id);
    const items = [];
    if (!s) {
      if (g) cfg().players.forEach((p) => items.push(`<button data-action="play-as" data-game="${esc(g.id)}" data-player="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span>Play as ${esc(p.name)}</button>`));
      else items.push(`<button data-action="play-found" ${attr}>${icon.play}Play</button>`);
    }
    items.push(`<button data-action="open-sheet" ${attr}>Open the game's page</button>`, "<hr>");
    if (g) items.push(`<button data-action="edit-game" ${attr} ${s ? "disabled" : ""}>Edit game</button>`);
    else items.push(`<button data-action="game-settings" ${attr}>Game settings</button>`);
    items.push(`<button data-action="open-game-folder" ${attr}>Open the game's folder</button>`,
      `<button data-action="${g ? "shortcut" : "found-shortcut"}" ${attr}>Create desktop shortcut</button>`);
    if (it.appId) items.push(`<button data-action="steam-verify" data-app="${it.appId}" ${s ? "disabled" : ""}>Check the game's files</button>`);
    items.push("<hr>", g ? `<button class="danger" data-action="remove-game" ${attr}>Turn off profiles</button>`
      : `<button class="danger" data-action="found-hide" ${attr}>${f.added ? "Remove from the Library" : "Hide from the Library"}</button>`);
    const m = $("#ctx");
    m.innerHTML = `<div class="ctx-title">${esc(it.name)}</div>${items.join("")}`;
    m.hidden = false;
    const r = m.getBoundingClientRect();
    m.style.left = Math.min(ev.clientX, innerWidth - r.width - 8) + "px";
    m.style.top = Math.min(ev.clientY, innerHeight - r.height - 8) + "px";
    const first = m.querySelector("button:not(:disabled)");
    if (first) first.focus({ preventScroll: true });
  }
  function closeContext() { const m = $("#ctx"); if (m && !m.hidden) { m.hidden = true; m.innerHTML = ""; } }
  document.addEventListener("contextmenu", (ev) => {
    const host = ev.target.closest(".card, .lib-cap, .lib-row, .sg-row, .ln-tile, .tn-card, .now-card");
    if (!host || S.picker || BP.on) return;
    const it = itemFor(host.matches("[data-game], [data-key]") ? host : host.querySelector("[data-game], [data-key]"));
    if (!it) return;
    ev.preventDefault();
    openContext(ev, it);
  });
  document.addEventListener("pointerdown", (ev) => { if (!ev.target.closest("#ctx")) closeContext(); }, true);
  window.addEventListener("blur", closeContext);
  document.addEventListener("scroll", closeContext, true);

  // The seat: who's at the PC, until someone switches or Lootnest quits.
  // Every Play is theirs; "or play as…" and the picker are for anyone else.
  const seated = () => (S.seat && player(S.seat)) || null;
  function renderWho() {
    const playing = {}, ps = cfg().players;
    (S.d.sessions || []).forEach((s) => { if (s.phase === "playing") playing[s.playerId] = s; });
    // Just you: someone is always seated, so Play is yours.
    const alone = solo(), single = ps.length <= 1;
    $("#app").classList.toggle("is-solo", alone);
    $("#app").classList.toggle("is-single", single);
    if (alone && !seated() && people()[0]) { S.seat = people()[0].id; api.setSeat(S.seat).catch(() => {}); }
    const seat = seated();
    $("#who-head").innerHTML = alone ? `<span>Playing as</span>` : `<span>Who's here</span>${ps.length ? `<button class="who-switch" data-action="seat-menu" aria-haspopup="menu" aria-label="${seat ? "Switch" : "Sit down"}" title="Who's at the PC?">${GI.swap}<span>${seat ? "Switch" : "Sit down"}</span></button>` : ""}`;
    $("#who").innerHTML = grouped(ps).map((p) => {
      const s = playing[p.id];
      const g = s && game(s.gameId);
      const on = seat && seat.id === p.id;
      return `<button class="who-item ln-player${s ? " playing" : ""}${on ? " seated" : ""}${parentOf(p) ? " who-run" : ""}" style="--player:${esc(p.color)}" data-action="seat" data-player="${esc(p.id)}" aria-pressed="${!!on}" aria-label="${esc(p.name)}${g ? `, playing ${esc(g.name)}` : ""}"
        title="${alone ? (on ? `Play starts as ${esc(p.name)}.` : `Play as ${esc(p.name)} from now on.`) : on ? `${esc(p.name)} is at the PC. Click to leave the seat empty.` : `Sit ${esc(p.name)} at the PC: Play starts as ${esc(p.name)}.`}">${lnAvatar(p)}<span class="who-name">${esc(p.name)}${on && !single ? ` <span class="ln-tag ln-tag--player">${alone ? "Playing" : "Seated"}</span>` : ""}</span>
        <span class="what">${g ? `Playing ${esc(g.name)}` : ""}</span></button>`;
    }).join("");
    $("#who").scrollLeft = 0;
  }
  // A menu in #ctx under the button that opened it.
  function menuUnder(el, html) {
    const m = $("#ctx");
    m.innerHTML = html;
    m.hidden = false;
    const b = el.getBoundingClientRect(), r = m.getBoundingClientRect();
    m.style.left = Math.min(b.left, innerWidth - r.width - 8) + "px";
    m.style.top = Math.min(b.bottom + 4, innerHeight - r.height - 8) + "px";
    return m;
  }
  function seatMenu(el) {
    const seat = seated();
    const m = menuUnder(el, `<div class="ctx-title">Who's at the PC?</div>${cfg().players.map((p) => `<button data-action="seat-set" data-player="${esc(p.id)}" role="menuitemradio" aria-checked="${!!(seat && seat.id === p.id)}"><span class="dot" style="background:${esc(p.color)}"></span><span class="grow">${esc(p.name)}</span>${seat && seat.id === p.id ? GI.tick : ""}</button>`).join("")}
      <hr><button data-action="seat-set" data-player="" role="menuitemradio" aria-checked="${!seat}">Nobody: ask who's playing each time</button>`);
    const first = m.querySelector('[aria-checked="true"]') || m.querySelector("button");
    if (first) first.focus({ preventScroll: true });
  }
  function setSeat(pid) {
    S.seat = pid && player(pid) ? pid : "";
    api.setSeat(S.seat).catch(() => {});
    render();
    toast(S.seat ? `${solo() ? "" : `${pname(S.seat)} is at the PC. `}Play starts as ${pname(S.seat)}.` : "Nobody's at the PC. Play asks who's playing.");
  }

  // The Lootnest mark, as in the sidebar.
  const BRAND_MARK = '<svg class="brand-mark" viewBox="0 0 20 20" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M2.75 10.25c0 3.7 3.25 6.25 7.25 6.25s7.25-2.55 7.25-6.25"/><path d="M5.75 10.25c0 2.2 1.9 3.5 4.25 3.5s4.25-1.3 4.25-3.5"/><circle cx="10" cy="6.25" r="2.5"/></svg>';


