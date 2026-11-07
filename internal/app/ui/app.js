"use strict";
(function () {
  // Lootnest's window never opens other windows (links open in the browser).
  try { window.open = () => null; } catch (e) { /* read-only */ }
  // ------------------------------------------------------------------ api --
  const call = (name, ...args) => {
    const f = window[name];
    if (typeof f !== "function") return Promise.reject("Lootnest isn't ready yet.");
    return f(...args);
  };
  const api = {
    ready: () => call("ps_ready"),
    state: () => call("ps_state"),
    savePlayers: (p) => call("ps_savePlayers", p),
    saveGame: (g) => call("ps_saveGame", g),
    deleteGame: (id) => call("ps_deleteGame", id),
    saveSettings: (s) => call("ps_saveSettings", s),
    steamGames: () => call("ps_steamGames"),
    inspectSteam: (id) => call("ps_inspectSteam", id),
    inspectExe: (p) => call("ps_inspectExe", p),
    checkPath: (p) => call("ps_checkPath", p),
    browse: (id, kind, start) => call("ps_browse", id, kind, start || ""),
    play: (r) => call("ps_play", r),
    load: (r) => call("ps_load", r),
    shortcut: (id) => call("ps_shortcut", id),
    open: (kind, arg) => call("ps_open", kind, arg || ""),
    log: () => call("ps_log"),
    quit: () => call("ps_quit"),
    steamSetup: () => call("ps_steamSetup"),
    copy: (t) => call("ps_copy", t),
    steamPick: (r) => call("ps_steamPick", r),
    cancelRequest: (id) => call("ps_cancelRequest", id),
    found: () => call("ps_found"),
    scan: () => call("ps_scan"),
    ignoreFound: (k) => call("ps_ignoreFound", k),
    addFound: (keys) => call("ps_addFound", keys),
    foundGame: (k) => call("ps_foundGame", k),
    applyHooks: (restart) => call("ps_applyHooks", restart),
    cover: (appId, name, key) => call("ps_cover", appId, name, key),
    launchFound: (key) => call("ps_launchFound", key),
    checkUpdate: () => call("ps_checkUpdate"),
    installUpdate: () => call("ps_installUpdate"),
    cloudStatus: (id) => call("ps_cloudStatus", id),
    fixBlocked: (gameId, playerId) => call("ps_fixBlocked", gameId, playerId || ""),
    moveInfo: (appId) => call("ps_moveInfo", appId),
    moveGame: (appId, lib) => call("ps_moveGame", appId, lib),
    saveLibraryPref: (key, name, args) => call("ps_saveLibraryPref", key, name, args),
    addLibraryGame: (kind, appId, path) => call("ps_addLibraryGame", kind, appId || 0, path || ""),
    gameIcon: (exe) => call("ps_gameIcon", exe),
    art: (appId, name, key, kind) => call("ps_art", appId || 0, name, key, kind),
    setAvatar: (id, path) => call("ps_setAvatar", id, path),
    removeAvatar: (id) => call("ps_removeAvatar", id),
    setBanner: (id, path) => call("ps_setBanner", id, path),
    removeBanner: (id) => call("ps_removeBanner", id),
    setShelf: (id, key, e) => call("ps_setShelf", id, key, e),
    gameNews: (key, appId, name, exe) => call("ps_gameNews", key, appId || 0, name, exe || ""),
    openGameFolder: (kind, id) => call("ps_openGameFolder", kind, id),
    foundShortcut: (key) => call("ps_foundShortcut", key),
    steamVerify: (appId) => call("ps_steamVerify", appId),
  };

  // Small per-PC preferences (sort, cover size), kept in the WebView's own
  // storage; the app works the same without it.
  function pref(k, d) { try { return localStorage.getItem("ln." + k) || d; } catch (e) { return d; } }
  function setPref(k, v) { try { localStorage.setItem("ln." + k, v); } catch (e) { /* not kept */ } }

  // ---------------------------------------------------------------- state --
  const S = {
    d: null,          // AppState from Go
    view: "home",
    from: "home",      // the view a game's page goes back to
    selected: null,    // {kind, id} of the game last opened: the Library's colours follow it
    q: "",             // library search
    filter: "all",     // library filter chip
    covers: {},        // cover key -> data: URL, or "" when there is none (yet)
    icons: {},         // cover key -> the game exe's icon as a data: URL, or "" (shown when there's no cover)
    art: {},           // "hero:<appid>" / "logo:<appid>" -> Steam's wide art and logo, or ""
    news: {},          // cover key -> {items, newer, none, error} from Steam's news feed, or {loading: true}
    page: null,        // the game page: {kind: "game"|"found", id, cloud}
    libScroll: 0,      // where the Library was scrolled to, for coming back
    sort: pref("sort", "name"),       // Library order: name | recent | played
    density: pref("density", "m"),    // cover size: s | m | l
    ctx: null,         // the right-click menu: {x, y, kind, id}
    update: {},        // {available, version, page, upToDate, checking, error, step}
    animateCards: true, // stagger the cards in on the next library render
    animateView: true,  // fade the page in on the next render
    menuFor: null,
    picker: null,     // {gameId, mode, ask, playerId, owner, newMode, working}
    dialog: null,     // {kind, ...}
    edit: null,       // editor state
    players: null,    // players being edited
    logText: "",
    hooks: null,        // Steam launch option status per game
    hideHookBanner: false,
    found: { scanning: true, games: [] },
    adding: false,
    hooksBusy: false,
    moving: null,     // {appId, name, step, done, total} while Game Mover copies a game
  };
  const browseWaiters = {};
  let seq = 0;

  const $ = (sel, root = document) => root.querySelector(sel);
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const errText = (e) => (typeof e === "string" ? e : (e && e.message) || "Something went wrong.");
  const cfg = () => S.d.config;
  const player = (id) => cfg().players.find((p) => p.id === id);
  const game = (id) => cfg().games.find((g) => g.id === id);
  const color = (id) => (player(id) || {}).color || "var(--dim)";
  const pname = (id) => (player(id) || {}).name || "Someone";
  const session = (gid) => (S.d.sessions || []).find((s) => s.gameId === gid);
  const hasProfile = (gid, pid) => ((S.d.profiles || {})[gid] || []).includes(pid);
  const possessive = (n) => (/s$/i.test(n) ? n + "’" : n + "’s");
  const fmtSize = (b) => (b < 1024 ? b + " B" : b < 1048576 ? (b / 1024).toFixed(0) + " KB" : b < 1073741824 ? (b / 1048576).toFixed(1) + " MB" : (b / 1073741824).toFixed(2) + " GB");
  const short = (name) => (name.length > 22 ? name.slice(0, 20).trim() + "…" : name);

  const icon = {
    dots: '<svg viewBox="0 0 16 16"><circle cx="3" cy="8" r="1.4" fill="currentColor"/><circle cx="8" cy="8" r="1.4" fill="currentColor"/><circle cx="13" cy="8" r="1.4" fill="currentColor"/></svg>',
    folder: '<svg viewBox="0 0 16 16"><path d="M1.75 3.5h4l1.5 1.5h7v7.25a.75.75 0 0 1-.75.75H2.5a.75.75 0 0 1-.75-.75z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/></svg>',
    x: '<svg viewBox="0 0 16 16"><path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>',
    back: '<svg viewBox="0 0 16 16"><path d="M10 3 5 8l5 5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    plus: '<svg viewBox="0 0 16 16"><path d="M8 3v10M3 8h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
    play: '<svg viewBox="0 0 16 16" aria-hidden="true"><path d="M5 3.2v9.6a.6.6 0 0 0 .92.5l7.3-4.8a.6.6 0 0 0 0-1L5.92 2.7A.6.6 0 0 0 5 3.2z" fill="currentColor"/></svg>',
  };

  // --------------------------------------------------------------- toasts --
  function toast(text, kind) {
    const el = document.createElement("div");
    el.className = "toast" + (kind === "error" ? " error" : "");
    el.textContent = text;
    $("#toasts").appendChild(el);
    setTimeout(() => el.remove(), kind === "error" ? 7000 : 3800);
  }

  // ---------------------------------------------------------------- events --
  window.__ps = function (ev) {
    switch (ev.type) {
      case "sessions":
        if (!S.d) return;
        S.d.sessions = ev.sessions || [];
        S.d.active = ev.active || {};
        onSessions();
        refresh();
        break;
      case "launch":
        openPicker(ev.gameId, ev.reqId ? "steam" : "play", ev.reqId);
        break;
      case "found":
        loadFound();
        break;
      case "hooks":
        loadHooks().then(() => { render(); if (S.dialog && S.dialog.kind === "steam") renderOverlay(); });
        break;
      case "blocked":
        if (S.picker) { S.picker = null; render(); }
        S.menuFor = null;
        S.dialog = { kind: "blocked", gameId: ev.gameId, playerId: ev.playerId || "" };
        renderOverlay();
        setTimeout(() => { const b = $('[data-action="fix-blocked"]'); if (b) b.focus(); }, 20);
        break;
      case "blockedDone": {
        const D = S.dialog && S.dialog.kind === "blocked" ? S.dialog : null;
        if (ev.error) {
          if (D) { D.busy = false; renderOverlay(); }
          toast(ev.error, "error");
          break;
        }
        if (D) { S.dialog = null; renderOverlay(); }
        refresh();
        loadHooks().then(render);
        const g = game(ev.gameId);
        if (ev.need) openPicker(ev.gameId, "play");
        else toast(`Starting ${g ? g.name : "the game"} for ${pname(ev.playerId)}. Next time, start it from the ${g ? g.name : "game"} (Lootnest) shortcut on your desktop to choose who's playing.`);
        break;
      }
      case "update":
        S.update = Object.assign({}, ev, { checking: false });
        if (ev.manual && ev.upToDate) toast(`Lootnest ${S.d.version} is the latest version.`);
        if (ev.error) toast(ev.error, "error");
        render();
        break;
      case "updateStep":
        if (ev.error) { S.update.step = ""; toast(ev.error, "error"); } else S.update.step = ev.step;
        render();
        break;
      case "cover":
        pendingCovers.delete(ev.key);
        S.covers[ev.key] = ev.url;
        showCover(ev.key);
        break;
      case "art":
        S.art[ev.key] = ev.url;
        if (S.view === "game" || S.view === "library" || S.view === "home") render();
        break;
      case "moveInfo": {
        const D = S.dialog && S.dialog.kind === "move" && S.dialog.appId === ev.info.appId ? S.dialog : null;
        if (!D) break;
        D.info = ev.info;
        const room = (l) => l.free < 0 || l.free >= ev.info.size + MOVE_MARGIN;
        const first = (ev.info.libraries || []).find((l) => !l.current && room(l));
        D.pick = first ? first.path : "";
        renderOverlay();
        break;
      }
      case "moveProgress":
        S.moving = Object.assign(S.moving || {}, ev);
        if (S.dialog && S.dialog.kind === "move") renderOverlay();
        break;
      case "moveDone": {
        S.moving = null;
        if (S.dialog && S.dialog.kind === "move" && S.dialog.appId === ev.appId) { S.dialog = null; renderOverlay(); }
        if (ev.error) toast(`Couldn't move ${ev.name}: ${ev.error}${/[.!?]$/.test(ev.error) ? "" : "."}`, "error");
        else if (ev.leftover) toast(`${ev.name} moved. Some old files were in use and are still in ${ev.leftover}; you can delete that folder.`);
        else toast(`${ev.name} is now in ${ev.lib}.`);
        refresh();
        break;
      }
      case "news":
        S.news[ev.key] = ev;
        if (S.view === "game" || S.view === "library" || S.view === "home") render();
        break;
      case "avatar":
      case "banner": {
        const D = S.dialog && S.dialog.kind === "profile" ? S.dialog : null;
        if (D) { D.busy = false; if (!ev.error && ev.type === "banner") D.p.banner = "image"; if (!ev.error && ev.type === "avatar") D.p.avatar = true; }
        if (ev.error) toast(ev.error, "error");
        refresh();
        break;
      }
      case "pickerClosed":
        if (S.picker && S.picker.reqId) { S.picker = null; render(); }
        break;
      case "toast":
        toast(ev.text, ev.kind);
        break;
      case "browse": {
        const w = browseWaiters[ev.reqId];
        if (w) { delete browseWaiters[ev.reqId]; w(ev); }
        break;
      }
    }
  };

  function browse(kind, start) {
    const id = "b" + ++seq;
    return new Promise((resolve) => {
      browseWaiters[id] = resolve;
      api.browse(id, kind, start).catch((e) => { delete browseWaiters[id]; resolve({ error: errText(e) }); });
    });
  }

  async function refresh() {
    try { S.d = await api.state(); } catch (e) { toast(errText(e), "error"); return; }
    render();
  }

  // ---------------------------------------------------------------- render --
  function render() {
    if (!S.d) return;
    document.documentElement.style.setProperty("--p1", (cfg().players[0] || {}).color || "#6C9BD2");
    document.documentElement.style.setProperty("--p2", (cfg().players[1] || {}).color || "#D9738E");
    const top = S.view === "game" ? S.from : S.view === "edit" ? "library" : S.view === "player" ? "players" : S.view;
    document.querySelectorAll(".nav[data-view]").forEach((t) => (t.dataset.view === top ? t.setAttribute("aria-current", "page") : t.removeAttribute("aria-current")));
    const n = libraryItems().length;
    $("#lib-count").textContent = n ? n : "";
    renderWho();
    const v = $("#view");
    const keepScroll = v.scrollTop;
    const typing = document.activeElement && document.activeElement.id === "lib-q" ? document.activeElement.selectionStart : -1;
    if (S.view === "home") v.innerHTML = viewHome();
    else if (S.view === "library") v.innerHTML = viewLibrary();
    else if (S.view === "game") v.innerHTML = viewGame();
    else if (S.view === "edit") v.innerHTML = viewEdit();
    else if (S.view === "players") v.innerHTML = viewPlayers();
    else if (S.view === "player") v.innerHTML = viewPlayer();
    else if (S.view === "settings") v.innerHTML = viewSettings();
    if (S.animateView && v.firstElementChild) v.firstElementChild.classList.add("enter");
    S.animateView = false;
    v.scrollTop = keepScroll;
    if (typing >= 0) { const q = $("#lib-q"); if (q) { q.focus(); q.setSelectionRange(typing, typing); } }
    renderOverlay();
    loadCovers();
    backdropForView();
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
    return libraryItems().find((x) => x.kind === kind && x.id === id) || null;
  }
  // The colours change when a game is chosen, not as the pointer passes:
  // Home takes the game played last, a game's page that game, and the
  // Library the game opened last.
  function defaultBackdrop() {
    const find = (p) => p && libraryItems().find((x) => x.kind === p.kind && x.id === p.id);
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
    items.push(`<button data-action="open-game-folder" ${attr}>Open the game's folder</button>`);
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
    const host = ev.target.closest(".card, .capsule-wrap");
    if (!host || S.picker) return;
    const it = itemFor(host.querySelector("[data-game], [data-key]"));
    if (!it) return;
    ev.preventDefault();
    openContext(ev, it);
  });
  document.addEventListener("pointerdown", (ev) => { if (!ev.target.closest("#ctx")) closeContext(); }, true);
  window.addEventListener("blur", closeContext);
  document.addEventListener("scroll", closeContext, true);

  function renderWho() {
    const playing = {};
    (S.d.sessions || []).forEach((s) => { if (s.phase === "playing") playing[s.playerId] = s; });
    $("#who").innerHTML = cfg().players.map((p) => {
      const s = playing[p.id];
      const g = s && game(s.gameId);
      return `<div class="who-item${s ? " playing" : ""}" style="--c:${esc(p.color)}">${avatarHTML(p, "")}<span class="who-name">${esc(p.name)}</span>
        <span class="what">${g ? `Playing ${esc(g.name)}` : ""}</span></div>`;
    }).join("");
  }

  // --------------------------------------------------------------- library --
  function sessionText(s) {
    const n = pname(s.playerId);
    return { loading: `Loading ${possessive(n)} files`, launching: "Starting", playing: `${n} is playing`, saving: `Saving ${possessive(n)} progress` }[s.phase] || "";
  }

  const sourceLabel = (src) => ({ steam: "Steam", epic: "Epic Games", gog: "GOG", shortcut: "Shortcut", folder: "Game folder", added: "Added by you" }[src] || "Other");
  const filterOf = (src) => (src === "steam" || src === "epic" || src === "gog" ? src : "other");

  // Every game on the PC: the ones with profiles, then everything the scan found.
  function libraryItems() {
    const items = cfg().games.map((g) => ({
      kind: "game", id: g.id, name: g.name, appId: g.launch.type === "steam" ? g.launch.appId : 0,
      source: g.launch.type === "steam" ? "steam" : (S.d.sources || {})[g.id] || "other", exe: g.launch.type === "exe" ? g.launch.exe : "", g,
    }));
    (S.found.games || []).forEach((f) => items.push({ kind: "found", id: f.key, name: f.name, appId: f.appId || 0, source: f.source, exe: f.appId ? "" : f.exe || "", f }));
    return items.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }));
  }

  function shownItems() {
    const q = S.q.trim().toLowerCase();
    const list = libraryItems().filter((it) => {
      if (q && !it.name.toLowerCase().includes(q)) return false;
      if (S.filter === "profiles") return it.kind === "game";
      if (S.filter !== "all") return filterOf(it.source) === S.filter;
      return true;
    });
    if (S.sort === "recent") list.sort((a, b) => playedAt(b) - playedAt(a));
    else if (S.sort === "played") list.sort((a, b) => Math.max(totalPlayed(b), steamTime(b)) - Math.max(totalPlayed(a), steamTime(a)) || playedAt(b) - playedAt(a));
    return list;
  }
  const playedAt = (it) => { const r = (S.d.recent || {})[recentKey(it)]; return r ? new Date(r.at).getTime() : 0; };
  const playtimeOf = (it) => (S.d.playtime || {})[recentKey(it)] || {};
  const totalPlayed = (it) => Object.values(playtimeOf(it)).reduce((a, b) => a + b, 0);
  const steamTime = (it) => (S.d.steamTime || {})[recentKey(it)] || 0;
  // How much a game has been played: Lootnest's own count, or Steam's.
  const playedText = (it) => totalPlayed(it) ? hours(totalPlayed(it)) + " played in all" : steamTime(it) ? hours(steamTime(it)) + " on Steam" : "";
  function hours(sec) {
    if (sec < 3600) return Math.max(1, Math.round(sec / 60)) + " min";
    const h = sec / 3600;
    return (h < 10 ? Math.round(h * 10) / 10 : Math.round(h)) + " h";
  }

  function viewLibrary() {
    const all = libraryItems();
    const withProfiles = cfg().games.length;
    const scanning = S.found.scanning;
    const refreshing = S.found.refreshing;
    const chips = [["all", "All"], ["profiles", "With profiles"], ["steam", "Steam"], ["epic", "Epic"], ["gog", "GOG"], ["other", "Other"]];
    const head = `<div class="lib-head"><div class="grow"><h1>Library</h1>
        <p class="sub">${all.length ? `${all.length === 1 ? "1 game" : all.length + " games"} on this PC, ${withProfiles === 1 ? "1 with profiles" : withProfiles ? withProfiles + " with profiles" : "none with profiles yet"}` : "Every game on this PC, in one place."}</p></div>
        <input type="search" id="lib-q" class="search" placeholder="Search your games" value="${esc(S.q)}" autocomplete="off" spellcheck="false" aria-label="Search your games (Ctrl+F)">
        <button class="btn" data-action="add-game">${icon.plus}Add game</button>
        <button class="btn quiet" data-action="rescan" ${refreshing ? "disabled" : ""}>${refreshing ? "Scanning" : "Scan again"}</button></div>`;
    if (!all.length && !scanning) {
      return `<div class="page wide">${head}<div class="empty"><div><h2>No games found yet</h2>
        <p class="sub">Lootnest looks in Steam, Epic, GOG, game folders and your shortcuts. Add a game by hand if yours isn't found.</p></div>
        <button class="btn primary" data-action="add-game">${icon.plus}Add a game</button></div></div>`;
    }
    const sorts = [["name", "Name"], ["recent", "Recently played"], ["played", "Most played"]];
    const sizes = [["s", "Small covers"], ["m", "Medium covers"], ["l", "Large covers"]];
    return `<div class="page wide">${head}${hookBanner()}
      <div class="lib-tools"><div class="filters" role="group" aria-label="Show">${chips.map(([k, l]) => `<button class="chip" data-action="filter" data-filter="${k}" aria-pressed="${S.filter === k}">${l}</button>`).join("")}</div>
        <div class="row tools-right"><label class="sort"><span class="hint">Sort by</span><select id="lib-sort" aria-label="Sort by">${sorts.map(([k, l]) => `<option value="${k}" ${S.sort === k ? "selected" : ""}>${l}</option>`).join("")}</select></label>
        <div class="seg" role="group" aria-label="Cover size">${sizes.map(([k, l], i) => `<button data-action="density" data-size="${k}" aria-pressed="${S.density === k}" aria-label="${l}" title="${l}"><span class="sz sz${i}"></span></button>`).join("")}</div></div></div>
      ${scanning ? `<div class="scanning"><span class="spinner"></span>Looking for games on this PC</div>` : ""}
      ${coverHint(all)}
      <div class="grid size-${esc(S.density)}" id="grid">${gridHTML()}</div></div>`;
  }

  // Home: what you play, not everything you have. The game played last,
  // the others played lately, news from them, and a way into the Library.
  function viewHome() {
    const all = libraryItems();
    const hero = libraryHero(all), shelves = continueShelf(all) + newsShelf(all);
    const hour = new Date().getHours();
    const hello = hour < 5 ? "Late night" : hour < 12 ? "Good morning" : hour < 18 ? "Good afternoon" : "Good evening";
    const body = hero ? hero + shelves : `<div class="empty"><div><h2>Nothing played yet</h2>
        <p class="sub">Start a game from your Library and it shows up here, with the others you play and their news.</p></div>
        <button class="btn primary" data-action="nav" data-view="library">Open the Library</button></div>`;
    return `<div class="page wide">${updateBanner()}${hookBanner()}
      <div class="home-head"><h1>${hello}</h1><button class="btn quiet" data-action="nav" data-view="library">All ${all.length === 1 ? "1 game" : all.length + " games"}</button></div>
      ${body}</div>`;
  }

  // Shelves, as in the best launchers: the games played lately, then what's
  // new in them. Both only show what Lootnest actually knows.
  function recentItems(all, skipFirst) {
    const list = all.filter((it) => playedAt(it) > 0).sort((a, b) => playedAt(b) - playedAt(a));
    return skipFirst ? list.slice(1) : list;
  }
  // Covers for games shown outside the grid (Home), loaded once each.
  const coverAsked = new Set();
  async function ensureCover(it) {
    const key = coverKey(it);
    if (coverAsked.has(key) || key in S.covers) return;
    coverAsked.add(key);
    const can = it.appId || cfg().settings.downloadCovers;
    let url = "";
    if (can) { try { url = (await api.cover(it.appId || 0, it.name, key)) || ""; } catch (e) { /* none */ } }
    if (url) S.covers[key] = url;
    else if (it.exe && !(key in S.icons)) { try { S.icons[key] = (await api.gameIcon(it.exe)) || ""; } catch (e) { S.icons[key] = ""; } }
    if (url || S.icons[key]) render();
  }

  function continueShelf(all) {
    const list = recentItems(all, true).slice(0, 8);
    if (!list.length) return "";
    list.forEach((it) => { loadArt(it); ensureCover(it); });
    return `<section class="shelf" aria-label="Continue playing"><h2 class="shelf-title">Continue playing</h2>
      <div class="capsules">${list.map((it) => {
        const attr = it.kind === "game" ? `data-game="${esc(it.id)}"` : `data-key="${esc(it.id)}"`;
        const t = playedText(it);
        const k = coverKey(it), poster = !S.art["hero:" + k] && (S.covers[k] || S.icons[k]);
        return `<div class="capsule-wrap"><button class="capsule" data-action="open-sheet" ${attr} aria-label="${esc(it.name)}">${artHTML(it)}
          ${poster ? `<span class="capsule-poster${S.covers[k] ? "" : " icon"}" style="background-image:url('${poster}')"></span>` : ""}
          <span class="capsule-body"><b>${esc(it.name)}</b><span>${esc(lastPlayed(it))}${t ? `<span class="sep"></span>${esc(t)}` : ""}</span></span></button>
          ${it.kind === "game" && session(it.id) ? "" : `<button class="quick-play" data-action="${it.kind === "game" ? "play" : "play-found"}" ${attr} aria-label="Play ${esc(it.name)}" title="Play">${icon.play}</button>`}</div>`;
      }).join("")}</div></section>`;
  }
  function ensureNews(it) {
    const key = coverKey(it);
    if (S.news[key] || !cfg().settings.gameNews) return;
    S.news[key] = { loading: true };
    api.gameNews(key, it.appId, it.name, storeKept(it.source) ? "" : it.exe).catch(() => {});
  }
  function newsShelf(all) {
    if (!cfg().settings.gameNews) return "";
    const games = recentItems(all, false).slice(0, 8);
    if (!games.length) return "";
    const cards = [];
    games.forEach((it) => {
      ensureNews(it);
      const N = S.news[coverKey(it)];
      ((N && N.items) || []).slice(0, 3).forEach((n, i) => cards.push({ it, n, i }));
    });
    if (!cards.length) return "";
    cards.sort((a, b) => new Date(b.n.date) - new Date(a.n.date));
    return `<section class="shelf" aria-label="News from your games"><h2 class="shelf-title">News from your games</h2>
      <div class="news-cards">${cards.slice(0, 6).map(({ it, n, i }) => {
        const pic = n.image || S.art["hero:" + coverKey(it)] || S.covers[coverKey(it)] || "";
        return `<button class="news-card" data-action="read-news" data-key="${esc(coverKey(it))}" data-i="${i}">
          <span class="news-pic"${pic ? ` style="background-image:url('${esc(pic)}')"` : ""}></span>
          <span class="news-card-body"><span class="news-game">${n.update ? '<span class="dot"></span>' : ""}${esc(it.name)}<span class="news-when">${esc(day(n.date))}</span></span>
          <b>${esc(n.title)}</b></span></button>`;
      }).join("")}</div></section>`;
  }

  // Asked once: keep Lootnest up to date by itself, or not.
  function updatesPrompt() {
    const s = cfg().settings;
    if (s.updatesAsked || s.checkUpdates || S.update.available) return "";
    return `<div class="banner"><div><b>Keep Lootnest up to date?</b>
      <p>Lootnest checks GitHub every few hours, tells you with a notification when a new version is out, and installs it while it's in the tray and no game is running. Nothing about you or your games is sent.</p></div>
      <div class="row"><button class="btn quiet" data-action="updates-no">Not now</button><button class="btn primary" data-action="updates-yes">Keep it up to date</button></div></div>`;
  }

  function updateBanner() {
    const U = S.update;
    if (!U.available) return updatesPrompt();
    if (U.step) {
      return `<div class="banner"><div><b>${U.step === "installing" ? `Installing Lootnest ${esc(U.version)}` : `Downloading Lootnest ${esc(U.version)}`}</b>
        <p>${U.step === "installing" ? "Lootnest closes and opens again in a few seconds." : "Your players, games and profiles stay as they are."}</p></div><div class="bar" aria-hidden="true"></div></div>`;
    }
    return `<div class="banner"><div><b>Lootnest ${esc(U.version)} is ready to install</b><p>You have ${esc(S.d.version)}. Updating keeps your players, games and profiles.</p></div>
      <div class="row">${U.page ? `<button class="btn quiet" data-action="update-notes">What's new</button>` : ""}<button class="btn primary" data-action="update-now">Update now</button></div></div>`;
  }

  function coverHint(all) {
    if (cfg().settings.downloadCovers || S.hideCoverHint) return "";
    const n = all.filter((it) => !it.appId).length;
    if (!n) return "";
    return `<div class="banner"><div><b>Find covers and artwork for ${n === 1 ? "1 game" : n + " games"}</b>
      <p>Lootnest looks ${n === 1 ? "it" : "them"} up on the Steam store by name, and on SteamGridDB if you add a key in Settings. Only the games' names are sent. Everything found is saved on this PC.</p></div>
      <div class="row"><button class="btn quiet" data-action="covers-later">Not now</button><button class="btn primary" data-action="covers-on">Find covers</button></div></div>`;
  }

  function gridHTML() {
    const items = shownItems();
    if (!items.length) return `<p class="hint">${S.q ? `No game matches “${esc(S.q)}”.` : "No games here."}</p>`;
    const animate = S.animateCards;
    S.animateCards = false;
    return items.map((it, i) => cardHTML(it, animate ? Math.min(i, 14) : -1)).join("");
  }

  // Steam games have a cover in Steam's cache; other games only when the
  // user turned on "Download covers".
  const coverKey = (it) => (it.appId ? "a" + it.appId : "n:" + it.name.toLowerCase());
  function coverHTML(it) {
    const key = coverKey(it);
    const c = S.covers[key];
    const ic = !c && S.icons[key];
    const can = it.appId || cfg().settings.downloadCovers || it.exe;
    const show = can && (c !== "" || ic !== "" && it.exe);
    const src = c ? ` src="${c}" class="ready"` : ic ? ` src="${ic}" class="icon ready"` : "";
    const img = show ? `<img alt="" data-cover="${esc(key)}" data-app="${it.appId || 0}" data-name="${esc(it.name)}" data-exe="${esc(it.exe || "")}"${src}>` : "";
    return `<div class="nocover${ic ? " with-icon" : ""}"${c ? " hidden" : ""}${ic ? ` style="--icon:url('${ic}')"` : ""}><span>${esc(it.name)}</span></div>${img}`;
  }

  function cardHTML(it, delay) {
    const g = it.g;
    const owner = g && S.d.active[g.id];
    const s = g && session(g.id);
    const badge = s
      ? `<span class="badge"><span class="dot" style="background:${esc(color(s.playerId))}"></span>${esc(sessionText(s))}</span>`
      : owner && player(owner) ? `<span class="badge"><span class="dot" style="background:${esc(color(owner))}"></span>${esc(pname(owner))}</span>` : "";
    const meta = esc(sourceLabel(it.source));
    const dots = g ? cfg().players.filter((p) => hasProfile(g.id, p.id)).map((p) => `<span class="dot" style="background:${esc(p.color)}" title="${esc(p.name)} has a profile"></span>`).join("") : "";
    const attr = it.kind === "game" ? `data-game="${esc(it.id)}"` : `data-key="${esc(it.id)}"`;
    return `<div class="card${s ? " busy" : ""}${delay >= 0 ? " in" : ""}"${delay > 0 ? ` style="animation-delay:${delay * 22}ms"` : ""}>
      <button class="cover-btn" data-action="open-sheet" ${attr} aria-label="${esc(it.name)}"><div class="cover">${coverHTML(it)}${badge}</div></button>
      ${s ? "" : `<button class="quick-play" data-action="${it.kind === "game" ? "play" : "play-found"}" ${attr} aria-label="Play ${esc(it.name)}" title="Play">${icon.play}</button>`}
      <div class="card-title">${esc(it.name)}</div><div class="card-meta"><span>${meta}</span>${g ? `<span class="profiles" title="Profiles on">${dots || '<span class="dot"></span>'}</span>` : ""}</div></div>`;
  }

  // Covers load lazily as cards scroll into view; Steam keeps them on this PC.
  let coverObserver = null;
  function loadCovers() {
    if (!coverObserver) {
      coverObserver = new IntersectionObserver((entries) => {
        entries.forEach((e) => { if (e.isIntersecting) { coverObserver.unobserve(e.target); fetchCover(e.target); } });
      }, { root: $("#view"), rootMargin: "300px" });
    }
    document.querySelectorAll("img[data-cover]:not(.ready)").forEach((img) => coverObserver.observe(img));
  }

  const pendingCovers = new Set(); // downloads Go is doing; a "cover" event ends them
  async function fetchCover(img) {
    const key = img.dataset.cover;
    const can = +img.dataset.app || cfg().settings.downloadCovers;
    if (!(key in S.covers) && !pendingCovers.has(key)) {
      let url = "";
      if (can) { try { url = (await api.cover(+img.dataset.app, img.dataset.name, key)) || ""; } catch (e) { /* no cover */ } }
      if (url) S.covers[key] = url;
      else if (can && cfg().settings.downloadCovers) pendingCovers.add(key); // may still arrive
      else S.covers[key] = "";
    }
    // No cover (yet): the game's own icon, read from its exe on this PC.
    if (!S.covers[key] && img.dataset.exe && !(key in S.icons)) {
      try { S.icons[key] = (await api.gameIcon(img.dataset.exe)) || ""; } catch (e) { S.icons[key] = ""; }
    }
    showCover(key);
  }

  function showCover(key) {
    const url = S.covers[key], ic = S.icons[key];
    if (!url && !ic && !(key in S.covers)) return; // still waiting for a download
    document.querySelectorAll(`img[data-cover="${CSS.escape(key)}"]`).forEach((el) => {
      const tile = el.previousElementSibling;
      if (url) {
        el.onload = () => { el.classList.remove("icon"); el.classList.add("ready"); if (tile) { tile.hidden = true; tile.classList.remove("with-icon"); } };
        el.src = url;
      } else if (ic) {
        el.classList.add("icon");
        if (tile) { tile.classList.add("with-icon"); tile.style.setProperty("--icon", `url("${ic}")`); }
        el.onload = () => el.classList.add("ready");
        el.src = ic;
      } else el.remove();
    });
    if (!bdShown) backdropForView();
  }

  // ------------------------------------------------------------ game page --
  // A game opens as its own page: the game's wide art, its cover, a Play
  // button per player, and everything else about it. Opening it from the
  // Library carries the cover from the grid into the page.
  const reduced = () => window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  function transition(update, before, after) {
    if (!document.startViewTransition || reduced()) { update(); if (after) after(); return; }
    if (before) before();
    const t = document.startViewTransition(update);
    t.finished.finally(() => { if (after) after(); });
  }

  function openSheet(kind, id, fromEl) {
    if (S.view === "edit" && !confirmLeaveEdit()) return;
    if (S.dialog && S.dialog.kind !== "welcome") S.dialog = null;
    const P = { kind, id, cloud: null };
    const go = () => {
      if (S.view === "library") S.libScroll = $("#view").scrollTop;
      if (S.view === "library" || S.view === "home") { S.from = S.view; S.fromScroll = $("#view").scrollTop; }
      S.page = P; S.selected = { kind, id };
      S.view = "game"; S.menuFor = null;
      render(); $("#view").scrollTop = 0;
      $("#view").focus({ preventScroll: true });
    };
    const cover = fromEl && fromEl.querySelector(".cover");
    transition(() => { if (cover) cover.style.viewTransitionName = ""; go(); }, () => { if (cover) cover.style.viewTransitionName = "poster"; });
    const g = kind === "game" && game(id);
    if (g && g.launch.type === "steam") {
      api.cloudStatus(id).then((st) => { P.cloud = st; if (S.page === P && S.view === "game") render(); }).catch(() => {});
    }
  }

  function backToLibrary() {
    const P = S.page;
    const find = () => P && document.querySelector(`.cover-btn[data-${P.kind === "game" ? "game" : "key"}="${CSS.escape(P.id)}"] .cover`);
    let card = null;
    transition(() => {
      S.view = S.from === "home" ? "home" : "library"; S.page = null;
      render(); $("#view").scrollTop = S.fromScroll || 0;
      card = find();
      if (card) card.style.viewTransitionName = "poster";
    }, null, () => { if (card) card.style.viewTransitionName = ""; });
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
    const hero = S.art["hero:" + coverKey(it)];
    const c = S.covers[coverKey(it)] || S.icons[coverKey(it)] || "";
    if (hero) return `<div class="art" style="background-image:url('${hero}')"></div>`;
    if (c) return `<div class="art soft" style="background-image:url('${c}')"></div>`;
    return `<div class="art none"></div>`;
  }
  function titleHTML(it, cls) {
    const logo = S.art["logo:" + coverKey(it)];
    return logo ? `<h1 class="${cls} has-logo"><img src="${logo}" alt="${esc(it.name)}"></h1>` : `<h1 class="${cls}">${esc(it.name)}</h1>`;
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

  // One Play button per player for games with profiles, in the player's
  // colour: playing here starts with choosing who plays.
  function playButtons(it, big) {
    if (it.kind === "found") return `<button class="btn primary${big ? " big" : ""}" data-action="play-found" data-key="${esc(it.id)}">${icon.play}Play</button>`;
    const g = it.g, s = session(g.id), owner = S.d.active[g.id];
    if (s) return `<span class="status big"><span class="dot" style="background:${esc(color(s.playerId))}"></span>${esc(sessionText(s))}</span>`;
    return cfg().players.map((p) => `<button class="play-as${big ? " big" : ""}" style="--c:${esc(p.color)}" data-action="play-as" data-game="${esc(g.id)}" data-player="${esc(p.id)}">
      ${icon.play}<span>Play as <b>${esc(p.name)}</b></span>${owner === p.id ? `<span class="loaded" title="${esc(possessive(p.name))} files are loaded">loaded</span>` : ""}</button>`).join("");
  }

  function libraryHero(all) {
    const rec = S.d.recent || {};
    let best = null, at = 0;
    all.forEach((it) => { const r = rec[recentKey(it)]; const t = r ? new Date(r.at).getTime() : 0; if (t > at) { best = it; at = t; } });
    if (!best) return "";
    loadArt(best); ensureCover(best);
    const attr = best.kind === "game" ? `data-game="${esc(best.id)}"` : `data-key="${esc(best.id)}"`;
    return `<section class="hero" aria-label="Last played">${artHTML(best)}
      <div class="hero-body">${titleHTML(best, "hero-title")}
        <p class="hero-when">${esc(lastPlayed(best))}${playedText(best) ? `<span class="sep"></span>${esc(playedText(best))}` : ""}</p>
        <div class="hero-play">${playButtons(best, false)}<button class="btn ghost" data-action="open-sheet" ${attr}>Details</button></div>
      </div></section>`;
  }

  const day = (d) => new Date(d).toLocaleDateString(undefined, { day: "numeric", month: "short", year: new Date(d).getFullYear() === new Date().getFullYear() ? undefined : "numeric" });

  // An update or announcement, read inside Lootnest. Its body is HTML that
  // Lootnest made safe in Go (core.NewsHTML): only known elements, images
  // from Steam's servers, links that open in the browser.
  function viewReader() {
    const D = S.dialog;
    const n = ((S.news[D.key] || {}).items || [])[D.i];
    if (!n) return "";
    const it = libraryItems().find((x) => coverKey(x) === D.key);
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
  function newsHTML(it) {
    const head = `<h2 class="gp-sub">Updates and news</h2>`;
    if (!cfg().settings.gameNews) {
      return `<section class="news">${head}<p class="hint">See this game's latest updates and patch notes from Steam's news feed${storeKept(it.source) ? "" : ", and find out when a newer version than yours is out"}. Only the game's Steam id, or its name, is sent.</p>
        <div><button class="btn small" data-action="news-on">Show updates and news</button></div></section>`;
    }
    const key = coverKey(it);
    const N = S.news[key];
    if (!N) {
      S.news[key] = { loading: true };
      api.gameNews(key, it.appId, it.name, storeKept(it.source) ? "" : it.exe).catch(() => {});
    }
    if (!N || N.loading) return `<section class="news">${head}<p class="hint">Reading Steam's news for ${esc(it.name)}</p></section>`;
    if (N.none) return `<section class="news">${head}<p class="hint">${esc(it.name)} isn't on the Steam store, so there's no news feed for it.</p></section>`;
    if (N.error) return `<section class="news">${head}<p class="hint">${esc(N.error)}</p></section>`;
    const items = (N.items || []).slice(0, 6);
    const newer = N.newer ? `<div class="newer"><b>A newer version is out</b><p>${esc(N.newer.title)}, released ${esc(day(N.newer.date))}. The files of your copy are older than that.</p></div>` : "";
    return `<section class="news">${head}${newer}
      ${items.length ? `<ul class="news-list">${items.map((n, i) => `<li><button data-action="read-news" data-key="${esc(key)}" data-i="${i}">
        <span class="news-date">${esc(day(n.date))}</span>
        <span class="news-text"><span class="news-title">${n.update ? '<span class="dot" title="Update"></span>' : ""}${esc(n.title)}</span>
        ${n.summary ? `<span class="news-sum">${esc(n.summary)}</span>` : ""}</span>
        ${n.image ? `<span class="news-thumb" style="background-image:url('${esc(n.image)}')"></span>` : ""}</button></li>`).join("")}</ul>` : `<p class="hint">Steam has no news for this game yet.</p>`}
    </section>`;
  }

  function viewGame() {
    const P = S.page;
    const it = P && libraryItems().find((x) => x.kind === P.kind && x.id === P.id);
    if (!it) { S.view = "library"; return viewLibrary(); }
    loadArt(it);
    const back = `<button class="btn ghost back" data-action="back-library">${icon.back}${S.from === "home" ? "Home" : "Library"}</button>`;
    const facts = [];
    const add = (k, v) => facts.push(`<div><dt>${k}</dt><dd>${v}</dd></div>`);
    add("Store", esc(sourceLabel(it.source)));
    const lp = lastPlayed(it);
    if (lp) add("Last played", esc(lp));
    if (steamTime(it)) add("On Steam", `${esc(hours(steamTime(it)))} on this Steam account`);
    let list, note = "";
    if (it.kind === "found") {
      const f = it.f;
      add("Profiles", "Off: everyone shares the same settings and saves");
      if (f.args) add("Launch options", `<code>${esc(f.args)}</code>`);
      note = f.ready
        ? "Turn on profiles so everyone who plays gets their own settings and saves. Lootnest already found where this game keeps them."
        : "Turn on profiles so everyone who plays gets their own settings and saves. You'll choose where this game keeps them.";
      list = `<div class="gp-group"><button data-action="found-add" data-key="${esc(f.key)}" ${S.adding ? "disabled" : ""}><b>Turn on profiles</b><span>${esc(note)}</span></button></div>
        <div class="gp-group">
          <button data-action="game-settings" data-key="${esc(f.key)}"><b>Game settings</b><span>Name and launch options</span></button>
          <button data-action="found-shortcut" data-key="${esc(f.key)}"><b>Create desktop shortcut</b><span>Starts the game directly</span></button>
          <button data-action="open-game-folder" data-key="${esc(f.key)}"><b>Open the game's folder</b><span>Where it's installed</span></button>
          ${f.source === "steam" && f.appId ? `<button data-action="steam-verify" data-app="${f.appId}"><b>Check the game's files</b><span>Steam downloads any that are missing or damaged</span></button>
          <button data-action="move-open" data-app="${f.appId}" data-name="${esc(f.name)}"><b>Move to another drive</b><span>To another of your Steam libraries</span></button>` : ""}
        </div>
        <div class="gp-group"><button class="danger" data-action="found-hide" data-key="${esc(f.key)}"><b>${f.added ? "Remove from the Library" : "Hide from the Library"}</b></button></div>`;
    } else {
      const g = it.g, s = session(g.id), owner = S.d.active[g.id];
      const withProfile = cfg().players.filter((p) => hasProfile(g.id, p.id));
      add("Profiles", withProfile.length
        ? withProfile.map((p) => `<span class="who"><span class="dot" style="background:${esc(p.color)}"></span>${esc(p.name)}</span>`).join("")
        : "On; nobody has played yet");
      const pt = playtimeOf(it), total = totalPlayed(it);
      if (total) {
        const ps = cfg().players.filter((p) => pt[p.id]);
        add("Time played", `<div class="playtime"><div class="pt-bar">${ps.map((p) => `<span style="flex:${pt[p.id]};background:${esc(p.color)}" title="${esc(p.name)}: ${esc(hours(pt[p.id]))}"></span>`).join("")}</div>
          <div class="pt-legend">${ps.map((p) => `<span class="who"><span class="dot" style="background:${esc(p.color)}"></span>${esc(p.name)} <b>${esc(hours(pt[p.id]))}</b></span>`).join("")}</div></div>`);
      }
      add("Loaded now", owner && player(owner) ? `<span class="who"><span class="dot" style="background:${esc(color(owner))}"></span>${esc(possessive(pname(owner)))} files</span>` : "Nobody's files yet");
      const cloud = P.cloud && {
        none: "Doesn't sync the files profiles swap",
        elsewhere: "Syncs other files of this game, not the ones profiles swap",
        overlap: `Syncs ${P.cloud.count === 1 ? "1 of the files" : P.cloud.count + " of the files"} profiles swap. Lootnest checks them each time the game starts, so a cloud copy never ends up in someone's profile.`,
      }[P.cloud.state];
      if (cloud) add("Steam Cloud", esc(cloud));
      list = `<div class="gp-group">
          <button data-action="load-only" data-game="${esc(g.id)}" ${s ? "disabled" : ""}><b>Load someone's files</b><span>Without starting the game</span></button>
          <button data-action="edit-game" data-game="${esc(g.id)}" ${s ? "disabled" : ""}><b>Edit game</b><span>How it starts and what profiles swap</span></button>
          ${g.launch.type === "steam" && !g.steamDirect ? `<button data-action="steam-setup" data-game="${esc(g.id)}"><b>Ask who's playing in Steam</b><span>When the game starts from Steam's Play button</span></button>` : ""}
          <button data-action="shortcut" data-game="${esc(g.id)}"><b>Create desktop shortcut</b><span>Asks who's playing every time</span></button>
        </div>
        <div class="gp-group">
          <button data-action="open-game-folder" data-game="${esc(g.id)}"><b>Open the game's folder</b><span>Where it's installed</span></button>
          <button data-action="open-path" data-path="${esc((g.paths[0] || {}).path || "")}"><b>Open the save folder</b><span>The files profiles swap</span></button>
          <button data-action="open-backups" data-game="${esc(g.id)}"><b>Open backups</b><span>A copy is made before every swap</span></button>
          ${g.launch.type === "steam" ? `<button data-action="steam-verify" data-app="${g.launch.appId}" ${s ? "disabled" : ""}><b>Check the game's files</b><span>Steam downloads any that are missing or damaged</span></button>
          <button data-action="move-open" data-app="${g.launch.appId}" data-name="${esc(g.name)}" ${s ? "disabled" : ""}><b>Move to another drive</b><span>To another of your Steam libraries</span></button>` : ""}
        </div>
        <div class="gp-group"><button class="danger" data-action="remove-game" data-game="${esc(g.id)}"><b>Turn off profiles</b></button></div>`;
    }
    return `<div class="game-page">
      <div class="gp-hero">${artHTML(it)}${back}</div>
      <div class="gp-head">
        <div class="gp-poster"><div class="cover">${coverHTML(it)}</div></div>
        <div class="gp-intro">${titleHTML(it, "gp-title")}
          <div class="gp-play">${playButtons(it, true)}</div>
        </div>
      </div>
      <div class="gp-body">
        <div class="gp-side"><dl class="facts">${facts.join("")}</dl>${shelfBlock(it)}${newsHTML(it)}</div>
        <div class="gp-manage">${list}</div>
      </div></div>`;
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
      return `<div class="banner"><div><b>Let Steam start ${what}</b><p>${text}</p></div>
        <div class="row"><button class="btn" data-action="hooks-apply" ${S.hooksBusy ? "disabled" : ""}>${S.hooksBusy ? "Updating" : S.hooks.steamRunning ? "Restart Steam and update" : "Update now"}</button>
        <button class="btn quiet" data-action="hide-banner">Later</button></div></div>`;
    }
    if (!missing.length) return "";
    const what = missing.length === 1 ? esc(missing[0].name) : missing.length + " games";
    const text = S.hooks.steamRunning
      ? `So that Steam's Play button asks who's playing ${what}, Steam has to restart once (about 10 seconds). Lootnest does it for you.`
      : `Lootnest can set Steam up so its Play button asks who's playing ${what}.`;
    return `<div class="banner"><div><b>Make Steam ask who's playing</b><p>${text}</p></div>
      <div class="row"><button class="btn" data-action="hooks-apply" ${S.hooksBusy ? "disabled" : ""}>${S.hooksBusy ? "Setting up" : S.hooks.steamRunning ? "Restart Steam and set up" : "Set up"}</button>
      <button class="btn quiet" data-action="hide-banner">Later</button></div></div>`;
  }

  async function loadFound() {
    try { S.found = await api.found(); } catch (e) { /* not ready */ }
    if (S.view === "library" && !S.picker) render();
  }

  // ---------------------------------------------------------------- picker --
  function openPicker(gameId, mode, reqId) {
    const g = game(gameId);
    if (!g) { toast("That game isn't in Lootnest anymore.", "error"); if (reqId) api.cancelRequest(reqId); return; }
    if (S.picker && S.picker.reqId && S.picker.reqId !== reqId) api.cancelRequest(S.picker.reqId);
    S.menuFor = null;
    if (S.dialog && S.dialog.kind !== "welcome") S.dialog = null;
    S.picker = { gameId, mode: mode || "play", reqId: reqId || "" };
    render();
    setTimeout(() => { const pk = $(".picker"); if (pk) pk.focus(); }, 30); // keyboard 1-9 works without favouring anyone
  }

  function closePicker() {
    if (S.picker && S.picker.reqId && !S.picker.working) api.cancelRequest(S.picker.reqId);
    S.picker = null;
    render();
  }

  function slotState(gid, pid) {
    if (S.d.active[gid] === pid) return "Files loaded now";
    if (hasProfile(gid, pid)) return "Has a profile";
    return "First time";
  }

  function viewPicker() {
    const P = S.picker, g = game(P.gameId);
    if (!g) return "";
    if (P.working) {
      const c = color(P.playerId);
      return `<div class="picker" style="--c:${esc(c)}"><div class="picker-head"><div></div></div>
        <div class="working"><div class="big" style="color:${esc(c)}">${esc(pname(P.playerId))}</div>
        <div class="bar" style="--c:${esc(c)}"></div><p class="hint">${P.mode !== "load" ? `Loading ${esc(possessive(pname(P.playerId)))} files and starting ${esc(g.name)}` : `Loading ${esc(possessive(pname(P.playerId)))} files`}</p></div><div></div></div>`;
    }
    if (P.ask === "owner") {
      return `<div class="picker"><div class="picker-head"><div></div><button class="btn quiet" data-action="picker-back">${icon.back}Back</button></div>
        <div class="question"><div><h2>Whose files are in ${esc(g.name)} right now?</h2>
        <p>This is the first time Lootnest handles this game. The settings and saves on this PC will become that person's profile.</p></div>
        <div class="choices">
          ${cfg().players.map((p) => `<button class="choice" data-action="answer-owner" data-player="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span><span><b>${esc(possessive(p.name))}</b></span></button>`).join("")}
          <button class="choice" data-action="answer-owner" data-player="none"><span class="dot"></span><span><b>Nobody's, or not sure</b><span>They're kept as a backup in case you need them.</span></span></button>
        </div></div><div></div></div>`;
    }
    if (P.ask === "newMode") {
      const owner = P.owner && P.owner !== "none" ? P.owner : S.d.active[g.id];
      const ownerName = owner && player(owner) ? possessive(pname(owner)) + " files" : "the current files";
      const n = pname(P.playerId);
      return `<div class="picker"><div class="picker-head"><div></div><button class="btn quiet" data-action="picker-back">${icon.back}Back</button></div>
        <div class="question"><div><h2>${esc(n)} doesn't have a profile for ${esc(g.name)} yet</h2>
        <p>Choose what ${esc(n)} starts with. Whatever happens, the files on the PC now are kept.</p></div>
        <div class="choices">
          <button class="choice" data-action="answer-new" data-mode="fresh"><span class="dot" style="background:${esc(color(P.playerId))}"></span><span><b>Start fresh</b><span>Game defaults and no saves, as if ${esc(n)} just installed it.</span></span></button>
          <button class="choice" data-action="answer-new" data-mode="copy"><span class="dot"></span><span><b>Start from a copy of ${esc(ownerName)}</b><span>Same settings and saves to begin with; changes stay separate from then on.</span></span></button>
        </div></div><div></div></div>`;
    }
    const ps = cfg().players;
    return `<div class="picker" role="dialog" tabindex="-1" aria-label="Who's playing ${esc(g.name)}?">
      <div class="picker-head"><div><h1>Who's playing ${esc(g.name)}?</h1>
        <p class="hint">${P.mode === "steam" ? "Steam is starting the game. Their settings and saves load first." : P.mode === "play" ? "Their settings and saves load before the game starts." : "Their settings and saves are loaded; the game isn't started."}</p></div>
        <button class="btn quiet" data-action="picker-close">${P.mode === "steam" ? "Don't start" : "Cancel"}</button></div>
      <div class="slots">
        ${ps.map((p, i) => `<button class="slot" style="--c:${esc(p.color)}" data-action="pick" data-player="${esc(p.id)}">
          <span class="initial" aria-hidden="true">${esc(p.name.trim().charAt(0).toUpperCase())}</span>
          ${i < 9 ? `<span class="key" aria-hidden="true">${i + 1}</span>` : ""}
          <span class="name">${esc(p.name)}</span><span class="state">${esc(slotState(g.id, p.id))}</span></button>`).join("")}
      </div>
      <div class="picker-foot"><span>Press ${ps.length > 1 ? "1–" + Math.min(ps.length, 9) : "1"} to choose, Esc ${P.mode === "steam" ? "to not start the game" : "to cancel"}</span><span>${esc(g.name)}</span></div>
    </div>`;
  }

  async function runPick() {
    const P = S.picker;
    const req = { gameId: P.gameId, playerId: P.playerId, firstOwner: P.owner || "", newProfile: P.newMode || "", reqId: P.reqId || "" };
    try {
      const res = await (P.mode === "steam" ? api.steamPick(req) : P.mode === "play" ? api.play(req) : api.load(req));
      if (!S.picker) return;
      if (res && res.need) { S.picker.ask = res.need; render(); return; }
      S.picker.working = true;
      render();
    } catch (e) {
      toast(errText(e), "error");
      if (S.picker) { S.picker.ask = null; S.picker.working = false; render(); }
    }
  }

  function onSessions() {
    const P = S.picker;
    if (!P || !P.working) return;
    const s = session(P.gameId);
    if (!s || s.phase === "launching" || s.phase === "playing") S.picker = null;
  }

  // ---------------------------------------------------------------- dialogs --
  let confirmResolve = null;
  function confirmDialog(o) {
    return new Promise((resolve) => { confirmResolve = resolve; S.dialog = Object.assign({ kind: "confirm" }, o); renderOverlay(); setTimeout(() => { const b = $(".dialog .btn.primary, .dialog .btn.danger-primary"); if (b) b.focus(); }, 20); });
  }
  function closeDialog(v) { const wasSteam = S.dialog && S.dialog.kind === "steam"; S.dialog = null; renderOverlay(); if (wasSteam) render(); if (confirmResolve) { const r = confirmResolve; confirmResolve = null; r(v); } }

  // Redrawing the dialog that's already open (a choice clicked, progress, an
  // answer from Go) must not play its opening animation again.
  let shownOverlay = null, shownStep = "";
  function renderOverlay() {
    const o = $("#overlay");
    const P = S.picker;
    const now = P || S.dialog;
    const step = P ? (P.ask || "") + (P.working ? ":working" : "") : "";
    o.classList.toggle("still", now !== null && now === shownOverlay && step === shownStep);
    shownOverlay = now; shownStep = step;
    if (S.picker) { o.innerHTML = viewPicker(); return; }
    if (!S.dialog) { o.innerHTML = ""; return; }
    const D = S.dialog;
    if (D.kind === "confirm") {
      o.innerHTML = `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="alertdialog" aria-modal="true">
        <div><h2>${esc(D.title)}</h2><p class="sub">${esc(D.body)}</p></div>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">${esc(D.cancel || "Cancel")}</button>
        <button class="btn primary" data-action="dialog-ok">${esc(D.ok || "OK")}</button></div></div></div>`;
    } else if (D.kind === "add") {
      o.innerHTML = viewAddDialog();
    } else if (D.kind === "welcome") {
      o.innerHTML = viewWelcome();
    } else if (D.kind === "steam") {
      o.innerHTML = viewSteamSetup();
    } else if (D.kind === "blocked") {
      o.innerHTML = viewBlocked();
    } else if (D.kind === "move") {
      o.innerHTML = viewMove();
    } else if (D.kind === "shelf-note") {
      o.innerHTML = viewShelfNote();
    } else if (D.kind === "profile") {
      const q = $("#pe-q"), pos = q && document.activeElement === q ? q.selectionStart : -1;
      o.innerHTML = viewProfileEditor();
      if (pos >= 0) { const q2 = $("#pe-q"); q2.focus(); q2.setSelectionRange(pos, pos); }
    } else if (D.kind === "news") {
      o.innerHTML = viewReader();
      if (!$("#overlay").classList.contains("still")) { const b = $(".reader-body"); if (b) b.scrollTop = 0; }
    } else if (D.kind === "game-settings") {
      o.innerHTML = viewGameSettings();
    }
  }

  // ------------------------------------------------------------ game mover --
  const MOVE_MARGIN = 512 * 1024 * 1024; // room kept free beyond the game's size

  function openMove(appId, name) {
    S.dialog = { kind: "move", appId, name, info: null, pick: "" };
    renderOverlay();
    if (!S.moving) api.moveInfo(appId).catch((e) => { if (S.dialog && S.dialog.kind === "move") { S.dialog.info = { error: errText(e), libraries: [] }; renderOverlay(); } });
  }

  function viewMove() {
    const D = S.dialog, M = S.moving, I = D.info;
    const head = (sub) => `<div><h2>Move ${esc(D.name)} to another drive</h2>${sub ? `<p class="sub">${sub}</p>` : ""}</div>`;
    const wrap = (inner) => `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Move ${esc(D.name)}">${inner}</div></div>`;
    if (M && M.appId === D.appId) {
      const pct = M.total ? Math.min(100, Math.round((M.done / M.total) * 100)) : 0;
      const text = M.step === "steam" ? "Closing Steam" : M.total ? `Copying ${fmtSize(M.done)} of ${fmtSize(M.total)}` : "Starting";
      return wrap(`${head("")}<div class="mover"><div class="meter"><i style="width:${pct}%"></i></div><p class="hint">${esc(text)}</p></div>
        <p class="hint">You can close this window; Lootnest keeps going and tells you when it's done. Keep Steam closed until then.</p>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    }
    if (M) {
      return wrap(`${head("")}<p class="note">Lootnest is moving ${esc(M.name || "another game")}. Wait for it to finish, then move this one.</p>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    }
    if (!I) return wrap(`${head("Working out the game's size")}<div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Cancel</button></div>`);
    if (I.error) return wrap(`${head("")}<p class="note">${esc(I.error)}</p><div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    const cur = I.libraries.find((l) => l.current);
    const others = I.libraries.filter((l) => !l.current);
    const room = (l) => l.free < 0 || l.free >= I.size + MOVE_MARGIN;
    const list = others.length
      ? `<div class="choices">${others.map((l) => `<button class="choice${D.pick === l.path ? " on" : ""}" data-action="move-pick" data-lib="${esc(l.path)}" ${room(l) ? "" : "disabled"} aria-pressed="${D.pick === l.path}">
          <span class="dot"></span><span><b>${esc(l.path)}</b><span>${l.free < 0 ? "Free space unknown" : fmtSize(l.free) + " free"}${room(l) ? "" : " · not enough room"}</span></span></button>`).join("")}</div>`
      : `<p class="note">Steam has one library on this PC. To move games to another drive, add a library there in Steam: <b>Settings > Storage</b>, then <b>Add drive</b>. Then come back here.</p>`;
    return wrap(`${head(`${fmtSize(I.size)} · now in ${esc(cur ? cur.path : "")}`)}${list}
      ${others.length ? `<p class="hint">The game is copied and checked before anything is removed from where it is now.${I.steamRunning ? " Steam closes while the game moves and opens again when it's done." : ""}</p>` : ""}
      <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Cancel</button>
        ${others.length ? `<button class="btn primary" data-action="move-go" ${D.pick ? "" : "disabled"}>Move</button>` : ""}</div>`);
  }

  // --------------------------------------------- game settings (no profiles) --
  function openGameSettings(key) {
    const f = (S.found.games || []).find((x) => x.key === key);
    if (!f) return;
    S.dialog = { kind: "game-settings", key, busy: false };
    renderOverlay();
    setTimeout(() => { const i = $("#gs-name"); if (i) i.focus(); }, 30);
  }

  function viewGameSettings() {
    const D = S.dialog;
    const f = (S.found.games || []).find((x) => x.key === D.key);
    if (!f) return "";
    const steam = f.source === "steam";
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="${esc(f.name)} settings">
      <div><h2>${esc(f.name)} settings</h2><p class="sub">These settings don't depend on profiles and stay if you turn them on.</p></div>
      <div class="field"><label for="gs-name">Name in the Library</label>
        <input id="gs-name" type="text" value="${esc(f.name)}" maxlength="60" placeholder="${esc(f.scanName || f.name)}" spellcheck="false">
        ${f.scanName ? `<p class="hint">Found as ${esc(f.scanName)}. Leave it empty to use that name again.</p>` : ""}</div>
      <div class="field"><label for="gs-args">Launch options</label>
        <input id="gs-args" type="text" value="${esc(f.args || "")}" placeholder="${steam ? "-novid -high" : "-windowed"}" spellcheck="false">
        <p class="hint">${steam
          ? "Added when Lootnest or its desktop shortcut starts the game. Launch options set in Steam still apply too."
          : "Added when Lootnest or its desktop shortcut starts the game."}</p></div>
      <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
        <button class="btn primary" data-action="game-settings-save" ${D.busy ? "disabled" : ""}>Save</button></div>
    </div></div>`;
  }

  // ---------------------------------------------------- anti-cheat block --
  function viewBlocked() {
    const D = S.dialog, g = game(D.gameId);
    if (!g) return "";
    const who = D.playerId && player(D.playerId) ? pname(D.playerId) : "";
    return `<div class="scrim"><div class="dialog narrow" role="alertdialog" aria-modal="true" aria-label="An anti-cheat blocked ${esc(g.name)}">
      <div><h2>An anti-cheat blocked ${esc(g.name)}</h2>
        <p class="sub">Anti-cheats such as FACEIT AC only let Steam start the game, so Lootnest couldn't start it.</p></div>
      <div class="note">Lootnest can fix this now: Steam will start ${esc(g.name)} by itself (Steam restarts once, about 10 seconds), and ${who ? `the game starts for <b>${esc(who)}</b>` : "you choose who's playing"}.
        <br>Lootnest also puts a <b>${esc(g.name)} (Lootnest)</b> shortcut on your desktop. Start the game from it, or from Play here, and it asks who's playing every time.</div>
      ${who ? `<p class="hint">${esc(possessive(who))} files are loaded.</p>` : ""}
      <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Not now</button>
        <button class="btn primary" data-action="fix-blocked" ${D.busy ? "disabled" : ""}>${D.busy ? "Fixing and starting" : "Fix it and start the game"}</button></div>
    </div></div>`;
  }

  // ------------------------------------------------------- steam set-up --
  async function loadHooks() {
    try { S.hooks = await api.steamSetup(); } catch (e) { S.hooks = null; }
  }

  async function openSteamSetup(gameId) {
    S.menuFor = null;
    S.dialog = { kind: "steam", focus: gameId || "", copied: "" };
    renderOverlay();
    await loadHooks();
    if (S.dialog && S.dialog.kind === "steam") renderOverlay();
  }

  function viewSteamSetup() {
    const D = S.dialog;
    const H = S.hooks;
    let body;
    if (!H) body = `<p class="hint">Reading Steam's settings</p>`;
    else if (!H.games.length) body = `<div class="note">No Steam games in Lootnest yet. Add one first.</div>`;
    else {
      const list = (D.focus ? H.games.filter((h) => h.gameId === D.focus) : H.games).filter((h) => !h.direct);
      body = list.map((h) => `<div class="hook${h.detected ? " done" : ""}">
          <div class="row" style="justify-content:space-between"><b>${esc(h.name)}</b>
            <span class="${h.detected ? "ok-text" : "hint"}">${h.detected ? "Set up: Steam asks who's playing" : "Not set up yet"}</span></div>
          <div class="hook-line"><code class="selectable">${esc(h.option)}</code>
            <button class="btn small" data-action="copy-hook" data-game="${esc(h.gameId)}">${D.copied === h.gameId ? "Copied" : "Copy"}</button></div>
          ${h.current && !h.detected ? `<span class="hint">Includes the launch options this game already had.</span>` : ""}
        </div>`).join("");
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog" role="dialog" aria-modal="true" aria-label="Ask who's playing when started from Steam">
      <div class="page-head" style="margin:0"><div><h2>Ask who's playing when started from Steam</h2>
        <p class="sub">Do this once per game. Afterwards, pressing Play in Steam opens the picker, loads that person's files, then starts the game.</p></div>
        <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></div>
      ${H && H.games.some((h) => !h.detected && !h.direct) ? `<div class="note"><b>Easiest:</b> let Lootnest do it. ${H.steamRunning ? "Steam closes and reopens by itself (about 10 seconds)." : "Steam is closed, so this takes a second."}
        <div class="row" style="margin-top:10px"><button class="btn primary" data-action="hooks-apply" ${S.hooksBusy ? "disabled" : ""}>${S.hooksBusy ? "Setting up" : "Set up automatically"}</button></div></div>
      <p class="hint">Or by hand:</p>` : ""}
      <ol class="steps">
        <li>Copy the line for the game.</li>
        <li>In Steam, right-click the game in your library and choose <b>Properties</b>.</li>
        <li>On the <b>General</b> page, paste it into <b>Launch options</b>, replacing what's there.</li>
      </ol>
      ${H && H.error ? `<div class="note">${esc(H.error)}</div>` : ""}
      <div class="hooks">${body}</div>
      <div class="dialog-foot" style="justify-content:space-between"><span class="hint">Steam can take a minute to save the change. Games outside Steam: use the desktop shortcut from the game's menu.</span>
        <button class="btn" data-action="steam-recheck">Check again</button></div>
    </div></div>`;
  }

  // -------------------------------------------------------------- add game --
  function viewAddDialog() {
    const D = S.dialog;
    const q = (D.q || "").toLowerCase();
    const inLibrary = new Set((S.found.games || []).filter((f) => f.appId).map((f) => f.appId)
      .concat(cfg().games.filter((g) => g.launch.type === "steam").map((g) => g.launch.appId)));
    let list;
    if (!S.d.steam.found) list = `<div class="note">Steam wasn't found on this PC. You can still add any game by choosing its .exe.</div>`;
    else if (!D.steam) list = `<p class="hint">Looking for installed Steam games</p>`;
    else {
      const items = D.steam.filter((g) => !q || g.name.toLowerCase().includes(q));
      list = items.length
        ? `<div class="steam-list">${items.map((g) => `<button class="steam-item${inLibrary.has(g.appId) ? " added" : ""}" data-action="add-steam" data-app="${g.appId}"><b>${esc(g.name)}</b><span>${inLibrary.has(g.appId) ? "In your Library" : "Steam"}</span></button>`).join("")}</div>`
        : `<p class="hint">No installed Steam game matches “${esc(D.q)}”.</p>`;
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog" role="dialog" aria-modal="true" aria-label="Add a game">
      <div class="page-head" style="margin:0"><div><h2>Add a game</h2><p class="sub">It goes into your Library, ready to play. Turn on profiles from its panel if more than one person plays it.</p></div>
      <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></div>
      ${S.d.steam.found ? `<input type="text" id="steam-q" placeholder="Search your Steam games" value="${esc(D.q || "")}" autocomplete="off">` : ""}
      ${list}
      <div class="dialog-foot" style="justify-content:space-between"><span class="hint">Epic, GOG, Battle.net, Xbox and standalone games: choose the .exe.</span>
      <button class="btn" data-action="add-exe">Choose an .exe</button></div>
    </div></div>`;
  }

  async function startAddDialog() {
    S.menuFor = null;
    S.dialog = { kind: "add", q: "" };
    renderOverlay();
    setTimeout(() => { const i = $("#steam-q"); if (i) i.focus(); }, 30);
    if (S.d.steam.found) {
      try { S.dialog.steam = await api.steamGames(); } catch (e) { S.dialog.steam = []; toast(errText(e), "error"); }
      if (S.dialog && S.dialog.kind === "add") { const q = $("#steam-q"); const pos = q ? q.selectionStart : 0; renderOverlay(); const q2 = $("#steam-q"); if (q2) { q2.focus(); q2.setSelectionRange(pos, pos); } }
    }
  }

  // Adding a game puts it in the Library, without profiles; its panel then
  // opens, where profiles can be turned on.
  async function addToLibrary(kind, appId, path) {
    S.dialog = null; renderOverlay();
    try {
      const r = await api.addLibraryGame(kind, appId, path);
      await loadFound();
      if (S.view !== "library") { S.view = "library"; S.animateView = true; render(); }
      if (r.gameId) { openSheet("game", r.gameId); return; }
      toast(`${r.name} is in your Library`);
      openSheet("found", r.key);
    } catch (e) { toast(errText(e), "error"); }
  }

  const addSteam = (appId) => addToLibrary("steam", Number(appId), "");

  async function addExe() {
    const r = await browse("exe");
    if (!r.path) { if (r.error) toast(r.error, "error"); return; }
    await addToLibrary("exe", 0, r.path);
  }

  // ---------------------------------------------------------------- editor --
  function openEditor(e) {
    e.game = JSON.parse(JSON.stringify(e.game));
    e.game.paths.forEach((p) => { if (!p._k) p._k = "k" + ++seq; });
    e.checks = {};
    e.error = "";
    S.edit = e;
    S.view = "edit";
    render();
    $("#view").scrollTop = 0;
    e.game.paths.forEach(checkPath);
  }

  function editGame(id) {
    const g = game(id);
    if (!g) return;
    S.menuFor = null;
    const e = { isNew: false, game: g, inspect: null };
    openEditor(e);
    if (g.launch.type === "steam") api.inspectSteam(g.launch.appId).then((ins) => { if (S.edit === e) { e.inspect = ins; render(); } }).catch(() => {});
  }

  async function checkPath(p) {
    const e = S.edit;
    if (!e) return;
    const raw = p.path;
    if (!raw.trim()) { e.checks[p._k] = null; updatePathInfo(p); return; }
    try {
      const c = await api.checkPath(raw);
      if (S.edit !== e || p.path !== raw) return;
      e.checks[p._k] = c;
    } catch (err) { e.checks[p._k] = { error: errText(err) }; }
    updatePathInfo(p);
  }

  function pathInfo(p) {
    const c = S.edit.checks[p._k];
    if (!c) return `<div class="path-info">Checking</div>`;
    if (c.error) return `<div class="path-info bad">${esc(c.error)}</div>`;
    if (!c.exists) return `<div class="path-info">${esc(c.resolved)}. Doesn't exist yet; it's created the first time the game saves.</div>`;
    return `<div class="path-info">${esc(c.resolved)} (${c.isDir ? "folder" : "file"}, ${fmtSize(c.size)})</div>`;
  }

  function updatePathInfo(p) {
    const row = document.querySelector(`.path[data-k="${p._k}"] .path-info`);
    if (row) row.outerHTML = pathInfo(p);
  }

  function viewEdit() {
    const E = S.edit, g = E.game, ins = E.inspect || {};
    const taken = new Set(g.paths.map((p) => p.path.toLowerCase()));
    const sugg = (ins.suggestions || []).filter((s) => !taken.has(s.path.toLowerCase()));
    const cands = (ins.candidates || []).filter((c) => c.name.toLowerCase() !== (g.processName || "").toLowerCase()).slice(0, 6);
    const launch = g.launch.type === "steam"
      ? `<div class="note">Launched through Steam (app ${esc(g.launch.appId)}).</div>
        <label class="toggle"><span><b style="font-weight:500">Let Steam start this game</b><p>Turn this on if an anti-cheat, such as FACEIT AC, blocks the game when Lootnest starts it. Steam's Play button then starts the game without asking who's playing. Lootnest adds a desktop shortcut that asks every time.</p></span>
          <span class="switch"><input type="checkbox" data-field="steamDirect" ${g.steamDirect ? "checked" : ""}><span></span></span></label>`
      : `<div class="row"><input type="text" class="grow" data-field="exe" value="${esc(g.launch.exe)}" spellcheck="false"><button class="btn" data-action="browse-exe">Browse</button></div>`;
    return `<div class="page">
      <div class="page-head"><div><button class="btn quiet small" data-action="edit-cancel" style="margin:0 0 10px -12px">${icon.back}Library</button>
      <h1>${E.isNew ? "Add " + esc(g.name || "a game") : "Edit " + esc(g.name)}</h1></div></div>
      <div class="form">
        <section class="section">
          <div class="field"><label for="f-name">Name</label><input id="f-name" type="text" data-field="name" value="${esc(g.name)}" maxlength="60"></div>
          <div class="field"><span class="label">How it starts</span>${launch}</div>
        </section>

        <section class="section">
          <div class="section-head"><h2>Game process</h2><p>The program that runs while someone plays. Lootnest watches it to know when to save that person's progress.</p></div>
          <div class="row"><input type="text" class="grow" data-field="processName" value="${esc(g.processName)}" placeholder="for example cs2" spellcheck="false" style="max-width:320px"><span class="hint">.exe</span></div>
          ${cands.length ? `<div class="row"><span class="hint">Other programs in the game folder:</span>${cands.map((c) => `<button class="btn small" data-action="set-proc" data-name="${esc(c.name)}">${esc(c.name)}</button>`).join("")}</div>` : ""}
        </section>

        <section class="section">
          <div class="section-head"><h2>What gets swapped</h2><p>Every folder or file where the game keeps settings or saves. Each person gets their own copy of everything listed here.</p></div>
          ${ins.preset && ins.preset.note && g.paths.length ? `<div class="note">${esc(ins.preset.note)}</div>` : ""}
          <div class="paths">
            ${g.paths.length ? g.paths.map((p) => `<div class="path" data-k="${p._k}">
                <input type="text" data-field="path" data-k="${p._k}" value="${esc(p.path)}" spellcheck="false" aria-label="Save location">
                <div class="path-tools"><button class="btn icon quiet" data-action="open-path" data-path="${esc(p.path)}" aria-label="Open in Explorer">${icon.folder}</button>
                <button class="btn icon quiet" data-action="remove-path" data-k="${p._k}" aria-label="Remove">${icon.x}</button></div>
                ${pathInfo(p)}</div>`).join("") : `<p class="hint">Nothing yet. Add the game's settings folder and its saves folder.</p>`}
          </div>
          <div class="row"><button class="btn" data-action="add-path" data-kind="folder">${icon.plus}Add folder</button><button class="btn" data-action="add-path" data-kind="file">${icon.plus}Add file</button>
            <button class="btn quiet" data-action="pcgw">Look up where ${esc(g.name || "this game")} saves</button></div>
          ${sugg.length ? `<div class="suggest"><span class="label">Found on this PC</span>${sugg.map((s) => `<div class="suggest-item"><div class="grow"><code title="${esc(s.resolved)}">${esc(s.path)}</code><span class="hint">${esc(s.reason)}, ${fmtSize(s.size)}</span></div><button class="btn small" data-action="add-sugg" data-path="${esc(s.path)}">Add</button></div>`).join("")}</div>` : ""}
        </section>

        <section class="section">
          <div class="section-head"><h2>Launch options</h2><p>Optional. Added when Lootnest starts the game, for example <span class="selectable">-novid</span> or <span class="selectable">-w 1920 -h 1080</span>.</p></div>
          <div class="args">
            <span class="label">Everyone</span><input type="text" data-field="args" value="${esc(g.launch.args || "")}" spellcheck="false">
            ${cfg().players.map((p) => `<span class="who"><span class="dot" style="background:${esc(p.color)}"></span>${esc(p.name)}</span><input type="text" data-field="parg" data-player="${esc(p.id)}" value="${esc((g.playerArgs || {})[p.id] || "")}" spellcheck="false">`).join("")}
          </div>
        </section>
      </div>
      <div class="form-foot"><button class="btn primary" data-action="save-game" ${E.saving ? "disabled" : ""}>${E.isNew ? "Add game" : "Save changes"}</button>
        <button class="btn quiet" data-action="edit-cancel">Cancel</button><span class="error-text" id="edit-error">${esc(E.error)}</span></div>
    </div>`;
  }

  async function saveGame() {
    const E = S.edit;
    if (!E) return;
    E.saving = true; E.error = "";
    const g = JSON.parse(JSON.stringify(E.game));
    g.paths = g.paths.filter((p) => p.path.trim()).map((p) => ({ id: p.id || "", path: p.path.trim() }));
    try {
      const saved = await api.saveGame(g);
      toast(E.isNew ? `Added ${saved.name}` : `Saved ${saved.name}`);
      S.edit = null; S.view = "library"; S.animateView = true;
      await refresh();
      loadFound();
      loadHooks().then(render);
    } catch (e) {
      E.saving = false; E.error = errText(e);
      const b = $('[data-action="save-game"]'); if (b) b.disabled = false;
      const t = $("#edit-error"); if (t) t.textContent = E.error;
    }
  }

  // --------------------------------------------------------------- players --
  function viewWelcome() {
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    return `<div class="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Welcome">
      <div><h2 style="font-size:22px">Who shares this PC?</h2><p class="sub">Each person gets a colour and their own settings and saves in every game you add. You can add more people later.</p></div>
      <div class="players">${ps.map((p, i) => playerRow(p, i, ps.length, true)).join("")}</div>
      <div class="dialog-foot"><span class="error-text grow" id="players-error"></span><button class="btn primary" data-action="welcome-done">Continue</button></div>
    </div></div>`;
  }

  function playerRow(p, i, total, welcome) {
    return `<div class="player">
      <div class="swatches" role="radiogroup" aria-label="Colour for ${esc(p.name || "player " + (i + 1))}">${S.d.colors.map((c) => `<button class="swatch" style="background:${c}" data-action="swatch" data-i="${i}" data-color="${c}" aria-pressed="${c.toLowerCase() === (p.color || "").toLowerCase()}" aria-label="${c}"></button>`).join("")}</div>
      <input type="text" data-field="pname" data-i="${i}" value="${welcome && /^Player \d$/.test(p.name) ? "" : esc(p.name)}" placeholder="${welcome ? (i === 0 ? "Your name" : "Their name") : "Name"}" maxlength="24" aria-label="Name">
      ${welcome ? "" : `<button class="btn icon quiet" data-action="remove-player" data-i="${i}" aria-label="Remove ${esc(p.name)}" ${total <= 1 ? "disabled" : ""}>${icon.x}</button>`}
    </div>`;
  }

  // ------------------------------------------------------------ profiles --
  // Each player's own page, which they arrange themselves: a banner, their
  // picture, a few words, the game they're into, their numbers, activity,
  // shelf (playing, up next, finished, stopped) and collection.
  const SECTIONS = [
    ["now", "Into right now", "The game you're into, with a word on it"],
    ["stats", "Numbers", "Time played, games, finished, streak, longest session"],
    ["favorites", "Favourite games", "Up to six games, in your order"],
    ["activity", "Activity", "When you played, the last six months"],
    ["shelf", "Shelf", "Playing, up next, finished and stopped, with your ratings"],
    ["collection", "Collection", "Where your games come from"],
    ["recent", "Played lately", "The last games you played"],
  ];
  const SHELF = [["playing", "Playing"], ["next", "Up next"], ["finished", "Finished"], ["stopped", "Stopped"]];
  const sectionsOf = (p) => { const known = SECTIONS.map((s) => s[0]), on = (p.sections || []).filter((s) => known.includes(s)); return on.length ? on : known; };
  const shelfOf = (pid) => (S.d.shelf || {})[pid] || {};
  const daysOf = (pid) => ((S.d.activity || {}).days || {})[pid] || {};
  const dayKey = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  const daySum = (day) => Object.values(day || {}).reduce((a, b) => a + b, 0);
  const stars = (n) => "★★★★★".slice(0, n) + "☆☆☆☆☆".slice(0, 5 - n);

  const avatarHTML = (p, cls) => {
    const u = (S.d.avatars || {})[p.id];
    const g = !u && p.avatarGame && itemByKey(p.avatarGame);
    if (g) ensureCover(g);
    const gu = g && (S.covers[coverKey(g)] || S.icons[coverKey(g)]);
    const inner = u ? `<img src="${u}" alt="">` : gu ? `<img src="${gu}" alt="">` : esc(p.name.trim().charAt(0).toUpperCase());
    return `<span class="avatar ${cls || ""}" style="--c:${esc(p.color)}" aria-hidden="true">${inner}</span>`;
  };
  const itemByKey = (k) => libraryItems().find((it) => recentKey(it) === k || it.id === k);
  const itemAttr = (it) => (it.kind === "game" ? `data-game="${esc(it.id)}"` : `data-key="${esc(it.id)}"`);

  function playerStats(p) {
    const items = libraryItems();
    let total = 0, top = null, topT = 0, played = 0;
    items.forEach((it) => { const t = (playtimeOf(it) || {})[p.id] || 0; total += t; if (t) played++; if (t > topT) { top = it; topT = t; } });
    const recent = items.map((it) => ({ it, r: (S.d.recent || {})[recentKey(it)] })).filter((x) => x.r && x.r.player === p.id)
      .sort((a, b) => new Date(b.r.at) - new Date(a.r.at));
    played = Math.max(played, recent.length);
    const profiles = cfg().games.filter((g) => hasProfile(g.id, p.id)).length;
    const shelf = shelfOf(p.id), shelfN = {};
    Object.values(shelf).forEach((e) => { if (e.status) shelfN[e.status] = (shelfN[e.status] || 0) + 1; });
    const rated = Object.values(shelf).filter((e) => e.rating).length;
    // Streaks: days in a row with some play, counted to today (or yesterday).
    const days = daysOf(p.id), keys = Object.keys(days).filter((k) => daySum(days[k]) > 0).sort();
    let best = 0, run = 0, prev = null;
    keys.forEach((k) => { const d = new Date(k + "T12:00:00"); run = prev && (d - prev) / 864e5 < 1.5 ? run + 1 : 1; best = Math.max(best, run); prev = d; });
    let streak = 0;
    for (let d = new Date(), i = 0; i < 400; i++, d.setDate(d.getDate() - 1)) {
      if (daySum(days[dayKey(d)]) > 0) streak++;
      else if (i > 0 || streak > 0) break;
    }
    const act = S.d.activity || {};
    const longest = (act.longest || {})[p.id], sessions = (act.sessions || {})[p.id] || 0;
    const sources = new Set(items.filter((it) => ((playtimeOf(it) || {})[p.id] || 0) > 0 || shelf[recentKey(it)] || recent.some((x) => x.it === it)).map((it) => sourceLabel(it.source)));
    return { total, top, topT, recent, profiles, played, shelf, shelfN, rated, streak, best, longest, sessions, sources };
  }

  function bannerHTML(p) {
    const y = `background-position:center ${p.bannerY || 0}%`;
    if (p.banner === "image" && (S.d.banners || {})[p.id]) return `<div class="art" style="background-image:url('${S.d.banners[p.id]}');${y}"></div>`;
    const it = p.banner && p.banner !== "image" && itemByKey(p.banner);
    if (it) { loadArt(it); ensureCover(it); }
    const u = it ? artOf(it) : "";
    return u ? `<div class="art" style="background-image:url('${u}');${y}"></div>` : `<div class="art plain"></div>`;
  }

  function viewPlayer() {
    const p = player(S.playerPage);
    if (!p) { S.view = "players"; return viewPlayers(); }
    const st = playerStats(p);
    const last = st.recent.find((x) => x.it.kind === "game" && !session(x.it.id));
    const body = sectionsOf(p).map((s) => ppSection(s, p, st)).filter(Boolean).join("");
    return `<div class="profile-page" style="--c:${esc(p.color)}">
      <div class="pp-banner">${bannerHTML(p)}<button class="btn ghost back" data-action="nav" data-view="players">${icon.back}Players</button></div>
      <div class="pp-head">${avatarHTML(p, "xl")}
        <div class="pp-name"><h1>${esc(p.name)}</h1>${p.tagline ? `<p class="pp-tag">${esc(p.tagline)}</p>` : ""}${p.bio ? `<p class="pp-bio">${esc(p.bio).replace(/\n/g, "<br>")}</p>` : ""}</div>
        <div class="pp-actions">${last ? `<button class="btn primary" data-action="play-as" data-game="${esc(last.it.id)}" data-player="${esc(p.id)}">${icon.play}Play ${esc(last.it.name)}</button>` : ""}
          <button class="btn" data-action="edit-profile" data-player="${esc(p.id)}">Edit profile</button></div></div>
      <div class="pp-body">${body || `<p class="hint">Nothing to show yet: <b>Edit profile</b> chooses what this page shows.</p>`}</div>
    </div>`;
  }

  function ppSection(s, p, st) {
    const sec = (title, inner, extra) => `<section class="pp-section pp-${s}"><div class="pp-sh"><h2 class="shelf-title">${title}</h2>${extra || ""}</div>${inner}</section>`;
    if (s === "now") {
      const it = p.now && itemByKey(p.now);
      if (!it) return "";
      loadArt(it); ensureCover(it);
      const t = (playtimeOf(it) || {})[p.id] || 0, e = st.shelf[recentKey(it)] || {};
      return `<section class="pp-section pp-now"><button class="now-card" data-action="open-sheet" ${itemAttr(it)}>
        ${artHTML(it)}<span class="now-text"><span class="now-label">Into right now</span><b>${esc(it.name)}</b>
        ${p.nowNote ? `<span class="now-note">${esc(p.nowNote)}</span>` : ""}
        <span class="now-meta">${t ? `<span>${esc(hours(t))} played</span>` : ""}${e.rating ? `<span class="stars" aria-label="${e.rating} of 5 stars">${stars(e.rating)}</span>` : ""}</span></span></button></section>`;
    }
    if (s === "stats") {
      const stat = (v, l) => `<div class="pstat"><b>${v}</b><span>${l}</span></div>`;
      return `<section class="pp-section pp-stats">
        ${stat(st.total ? esc(hours(st.total)) : "None yet", "played with Lootnest")}
        ${stat(String(st.played), st.played === 1 ? "game played" : "games played")}
        ${stat(String(st.shelfN.finished || 0), "finished")}
        ${stat(st.streak ? st.streak + (st.streak === 1 ? " day" : " days") : "None", st.best > st.streak ? `in a row now, best ${st.best}` : "in a row")}
        ${stat(st.longest ? esc(hours(st.longest.seconds)) : "None yet", st.longest && itemByKey(st.longest.key) ? "longest session, " + esc(itemByKey(st.longest.key).name) : "longest session")}
        ${stat(st.top ? esc(st.top.name) : "None yet", st.top ? esc(hours(st.topT)) + ", most played" : "most played")}
      </section>`;
    }
    if (s === "favorites") {
      const favs = (p.favorites || []).map(itemByKey).filter(Boolean);
      favs.forEach(ensureCover);
      return sec("Favourite games", favs.length ? `<div class="grid size-l">${favs.map((it) => cardHTML(it, -1)).join("")}</div>`
        : `<p class="hint">Pick up to six games to show here: <b>Edit profile</b>, then <b>Showcase</b>.</p>`);
    }
    if (s === "activity") return sec("Activity", activityHTML(p, st));
    if (s === "shelf") {
      const tab = S.shelfTab && SHELF.some((x) => x[0] === S.shelfTab) ? S.shelfTab : (SHELF.find((x) => st.shelfN[x[0]]) || SHELF[0])[0];
      const list = Object.entries(st.shelf).filter(([, e]) => e.status === tab).map(([k, e]) => ({ it: itemByKey(k), e })).filter((x) => x.it)
        .sort((a, b) => new Date(b.e.updated) - new Date(a.e.updated));
      list.forEach((x) => ensureCover(x.it));
      const tabs = `<div class="segt" role="tablist">${SHELF.map(([k, l]) => `<button role="tab" aria-selected="${k === tab}" data-action="shelf-tab" data-tab="${k}">${l}<span class="n">${st.shelfN[k] || 0}</span></button>`).join("")}</div>`;
      return sec("Shelf", list.length ? `<div class="shelf-grid">${list.map(({ it, e }) => `<button class="shelf-card" data-action="open-sheet" ${itemAttr(it)}>
          <div class="cover">${coverHTML(it)}</div><b>${esc(it.name)}</b>${e.rating ? `<span class="stars" aria-label="${e.rating} of 5 stars">${stars(e.rating)}</span>` : ""}
          ${e.note ? `<span class="shelf-note">${esc(e.note)}</span>` : ""}</button>`).join("")}</div>`
        : `<p class="hint">Nothing here yet. On a game's page, <b>Shelf</b> says where it stands for you: playing, up next, finished or stopped.</p>`, tabs);
    }
    if (s === "collection") {
      const items = libraryItems(), by = {};
      items.forEach((it) => {
        const t = (playtimeOf(it) || {})[p.id] || 0;
        if (!t && !st.shelf[recentKey(it)] && !st.recent.some((x) => x.it === it)) return;
        const l = sourceLabel(it.source);
        by[l] = by[l] || { n: 0, t: 0 }; by[l].n++; by[l].t += t;
      });
      const rows = Object.entries(by).sort((a, b) => b[1].n - a[1].n || b[1].t - a[1].t);
      const max = Math.max(1, ...rows.map((r) => r[1].n));
      return sec("Collection", rows.length ? `<ul class="bars">${rows.map(([l, v]) => `<li><span class="bl">${esc(l)}</span>
          <span class="bt"><span style="width:${Math.max(4, (v.n / max) * 100)}%"></span></span><span class="bv">${v.n} ${v.n === 1 ? "game" : "games"}${v.t ? ", " + esc(hours(v.t)) : ""}</span></li>`).join("")}</ul>
          <p class="hint">${items.length} games in the Library on this PC, from ${new Set(items.map((it) => sourceLabel(it.source))).size} places.</p>`
        : `<p class="hint">Games you play or put on your shelf show up here, by where they come from.</p>`);
    }
    if (s === "recent") {
      if (!st.recent.length) return "";
      return sec("Played lately", `<ul class="pp-recent">${st.recent.slice(0, 8).map(({ it, r }) => `<li><button data-action="open-sheet" ${itemAttr(it)}><b>${esc(it.name)}</b><span>${esc(ago(r.at))}</span></button></li>`).join("")}</ul>`);
    }
    return "";
  }

  // 26 weeks of play, a column a week, a square a day, darker for more.
  function activityHTML(p, st) {
    const days = daysOf(p.id), today = new Date();
    const start = new Date(today); start.setHours(12, 0, 0, 0);
    start.setDate(start.getDate() - ((start.getDay() + 6) % 7) - 25 * 7);
    const lvl = (s) => (s <= 0 ? 0 : s < 1800 ? 1 : s < 3600 ? 2 : s < 7200 ? 3 : 4);
    let cells = "", week = 0, month = 0, months = "";
    const tKey = dayKey(today), byGame = {};
    for (let i = 0, d = new Date(start); i < 26 * 7; i++, d.setDate(d.getDate() + 1)) {
      const k = dayKey(d), sum = daySum(days[k]), future = k > tKey;
      if (i % 7 === 0 && d.getDate() <= 7) months += `<span style="grid-column:${i / 7 + 1}">${d.toLocaleDateString(undefined, { month: "short" })}</span>`;
      cells += `<span class="hm l${future ? "x" : lvl(sum)}" style="grid-column:${Math.floor(i / 7) + 1};grid-row:${(i % 7) + 1}" title="${esc(d.toLocaleDateString(undefined, { weekday: "short", day: "numeric", month: "short" }))}${sum ? ": " + esc(hours(sum)) : ""}"></span>`;
    }
    for (let i = 0, d = new Date(today); i < 30; i++, d.setDate(d.getDate() - 1)) {
      const day = days[dayKey(d)] || {};
      if (i < 7) week += daySum(day);
      month += daySum(day);
      Object.entries(day).forEach(([k, v]) => { byGame[k] = (byGame[k] || 0) + v; });
    }
    const top = Object.entries(byGame).map(([k, v]) => ({ it: itemByKey(k), v })).filter((x) => x.it).sort((a, b) => b.v - a.v).slice(0, 4);
    const max = Math.max(1, ...top.map((x) => x.v));
    return `<div class="activity">
      <div class="heat-wrap"><div class="heat-months">${months}</div><div class="heat" aria-label="Time played per day">${cells}</div>
        <div class="heat-key"><span>Less</span><span class="hm l0"></span><span class="hm l1"></span><span class="hm l2"></span><span class="hm l3"></span><span class="hm l4"></span><span>More</span></div></div>
      <div class="act-side"><div class="act-n"><div><b>${week ? esc(hours(week)) : "0 min"}</b><span>the last 7 days</span></div><div><b>${month ? esc(hours(month)) : "0 min"}</b><span>the last 30 days</span></div><div><b>${st.sessions}</b><span>${st.sessions === 1 ? "session" : "sessions"} in all</span></div></div>
        ${top.length ? `<ul class="bars">${top.map(({ it, v }) => `<li><span class="bl">${esc(it.name)}</span><span class="bt"><span style="width:${Math.max(4, (v / max) * 100)}%"></span></span><span class="bv">${esc(hours(v))}</span></li>`).join("")}</ul>`
          : `<p class="hint">Games started from Lootnest are counted here, day by day.</p>`}</div></div>`;
  }

  // The game page's Shelf: where the game stands for each player.
  function shelfBlock(it) {
    const k = recentKey(it);
    return `<div class="gp-shelf"><h2 class="gp-sub">Shelf</h2>${cfg().players.map((p) => {
      const e = shelfOf(p.id)[k] || {};
      return `<div class="sh-row" style="--c:${esc(p.color)}"><div class="sh-top">${avatarHTML(p, "sm")}<span class="sh-name">${esc(p.name)}</span>
        <button class="btn quiet sh-note" data-action="shelf-note" data-player="${esc(p.id)}" data-k="${esc(k)}">${e.note ? "Edit note" : "Add a note"}</button></div>
        <div class="sh-ctl"><div class="segt small" role="group" aria-label="${esc(p.name)}: where this game stands">${SHELF.map(([s, l]) => `<button data-action="shelf-set" data-player="${esc(p.id)}" data-k="${esc(k)}" data-status="${s}" aria-pressed="${e.status === s}">${l}</button>`).join("")}</div>
        <div class="rate" role="group" aria-label="${esc(p.name)}'s rating">${[1, 2, 3, 4, 5].map((n) => `<button data-action="shelf-rate" data-player="${esc(p.id)}" data-k="${esc(k)}" data-n="${n}" aria-label="${n} ${n === 1 ? "star" : "stars"}" aria-pressed="${(e.rating || 0) >= n}">★</button>`).join("")}</div></div>
        ${e.note ? `<p class="sh-text">${esc(e.note)}</p>` : ""}</div>`;
    }).join("")}</div>`;
  }
  async function setShelf(pid, k, change) {
    const sh = (S.d.shelf = S.d.shelf || {});
    const mine = (sh[pid] = sh[pid] || {});
    const e = { status: "", rating: 0, note: "", ...(mine[k] || {}), ...change };
    if (!e.status && !e.rating && !e.note) delete mine[k]; else mine[k] = { ...e, updated: new Date().toISOString() };
    render();
    try { await api.setShelf(pid, k, { status: e.status, rating: e.rating, note: e.note }); }
    catch (err) { toast(errText(err), "error"); refresh(); }
  }
  function viewShelfNote() {
    const D = S.dialog, p = player(D.pid), it = itemByKey(D.k);
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Note">
      <div><h2>${esc(p ? p.name : "")}'s note on ${esc(it ? it.name : "this game")}</h2><p class="sub">Only people using this PC see it.</p></div>
      <textarea id="sn-text" rows="6" maxlength="500" placeholder="What you thought, where you got to, what to try next">${esc(D.note || "")}</textarea>
      <div class="dialog-foot"><span class="error-text grow" id="sn-error"></span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="shelf-note-save">Save note</button></div>
    </div></div>`;
  }

  function viewProfileEditor() {
    const D = S.dialog, p = D.p, tab = D.tab || "look";
    const tabs = [["look", "Look"], ["about", "About"], ["showcase", "Showcase"], ["page", "Page"]];
    const gameOptions = (sel, none) => `<option value="">${none}</option>${libraryItems().map((it) => `<option value="${esc(recentKey(it))}" ${sel === recentKey(it) ? "selected" : ""}>${esc(it.name)}</option>`).join("")}`;
    let body = "";
    if (tab === "look") {
      const hasPic = !!(S.d.avatars || {})[p.id], hasBanner = !!(S.d.banners || {})[p.id];
      const kind = p.banner === "image" ? "image" : p.banner && !p.banner.startsWith("style:") ? "game" : "plain";
      body = `
      <div class="field"><span class="label">Picture</span><div class="pe-row">
        <button class="btn" data-action="avatar-pick" ${D.busy ? "disabled" : ""}>${D.busy === "avatar" ? "Reading the picture" : hasPic ? "Change picture" : "Choose a picture"}</button>
        ${hasPic ? `<button class="btn quiet" data-action="avatar-remove">Remove picture</button>` : `<select data-pe="avatarGame" aria-label="Or a game's cover">${gameOptions(p.avatarGame || "", "Or use a game's cover")}</select>`}</div></div>
      <div class="field"><span class="label">Colour</span><div class="pe-row"><div class="swatches">${S.d.colors.map((c) => `<button class="swatch" style="background:${c}" data-action="pe-color" data-color="${c}" aria-pressed="${c.toLowerCase() === (p.color || "").toLowerCase()}" aria-label="${c}"></button>`).join("")}</div>
        <label class="custom-color"><input type="color" value="${esc((p.color || "#6c9bd2").toLowerCase())}" data-pe="color" aria-label="Any colour"><span>Any colour</span></label></div></div>
      <div class="field"><span class="label">Banner</span>
        <div class="segt">${[["plain", "Colour"], ["image", "Your picture"], ["game", "Game art"]].map(([v, l]) => `<button data-action="pe-banner" data-v="${v}" aria-pressed="${kind === v}">${l}</button>`).join("")}</div>
        ${kind === "game" ? `<select data-pe="banner" aria-label="Game">${gameOptions(p.banner, "Choose a game")}</select>` : ""}
        ${kind === "image" ? `<div class="pe-row"><button class="btn" data-action="banner-pick" ${D.busy ? "disabled" : ""}>${D.busy === "banner" ? "Reading the picture" : "Choose another picture"}</button><button class="btn quiet" data-action="banner-remove">Remove picture</button></div>` : ""}
        ${kind === "image" || kind === "game" ? `<label class="range"><span>Position</span><input type="range" min="0" max="100" step="1" value="${p.bannerY || 0}" data-pe="bannerY"></label>` : ""}
        <span class="hint">A picture is cropped to a wide strip (3:1) and kept on this PC.</span></div>`;
    } else if (tab === "about") {
      body = `
      <div class="field"><label for="pe-name">Name</label><input id="pe-name" type="text" maxlength="24" value="${esc(p.name)}" data-pe="name"></div>
      <div class="field"><label for="pe-tag">A line about you</label><input id="pe-tag" type="text" maxlength="80" value="${esc(p.tagline || "")}" placeholder="Support main. Never the driver." data-pe="tagline"></div>
      <div class="field"><label for="pe-bio">About <span class="hint">up to 300 characters</span></label><textarea id="pe-bio" rows="4" maxlength="300" placeholder="What you play, who you play with, the game you'd take to an island" data-pe="bio">${esc(p.bio || "")}</textarea></div>
      <div class="field"><label for="pe-now">Into right now</label><select id="pe-now" data-pe="now">${gameOptions(p.now || "", "Nothing in particular")}</select>
        <input type="text" maxlength="80" value="${esc(p.nowNote || "")}" placeholder="A word on it: “Act 3, finally”" data-pe="nowNote" aria-label="A word on it" style="margin-top:8px"></div>`;
    } else if (tab === "showcase") {
      const q = (D.q || "").toLowerCase(), fav = p.favorites || [];
      const games = libraryItems().filter((it) => !q || it.name.toLowerCase().includes(q));
      body = `
      <div class="field"><span class="label">Favourite games <span class="hint">${fav.length} of 6, in this order</span></span>
        ${fav.length ? `<ol class="fav-order">${fav.map((k, i) => { const it = itemByKey(k); return `<li><span>${esc(it ? it.name : "A game no longer here")}</span>
          <button class="btn quiet icon" data-action="pe-fav-move" data-i="${i}" data-d="-1" aria-label="Earlier" ${i === 0 ? "disabled" : ""}>↑</button>
          <button class="btn quiet icon" data-action="pe-fav-move" data-i="${i}" data-d="1" aria-label="Later" ${i === fav.length - 1 ? "disabled" : ""}>↓</button>
          <button class="btn quiet icon" data-action="pe-fav" data-k="${esc(k)}" aria-label="Remove">${icon.x}</button></li>`; }).join("")}</ol>` : ""}
        <input type="search" id="pe-q" placeholder="Find a game" value="${esc(D.q || "")}" autocomplete="off">
        <div class="pe-games">${games.map((it) => { const k = recentKey(it), on = fav.includes(k); return `<button class="pe-game${on ? " on" : ""}" data-action="pe-fav" data-k="${esc(k)}" aria-pressed="${on}" ${!on && fav.length >= 6 ? "disabled" : ""}><span class="check"></span>${esc(it.name)}</button>`; }).join("")}</div></div>`;
    } else {
      const on = sectionsOf(p);
      const order = [...on, ...SECTIONS.map((s) => s[0]).filter((s) => !on.includes(s))];
      body = `<p class="sub" style="margin:0">What your page shows, top to bottom.</p><ul class="sec-list">${order.map((k) => {
        const s = SECTIONS.find((x) => x[0] === k), i = on.indexOf(k), shown = i >= 0;
        return `<li class="${shown ? "" : "off"}"><button class="sec-toggle" data-action="pe-sec" data-k="${k}" aria-pressed="${shown}" ${shown && on.length === 1 ? "disabled" : ""}><span class="check"></span><span><b>${s[1]}</b><span>${s[2]}</span></span></button>
          ${shown ? `<button class="btn quiet icon" data-action="pe-sec-move" data-k="${k}" data-d="-1" aria-label="Higher" ${i === 0 ? "disabled" : ""}>↑</button><button class="btn quiet icon" data-action="pe-sec-move" data-k="${k}" data-d="1" aria-label="Lower" ${i === on.length - 1 ? "disabled" : ""}>↓</button>` : ""}</li>`;
      }).join("")}</ul>`;
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog pe" role="dialog" aria-modal="true" aria-label="Edit ${esc(p.name)}'s profile" style="--c:${esc(p.color)}">
      <div class="pe-preview"><div class="pe-pbanner">${bannerHTML(p)}</div>${avatarHTML(p, "lg")}
        <div class="pe-pname"><b id="pe-prev-name">${esc(p.name)}</b><span id="pe-prev-tag">${esc(p.tagline || "")}</span></div>
        <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></div>
      <div class="segt pe-tabs" role="tablist">${tabs.map(([k, l]) => `<button role="tab" aria-selected="${k === tab}" data-action="pe-tab" data-tab="${k}">${l}</button>`).join("")}</div>
      <div class="pe-body">${body}</div>
      <div class="dialog-foot"><span class="error-text grow" id="pe-error"></span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="pe-save">Save profile</button></div>
    </div></div>`;
  }

  function viewPlayers() {
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    const dirty = JSON.stringify(ps) !== JSON.stringify(cfg().players);
    const cards = cfg().players.map((p) => { const st = playerStats(p); return `<button class="pcard" data-action="open-player" data-player="${esc(p.id)}" style="--c:${esc(p.color)}">
        <span class="pcard-banner">${bannerHTML(p)}</span>${avatarHTML(p, "lg")}
        <span class="pcard-body"><b>${esc(p.name)}</b><span>${p.tagline ? esc(p.tagline) : "No line about them yet"}</span>
        <span class="pcard-nums"><span><b>${st.total ? esc(hours(st.total)) : "0 h"}</b> played</span><span><b>${st.shelfN.playing || 0}</b> playing</span><span><b>${st.shelfN.finished || 0}</b> finished</span></span></span></button>`; }).join("");
    return `<div class="page"><div class="page-head"><div><h1>Players</h1><p class="sub">Everyone who plays on this PC. Their colour shows whose files are loaded in each game.</p></div></div>
      <div class="pcards">${cards}</div>
      <h2 class="shelf-title" style="margin-top:36px">Names and colours</h2>
      <div class="players">${ps.map((p, i) => playerRow(p, i, ps.length, false)).join("")}</div>
      <div class="row" style="margin-top:14px">${ps.length < 8 ? `<button class="btn" data-action="add-player">${icon.plus}Add player</button>` : ""}</div>
      <div class="row" style="margin-top:28px"><button class="btn primary" data-action="save-players" ${dirty ? "" : "disabled"}>Save players</button>
      ${dirty ? `<button class="btn quiet" data-action="reset-players">Discard changes</button>` : ""}<span class="error-text" id="players-error"></span></div></div>`;
  }

  async function savePlayers(welcome) {
    const ps = S.players.map((p, i) => ({ ...p, name: (p.name || "").trim() || (welcome ? "Player " + (i + 1) : "") }));
    const removed = cfg().players.filter((o) => !ps.some((p) => p.id === o.id));
    if (removed.length && !welcome) {
      const ok = await confirmDialog({ title: `Remove ${removed.map((p) => p.name).join(" and ")}?`, body: "Their profiles are moved to the “removed” folder inside Lootnest's data folder, so nothing is deleted for good.", ok: "Remove" });
      if (!ok) return;
    }
    try {
      await api.savePlayers(ps);
      S.players = null;
      if (welcome) S.dialog = null;
      await refresh();
      if (!welcome) toast("Players saved");
    } catch (e) { const t = $("#players-error"); if (t) t.textContent = errText(e); }
  }

  // -------------------------------------------------------------- settings --
  function sw(field, title, desc, checked) {
    return `<label class="toggle"><span><b style="font-weight:500">${title}</b><p>${desc}</p></span>
      <span class="switch"><input type="checkbox" data-setting="${field}" ${checked ? "checked" : ""}><span></span></span></label>`;
  }

  function viewSettings() {
    const s = cfg().settings, st = S.d.steam;
    return `<div class="page"><div class="page-head"><div><h1>Settings</h1></div></div>
      <div class="form">
        <section class="section"><div class="toggles">
          ${sw("autostart", "Start with Windows", "Lootnest opens quietly in the tray when you sign in, so it's always watching your games.", S.d.autostart)}
          ${sw("closeToTray", "Keep running when the window is closed", "Closing the window leaves Lootnest in the tray. Quit from the tray icon.", s.closeToTray)}
          ${sw("hideWhilePlaying", "Hide while a game runs", "The window gets out of the way once the game starts.", s.hideWhilePlaying)}
          ${sw("watchOutsideLaunches", "Follow games started from Steam or elsewhere", "If a game starts without Lootnest, its progress is saved to whoever's files are loaded.", s.watchOutsideLaunches)}
          ${sw("autoSteamHooks", "Set up Steam automatically", "Steam games you add get a launch option so Steam's Play button asks who's playing. Removed games get their own options back.", s.autoSteamHooks)}
          ${sw("notifications", "Notifications", "A short Windows notification when progress is saved.", s.notifications)}
          ${sw("checkUpdates", "Check for updates automatically", "Every few hours, Lootnest asks GitHub for its latest version number and tells you with a notification when a new one is out. Nothing about you or your games is sent.", s.checkUpdates)}
          ${s.checkUpdates ? sw("autoInstallUpdates", "Install updates by themselves", "A new version installs while Lootnest is in the tray and no game is running, then Lootnest opens again in the tray. Your players, games and profiles stay as they are.", s.autoInstallUpdates) : ""}
          ${sw("gameNews", "Show game updates and news", "Each game's page shows its latest updates and patch notes from Steam's news feed, and tells you when a game no store keeps up to date has a newer version than yours. Only the game's Steam id, or its name, is sent.", s.gameNews)}
          ${sw("downloadCovers", "Download covers and artwork", "Covers, wide artwork and logos for games Steam hasn't saved on this PC, looked up on the Steam store by name (and on SteamGridDB with a key below), saved on this PC. Only the game's name or Steam id is sent. Off: those games show their own icon.", s.downloadCovers)}
        </div></section>
        <section class="section"><div class="numbers">
          <div class="field wide"><label for="s-grid">SteamGridDB API key <span class="hint">(optional)</span></label>
            <input id="s-grid" type="password" autocomplete="off" spellcheck="false" data-setting="gridDbKey" value="${esc(s.gridDbKey || "")}" placeholder="Paste your key">
            <span class="hint">Covers and artwork for games the Steam store doesn't know. Get a free key on steamgriddb.com, under Preferences, API. It's sent only to SteamGridDB.</span></div>
          <div class="field"><label for="s-backups">Backups kept per game</label><input id="s-backups" type="number" min="1" max="100" data-setting="backupsToKeep" value="${s.backupsToKeep}"></div>
          <div class="field"><label for="s-timeout">Seconds to wait for a game to start</label><input id="s-timeout" type="number" min="30" max="1800" data-setting="startTimeoutSeconds" value="${s.startTimeoutSeconds}"></div>
          ${st.found ? `<div class="field"><label for="s-steam">Steam account</label><select id="s-steam" data-setting="steamAccountId">
            <option value="">Automatic${st.accountInUse && !s.steamAccountId ? " (" + esc(st.accountInUse) + ")" : ""}</option>
            ${(st.accounts || []).map((a) => `<option value="${esc(a)}" ${a === s.steamAccountId ? "selected" : ""}>${esc(a)}</option>`).join("")}</select>
            ${st.accountError ? `<span class="error-text">${esc(st.accountError)}</span>` : ""}</div>` : ""}
        </div></section>
        <section class="section">
          <div class="section-head"><h2>Data</h2><p class="selectable">Profiles and backups are stored in ${esc(S.d.dataDir)}</p></div>
          <div class="row"><button class="btn" data-action="open-data">Open data folder</button><button class="btn" data-action="show-log">Show recent activity</button></div>
          ${S.logText ? `<div class="log">${esc(S.logText)}</div>` : ""}
        </section>
        <section class="section">
          <div class="section-head"><h2>About</h2><p>Lootnest ${esc(S.d.version)}. Uses the Microsoft Edge WebView2 runtime. To uninstall, use Settings, Apps in Windows.</p></div>
          <div class="row">${S.update.available
            ? `<button class="btn primary" data-action="update-now" ${S.update.step ? "disabled" : ""}>${S.update.step ? "Updating" : `Update to ${esc(S.update.version)}`}</button>`
            : `<button class="btn" data-action="update-check" ${S.update.checking ? "disabled" : ""}>${S.update.checking ? "Checking" : "Check for updates"}</button>`}
            <button class="btn quiet danger" data-action="quit">Quit Lootnest</button></div>
        </section>
      </div></div>`;
  }

  async function saveSetting(el) {
    const s = JSON.parse(JSON.stringify(cfg().settings));
    let autostart = S.d.autostart;
    const f = el.dataset.setting;
    if (f === "autostart") autostart = el.checked;
    else if (el.type === "checkbox") s[f] = el.checked;
    else if (el.type === "number") s[f] = parseInt(el.value, 10) || 0;
    else s[f] = el.value;
    try {
      await api.saveSettings({ settings: s, autostart });
      if (f === "downloadCovers" || f === "gridDbKey") { S.covers = {}; S.icons = {}; S.art = {}; artWanted.clear(); pendingCovers.clear(); }
      await refresh();
    } catch (e) { toast(errText(e), "error"); await refresh(); }
  }

  // --------------------------------------------------------------- actions --
  const actions = {
    nav(el) {
      if (S.view === "edit" && !confirmLeaveEdit()) return;
      S.view = el.dataset.view; S.menuFor = null; S.edit = null; S.animateView = true;
      if (S.view === "library") S.animateCards = true;
      if (S.view !== "players") S.players = null;
      render(); $("#view").scrollTop = 0;
    },
    "add-game": startAddDialog,
    "add-steam": (el) => addSteam(el.dataset.app),
    "add-exe": addExe,
    play: (el) => openPicker(el.dataset.game, "play"),
    "open-sheet": (el) => openSheet(el.dataset.game ? "game" : "found", el.dataset.game || el.dataset.key, el.closest(".card")),
    "back-library": () => backToLibrary(),
    "open-player"(el) { S.playerPage = el.dataset.player; S.view = "player"; S.animateView = true; render(); $("#view").scrollTop = 0; },
    "edit-profile"(el) { const p = player(el.dataset.player); if (!p) return; S.dialog = { kind: "profile", p: JSON.parse(JSON.stringify(p)), q: "", tab: "look" }; renderOverlay(); },
    "pe-tab"(el) { S.dialog.tab = el.dataset.tab; renderOverlay(); },
    "pe-set"(el) { S.dialog.p[el.dataset.f] = el.dataset.v; renderOverlay(); },
    "pe-color"(el) { S.dialog.p.color = el.dataset.color; renderOverlay(); },
    "pe-banner"(el) {
      const D = S.dialog, v = el.dataset.v;
      if (v === "image") { if (!(S.d.banners || {})[D.p.id]) return actions["banner-pick"](); D.p.banner = "image"; }
      else if (v === "game") { const f = (D.p.favorites || [])[0] || (libraryItems()[0] && recentKey(libraryItems()[0])) || ""; D.p.banner = D.p.banner && D.p.banner !== "image" && !D.p.banner.startsWith("style:") ? D.p.banner : f; }
      else D.p.banner = "";
      renderOverlay();
    },
    async "banner-pick"() {
      const D = S.dialog;
      const r = await browse("image");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = "banner"; renderOverlay();
      try { await api.setBanner(D.p.id, r.path); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "banner-remove"() {
      try { await api.removeBanner(S.dialog.p.id); if (S.dialog.p.banner === "image") S.dialog.p.banner = ""; await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "pe-fav"(el) {
      const f = S.dialog.p.favorites = S.dialog.p.favorites || [];
      const i = f.indexOf(el.dataset.k);
      if (i >= 0) f.splice(i, 1); else if (f.length < 6) f.push(el.dataset.k);
      renderOverlay();
    },
    "pe-fav-move"(el) {
      const f = S.dialog.p.favorites, i = +el.dataset.i, j = i + +el.dataset.d;
      if (j < 0 || j >= f.length) return;
      [f[i], f[j]] = [f[j], f[i]]; renderOverlay();
    },
    "pe-sec"(el) {
      const p = S.dialog.p, on = sectionsOf(p).slice(), i = on.indexOf(el.dataset.k);
      if (i >= 0) { if (on.length > 1) on.splice(i, 1); } else on.push(el.dataset.k);
      p.sections = on; renderOverlay();
    },
    "pe-sec-move"(el) {
      const p = S.dialog.p, on = sectionsOf(p).slice(), i = on.indexOf(el.dataset.k), j = i + +el.dataset.d;
      if (i < 0 || j < 0 || j >= on.length) return;
      [on[i], on[j]] = [on[j], on[i]]; p.sections = on; renderOverlay();
    },
    async "avatar-pick"() {
      const D = S.dialog;
      const r = await browse("image");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = "avatar"; renderOverlay();
      try { await api.setAvatar(D.p.id, r.path); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "avatar-remove"() { try { await api.removeAvatar(S.dialog.p.id); S.dialog.p.avatar = false; await refresh(); } catch (e) { toast(errText(e), "error"); } },
    async "pe-save"() {
      const D = S.dialog, p = { ...D.p };
      p.avatar = !!(S.d.avatars || {})[p.id];
      if (p.banner === "image" && !(S.d.banners || {})[p.id]) p.banner = "";
      p.bannerY = Math.round(+p.bannerY || 0);
      const ps = cfg().players.map((x) => (x.id === p.id ? p : x));
      try { await api.savePlayers(ps); S.dialog = null; S.players = null; await refresh(); toast("Profile saved"); }
      catch (e) { const t = $("#pe-error"); if (t) t.textContent = errText(e); }
    },
    "shelf-tab"(el) { S.shelfTab = el.dataset.tab; render(); },
    "shelf-set"(el) {
      const cur = (shelfOf(el.dataset.player)[el.dataset.k] || {}).status;
      setShelf(el.dataset.player, el.dataset.k, { status: cur === el.dataset.status ? "" : el.dataset.status });
    },
    "shelf-rate"(el) {
      const cur = (shelfOf(el.dataset.player)[el.dataset.k] || {}).rating || 0, n = +el.dataset.n;
      setShelf(el.dataset.player, el.dataset.k, { rating: cur === n ? 0 : n });
    },
    "shelf-note"(el) {
      S.dialog = { kind: "shelf-note", pid: el.dataset.player, k: el.dataset.k, note: (shelfOf(el.dataset.player)[el.dataset.k] || {}).note || "" };
      renderOverlay(); setTimeout(() => { const t = $("#sn-text"); if (t) t.focus(); }, 30);
    },
    async "shelf-note-save"() {
      const D = S.dialog, note = $("#sn-text").value;
      if (note.trim().length > 500) { $("#sn-error").textContent = "Keep the note under 500 characters."; return; }
      S.dialog = null; renderOverlay();
      await setShelf(D.pid, D.k, { note: note.trim() });
    },
    density(el) { S.density = el.dataset.size; setPref("density", S.density); render(); },
    "open-news": (el) => api.open("news", el.dataset.url).catch((e) => toast(errText(e), "error")),
    "open-link": (el) => api.open("link", el.dataset.url).catch((e) => toast(errText(e), "error")),
    "read-news"(el) { S.dialog = { kind: "news", key: el.dataset.key, i: +el.dataset.i }; renderOverlay(); setTimeout(() => { const b = $(".reader .btn.quiet.icon"); if (b) b.focus({ preventScroll: true }); }, 30); },
    async "news-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.gameNews = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "play-as"(el) { openPicker(el.dataset.game, "play"); S.picker.playerId = el.dataset.player; runPick(); },
    async "play-found"(el) {
      const f = (S.found.games || []).find((x) => x.key === el.dataset.key);
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      try { await api.launchFound(el.dataset.key); toast(`Starting ${f ? f.name : "the game"}`); } catch (e) { toast(errText(e), "error"); }
    },
    filter(el) { S.filter = el.dataset.filter; S.animateCards = true; render(); },
    async "covers-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.downloadCovers = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); S.covers = {}; S.icons = {}; S.art = {}; artWanted.clear(); pendingCovers.clear(); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "covers-later"() { S.hideCoverHint = true; render(); },
    async "updates-yes"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.checkUpdates = true; s.autoInstallUpdates = true; s.updatesAsked = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); toast("Lootnest keeps itself up to date. You can change this in Settings."); api.checkUpdate().catch(() => {}); } catch (e) { toast(errText(e), "error"); }
    },
    async "updates-no"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.updatesAsked = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "update-check"() { S.update = { checking: true }; render(); api.checkUpdate().catch((e) => { S.update = {}; toast(errText(e), "error"); render(); }); },
    async "update-now"() {
      try { await api.installUpdate(); } catch (e) { toast(errText(e), "error"); }
    },
    "update-notes"() { api.open("release").catch((e) => toast(errText(e), "error")); },
    "load-only": (el) => openPicker(el.dataset.game, "load"),
    menu(el) { S.menuFor = S.menuFor === el.dataset.game ? null : el.dataset.game; render(); const m = $(".menu button"); if (m) m.focus(); },
    "edit-game": (el) => { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; editGame(el.dataset.game); },
    async shortcut(el) {
      S.menuFor = null; render();
      try { await api.shortcut(el.dataset.game); toast("Shortcut added to your desktop"); } catch (e) { toast(errText(e), "error"); }
    },
    "open-path": (el) => { S.menuFor = null; render(); api.open("path", el.dataset.path).catch((e) => toast(errText(e), "error")); },
    "open-backups": (el) => { S.menuFor = null; render(); api.open("backups", el.dataset.game).catch((e) => toast(errText(e), "error")); },
    async "remove-game"(el) {
      S.menuFor = null; render();
      const g = game(el.dataset.game);
      const ok = await confirmDialog({ title: `Turn off profiles for ${g.name}?`, body: "The game stays in your library. Everyone's profiles are moved to the “removed” folder in Lootnest's data, and the game keeps the files of whoever played last.", ok: "Turn off profiles" });
      if (!ok) return;
      try { await api.deleteGame(g.id); toast(`Profiles are off for ${g.name}`); await refresh(); loadFound(); } catch (e) { toast(errText(e), "error"); }
    },
    pick(el) { S.picker.playerId = el.dataset.player; S.picker.owner = ""; S.picker.newMode = ""; runPick(); },
    "answer-owner"(el) { S.picker.owner = el.dataset.player; S.picker.ask = null; runPick(); },
    "answer-new"(el) { S.picker.newMode = el.dataset.mode; S.picker.ask = null; runPick(); },
    "picker-back"() { S.picker.ask = null; S.picker.owner = ""; S.picker.newMode = ""; render(); },
    "picker-close": closePicker,
    "dialog-ok": () => closeDialog(true),
    async "fix-blocked"() {
      const D = S.dialog;
      if (!D || D.kind !== "blocked" || D.busy) return;
      D.busy = true; renderOverlay();
      try { await api.fixBlocked(D.gameId, D.playerId); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    "dialog-cancel": () => closeDialog(false),
    "game-settings"(el) { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; openGameSettings(el.dataset.key); },
    async "game-settings-save"() {
      const D = S.dialog;
      if (!D || D.kind !== "game-settings") return;
      const name = $("#gs-name").value, args = $("#gs-args").value;
      D.busy = true; renderOverlay();
      try {
        await api.saveLibraryPref(D.key, name, args);
        await loadFound();
        if (S.dialog === D) { S.dialog = null; renderOverlay(); }
        toast("Saved");
      } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "open-game-folder"(el) {
      try { await (el.dataset.game ? api.openGameFolder("game", el.dataset.game) : api.openGameFolder("found", el.dataset.key)); } catch (e) { toast(errText(e), "error"); }
    },
    async "found-shortcut"(el) {
      try { await api.foundShortcut(el.dataset.key); toast("Shortcut added to your desktop"); } catch (e) { toast(errText(e), "error"); }
    },
    async "steam-verify"(el) {
      try { await api.steamVerify(+el.dataset.app); toast("Steam is checking the game's files. It downloads any that are missing or damaged."); } catch (e) { toast(errText(e), "error"); }
    },
    "move-open"(el) { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; openMove(+el.dataset.app, el.dataset.name); },
    "move-pick"(el) { if (S.dialog && S.dialog.kind === "move") { S.dialog.pick = el.dataset.lib; renderOverlay(); } },
    async "move-go"() {
      const D = S.dialog;
      if (!D || D.kind !== "move" || !D.pick) return;
      try {
        await api.moveGame(D.appId, D.pick);
        if (S.dialog !== D) return; // already done
        S.moving = Object.assign({ appId: D.appId, name: D.name, step: "", done: 0, total: 0 }, S.moving && S.moving.appId === D.appId ? S.moving : {});
        renderOverlay();
      } catch (e) { toast(errText(e), "error"); }
    },
    scrim(el, ev) { if (ev.target === el && S.dialog && S.dialog.kind !== "welcome") closeDialog(false); },
    "edit-cancel"() { if (!confirmLeaveEdit()) return; S.edit = null; S.view = "library"; S.animateView = true; render(); },
    async "browse-exe"() {
      const r = await browse("exe", S.edit.game.launch.exe);
      if (r.path && S.edit) { S.edit.game.launch.exe = r.tokenized || r.path; render(); }
    },
    "set-proc"(el) { S.edit.game.processName = el.dataset.name; render(); },
    async "add-path"(el) {
      const start = (S.edit.game.paths[0] || {}).path || "";
      const r = await browse(el.dataset.kind, start);
      if (r.error) toast(r.error, "error");
      if (!r.path || !S.edit) return;
      addPath(r.tokenized || r.path);
    },
    "add-sugg"(el) { addPath(el.dataset.path); },
    "remove-path"(el) { S.edit.game.paths = S.edit.game.paths.filter((p) => p._k !== el.dataset.k); render(); },
    pcgw() { api.open("pcgw", S.edit.game.name).catch((e) => toast(errText(e), "error")); },
    "save-game": saveGame,
    swatch(el) { S.players[+el.dataset.i].color = el.dataset.color; S.dialog && S.dialog.kind === "welcome" ? renderOverlay() : render(); },
    "add-player"() { const used = new Set(S.players.map((p) => p.color)); S.players.push({ id: "", name: "", color: S.d.colors.find((c) => !used.has(c)) || S.d.colors[0] }); render(); const ins = document.querySelectorAll('[data-field="pname"]'); ins[ins.length - 1].focus(); },
    "remove-player"(el) { S.players.splice(+el.dataset.i, 1); render(); },
    "reset-players"() { S.players = null; render(); },
    "save-players": () => savePlayers(false),
    "welcome-done": () => savePlayers(true),
    "open-data": () => api.open("data").catch((e) => toast(errText(e), "error")),
    "steam-setup": (el) => openSteamSetup(el.dataset.game),
    async "steam-recheck"() { S.hooks = null; renderOverlay(); await loadHooks(); if (S.dialog && S.dialog.kind === "steam") renderOverlay(); render(); },
    async "copy-hook"(el) {
      const h = S.hooks && S.hooks.games.find((x) => x.gameId === el.dataset.game);
      if (!h) return;
      try { await api.copy(h.option); S.dialog.copied = h.gameId; renderOverlay(); } catch (e) { toast(errText(e), "error"); }
    },
    "hide-banner"() { S.hideHookBanner = true; render(); toast("Lootnest will finish this the next time Steam is closed."); },
    async "hooks-apply"() {
      S.hooksBusy = true; render(); if (S.dialog) renderOverlay();
      try {
        const onlyLeaving = S.hooks && !S.hooks.games.some((h) => !h.detected && !h.direct);
        const r = await api.applyHooks(true);
        await loadHooks();
        if (r && r.changed && onlyLeaving) toast(r.restarted ? "Steam restarted. It now starts those games by itself." : "Done. Steam now starts those games by itself.");
        else if (r && r.changed) toast(r.restarted ? "Steam restarted. Its Play button now asks who's playing." : "Done. Steam's Play button now asks who's playing.");
      } catch (e) { toast(errText(e), "error"); }
      S.hooksBusy = false; render(); if (S.dialog) renderOverlay();
    },
    rescan() { S.found = { scanning: !(S.found.games || []).length, refreshing: true, games: S.found.games || [] }; render(); api.scan(); },
    async "add-all"() {
      const keys = (S.found.games || []).filter((f) => f.ready).map((f) => f.key);
      await addFound(keys);
    },
    async "found-add"(el) {
      const f = (S.found.games || []).find((x) => x.key === el.dataset.key);
      if (!f) return;
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      if (f.ready) { await addFound([f.key]); return; }
      openEditor({
        isNew: true,
        game: { id: "", name: f.name, launch: f.source === "steam" ? { type: "steam", appId: f.appId, args: f.args || "" } : { type: "exe", exe: f.exe, args: f.args || "" },
          processName: f.processName || "", playerArgs: {}, paths: (f.paths || []).map((p) => ({ id: "", path: p })) },
        inspect: { suggestions: f.suggestions || [], candidates: [] },
      });
    },
    async "found-hide"(el) {
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      try { await api.ignoreFound(el.dataset.key); S.found.games = S.found.games.filter((f) => f.key !== el.dataset.key); render(); } catch (e) { toast(errText(e), "error"); }
    },
    async "show-log"() { try { S.logText = (await api.log()) || "Nothing yet."; } catch (e) { S.logText = errText(e); } render(); },
    quit: () => api.quit(),
  };

  async function addFound(keys) {
    S.adding = true; render();
    try {
      const r = await api.addFound(keys);
      if (r.added.length) toast(r.added.length === 1 ? `Profiles are on for ${r.added[0]}` : `Profiles are on for ${r.added.length} games`);
      if (r.failed.length) toast(r.failed.join("\n"), "error");
      await refresh();
      if (S.view === "game" && S.page && S.page.kind === "found" && r.added.length === 1) {
        const g = cfg().games.find((x) => x.name === r.added[0]);
        if (g) S.page = { kind: "game", id: g.id, cloud: null };
      }
      await loadFound();
      await loadHooks();
    } catch (e) { toast(errText(e), "error"); }
    S.adding = false; render();
  }

  function addPath(path) {
    const E = S.edit;
    if (E.game.paths.some((p) => p.path.toLowerCase() === path.toLowerCase())) return;
    const p = { id: "", path, _k: "k" + ++seq };
    E.game.paths.push(p);
    render();
    checkPath(p);
  }

  function confirmLeaveEdit() {
    return true; // edits are explicit; leaving discards them
  }

  document.addEventListener("click", (ev) => {
    const el = ev.target.closest("[data-action]");
    if (!el) {
      if (S.menuFor && !ev.target.closest(".menu")) { S.menuFor = null; render(); }
      return;
    }
    if (el.dataset.action !== "menu" && S.menuFor && !el.closest(".menu")) S.menuFor = null;
    if (el.closest("#ctx")) closeContext();
    const f = actions[el.dataset.action];
    if (f) f(el, ev);
  });

  document.addEventListener("input", (ev) => {
    const el = ev.target;
    if (S.dialog && S.dialog.kind === "profile") {
      const f = el.dataset.pe, D = S.dialog;
      if (f) D.p[f] = f === "bannerY" ? +el.value : el.value;
      if (f === "name") { const n = $("#pe-prev-name"); if (n) n.textContent = el.value; }
      else if (f === "tagline") { const n = $("#pe-prev-tag"); if (n) n.textContent = el.value; }
      else if (f === "color") { const d = $(".dialog.pe"); if (d) d.style.setProperty("--c", el.value); document.querySelectorAll('.dialog.pe [style*="--c"]').forEach((a) => a.style.setProperty("--c", el.value)); }
      else if (f === "bannerY") { const a = $(".pe-pbanner .art"); if (a) a.style.backgroundPosition = `center ${el.value}%`; }
      else if (f === "banner" || f === "avatarGame") renderOverlay();
      if (el.id === "pe-q") { D.q = el.value; renderOverlay(); }
      return;
    }
    if (el.id === "lib-q") {
      S.q = el.value;
      const grid = $("#grid");
      if (grid) { grid.innerHTML = gridHTML(); loadCovers(); }
      return;
    }
    if (el.id === "steam-q" && S.dialog) {
      S.dialog.q = el.value;
      const pos = el.selectionStart;
      renderOverlay();
      const q = $("#steam-q"); q.focus(); q.setSelectionRange(pos, pos);
      return;
    }
    const f = el.dataset.field;
    if (!f) return;
    if (f === "pname") { S.players[+el.dataset.i].name = el.value; if (S.view === "players" && !S.dialog) { const b = $('[data-action="save-players"]'); if (b) b.disabled = false; } return; }
    const E = S.edit;
    if (!E) return;
    const g = E.game;
    if (f === "name") g.name = el.value;
    else if (f === "exe") g.launch.exe = el.value;
    else if (f === "processName") g.processName = el.value;
    else if (f === "steamDirect") g.steamDirect = el.checked;
    else if (f === "args") g.launch.args = el.value;
    else if (f === "parg") { g.playerArgs = g.playerArgs || {}; g.playerArgs[el.dataset.player] = el.value; }
    else if (f === "path") { const p = g.paths.find((x) => x._k === el.dataset.k); if (p) p.path = el.value; }
  });

  document.addEventListener("change", (ev) => {
    const el = ev.target;
    if (el.dataset.setting) { saveSetting(el); return; }
    if (el.id === "lib-sort") { S.sort = el.value; setPref("sort", S.sort); render(); return; }
    if (el.dataset.field === "path" && S.edit) { const p = S.edit.game.paths.find((x) => x._k === el.dataset.k); if (p) checkPath(p); }
  });

  document.addEventListener("keydown", (ev) => {
    if (!$("#ctx").hidden) {
      if (ev.key === "Escape") { ev.preventDefault(); closeContext(); return; }
      if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
        ev.preventDefault();
        const b = [...document.querySelectorAll("#ctx button:not(:disabled)")];
        const i = b.indexOf(document.activeElement);
        b[(i + (ev.key === "ArrowDown" ? 1 : b.length - 1)) % b.length].focus();
        return;
      }
    }
    // Ctrl+F or / finds a game.
    if (!S.picker && !S.dialog && ((ev.ctrlKey && ev.key.toLowerCase() === "f") || (ev.key === "/" && !ev.target.closest("input, textarea, select")))) {
      ev.preventDefault();
      if (S.view !== "library") { S.view = "library"; S.page = null; render(); }
      const q = $("#lib-q"); if (q) { q.focus(); q.select(); }
      return;
    }
    if (S.picker && !S.picker.working) {
      if (ev.key === "Escape") { ev.preventDefault(); if (S.picker.ask) actions["picker-back"](); else closePicker(); return; }
      if (!S.picker.ask && /^[1-9]$/.test(ev.key)) {
        const p = cfg().players[+ev.key - 1];
        if (p) { ev.preventDefault(); S.picker.playerId = p.id; runPick(); }
      }
      return;
    }
    if (ev.key === "Escape") {
      if (S.dialog && S.dialog.kind !== "welcome") { closeDialog(false); return; }
      if (S.menuFor) { S.menuFor = null; render(); return; }
      if (S.view === "game" && !ev.target.closest("input, textarea, select")) backToLibrary();
    }
    if (ev.key === "Enter" && S.dialog && S.dialog.kind === "welcome" && ev.target.dataset.field === "pname") savePlayers(true);
  });

  // block browser shortcuts that make no sense in an app window
  document.addEventListener("keydown", (ev) => {
    if ((ev.ctrlKey && ["p", "f", "g", "r", "u", "s", "+", "-", "=", "0"].includes(ev.key.toLowerCase())) || ev.key === "F5" || ev.key === "F3") ev.preventDefault();
  }, true);
  // No page zoom (Ctrl+wheel, pinch) and no browser menu outside text fields.
  document.addEventListener("wheel", (ev) => { if (ev.ctrlKey) ev.preventDefault(); }, { passive: false });
  document.addEventListener("contextmenu", (ev) => {
    if (!ev.target.closest("input, textarea, .selectable")) ev.preventDefault();
  });
  document.addEventListener("dragover", (ev) => ev.preventDefault());
  document.addEventListener("drop", (ev) => ev.preventDefault());

  // ------------------------------------------------------------------ boot --
  async function boot() {
    try {
      S.d = await api.state();
    } catch (e) {
      setTimeout(boot, 150); // bindings not injected yet
      return;
    }
    if (S.d.loadError) toast(S.d.loadError, "error");
    if (S.d.firstRun) { S.players = JSON.parse(JSON.stringify(cfg().players)); S.dialog = { kind: "welcome" }; }
    render();
    loadHooks().then(() => { if (!S.picker) render(); });
    loadFound();
    if (S.d.firstRun) setTimeout(() => { const i = $('[data-field="pname"]'); if (i) i.focus(); }, 50);
    try {
      const r = await api.ready();
      if (r && r.launch) openPicker(r.launch, r.reqId ? "steam" : "play", r.reqId);
    } catch (e) { /* ignore */ }
  }
  boot();
})();
